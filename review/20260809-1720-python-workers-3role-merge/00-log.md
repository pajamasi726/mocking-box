# 파이썬 크롤러 워커 3종 완전 통합 (브라우저 버전 상향 + 개발계 전면 재기동)

**사용자 원문 요청**
> /team 자체 리뷰로 pg-receipt-crawler 그대로 둠 — 브라우저 라이브러리 버전이 A와 충돌 (1.55 vs 1.42)
> 이건 또 뭔지 모르겠네
> 영수증 크롤러? 실제 쓰나?
> 브라우저는 버전 높은거로 통합 하면 안돼?
> 무튼 그래서 신규로 모두 마이그해서 합쳐봐
> 그렇게 해서 신규 스펙으로 개발계에 새로 다띄워
> 그리고 전체 테스트해

**구성**: writer=claude · reviewer=claude(단독 셀프리뷰) · mode=review · rounds=2 · quality 전부 off · cast=auto
**대상 저장소**: `/Users/steve/steve/legal-care/crawler-worker` (HEAD `488e829`, 역할 2개 → 3개로 확장)

## 17:20 오케스트레이터 → 착수 전 실측 (PM 지시서 입력용 사실)

**pg-receipt-crawler 정체** — 리뷰부스터 영수증 크롤러. **HTTP 라우트 0개**(FastAPI 없음, `worker.py` 카프카 루프 단독).
국세청(KCS)·KSTA 두 곳을 playwright chromium 으로 긁어 결과를 카프카로 돌려준다.
- 토픽: `medilawyer.crawling.review-boost.receipt.crawling`(KCS) / `...ksta.receipt.crawling`(KSTA)
  → `...receipt.notification`(완료) / `...receipt.dlq`
- 개발계 누적: KCS 24건 · KSTA 1건 · notification 0건. `review-boost-receipt-notification-group` LAG 0
- 컨테이너 `legalcare-local-pg-receipt-worker-1` (이미지 `pg-receipt-crawler:dev-20260806-safe`) 2일째 healthy

> **[08-09 round-1 정정]** 위 "KCS 24건"은 **어느 브로커인지를 안 적어 틀리게 읽힌다.** DELL 에는 카프카
> 클러스터가 둘 있다. 24·1·0 은 `kafka1`/`kafka2`(compose 프로젝트 `kafka`, 망 `kafka_default`)의
> **운영 이름 규칙 토픽** `medilawyer.crawling.review-boost.receipt.crawling` 누적치가 맞다(재확인함).
> 그런데 **워커가 실제로 붙는 브로커는 `legalcare-local-kafka-1`**(별칭 `kafka`, 172.28.0.3)이고,
> 거기서 보는 fixture 토픽은 kcs 2 · ksta 1 · completed 2 · dlq 1 이며 `retention.ms=86400000` 이라
> 지금 보관 레코드는 전부 0건이다. 두 클러스터는 망이 달라 서로 못 본다.
> 가정 A1(실사용량 미미)은 어느 숫자로 읽어도 성립한다 — 오히려 더 강해진다.
> 덤으로 확인된 것: `legalcare-local-kafka-1` 에도 `medilawyer.crawling.review-boost.receipt.notification`
> 토픽이 있고 `legalcare-local-booster-worker-1` 이 붙어 있는데, receipt 는 fixture `completed` 로 발행한다
> → **개발계의 receipt→booster 루프는 이번 작업 이전부터 끊겨 있었다.** 이번 교체가 만든 상태가 아니다.

**버전 충돌 실측** (통합 대상 3종)
| 패키지 | thread(crawler) | delegate(ai-delegate) | receipt(pg-receipt) | 통일 목표 |
|---|---|---|---|---|
| playwright | 1.42.0 | 1.42.0 | **1.55.0** | 1.55.0 |
| pydantic | 2.9.2 | 2.9.2 | **2.12.0** | 2.12.0 |
| kafka 클라이언트 | aiokafka 0.10.0 | aiokafka 0.10.0 | **kafka-python 2.2.15** | 둘 다 유지(다른 패키지) |

**HTTP 호출자 전수 조사(APM 90일)** — 이번 작업의 위험 가늠자
- 크롤러를 HTTP 로 부르는 서비스는 **`legacy-service` 하나뿐**. 운영 90일 1,075건(하루 12건).
  `prod-delegate...:443` 1,014 · `172.31.40.145:42030` 61. 개발 11,220건(대부분 workstation delegate)
- thread 라우트별(52일): `/check_id` 56 · `/naver_login_captcha` 13 · `/naver_cafe_passive_crawl` 1
- → **실제 일감은 카프카로 들어온다**(사용자 예측 맞음). HTTP 회귀 위험보다 **브라우저 상향 회귀 위험이 크다**

## 17:26 team-pm → 오케스트레이터
- R1~R7 / AC1~AC7. **RISK: HIGH** (3역할 브라우저 엔진 상향 + 개발계 크롤러 전면 교체 + 스택 중단 전례).
- 핵심 판단 2개: ①1.55 에서 깨지면 원본은 무수정 유지, `roles/` 사본만 최소 수정 + `roles/PATCHES.md` 기록(R4)
  ②"새로 다띄워"=기존 3컨테이너 정지·제거(AC4) 하되 이미지·env·별칭 덤프(R7)와 롤백 왕복 리허설(AC7)로 되돌릴 수 있게
- OPEN QUESTIONS 없음. STATUS: READY

## 17:30 오케스트레이터 → 지시서 자가점검 + AC1 실현가능성 정정
사용자 의도와 일치 확인(3종 통합·높은 버전 통일·개발계 재기동·전체 테스트). 다만 **AC1 을 실측으로 좁힌다.**
- `pg-receipt-crawler/vault.py:9` — 자격정보는 Vault(`vault.medilawyer.co.kr`, `VAULT_GITHUB_TOKEN`)에서 온다.
  AC1 의 "receipt KCS/KSTA 각 1건"을 문자 그대로 하면 **운영 자격정보로 국세청·KSTA 에 실접속**하게 된다. 범위 밖이다.
- AC1 의 목적은 "playwright 1.55 가 chromium 을 제대로 몬다"의 확인이지 국세청 로그인 회귀 확인이 아니다.
  → **정정**: receipt 는 통합 이미지에서 chromium 을 실제로 띄워 페이지 로드·조작·스크린샷까지 되는지로 검증한다
  (국세청·KSTA 실접속 금지). thread/delegate 도 동일 기준.
- receipt 에 `ENABLE_EXTERNAL_ACTIONS` 안전 스위치가 이미 있다(`config.py:28` 기본 true, `worker.py:67` 게이트).
  개발계 기동은 다른 두 역할과 같이 false 로 간다.
- 이 정정은 00-log 에만 남기고 지시서는 재작성하지 않는다(PM 재호출 불필요, 범위 축소 방향).

## 18:21 team-dev(IMPLEMENT) → 오케스트레이터
- `roles/receipt`(17파일) 추가, entrypoint receipt 분기(`python worker.py`, 포트 없음), requirements 3-way union,
  compose 2종에 receipt 추가, README/.env.example 갱신. **R4 발동 없음** — 1.55 에서 깨진 코드 0.
- 개발계 교체 완료: 신규 3역할 healthy · 구 3컨테이너 exited · 롤백 왕복 1회 성공. 상세 `04-changes.md`·`00-baseline.md`
- 리뷰어에게 넘긴 판단 3건: ①boto3 계열 하위핀 유지(aiobotocore 가 botocore<1.34.35 못박음) ②receipt 는
  `ENABLE_KAFKA_CONSUMER` 스위치가 없어 격리 토픽·그룹으로 안전 기동 ③receipt healthcheck 가 컨슈머 생존을 못 봄
- STATUS: DONE

## 18:35 오케스트레이터 → 독립 재검증 (작성자 보고 교차확인, DELL 실측)
| 항목 | 작성자 주장 | 내 재실측 | 판정 |
|---|---|---|---|
| roles 3종 sha256 | 원본과 동일 | 3역할 전부 **내용상이 0 · 사본전용 0**. receipt 는 누락도 0 | 확인 |
| 자격정보 혼입 | `.env*` 제외 | `roles/receipt` 에 `.env`·`.env.develop`·`.env.release`·`.env.workstation` **전부 없음** | 확인 |
| 중첩 `.gitignore` | 언급 없음 | roles 3곳에 원본 `.gitignore` 가 딸려왔으나 IDE·`__pycache__`만 무시. **git 무시 파일 0건** | 무해 |
| 라이브러리 상향 | playwright 1.55 / pydantic 2.12 | 이미지에서 `pip show` 실측 **1.55.0 / 2.12.0** | 확인 |
| 라우트 A/B | 0 diff | 통합 전 원본 이미지 vs 1.55 상향본: thread 14=14 · delegate 21=21 **집합 동일**. receipt fastapi 참조 0 | 확인 |
| 스위트 | delegate 38 · receipt 4 | 내가 재실행 **38 passed / 4 passed** | 확인 |
| 별칭 | 신규만 가리킴 | `crawler-web`→172.28.0.22 · `ai-delegate-web`→172.28.0.23 · `pg-receipt-worker`→172.28.0.21, **각각 IP 1개**이고 신규 컨테이너와 일치. 별칭 HTTP 200 | 확인 |
| 스택 무중단 | 28/28 running | `legalcare-local_legalcare` 28컨테이너 **전부 running** | 확인 |
| 원본 무수정 | mtime 0건 | 원본 5개 저장소 오늘(8/9) 수정 파일 **0건** | 확인 |

작성자 보고에서 뒤집힌 주장 없음. `00-baseline.md` 168줄에 롤백 근거(이미지ID·env·별칭·마운트0·포트0) 확보 — `docker start` 한 번으로 복구 가능.

## 19:00 리뷰어(독립 컨텍스트) → 작성자
- 지적 11건(MAJOR 3 · MINOR 8). 상세 `round-1-review.md`. **VERDICT: REQUEST_CHANGES**
- MAJOR ①receipt healthcheck 가 **항상 통과**(검사 프로세스 자신의 `-c` 소스에 "worker.py" 문자열이 있어 자기를 발견)
  ②낮춘 의존성 12건이 한 줄도 실행 안 됨(전부 `ENABLE_EXTERNAL_ACTIONS` 뒤) ③receipt 가 Debian 12→13 으로 갈아탔는데 미기재
- 리뷰어가 작성자 대신 pydantic 2.9→2.12 동작 A/B 를 직접 돌려 **3역할 전 모델 동일** 확인(OpenAPI 한 줄 차이만 발견)

## 19:05 오케스트레이터 → MAJOR 3건 독립 확인
- **①확인.** worker.py 가 안 도는 컨테이너에서 같은 명령 실행 → `exit=0`. PID 1·7 둘 다 자기 cmdline 에 "worker.py" 를 담고 있다.
  → 실제로 **항상 healthy**. AC4 의 "receipt healthy"는 공허하다. 리뷰어 지적이 맞다.
- **③확인, 단 범위는 receipt 한정.** 원본 `pg-receipt-crawler/Dockerfile:4` 만 `python:3.12.10-slim`(Debian 12 bookworm)로 고정돼 있었고
  crawler·ai-delegate 는 원래부터 `python:3.12-slim`(Debian 13 trixie). 실이미지로 확인:
  `pg-receipt-crawler:dev-20260806-safe`=Debian 12 / 나머지 전부 Debian 13. → **receipt 만 12→13 이동**이 맞다.
- ②는 성격상 재실측이 아니라 "근거 부재" 지적이라 그대로 인정.

## 19:50 team-dev(RESPOND) → 오케스트레이터
- 11건 처리(반영 9 · 부분반박 2). 상세 `round-1-response.md`. STATUS: REVISED
- MAJOR1: healthcheck 를 이름매칭 폐기 → **위치(`ppid==1`, tini 가 exec 한 자식) + 브로커 ESTABLISHED 소켓** 으로 재작성.
  워커 없는 컨테이너에서 구=exit 0 / 신=exit 1 실증, 도커 자체 판정도 unhealthy 로 뒤집힘 확인.
  "3일 사고를 잡았을까" → **약 11분이면 잡는다**(브로커 `connections.max.idle.ms=600000` 실측 + interval 15s×retries 3)
- MAJOR2: stdlib 스텁으로 낮춘 12건을 원본 이미지와 A/B → **"표면 동일" 주장이 3건에서 틀렸다**
  (boto3 PutObject 무결성헤더 CRC32→Content-MD5 · fake-useragent UA Chrome 135→117 · certifi 루트 7개 누락)
- MAJOR3: **receipt 만** Debian 12.11→13.6 이동임을 정정 기재 + `ARG PYTHON_BASE=...@sha256:229a2c5b…` 다이제스트 고정
- MINOR4 에서 문서가 정반대였음 발견: chromium 7프로세스 전부 `headless_shell`, 594MB full chromium 미사용

## 20:02 오케스트레이터 → MAJOR1 수정 독립 확인
새 healthcheck 를 워커 없는 컨테이너에서 실행 → **exit=1**(구 검사는 exit=0). 실제 receipt 컨테이너 healthy/streak 0.
3역할 healthy · 공용망 28컨테이너 전부 running.

## 20:05 리뷰어(2라운드, 독립 컨텍스트) → **VERDICT: APPROVE**
- healthcheck 구멍 없음: 배포본에서 tini `ppid=0` · 워커 PID7 `ppid=1` · `docker exec` 계열 `ppid=0/579` 실측 →
  `ppid==1` 필터에 검사 프로세스가 구조적으로 안 걸린다. 리밸런싱은 같은 TCP 세션 위라 ESTABLISHED 안 끊김(3본 보유).
  env 비거나 형식 달라도 **fail-closed**
- 거짓 unhealthy 의 대가 = **상태 플래그뿐**. 도커는 헬스로 재시작을 걸지 않고, 공용망에 autoheal·`depends_on: service_healthy` 0건
- 다이제스트 `229a2c5b…` 는 실재 OCI image index 이고 DELL 의 `python:3.12-slim` RepoDigest 와 일치. rootfs 레이어 동일
- 반박 2건 둘 다 타당(#7 다른 클러스터에서 `receipt.crawling:0:24` 직접 확인 / #9 `roles/receipt/requirements.txt:3` 에 원래 있음)
- 범위 이탈 0건(`docker history` apt 명령·이미지 내 entrypoint sha256·`pip list` 핀이 소스와 일치)
- 남은 MINOR 2: **F1** 브로커 45초+ 중단 시 워커가 멀쩡해도 unhealthy 로 보이는 점이 README 에 없음 ·
  **F2** `04-changes.md:8` 표에 1라운드 #4 가 반증한 문구가 남음
- renew-replay 이중등록: 구 3컨테이너가 `exited`+`unless-stopped` 라 **`start` 하나로도** 별칭 이중등록+그룹 동시참여.
  저장소 밖 파일이라 인수인계 사항

## 20:20 사용자 추가 지시 → 범위 확장 예고
> "이름을 새로 지어주던지 해서 신규 셋트 하나를 dev에 띄워서 다 테스트 해보고 그대로 운영에 띄울거야"

**운영 배포 방식 실측** (이 지시의 산출물 모양을 결정한다) — 셋이 서로 **다른 방식**으로 뜬다.

| 역할 | 원본 저장소 | 운영 배포 방식 | ECR 저장소 |
|---|---|---|---|
| thread | crawler | **EC2 에 `docker run`** (GitHub Actions `appleboy/ssh-action`) → 52.78.94.62 | `crawler-prod` |
| delegate | ai-delegate-crawler | **ECS Fargate** (`amazon-ecs-deploy-task-definition`, task def + service) | `ai-delegate-crawler-prod` |
| receipt | pg-receipt-crawler | **배포 워크플로 자체가 없다** (`.github/` 디렉토리 부재) | 미확인 |

부수 발견: `crawler/task-definition.json` 의 family 가 `task-ai-delegate-crawler-prod` 로 **ai-delegate 것을 복사한 흔적**이다
(포트만 42030 으로 다름). 다만 crawler 는 ECS 를 안 쓰고 EC2 로 뜨므로 이 파일은 죽은 파일로 보인다.

→ "그대로 운영에 띄운다"를 하려면 **배포 방식을 하나로 모으는 일**이 별도로 필요하다. 통합 이미지는 1개인데
운영 경로가 3갈래(EC2 docker run / ECS Fargate / 없음)라 이미지만으로는 프로모션이 안 된다.
receipt 가 현재 운영에 떠 있기는 한지도 미확인(AWS 자격정보 만료로 이번 세션에서는 조회 불가).

## 20:40 team-qa → **QA_VERDICT: PASS** (29시나리오 · 약150체크 · 28 PASS / 1 FAIL 경미)
- AC1~AC7 전부 PASS. QA 가 크롤러 소스를 읽고 **실제 쓰는 playwright API 24종만 골라 하니스를 새로 작성** → 3역할×24 = 72/72 PASS
- AC5 는 정상 1건(completed+1·DLQ+0) + 위반 1건(DLQ+1·completed+0) 양쪽으로 돌려 탐지력 확보. DLQ 봉투에 자격정보 미포함 확인
- 부수: 이미지 빌드시각이 Dockerfile mtime 보다 앞서 "도는 이미지≠소스" 의심 → 168파일 sha256 전수 대조 0 불일치로 확정
- 결함 3건 전부 MINOR: D1 receipt healthcheck 가 **망 단절 시 60초+ healthy 유지**(커널이 소켓을 붙들어서. 브로커 idle 종료 근거는 망이 살아있을 때만 성립) ·
  D2 fake-useragent 하위핀 UA 불일치 · D3 `dump_routes.py` 산출물에 APM 로그 혼입

## 20:50 오케스트레이터 → D2(UA) 실체 규명 — **보고서에 실을 판단 사항**
QA 가 "UA 가 Chrome 116/117"이라 한 건 맞지만, **어느 역할에 어떻게 물리는지가 갈린다.** 실측:

| 역할 | 쓰는 속성 | 어디에 물리나 |
|---|---|---|
| thread | `ua.random` | **HTTP 요청 헤더**(`roles/thread/config.py:211,216,235,249,264`) |
| delegate | `ua.random` | **HTTP 요청 헤더**(`roles/delegate/config.py:223,228,247,261,276`) |
| receipt | `ua.chrome` | **브라우저 컨텍스트 자체**(`roles/receipt/crawlers/crawling_utils.py:26` → `kcs_receipts_crawler.py:33` · `ksta_receipts_crawler.py:43` 의 `browser.new_context(user_agent=driver_ua)`) |

즉 **국세청·KSTA 가 보는 값이 바뀐 건 receipt 뿐**이다. 원본 receipt 는 `fake-useragent==2.2.0`,
통합본은 thread/delegate 핀인 `1.4.0`. 실측한 `.chrome` 반환값:
- 2.2.0 → `Mozilla/5.0 (Linux; Android 10; K) ... Chrome/135.0.0.0 **Mobile** Safari/537.36`  ← **모바일**
- 1.4.0 → `Mozilla/5.0 (X11; Linux x86_64) ... Chrome/117.0.0.0 Safari/537.36`  ← **데스크톱**

thread/delegate 는 원래도 1.4.0 이라 **회귀 아님**(`ua.random` 풀 불변).
receipt 는 운영에서 모바일 UA 로 긁고 있었는데 통합본은 데스크톱 UA 로 간다 — **대상 사이트가 보는 값이 달라진다.**
버전 숫자만 보면 데스크톱이 더 일관돼 보이지만(엔진이 데스크톱 크로미움 140), **국세청·KSTA 가 실제로 무엇을 받아주는지는
접속 금지 제약상 검증할 수 없다.** 사용자 판단 필요 — 최종 보고 "판단 필요"에 싣는다.

## 22:10 team-dev → UA 상향 완료, 그러나 **내 전제가 틀렸음이 드러남**
- 작성자 발견: receipt 는 `ua.chrome` 을 직접 안 쓴다. `roles/receipt/crawlers/crawling_utils.py:23-31` 의
  **`get_ua()` 가 모바일을 걸러내는 while 루프**다. 즉 **운영이 국세청·KSTA 에 보내는 값은 데스크톱 Chrome/135**.
  내가 20:50 에 "운영=모바일 135" 라고 쓴 건 `ua.chrome` 을 직접 잰 값이라 **경로가 틀렸다.**
- **결정(2.2.0 채택) 자체는 그대로 옳다** — 운영과 같은 풀(Chrome 133~135 데스크톱)을 복원하고,
  1.4.0 이 섞던 Opera 20% 가 사라진다. 실측: 상향본 `get_ua()` 표본 60건 **모바일 유출 0건**, Chrome 135:31·134:18·133:4

## 22:20 오케스트레이터 → 대가 실측 + 해법 확인
작성자가 남긴 한계가 사실인지 직접 쟀다. **사실이고, 생각보다 크다.**

| | `ua.random` 모바일 비율 | 브라우저 분포 |
|---|---|---|
| 1.4.0 (통합 전 thread/delegate) | **0%** | Firefox 42% · Chrome 30% · Edge 14% · Opera 4% |
| 2.2.0 (상향본) | **68~87%** (표본에 따라) | Chrome계열 42% · 기타(모바일 등) 50% |

thread/delegate 는 이 값을 **HTTP 헤더**로 내보내는데(`roles/thread/config.py:211~264` 등 9개 dict),
그중 **7개가 `sec-ch-ua-mobile: "?0"` + `sec-ch-ua-platform: "Windows"` 를 하드코딩**한다(실측 확인).
게다가 dict 가 **모듈 최상위 리터럴**이라 `ua.random` 이 import 시 1회만 평가돼 **프로세스당 고정**된다.
→ "UA 는 아이폰인데 클라이언트 힌트는 윈도우 데스크톱"이 프로세스 수명 내내 굳는다. 봇 판별 신호다.

**해법 실증**: 역할별로 다른 버전을 물릴 수 있다.
```
pip install --target /opt/ov fake-useragent==1.4.0  → PYTHONPATH=/opt/ov 로 우선
  기본(2.2.0)      : 모바일 85%
  오버라이드(1.4.0): 모바일 0%,  모듈경로 /opt/ov/fake_useragent
  용량 140K
```
→ **base=2.2.0(receipt) + thread/delegate 만 1.4.0 오버라이드** 하면 **세 역할 전부 운영과 동일**해진다.
`roles/*` 무수정 원칙도 안 깨진다(Dockerfile + entrypoint 만 손댄다). 사용자 결정 "운영 그대로"의 일관 적용.

## 22:40 사용자 지시 → **배포 방식·주소 체계 전환** (개발계 먼저)
> "이어 받는게 아니라 새로 띄운다음에 기존거를 내릴거야" / "일단은 개발계를 먼저 하자 각각 뜨면돼"
> "cname 통신으로 하고 80포트면 되는거 아냐 왜 별도 포트를 다 쓰는거야"
> "서비스name 에 80 포트 내부니까 http 통신 하고, 외부랑은 https로 하면 되고, aws 올 애들은 egress 통해서 사무실로 https로"
> (이름 확인) "응 그렇게해"

**결정 확정**
1. **병행 기동** — 신규를 옆에 띄우고 검증, 기존은 나중에 내린다. `docker-compose.cutover.yml` 은 보관만.
2. **서비스명 = `crawler-worker-thread` / `-delegate` / `-receipt`** (현 컨테이너명 그대로)
3. **내부 통신은 서비스명 + 포트 80 평문 HTTP.** 외부는 HTTPS. AWS→사무실은 egress 로 HTTPS.

**포트 80 전환이 가능한 근거(실측)**
| 확인 | 결과 |
|---|---|
| 기존 3컨테이너 호스트 포트 바인딩 | **0건** — 서비스명으로만 불린다 |
| 같은 망에서 80 동시 사용 | 이미 3개(dev-edge · embedding-bridge · legacy-bridge) — 컨테이너마다 IP 가 달라 충돌 없음 |
| 원본 소스가 자기 listen 포트를 하드코딩? | **안 한다.** 포트는 전부 우리 파일에만 있다 — `docker/entrypoint.sh:14,22`(PORT=) · `Dockerfile:159-160`(EXPOSE) · `docker-compose.dev.yml:46,49,69,71` |
→ **`roles/*` 무수정 원칙을 깨지 않고 80 으로 옮길 수 있다.** 대신 부르는 쪽 설정(`HANDLER_CRAWLER_URL` 등)은 바뀐다.

**부수 발견 — 주소 정리 때 같이 다뤄야 할 것**
`roles/delegate/agents/naver/cafe/services/searchapi.py:84` 에 **하드코딩된 EC2 주소**가 살아 있다:
`http://ec2-43-202-210-112.ap-northeast-2.compute.amazonaws.com:42010/crawling/naver/cafe`
죽은 코드가 아니라 `searchapi.py:152`·`naver_cafe_crawling.py:310` 에서 실제 호출한다.
**delegate 가 다른 delegate 인스턴스를 EC2 공인 주소로 부른다.** thread 쪽 같은 줄은 주석 처리 상태.
`roles/*` 안이라 무수정 원칙에 걸린다 — 고치려면 원본 저장소를 손대야 한다.

## 23:30 team-dev → 역할별 UA 분리 + 병행 기동 완료 (STATUS: DONE, `12-ua-per-role.md`)
- `Dockerfile` builder 에서 `pip install --no-deps --target /opt/ua-compat fake-useragent==1.4.0`(140KB) → runtime COPY.
  `entrypoint.sh` 가 **thread·delegate 에만** 그 경로를 PYTHONPATH 에 끼운다(역할 디렉토리는 계속 최우선).
- 실측: thread/delegate = `/opt/ua-compat` **1.4.0**, receipt = site-packages **2.2.0**.
  `ua.random` 모바일 n=400 → thread 0% · delegate 0%. receipt `get_ua()` n=120 → 모바일 유출 0, Chrome 133~135 80%
- 무한루프 질문 답: `ua.chrome` 데스크톱 비율 p=1381/3000=0.460 → 기대 반복 2.17회(~4.7ms).
  100연속 모바일 확률 1.6e-27. **p=0 일 때만 위험** — 다음 상향 때 이 값만 보면 된다

## 23:35 오케스트레이터 → 숫자 3건 교차확인 (작성자가 "정정"이라 한 것들)
| 항목 | 실측 결론 |
|---|---|
| delegate 라우트 21 vs 16 | **둘 다 맞다. 세는 단위가 다르다** — 21=(경로,메서드) 쌍, 16=고유 경로. 통합 전/후 **양쪽 다 21쌍·16경로로 동일**(직접 재측정). 정정이 아니라 단위 차이 |
| delegate 테스트 38 vs 43 | **마운트 조건 차이.** tests 만 마운트=수집실패 → `--ignore` 하면 38. `fixture_runtime.py` 까지 마운트하면 **43**. 통합 전 원본도 같은 조건에서 **43** → 양쪽 동일. 43 이 더 넓은 검증이라 채택 |
| import 감사 thread 57/8 → 58/7 | 기준 이미지와 일치한다는 보고 그대로 수용(재실행 안 함) |

## 23:40 오케스트레이터 → 병행 상태 독립 확인
- **6개 동시 healthy** (구 3 + 신 3)
- **별칭 3종이 전부 구 컨테이너를 가리킨다**: `crawler-web`→172.28.0.22 · `ai-delegate-web`→172.28.0.23 ·
  `pg-receipt-worker`→172.28.0.30 = 구 컨테이너 IP. 신규(192.168.80.x)는 별칭 **0개**
- **receipt 카프카 격리 확인**: 브로커 `legalcare-local-kafka-1` 에서
  구 그룹 `legalcare-local-pg-receipt-fixture-v1` 멤버 = /172.28.0.30(구) ·
  신 그룹 `crawler-worker-3role-verify-v1` 멤버 = /172.28.0.21(신). **그룹·토픽 모두 분리, 리밸런싱 없음**
