# 작업지시서: 파이썬 워커 5 → 배포단위 3 통합 (묶음A 실행 · 묶음C 계획검증)

## 배경 및 목표
> 3개로 줄일수있구나 그러면 셀프 리뷰모드로 분리해서 합쳐봐

묶음A(crawler + ai-delegate-crawler)를 **저장소 1개·이미지 1개**로 통합하고, 묶음B(pg-receipt-crawler)는 단독 유지, 묶음C(ai-model-serve + embedding-service)는 이번 라운드에 통합 계획·검증 절차만 만든다.

## 범위
- 포함: `crawler-worker/` 신설(역할 2개 복사 배치) · union requirements · 단일 Dockerfile + 역할 분기 entrypoint · 개발계(DELL) 병렬 기동 검증 · 묶음C 계획서
- 제외: 운영/HP 배포, git 이력 병합(단순 파일 복사로 충분), 묶음B 손대기(playwright 1.55/pydantic 2.12 충돌 — 단독이 맞음), 묶음C 실제 코드 통합, 중복 `agents/` dedup, 파이썬→자바 재작성, push

## 기능 요구사항
R1. 통합 형태 = **단일 이미지 · 역할별 프로세스**(`CRAWLER_ROLE=thread|delegate` 로 컨테이너 2개). 한 프로세스 병합 금지 — 근거: 두 서비스가 같은 이름의 top-level 패키지(`config.py`·`kafka/`·`routes/`·`agents/`)를 서로 다른 내용으로 갖고, `/check_id`·`/kakao_login_2`·`/naver_login_captcha` 3개 경로가 정면 충돌하며, Kafka consumer 가 서로 다른 토픽을 구독한다(소비 격리·스케일 독립성 상실).
R2. 새 위치 `/Users/steve/steve/legal-care/crawler-worker/` 에 `roles/thread/`(=crawler 원본), `roles/delegate/`(=ai-delegate-crawler 원본)를 파일 복사한다. 원본 5개 저장소는 무변경.
R3. 역할 소스는 내용 수정 금지(경로 이동만). 실행은 역할 디렉토리를 sys.path 루트로 두고(`WORKDIR /app/roles/<role>`) `uvicorn main:app` — import 문 무수정. 포트 thread=42030, delegate=42010 유지.
R4. Dockerfile 1개로 두 역할 소스 + union requirements + playwright chromium(1회 설치)을 담고, entrypoint 가 `CRAWLER_ROLE` 로 분기한다. uid/gid 10001·tini·`PLAYWRIGHT_BROWSERS_PATH` 유지.
R5. requirements 는 union 으로 만들고 **동일 패키지 상이 버전이 1건이라도 나오면 통합을 중단하고 목록을 보고**한다(임의 상향 금지).
R6. 불변 계약: 역할별 Kafka group.id·구독 토픽, FastAPI 라우트 경로 집합, `ENABLE_KAFKA_CONSUMER`/`ENABLE_EXTERNAL_ACTIONS`/`ENABLED_CRAWLING_CHANNELS` 의미와 **역할별 기본값**, `validate_runtime_role` 가드, `require_external_actions` 503 동작.
R7. safe/fixture 자산(`Dockerfile.fixture`·`fixture_app.py`·`fixture_runtime.py`·`requirements.fixture.txt`·`tests/`)은 delegate 역할 하위로 그대로 옮겨 **별도 이미지로 계속 빌드 가능**해야 한다. 통합 이미지에 섞지 않는다.
R8. 개발계 검증은 소비 정지 상태로만: 두 역할 모두 `ENABLE_KAFKA_CONSUMER=false`·`ENABLE_EXTERNAL_ACTIONS=false`, 호스트 포트는 임시(42130/42110)로 매핑해 기존 컨테이너를 중단·교체하지 않는다.
R9. [묶음C] 이번엔 계획·검증만 — `{TASK_DIR}/bundle-c-plan.md` 만 만들고 코드는 손대지 않는다. 근거: embedding-service 는 컨테이너 기동 시 학습을 실행하고(`Dockerfile` CMD) Qwen2.5-3B LoRA 를 로드하는 HP(prod) 상시 GPU 워커라, py3.11↔3.12·torch 2.6.0↔2.5.1 통일은 출력 동등성 실측이 선행돼야 한다.

## 수용 기준
AC1. 통합 Dockerfile 로 이미지 1개 빌드 성공(두 역할 소스·chromium 포함).
AC2. 같은 이미지에서 `CRAWLER_ROLE=thread`·`delegate` 컨테이너 2개가 동시 기동되고 각각 `GET /health` 200, JSON 키 집합이 통합 전과 동일하다.
AC3. 라우트 동등성: 역할별 (경로, 메서드) 집합 덤프가 통합 전 원본과 차집합 0. thread 는 `/`,`/health`,`/check_id`,`/kakao_login_2`,`/naver_login_captcha`,`/naver_cafe_passive_crawl` 를 포함한다.
AC4. 외부 수집 미가동 실증: 외부 동작 라우트 호출 시 503, 컨테이너 로그에 naver/kakao/modoodoc/google 요청 0건, Kafka 브로커에 새 consumer group 참여 0건.
AC5. union requirements 에 동일 패키지 상이 버전 0건이고, fastapi·uvicorn·playwright·pydantic·aiokafka 버전이 통합 전 두 서비스 값과 같다.
AC6. 원본 5개 저장소 `git status` 변경 0 · push 0. 신규 파일은 `crawler-worker/` 와 TASK_DIR 안에만 존재한다.
AC7. `bundle-c-plan.md` 에 (a) 통일 대상 버전 표(python·torch·transformers) (b) 출력 동등성 검증 절차(동일 입력 20건 이상 + 통과 임계값 숫자 명시) (c) 실행 순서가 있고, 묶음C 소스 변경 0.

## 기술 제약
- 양쪽 모두 repo-root 절대 import(`from config import ...`, `import kafka.kafka_consumer`) 사용 → 역할 디렉토리가 sys.path 루트여야 동작하며, top-level `kafka` 패키지는 PyPI 패키지명과도 겹친다(crawler/main.py:18-22, ai-delegate-crawler/main.py:9-18).
- 플래그 기본값이 역할마다 다르다: crawler `config.py:47-48` = true(운영 보존), ai-delegate `app/config.py:26-28` = false(fail-safe). 통합 후에도 그대로 두고 개발계는 env 로 명시한다.
- Kafka: thread = `{env}.medilawyer.event.system.thread.crawling.request`(crawler/kafka/kafka_consumer.py:40), delegate = `{env}.medilawyer.event.system.post.crawling.request` + `.dlq`(ai-delegate-crawler/config.py:132-135). group.id 기본값이 양쪽 동일(`{env}-medialawyer-crawling-service-group`) — 현행 유지, 변경 금지.
- 개발계 = DELL(192.168.1.55) `legalcare-local` 스택. 기존 컨테이너 유지한 채 새 포트로 병렬 기동, HP/운영 미접촉.

## 가정
A1. "3개"는 배포 단위(저장소·Dockerfile·이미지·CI) 기준이며, 묶음A 의 실행 컨테이너는 소비 격리를 위해 2개로 유지한다.
A2. 두 크롤러의 중복 `agents/` 코드 통합(dedup)은 후속 과제다. 이번 라운드는 동작 보존이 우선이다.

## 리스크
RISK: MEDIUM — 복사 기반·원본 무변경·개발계 한정이라 롤백이 디렉토리 삭제로 끝나지만, 플래그가 잘못 켜지면 외부 사이트 수집·계정 잠금으로 번진다. 완화: R8/AC4 로 Kafka 소비와 외부 동작을 이중 차단하고, 역할별 기본값 변경을 금지한다.

## 관련 파일
- `/Users/steve/steve/legal-care/crawler/` — main.py(42030, lifespan+consumer), config.py(플래그·토픽·자격정보), runtime_guards.py(503 가드), kafka/{kafka_consumer,consumer}.py, routes/(6), agents/, Dockerfile, requirements.txt(84), .env.workstation
- `/Users/steve/steve/legal-care/ai-delegate-crawler/` — main.py(42010, 역할검증+조건부 라우터), app/{config,runtime_validation,kafka_manager,routes,eureka_client}.py, config.py(토픽·DLQ), routes/(8), agents/, Dockerfile, Dockerfile.fixture+fixture_*.py+requirements.fixture.txt+tests/, requirements.txt(117), .env.workstation
- `/Users/steve/steve/legal-care/pg-receipt-crawler/` — 묶음B, 이번 변경 대상 아님(충돌 근거만 참조)
- `/Users/steve/steve/legal-care/ai-model-serve/`, `/Users/steve/steve/legal-care/embedding-service/` — 묶음C, Dockerfile·requirements.txt 만 계획서 근거로 참조(코드 무변경)
- `{TASK_DIR}/00-log.md` — 의존성 선실측 표(재조사 금지)

## 의도 확인
사용자 요청 = "합칠 수 있는 것끼리 합쳐 5개를 3개로 줄이고, 셀프 리뷰로 검증하라". 이 지시서는 가장 안전한 묶음A만 실제로 합치되 소스 무수정 복사 + 역할 분리 프로세스로 동작 계약(토픽·엔드포인트·안전모드)을 그대로 보존하고, 위험한 묶음C는 버전 통일 검증 계획으로 분리했다. 개발계에서 빌드·기동·라우트 동등성까지만 확인하고 외부 수집은 켜지 않는다.

## OPEN QUESTIONS
없음
