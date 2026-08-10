# round-1 교차 리뷰 — 크롤러 워커 3역할 통합 (reviewer: claude, 권한 review)

diff·`roles/receipt/` 원본·DELL 실측을 직접 봤다. 오케스트레이터가 이미 확인한 8개 항목(sha256·`.env` 혼입·
이미지 내 버전·라우트 집합·스위트·별칭·28컨테이너·원본 무수정)은 재검증하지 않고, 그 검사들이 못 보는
자리만 팠다. 결론부터: **교체 자체는 되돌릴 수 있는 상태로 잘 해 놨는데, "healthy" 라는 신호 하나가 가짜다.**
그리고 이번에 안 켜진 코드(외부 접속 경로)의 의존성이 원본보다 낮아졌는데 그게 도는지는 아무도 안 봤다.

먼저 확인된 것부터. **AC7 왕복 리허설은 진짜다** — 구 3컨테이너 `StartedAt` 이 오늘 09:33:37~09:33:41,
`FinishedAt` 09:35:14~09:35:44 이고 신규 3개 `Created` 가 09:35:45 다. 신→구→신이 실제로 돌았다.
**env 승계도 한 칸씩 맞다** — 00-baseline 대비 thread 10/10, delegate 9/9(`ENABLE_EUREKA=false` 와
`ENABLED_CRAWLING_CHANNELS` 나열 순서까지), receipt 13/13(+receipt 가 안 읽는 `ELASTIC_APM_ENABLED` 하나 더).
**allowlist 도 지켜졌다** — 돌고 있는 이미지에서 `find /app` 하면 `docker/entrypoint.sh`·`requirements.txt`·
`roles/{thread,delegate,receipt}` 뿐이고 `.env*`·`tests`·`Dockerfile`·`docker-compose*`·fixture 는 0건이다.

---

## 1. (MAJOR) receipt healthcheck 는 항상 통과한다 — 자기 자신을 세고 있다

`docker-compose.dev.yml:97`

```
test: ["CMD","python","-c","import os,sys; sys.exit(0 if any(b'worker.py' in open(f'/proc/{p}/cmdline','rb').read() for p in os.listdir('/proc') if p.isdigit()) else 1)"]
```

`/proc/<pid>/cmdline` 을 훑어 `worker.py` 를 찾는데, **이 검사 프로세스 자신의 cmdline 안에 그 문자열이
들어 있다**(`-c` 소스 코드에 리터럴로 박혀 있으니까). 그래서 워커가 죽어도 자기를 발견하고 0 을 돌려준다.
worker.py 가 안 뜬 컨테이너에서 같은 명령을 그대로 돌려 확인했다:

```
$ docker run --rm -e CRAWLER_ROLE=receipt crawler-worker:dev-3role \
    python -c "import os,sys; sys.exit(0 if any(b'worker.py' in open(f'/proc/{p}/cmdline','rb').read() for p in os.listdir('/proc') if p.isdigit()) else 1)"
$ echo $?
0                      # ← 프로세스 목록에 tini 와 이 검사 자신뿐인데도 healthy
```

무게가 큰 이유는 세 가지다. 첫째, **AC4 의 "3컨테이너 healthy" 가 receipt 에 대해서는 공허하다** — 컨테이너가
살아 있기만 하면 무조건 healthy 다. 둘째, `docker-compose.dev.yml:95-96` 과 README("실제 worker.py 프로세스를
직접 확인한다")가 **하지 않는 일을 한다고 적어 놨다.** 셋째, 04-changes 에서 스스로 지적한 사고 —
구 컨테이너가 3일간 healthy 인데 컨슈머 그룹은 `Empty` 였던 그 사고 — 를 **같은 실명 상태로 재현**했다.
원본 `os.kill(1,0)` 보다 나빠지진 않았지만 나아지지도 않았고, 문서만 나아졌다고 말한다.

방향만 적자면 자기 PID 제외(`p != str(os.getpid())`)로는 부족하다. `docker exec` 로 healthcheck 를 띄우는
쉘까지 셀 수 있으니, `/proc/<pid>/cmdline` 의 **첫 토큰이 `python` 이고 두 번째가 `worker.py`** 인지로
좁히거나(=`-c` 실행은 두 번째 토큰이 `-c` 라 안 걸린다), 아예 컨슈머 그룹 참여를 보는 편이 낫다.

## 2. (MAJOR) 낮춘 의존성 12건은 **한 줄도 안 돌려 봤다**

이미지 안 실측(`pip list`, 왼쪽이 통합본 / 오른쪽이 receipt 원본):

```
boto3 1.34.34 / 1.40.51      botocore 1.34.34 / 1.40.51   s3transfer 0.10.3 / 0.14.0
requests 2.31.0 / 2.32.5     urllib3 2.0.7 / 2.5.0        certifi 2024.8.30 / 2025.10.5
idna 3.10 / 3.11             six 1.16.0 / 1.17.0          pytz 2024.2 / 2025.2
python-dotenv 1.0.1 / 1.1.1  charset-normalizer 3.4.0     fake-useragent 1.4.0 / 2.2.0
```

이걸 쓰는 코드는 전부 `ENABLE_EXTERNAL_ACTIONS` 뒤에 있다 — `roles/receipt/config.py:43-44`(Vault),
`roles/receipt/s3_save_review_boost_receipt.py:9-16`(boto3), `roles/receipt/worker.py:76-77`(크롤러 import).
그런데 개발계 컨테이너도, 4개 유닛 테스트(`tests/test_worker.py:7`)도, AC5 카프카 왕복도 전부
`ENABLE_EXTERNAL_ACTIONS=false` 로 돌았다. **즉 낮춘 버전으로 hvac·requests·urllib3·certifi·boto3 가
실행된 적이 한 번도 없다.** `pip check` 는 선언 충돌만 본다.

`fake-useragent` 만은 확인해 뒀다 — 이미지 안에서 `crawlers.crawling_utils.get_ua()` 를 직접 호출해
1.4.0 에서 데스크톱 UA 를 돌려주는 것까지 봤다. 그 주장은 맞다. 나머지 11건은 근거가 "표면이 같다"는
추론뿐이고, 특히 **certifi 가 14개월 낡은 CA 번들**이라는 점은 `vault.medilawyer.co.kr` HTTPS 와
S3 엔드포인트에 그대로 걸린다(원본은 2025.10.5). urllib3 2.0.7 도 그 사이 보안 수정이 여러 번 있었다.
범위를 벗어나 올리라는 얘기가 아니다 — **"이 12건은 검증하지 않았다"가 04-changes 의 한계 절에
"흔들 이유가 없었다" 대신 적혀야 한다.** 다음 사람이 EXTERNAL 을 켜는 순간 처음 도는 코드다.

## 3. (MAJOR) receipt 가 Debian 12 → 13 으로 갈아탔는데 어디에도 안 적혀 있다

```
legalcare/pg-receipt-crawler:dev-20260806-safe  → Debian GNU/Linux 12 (bookworm)
crawler-worker:dev-3role                        → Debian GNU/Linux 13 (trixie)
```

원본 `roles/receipt/Dockerfile:4,16` 은 `python:3.12.10-slim` 으로 **패치 버전까지 고정**했는데, 통합
`Dockerfile:8,21` 은 `python:3.12-slim` 이라 태그가 떠 있다. thread/delegate 는 원본도 이미 trixie 라
이번 diff 의 문제가 아니지만 **receipt 에게는 이번에 새로 생긴 변경**이고, glibc·OpenSSL 메이저가 같이
올라간다(그 여파로 `libatk1.0-0`→`libatk1.0-0t64`, `libcups2`→`libcups2t64` 처럼 t64 전환 패키지로
치환돼 설치됐다 — `dpkg -l` 확인). TLS 로 Vault·S3·국세청을 붙는 워커에서 OS 메이저 상향은
playwright/pydantic 상향과 같은 급의 사실인데, 04-changes 의 버전 델타 목록에는 없다.

덤으로, 태그가 안 고정돼 있으니 **다음 빌드가 또 다른 베이스로 나올 수 있다.** "원본 무수정 복제"라는
이 작업의 대원칙이 런타임 플랫폼에서는 지켜지지 않는다.

## 4. (MINOR) apt 18줄과 630MB 의 근거 문구가 사실과 반대다

`Dockerfile:25-26` 은 "playwright 1.55 는 headless 도 full chromium 으로 뜬다(1.49 부터 new headless 가
기본)" 라고 적고, 그것이 receipt 쪽 라이브러리 목록을 합쳐야 하는 "실질적 이유"라고 한다.
이미지에 들어 있는 드라이버는 정반대로 말한다:

```
playwright/driver/package/lib/server/chromium/chromium.js:318
    return options.headless ? "chromium-headless-shell" : "chromium";
```

컨테이너에서 도는 크롤러는 전부 headless 다(`roles/receipt/crawlers/kcs_receipts_crawler.py:29`,
`ksta_receipts_crawler.py:34-35` 는 ENVIRONMENT=dev 라 headless=True, thread/delegate 도 마찬가지).
따라서 실제로 쓰이는 건 `chromium_headless_shell-1187`(321MB)이고 `chromium-1187`(594MB)은
**한 번도 안 뜬다.** `headless=False` 호출부는 `roles/delegate/agents/gangnam/review/admin_playwright.py:10`,
`post_test.py:32`, `playwright_test.py:85` 뿐인데 전부 개발용 스크립트고 X 없는 컨테이너에서는 어차피 못 뜬다.

그러니 04-changes 의 "줄이려면 `--only-shell` 인데 그러면 1.49+ new headless 동작과 달라져 회귀 위험"도
뒤집혀 있다 — 원본 receipt 가 정확히 `--only-shell` 로 빌드했고(`roles/receipt/Dockerfile:77`),
세 역할 모두 headless 라 `--only-shell` 이면 오히려 원본과 같은 바이너리다. 594MB 를 지금 당장 빼라는
지적이 아니라(상향 검증이 목적이었던 건 이해한다), **판단 근거가 틀린 채로 남아 있으면 다음 사람이
그걸 믿고 결정한다**는 게 문제다. `libappindicator3-1`(→`libayatana-appindicator3-1` 로 치환 설치됨)과
`libu2f-udev` 는 크롬 데스크톱 .deb 잔재라 headless chromium 과 무관하다는 것도 같이 적어 두면 좋겠다.
apt 추가 자체는 "원본 3개 Dockerfile 의 합집합"이라는 다른 근거로 범위 안이다 — 근거를 바꿔 적으면 된다.

## 5. (MINOR) pydantic 2.12 로 **OpenAPI 계약이 실제로 바뀌었다**. 라우트 A/B 는 그걸 못 본다

`tools/dump_routes.py:19-26` 은 `(path, method)` 집합만 뽑는다. 그래서 "0 diff" 는 스키마를 보증하지 않는다.
통합 전 원본 이미지와 `crawler-worker:dev-3role` 에서 `app.openapi()` 를 떠서 붙여 봤다:

```
thread   : components.schemas.CheckKakaoIdData.properties.cookieDict
           + "additionalProperties": true      ← 이 한 줄만 다름
delegate : 동일한 한 줄만 다름
```

`cookieDict` 가 dict 타입이라 2.12 의 JSON Schema 생성이 달라진 것으로, 검증 동작 자체는 아니다.
다만 OpenAPI 로 클라이언트를 뽑는 쪽이 있으면 계약 diff 다.

그리고 이건 작성자에게 빚진 이야기인데 — **검증 동작 쪽은 내가 대신 돌려 봤고 깨끗했다.** 세 역할의
모든 BaseModel(thread `reputation_dto.py`·`routes/kafka_routes.py`, delegate 4개 모듈, receipt `models.py`)에
대해 4가지 입력형(정상/문자열 강제변환/미지의 키 추가/빈 dict)으로 `model_validate` → `model_dump(by_alias=True)`
결과와 ValidationError 의 `(loc,type)` 을 2.9.2 이미지와 2.12.0 이미지에서 각각 찍어 비교했다:
thread 104행 · delegate 112행 · receipt 8행 **전부 동일**. pydantic 상향의 실질 위험은 낮다는 뜻이다.
**문제는 이 근거가 04-changes 에 없었다는 것이다.** 마이너 3개를 건너뛴 직렬화 라이브러리인데
"import 성공"과 "라우트 집합 동일"만 근거로 AC 를 ✅ 처리했다.

## 6. (MINOR) AC5 왕복은 진짜지만, 크레딧이 과하다

브로커에서 확인한 사실은 작성자 주장과 맞다:

```
legalcare.test.3role.receipt.kcs        end=1
legalcare.test.3role.receipt.completed  end=1  → {"storeId":910101,...,"imageUrl":"https://fixture.invalid/legalcare/receipt.png"}
legalcare.test.3role.receipt.dlq        end=0
group crawler-worker-3role-verify-v1    kcs 커밋오프셋 1 (지금은 Empty — 컨테이너가 교체용 그룹으로 옮겨가서)
```

소비→검증→발행→커밋이 실제로 일어났다. 직전 라운드처럼 탐지력 0인 검사는 아니다. 다만 덮는 범위는
좁다. `ENABLE_EXTERNAL_ACTIONS=false` 라 `worker.py:67-73` 에서 바로 돌아나가므로 **`worker.py:79-83` 의
KCS/KSTA 토픽 분기는 실행되지 않았다**(test ksta 토픽은 지금도 0/0). 크롤러·S3·Vault 도 마찬가지고,
성공 분기 하나만 봤다 — DLQ/retry 는 가짜 프로듀서를 쓰는 유닛 테스트 근거뿐이다.
그리고 `worker.py:177`(성공 커밋)과 `worker.py:95`(DLQ 커밋)가 둘 다 커밋하므로 **커밋 오프셋만으로는
성공과 DLQ 를 구분할 수 없다** — completed=1/dlq=0 을 같이 봐야 성립하는 검사다. 그 두 값을 04-changes 가
같이 적어 둔 건 잘한 것이다.

## 7. (MINOR) 교체된 컨테이너는 아직 실메시지를 한 건도 안 먹었다 + 00-log 의 건수가 브로커와 안 맞는다

```
group legalcare-local-pg-receipt-fixture-v1 : Stable(1), member 172.28.0.21 (= crawler-worker-receipt), LAG 0
legalcare.dev.fixture.receipt.kcs  earliest=2 latest=2      ksta  earliest=1 latest=1
legalcare.dev.fixture.receipt.completed 2/2                 dlq   1/1        (retention.ms=86400000)
docker logs crawler-worker-receipt → 2줄 (entrypoint + "receipt worker starting ... group=...fixture-v1")
```

네 토픽 모두 earliest==latest, 즉 **보관된 레코드가 0건**이라 먹을 게 없다. "신규는 Stable(1) 로 정상
참여 확인"은 그룹 참여까지만이라는 뜻이다. 04-changes 가 이걸 "정상 참여"라고만 적어 두면 다음 사람이
"실동작 확인됐다"로 읽는다.

같이: 00-log 는 "개발계 누적 KCS 24건·KSTA 1건"이라고 적었는데 브로커의 fixture kcs 는 log-end-offset 이
**2**다. 그 24가 무엇을 센 숫자든 이 토픽은 아니다. 가정 A1(실사용량 미미)이 이 숫자에 기대고 있으니
정정하는 편이 좋다(결론 자체는 오히려 더 강해진다).

## 8. (MINOR) 승계한 그룹에 `KAFKA_AUTO_OFFSET_RESET=earliest` 는 다음 교체 때 리플레이 함정이다

`docker-compose.dev.yml:82` 의 `earliest` 가 cutover 에서 그대로 상속된다(cutover 는 그룹·토픽만 덮어씀).
지금은 무해하다 — 커밋 오프셋 == log-end 이고 보관 레코드가 0이라 리플레이할 게 없다. 문제는 조건이다.
교체 직전 이 그룹은 **3일째 `Empty`** 였고, Kafka 는 빈 그룹의 오프셋을 `offsets.retention.minutes`
(기본 7일) 후 지운다. 다음 롤백/재교체가 그 창을 넘겨서 일어나고 토픽에 레코드가 남아 있으면
**처음부터 다시 먹고 fixture completed 를 그만큼 재발행한다.** README·cutover 어디에도 이 얘기가 없다.

## 9. (MINOR) `boto3-stubs` 가 런타임 이미지에 들어간다

`requirements.txt:150`. 런타임 import 가 없는 타입 스텁인데 이미지에 실려 있고(`pip list` 확인),
`botocore-stubs 1.43.67` 까지 딸려 왔다. 게다가 **하향 조정에서 유일하게 빠진 receipt 핀**이라
스텁은 boto3 1.40.51 을 설명하는데 런타임은 1.34.34 다. 동작에 지장은 없지만 union 규칙과 어긋나고
allowlist 로 소스는 깐깐하게 거르면서 이건 그냥 들어간 셈이다.

## 10. (MINOR, 공개된 한계의 확장) 롤백은 "이미 고장 난 컨슈머"를 되살린다

되돌리기 자체는 확실하다 — 구 이미지 3개 모두 로컬에 남아 있고(`433a8a07a602`/`42b79a73a960`/`f8bf1c4519e8`),
컨테이너는 정지만 했고 마운트·포트 바인딩 0건이라 `docker start` 한 번이 맞다. 다만 되돌아가는 그 상태가
**그룹 멤버 0인데 docker 는 healthy 라고 말하던 상태**다. 1번과 겹쳐서, 구·신 어느 쪽으로 가든 receipt 의
컨슈머 생존을 알려주는 신호가 이 환경에 없다.
(참고로 구 receipt 는 `ExitCode 137`·정지에 약 30초가 걸렸는데, 원본은 python 이 PID 1 이라 SIGTERM 을
무시해서다. 통합본은 tini 밑이라 이 부분은 개선됐다.)

## 11. (MINOR) compose 서비스 이름이 공용 망에 별칭으로 같이 붙는다 — `thread`·`delegate`·`receipt`

별칭 규율은 신경 써서 짰는데(`docker-compose.cutover.yml:14-31`) 한 겹을 빠뜨렸다. compose 는 명시한
`aliases:` 말고 **서비스 이름도 별칭으로 단다.** 실제 상태:

```
crawler-worker-thread    legalcare-local_legalcare => [crawler-worker-thread thread crawler-web]
crawler-worker-delegate  legalcare-local_legalcare => [crawler-worker-delegate delegate ai-delegate-web]
crawler-worker-receipt   legalcare-local_legalcare => [crawler-worker-receipt receipt pg-receipt-worker]
```

즉 `thread`·`delegate`·`receipt` 라는 아주 흔한 이름 3개를 공용 개발망에 새로 점유했다.
지금은 충돌 없다 — 6개 이름 전부 IP 하나씩만 돌려주는 것까지 확인했다(`thread`→172.28.0.22,
`delegate`→.23, `receipt`→.21, `crawler-web`→.22, `ai-delegate-web`→.23, `pg-receipt-worker`→.21).
다만 `legalcare-local` 묶음에 같은 이름의 서비스가 하나라도 생기면 그때부터 조용히 DNS 가 갈린다.
검증용(`docker-compose.dev.yml:88-90`)이 "별칭 없이 붙는다"고 적힌 것도 정확히는 사실이 아니다 —
별칭 `receipt` 를 달고 붙는다. 서비스 이름을 `crawler-worker-receipt` 처럼 길게 두거나
문서에 이 사실을 적어 두는 정도면 충분하다.

---

## 범위 이탈(orphan) 점검 — 미매핑 변경 0건

| 변경 | R# | 판정 |
|---|---|---|
| `roles/receipt/` 17파일 | R1 | ✓ |
| `docker/entrypoint.sh` receipt 분기 | R2 | ✓ |
| `Dockerfile` receipt COPY · screenshots mkdir | R1·R2 | ✓ |
| `Dockerfile` apt 18줄 | R1(원본 3종 union) | ✓ 범위 안, 단 근거 문구가 틀림(#4) |
| `requirements.txt` union | R3 | ✓ (boto3-stubs 만 #9) |
| `docker-compose.dev.yml` receipt·격리 토픽 | R6 | ✓ |
| `docker-compose.dev.yml` thread/delegate healthcheck | AC4 | ✓ 경계선 — "healthy" 판정에 필요 |
| `docker-compose.dev.yml` 이미지 태그·`-p` 프로젝트명 | R6 | ✓ |
| `docker-compose.cutover.yml` receipt·env 승계 | R5·R7 | ✓ |
| `README.md` · `.env.example` | R7 | ✓ |

R4(`roles/PATCHES.md`)는 미발동이 맞다 — 사본 수정 0건을 sha256 로 확인했다.
AC6 도 성립한다: 공용 망 컨테이너 28개, 구 3개가 비운 자리를 신 3개가 채웠다.

## 정리

교체·롤백 설계와 env 승계는 꼼꼼하다. 되돌릴 수 있는 상태로 남긴 것도, 검증용/교체용 토픽·그룹을
나눈 것도 맞는 판단이다. 막는 건 1번 하나다 — **가짜 healthy 는 다음 사고를 그대로 반복시키고,
심지어 문서는 그게 고쳐졌다고 말한다.** 2·3번은 코드 수정 없이 04-changes 의 한계 절과 README 를
사실대로 고치면 닫힌다(안 켜 본 것을 안 켜 봤다고, OS 가 바뀐 것을 바뀌었다고). 4·5번은 문구·근거 정정,
6~11번은 기록 보강이면 된다. 개발계를 되돌릴 필요는 없다고 본다.

VERDICT: REQUEST_CHANGES
