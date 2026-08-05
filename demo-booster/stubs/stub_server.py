#!/usr/bin/env python3
"""local-stack 만능 스텁 — 외부API·GW 타깃(legacy/lawkit-admin) 대역.
모든 요청을 stdout 로깅하고, 경로별 최소 응답 또는 요청 에코(JSON)를 돌려준다.
에코에 요청 헤더 전체가 실리므로 GW 의 X-Auth-* 주입/스푸핑-스트립 검증에 쓴다."""
import json
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

# 경로 startswith → 고정 응답. 나머지는 전부 에코.
CANNED = {
    # 공공데이터포털 공휴일 API (batch holidayFetch) — 빈 items 정상형
    "/B090041": {"response": {"header": {"resultCode": "00", "resultMsg": "NORMAL SERVICE."},
                              "body": {"items": "", "numOfRows": 100, "pageNo": 1, "totalCount": 0}}},
    # 카카오 선물 비즈 (batch boostPresentStatus / booster product)
    "/giftbiz": {"code": "OK", "orders": []},
    # NCP SENS 알림톡/SMS (dry-run 이라 보통 도달 안 함 — 안전망)
    "/alimtalk": {"statusCode": "202", "statusName": "success"},
    "/sms": {"statusCode": "202", "statusName": "success"},
    # reviewed search 부팅 사전(T5 하드닝 후엔 lazy — 남겨두면 무해)
    "/internal/search/hospital-keyword-dictionary": {"data": []},
    # Toss (로컬에선 결제 플로우 미검증 — 도달 시 형태만)
    "/toss": {"mId": "stub", "status": "DONE"},
}


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def _respond(self):
        length = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(length) if length else b""
        for prefix, resp in CANNED.items():
            if self.path.startswith(prefix):
                payload = resp
                break
        else:
            payload = {"stub": True, "method": self.command, "path": self.path,
                       "headers": dict(self.headers.items()),
                       "body": body.decode("utf-8", "replace")[:2000]}
        data = json.dumps(payload, ensure_ascii=False).encode()
        print(f"{self.command} {self.path}", flush=True)
        self.send_response(200)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    do_GET = do_POST = do_PUT = do_DELETE = do_PATCH = do_OPTIONS = _respond

    def log_message(self, *args):
        pass  # 요청 로깅은 _respond 에서 한 줄만


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8099
    print(f"stub listening :{port}", flush=True)
    ThreadingHTTPServer(("0.0.0.0", port), Handler).serve_forever()
