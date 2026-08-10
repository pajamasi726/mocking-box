# 작업지시서: 크롤러 워커 3역할 통합 (playwright 1.55 상향 + 개발계 신규 기동)
## 배경 및 목표
> /team 자체 리뷰로 pg-receipt-crawler	그대로 둠 — 브라우저 라이브러리 버전이 A와 충돌 (1.55 vs 1.42)
>
> 이건 또 뭔지 모르겠네
> 영수증 크롤러? 실제 쓰나?
> 브라우저는 버전 높은거로 통합 하면 안돼?
> 무튼 그래서 신규로 모두 마이그해서 합쳐봐
> 그렇게 해서 신규 스펙으로 개발계에 새로 다띄워
>
> 그리고 전체 테스트해

목표: `crawler-worker` 단일 이미지에 receipt 역할을 추가해 3역할 통합, 상향된 라이브러리(playwright 1.55.0 / pydantic 2.12.0)로 개발계 크롤러 3종을 신규 스펙으로 교체 기동한다.
## 범위
### 포함
- `roles/receipt` 추가(pg-receipt-crawler 소스 무수정 복제) + `CRAWLER_ROLE=receipt` 분기 · requirements 3-way union
- playwright 1.55 상향 회귀 검증(3역할 전체) · 3역할 테스트 스위트 · thread/delegate 라우트 A/B 동등성
- 개발계(DELL) 3역할 신규 기동 + 기존 크롤러 3컨테이너 교체 및 롤백 절차
### 명시적 제외
- 운영(prod) 배포·이관, 원본 5개 저장소 수정, 크롤러 기능 개선·리팩터링, receipt 에 HTTP API 신설, CI/CD 변경
## 기능 요구사항
R1. `roles/receipt/` 파일 트리는 `/Users/steve/steve/legal-care/pg-receipt-crawler` 와 sha256 동일하다(`.git`·`__pycache__`·`.env*` 제외).
R2. `docker/entrypoint.sh` 는 role=receipt 시 `roles/receipt` 를 cwd 로 `python worker.py` 를 exec 한다(uvicorn·포트 바인딩 없음). 미지정·오타 역할은 기존대로 exit 64.
R3. `requirements.txt` 는 3역할 union 이며 playwright==1.55.0, pydantic==2.12.0 으로 통일하고 aiokafka·kafka-python 은 공존시킨다.
R4. playwright 1.55 에서 thread/delegate 소스가 깨지면 원본 저장소는 건드리지 말고 `roles/` 사본에만 최소 수정하고, 수정 파일·사유·diff 를 `roles/PATCHES.md` 에 기록한다(무수정으로 동작하면 R1 원칙 유지).
R5. `docker-compose.cutover.yml` 에 receipt 서비스를 추가한다. 기존 네트워크 별칭 `crawler-web`/`ai-delegate-web` 유지, receipt 는 원본과 동일 토픽·컨슈머 그룹을 쓴다.
R6. 개발계 교체는 별도 compose 프로젝트 또는 `docker run` 치환으로만 한다. 기존 개발계 compose 파일 묶음에 새 파일을 섞지 않는다(2026-08-07 스택 전체 중단 실사고).
R7. 교체 전 기존 3컨테이너의 이미지 태그·env·네트워크 별칭을 덤프해 남기고, 되돌리는 절차를 README 에 적는다.
## 수용 기준
AC1. (최우선) 통합 이미지에서 3역할 chromium 기동 + 대표 크롤링 경로 성공: thread `/check_id` · delegate 스크린샷 1건 · receipt KCS/KSTA 각 1건. 실패 시 R4 경로 처리 후 재검증.
AC2. `tools/dump_routes.py` 결과가 thread/delegate 원본 대비 라우트 집합 0 diff. receipt 는 라우트 0개임을 산출물로 확인.
AC3. 3역할 각각의 기존 테스트 스위트가 통합 이미지 안에서 실행되고 통합 전과 동일 결과(receipt 는 `tests/test_worker.py`).
AC4. 개발계에서 신규 스펙 3컨테이너가 healthy 이고 기존 3컨테이너는 정지·제거 상태다. `crawler-web`/`ai-delegate-web` 별칭 HTTP 응답 정상.
AC5. receipt 가 `...receipt.crawling` 테스트 메시지 1건을 소비해 `...receipt.notification` 로 발행하고 DLQ 증가 없음, 컨슈머 그룹 LAG 0 복귀.
AC6. 교체 전후로 `legalcare-local_legalcare` 망의 나머지 컨테이너가 모두 running 이다(스택 전체 중단 없음).
AC7. 롤백 왕복 리허설 1회 성공(기존 이미지로 되돌렸다가 신규로 재기동).
## 기술 제약
- 원본 5개 저장소 무수정. 통합물은 `crawler-worker` 저장소에만 만든다.
- 개발계 접속은 `ssh -p 50022 legalcare@114.203.1.178` 만 사용(192.168.1.55 직접 접근 불가).
- receipt 는 FastAPI·포트 없음 → 헬스체크를 프로세스/컨슈머 기준으로 정의해야 한다.
## 가정
A1. receipt 개발계 실사용량이 미미(KCS 24 · KSTA 1 · notification 0)해 교체 중 유실 위험이 낮다.
A2. HTTP 호출자는 `legacy-service` 하나(90일 1,075건)뿐이라 라우트 회귀 폭이 제한적이다.
A3. 이번 범위는 개발계까지이며 운영 배포는 후속 태스크다.
## 리스크
RISK: HIGH — 3역할 전부의 브라우저 엔진을 상향하고 개발계 크롤러를 전면 교체하며, 개발계 compose 조작으로 스택 전체가 끊긴 전례가 있다.
- P1 playwright 1.55 크롤링 동작 회귀 → AC1 선통과, 실패 시 R4 / P1 개발계 스택 중단 → R6·AC6 / P2 교체 중 카프카 유실 → AC5
## 관련 파일
- `crawler-worker/Dockerfile` · `docker/entrypoint.sh` · `requirements.txt` — 역할 추가·버전 상향의 3개 수정 지점
- `crawler-worker/docker-compose.dev.yml` · `docker-compose.cutover.yml` — 검증용 병렬기동(42130/42110) · 개발계 교체용
- `crawler-worker/tools/dump_routes.py` · `tools/route_inventory.py` — 라우트 A/B 비교 근거 생성
- `crawler-worker/roles/thread` (72파일) · `roles/delegate` (108파일) — 원본 sha256 동일 사본, R4 발생 시에만 수정
- `crawler-worker/README.md` · `.env.example` — 역할·롤백 절차 문서화 대상
- `pg-receipt-crawler/worker.py` — 카프카 소비 루프(receipt 엔트리포인트) · `topic.py` — 4개 토픽 정의
- `pg-receipt-crawler/crawlers/{kcs,ksta}_receipts_crawler.py` — playwright 1.55 사용처, AC1 핵심 검증 대상
- `pg-receipt-crawler/{config,models,utils,logger,vault,s3_save_review_boost_receipt}.py` · `tests/test_worker.py` · `Dockerfile` · `requirements.txt` — 복제 및 union 입력
## 의도 확인
사용자 요청 재진술: "영수증 크롤러까지 포함해 세 개를 하나로 합치고, 브라우저는 높은 버전으로 통일해서 개발계에 새로 띄우고 전부 테스트하라."
이 지시서는 (1) receipt 를 3번째 역할로 추가(R1~R3), (2) 1.55 통일과 그로 인한 회귀 처리 방침(R4·AC1), (3) 개발계 교체를 안전하게·되돌릴 수 있게(R6·R7·AC4·AC6·AC7), (4) 전체 테스트를 역할별 스위트+동등성+실기동으로 분해(AC2·AC3·AC5) 하여 답한다.
## OPEN QUESTIONS
없음
