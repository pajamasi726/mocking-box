# QA 리포트: 크롤러 워커 3역할 통합 (playwright 1.55 상향 + 개발계 신규 기동)

검증자: QA(독립 실행) · 2026-08-09
판정 근거는 **전부 QA 가 직접 돌린 명령의 출력**이다. 작성자·리뷰어 문서는 판정 근거로 쓰지 않았고,
주장이 사실인지 확인하는 대조 대상으로만 썼다.

## 테스트 환경

- 대상 저장소: `/Users/steve/steve/legal-care/crawler-worker` (커밋 `488e829` + 미커밋 변경 7파일 + 미추적 `roles/receipt/`)
- 개발계: `ssh -p 50022 legalcare@114.203.1.178` (DELL / `worksationdell`, x86)
- 이미지: `crawler-worker:dev-3role` (`sha256:f1c10c1d5440`, 3.14GB, 빌드 2026-08-09T09:15:08Z)
- A/B 대조 원본: `legalcare/crawler:dev-20260806-safe`(`433a8a07a602`) · `legalcare/ai-delegate-crawler:dev-20260806-reliable5`(`42b79a73a960`) · `legalcare/pg-receipt-crawler:dev-20260806-safe`(`f8bf1c4519e8`)
- 런타임: Python 3.12.13 / Debian 13 trixie / **playwright 1.55.0** / pydantic 2.12.0 / chromium **140.0.7339.16**
- 브라우저 검증은 전부 `--network none` + `file://`·`data:` 픽스처. **외부 실사이트·실 S3·실 Vault 접속 0건.**
- 카프카는 검증 토픽 `legalcare.test.3role.receipt.*` 만 사용. **실토픽·fixture 실토픽에 발행 0건.**
- QA 가 직접 작성해 실행한 검증 코드: `~/qa-3role/ac1_harness.py`(playwright API 재현 하니스, 24 시나리오).
  프로덕션 코드는 한 줄도 수정하지 않았다.
- 기존 컨테이너 정지·삭제·compose 재생성 **0건**. 일회성 `docker run --rm` 과 QA 전용 임시 컨테이너
  3개(`qa-ac5-receipt`·`qa-hc-receipt`·`qa-hc2-receipt`)만 만들었고 검증 후 전부 제거했다.

## 시나리오 결과

| # | 시나리오 | 관련 AC | 기대 결과 | 실제 결과 | 판정 |
|---|---|---|---|---|---|
| 1 | 3역할 chromium 실구동 — 크롤러 소스가 실제로 쓰는 playwright API 24종 재현 (중첩 iframe `content_frame` 2단 · `Frame.select_option` · `expect_popup` + `Frame.evaluate(js,arg)` · `expect_download` · `Locator.screenshot(path)` · `expect_navigation` · `frame_locator`+`get_by_text` · `page.route` · sync API · deprecated `ElementHandle.type`) | AC1 | 3역할 전부 동작 | thread 24/24 · delegate 24/24 · receipt 24/24 = **72/72 PASS**, chromium 140.0.7339.16 기동 | PASS |
| 2 | receipt 크롤러 모듈 전수 import + KCS/KSTA dispatch 분기 (`ENABLE_EXTERNAL_ACTIONS=true`) | AC1 | import 0 실패 · 토픽별 올바른 크롤러 호출 | 12/12 import OK · kcs→kcs.process · ksta→ksta.process · 미지정 토픽→`ValueError` | PASS |
| 3 | 배포된 thread `/check_id` · delegate `/screenshot` 대표 경로 호출 | AC1 | baseline 과 동일 거동 | 둘 다 HTTP **503** = 외부동작 차단 가드. 교체 전 컨테이너도 `ENABLE_EXTERNAL_ACTIONS=false` 라 **동일 거동**(회귀 아님). 단 실브라우저 경로는 미실행 — 아래 "덮지 못한 범위" 참조 | PASS(제약 하) |
| 4 | 라우트 A/B — thread (`--network none` + `ENABLE_EXTERNAL_ACTIONS=true`) | AC2 | 0 diff | old 14 / new 14, old-only 0 · new-only 0 → **0 DIFF** | PASS |
| 5 | 라우트 A/B — delegate | AC2 | 0 diff | old 21 / new 21, old-only 0 · new-only 0 → **0 DIFF** | PASS |
| 6 | receipt 라우트 0개 | AC2 | 라우트 0 | `main.py` 부재 · `/app/roles/receipt` 내 fastapi/uvicorn/starlette/APIRouter 참조 **0파일** · `dump_routes.py` → `ModuleNotFoundError: main` | PASS |
| 7 | delegate 스위트 A/B (`pytest tests -q --ignore=tests/test_fixture_runtime.py`) | AC3 | 통합 전과 동일 | 통합 이미지 **38 passed** / 원본 이미지 **38 passed** — 동일 | PASS |
| 8 | `--ignore` 정당성 (누락 자산이 원본에도 없는지) | AC3 | 원본도 동일 실패 | 양쪽 모두 `ModuleNotFoundError: fixture_runtime` 로 collection error — 통합이 만든 결함 아님 | PASS |
| 9 | receipt 스위트 A/B (`pytest tests/test_worker.py`) | AC3 | 통합 전과 동일 | 통합 **4 passed** / 원본 이미지엔 pytest 미설치 → `unittest` 로 **4 OK**(동일 테스트 4종) | PASS |
| 10 | thread 스위트 존재 여부 | AC3 | 스위트 없음 확인 | 원본 `crawler` 저장소에 `tests/` 없음 — "스위트 없음" 사실 | PASS |
| 11 | 신규 3컨테이너 healthy · 구 3컨테이너 정지 | AC4 | healthy 3 / exited 3 | thread·delegate·receipt 전부 `running/healthy` failingStreak=0 · 구 3종 `Exited(143/143/137)`, **삭제되지 않고 잔존** | PASS |
| 12 | 별칭이 각각 신규 IP **하나만** 가리키는가 (망 내 이웃 컨테이너에서 DNS 조회) | AC4 | 1:1 매핑 | `crawler-web→172.28.0.22`(thread) · `ai-delegate-web→172.28.0.23`(delegate) · `pg-receipt-worker→172.28.0.21`(receipt), 각 1개 IP만 응답 | PASS |
| 13 | 별칭 HTTP 응답 | AC4 | 정상 | `crawler-web:42030/health` **200** · `ai-delegate-web:42010/health` **200**, 페이로드 정상 | PASS |
| 14 | 교체 대상 env 승계 (구↔신 전체 env 차집합) | AC4·R5 | 누락 0 | thread·delegate: 구에만 있는 env **0건** · receipt: 앱 env 0건 누락(차이는 `PYTHON_VERSION` 베이스 상향 + `ELASTIC_APM_ENABLED=false` 추가뿐) | PASS |
| 15 | receipt 카프카 왕복 — 정상 1건 | AC5 | completed +1, DLQ +0, LAG 0 | kcs 3→4 · **completed 2→3(+1)** · dlq 1→1(+0) · LAG 0. 페이로드 `{"storeId":11,"receiptId":22,"paymentId":33,"imageUrl":"https://fixture.invalid/qa/ac5.png"}` | PASS |
| 16 | receipt 카프카 왕복 — 스키마 위반 1건 (DLQ 분기) | AC5 | DLQ +1, completed +0 | kcs 4→5 · completed 3→3(+0) · **dlq 1→2(+1)** · LAG 0. 봉투에 `invalidFields:["paymentDate"]`, **`userId`/`userPassword` 미포함(자격정보 마스킹 정상)** | PASS |
| 17 | 교체 전후 `legalcare-local_legalcare` 망 전체 상태 | AC6 | 전부 running | 검증 전 **28개 전부 running** · QA 작업(일회성 컨테이너 3개 생성·삭제 포함) 후에도 **28개 전부 running**. baseline 28개 목록과 대상 3컨테이너 치환 외 차이 없음 | PASS |
| 18 | 롤백 가능성 — 구 컨테이너 정의 잔존 (이미지·별칭·restart·mounts·ports) | AC7 | 되돌릴 수 있음 | 3종 모두 잔존: `restart=unless-stopped` · `mounts=0` · `ports=0` · 별칭(`crawler-web`/`ai-delegate-web`/`pg-receipt-worker`) 정의에 그대로 유지 → `docker start` 한 번으로 복구 가능 | PASS |
| 19 | 롤백 가능성 — 구 이미지 3종 존재·ID 일치 | AC7 | baseline 과 동일 | `433a8a07a602`·`42b79a73a960`·`f8bf1c4519e8` — **baseline 기재값과 3/3 일치** | PASS |
| 20 | `00-baseline.md` 기재값의 현재 실측 부합 | AC7 | 일치 | 구 crawler-web env 10건 · 구 pg-receipt-worker env 13건 **전부 일치** · `Mounts: []` 6컨테이너 전부 사실 · `PortBindings: {}` 사실 | PASS |
| 21 | 롤백 왕복 리허설이 실제로 일어났는가 (타임스탬프 정황) | AC7 | 증거 존재 | 구 3종 `StartedAt=09:33:37~41` → `FinishedAt=09:35:14~44`, 신규 `Created=09:35:45` → 신→구→신 왕복과 정합. **QA 는 지시대로 롤백을 재실행하지 않았다** | PASS(정황) |
| 22 | receipt healthcheck 가 **실제 실패를 만들어내는가** (독립 실증, 일회성 컨테이너) | 추가 | 죽으면 실패 | 살아있는 실컨테이너 exit **0** · 워커 없는 컨테이너 exit **1** · `worker.py` 문자열 미끼 프로세스(브로커 연결 없음) exit **1** → 1라운드 "항상 통과" 결함 **해소 확인**. 검사 프로세스는 `ppid=0` 으로 관측돼 구조적으로 자기 자신을 못 센다 | PASS |
| 23 | receipt healthcheck 의 사각 — 망 단절 시 | 추가 | — | 워커 살아있고 컨테이너도 running 인 채 브로커 세션만 끊으면 **60초+ 동안 healthy 유지** → **D1** | FAIL(경미) |
| 24 | 이미지 위생 — allowlist COPY 준수 | 추가 | 금지 자산 0 | `.env*` 0 · `tests/` 0 · `fixture*` 0 · `*.md`/`docker-compose*`/`Dockerfile`/`.dockerignore`/`.gitignore` 0 · `tools/` 0. `/app` 최상위 = `docker`,`requirements.txt`,`roles` 뿐. 실행 uid **10001 비루트** | PASS |
| 25 | 이미지 == 현재 소스 트리인가 (빌드 09:15 vs Dockerfile mtime 10:43 의심 검증) | 추가 | 일치 | 이미지 내 168파일 전수 sha256 대조: **불일치 0 · 이미지에만 있는 파일 0** · 소스에만 있는 12파일은 전부 allowlist 제외 대상(tests 10 + fixture 2). 베이스 하위 4레이어 = Dockerfile 핀 `@sha256:229a2c…` 와 **일치** → 빌드 후 Dockerfile 변경은 기능 무영향(주석) | PASS |
| 26 | R1 receipt 무수정 복제 (양방향 sha256) | R1 | 완전 일치 | `roles/receipt` 17파일 ↔ `pg-receipt-crawler` 17파일: **누락 0 · 추가 0 · 불일치 0** | PASS |
| 27 | R2 entrypoint 역할 분기 · 오타/미지정 exit 64 | R2 | 사양대로 | `''`·`recipt`·`RECEIPT`·`worker` → **exit 64** · thread/delegate/receipt → exit 0. receipt 는 `cwd=/app/roles/receipt`, `PYTHONPATH=/app/roles/receipt`, 프로세스 `python worker.py`(ppid=1), **앱 LISTEN 포트 0개**(관측된 1건은 도커 내장 DNS 127.0.0.11) | PASS |
| 28 | R3 requirements union · 버전 통일 · 의존성 정합 | R3 | 사양대로 | `playwright==1.55.0` · `pydantic==2.12.0` · `aiokafka==0.10.0` 와 `kafka-python==2.2.15` **공존** · `pip check` → `No broken requirements found` | PASS |
| 29 | R4 판단 근거 — 사본이 정말 무수정인가 | R4 | 수정 0 | thread·delegate·receipt 3역할 전부 **원본 대비 해시 불일치 0건**(제외분은 `.github/`·`Dockerfile`·`Pipfile*`·`nb/*.ipynb`·`app.log` 등 빌드/CI 산출물, 역할 소스 `.py` 는 각 1건 `local/lambda_server.py` 뿐). → `roles/PATCHES.md` 부재가 정당 | PASS |

**요약: 29 시나리오 중 28 PASS · 1 FAIL(경미).** 개별 체크 건수로는 약 150건(AC1 하니스 72 + import/dispatch 15 + 라우트 A/B 3 + 스위트 6 + env 차집합 3 + 카프카 8 + AC7 12 + 위생·해시 대조 168파일 등).

## 발견 결함

### D1. [심각도: MINOR] receipt healthcheck 는 망 단절 시 60초+ 동안 healthy 를 유지한다

- **재현 절차**
  1. `docker run -d --name qa-hc2-receipt --network legalcare-local_legalcare -v ~/qa-3role:/qa:ro --health-cmd 'python /qa/hc.py' --health-interval 5s --health-retries 2 -e CRAWLER_ROLE=receipt -e KAFKA_SERVER_1=kafka:9092 -e KAFKA_CONSUMER_GROUP_ID=<throwaway> … crawler-worker:dev-3role`
     (`hc.py` 는 `docker inspect crawler-worker-receipt -f '{{index .Config.Healthcheck.Test 3}}'` 로 **실배포 컨테이너에서 그대로 추출**한 것)
  2. `health=healthy` 될 때까지 대기 (10초)
  3. `docker network disconnect legalcare-local_legalcare qa-hc2-receipt`
  4. 5초 간격으로 `docker inspect -f '{{.State.Health.Status}} {{.State.Health.FailingStreak}}'` 관측
- **기대 vs 실제**: 브로커와 통신 불가 → 수십 초 내 `unhealthy` 기대. 실제로는 **60초 내내 `healthy`, FailingStreak=0**, 워커 프로세스는 살아 있음(`ps` 로 `python worker.py` ppid=1 확인).
- **원인**: 인터페이스가 사라져도 커널은 TCP 소켓을 `ESTABLISHED`(`/proc/net/tcp` state `01`)로 유지한다(재전송 타임아웃까지 ~15분). 검사는 `ESTABLISHED` 존재만 보므로 통과한다.
- **문서와의 불일치**: `docker-compose.dev.yml:116-117` 과 `04-changes.md` 는 "브로커가 `connections.max.idle.ms`(600초) 후 연결을 닫아 `ESTABLISHED` 가 사라진다"고 근거를 적었다. 그 메커니즘은 **망이 살아 있을 때만** 성립한다. 망 단절·브로커 소실 구간에는 적용되지 않으므로 사각이 문서가 말하는 ~11분보다 넓다.
- **관련 파일**: `crawler-worker/docker-compose.dev.yml:109-160` (healthcheck 정의·주석)
- **완화 판단(경미 사유)**: (1) 워커 프로세스가 죽는 주 경로는 tini 가 함께 종료돼 컨테이너가 내려가고 `restart: unless-stopped` 가 재기동하므로 healthcheck 없이도 복구된다(실측 확인). (2) 교체 대상이던 구 컨테이너의 검사는 `python -c 'import os; os.kill(1, 0)'` 로 **이보다 더 약했다** — 회귀가 아니라 개선이다. (3) 1라운드에서 지적된 "항상 통과" 결함은 실제로 해소됐다(시나리오 22).

### D2. [심각도: MINOR] `fake-useragent` 하위 핀으로 크롤러가 광고하는 UA 가 실제 브라우저보다 23 메이저 뒤처지고, Chrome 아닌 **Opera UA** 도 섞여 나온다

- **재현 절차**
  ```bash
  # 통합 이미지 (fake-useragent 1.4.0)
  docker run --rm --network none -e CRAWLER_ROLE=receipt crawler-worker:dev-3role \
    python -c "from crawlers import crawling_utils as cu; [print(cu.get_ua()) for _ in range(3)]"
  # 원본 receipt 이미지 (fake-useragent 2.2.0)
  docker run --rm --network none --entrypoint python legalcare/pg-receipt-crawler:dev-20260806-safe \
    -c "from crawlers import crawling_utils as cu; [print(cu.get_ua()) for _ in range(3)]"
  ```
- **기대 vs 실제**
  - 원본: `Chrome/134.0.0.0` · `Chrome/135.0.0.0` · `Chrome/131.0.0.0`
  - 통합: `Chrome/117.0.0.0` · **`Chrome/116.0.0.0 … OPR/102.0.0.0`** · `Chrome/117.0.0.0`
  - 실제 기동하는 바이너리는 **Chrome/140.0.7339.16**. UA 와 엔진이 23 메이저 어긋나고, `get_ua()` 이름·용도(Chrome UA)와 달리 Opera UA 가 반환될 수 있다.
- **영향 경로**: `roles/receipt/crawlers/crawling_utils.py:22-30` `get_ua()` → `kcs_receipts_crawler.py:32-33` · `ksta_receipts_crawler.py:42-43` 의 `new_context(user_agent=driver_ua)`. 국세청(KCS)·KSTA 는 봇 탐지가 있는 사이트라 UA/엔진 불일치는 차단 위험 요인이다.
- **현재 폭발하지 않는 이유**: 개발계 3역할 모두 `ENABLE_EXTERNAL_ACTIONS=false` 라 이 경로가 실행되지 않는다. `get_ua()` 자체는 예외 없이 동작하며, 반환 UA 로 `new_context` → `navigator.userAgent` 왕복도 일치 확인(PASS).
- **문서 대비**: `04-changes.md` 는 "fake-useragent(UA Chrome 최대 135→117)"까지만 적었다. **Opera UA 반환 가능성은 미기재**다.
- **관련 파일**: `crawler-worker/requirements.txt` (`fake-useragent==1.4.0`) · `roles/receipt/crawlers/crawling_utils.py:22`

### D3. [심각도: MINOR] `tools/dump_routes.py` 산출물이 delegate 에서 유효한 JSON 이 아니다

- **재현 절차**
  ```bash
  docker run --rm --network none -e ENABLE_EXTERNAL_ACTIONS=true -e CRAWLER_ROLE=delegate \
    -v ~/crawler-worker-build/tools:/tools:ro crawler-worker:dev-3role python /tools/dump_routes.py > out.json
  python3 -c "import json; json.load(open('out.json'))"
  ```
- **기대 vs 실제**: 기대 — JSON 한 줄. 실제 — 첫 줄에 `[APM] Elastic APM disabled` 가 섞여 `JSONDecodeError: Expecting value: line 1 column 2`. thread 원본 이미지에서는 lambda-proxy 로그도 stdout 에 섞인다.
- **영향**: AC2 산출물을 기계 비교하려면 사람이 매번 앞줄을 걷어내야 한다. **A/B 판정 자체는 영향 없음**(양쪽 이미지에 동일하게 섞이며, 줄 파싱 후 비교하면 thread 14=14 · delegate 21=21 로 0 diff). 산출물 신뢰성·자동화 문제다.
- **수정 방향**: 라우트 JSON 을 `stderr` 가 아닌 전용 파일(`--out`)로 쓰거나, `print` 대신 `sys.stdout` 을 마지막에 한 번만 쓰도록 격리.
- **관련 파일**: `crawler-worker/tools/dump_routes.py:20-30`

## 엣지 케이스 검증 내역

개발자가 언급하지 않았거나 QA 가 독립적으로 추가한 것들.

- **역할 문자열 대소문자·오타·공백**: `RECEIPT`(대문자)도 `exit 64` 로 거부됨을 확인. 사양상 소문자만 허용이라 의도대로지만, 운영에서 대문자 오기입 시 컨테이너가 조용히 안 뜨는 게 아니라 즉시 64 로 죽는다는 점은 정상 동작.
- **인자 없는 `chromium.launch()`**: `roles/thread/agents/naver/naver_map_place_screenshot.py:21` 은 `--no-sandbox` 없이 launch 한다. 비루트(uid 10001) 컨테이너에서 이 호출이 실패할 수 있어 별도 시나리오로 확인 → **3역할 전부 PASS**.
- **deprecated API 잔존**: playwright 최신 계열에서 권장되지 않는 `ElementHandle.type()`(소스 3건) · `page.expect_navigation()`(소스 4건)이 1.55 에서 여전히 동작하는지 개별 확인 → 둘 다 동작.
- **중첩 iframe 2단**: KCS 크롤러의 `iframe#iframe_363` → `iframe#iframe_inner_card` 구조를 픽스처로 재현해 `ElementHandle.content_frame()` 2단 체이닝 확인.
- **DLQ 봉투의 자격정보 노출**: 스키마 위반 메시지에 `userId`/`userPassword` 를 넣고 DLQ 레코드를 실제로 읽어 **자격정보가 봉투에 포함되지 않음**을 확인(`safe_identifiers()` 가 숫자 식별자만 통과).
- **알 수 없는 소스 토픽**: `worker.process_request('unknown.topic', …)` → `ValueError` 로 방어됨 확인.
- **`ENABLE_EXTERNAL_ACTIONS` 미지정 시 기본값**: receipt·thread 모두 **기본값 True**. 즉 env 를 흘리면 실크롤링이 켜진다. 다만 이는 교체 전 원본과 동일한 성질이고 compose 가 명시적으로 `false` 를 주입하므로 현재 배포 상태는 안전(회귀 아님). 롤백/재교체 시 env 누락에 주의.
- **receipt LISTEN 포트**: R2 의 "포트 바인딩 없음"을 `/proc/net/tcp` 로 직접 확인. LISTEN 1건이 잡혀 정밀 조사한 결과 `127.0.0.11`(도커 내장 DNS)이었고 앱 포트는 0개.
- **컨슈머 그룹 오염 여부**: 배포된 receipt 그룹 `legalcare-local-pg-receipt-fixture-v1` 이 검증 전후로 동일(kcs 2/2 LAG 0 · ksta 1/1 LAG 0, 멤버 1, Stable). QA 검증은 별도 그룹·별도 토픽으로만 수행.
- **이미지에 섞인 테스트 파일**: `agents/google/test_google_map_place_crawling.py` 가 thread·delegate 이미지에 들어간다(`agents/` 디렉토리 통째 COPY 때문). **원본 이미지 2종에도 동일하게 존재**함을 확인해 회귀가 아님을 판정(위생 지적 사항으로만 기록).
- **빌드 시각과 Dockerfile mtime 불일치**: 이미지 빌드 `09:15Z` 인데 Dockerfile mtime 이 `10:43Z` 라 "실행 중 이미지가 현재 소스와 다를 수 있다"고 의심했고, 168파일 해시 전수 대조 + 베이스 다이제스트 레이어 대조로 **기능적으로 동일**(변경분은 주석)임을 확인.
- **로컬↔개발계 소스 동기화**: 처음 트리 해시가 달라 보였으나 macOS/Linux `sort` 로케일 차이였고, `LC_ALL=C` 로 재정렬해 **173개 `.py` 전부 동일** 확인. (판정 근거로 쓰는 트리가 같은 것임을 먼저 못박았다.)

## 덮지 못한 범위 (다음 사람이 반드시 읽어야 할 것)

지시상 금지되었거나 환경상 불가능해 **검증하지 못한** 구간이다. "안 돌려봤다"는 뜻이지 "된다"는 뜻이 아니다.

1. **실사이트 크롤링 0건.** 국세청(KCS)·KSTA·네이버·카카오 접속 금지 제약으로, AC1 원문의 "receipt KCS/KSTA 각 1건 성공"은 **로컬 픽스처 기반 API 동등성으로 대체**했다. 셀렉터·로그인 플로우·팝업 타이밍 등 사이트 의존 동작은 미검증.
2. **실 S3·실 Vault 0건.** `s3_save_review_boost_receipt.py`·`vault.py` 는 import 만 확인했다. D2 와 함께, boto3 `1.40.51→1.34.34`(PutObject 무결성 헤더 CRC32→Content-MD5) · certifi `2025.10.5→2024.8.30`(루트 CA 7개 감소) 하위 핀이 실제로 무는 지점이 여기다.
3. **`ENABLE_EXTERNAL_ACTIONS=true` 운영 경로 0건.** 개발계 3역할 전부 `false` 라 배포된 컨테이너는 playwright 를 한 번도 태우지 않는다. 브라우저 검증은 전부 QA 하니스와 일회성 컨테이너에서 이뤄졌다.
4. **롤백 실행 미수행.** 지시대로 개발계를 신규 상태로 유지했다. AC7 은 "되돌릴 수 있는 조건이 전부 갖춰졌는가"(구 컨테이너·구 이미지·별칭 정의·무볼륨)와 타임스탬프 정황까지만 검증했다.
5. **리플레이 함정 미실증.** `KAFKA_AUTO_OFFSET_RESET=earliest` + 그룹 오프셋 만료(기본 7일) 조합에서 재교체 시 재소비가 일어나는 시나리오는 시간상 재현하지 않았다. 현재 fixture 토픽에 레코드 2건(kcs)·1건(ksta)이 남아 있고 오프셋이 커밋돼 있어 **지금 당장은 무해**하지만, 7일 넘긴 재교체 전에는 그룹 오프셋 존재를 반드시 확인해야 한다(compose 주석·README 에 기재됨).

## 사용자 의도 부합 확인

원문: *"영수증 크롤러? 실제 쓰나? 브라우저는 버전 높은거로 통합 하면 안돼? 무튼 그래서 신규로 모두 마이그해서 합쳐봐. 그렇게 해서 신규 스펙으로 개발계에 새로 다띄워. 그리고 전체 테스트해"*

- "**합쳐봐**" → 단일 이미지 3역할(`CRAWLER_ROLE`)로 통합, receipt 는 sha256 무수정 복제. **부합**.
- "**브라우저는 버전 높은거로**" → 3역할 전부 playwright **1.55.0** 단일화. 실측상 thread/delegate 가 1.42.0 → 1.55.0 으로 올라갔고, receipt 원본 이미지는 **이미 1.55.0·pydantic 2.12.0** 이었다(즉 receipt 쪽이 높은 버전이었고 그쪽으로 통일된 것). **부합**.
  - 다만 union 결과 receipt 의 **비브라우저 의존성 일부는 뒤로 갔다**(boto3 1.40.51→1.34.34, certifi 2025.10.5→2024.8.30, fake-useragent 2.2.0→1.4.0). 사용자가 말한 "버전 높은거로"는 브라우저 한정이므로 지시 위반은 아니지만, "합치면 전반적으로 최신이 된다"는 인상과는 반대 방향의 변화가 3건 있다는 점은 명시적으로 남긴다(D2 · `04-changes.md` 에도 공개돼 있음).
- "**개발계에 새로 다띄워**" → 3컨테이너 healthy, 별칭 승계로 호출자 설정 무변경, 구 컨테이너는 정지만 하고 보존. **부합**.
- "**전체 테스트해**" → 3역할 스위트 + 라우트 A/B + 카프카 왕복 + 브라우저 실구동까지 수행. 단 위 "덮지 못한 범위" 1~3 이 남는다. **조건부 부합** — 실사이트·실 S3·실 Vault 검증은 제약으로 불가했고, 이는 후속 태스크의 첫 항목이 되어야 한다.
- "**영수증 크롤러? 실제 쓰나?**"(사용량 질문) → `01-requirements.md` 가정 A1(KCS 24 · KSTA 1 · notification 0)로 답이 정리돼 있고, 실측상 개발계 fixture 토픽 보관 레코드는 kcs 2 · ksta 1 로 미미함을 재확인했다. **부합**.

## 총평

AC1~AC7 을 QA 가 직접 재실행해 **7개 수용 기준 전부 충족**을 확인했다. 특히 판정의 무게가 실린 세 곳은 남의 주장이 아니라 직접 만든 증거로 눌렀다.

- **AC1** 은 개발자 하니스를 쓰지 않고 QA 가 크롤러 소스(`roles/receipt/crawlers/*.py`, `roles/{thread,delegate}/agents/**`)를 읽어 **실제 호출되는 API 24종만 골라** 픽스처를 새로 짰고, 3역할 × 24 = 72/72 PASS 를 받았다. 중첩 iframe 2단·`Frame.select_option`·`expect_popup`+`evaluate(arg)`·`expect_download` 같은 1.55 상향에서 깨지기 쉬운 지점이 전부 포함돼 있다.
- **AC2/AC3** 은 통합 전 원본 이미지 2종을 직접 띄워 A/B 했다. 라우트 14=14·21=21 0 diff, delegate 38=38, receipt 4=4. `--ignore` 가 결함 은폐가 아니라 **원본에도 동일한 선재 실패**임을 양쪽에서 재현해 확인했다.
- **1라운드 healthcheck 결함**은 실배포 컨테이너에서 검사 스크립트를 그대로 추출해 3가지 상황(정상/워커 없음/문자열 미끼)에 돌려 **해소를 독립 확인**했다.

의심 지점도 그냥 넘기지 않았다. 이미지 빌드 시각(09:15Z)이 Dockerfile mtime(10:43Z)보다 앞서 "도는 이미지가 소스와 다를 수 있다"고 봤고, 이미지 내 168파일 sha256 전수 대조 + 베이스 다이제스트 레이어 대조로 기능적 동일성을 확정했다. 로컬과 개발계 트리 해시가 달라 보인 것도 로케일 문제임을 먼저 못박고 나서 검증을 진행했다.

결함 3건은 전부 MINOR 다. D1(healthcheck 망 단절 사각)은 교체 전 검사(`os.kill(1,0)`)보다 오히려 강해진 것이라 회귀가 아니고, 주 실패 경로는 컨테이너 재기동으로 덮인다 — 다만 compose 주석의 근거 설명이 부정확하니 문구는 고치는 게 좋다. D2(fake-useragent 하위 핀)와 D3(dump_routes 산출물 오염)도 이번 개발계 범위에서는 터지지 않는다.

**남는 실질 리스크는 결함이 아니라 "덮지 못한 범위"에 있다.** `ENABLE_EXTERNAL_ACTIONS=true` 를 켜는 순간 처음 도는 코드가 곧 미검증 구간(실사이트 셀렉터 · boto3 무결성 헤더 · certifi 루트 CA · UA 핑거프린트)이며, 이 네 가지가 한 지점(receipt 크롤러 + S3 업로드)에 몰려 있다. 운영 배포 태스크는 여기부터 시작해야 한다.

개발계는 검증 시작 시점과 동일한 상태로 되돌려 놓았다(신규 3역할 healthy · 구 3컨테이너 exited · 망 28개 전부 running · 별칭 1:1 · 배포 컨슈머 그룹 LAG 0 · QA 임시 컨테이너 3개 전부 제거).

QA_VERDICT: PASS
