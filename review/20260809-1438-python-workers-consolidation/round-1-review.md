# Round 1 교차 리뷰 — 파이썬 워커 묶음A 통합 (crawler-worker)

수정 권한 없음(review). 코드는 한 줄도 건드리지 않았고, 대신 **작성자 보고를 믿지 않고 DELL 에서 전부 재실행**했다.
결론부터: **코드는 거의 흠잡을 데가 없다.** 파일 무손실·플래그 기본값·Kafka 계약·라우트 동등성·테스트, 내가 다시 돌린 것 전부 통과했고
작성자가 스스로 flag 한 psutil 건은 실제로는 문제가 아니다(아래 §해소). 다만 **04-changes.md 에 적힌 근거 두 개가 실제로는 그 사실을 증명하지 못한다.**
특히 AC4(브로커 소비 0건)는 존재할 수 없는 그룹명을 조회한 결과라 탐지력이 0이다. 결론이 맞다고 근거까지 맞는 건 아니라서 REQUEST_CHANGES 를 건다.

## 내가 직접 재실행한 것 (작성자 보고와 별개)

| 항목 | 방법 | 결과 |
|---|---|---|
| 소스 무손실(양방향) | `comm -23`/`comm -13` 원본↔복사본 파일집합 | thread 72/72, delegate 108/108. **복사본에만 있는 파일 0** |
| 내용 무수정 | 복사된 180개 전부 sha256 대조 | **전건 일치** (`agents/aws/` 포함) |
| 배포물 동일성 | DELL `~/crawler-worker-build` ↔ Mac 187파일 per-file 해시 | **IDENTICAL** — 검증된 게 리뷰 대상 그 물건이 맞다 |
| /health (AC2) | baseline 이미지 vs 통합 이미지, 동일 env, `--network none` | 두 역할 모두 JSON **바이트 동일** |
| 라우트 (AC3) | baseline vs 통합, `EXTERNAL_ACTIONS=true`, `--network none` | thread 14/14, delegate 22/22 **diff 0** |
| Kafka 계약 | group.id·topic·DLQ 상수 offline 출력 대조 | baseline 과 **완전 일치**, thread 는 `thread.crawling.request` 단독 구독(`kafka/kafka_consumer.py:39-40`) |
| 플래그 기본값 | env 완전 미주입 상태로 baseline vs 통합 | 양쪽 `APM=True/3.34.122.48:8201, KAFKA=True, EXT=True, LAMBDA=True` — **원본 그대로** |
| 503 (AC4) | 7개 외부 라우트 POST | 전부 503 (thread 4 = `runtime_guards.py:6`, delegate 3 = catch-all `main.py:144`) |
| 소비 0건 | 두 컨테이너 `/proc/net/tcp*` ESTABLISHED | **0 / 0** |
| entrypoint fail-fast | `CRAWLER_ROLE=bogus` / 미지정 | 둘 다 **exit 64**, 조용한 한쪽 실행 없음 |
| PID1·비루트 | `ps -o pid,user,args -p 1`, `id` | `tini` PID1, uid=gid=10001 |
| R7 fixture | `--target test` 재빌드 후 **캐시 무시하고 실제 실행** | 5 passed |
| delegate 스위트 | 통합 이미지 안 pytest(tests 마운트) | **38 passed** |
| R7 격리 | 통합 이미지 내 fixture 자산 존재 여부 | `fixture_app.py`·`fixture_runtime.py`·`requirements.fixture.txt`·`tests` **전부 없음** |
| 시크릿 | 이미지 내 `.env*`/`env.*`/`*.pem` | **0건** |
| AC6 | 원본 5개 저장소 `-newermt "2026-08-09 14:30"` | **변경 0건** |
| orphan | `mocking-box git status` | `review/` `workhistory/` 만. workhistory 는 00:18 생성 = 이번 작업 무관 |

## 지적

**1. [MAJOR] AC4 의 브로커 근거가 존재할 수 없는 그룹명을 조회했다 — 탐지력 0**
`04-changes.md:29` 이 "브로커에 `dev-medialawyer-crawling-service-group` 자체가 미존재(신규 참여 0)"라고 썼다.
그런데 실제 dev 브로커(`kafka1`)에 있는 그룹은 **`dev-medilawyer-crawling-service-group`** 이다 — `media`가 아니라 `medi`.
코드 기본값(`roles/delegate/config.py:131`)이 `medialawyer` 인 건 맞지만 운영은 env 로 `medilawyer` 를 주입해서 뜬다.
게다가 조회한 브로커도 `legalcare-local-kafka-1`(33개 그룹, 크롤링 그룹 없음)로 보이는데 실제 그룹은 `kafka1`에 있다.
즉 **틀린 이름을 틀린 브로커에서 찾고 "없다"고 결론**낸 것이라, 통합 컨테이너가 진짜로 붙었어도 이 검사는 통과했을 것이다.
내가 `kafka1`에서 확인한 실제 상태는 멤버 1개(`aiokafka-0.10.0`, host `/172.19.0.1`)로 **기존 컨슈머 하나뿐**이고,
두 신규 컨테이너의 ESTABLISHED 소켓이 0이라 결론(신규 참여 0)은 맞다. 근거만 갈아끼우면 된다.
검사를 이걸로 바꿔라: `docker exec kafka1 kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group dev-medilawyer-crawling-service-group --members`.

**2. [MAJOR] AC3 "통합 전 컨테이너 덤프와 문자열 동일"은 delegate 에서 성립할 수 없다**
delegate 는 `roles/delegate/main.py:128-153` 에서 `ENABLE_EXTERNAL_ACTIONS` 가 false 면 라우터를 아예 include 하지 않는다.
그래서 R8 이 강제한 안전 env 로 뜬 `crawler-worker-delegate` 의 런타임 덤프는 `/` + `/health` **딱 2개**다(내가 openapi 로 확인).
반면 비교 대상이었을 pre-merge 컨테이너 `ai-delegate-crawler`(5일째 가동)는 12개이고 `/health` 조차 없다 — 세대가 다른 이미지다.
어느 쪽으로 붙여도 "문자열 동일"이 나올 수 없다. 결과적으로 **delegate 의 11개 업무 라우트는 런타임으로 검증된 적이 없다.**
다만 이건 근거 문제지 결함이 아니다. 내가 `--network none` 으로 `ENABLE_EXTERNAL_ACTIONS=true` 임포트 덤프를 떠서
`legalcare/ai-delegate-crawler:dev-20260806-reliable5` 와 대조한 결과 **22/22 diff 0**(thread 도 14/14 diff 0)이었다.
덤으로 이게 "union deps 로 전체 라우트 임포트가 실제로 성공한다"까지 증명하는데, 안전모드 기동만으로는 그게 안 잡힌다.
재현: `docker run --rm --network none -e CRAWLER_ROLE=delegate -e ENABLE_EXTERNAL_ACTIONS=true -e ENABLE_KAFKA_CONSUMER=false -e ENABLE_EUREKA=false -e ELASTIC_APM_ENABLED=false <img> python -c 'import main; ...'`

**3. [MAJOR] Dockerfile 이 원본의 allowlist COPY 를 denylist 로 뒤집었다**
원본 둘 다 `crawler/Dockerfile:51-55`, `ai-delegate-crawler/Dockerfile:53-58` 에서 **"실행에 필요한 소스만 복사한다. 배포 정의와 로컬 파일은 이미지에 넣지 않는다"**
라고 주석까지 달고 파일을 하나씩 열거했다. 통합본은 `Dockerfile:60-61` 에서 `COPY roles/thread ./roles/thread` 로 통째로 넣고
`.dockerignore` 로 빼는 방식이다. 지금 결과물은 깨끗하다(내가 이미지 안에서 확인: delegate 는 원본 COPY 목록과 동일한 10개 항목만 존재, `.env*` 0건).
하지만 **원본이 의도적으로 건 통제가 방향만 반대로 뒤집혔고 04-changes 에 언급도 없다.** 앞으로 `roles/` 밑에 뭘 하나 떨어뜨리면 자동으로 이미지에 들어간다.
`.dockerignore:20` 이 `**/task-definition.json` 만 막고 원본 `crawler/aws/taskdef.json` 이름은 안 막는 것처럼, 빠뜨리기도 쉽다.
역할별 명시 COPY 로 되돌리길 권한다(단일 Dockerfile 에서도 `COPY roles/thread/main.py roles/thread/config.py ... ./roles/thread/` 로 충분히 된다).

**4. [MINOR] `roles/delegate/.dockerignore` 가 복사되지 않아 fixture 빌드 컨텍스트에 ignore 규칙이 없다**
원본 `ai-delegate-crawler/.dockerignore`(17줄)는 있는데 복사본엔 없다. fixture 빌드 컨텍스트는 `roles/delegate` 라
(README 도 그렇게 안내) 루트 `.dockerignore` 가 적용되지 않고, 108개 파일 전체가 컨텍스트로 넘어간다.
`Dockerfile.fixture:11,15,21` 이 4개 경로만 COPY 하므로 **이미지 내용에는 영향 없다**(내가 재빌드해 확인). 컨텍스트 전송만 낭비다.

**5. [MINOR] 통합으로 "역할 2개가 같은 group.id 로 다른 토픽 구독"이 훨씬 쉬워졌는데 경고가 약하다**
`README.md` 의 group.id 경고는 "바꾸면 오프셋 초기화"만 말한다. 실제 위험은 반대쪽이다 —
두 역할 기본 group.id 가 같고(`roles/thread/config.py`, `roles/delegate/config.py:131`) 구독 토픽은 다르다.
저장소·이미지가 하나가 되면 compose `env_file` 이나 taskdef base 를 **한 벌로 공유**하기 쉬워지고, 거기에 `ENABLE_KAFKA_CONSUMER=true` 가 들어가는 순간
같은 그룹에 이종 구독 멤버 2개가 생겨 리밸런스가 서로를 물고 늘어진다. R6 이 group.id 변경을 금지하니 코드는 그대로 두되,
**"두 역할은 절대 같은 env 블록을 공유하지 말 것"** 을 README 에 한 줄 박아라.

**6. [MINOR] 통합 저장소에 env 를 실어 나르는 배포 산출물이 하나도 없다**
`04-changes.md:37` 이 밝힌 대로 `.env*`·`.github/`·`task-definition.json`·`aws/` 를 안 가져왔다. 이미지 기본값은 원본과 100% 동일해서
(내가 env 미주입 대조로 확인: `KAFKA=True EXT=True LAMBDA=True APM=True→3.34.122.48:8201`) **회귀는 아니다.**
문제는 원본에선 그 위험한 기본값 옆에 항상 env 를 채워주는 워크플로/taskdef/`.env.workstation` 이 같이 있었는데, 통합본엔 README 스니펫뿐이라는 것이다.
누가 `docker run -e CRAWLER_ROLE=thread` 만 하면 그대로 외부 수집 + Kafka 소비 + APM 실연결이다.
CI 통합이 범위 밖인 건 동의하지만, 최소한 `docker-compose.dev.yml` 같은 **안전 env 를 코드로 고정한 파일 1개**는 이 저장소에 있어야 한다.

**7. [MINOR] README 플래그 표에 `ENABLE_LAMBDA_PROXY` 가 빠졌다**
`roles/thread/config.py:73` 기본값 `true` 인 외부 동작 플래그인데, README 실행 예시에는 `-e ENABLE_LAMBDA_PROXY=false` 가 들어있고 표에는 없다.
표만 보고 env 를 구성하면 놓친다. 한 줄 추가.

**8. [MINOR] union 에 같은 패키지 이중 선언 21건**
`requirements.txt:70` `requests==2.31.0` 과 `:98` `requests~=2.31.0` 처럼 thread 의 `==X` 와 delegate 의 `~=X` 가 나란히 남았다(총 21쌍).
전부 base 버전이 같아 교집합이 핀 버전으로 떨어지고 실제 설치본도 baseline 과 동일하다(이미지 실측 fastapi 0.110.0/uvicorn 0.28.0/pydantic 2.9.2/aiokafka 0.10.0/playwright 1.42.0).
AC5 는 성립한다. 다만 한쪽만 bump 하면 파일이 즉시 해결 불가가 되니, 주석으로라도 "쌍으로 고쳐라"를 남겨라.
누락·추가 검사도 내가 돌렸다 — 원본 대비 **빠진 패키지 0, 없던 패키지 0**.

**9. [MINOR] `CRAWLER_PORT` 는 어느 R# 에도 없는 신규 노브**
`docker/entrypoint.sh:26`. 기본값이 역할별 원본 포트라 R3 는 만족하지만, 지시서에 없는 확장이다. 남길 거면 R# 매핑을 붙여라.

## 작성자가 판단 요청한 2건 — 둘 다 현행 유지가 맞다

**psutil 7.2.2 → 6.1.1: 문제 없음. 이 항목은 닫아도 된다.**
근거가 작성자 설명보다 강하다. ① thread 에서 psutil 을 끌어오던 건 `crawler/requirements.txt:84` 의 `soynlp>=0.0.493`(psutil 하한만 요구)이고,
② thread 안에서 `import psutil` 하는 유일한 파일 `roles/thread/agents/naver/cafe/services/crawler_selenium.py:13` 은
**트리 전체에서 아무도 참조하지 않는 죽은 코드**다(`grep -rn crawler_selenium roles/thread` → 자기 자신 외 0건).
쓰는 API(`process_iter`/`NoSuchProcess`/`AccessDenied`)가 6/7 동일한 것과 별개로, thread 런타임에 도달할 경로 자체가 없다.
이미지 실측 psutil 6.1.1 상태에서 두 역할 전체 라우트 임포트·38 테스트 모두 통과했다. R5 위반도 아니다(원본에 상이 버전이 아니라 미선언이었으니).

**thread APM 기본값 true + 주소 하드코딩: 그대로 둔 판단이 맞다.**
R6 이 "역할별 기본값 보존"을 계약으로 못박았고, baseline 이미지도 env 미주입 시 동일하게 `3.34.122.48:8201` 로 붙는다(내가 대조 확인).
여기서 기본값을 false 로 바꾸면 운영 배포가 env 를 안 주는 순간 APM 이 조용히 꺼져 관측이 죽는다 — 그게 더 나쁜 사고다.
다만 "검증 중 실제 연결이 맺혔다"는 사실이 말해주는 건 **기본값이 아니라 §6(배포 산출물 부재)** 이다. 그쪽으로 고쳐라.

## bundle-c-plan.md — 실행 가능하다

계획서로서 드물게 좋다. `4.2` 임계값이 지표별로 숫자까지 박혀 있고(라벨 100%, 회귀 max abs ≤1e-4, 코사인 ≥0.9999, 생성 완전일치 ≥95% + 불일치 시 chrF ≥0.98),
`4.2` 말미의 "CPU↔CPU 기준, GPU 비교는 동일 GPU·동일 dtype 에서만" 단서가 부동소수 비교의 가장 흔한 자기기만을 미리 막는다.
특히 `3.B1`(torch 2.6 의 `torch.load` `weights_only` 기본값 변화로 `ai-model-serve/dl/model_utils.py:50,72` 가 확정적으로 깨짐)을 **계획 단계에서 이미 확정된 실패**로 규정하고,
`5.2` 에서 그 코드 수정을 torch 2.5.1 에서 먼저 적용해 **코드 변경과 버전 변경을 분리**하도록 순서를 잡은 게 이 문서의 핵심 가치다.
`5.6` 롤백 태그 사전 pull, `B5` HP 무접촉도 적절하다. AC7 충족. 묶음C 소스 변경 0건도 확인했다.
보완할 건 하나뿐: `4.3` 의 `tools/bundle-c-diff.py` 가 **어느 저장소에 놓일지**가 안 적혀 있다. 묶음C 소스 무변경 원칙과 부딪히지 않게 위치를 명시해라.

## 정리

R1~R9·AC1~AC7 의 **실질은 전부 충족**한다. 내가 처음부터 다시 돌려서 확인했고, 원본 5개 저장소도 무변경이며 R# 밖 고아 변경도 없다.
막아야 할 건 코드가 아니라 기록이다. 지적 1·2 는 "결론은 맞는데 근거가 그 결론을 못 만든다"는 문제고,
이 기록을 믿고 다음 사람이 배포하면 같은 검사를 다시 돌려도 사고를 못 잡는다. 3 은 원본이 명시적으로 걸어둔 통제가 소리 없이 뒤집힌 건이다.
1·2 의 근거 교체(위에 명령까지 적어뒀다), 3 의 COPY 복원 또는 최소한 04-changes 에 트레이드오프 명시, 6 의 안전 env 파일 1개 — 이 셋만 처리하면 승인한다.

VERDICT: REQUEST_CHANGES
