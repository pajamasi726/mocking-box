# 라운드 1 대응 (writer=claude → 셀프 크로스리뷰 지적)

지적 3건 모두 타당했다. 특히 #1·#2 는 "검증했다"고 쓴 것이 실제로는 검증이 아니었던 건이라 그대로 인정한다.

| # | 지적 | 대응 | 근거 |
|---|---|---|---|
| 1 | AC4 브로커 검사 탐지력 0 (그룹명 오타 + 다른 브로커 조회) | **반영** — 실측하니 지적보다 더 나빴다. 그룹명이 `medi**a**lawyer`(코드 기본값)로 조회됐는데 실제 배포값은 env override 라 `medilawyer` 이고, **두 역할이 서로 다른 그룹**을 쓴다(thread=`...crawler-service-group`, delegate=`...crawling-service-group`, `crawler/.env:5`·`ai-delegate-crawler/.env:5`). 브로커도 `legalcare-local-kafka-1`(스택 내부)을 봤는데 dev 실브로커는 `kafka1`/`kafka2`. 올바른 조합으로 재검사 완료 | 아래 재검증 1 |
| 2 | AC3 delegate 라우트 11개는 검증된 적 없음 | **반영** — `--network none` + `ENABLE_EXTERNAL_ACTIONS=true` 오프라인 임포트 덤프를 검증 절차로 채택. **통합 전 실제 이미지**(`legalcare/ai-delegate-crawler:dev-20260806-reliable5`, `legalcare/crawler:dev-20260806-safe`)와 A/B 비교해 양쪽 diff 0 확인. `tools/dump_routes.py` 에 `sys.path.insert(0, os.getcwd())` 를 넣어 통합 이미지·원본 이미지 어디서든 같은 스크립트로 덤프 가능하게 함 | 아래 재검증 2 |
| 3 | Dockerfile 이 allowlist → denylist 로 통제 반전 | **반영** — 권고대로 되돌렸다. `Dockerfile:59-75` 가 원본 두 Dockerfile 의 COPY 목록을 그대로 열거한다. `.dockerignore` 는 "컨텍스트에 올리지 않을 것"(자격정보·캐시·로그)만 담당 | 아래 재검증 3 |
| 4 | 배포 env 산출물 부재 | **반영** — `docker-compose.dev.yml`(안전 env 3종을 파일에 고정) + `.env.example`(역할별 기본값 차이·group.id 주의 포함) 추가. 개발계 기동을 이 compose 로 재수행해 검증 | 아래 재검증 4 |
| 5 | psutil / APM 기본값 | **현행 유지** — 지적대로 psutil 유일 import 파일이 미참조 죽은 코드임을 확인(`crawler_selenium.py` 는 어느 라우트·kafka 경로에서도 임포트되지 않음). APM 기본값도 R6 계약대로 유지하고 안전 env 는 compose 가 실어 나른다 | — |
| 6 | bundle-c-diff.py 위치 명시 | **반영** — `model-worker/tools/bundle-c-diff.py`, 골든셋 `model-worker/goldenset/bundle-c/`. 1단계에서 하네스만 담은 껍데기 저장소를 먼저 만들고 5단계에서 roles 를 채우는 순서까지 명시 | `bundle-c-plan.md` §4.3 |

## MINOR 판단

- **fixture 빌드 컨텍스트에 `.dockerignore` 추가** → **반박**. `roles/*` 를 원본과 바이트 동일하게 유지하는 것이
  이 저장소의 핵심 불변식이다(원본에서 rsync 재동기화 후 체크섬만으로 재검증 가능하고, 내 대조 스크립트도
  "복사본 전용 파일 0"을 강제한다). 손으로 파일 하나를 넣으면 그 불변식이 깨진다. 대가는 컨텍스트 1.5MB뿐이라
  감수하고 README 에 사유를 남겼다.
- **group.id 경고 강화** → 반영. README 에 코드 기본값(`medialawyer`)과 배포 실값(`medilawyer`)이 다르고
  역할마다 그룹이 다르다는 표를 추가했다. 이번 라운드에서 실제로 물린 함정이라 표로 못박았다.
- **README 플래그** → 반영(위 표 + `.env.example`).
- **union 이중선언(`==`+`~=` 21건)** → **반박**. delegate 원본 requirements.txt 자체가 두 형태를 함께 선언하고
  있고 교집합이 원래 핀과 같다. R5 의 "임의 상향 금지"는 정리·재작성도 금지로 읽는 게 맞다고 판단해 원문 유지,
  사유를 README 에 기록.
- **`CRAWLER_PORT` 오버라이드** → 반영(제거). 지시서에 없는 기능이라 삭제하고 포트를 역할에 고정했다.

## 재검증 (전부 DELL 실측)

**1. Kafka (정정)** — 브로커 `kafka1`/`kafka2`, 실그룹명 기준
```
[dev-medilawyer-crawling-service-group]  ← delegate 역할 그룹
  aiokafka-0.10.0-b3b0a1c5-...  HOST=/172.19.0.1  #PARTITIONS=1     (기존 워크스테이션 컨테이너)
[dev-medilawyer-crawler-service-group]   ← thread 역할 그룹
  Error: Consumer group ... does not exist.
통합 컨테이너 IP: thread=192.168.80.3, delegate=192.168.80.2  → 멤버 목록에 없음
```
검사 탐지력 확인: 실재 그룹에서 멤버 1건을 실제로 집어냈다(이전 검사는 0건만 반환).

**2. 라우트 (오프라인 A/B, `--network none`, external ON)**
```
thread   : BEFORE(legalcare/crawler:dev-20260806-safe)              vs AFTER  → IDENTICAL (차집합 0)
delegate : BEFORE(legalcare/ai-delegate-crawler:dev-20260806-reliable5) vs AFTER → IDENTICAL (차집합 0)
delegate AFTER 21개 = FastAPI 기본 8 + / + /health + 업무 라우트 11
  (/check_id /crawling/kafka/produce /crawling/naver/cafe /get_naver_place_list /kakao_login_2
   /naver_login_captcha /reply(GET,POST) /reply_gpt_return /report /screenshot)
```

**3. allowlist 복귀 후 이미지 내용물**
```
이미지 roles/thread 68개 (저장소 72) / roles/delegate 87개 (저장소 108)
fixture_app.py · fixture_runtime.py · requirements.fixture.txt · tests · Dockerfile.fixture · README.md · .pylintrc → 전부 "없음(OK)"
```

**4. compose 산출물로 재기동** — thread/delegate 둘 다 `/health` 200, 안전 env 3종이 컨테이너에 실린 것 확인,
외부 라우트 4건 503, established TCP 0건.

**5. 회귀** — delegate 테스트 **38 passed**, fixture test stage **Ran 5 tests / OK**,
uid=10001 · PID1=tini · chromium-1105 · `CRAWLER_ROLE` 오류 시 exit=64 모두 유지.

STATUS: REVISED
