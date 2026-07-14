"""Diff engine: HTTP response diff + write-set diff, both noise-aware.

Verdicts (per request):
    MATCH          responses and write-sets are equivalent
    RESPONSE_DIFF  response bodies/status differ
    WRITESET_DIFF  DB row changes differ
    BOTH_DIFF      both differ
    ERROR          one side failed to respond / harness error
"""

from __future__ import annotations

import base64
import datetime
import decimal
import json
from dataclasses import dataclass, field

from .binlog import RowChange

MATCH = "MATCH"
RESPONSE_DIFF = "RESPONSE_DIFF"
WRITESET_DIFF = "WRITESET_DIFF"
BOTH_DIFF = "BOTH_DIFF"
ERROR = "ERROR"


@dataclass
class Difference:
    kind: str  # "response" | "writeset"
    path: str
    old: object
    new: object

    def as_dict(self) -> dict:
        return {"kind": self.kind, "path": self.path, "old": self.old, "new": self.new}


@dataclass
class RequestResult:
    name: str
    request: str
    verdict: str
    differences: list[Difference] = field(default_factory=list)
    old_status: int | None = None
    new_status: int | None = None
    old_writes: int = 0
    new_writes: int = 0
    error: str | None = None
    old_writeset: list[dict] = field(default_factory=list)
    new_writeset: list[dict] = field(default_factory=list)


# --------------------------------------------------------------------------
# value normalization (binlog & JSON values -> comparable primitives)
# --------------------------------------------------------------------------


def norm_value(v: object) -> object:
    if isinstance(v, decimal.Decimal):
        # scale-insensitive: Decimal("50000") == Decimal("50000.0")
        return int(v) if v == v.to_integral_value() else float(v)
    if isinstance(v, (datetime.datetime, datetime.date, datetime.time)):
        return v.isoformat()
    if isinstance(v, datetime.timedelta):
        return str(v)
    if isinstance(v, (bytes, bytearray)):
        return base64.b64encode(bytes(v)).decode()
    if isinstance(v, dict):
        return {k: norm_value(x) for k, x in v.items()}
    if isinstance(v, (list, tuple)):
        return [norm_value(x) for x in v]
    return v


# --------------------------------------------------------------------------
# noise path matching for JSON bodies
#   pattern segments: literal | * (exactly one) | ** (zero or more)
# --------------------------------------------------------------------------


def path_matches(pattern: str, path: list[str]) -> bool:
    return _match(pattern.split("."), path)


def _match(pat: list[str], path: list[str]) -> bool:
    if not pat:
        return not path
    head, rest = pat[0], pat[1:]
    if head == "**":
        return any(_match(rest, path[i:]) for i in range(len(path) + 1))
    if not path:
        return False
    if head == "*" or head == path[0]:
        return _match(rest, path[1:])
    return False


def strip_noise(obj: object, patterns: list[str], _path: list[str] | None = None) -> object:
    """Return a copy of obj with noise-matching keys removed."""
    path = _path or []
    if isinstance(obj, dict):
        out = {}
        for k, v in obj.items():
            child = path + [str(k)]
            if any(path_matches(p, child) for p in patterns):
                continue
            out[k] = strip_noise(v, patterns, child)
        return out
    if isinstance(obj, list):
        return [strip_noise(v, patterns, path + [str(i)]) for i, v in enumerate(obj)]
    return obj


# --------------------------------------------------------------------------
# response diff
# --------------------------------------------------------------------------


def _try_json(body: str | None) -> object:
    if body is None:
        return None
    try:
        return json.loads(body)
    except (ValueError, TypeError):
        return body


def diff_json(old: object, new: object, path: list[str], out: list[Difference]) -> None:
    if isinstance(old, dict) and isinstance(new, dict):
        for k in sorted(set(old) | set(new)):
            child = path + [str(k)]
            if k not in old:
                out.append(Difference("response", ".".join(child), "<absent>", norm_value(new[k])))
            elif k not in new:
                out.append(Difference("response", ".".join(child), norm_value(old[k]), "<absent>"))
            else:
                diff_json(old[k], new[k], child, out)
        return
    if isinstance(old, list) and isinstance(new, list):
        if len(old) != len(new):
            out.append(
                Difference("response", ".".join(path) + ".length", len(old), len(new))
            )
        for i, (o, n) in enumerate(zip(old, new)):
            diff_json(o, n, path + [str(i)], out)
        return
    if norm_value(old) != norm_value(new):
        out.append(Difference("response", ".".join(path) or "$", norm_value(old), norm_value(new)))


def diff_responses(
    old_status: int,
    old_body: str | None,
    new_status: int,
    new_body: str | None,
    noise_paths: list[str],
    old_headers: dict[str, str] | None = None,
    new_headers: dict[str, str] | None = None,
    compare_headers: list[str] | None = None,
) -> list[Difference]:
    diffs: list[Difference] = []
    if old_status != new_status:
        diffs.append(Difference("response", "status", old_status, new_status))

    for header in compare_headers or []:
        ov = (old_headers or {}).get(header)
        nv = (new_headers or {}).get(header)
        if ov != nv:
            diffs.append(Difference("response", f"header.{header}", ov, nv))

    old_parsed, new_parsed = _try_json(old_body), _try_json(new_body)
    if isinstance(old_parsed, (dict, list)) or isinstance(new_parsed, (dict, list)):
        old_clean = strip_noise(old_parsed, noise_paths)
        new_clean = strip_noise(new_parsed, noise_paths)
        diff_json(old_clean, new_clean, [], diffs)
    elif (old_body or "") != (new_body or ""):
        diffs.append(Difference("response", "body", old_body, new_body))
    return diffs


# --------------------------------------------------------------------------
# write-set diff
# --------------------------------------------------------------------------


def _col_is_noise(table: str, column: str, patterns: list[str]) -> bool:
    bare_table = table.split(".")[-1]
    for pat in patterns:
        pt, _, pc = pat.partition(".")
        if (pt in ("*", bare_table, table)) and (pc in ("*", column)):
            return True
    return False


def normalize_writeset(
    changes: list[RowChange],
    noise_columns: list[str],
    ignore_tables: list[str],
) -> list[dict]:
    """Canonical, order-insensitive representation of a request's write-set.

    UPDATE keeps only genuinely-changed, non-noise columns (before -> after).
    Rows are keyed by their `id` column when present so old/new can be paired.
    """
    out: list[dict] = []
    for ch in changes:
        bare_table = ch.table.split(".")[-1]
        if bare_table in ignore_tables or ch.table in ignore_tables:
            continue

        def clean(row: dict | None, *, table: str = ch.table) -> dict | None:
            if row is None:
                return None
            return {
                k: norm_value(v)
                for k, v in row.items()
                if not _col_is_noise(table, k, noise_columns)
            }

        entry: dict = {"table": bare_table, "op": ch.op}
        pk_source = ch.after if ch.op != "DELETE" else ch.before
        entry["pk"] = norm_value((pk_source or {}).get("id"))

        if ch.op == "UPDATE":
            before, after = clean(ch.before) or {}, clean(ch.after) or {}
            changed = {
                k: {"before": before.get(k), "after": after.get(k)}
                for k in sorted(set(before) | set(after))
                if before.get(k) != after.get(k)
            }
            if not changed:  # only noise columns changed -> drop entirely
                continue
            entry["changed"] = changed
        elif ch.op == "INSERT":
            entry["values"] = clean(ch.after)
        else:  # DELETE
            entry["values"] = clean(ch.before)
        out.append(entry)

    out.sort(key=lambda e: json.dumps(e, sort_keys=True, default=str))
    return out


def diff_writesets(old_ws: list[dict], new_ws: list[dict]) -> list[Difference]:
    """Compare canonical write-sets. Pair by (table, op, pk) where possible."""
    diffs: list[Difference] = []

    def key(e: dict) -> tuple:
        return (e["table"], e["op"], json.dumps(e.get("pk"), default=str))

    old_by_key: dict[tuple, list[dict]] = {}
    for e in old_ws:
        old_by_key.setdefault(key(e), []).append(e)
    new_by_key: dict[tuple, list[dict]] = {}
    for e in new_ws:
        new_by_key.setdefault(key(e), []).append(e)

    for k in sorted(set(old_by_key) | set(new_by_key), key=str):
        olds, news = old_by_key.get(k, []), new_by_key.get(k, [])
        label = f"{k[0]}[{k[1]} pk={k[2]}]"
        for o, n in zip(olds, news):
            payload_o = o.get("changed") or o.get("values")
            payload_n = n.get("changed") or n.get("values")
            if payload_o != payload_n:
                _diff_payload(label, payload_o, payload_n, diffs)
        for extra in olds[len(news):]:
            diffs.append(
                Difference("writeset", label, extra.get("changed") or extra.get("values"), "<absent>")
            )
        for extra in news[len(olds):]:
            diffs.append(
                Difference("writeset", label, "<absent>", extra.get("changed") or extra.get("values"))
            )
    return diffs


def _diff_payload(label: str, old: dict | None, new: dict | None, out: list[Difference]) -> None:
    old, new = old or {}, new or {}
    for col in sorted(set(old) | set(new)):
        if old.get(col) != new.get(col):
            out.append(
                Difference("writeset", f"{label}.{col}", old.get(col, "<absent>"), new.get(col, "<absent>"))
            )


def verdict_of(response_diffs: list[Difference], writeset_diffs: list[Difference]) -> str:
    if response_diffs and writeset_diffs:
        return BOTH_DIFF
    if response_diffs:
        return RESPONSE_DIFF
    if writeset_diffs:
        return WRITESET_DIFF
    return MATCH
