# 변경 요약: 파이썬 워커 묶음A 통합 (crawler + ai-delegate-crawler → crawler-worker)

> 라운드 1 지적 반영본. 지적별 대응은 `round-1-claude.md` 참고. 아래 검증 결과는 전부 재실행한 값이다.

## 변경 파일
| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `legal-care/crawler-worker/Dockerfile` | 단일 이미지. **원본 두 Dockerfile 의 allowlist COPY 목록을 그대로 열거**(열거 안 된 파일은 이미지에 안 들어감). uid/gid 10001·tini·chromium 1회·PLAYWRIGHT_BROWSERS_PATH 유지 | R1,R4,R7 |
| `legal-care/crawler-worker/docker/entrypoint.sh` | `CRAWLER_ROLE` 분기 → `cd roles/<role>` → uvicorn/gunicorn. 포트는 역할 고정(42030/42010), prod 옵션 역할별 원본 유지 | R1,R3,R4 |
| `legal-care/crawler-worker/docker-compose.dev.yml` | 개발계 배포 산출물. 안전 env(소비/외부/APM OFF)를 파일에 고정해 빠뜨릴 수 없게 함 | R6,R8 |
| `legal-care/crawler-worker/.env.example` | 역할별 env 전체 목록 + 기본값 차이 + group.id 함정 경고 | R6 |
| `legal-care/crawler-worker/requirements.txt` | 두 역할 requirements union (128줄) | R5 |
| `legal-care/crawler-worker/.dockerignore` | 빌드 컨텍스트 제외(자격정보·캐시·로그)만 담당 | R4 |
| `legal-care/crawler-worker/README.md` | 역할·실행법·플래그 기본값·group.id 실값 표·설계 판단 근거 | R1,R6 |
| `legal-care/crawler-worker/roles/thread/**` (72개) | crawler 원본 무수정 복사 | R2,R3 |
| `legal-care/crawler-worker/roles/delegate/**` (108개) | ai-delegate-crawler 원본 무수정 복사(fixture·tests 포함) | R2,R3,R7 |
| `legal-care/crawler-worker/tools/dump_routes.py`, `route_inventory.py` | 라우트 집합 덤프(런타임/정적) — AC3 근거 생성용 | AC3 |
| `{TASK_DIR}/bundle-c-plan.md` | 묶음C 통합 계획서 (코드 변경 0) | R9 |

## 테스트 결과
실행 위치: **DELL(192.168.1.55, x86_64, docker 29.1.4)** — 실제 실행 결과다.
- `docker build` → **성공, 2.51GB**. allowlist 적용 후 이미지 내 `roles/thread` 68개 / `roles/delegate` 87개(저장소는 72/108). fixture·tests·문서·lint 설정은 이미지에 **없음** 확인.
- `docker compose -f docker-compose.dev.yml up -d` → 두 역할 동시 running, 안전 env 3종이 컨테이너에 실린 것 확인.
- delegate 테스트 스위트(통합 이미지, tests 마운트) → **38 passed, 5 warnings**
- fixture 별도 이미지 빌드 + `--target test` 재실행 → **Ran 5 tests / OK**
- thread 역할은 원본에 테스트 스위트가 없어 기동·라우트·503 실증으로 대체.
- 초기 실패 2건(정직 기록): ① rsync `--exclude 'aws/'` 가 `agents/aws/` 까지 지워 thread 가 `ModuleNotFoundError: agents.aws` 로 기동 실패 → 앵커드 패턴 재복사 + 양방향 파일집합 대조 도입. ② 1라운드 AC3/AC4 검증이 실제로는 탐지력이 없었음 → 아래처럼 재설계.

## AC 자가 점검
- AC1 ✅ 이미지 1개 빌드 성공(두 역할 소스+chromium-1105 포함).
- AC2 ✅ compose 로 thread/delegate 동시 기동, `/health` 둘 다 200. 키·값이 통합 전 컨테이너와 동일 — thread `{status,kafkaConsumerEnabled,externalActionsEnabled}`, delegate `{status,kafkaConsumerEnabled,eurekaEnabled,externalActionsEnabled,enabledCrawlingChannels}`.
- AC3 ✅ **`--network none` + `ENABLE_EXTERNAL_ACTIONS=true` 오프라인 임포트 덤프**를 **통합 전 실제 이미지**(`legalcare/crawler:dev-20260806-safe`, `legalcare/ai-delegate-crawler:dev-20260806-reliable5`)와 A/B 비교 → **양쪽 diff 0**. delegate 21개(기본 8 + `/` + `/health` + 업무 라우트 11) 전부 확인됐다. 1라운드의 "런타임 덤프 동일" 주장은 R8 환경에서 delegate 라우터가 include 되지 않아 성립하지 않았고, 그 서술을 이 방식으로 교체했다. thread 는 `/`,`/health`,`/check_id`,`/kakao_login_2`,`/naver_login_captcha`,`/naver_cafe_passive_crawl` 포함.
- AC4 ✅ 외부 라우트 503(thread `require_external_actions`, delegate catch-all), established TCP 0건, 로그 내 naver/kakao/modoodoc/google 0건. **Kafka 재검사(정정)**: 브로커 `kafka1`/`kafka2`(dev 실브로커), 그룹 `dev-medilawyer-crawling-service-group`(delegate)·`dev-medilawyer-crawler-service-group`(thread). 전자는 멤버 1건이나 HOST `/172.19.0.1` 로 기존 컨테이너이고, 통합 컨테이너(192.168.80.2/.3)는 없음. 후자는 그룹 자체가 미존재. 1라운드 검사는 브로커·그룹명이 모두 틀려 탐지력이 0이었다.
- AC5 ✅ union 동일 패키지 상이 `==` 버전 **0건**. 실측 fastapi 0.110.0 / uvicorn 0.28.0 / playwright 1.42.0 / pydantic 2.9.2 / aiokafka 0.10.0 — 통합 전과 동일. 미고정(google-genai 1.2.0, pillow 12.3.0, aio-pika 9.4.3)도 baseline 일치.
- AC6 ✅ 원본 5개 저장소 세션 시작 이후 수정 파일 **0건**(mtime), push 0. `roles/*` 는 원본과 sha256 동일·복사본 전용 파일 0.
- AC7 ✅ `bundle-c-plan.md` 에 (a) 버전표 (b) 골든셋 200/150건 + 임계값 숫자 (c) 6단계 실행 순서 수록. 하네스 위치 `model-worker/tools/bundle-c-diff.py` 명시. 묶음C 소스 변경 0.

## 알려진 한계 / 리뷰어에게
1. **psutil 7.2.2 → 6.1.1 은 그대로 둔다(종결).** 유일한 import 지점 `crawler_selenium.py` 를 양 역할 트리 전체에서 아무도 임포트하지 않는 죽은 코드임을 직접 확인했다(참조 0건). 영향 없음.
2. **thread 역할 `ELASTIC_APM_ENABLED` 기본값 `true` + `config.py:34` 하드코딩 주소는 R6 계약대로 유지.** 대신 안전 env 를 `docker-compose.dev.yml`·`.env.example` 이 실어 나르게 했다(1라운드에서 이걸 빠뜨려 실제로 APM 연결이 맺혔던 건이 재발 방지 대상이다).
3. **group.id 는 코드 기본값과 배포 실값이 다르고 역할마다 다르다.** 코드 `medi**a**lawyer`, 배포 `medilawyer`, thread=`crawler-service-group` / delegate=`crawling-service-group`. 통합에서 config·env 모두 무변경이라 현행 유지되지만 README 에 표로 못박았다.
4. 원본에서 안 가져온 것: `.env*`, `.github/`(기존 per-repo 배포 워크플로), `nb/`, `Pipfile*`, `aws/`, `local/lambda_server.py`, `task-definition.json`, 각 repo `Dockerfile`/`start.sh`. **CI/배포 파이프라인 통합은 범위 밖**이라 신규 저장소에 워크플로가 없다.
5. 검증용 컨테이너 2개(42130/42110)를 DELL 에 띄워둔 상태다(기존 컨테이너 무접촉). `docker compose -f docker-compose.dev.yml down` 으로 정리.
6. `agents/` 중복 dedup 은 A2 대로 후속 과제. 이미지 2.51GB 의 주원인이다.

STATUS: REVISED
