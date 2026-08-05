#!/usr/bin/env python3
"""
edge_teediff — 앞단(edge) dual-fire 차등 프록시.

브라우저 → 이 프록시 → 구(OLD)·신(NEW) 양쪽에 같은 요청을 2벌로 tee →
응답(status+JSON body)을 노이즈 필터 걸어 diff → 콘솔+jsonl 리포트.
브라우저에는 NEW(renew) 응답을 그대로 반환(CORS 헤더 포함 pass-through).

용도: legalcare renew 검증. OLD=개발계 구 게이트웨이, NEW=renew localdev. 둘 다 같은 .55 DB.
읽기는 깨끗하게 비교됨. 쓰기는 공유 DB라 2벌 반영(개발계라 무해) — OTP 소비처럼 상태공유
엔드포인트는 위양성(예: 한쪽 200 / 한쪽 이미소비) 날 수 있으니 리포트에서 감안.

  OLD=http://192.168.1.55:10000 NEW=http://127.0.0.1:10000 PORT=10099 \
  UPSTREAM_HOST=gateway.dev.revieworks.com python3 edge_teediff.py
"""
import gzip
import io
import json
import os
import sys
import threading
import time
import urllib.request
import urllib.error
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

OLD = os.environ.get("OLD", "http://192.168.1.55:10000").rstrip("/")
NEW = os.environ.get("NEW", "http://127.0.0.1:10000").rstrip("/")
PORT = int(os.environ.get("PORT", "10099"))
# 업스트림에 실어보낼 Host 헤더(게이트웨이가 호스트 접미사로 라우팅). 비우면 원본 Host 유지.
UPSTREAM_HOST = os.environ.get("UPSTREAM_HOST", "gateway.dev.revieworks.com")
REPORT = os.environ.get("REPORT", os.path.join(os.path.dirname(__file__), "teediff-report.jsonl"))
SERVE = os.environ.get("SERVE", "new").lower()  # 브라우저에 반환할 쪽: new(기본) | old
# 상태공유(같은 redis/DB에 쓰는) 인증류는 dual-fire 시 서로 덮어써 위양성·이중문자 유발 →
# NEW 만 쏘고 diff 스킵. 경로에 이 부분문자열이 있으면 single-fire.
SINGLE_FIRE = [s for s in os.environ.get(
    "SINGLE_FIRE", "send-otp,sign-in,sign-up,/refresh,reissue").split(",") if s]

HOP = {"connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
       "te", "trailer", "transfer-encoding", "upgrade", "content-length", "host"}

# diff 에서 제외/정규화할 JSON 키(소문자, 부분일치 아님 — 정확일치). 시간·토큰·추적자.
NOISE_KEYS = {
    "accesstoken", "refreshtoken", "token", "expiredate", "refreshduration",
    "createdat", "updatedat", "created_at", "updated_at", "modifiedat", "modified_at",
    "traceid", "trace_id", "timestamp", "iat", "exp", "serverdatetime",
    "requestid", "request_id", "now", "date", "issuedat", "expiresat", "expires_at",
}

_report_lock = threading.Lock()
_stats = {"total": 0, "match": 0, "order": 0, "diff": 0, "err": 0}


def _decode_body(raw, headers):
    if raw and headers.get("content-encoding", "").lower() == "gzip":
        try:
            raw = gzip.decompress(raw)
        except Exception:
            pass
    return raw


def _scrub(obj):
    """noise 키 제거하고 재귀 정규화."""
    if isinstance(obj, dict):
        return {k: _scrub(v) for k, v in obj.items() if k.lower() not in NOISE_KEYS}
    if isinstance(obj, list):
        return [_scrub(x) for x in obj]
    return obj


def _json_or_none(raw, headers):
    ctype = headers.get("content-type", "")
    if "json" not in ctype.lower():
        return None
    try:
        return json.loads(raw.decode("utf-8"))
    except Exception:
        return None


def _canon(obj):
    """멀티셋 비교용 안정 정규형(키 정렬)."""
    return json.dumps(obj, sort_keys=True, ensure_ascii=False)


def _multiset_eq(a, b):
    return sorted(_canon(x) for x in a) == sorted(_canon(x) for x in b)


def _diff_paths(a, b, path="$", val=None, order=None, limit=25):
    """(a=NEW, b=OLD) 비교. val=진짜 값차이, order=순서만 다른 배열 노트."""
    if val is None:
        val, order = [], []
    if len(val) >= limit:
        return val, order
    if type(a) is not type(b):
        val.append(f"{path}: type {type(a).__name__}(new)≠{type(b).__name__}(old)")
        return val, order
    if isinstance(a, dict):
        for k in sorted(set(a) | set(b)):
            if k not in a:
                val.append(f"{path}.{k}: OLD에만 있음")
            elif k not in b:
                val.append(f"{path}.{k}: NEW에만 있음")
            else:
                _diff_paths(a[k], b[k], f"{path}.{k}", val, order, limit)
    elif isinstance(a, list):
        if len(a) != len(b):
            val.append(f"{path}: len {len(a)}(new)≠{len(b)}(old)")
        elif a != b and _multiset_eq(a, b):
            # 원소 집합은 같고 순서만 다름 → 순서 노트(진짜 값차이 아님)
            order.append(f"{path}: 순서만 다름 (원소 {len(a)}개 동일)")
        else:
            for i in range(min(len(a), len(b))):
                _diff_paths(a[i], b[i], f"{path}[{i}]", val, order, limit)
    elif a != b:
        val.append(f"{path}: {repr(b)[:60]}(old) → {repr(a)[:60]}(new)")
    return val, order


def _fire(base, method, path, headers, body):
    url = base + path
    req_headers = {k: v for k, v in headers.items() if k.lower() not in HOP
                   and k.lower() != "accept-encoding"}
    if UPSTREAM_HOST:
        req_headers["Host"] = UPSTREAM_HOST
    req_headers["Accept-Encoding"] = "identity"
    req = urllib.request.Request(url, data=body if body else None, method=method)
    for k, v in req_headers.items():
        req.add_header(k, v)
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            raw = r.read()
            return {"status": r.status, "headers": dict(r.headers), "body": raw,
                    "ms": int((time.time() - t0) * 1000)}
    except urllib.error.HTTPError as e:
        raw = e.read()
        return {"status": e.code, "headers": dict(e.headers), "body": raw,
                "ms": int((time.time() - t0) * 1000)}
    except Exception as e:
        return {"error": str(e), "ms": int((time.time() - t0) * 1000)}


def _compare_and_log(method, path, old_r, new_r):
    _stats["total"] += 1
    verdict, detail = "MATCH", []
    if old_r.get("error") or new_r.get("error"):
        verdict = "ERROR"
        detail = [f"OLD={old_r.get('error','ok')} NEW={new_r.get('error','ok')}"]
        _stats["err"] += 1
    else:
        st_old, st_new = old_r["status"], new_r["status"]
        ob = _decode_body(old_r["body"], {k.lower(): v for k, v in old_r["headers"].items()})
        nb = _decode_body(new_r["body"], {k.lower(): v for k, v in new_r["headers"].items()})
        oh = {k.lower(): v for k, v in old_r["headers"].items()}
        nh = {k.lower(): v for k, v in new_r["headers"].items()}
        oj = _json_or_none(ob, oh)
        nj = _json_or_none(nb, nh)
        if st_old != st_new:
            verdict = "STATUS_DIFF"
            detail.append(f"status {st_old}(old) ≠ {st_new}(new)")
        if oj is not None and nj is not None:
            val, order = _diff_paths(_scrub(nj), _scrub(oj))  # (new, old)
            if val:
                verdict = "BODY_DIFF" if verdict != "STATUS_DIFF" else verdict
                detail += val
            elif order and verdict == "MATCH":
                verdict = "ORDER_DIFF"
            if order:
                detail += order
        elif ob != nb:
            if verdict == "MATCH":
                verdict = "BODY_DIFF"
            detail.append(f"non-json body len {len(ob)}(old) vs {len(nb)}(new)")
        if verdict == "MATCH":
            _stats["match"] += 1
        elif verdict == "ORDER_DIFF":
            _stats["order"] += 1
        else:
            _stats["diff"] += 1

    icon = {"MATCH": "✅", "ORDER_DIFF": "🔀", "BODY_DIFF": "⚠️",
            "STATUS_DIFF": "🔴", "ERROR": "💥"}.get(verdict, "?")
    line = f"{icon} {verdict:11} {method:6} {path[:70]}  old={old_r.get('status','-')}/{old_r.get('ms','-')}ms new={new_r.get('status','-')}/{new_r.get('ms','-')}ms"
    print(line, flush=True)
    if detail and verdict != "MATCH":
        for d in detail[:12]:
            print(f"      · {d}", flush=True)
    with _report_lock:
        with open(REPORT, "a") as f:
            f.write(json.dumps({
                "ts": int(time.time()), "verdict": verdict, "method": method, "path": path,
                "old_status": old_r.get("status"), "new_status": new_r.get("status"),
                "old_ms": old_r.get("ms"), "new_ms": new_r.get("ms"),
                "detail": detail,
            }, ensure_ascii=False) + "\n")


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *a):  # 자체 로깅 억제
        pass

    def _handle(self):
        method = self.command
        path = self.path
        length = int(self.headers.get("Content-Length", 0) or 0)
        body = self.rfile.read(length) if length else b""
        headers = {k: v for k, v in self.headers.items()}
        primary = NEW if SERVE == "new" else OLD

        # 상태공유 인증류: single-fire(NEW만), diff 스킵.
        if any(s in path for s in SINGLE_FIRE):
            served = _fire(primary, method, path, headers, body)
            print(f"➡️  SINGLE      {method:6} {path[:70]}  "
                  f"{SERVE}={served.get('status','-')}/{served.get('ms','-')}ms", flush=True)
            self._respond(served, method)
            return

        # OLD 는 백그라운드, primary 는 여기서 대기(브라우저엔 primary 반환).
        results = {}
        def run_old():
            results["old"] = _fire(OLD, method, path, headers, body)
        to = threading.Thread(target=run_old, daemon=True)
        to.start()
        new_r = _fire(NEW, method, path, headers, body)
        results["new"] = new_r
        to.join(timeout=31)
        old_r = results.get("old", {"error": "timeout"})

        threading.Thread(target=_compare_and_log,
                         args=(method, path, old_r, new_r), daemon=True).start()
        self._respond(new_r if SERVE == "new" else old_r, method)

    def _respond(self, served, method):
        if served.get("error"):
            self.send_response(502)
            self.send_header("Content-Type", "application/json")
            msg = json.dumps({"teediff_error": served["error"], "which": SERVE}).encode()
            self.send_header("Content-Length", str(len(msg)))
            self.end_headers()
            self.wfile.write(msg)
            return
        raw = _decode_body(served["body"], {k.lower(): v for k, v in served["headers"].items()})
        self.send_response(served["status"])
        for k, v in served["headers"].items():
            if k.lower() in HOP or k.lower() == "content-encoding":
                continue
            self.send_header(k, v)
        self.send_header("Content-Length", str(len(raw)))
        self.send_header("X-TeeDiff-Served", SERVE)
        self.end_headers()
        if method != "HEAD":
            self.wfile.write(raw)

    do_GET = do_POST = do_PUT = do_DELETE = do_PATCH = do_OPTIONS = do_HEAD = _handle


def main():
    print(f"edge_teediff  listen :{PORT}\n  OLD={OLD}\n  NEW={NEW}  (serve={SERVE})\n"
          f"  UPSTREAM Host={UPSTREAM_HOST}\n  report={REPORT}\n"
          f"  프론트: NEXT_PUBLIC_BASE_URL=http://gateway.dev.revieworks.com:{PORT}\n", flush=True)
    srv = ThreadingHTTPServer(("0.0.0.0", PORT), Handler)
    try:
        srv.serve_forever()
    except KeyboardInterrupt:
        print(f"\n집계: {_stats}", flush=True)


if __name__ == "__main__":
    main()
