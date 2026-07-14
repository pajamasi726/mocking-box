# mocking-box 설계 문서 (v0.1)

> 작성일: 2026-07-14 · 선행 문서: [01-differential-testing-tools.md](01-differential-testing-tools.md)

## 1. 목표

프레임워크/ORM이 완전히 바뀐 리라이트(legal-care ~70개 MSA → legalcare-renew 단일 서비스, jOOQ→MyBatis)를
**외부 관측점만으로** 검증하는 범용 차등 테스트 박스.

- 판정 기준 = **HTTP 응답 diff + 요청 단위 write-set diff** (SQL 텍스트는 절대 비교하지 않음)
- 대상 앱의 언어/프레임워크 불문 (Spring 아니어도 동작)
- Keploy처럼 독립된 "박스" 형태, docker-compose 한 방 기동

## 2. 핵심 개념: write-set

**write-set = 요청 하나가 DB에 실제로 일으킨 행 변경의 집합.**
"어떤 SQL을 실행했나(명령)"가 아니라 "그 결과 어느 행이 어떻게 바뀌었나(결과)".

```
[rid=42  POST /wallet/3/charge {amount:10000} 의 write-set]
UPDATE wallet          pk(id=3)   balance: 50000 → 60000
INSERT wallet_history  pk(id=901) {wallet_id:3, type:CHARGE, amount:10000, ...}
```

- jOOQ가 `UPDATE ... SET balance = balance + ?` 한 방을 쓰든,
  MyBatis가 `SELECT ... FOR UPDATE` 후 `UPDATE ... SET balance = ?`를 쓰든
  **write-set은 동일** → ORM 차이가 무력화됨.
- 출처는 앱이 아니라 **MySQL ROW binlog** (before/after 행 이미지가 이미 기록됨).
  앱 무침투·언어 불문의 근거.

## 3. 아키텍처

```
                        ┌────────────────────────────────────────────┐
                        │               mocking-box                  │
  corpus (JSONL/HAR) ─▶ │ ┌──────────┐    ┌─────────────────────────┐│
  (GoReplay/Keploy      │ │ Corpus   │──▶│ Replayer (순차 실행)       ││
   캡처 변환 예정)        │ │ Loader   │    │  req N → old → quiesce   ││
                        │ └──────────┘    │  req N → new → quiesce   ││
                        │                 └────┬──────────┬──────────┘│
                        │                      ▼          ▼           │
                        │      ┌────────────────┐  ┌────────────────┐ │
                        │      │ old stack (앱)  │  │ new stack (앱) │ │
                        │      │ MySQL-old      │  │ MySQL-new      │ │
                        │      └───────┬────────┘  └───────┬────────┘ │
                        │              │ binlog(ROW)       │ binlog   │
                        │      ┌───────▼────────────────────▼───────┐ │
                        │      │ BinlogCapture ×2 (replication client)│ │
                        │      │  → 요청 윈도우별 write-set 귀속       │ │
                        │      └───────┬────────────────────────────┘ │
                        │  ┌───────────▼───────────┐ ┌──────────────┐ │
                        │  │ Differ                │ │ Reporter     │ │
                        │  │  response diff (noise │▶│ console/JSON │ │
                        │  │  rule) + writeset diff│ │ 카운터+상세    │ │
                        │  └───────────────────────┘ └──────────────┘ │
                        └────────────────────────────────────────────┘
```

- **양쪽 MySQL은 동일 시드에서 기동** (동일 베이크드 이미지 / 동일 init.sql)
  → auto_increment 카운터까지 일치 → 생성 ID divergence 최소화.
- 박스는 각 MySQL에 replication client로 붙어 binlog 스트림을 상시 수신.

## 4. 요청↔write-set 귀속 (Attribution) — 3단계

| Level | 방식 | 앱 침투 | 비동기 정확도 |
|---|---|---|---|
| **0** (MVP 기본) | 순차 리플레이 + quiesce 윈도우: 요청 전송 후 `innodb_trx` 비어있음 + binlog가 T ms 조용해질 때까지의 이벤트를 해당 요청에 귀속 | 없음 (완전 범용) | 시간 윈도우 기반 — fire-and-forget 비동기는 다음 구간으로 샐 수 있음 |
| **1** | OTel/SQLCommenter 표준으로 SQL 주석에 trace-id 주입 → `binlog_rows_query_log_events=ON`이면 Rows_query 이벤트로 회수 | 계측 에이전트/라이브러리 (비즈니스 코드 무수정, 크로스-언어) | 태그 기반 — 정확 |
| **2** | 프레임워크별 어댑터 (Spring datasource-proxy 등) | 설정 bean 1개 | 태그 기반 — 정확 |

MVP는 Level 0으로 동작하되, Rows_query 이벤트가 오면 쿼리 텍스트를 write-set에 함께 기록해
Level 1로의 승격(rid 파싱)이 코드 몇 줄이 되도록 준비해 둔다.

- quiesce 판정: (a) binlog 이벤트가 `quiet_ms`(기본 300ms) 동안 없음, (b) `information_schema.innodb_trx` 비어 있음, (c) 상한 `timeout_ms`(기본 5000ms).
- binlog는 **커밋 시점에만** 이벤트가 나오므로 (b)가 인플라이트 트랜잭션을 커버.
- 안전망: 시나리오(코퍼스) 종료 시 전체 테이블 정규화 해시 비교(v0.2)로 귀속 누수 검출.

## 5. Diff 규칙

### 5.1 응답 diff
- 비교 대상: status code + body. (헤더는 기본 제외 — 노이즈)
- body가 JSON이면 파싱 후 딥 비교, 아니면 문자열 비교.
- 노이즈 경로 제외: 설정의 dotted-path 패턴 (`data.updated_at`, `**.trace_id`, `*` 와일드카드).

### 5.2 write-set diff
- 이벤트 정규화: `(table, op, pk, values)` — UPDATE는 **변경된 컬럼만** before→after로.
- 노이즈 컬럼 제외: `*.created_at`, `*.updated_at` 등 설정 기반.
- **순서 무관 비교**: 요청 내 정렬 키 `(table, op, pk)`로 canonical sort 후 비교
  (ORM별 실행 순서 차이를 무시하기 위함).
- 판정: `MATCH` / `RESPONSE_DIFF` / `WRITESET_DIFF` / `BOTH_DIFF` / `ERROR`.

### 5.3 노이즈 자동 학습 (v0.2)
같은 코퍼스를 old 스택에만 2회 실행 → 자기 자신과 불일치하는 필드/컬럼을 수집 → 노이즈 규칙 초안 자동 생성 (Diffy의 primary/secondary 아이디어 축약).

## 6. 데이터 포맷

### corpus (JSONL, 1줄 = 1요청)
```json
{"name": "charge-5000", "method": "POST", "path": "/wallet/3/charge",
 "headers": {"content-type": "application/json"}, "body": {"amount": 5000}}
```
HAR 로더 지원(브라우저/프록시 캡처 직결). Keploy YAML·GoReplay .gor 변환기는 v0.2.

### config.yaml
```yaml
old: { base_url: "http://app-old:8080",
       mysql: { host: mysql-old, port: 3306, user: root, password: root } }
new: { base_url: "http://app-new:8080",
       mysql: { host: mysql-new, port: 3306, user: root, password: root } }
attribution: { quiet_ms: 300, timeout_ms: 5000 }
noise:
  response_paths: ["**.updated_at", "**.created_at", "**.trace_id"]
  columns: ["*.created_at", "*.updated_at"]
  tables_ignore: ["_replay_marker"]
report: { dir: "./report" }
```

## 7. 구현 선택

- **Python 3.12+ 단일 패키지** (`mockingbox/`): httpx(요청), mysql-replication(binlog), PyMySQL(quiesce 체크), PyYAML.
  선택 이유: binlog replication client 라이브러리가 성숙(pymysqlreplication — Debezium과 동일 개념),
  코드가 짧아 유지보수 용이, 컨테이너로 배포하므로 런타임 무게 무관.
  (추후 트래픽 많아지면 Go 포팅 여지 — 아키텍처는 동일)
- MySQL 요구 설정: `binlog_format=ROW`(8.x 기본), `binlog_row_image=FULL`(기본),
  `binlog_row_metadata=FULL`(컬럼명 확보), `binlog_rows_query_log_events=ON`(Level 1 대비).
- binlog 계정 권한: `REPLICATION SLAVE, REPLICATION CLIENT` (+ quiesce용 `PROCESS`).

## 8. 데모 (개념 증명, demo/)

- `mysql-old`/`mysql-new`: 동일 init.sql (wallet, wallet_history 시드)
- `app-old`: 레거시 흉내 — `UPDATE wallet SET balance = balance + ?` 스타일
- `app-new`: 리뉴얼 흉내 — `SELECT ... FOR UPDATE` 후 계산값 UPDATE (SQL 모양 완전 다름)
  + **의도적 버그**: withdraw 시 wallet_history INSERT 누락
- 기대 결과: charge/GET은 SQL이 달라도 `MATCH`, withdraw는 `WRITESET_DIFF`로 검출
  → "쿼리가 달라도 통과, 행동이 다르면 검출"의 증명.

## 9. 로드맵

| 버전 | 내용 |
|---|---|
| v0.1 (MVP) | corpus 순차 리플레이, Level 0 귀속, 응답+write-set diff, console/JSON 리포트, 데모 |
| v0.2 | 노이즈 자동 학습, 시나리오 종료 상태 해시 안전망, Keploy YAML/GoReplay 변환기, HTML 리포트 |
| v0.3 | Level 1(rid/SQLCommenter) 귀속, 생성 ID 매핑 정규화, ID 치환 세션(`{{last_created_id}}`) |
| v0.4 | Debezium embedded 리더로 DB 범용화(Postgres 등), 병렬 시나리오 |

## 10. 한계 (명시)

- Level 0 귀속은 fire-and-forget 비동기(스케줄러, 큐 컨슈머)의 늦은 커밋을 다음 요청 구간으로 오귀속할 수 있음 → quiet_ms 조정 + v0.2 상태 해시 안전망으로 보완.
- DDL, 외부 API 부수효과(메일 발송 등)는 write-set 범위 밖 — 외부 콜은 아웃바운드 mock/기록(별도 과제).
- 리플레이는 순차 단일 스레드 — 처리량보다 정합성 우선. 동시성 버그는 이 툴의 목표 아님.
