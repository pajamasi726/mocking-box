# 작업지시서: legacy 실사용 검증(A) + lawkit-admin 흡수(B)
## 배경 및 목표
> 일단 제일 쉬운건 lawkit-admin 을 흡수시키는거고 그 다음에 레거시 서비스는 실제 호출되는게 몇개나 되나, 110여개를 다 쓰나? 셀프 리뷰로 검증해봐 로킷 어드민은 합쳐

A) 선실측(legacy 147개 중 실사용 21개 = 14%)을 셀프리뷰로 검증·보강해 은퇴 로드맵을 낸다. B) lawkit-admin(FastAPI, 엔드포인트 35개, 약 8.1k LOC)의 edge 를 lawkit-app 으로 흡수하고 게이트웨이를 `ServiceId.LAWKIT` 으로 전환한다.
## 범위 (TASK_DIR = `/Users/steve/steve/mocking-box/review/20260809-1355-legacy-usage-lawkit-admin`)
- 포함: (A) 검증 문서 `{TASK_DIR}/legacy-usage.md`, 코드 무변경 · (B) lawkit-app `analysis` 도메인 신설 + Python 내부 위임 + 게이트웨이 라우팅 전환 + 분류표 `{TASK_DIR}/absorb-plan.md` + 개발계 빌드·라우팅 검증
- 제외: legacy 엔드포인트 실제 삭제/배포, Python LLM 파이프라인(agents·llm·외부수집 약 5.9k LOC)의 Java 전면 재작성, lawkit 프론트 수정, `.env.prod` 평문 시크릿 로테이션, push
## 기능 요구사항
R1. [A] 실사용 21개 각각에 대해 신 booster-app 대응 유무를 `있음/부분/없음` + 근거 파일 경로로 대조한다.
R2. [A] legacy 147 − 21 = 126개 미사용분을 `폐기후보 / 저빈도 실사용 가능(배치·관리자·월간) / 확인필요` 로 전량 배정하고, 분류 규칙과 "26시간 캡처 창" 한계·보완 측정 1건을 문서 최상단에 명시한다.
R3. [A] legacy 은퇴를 3단계 이상으로 순서화하고, 각 단계에 "무엇이 참이면 다음 단계로 간다"는 통과 판정 기준을 1줄씩 붙인다.
R4. [A] 코드를 수정하지 않는다. 산출물은 표와 판정뿐이다.
R5. [B] lawkit-admin 35개(router 11 · admin_router 18 · evidence_router 5 · /health 1)를 전량 `Java 이관` / `Python 잔류` 로 분류한다. 판정 기준: 요청 처리 경로에 Gemini 호출·pipeline 실행·외부 수집(superlawyer/lbox/moleg)·파일 텍스트추출이 **하나라도** 있으면 Python 잔류, 나머지(DB 조회·갱신·삭제, S3 스트리밍, 플래그 세팅)는 Java 이관.
R6. [B] Java 이관분은 `legalcare.lawkit.api.domain.analysis` 로 구현하고, 대상 9테이블(analysis_requests/results/precedent_map/evidence, precedents, precedent_graphs, precedent_notes, chat_messages, llm_prompts, llm_model_configs)은 `admin` 스키마 원본을 읽는다 — `lawkit` 스키마 복제·데이터 이중화 금지.
R7. [B] Python 잔류분은 게이트웨이에 직접 노출하지 않는다. lawkit-app 이 유일한 edge 로서 RestClient 로 내부 위임하고, 위임 응답 본문을 가공 없이 그대로 반환한다.
R8. [B] 응답 계약 무변경 — 기존 FastAPI 응답 형태(snake_case 필드명, `Api<T>` 미적용, 상태코드)를 필드 단위로 유지한다. lawkit 프론트는 수정 대상이 아니다.
R9. [B] 인증 경계를 명시적으로 결정한다(`/admin`·`/analysis` 를 bypass 로 둘지, 인증 필수로 올릴지). 결정·근거를 문서화하고 설정에 반영한다. 무결정 시 흡수 즉시 전 요청 401 회귀.
R10. [B] 게이트웨이 `Routes.java` 의 `lawkit-admin`·`lawkit-analysis` 라우트를 `ServiceId.LAWKIT` 으로 전환하고 `RouteTableTest` 를 갱신한다. `ServiceId.LAWKIT_ADMIN` 과 `GatewayProperties` URI 는 내부 위임이 남는 한 유지한다.
## 수용 기준
AC1. `legacy-usage.md` 에 실사용 21행 대조표(대응 판정 + 근거 경로)와 미사용 3분류표가 있고, 3분류 합계가 126 과 산술적으로 일치한다.
AC2. `legacy-usage.md` 최상단 10줄 안에 26시간 창 한계 + 보완 측정 제안이 있고, 은퇴 단계마다 통과 판정 기준이 붙어 있다.
AC3. `absorb-plan.md` 분류표 행 수 = 35, 각 행에 Java/Python 판정 + 판정 근거(호출하는 외부 의존)가 있다. Python 판정 행은 게이트웨이 라우트를 갖지 않는다.
AC4. `apps/lawkit-app` `./gradlew build` 그린, `apps/gateway-app` `./gradlew test` 그린, `verification/baseline/analysis.json` 추가로 신설 표면이 `verify_surface.py` 에 인식된다.
AC5. 개발계에서 게이트웨이 경유 호출 시 Java 이관 엔드포인트 3개 이상이 2xx, Python 위임 엔드포인트 1개 이상이 2xx 이며, 각 응답의 JSON 최상위 키 집합이 흡수 전 lawkit-admin 응답과 동일하다.
AC6. 인증 결정대로 검증된다 — bypass 결정 시 미인증 요청 200, 인증 필수 결정 시 토큰 없는 요청 401.
AC7. 변경 범위가 worktree `/Users/steve/steve/legalcare-renew-prodsync-wt` 내부에 한정되고, lawkit-admin 저장소 변경 0, push 0.
## 기술 제약
- lawkit-app = Java 25 / Spring Boot 4.0.7 / 2모듈(`core`,`app`) / 도메인 패키지 `legalcare.lawkit.api.domain.<도메인>` (apps/lawkit-app/settings.gradle, build.gradle:9,40)
- Feign 폐기(2026-08-03) — 내부·외부 HTTP 는 `RestClient` 만 사용 (build.gradle:41-42, `*RestClientImpl.java` 패턴)
- DB: **동일 RDS 인스턴스·동일 계정(lawkit), 스키마만 다름** — Python=`admin`(lawkit-admin/.env.prod:7), Java=`lawkit`(application-release.yml:10). 스키마 DDL 소유는 Alembic 유지, Flyway/`ddl-auto` 이중관리 금지(엔티티는 매핑만)
- 게이트웨이 RouteTable 매칭 = `prefix equals || startsWith(prefix+"/")` → `/analysis/{id}/...` 같은 동적 경로를 게이트웨이에서 Java/Python 으로 가를 수 없다. 분기는 lawkit-app 내부에서만 한다
- 인증: lawkit-app 은 `TokenVerificationFilter` 적용, `bypass.url.base`(application.yml:103)에 `admin`·`analysis` 없음. lawkit-admin 은 앱 레벨 인증이 없다(게이트웨이 통과 전제)
## 가정
A1. "흡수"의 1차 종착점은 **edge 단일화**(게이트웨이·프론트가 보는 서비스 1개)이며, Python 컨테이너는 게이트웨이 미노출 내부 워커로 잔존한다. 컨테이너 완전 제거는 후속 과제.
A2. lawkit-app 이 동일 계정으로 `admin` 스키마를 읽을 수 있다 — 권한은 개발계에서 실증한다.
## 리스크
RISK: HIGH — 운영 프론트가 쓰는 `/admin`·`/analysis` 의 백엔드 주체 교체 + 인증 경계 이동(공개 경로 계약 변경 + 인증 변경). 완화: ①인증 결정을 R9 로 선행 계약 ②응답 키 집합 동등성 개발계 실측(AC5) ③라우팅 전환은 최종 단계, 되돌리기는 `Routes.java` 2줄 revert.
## 관련 파일
- `/Users/steve/steve/legal-care/lawkit-admin/app/services/analysis/{router,admin_router,evidence_router}.py`, `app/main.py` — 흡수 대상 35개 엔드포인트 원본
- `.../lawkit-admin/app/services/analysis/{models,llm_models}.py`, `alembic/versions/`(7개) — `admin` 스키마 9테이블 정의·마이그레이션 소유자
- `.../lawkit-admin/app/services/analysis/{pipeline.py,agents/,superlawyer/,lbox/,moleg/}`, `app/core/{llm,storage}/`, `.env.prod` — Python 잔류 후보 본체 및 접속 정보(평문 시크릿 주의)
- `/Users/steve/steve/legalcare-renew-prodsync-wt/apps/lawkit-app/` — `app/src/main/java/legalcare/lawkit/api/domain/`(신설 위치), `app/src/main/resources/{application.yml,application-release.yml,config/user.yml}`, `verification/{verify_surface.py,baseline/}`
- `.../apps/gateway-app/src/main/java/legalcare/medilawyer/gateway/` — `route/Routes.java:42-43`, `route/ServiceId.java:10-11`, `config/GatewayProperties.java:31-32`, `src/test/.../route/RouteTableTest.java:117-118,140`
- [A 근거] `/Users/steve/steve/legal-care/medilawyer-boot`(origin/main, legacy 147개 원본) · `.../apps/booster-app/modules/{user,post,notification}/`(대응 확인처) · `/private/tmp/claude-501/-Users-steve-steve-mocking-box/8f23c7f7-7c11-47e6-b28c-5c139f6de797/scratchpad/synthesis.md`
## 의도 확인
사용자 요청 = "lawkit-admin 을 신 앱에 합치고, legacy 110여개가 실제로 다 쓰이는지 셀프리뷰로 확인하라".
B는 "합쳐"를 엔드포인트 단위 판정 기준(R5)으로 못 박아 실행 가능하게 만들고 게이트웨이 전환까지 계약화했다. A는 21/147 실측을 대조표·3분류·은퇴 순서로 검증해 "다 쓰나?"에 근거 있는 답을 남긴다. 되돌리기 어려운 행위(legacy 삭제, LLM 재작성)는 전부 범위 밖으로 뺐다.
## OPEN QUESTIONS
- (진행 차단 아님) 흡수의 최종 목표가 "Python 컨테이너 0"이라면 LLM 파이프라인 약 5.9k LOC 의 Java 재작성이 별도 대형 과제로 필요하다. 본 태스크는 A1(edge 단일화)로 진행하며, 목표가 다르면 알려주면 범위를 재작성한다.
