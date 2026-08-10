# 설계: D 리뷰부스터 v2 통합앱 이관 계획 (R5·R7·R8·R9)

R6 DDL은 이번에 작성·검증까지 끝냈고(04-changes-D-ddl.md), 이 문서는 나머지 코드 이관의 파일 단위 지도다. 구 코드는 전부 origin/main에서 실측했다.

## 모듈 배치 원칙 — 의존 방향이 결정한다

booster-app 모듈 의존을 실측하니 `notification → product`(modules/notification/build.gradle, NotificationProductLocalAdapter 선례)와 `product → crawling`(modules/product/build.gradle)이 이미 있다. 즉 **product→notification, crawling→product 직접 의존은 사이클이라 불가**. 그래서:
- **발송본체(알림톡)는 notification 모듈로**: notification이 product에 의존하므로 boost_message/customer/template 엔티티(product 소유)를 직접 쓸 수 있다. medilawyer-boot의 자기완결 구조가 거의 그대로 보존된다. 알림톡 발송이 필요한 역방향 접점(product 구독등록, crawling 자동발송)은 기확립 패턴대로 **인터페이스는 호출 모듈에, 구현은 app 계층 LocalAdapter(@Primary)** 로 잇는다(UserLocalAdapter/ProductCrawlingLocalAdapter 선례).
- 통합앱이라 notification의 읽기측 별도 엔티티(d40664f)는 **쓰기측 엔티티로 통합**한다(이중 엔티티 불필요). 단 boost_message_template류는 user 모듈 기존 엔티티와 JPA 이름 충돌 → `@Entity(name="NotificationBoostMessageTemplate")`·`@Repository("notificationBoostMessageTemplateRepository")` 리네임은 여전히 필요(01-requirements 기술 제약 그대로).

## 파일 매핑 (구 origin/main → 신 booster/reviewed)

| # | 구 파일(그룹) | 목적지 | 적응 |
|---|---|---|---|
| 1 | medilawyer-boot `apis/reviewBoost/v2/` 서비스 12종+converter 3+dto 10 (MessageSend/MessageCreation/Delivery/RecipientBlock/RequestSetting/HospitalTemplatePolicy/AlimTalkService·PayloadRenderer/CustomerCreation/SendAuthorizationValidator) | `modules/notification/.../notification/reviewboost/` | Kotlin→Java 전환(아래 절), 패키지 `legalcare.medilawyer.notification.reviewboost` |
| 2 | medilawyer-boot `apis/reviewBoost/v2/controller/ReviewBoostV2Controller.kt` (POST `/api/v2/boost1/organizations/{o}/teams/{t}/messages`) | notification controller | 경로 유지 필수(R9 게이트웨이 프리픽스 매칭) |
| 3 | medilawyer-boot `apis/internal/controller/InternalCrawlingReviewBoostController.kt`(자동발송)·`InternalProductReviewBoostController.kt`(PUT request-setting) + request/response 모델 5종 | notification internal controller — 단 통합앱에선 HTTP 대신 **app 계층 어댑터가 1차 소비자**(컨트롤러는 기존 internal 관례 따라 유지 여부만 결정) | de-Feign |
| 4 | medilawyer-boot `db/domain/{messageDelivery,alimTalkBlocklist,reviewRequestSetting}` 엔티티 3+repo 3 | notification persistence | Kotlin→Java, BaseEntity→@CreatedDate/@LastModifiedDate 직접 부여 |
| 5 | medilawyer-boot `module-client/.../bizmessage/ReviewBoostV2BizMessageService.kt`(SENS HMAC 서명 직접발송) + `enums/` 7종(KakaoAlimTalk*, ReviewBoostDeliveryMode) | notification `client/alimtalk/`(기존 디렉토리)·core code | RestTemplate→기존 신 앱 HTTP 클라이언트 관례 |
| 6 | product-service v2 스택: 엔티티 6(MatchAnalysis·SendRequest·PlatformClick·StorePresentSetting·PresentLedger·Order확장)+repo 7+`ReviewBoostService`·`ReviewPresentService`·`BoostPresentOrderTransactionService`·`BoostStorePresentSettingService`·`SubscribeProductRegistrationService`+컨트롤러 모델·검수 API·`StorePresentValidator` | `modules/product/` 기존 동일 경로에 **덮어쓰기 병합**(신 앱에 구버전 ReviewBoostService:119 등 이미 존재) | import 리매핑, jOOQ 잔재 제거(구 커밋이 이미 삭제) |
| 7 | crawling-service 체인A: `ReviewBoostAutoSendService`+`client/legacy/ReviewBoostAutoMessageRequest`+`ReviewBoosterService` 갱신분(59bf680→843aba2 순효과) | `modules/crawling/` | legacy Feign 호출 → `ReviewBoostAutoSendClient` 인터페이스(crawling 모듈) + app `CrawlingNotificationLocalAdapter` 구현 |
| 8 | notification-service `ReviewBoostTemplateService`+`ReviewBoostV2Controller`(`/apis/v2/review-boost/...`)+예외 4종+에러코드 08004~08007 (a6d5156) | modules/notification | 엔티티는 #4로 통합, 게이트웨이 `/notification/apis` 라우트 기존재 |
| 9 | reviewed-admin-web 병원템플릿 전환 배선(1ed9ef3 legacy 알림톡 클라이언트층 포함) | `apps/reviewed-app/app/.../api/domain/adminweb/` | Feign→RestClient(PORTING-NOTES 관례), booster internal 2엔드포인트 HTTP 호출(타 앱이라 LocalAdapter 불가) |
| 10 | gateway `Routes.java:35` `legacy-v2`→`ServiceId.LEGACY` | `ServiceId.BOOSTER`로 전환 + RouteTable 테스트 | `/api/v2` 사용 컨트롤러가 ReviewBoostV2Controller뿐임을 실측 — 프리픽스 통전환 안전 (R9) |

## Kotlin→Java 전환 원칙 (발송본체)

booster-app은 Java 전용(빌드에 kotlin 플러그인 없음)이라 "코드 그대로"의 실제 의미는 **구조·이름·로직 1:1 보존 변환**이다: 클래스/메서드/필드명 유지, data class→record 또는 기존 모델 관례, `runCatching`→try-catch, protected set→명시적 상태전이 메서드(`switchToHospitalCustom` 등 그대로). 교훈 승계 2건 — REQUIRES_NEW는 runCatching 안이 아니라 TransactionTemplate으로(crawling_fail_log 리뷰 교훈), jsonb 저장 전 NUL 제거.

## R7 안전장치 — 각각 어디에 사는가

- **SHA256 멱등키** `review-boost-auto-<SHA256>`: 생성은 crawling `ReviewBoostAutoSendService`(#7), 소비·중복거절은 notification 발송 서비스의 `findByIdempotencyKey` + DB 최후방어 `uq_kash_idempotency_key`(R6 DDL, 테스트 통과).
- **수신차단**: `ReviewBoostRecipientBlockService`(#1) — send()/sendAuto() 최앞단(메시지·멱등키 생성 이전), 차단 시 BLOCKED 감사 이력만 기록(CHECK에 BLOCKED 포함 확인 완료).
- **공통이력**: `KakaoAlimTalkSendHistory.pending/blocked` + `claimPending`(상태 선점 UPDATE — 동시 발송 방어).
- **callbackOrder 멱등가드**(82a6fcd): 구 `ReviewBoostService.callbackOrder:153` — `findByUuidForUpdate` 비관적 잠금 + 최종상태 주문 중복·역순 콜백 no-op + 연결된 send_request 완료 처리. 신 앱 `ReviewBoostService.callbackOrder:119`에 가드 없음(실측) → #6에서 구 버전으로 교체 + `ReviewBoostServiceCallbackOrderTest` 동반 이식. **돈버그라 #6에서 최우선.**

## R8 백필 / R9 라우트

R8은 R6에서 이미 스크립트화(`db/reviewboost-v2-backfill.sql` — ACTIVE 선물→setting 시드, 멱등 재실행 검증 완료). 적용 순서 강제: DDL→백필→#6 배포. 미백필 배포 시 전면중단 사유는 resolveActivePresent가 설정 테이블만 보기 때문. R9는 #1~#8 완료 후 마지막에 전환한다 — 먼저 바꾸면 v2 수동발송이 미구현 booster로 라우팅돼 즉사.

## 왜 한 번에 못 옮기나 → 커밋 단위 의존 순서

규모(~70파일+Kotlin 전환), 사이클 검증이 필요한 의존 엣지 2개, 발송 활성화의 비용 리스크(A2 게이트 최종단), post 모듈 WIP(mldelegator) 회피가 이유다. 순서: **0)** DDL+백필(완료) → **1)** #6 product v2 스택+callbackOrder 가드(발송은 아직 legacy가 담당 — 독립 검증 가능) → **2)** #1~#5 발송본체+공통 인프라(수동발송 e2e, A2 스텁) → **3)** #8 notification 조회측 → **4)** #7 crawling 자동발송 전환(발송 ON, QA 게이트) → **5)** #9 adminweb → **6)** #10 게이트웨이 전환+회귀. 각 단계 셀프리뷰 커밋 1개.

STATUS: DONE
