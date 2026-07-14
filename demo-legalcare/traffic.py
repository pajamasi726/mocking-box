"""운영 트래픽 시뮬레이터 — 워커별 user_id 파티셔닝(동시성에도 상태 재현성 보존).

각 워커는 자기 user_id의 계약만 쓰기(생성/서명/취소)하고, 읽기는 자유.
게이트웨이(:10000, mirrorsim)를 통과하므로 전 트래픽이 VXLAN으로 미러링된다.
"""

import json
import random
import sys
import threading
import time
import urllib.request

GATEWAY = "http://localhost:10000"
DURATION_S = float(sys.argv[1]) if len(sys.argv) > 1 else 20
WORKERS = 4  # user_id 1..4
SEED_IDS = {1: [1, 2], 2: [3, 4], 3: [5, 6], 4: [7, 8]}

counts = {"total": 0, "errors": 0}
lock = threading.Lock()


def call(method, path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(GATEWAY + path, data=data, method=method)
    if data:
        req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            resp.read()
            ok = True
    except urllib.error.HTTPError as e:
        e.read()
        ok = True  # 4xx도 유효한 시나리오 (409 등)
    except Exception:
        ok = False
    with lock:
        counts["total"] += 1
        if not ok:
            counts["errors"] += 1


def worker(user_id):
    rng = random.Random(user_id * 7919)  # 워커별 결정적 시드
    my_ids = list(SEED_IDS[user_id])
    deadline = time.time() + DURATION_S
    while time.time() < deadline:
        r = rng.random()
        if r < 0.55 and my_ids:  # 단건 조회
            call("GET", f"/contract/{rng.choice(my_ids)}")
        elif r < 0.75:  # 목록 조회
            call("GET", f"/contracts?user_id={user_id}")
        elif r < 0.87:  # 계약 생성
            call("POST", "/contract", {
                "user_id": user_id,
                "title": f"자문계약-{user_id}-{len(my_ids)}",
                "amount": rng.choice([200000, 500000, 1000000]),
            })
            # 생성 id는 시드 8건 이후 순차 — 정확한 id 추적 대신 서명/취소는 시드 계약에만
        elif r < 0.94 and my_ids:  # 서명
            call("POST", f"/contract/{rng.choice(my_ids)}/sign")
        elif my_ids:  # 취소
            call("POST", f"/contract/{rng.choice(my_ids)}/cancel")
        time.sleep(rng.uniform(0.03, 0.12))


threads = [threading.Thread(target=worker, args=(uid,)) for uid in SEED_IDS]
start = time.time()
for t in threads:
    t.start()
for t in threads:
    t.join()
print(f"done: {counts['total']} requests in {time.time()-start:.1f}s, errors={counts['errors']}")
