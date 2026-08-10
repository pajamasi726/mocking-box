# round-1 리뷰 대응 — 크롤러 워커 3역할 통합 (responder: claude)

11건 전부 처리했다. **반영 9 · 부분반박 2**(#7 숫자의 정체, #9 union 규칙 위반 여부).
1번 지적이 맞았다 — 그 검사는 자기를 세고 있었고, 문서는 고쳐졌다고 말하고 있었다. 고치고 실증했다.
2번도 맞았다. 그리고 실제로 돌려 봤더니 **"표면이 같다"던 12건 중 3건이 진짜로 다르다.** 리뷰가 없었으면
"검증 안 됨"이 아니라 "틀린 주장"으로 남을 뻔했다.

| # | 판정 | 요약 |
|---|---|---|
| 1 MAJOR healthcheck | **반영** | 이름 매칭 폐기 → tini 자식의 브로커 ESTABLISHED 세션으로 판정. 도커가 실제로 `unhealthy` 로 넘어가는 것까지 실증 |
| 2 MAJOR 의존성 12건 | **반영** | 로컬 스텁으로 실행 A/B. **3건이 실제로 다름**(boto3 체크섬 헤더 · UA 최대 135→117 · certifi 루트 7개 누락), 9건 동일, 실엔드포인트 검증 0건은 한계로 명시 |
| 3 MAJOR 베이스 이미지 | **반영**(범위 정정) | receipt 만 Debian 12→13. thread/delegate 는 원래도 trixie. 떠 있는 태그는 **다이제스트로 고정** |
| 4 MINOR playwright 근거 | **반영** | 주석이 정반대였다. `/proc` 실측으로 정정 |
| 5 MINOR OpenAPI 한 줄 | **반영**(영향 판정 첨부) | 계약 diff 는 사실. **실제 응답·검증에는 영향 없음**, 소비자 0건 |
| 6 MINOR AC5 크레딧 | **반영** | 실브로커 DLQ 왕복을 추가 실증. 못 덮는 범위는 명시 |
| 7 MINOR 실메시지 0건 / 24건 | **반영 + 부분반박** | 실메시지 0건 맞다. 다만 **24는 실재하는 정확한 숫자**다 — 다른 카프카 클러스터 |
| 8 MINOR earliest 함정 | **반영** | compose 주석 + README 재교체 전 확인 절차 |
| 9 MINOR boto3-stubs | **부분반박** | union 규칙 위반 아님(원본 requirements·원본 이미지 둘 다에 있음). 스텁/런타임 버전 어긋남은 인정·문서화 |
| 10 MINOR 롤백 대상의 컨슈머 | **반영** | #1 수정으로 구·신 어느 쪽이든 신호가 생겼다 |
| 11 MINOR 서비스 이름 별칭 | **반영** | compose 서술 정정. **더 큰 위험을 하나 더 찾았다** |

---

## 1. (MAJOR) healthcheck — 이름으로 찾는 걸 그만뒀다

지적대로다. 자기 프로세스를 세는 검사는 무엇을 하든 자기를 잡는다. **첫 토큰이 `python` 인지 보는 식의
정교화도 하지 않았다** — 여전히 이름 매칭이고, 다음에 실행 형태가 조금만 바뀌면 같은 방식으로 무너진다.
대신 **이름이 아니라 위치와 관측 가능한 신호**로 바꿨다(`docker-compose.dev.yml`).

1. **위치** — tini(PID 1)가 exec 한 자식(`ppid==1`)만 워커로 본다. healthcheck·`docker exec` 프로세스는
   PID 네임스페이스 바깥이 부모라 `ppid=0` 으로 잡히므로 **구조적으로 자기를 못 센다.** 실측:
   ```
   1 /usr/bin/tini -- /app/docker/entrypoint.sh   ppid=0
   7 python worker.py                              ppid=1   ← 워커
   980 sh -c (docker exec 로 띄운 것)               ppid=0   ← 걸러짐
   ```
2. **신호** — 그 프로세스가 브로커 포트로 **ESTABLISHED** 인 소켓을 쥐고 있는지(`/proc/<pid>/fd` →
   inode → `/proc/net/tcp`). 포트는 `KAFKA_SERVER_1/2` env 에서 뽑는다.

### 실증 (전부 배포본 그대로, byte-exact)

**(a) 리뷰어 재현 케이스 — 워커가 없는 컨테이너**
```
$ docker run --rm -e CRAWLER_ROLE=receipt crawler-worker:dev-3role python -c "<구검사>"; echo $?
0        ← 워커가 없는데 healthy (리뷰 #1 그대로)
$ docker run --rm -e CRAWLER_ROLE=receipt crawler-worker:dev-3role python -c "<신검사>"; echo $?
1        ← 정상적으로 실패
```

**(b) 도커 자체 health 판정** — 워커 대신 `sleep` 만 도는 컨테이너 2개에 각각 구/신 검사를 달았다.
수동 실행이 아니라 **도커가 내리는 판정**이다.
```
hcfail-old  → healthy   failingStreak=0     (구검사: 끝내 안 잡는다)
hcfail-new  → unhealthy failingStreak=6     (신검사: 잡는다)
  PID 1: /usr/bin/tini -- /app/docker/entrypoint.sh sh -c sleep 300
  PID 7: sh -c sleep 300      ← ppid==1 자식은 있다. 소켓 조건에서 걸린 것
```

**(c) 사고 상태 재현** — 프로세스는 살려 둔 채 컨슈머만 그룹에서 뺐다(`consumer.unsubscribe()`).
파일명까지 `worker.py` 로 맞춰 구·신 검사를 같은 조건에서 비교했다.
```
T+1~T+9분   소켓=[ESTABLISHED ESTABLISHED]   신검사=0   ← 못 잡는 구간
T+10분      소켓=[ESTABLISHED CLOSE_WAIT]    신검사=0
T+11분      소켓=[CLOSE_WAIT  CLOSE_WAIT]    신검사=1   ← 여기서 잡는다
              같은 시점 구검사 exit=0                    ← 구검사는 끝까지 healthy
              워커 PID7 여전히 생존: python /tmp/worker.py
              그룹 crawler-worker-hcdemo-v1: does not exist (Empty 로 소멸)
```
브로커의 `connections.max.idle.ms` 가 600000(개발계 실측, 브로커 기본값)이라 컨슈머가 말을 멈추면
10분 뒤 브로커가 연결을 닫고, 그때 ESTABLISHED 가 사라진다.

### "그 3일 사고를 이 검사가 잡아냈을까" — **잡는다. 3일이 아니라 약 11분이다.**

구 컨테이너 로그를 보면 마지막 처리가 `2026-08-06 10:29:47` 이고 그 뒤 3일간 **로그 한 줄 없이**
그룹만 `Empty` 였다(예외가 났으면 `asyncio.run` 이 끝나 프로세스가 죽었을 텐데 죽지도 않았다).
즉 "프로세스는 살아 있는데 컨슈머가 말을 안 하는" 상태 — 위 (c)가 재현한 바로 그 모양이다.
그 상태에서 구검사는 **끝까지 healthy**, 신검사는 **T+11분에 unhealthy** 로 넘어갔다.
`interval 15s · retries 3` 을 더하면 도커 판정까지 최대 약 11분 45초다.

**못 보는 것은 정직하게 적었다.** 컨슈머 그룹 참여(Stable/Empty)는 컨테이너 안에서 볼 수 없다.
그룹을 떠난 직후 ~10분은 여전히 healthy 다. 그래서 README 에 **밖에서 보는 절차**를 남겼다:
```bash
docker exec legalcare-local-kafka-1 /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 --describe --group legalcare-local-pg-receipt-fixture-v1 --state
# STATE=Stable / #MEMBERS=1 이어야 한다. 교체·롤백 직후에는 이걸로 한 번 더 본다.
```
카프카 admin 프로토콜을 healthcheck 안에서 구현하는 길도 있었지만(코디네이터 탐색 + DescribeGroups),
stdlib 로 짜면 검사가 수십 줄이 되고 브로커 장애 시 오탐이 늘어 택하지 않았다. **소스 수정은 안 했다**
— `roles/*` sha256 동일 불변식은 그대로다(변경 파일 0건).

부수 효과: 배포된 검사 실행 시간 0.05~0.06초(timeout 3s), `start_period` 는 10s→30s 로 올렸다
(카프카 연결이 붙기 전에 실패로 세지 않도록).

## 2. (MAJOR) 낮춘 의존성 12건 — 실제로 돌렸고, 나쁜 소식이 나왔다

"흔들 이유가 없었다"는 추론이었다는 지적이 맞다. (a) 경로로 갔다 — 국세청·KSTA·실 S3·실 Vault 접속
없이, stdlib `http.server` 스텁 S3 / 스텁 Vault / 자체서명 TLS 서버로 **실제 receipt 함수를 호출**하고
통합 이미지 vs receipt 원본 이미지를 A/B 했다(두 이미지의 receipt 소스 8파일 sha256 동일 선확인).

**결과가 다른 3건 — 원래 주장이 틀렸다.**
- **boto3 1.34.34**: `upload_screenshot_to_s3()` 의 PutObject 무결성 헤더가 **CRC32(원본) → Content-MD5(통합)**
  로 바뀐다(botocore 1.36+ flexible checksum 기본값 차이). URL·서명 알고리즘·바디·멀티파트 시퀀스는 동일.
- **fake-useragent 1.4.0**: `get_ua()` 60회 샘플링에서 Chrome 메이저 **최대 135 → 117**(2023-09 수준).
  번들 Chromium 은 양쪽 1187(≈Chrome 139)이다. 04-changes 가 "확인해 뒀다"고 한 그 항목인데,
  동작이 아니라 **값이 낮아진다**는 걸 못 본 것이다. 봇 감지·미지원 브라우저 게이트에 그대로 노출된다.
- **certifi 2024.08.30**: 루트 **7개가 없다**(TrustAsia TLS ECC/RSA, D-TRUST BR/EV Root CA 2 2023,
  OISTE Server Root ECC/RSA G1, SwissSign RSA TLS 2022-1). 다만 **Amazon Root CA 1~4 · Starfield Services ·
  ISRG X1/X2 · DigiCert Global CA/G2 · GlobalSign R3/R6 은 구 번들에도 있다** → S3·Let's Encrypt·DigiCert
  계열은 안전하고 위험은 위 7개 루트에 한정된다.

**같은 9건**: requests·urllib3(Vault KV v2 왕복 3요청 요청라인·헤더집합·반환 해시 일치), TLS 검증 동작
(커스텀 CA 성공 / 기본 번들 실패 예외 타입까지 동일), idna·six·pytz·python-dotenv·charset-normalizer.
이 중 pytz(`utils.now_date()` 호출처 0)·dotenv(`ALLOW_DOTENV=false`)·idna(호스트가 전부 ASCII)는
프로덕션 경로에서 dead code 이거나 no-op 이다. 덤: OpenSSL 은 통합본이 **더 높다**(3.5.6 vs 3.0.16).

**검증 못 한 것을 목록으로 남겼다**(README "낮아진 12건" 절 · 04-changes 한계 절): 실 S3 가 Content-MD5
형태를 받는지(`s3_save_review_boost_receipt.py:18-30`), 실 Vault TLS 체인이 구 번들로 검증되는지
(`vault.py:34,37,68`), KCS/KSTA 가 Chrome 117 UA 를 막는지(`crawlers/crawling_utils.py:23-30`),
그리고 실왕복 전체. **실 엔드포인트 검증은 여전히 0건**이라고 적었다.

## 3. (MAJOR) 베이스 이미지 — 사실 인정, 범위는 receipt 한정

문서에 없었던 것 맞다. 다만 **"3역할이 갈아탔다"가 아니라 receipt 한 역할이다.** 실측:

| 역할 | 원본 FROM | 원본 이미지 | 통합 | 변화 |
|---|---|---|---|---|
| thread | `python:3.12-slim` | 3.12.13 · Debian 13.6 | 3.12.13 · 13.6 | 없음 |
| delegate | `python:3.12-slim` | 3.12.13 · Debian 13.6 | 3.12.13 · 13.6 | 없음 |
| receipt | `python:3.12.10-slim` | 3.12.10 · **Debian 12.11** | 3.12.13 · 13.6 | **12 → 13** |

README 에 표로 명시하고 t64 치환(`libatk1.0-0t64`·`libcups2t64`)도 적었다.

**태그 고정은 했다.** `Dockerfile` 에 `ARG PYTHON_BASE=python:3.12-slim@sha256:229a2c5b…` 를 넣고
두 스테이지가 모두 그걸 쓴다. 판단 근거:
- **지금 이미지가 안 바뀐다.** 고정본으로 다시 빌드해 rootfs 레이어가 `crawler-worker:dev-3role` 과
  **전부 동일**함을 확인했다(10초 캐시 히트, 레이어 목록 일치). 즉 이번에 검증한 그 베이스를 박은 것이라
  thread/delegate 동작도 바뀌지 않는다. 오케스트레이터가 우려한 "고정하면 기존 동작이 바뀐다"는
  **receipt 원본의 3.12.10-slim 으로 되돌릴 때** 생기는 문제고, 그 방향은 택하지 않았다 — 그러면
  원래 trixie 였던 두 역할이 검증 안 된 조합으로 내려간다.
- 대가는 보안 패치 자동 추적이 끊기는 것. 올릴 때 `ARG` 한 줄 바꾸고 AC1~AC3 재검증하라고 적어 뒀다.

## 4. (MINOR) playwright 근거 문구 — 정반대였다. 고쳤다

리뷰어가 맞다. 통합 이미지에서 `headless=True` 로 띄우고 `/proc` 를 훑으면 **chromium 프로세스 7개가
전부 `chromium_headless_shell-1187/chrome-linux/headless_shell`** 이고 594MB 짜리 `chromium-1187` 은
한 프로세스도 안 뜬다. 렌더러 인자가 `--headless=old` 라 "1.49 부터 new headless 가 기본" 도 틀렸다.
(오해의 출처로 보이는 것: `executable_path` 프로퍼티는 full chromium 경로를 돌려준다.)
`headless=False` 3곳은 전부 `__main__` 스크립트이고 저장소 전체에서 import 0건, `channel=` 사용 0건,
이미지에 Xvfb·`DISPLAY` 도 없다. `libu2f-udev` 는 Debian 13 에서 **파일 0개인 transitional 패키지**이고
`libappindicator3-1` 은 `libayatana-appindicator3-1` 로 치환됐는데, `headless_shell` 의 의존성 46개
어디에도 안 나온다(`ldd` 0건).

→ `Dockerfile:23-40` 주석을 **"union 이라서 넓다"** 로 바꾸고 위 사실을 근거와 함께 적었다.
04-changes 의 `--only-shell` 서술도 뒤집어 정정했다 — `--only-shell` 이면 **원본 receipt(Dockerfile:77)와
같은 바이너리 구성**이 되고 594MB 가 빠진다. 지금 안 바꾼 이유는 재빌드 후 AC1~AC3 재검증이 필요해서다
(후속 여지로 주석에 명시).

## 5. (MINOR) OpenAPI 한 줄 — 사실이고, 실제 응답에는 영향 없다

diff 재현했다(thread 373→374줄, delegate 824→825줄, 둘 다 `CheckKakaoIdData.properties.cookieDict` 에
`"additionalProperties": true` 추가). 원인은 `reputation_dto.py:79` 의 bare `dict`.
**영향 판정: 없음.** 같은 모델을 8가지 입력(정상/본문 추가키/중첩 dict 추가키/빈 dict/list/문자열/None/정수키)으로
두 이미지에서 돌려 `model_dump_json` 과 ValidationError `(loc,type)` 이 전부 동일했다.
`model_config` 는 양쪽 다 비어 있어 `extra` 정책이 바뀐 게 아니고, `additionalProperties` 는
`model_json_schema()` 산출물에만 나타나는 문서 필드다. **OpenAPI 로 클라이언트를 뽑는 소비자도 없다** —
99개 저장소에 코드젠 설정 0건, `/openapi.json` fetch 0건, 실제 호출자는 전부 수기 클라이언트
(`MlService.kt:53,128,149` · `DelegatorService.java:9-13` · `AiDelegatorService.java:14-17`).
즉 영향 범위는 `/docs` 화면 한 줄이다. 리뷰가 지적한 "라우트 A/B 가 스키마를 보증하지 않는다"는 맞고,
그 근거가 04-changes 에 없었다는 것도 맞다 — 위 결과를 한계 절에 넣었다.

## 6. (MINOR) AC5 크레딧 — 덮는 범위를 좁게 적고, DLQ 를 실제로 돌렸다

지적 수용. 추가로 **실브로커 DLQ 왕복**을 실증했다(검증 토픽·검증 그룹, 실토픽 무관):
```
kcs 1→3   completed 1→2   dlq 0→1
completed: {"storeId":910301,...,"imageUrl":"https://fixture.invalid/legalcare/receipt.png"}
dlq      : {"schemaVersion":1,"sourceTopic":"...kcs","partition":0,"offset":2,
            "errorType":"ValidationError","storeId":910401,...,"invalidFields":["paymentDate"]}
group crawler-worker-3role-verify-v1  kcs 3/3 LAG 0
```
여전히 못 덮는 것: `ENABLE_EXTERNAL_ACTIONS=false` 라 `worker.py:79-83` 의 KCS/KSTA 크롤러 분기·S3·Vault 는
실행되지 않았다. 04-changes 한계 절에 그대로 적었다. 커밋 오프셋만으로 성공/DLQ 를 구분할 수 없다는
지적도 맞다 — 그래서 이번에도 completed/dlq 카운트를 같이 적었다.

## 7. (MINOR) 실메시지 0건 — 맞다 / 24건 — **부분 반박**

**앞부분 수용**: fixture 토픽 4종은 `retention.ms=86400000` 이라 보관 레코드가 0건이고, 교체된 컨테이너는
실메시지를 한 건도 안 먹었다. "Stable(1) 정상 참여"는 그룹 참여까지라고 04-changes 에 명시했다.

**뒷부분 반박**: "그 24가 무엇을 센 숫자든 이 토픽은 아니다"는 맞지만, **24는 실재하고 정확한 숫자다.**
DELL 에 카프카 클러스터가 둘 있다.
- `kafka1`/`kafka2`(compose 프로젝트 `kafka`, 망 `kafka_default`) — `medilawyer.crawling.review-boost.receipt.crawling` **24** · `...ksta.receipt.crawling` **1** · `...notification` **0**. 00-log 의 세 숫자와 정확히 일치한다.
- `legalcare-local-kafka-1`(별칭 `kafka`, 172.28.0.3) — **워커가 실제로 붙는 브로커.** fixture 토픽 kcs 2 · ksta 1 · completed 2 · dlq 1.

두 클러스터는 망이 달라 서로 못 본다. 그러니 00-log 는 틀린 게 아니라 **브로커를 안 적은 것**이고,
정정이 아니라 한정어를 붙였다(`00-log.md:21` 아래 주석). 가정 A1 은 어느 숫자로 읽어도 성립한다.

덤으로 하나 나왔다: `legalcare-local-kafka-1` 에도 `medilawyer.crawling.review-boost.receipt.notification`
토픽이 있고 `legalcare-local-booster-worker-1` 이 붙어 있는데, receipt 는 fixture `completed` 로 발행한다
→ **개발계 receipt→booster 루프는 이번 작업 이전부터 끊겨 있었다.** 00-log 에 적었다.

## 8. (MINOR) `earliest` 리플레이 함정 — 반영

`docker-compose.dev.yml` 의 해당 env 에 주석으로 조건을 적고(빈 그룹 오프셋은 `offsets.retention.minutes`
기본 7일 후 소멸 → 그 창을 넘긴 재교체 + 잔존 레코드 = 전량 재소비·재발행), README 에 **"리플레이 함정"**
절과 재교체 전 확인 명령(`--describe` 로 CURRENT-OFFSET 생존 확인, 없으면 `latest` 로 한 번 띄워 오프셋
생성 후 되돌리기)을 넣었다.

## 9. (MINOR) `boto3-stubs` — **부분 반박**

"union 규칙과 어긋난다 / 그냥 들어간 셈"은 사실이 아니다. `pg-receipt-crawler/requirements.txt:3` 에
`boto3-stubs==1.40.51` 이 있었고, **원본 receipt 이미지에도 실제로 설치돼 있다**(`boto3-stubs 1.40.51` ·
`botocore-stubs 1.43.14` · `types-awscrt` · `types-s3transfer`). 통합본은 원본을 따른 것이고, 오히려
`types-awscrt` 가 빠져 원본보다 덜 재현됐다. 용량도 약 1.4MB(이미지의 0.04%)라 근거가 못 된다.

**다만 리뷰가 짚은 실질 문제는 인정한다**: 스텁은 boto3 1.40.51 을 기술하는데 런타임은 1.34.34 라
IDE·mypy 가 없는 API 를 있다고 말한다. 고칠 방향은 제거가 아니라 `boto3-stubs==1.34.34` 로 런타임에
맞추는 것이다. 지금 안 바꾼 이유: `requirements.txt` 를 건드리면 재빌드가 필요하고 그러면 이번에
검증한 이미지가 아니게 된다. README "의존성" 절에 어긋남과 처리 방향을 적었다.

## 10. (MINOR) 롤백이 되살리는 상태 — 반영

지적대로 되돌아가는 그 상태가 "그룹 멤버 0인데 healthy" 였다. **#1 수정으로 신규 쪽에는 신호가 생겼고**
(약 11분 내 unhealthy), 구 컨테이너(`os.kill(1,0)`)에는 여전히 없다. 그래서 README 롤백 절차 옆에
브로커 그룹 조회를 **교체·롤백 직후 필수 확인**으로 적었다. 구 컨테이너의 healthcheck 자체는
`legalcare-local` 소속이라 이 작업 범위에서 바꾸지 않는다(R6). SIGTERM 개선 관찰은 감사히 받는다.

## 11. (MINOR) 서비스 이름이 별칭으로 붙는다 — 반영, 그리고 더 큰 것 하나

`docker-compose.dev.yml` 의 "별칭 없이 붙는다"가 틀렸다는 지적이 맞다. 실측 재확인:
```
crawler-worker-receipt  legalcare-local_legalcare => [crawler-worker-receipt receipt pg-receipt-worker]
```
compose 서술을 고치고 README 에 "네트워크 별칭" 절을 넣었다. 지금 충돌은 없다 — 공용망 컨테이너 전수의
별칭에 `thread`/`delegate`/`receipt` 는 0건이고 DNS 도 각각 IP 하나만 돌려준다.

**추가로 찾은 것**: `/mnt/ex_disk1/renew-replay/docker-compose.python.yml` 이 `crawler-web` ·
`ai-delegate-web` · `pg-receipt-worker` **세 서비스를 아직 그대로 정의하고 있다.** 그 묶음이 다시 `up`
되면 별칭 3개가 즉시 이중 등록된다 — 서비스 이름 3개보다 이쪽이 확실한 위험이라 README 에 같이 적었다.

---

## 수정 파일

| 파일 | 무엇을 | 지적 |
|---|---|---|
| `docker-compose.dev.yml` | receipt healthcheck 교체(+`start_period` 30s) · `earliest` 함정 주석 · 별칭 서술 정정 | #1 #8 #11 |
| `Dockerfile` | 베이스 다이제스트 고정(`ARG PYTHON_BASE`, 두 스테이지) · apt 근거 주석 정정 | #3 #4 |
| `README.md` | receipt healthcheck 절(+그룹 확인 절차) · Debian 12→13 표와 고정 근거 · 낮아진 12건 실행 결과와 미검증 목록 · boto3-stubs · 네트워크 별칭 · 리플레이 함정 | #1~#4 #8 #9 #11 |
| `04-changes.md` | 한계 절 4개 정정 + 3개 추가(AC5 범위 · 실메시지 0건) | #1 #2 #3 #4 #6 #7 |
| `00-log.md` | 24건에 브로커 한정어 추가 | #7 |

**`roles/*` 수정 0건** — R1 sha256 동일 불변식 유지. 원본 5개 저장소 무수정. 커밋하지 않았다.
이미지도 재빌드하지 않았다(healthcheck 는 compose 에 있고, Dockerfile 변경은 다이제스트 고정 + 주석이라
rootfs 레이어 동일 확인 완료).

## 재검증 결과 (DELL, 수정 후)

```
[1] crawler-worker-receipt   Up (healthy)   crawler-worker:dev-3role   ← 새 healthcheck 로 재기동
    crawler-worker-thread    Up (healthy)
    crawler-worker-delegate  Up (healthy)
[2] legalcare-local_legalcare 망 28개 · 전부 running (AC6 기준선과 동일)
[3] crawler-web/health 200 · ai-delegate-web/health 200
[4] group legalcare-local-pg-receipt-fixture-v1 → Stable, #MEMBERS=1
[5] env 13/13 승계 유지 · 별칭 [crawler-worker-receipt receipt pg-receipt-worker] · pg-receipt-worker → 172.28.0.21
[6] 임시 컨테이너 잔여 0개 (hcdemo/hcfail-*/verify-receipt/depverify-*/factcheck-* 전부 제거)
[7] docker compose -f dev -f cutover config → 파싱 OK
[8] 고정 다이제스트 재빌드 → rootfs 레이어 dev-3role 과 전부 동일 (10초, 전 레이어 캐시 히트)
```
교체 작업은 전부 `docker compose -p crawler-worker` 안에서만 했다. `legalcare-local` 스택과
`/mnt/ex_disk1/renew-replay` compose 묶음은 어느 단계에서도 호출하지 않았다.

## 남은 것 (다음 사람에게)

- 실 S3 · 실 Vault · 실 국세청/KSTA 대상 검증 0건. `ENABLE_EXTERNAL_ACTIONS=true` 를 켤 때 위 3건
  (boto3 체크섬 헤더 · UA 117 · certifi 루트 7개)을 먼저 본다.
- `--only-shell` 로 594MB 감축 + `boto3-stubs` 1.34.34 정렬 — 둘 다 재빌드 필요라 묶어서 처리.
- healthcheck 는 컨슈머 그룹을 못 본다. 교체·롤백 직후에는 브로커 조회를 손으로 한 번 더.

STATUS: REVISED
