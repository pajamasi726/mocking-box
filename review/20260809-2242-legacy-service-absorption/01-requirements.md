# 작업지시서: legacy-service(medilawyer-boot) → booster-app 흡수
## 배경 및 목표
> 다 하고 나면 레거시 서비스도 부스터 api 로 합쳐버려 그리고 개발계에서 전체 테스트해 자체 리뷰로
> 기존에 테스트한 방식있을꺼야 그대로 셀프 리뷰로 해서 진행해 진행한 방식 따로 기록도 해두고
> 구는 그대로 두고 신규를 시스템 전체 한셋트를 옮겨서 100% 맞추는게 목표야
> 응 진행해, 그리고 개발계에서 전체 테스트 다해

구 legacy-service(638파일·28,861줄·컨트롤러 38·`/api/v1` 147개)를 booster-app 으로 **1:1 이식**하고, 개발계 골든셋 리플레이 차등검증으로 동등을 입증한 뒤 게이트웨이 `/api/v1` 라우트를 BOOSTER 로 돌린다. 구는 정지하지 않는다.
## 범위
- 포함: 신규 PG 스키마 동기화(접촉분) · `/api/v1` 147개 · `/internal/**` 4컨트롤러 · `@KafkaListener` 1 · `@Scheduled` 3 의 booster 이식 / 인증 경계 이설 / 게이트웨이 라우트 전환 / 개발계 전체 검증 / 진행방식·판정 기록
- 제외: 구 legacy-service·legacy-bridge 정지·삭제, 운영 배포·운영 라우트 전환, 리팩터링·성능개선·API 계약 개선, **미접촉 테이블 DDL·구 PG 데이터 전량 복사**, `medilawyer-boot` 저장소 수정·push, `/api/v2`(선행 태스크 완료분)
## 단계 분할 · 기능 요구사항(R) · 수용 기준(AC) — 표 순서 = 실행 순서, 앞 단계 AC 그린 없이 다음 단계 착수 금지
| 단계 | R# 이식 범위 (구 → 신) | AC# 완료 판정 |
|---|---|---|
| P-1 스키마 동기화(선행) | R11 신규 PG(`deploy-postgres-new-1`, public 94)가 구 PG(`postgresql` 컨테이너, public 153)보다 **59개 부족**(파티션 접으면 논리 약 18 — 알림톡 발송이력·크롤 실패로그·리뷰부스터 v2·GEO·병원공식 계열, 상세 `00-log.md`). ①147 엔드포인트 × 59테이블 **접촉 매핑** ②접촉분만 DDL 선행 적용 ③데이터는 **전량 복사 금지**, 리플레이 판정에 필요한 접촉 테이블만 캡처 시점 기준 시드(빈 표는 조회 응답이 구와 달라져 불일치 노이즈가 된다) ④미접촉분은 범위 밖으로 명시 | AC11 접촉 매핑표(엔드포인트→테이블) 공란 0 + 접촉 테이블이 신규 PG `\dt` 에 존재하고 DDL 재실행 멱등 + 미접촉 목록이 "범위 밖" 근거와 함께 남는다. 접촉 판정은 구 소스 정적탐색 + P1~P4 리플레이 로그의 `relation does not exist` **0건**으로 교차 확인 |
| P0 기반·인증 | R1 booster 에 `modules/legacy` 신설(경로·요청/응답 바디·HTTP 상태·에러코드 무변경 틀) + `HealthCheck`(`/api/v1/health`). R2 구 인증 필터의 경로별 인증 요구를 booster 필터로 이설. 라우트는 LEGACY 유지 | AC1 `:app:test` 그린 + 기동 로그에 `/api/v1/health` 매핑 노출 + 게이트웨이는 여전히 legacy-bridge 로 감(`RouteTableTest` 무변경). AC2 147경로 인증경계 대조표(구 필터 vs `bypass.url`) 공란 0 + 무토큰 요청 10경로 응답코드가 구와 동일 |
| P1 읽기(저위험) | R3 `PublicHospitalController`·`CommunityController`·`PostWordController`·`StoreChannelController`(검색5)·`PostController`·`PostReplyController`·`PostReplyAiRecommendController` | AC3 해당 경로 read 샤드 리플레이에서 **"구 200 → 신 비200" 0건**, 불일치는 `err_clusters.py` 원인별로 회귀/데이터차이 판정 기록 |
| P2 코어 쓰기 | R4 `Store`·`AppUser`·`AppUserStore`·`AppUserChannelAccount`·`Team`·`TeamAppUser`·`TeamAuth`·`TeamChannelAccount`·`Organization`·`OrganizationAppUser`·`Notice`·`Product`(`products/{name}/purchases`) | AC4 write 샤드 **순차** 리플레이(시작 전 신규 PG 를 캡처 시점 상태로 되돌림) 후 회귀 0 + 요청별 DB 행 변화 일치 |
| P3 인증 발급 | R5 `AuthController`·`OtpController`·`JwtController`(`/api/v1/{auth,otp,jwt}`) | AC5 구 발급 토큰이 신에서 검증되고 그 역도 성립(구/신 × 발급/검증 4조합) + OTP·SMS 는 스텁, 실발송 0건 |
| P4 부스트·관리자·비HTTP | R6 `BoostController` 15(`/api/v1/boost1/**`)·`AdminController`·`AdminSeoController`·`Airtable`·`Ml`·`Kafka`·`EventScheduler`·`Scheduled` + `/internal/**` 4 + `KafkaConsumer` + `@Scheduled` 3 | AC6 boost1 리플레이 회귀 0 + 기프트·발송 원장 이중기록 0건 + `modules/crawling` 의 `LegacyService` HTTP 호출이 인프로세스로 전환되어 booster→legacy HTTP 호출 0건 |
| P5 개발계 전체 검증 | R7 골든셋 35파일 전량 리플레이 + 리플레이가 못 보는 3종 + 신규 셋트 전체 기동 | AC7 ①전량 리플레이 회귀 0(원인별 판정표 첨부) ②카프카 컨슈머 그룹 참여 + 토픽 왕복 정상1·실패1 ③`@Scheduled` 3건 수동 트리거 성공 ④신규 스택 전 컨테이너 healthy·기동 예외 0 ⑤외부 실호출 0건 |
| P6 라우트 전환 | R8 `Routes.java:34` `legacy` 목적지 `ServiceId.LEGACY`→`BOOSTER`, `RouteTableTest` 갱신 | AC8 개발계 게이트웨이 경유 `/api/v1` 호출이 booster-web 에 도달(양쪽 액세스 로그 실측) + 1줄 revert 로 즉시 원복되는 것을 실제로 1회 재현 |
| 횡단 A | R9 **147개 전량 이식** — APM 미사용 71개를 폐기 근거로 쓰지 않는다(관측창 한계). 실사용 76은 리플레이로, 미사용 71은 계약 스모크(경로·메서드·인증경계)로 판정. 진행 방식·판정을 `{TASK_DIR}` 에 단계별 기록 | AC9 이식 대조표 147행(구 파일:라인 → 신 파일:라인 + 검증방식 리플레이/스모크 + 판정) 누락 0 + 방식 기록 문서 존재 |
| 횡단 B | R10 병행·롤백: 구 legacy-service·legacy-bridge 를 정지·삭제하지 않는다. 신규의 스케줄러·카프카 컨슈머는 **기본 OFF 플래그**, 검증 시에만 개발계에서 구를 잠시 내리고 단독 실행. 단계마다 커밋 1개 | AC10 구·신 동시 가동 중 알림톡 발송이력·크롤 트리거 **이중 실행 0건**(양쪽 DB 각각 카운트 실측) + 단계별 롤백 절차 1줄씩 문서화(신규 PG 에 쌓인 행 처리 방침 포함) |
## 기술 제약
- Kotlin→Java **1:1 이식**(클래스·메서드·필드명·로직·URL 경로 보존). 개선·리팩터링·계약 변경 금지 — 선례 `review/20260809-0854-reviewboost-v2-port/01-requirements.md` R2. booster 는 Java 전용(kotlin 플러그인 없음).
- 구·신은 **서로 다른 PG 인스턴스**다(구 `127.0.0.1:5432`=`postgresql` 컨테이너 / 신 `deploy-postgres-new-1:5432`, DB명은 둘 다 medilawyer). 병행 기간에는 **같은 업무가 양쪽 DB 에 따로 쌓인다** — 구로 들어온 요청은 구 PG 에, 신으로 들어온 요청은 신 PG 에 기록되어 발송이력·원장·시퀀스가 이원화되므로, 개발계 검증 트래픽 외에 실업무를 신으로 흘리지 않고 롤백 시 신 PG 잔여 행 처리를 반드시 정한다. (저장소 `core-storage.yml` 의 AWS RDS 주소는 실환경 근거가 아니다 — 구는 `spring.profiles.active=workstation,git,vault` 로 config-server·Vault 에서 설정을 받는다.)
- booster `TokenVerificationFilter.useToken()` 은 `bypass.url` 을 **substring** 매칭(`application.yml:321`) → 이식 즉시 `/api/v1/**` 전체가 인증 대상이 된다. R2 무결정 시 흡수와 동시에 전 요청 401 회귀.
- 모듈 간 역방향 의존 금지(인터페이스는 호출 모듈, 구현은 app 계층 `*LocalAdapter` @Primary). Feign 폐기 — 내부·외부 HTTP 는 RestClient. 검증 절차는 `검증방식-리플레이-차등검증.md` 를 그대로 따른다.
## 가정
- A1. 착수 기준 브랜치는 리뷰부스터 v2 이관(`feature/prod-sync-candidates`, `legacy-v2` 라우트 포함)이 반영된 상태다. 미반영이면 해당 worktree `/Users/steve/steve/legalcare-renew-prodsync-wt` 에서 작업한다.
- A2. 검증 전제 — 골든셋 35파일에 `/api/v1` 트래픽이 포함되고, write 샤드 전 신규 PG 를 캡처 시점 상태로 되돌릴 수 있으며, **개발계에 한해** 구의 스케줄러·컨슈머를 일시 정지할 수 있다.
- A3. 신규 PG 에 DDL 을 적용할 권한이 있고, 부족 59개의 스키마 원본은 구 PG 에서 `pg_dump -s` 로 얻는다(운영 DDL 소유권은 기존 절차 유지, 이번 작업은 개발계 신규 PG 에만 적용).
## 관련 파일
- `medilawyer-boot/module-core/core-api/.../{apis/http,apis/reviewBoost,apis/internal,admin,auth,schedule,kafka,eventScheduler,notice,airtable}/` — 이식 원본 38컨트롤러(경로는 각 `@*Mapping` 문자열 그대로) · `module-client/core-client/.../ml/service/MlService.kt`(크롤러 HTTP 호출 주체) · 구 실제 DB = 개발계 `postgresql` 컨테이너(153테이블)
- `legalcare-renew/apps/booster-app/modules/`(9모듈 + 신설 `legacy`) · `app/src/main/resources/application.yml:291-321`(jwt·bypass) · `modules/external/.../InternalLegacyController.java` · `modules/crawling/.../client/legacy/service/LegacyService.java:16`(`/api/v1/boost1/.../messages` HTTP 호출 → P4 인프로세스 전환) · `apps/gateway-app/.../route/Routes.java:34` + `RouteTableTest` · `docker-compose.yml:97,128,190`(스텁 주입)
- 검증: `legalcare-consulting/_analysis/legalcare-map/08-신개발계/검증방식-리플레이-차등검증.md`(절차 원본) · 개발계 `/mnt/ex_disk1/{prod-capture,replay-20260807,renew-replay}/` · 신규 DB = `deploy-postgres-new-1`(94테이블) · 접속 `ssh -p 50022 legalcare@114.203.1.178`
## 의도 확인
재진술: "레거시 서비스를 부스터 API 로 합치고, 구는 그대로 둔 채 신규 한 셋트로 100% 맞춘 뒤 개발계에서 전체 테스트하고 방식을 기록하라." → 28,861줄을 한 번에 옮기면 검증이 불가능하므로 위험도·의존 순으로 P-1~P6 으로 쪼개고 각 단계 완료를 리플레이 판정(AC)으로 못 박았다. "한 셋트"에는 코드뿐 아니라 **신규 PG 에 없는 스키마**가 포함되므로 P-1 을 선행에 뒀고, "구는 그대로"는 R10(정지·삭제 금지 + 스케줄러/컨슈머 OFF + 라우트 1줄 revert), "100%"는 R9(147 전량 이식 + 대조표), "전체 테스트"는 R7(리플레이 + 카프카 + 스케줄러 + 전체 기동 + 외부 0)로 답한다.
## OPEN QUESTIONS — 없음 | 개정 이력 v2(검수 반려 반영): "구·신 동일 DB · 이관 불필요" 전제 철회(실측: 별도 PG 인스턴스, 신규에 테이블 59개 부재) → P-1 스키마 동기화 단계 신설(R11/AC11), 이중매핑 제약을 **병행기간 데이터 이원화** 위험으로 재작성, 범위 포함/제외·가정 A2·A3 갱신. 나머지 단계 분할·전량 이식·검증 강도 차등은 v1 유지.
RISK: HIGH — 돈(기프트·알림톡)과 인증 발급 주체 이동에 더해, 구·신이 **다른 PG** 를 보는 상태에서 신규에 테이블 59개가 없어 스키마 선행 없이는 이식분이 런타임에 깨지고 병행 중 업무 데이터가 양쪽에 이원화된다.
