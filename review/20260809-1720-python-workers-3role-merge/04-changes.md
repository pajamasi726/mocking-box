# 변경 요약: 크롤러 워커 3역할 통합 (playwright 1.55 상향 + 개발계 교체)

## 변경 파일
| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `roles/receipt/` (신규 17파일) | pg-receipt-crawler **작업트리** 무수정 복제(git HEAD 아님 — `tests/` 가 미추적이라 커밋 기준이면 통째로 빠진다) | R1 |
| `docker/entrypoint.sh` | receipt 분기 → cwd=`roles/receipt` · `python worker.py`. 오타·미지정 역할 exit 64 유지 | R2 |
| `Dockerfile` | receipt allowlist COPY(원본 COPY 목록 그대로) · 1.55 new-headless 용 런타임 라이브러리 보강 · screenshots 디렉토리 | R1·R2 |
| `requirements.txt` | 3-way union. 충돌 18건 해소, 상향은 playwright·pydantic 과 그 폐포 7건뿐 | R3 |
| `docker-compose.dev.yml` | 검증용 receipt(격리 토픽·그룹) + 3역할 healthcheck | R5·AC4 |
| `docker-compose.cutover.yml` | receipt 서비스 · 별칭 `pg-receipt-worker` · 교체 대상 env 실측값 승계 | R5·R7 |
| `README.md` | 3역할 구조 · 교체/롤백 절차 · 의존성 결정 근거 | R7 |
| `.env.example` | receipt 전용 env(토픽 4종 · FAILURE_POLICY · Vault) | R7 |

**R4 발동 없음** — 1.55 에서 깨진 코드가 하나도 없어 `roles/PATCHES.md` 를 만들지 않았다(R1 무수정 원칙 유지).
교체 전 실측 덤프는 `00-baseline.md`. 원본 5개 저장소 무수정(오늘 mtime 변경 파일 0건).
⚠️ `roles/receipt/` 는 **미추적**이라 `git diff HEAD` 에 안 잡힌다(커밋 금지 지시라 `git add` 도 안 했다).
그쪽은 `git status --short` + 디렉토리를 직접 보거나, 위 sha256 대조 결과로 갈음하면 된다.

## 테스트 결과
전부 DELL(x86) `crawler-worker:dev-3role` 실측. 상향 전 `crawler-worker:dev-consolidated` 와 통합 전 원본 이미지 2종을 남겨 A/B 에 썼다.

- **R1 양방향 sha256**: 원본 17 / 사본 17 · 누락 0 · 추가 0 · 불일치 0. 이미지 안 13개(allowlist)도 해시 일치
- **`pip check` → `No broken requirements found`** · 3역할 `compileall` exit 0 · 역할 오타 exit 64
- **import 감사**(역할 소스 모듈 전수를 실제로 import, 통합 전 원본 이미지와 A/B): thread 65개(ok 57) · delegate 84개(ok 81) · receipt 12개(ok 12) → **회귀 0건.** 실패분은 old 와 사유 문자열까지 동일(자격정보 없음 · `--network none` · webdriver_manager 미설치)
- **라우트 A/B**(`ENABLE_EXTERNAL_ACTIONS=true` + `--network none`): thread 14쌍 · delegate 21쌍 **0 diff**. receipt 는 fastapi/uvicorn/starlette 참조 0파일 → **라우트 0**
- **chromium 실구동**(3역할 각각 · 외부 접속 없이 로컬 HTML): 140.0.7339.16 기동, `ElementHandle.type` · 중첩 iframe `content_frame` · `Frame.select_option` · `expect_popup` · `locator.all` · `evaluate(arg)` · 페이지/엘리먼트 스크린샷 전부 PASS
- **스위트**: delegate `pytest tests -q --ignore=tests/test_fixture_runtime.py` → **38 passed**(상향 전 이미지도 38 passed) · receipt `pytest tests/test_worker.py -q` → **4 passed**(원본 이미지 `unittest discover` 도 4 OK) · thread 는 원본에 스위트 없음
- **카프카 왕복**: 검증 토픽에 1건 발행 → `...completed` 1건, `...dlq` 0건, LAG 0, 그룹 Stable(1 member)

## AC 자가 점검
- AC1 ✅ 3역할 chromium 실구동 + 크롤러가 실제로 쓰는 API 전수 재현 PASS (오케스트레이터 정정본 — 국세청·KSTA 실접속 없음)
- AC2 ✅ thread/delegate 라우트 집합 0 diff · receipt 라우트 0 산출물 확보
- AC3 ✅ delegate 38 / receipt 4 로 통합 전과 동일
- AC4 ✅ 신규 3컨테이너 healthy · 구 3컨테이너 exited · 별칭 3종이 각각 신규 IP 하나만 가리킴 · `crawler-web`/`ai-delegate-web` `/health` 정상(+`/check_id` 는 503 외부동작 차단 가드 동작)
- AC5 ✅ 단, **실토픽 대신 검증 토픽**으로 했다(오케스트레이터 제약 7). 개발계 receipt 는 원래부터 fixture 토픽만 보므로 실토픽 오염 자체가 없다
- AC6 ✅ 교체 전 28 / 교체 후 28, 전부 running. 차이는 대상 3컨테이너 치환뿐
- AC7 ✅ 신→구→신 왕복 1회 성공. 매 단계 별칭·health 확인

## 알려진 한계 / 리뷰어에게

> **round-1 리뷰 반영으로 아래 4개 항목이 정정·확장됐다.** 자세한 대응은 `round-1-response.md`.

- **하위 핀 유지 12건**: receipt 는 원본보다 낮은 버전으로 돈다. boto3 계열은 `aiobotocore[boto3]==2.11.2` 가 `botocore<1.34.35` 를 못박아 선택지가 없었다. ~~나머지는 표면이 동일해 흔들지 않았다~~ → **round-1 에서 로컬 스텁으로 실제 실행해 A/B 했더니 3건이 실제로 다르다**: boto3(PutObject 무결성 헤더 CRC32→Content-MD5) · fake-useragent(UA Chrome 최대 135→117) · certifi(루트 7개 누락). 나머지 9건은 동일. **실 S3·실 Vault·실 국세청 대상 검증은 여전히 0건이다** — 다음 사람이 `ENABLE_EXTERNAL_ACTIONS=true` 를 켜면 거기가 처음 도는 코드다. 노출 함수 목록은 README "낮아진 12건" 절.
- **receipt healthcheck 는 ~~프로세스 존재까지만 본다~~ → 고쳤다.** 옛 검사(`worker.py` 문자열 매칭)는 검사 자신의 cmdline 에 그 문자열이 있어 **항상 통과했다**(리뷰 #1, 실증 완료). 지금은 tini 의 자식(`ppid==1`)이 브로커와 맺은 ESTABLISHED 세션을 본다. 워커 없는 컨테이너에서 도커가 실제로 `unhealthy` 로 넘어가는 것까지 확인했다. **컨슈머 그룹 참여는 여전히 컨테이너 안에서 못 본다** — 다만 그룹을 떠나면 브로커가 idle 연결을 끊어 약 11분 뒤 unhealthy 로 넘어간다(실측). 즉시 확인은 README 의 브로커 조회 절차.
- **receipt 는 베이스 OS 가 Debian 12 → 13 으로 바뀌었다**(리뷰 #3). 원본 `python:3.12.10-slim`(bookworm/3.12.10) → 통합 `python:3.12-slim`(trixie/3.12.13). thread/delegate 는 원래도 trixie 라 변화 없다. 떠 있는 태그라 다음 빌드가 또 움직이는 문제는 **다이제스트 고정으로 막았다**(rootfs 레이어 동일 확인 = 현재 이미지 무변화).
- 검증 전용 토픽 4종(`legalcare.test.3role.receipt.*`)을 개발계 브로커에 **남겨 뒀다**. `docker-compose.dev.yml` 이 참조하므로 지우면 검증 경로가 깨진다. 실토픽·실그룹은 건드리지 않았다.
- 이미지가 2.51GB → **3.14GB**. ~~playwright 1.55 가 chromium 과 headless-shell 을 둘 다 깔아서고, `--only-shell` 로 줄이면 new headless 동작과 달라져 회귀 위험~~ → **틀린 설명이었다**(리뷰 #4). 실제로 뜨는 건 headless-shell 하나뿐이고 594MB 짜리 full chromium 은 한 프로세스도 안 뜬다(`/proc` 실측). `--only-shell` 이면 **원본 receipt 와 같은 바이너리 구성**이 되고 594MB 가 빠진다 — 재빌드+AC1~AC3 재검증이 필요해 이번 범위에서만 제외했다.
- **AC5 왕복의 덮는 범위**(리뷰 #6): 최초 왕복은 성공 분기 하나뿐이었다. round-1 에서 **실브로커 DLQ 왕복을 추가로 실증**했다(정상 1건 → `completed` +1, 스키마 위반 1건 → `dlq` +1, `invalidFields:["paymentDate"]`, LAG 0). 여전히 `ENABLE_EXTERNAL_ACTIONS=false` 라 KCS/KSTA 크롤러 분기(`worker.py:79-83`)·S3·Vault 는 실행되지 않았다.
- **교체된 컨테이너는 실메시지를 한 건도 안 먹었다**(리뷰 #7). fixture 토픽 4종은 `retention.ms=86400000` 로 보관 레코드가 0건이라 먹을 게 없다. "Stable(1) 정상 참여"는 **그룹 참여까지만**이라는 뜻이다.
- import 감사에서 통합 전 delegate 이미지 모듈 수가 163, 신규가 84 로 보이는 건 **옛 이미지에 macOS AppleDouble(`._*.py`) 79개가 섞여 있어서**다(현재 원본 저장소·사본 둘 다 0개). 회귀가 아니라 청소된 것이다.
- 커밋하지 않았다. 개발계는 신규 스펙(3역할 healthy)으로 두고 마쳤고, 되돌리기는 `00-baseline.md` 절차 그대로 `docker start` 한 번이다.

STATUS: DONE
