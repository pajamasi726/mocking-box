# demo-legalcare — 리걸케어 축소판 시뮬레이션

실제 리걸케어 인프라를 본뜬 로컬 검증 시나리오:

| 실제 | 시뮬레이션 |
|---|---|
| Spring Cloud Gateway :10000 | mirrorsim :10000 (게이트웨이 홉 + VPC 미러링 대역) |
| lawkit-contract :10103 (jOOQ) | app-old :10103 (상대 UPDATE, ORDER BY ASC) |
| legalcare-renew (MyBatis) | app-new :10104 (FOR UPDATE + 절대 UPDATE, 버그 3종 내장) |
| RDS MySQL **8.4.7** | mysql:**8.4** ×2 (`SHOW BINARY LOG STATUS` 경로 검증) |
| AWS VPC Traffic Mirroring | mirrorsim의 VXLAN 복제 → `mockingbox mirror` |
| 운영 동시 트래픽 | traffic.py (워커 4개, 유저별 파티셔닝, ~1100 req/25s) |

신스택에 심어둔 것: ① sign 시 `contract_history` INSERT 누락(응답 동일 — write-set만 검출 가능)
② cancel 응답 철자 `CANCELED`(응답 diff 검출) ③ 목록 ORDER BY DESC(매퍼 차이 — sort_arrays로 흡수).

## 실행 순서

```bash
go build -o bin/mockingbox ./cmd/mockingbox && go build -o bin/mirrorsim ./cmd/mirrorsim
cd demo-legalcare && docker compose up -d --build

# ① 미러 수신기 (읽기 30% 샘플링, 쓰기 전수)
../bin/mockingbox mirror --listen 127.0.0.1:14789 --port 10103 --sample 0.3 \
  --out corpus/legalcare-sim.golden.jsonl --duration 40s &

# ② 게이트웨이(VPC 미러링 시뮬레이터): :10000 → old, VXLAN 사본 → :14789
../bin/mirrorsim --listen :10000 --upstream 127.0.0.1:10103 --mirror 127.0.0.1:14789 &

# ③ 운영 트래픽 (동시 4워커 25초)
python3 traffic.py 25
wait  # 미러 수신 종료

# ④ 검증 — 신서버(T0 시드 상태)에만 재생
../bin/mockingbox verify -c config.yaml --golden corpus/legalcare-sim.golden.jsonl
```

## 실측 결과 (2026-07-14)

- 트래픽 1,099건 → 골든 534건 (읽기 30% 샘플 + 쓰기 전수)
- 1차(정렬 규칙 off): MATCH 455 / DIFF 79 — 75건이 목록 정렬(③)
- 2차(`sort_arrays: [{path: contracts, by: contract_id}]` on):
  **MATCH 526 / DIFF 8 (98.5%)** — cancel 철자 버그(②) 2건 + 동시 create 1쌍의 id 스왑 6건
- 3차(직렬화 프록시 골든, write-set 포함): sign 2건 **WRITESET_DIFF**
  (`contract_history[INSERT] → <absent>`) — 패시브로 안 보이던 버그(①) 검출

## 확인된 것 / 한계

- ✅ MySQL 8.4 binlog 경로, VXLAN 수신, 동시 트래픽 하의 요청 도착순 정렬(SYN 필수 — mirrorsim이 핸드셰이크까지 합성), 샘플링(쓰기 전수 원칙)
- ⚠️ 동시 create의 캡처순≠커밋순 레이스 → id 스왑 소량(6/534) — 생성 ID 매핑 정규화(로드맵)로 해소 예정
- ⚠️ 패시브 골든은 write-set 없음(동시 트래픽 귀속 불가) — write 버그는 직렬화 프록시 골든 또는 종료 후 상태 해시(로드맵)로
