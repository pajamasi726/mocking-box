"""Reporting: console summary + machine-readable JSON report."""

from __future__ import annotations

import datetime
import json
from collections import Counter
from pathlib import Path

from .diff import BOTH_DIFF, ERROR, MATCH, RESPONSE_DIFF, WRITESET_DIFF, RequestResult

_GREEN, _RED, _YELLOW, _DIM, _RESET = (
    "\033[32m",
    "\033[31m",
    "\033[33m",
    "\033[2m",
    "\033[0m",
)

_COLORS = {
    MATCH: _GREEN,
    RESPONSE_DIFF: _RED,
    WRITESET_DIFF: _RED,
    BOTH_DIFF: _RED,
    ERROR: _YELLOW,
}


def print_console(results: list[RequestResult]) -> None:
    name_w = max([len(r.name) for r in results] + [4])
    print()
    print(f"{'#':>3}  {'name':<{name_w}}  {'verdict':<14} {'writes':<9} detail")
    print("─" * (name_w + 60))
    for i, r in enumerate(results, start=1):
        color = _COLORS.get(r.verdict, "")
        writes = f"{r.old_writes}/{r.new_writes}"
        detail = ""
        if r.error:
            detail = r.error
        elif r.differences:
            first = r.differences[0]
            detail = f"{first.path}: {_short(first.old)} → {_short(first.new)}"
            if len(r.differences) > 1:
                detail += f"  (+{len(r.differences) - 1} more)"
        print(
            f"{i:>3}  {r.name:<{name_w}}  {color}{r.verdict:<14}{_RESET}"
            f" {writes:<9} {_DIM}{detail}{_RESET}"
        )

    counts = Counter(r.verdict for r in results)
    print("─" * (name_w + 60))
    summary = "  ".join(
        f"{_COLORS.get(v, '')}{v}={counts[v]}{_RESET}"
        for v in (MATCH, RESPONSE_DIFF, WRITESET_DIFF, BOTH_DIFF, ERROR)
        if counts[v]
    )
    ok = counts[MATCH] == len(results)
    print(f"Total {len(results)}   {summary}")
    print(("✅ old and new behave identically" if ok else "❌ divergence detected"))
    print()


def _short(v: object, limit: int = 60) -> str:
    s = json.dumps(v, ensure_ascii=False, default=str) if not isinstance(v, str) else v
    return s if len(s) <= limit else s[: limit - 1] + "…"


def write_json(results: list[RequestResult], report_dir: Path) -> Path:
    report_dir.mkdir(parents=True, exist_ok=True)
    stamp = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
    path = report_dir / f"report-{stamp}.json"
    payload = {
        "generated_at": datetime.datetime.now().isoformat(),
        "summary": dict(Counter(r.verdict for r in results)),
        "results": [
            {
                "name": r.name,
                "request": r.request,
                "verdict": r.verdict,
                "old_status": r.old_status,
                "new_status": r.new_status,
                "error": r.error,
                "differences": [d.as_dict() for d in r.differences],
                "old_writeset": r.old_writeset,
                "new_writeset": r.new_writeset,
            }
            for r in results
        ],
    }
    path.write_text(json.dumps(payload, indent=2, ensure_ascii=False, default=str))
    return path
