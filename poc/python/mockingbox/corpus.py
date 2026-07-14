"""Request corpus loading: JSONL (native) and HAR (browser/proxy captures).

JSONL format, one request per line:
    {"name": "charge", "method": "POST", "path": "/wallet/3/charge",
     "headers": {"content-type": "application/json"}, "body": {"amount": 5000}}

`body` may be a JSON object/array (sent as JSON) or a string (sent verbatim).
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from pathlib import Path
from urllib.parse import urlsplit


@dataclass
class RequestSpec:
    name: str
    method: str
    path: str  # path + optional query string, joined onto each stack's base_url
    headers: dict[str, str] = field(default_factory=dict)
    body: object = None  # dict/list -> JSON, str -> raw, None -> no body

    def describe(self) -> str:
        return f"{self.method} {self.path}"


def load_corpus(path: str | Path) -> list[RequestSpec]:
    path = Path(path)
    if path.suffix.lower() == ".har":
        return load_har(path)
    return load_jsonl(path)


def load_jsonl(path: Path) -> list[RequestSpec]:
    specs: list[RequestSpec] = []
    for i, line in enumerate(path.read_text().splitlines(), start=1):
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        raw = json.loads(line)
        specs.append(
            RequestSpec(
                name=raw.get("name") or f"req-{i}",
                method=raw["method"].upper(),
                path=raw["path"],
                headers={k.lower(): str(v) for k, v in (raw.get("headers") or {}).items()},
                body=raw.get("body"),
            )
        )
    return specs


_HAR_SKIP_HEADERS = {"host", "content-length", "cookie", "connection", "accept-encoding"}


def load_har(path: Path) -> list[RequestSpec]:
    har = json.loads(path.read_text())
    specs: list[RequestSpec] = []
    for i, entry in enumerate(har["log"]["entries"], start=1):
        req = entry["request"]
        url = urlsplit(req["url"])
        req_path = url.path + (f"?{url.query}" if url.query else "")
        headers = {
            h["name"].lower(): h["value"]
            for h in req.get("headers", [])
            if h["name"].lower() not in _HAR_SKIP_HEADERS and not h["name"].startswith(":")
        }
        body = None
        post = req.get("postData")
        if post and post.get("text"):
            text = post["text"]
            if "json" in (post.get("mimeType") or ""):
                try:
                    body = json.loads(text)
                except ValueError:
                    body = text
            else:
                body = text
        specs.append(
            RequestSpec(
                name=f"har-{i}-{req['method']}-{url.path}",
                method=req["method"].upper(),
                path=req_path,
                headers=headers,
                body=body,
            )
        )
    return specs
