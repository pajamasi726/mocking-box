#!/usr/bin/env python3
"""
sweep — 코퍼스(엔드포인트 목록)를 tee(:10099)로 전수 발사하고 verdict 요약.
경로변수 치환({storeId}→229 등), 인증 토큰 자동 첨부. tee 가 diff 를 report 에 기록.

  python3 sweep.py corpus.jsonl
corpus 한 줄: {"method":"GET","path":"/x/{storeId}","auth":true,"body":{...}}
"""
import json
import os
import sys
import time
import urllib.request
import urllib.error

TEE = os.environ.get("TEE", "http://localhost:10099")
HOSTH = os.environ.get("UPSTREAM_HOST", "gateway.dev.revieworks.com")
TOKEN = ""
tf = "/tmp/claude-501/steve.token"
if os.path.exists(tf):
    TOKEN = open(tf).read().strip()

# 경로변수 → 알려진 실제 값(steve 계정 기준)
PARAMS = {
    "storeId": "229", "appStoreId": "229", "store-id": "229",
    "organizationId": "01JVNQRK6X043XX0J4621XXMVS", "orgId": "01JVNQRK6X043XX0J4621XXMVS",
    "appUserId": "836", "userId": "836",
    "teamId": "01JVNQRP00HFKRFQJHTP7WM645", "team-id": "01JVNQRP00HFKRFQJHTP7WM645",
    "productType": "REVIEW_BOOST",
}


def subst(path):
    missing = []
    out = path
    import re
    for m in re.findall(r"\{([^}]+)\}", path):
        if m in PARAMS:
            out = out.replace("{" + m + "}", PARAMS[m])
        else:
            missing.append(m)
    return out, missing


def fire(method, path, body, auth):
    url = TEE + path
    data = None
    headers = {"Host": HOSTH, "X-Language": "ko"}
    if auth and TOKEN:
        headers["Authorization"] = TOKEN
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, method=method)
    for k, v in headers.items():
        req.add_header(k, v)
    try:
        with urllib.request.urlopen(req, timeout=25) as r:
            return r.status
    except urllib.error.HTTPError as e:
        return e.code
    except Exception as e:
        return f"ERR:{e}"


def main():
    corpus = sys.argv[1]
    skipped = []
    n = 0
    for line in open(corpus):
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        c = json.loads(line)
        method = c.get("method", "GET").upper()
        path, missing = subst(c["path"])
        if missing:
            skipped.append((method, c["path"], missing))
            continue
        fire(method, path, c.get("body"), c.get("auth", True))
        n += 1
        time.sleep(0.05)
    print(f"fired {n} requests, skipped {len(skipped)} (missing params)")
    for m, p, miss in skipped:
        print(f"  SKIP {m} {p}  missing={miss}")


if __name__ == "__main__":
    main()
