# 작업지시서: 리뷰부스터 알림톡 v2 통합앱 이관 (개발계 검증까지)

## 배경 및 목표
> 음? 코드가 대형 알림톡? 인건가? 알림톡 부분만 새로들어간거 아니야? 그거 PR만 가져와서 옮기면 되잖아. 검토하고 바로 착수해서, 개발계테스트까지 진행해. 배치는 이미 되어져 있는거로 알아. 코덱스 없으니 동일하게 싱글로.

릴리님 v2 PR 전량(prod 179파일 +6,354줄)을 통합앱으로 옮기고 개발계에서 도는 것까지 확인한다. DDL·백필(커밋 `691046c`)·배치잡·`/api/v2` 라우트는 선행 완료 → 이번 범위는 **코드 이관 + 개발계 구동검증**. 파일단위 매핑·모듈배치·의존순서는 `03-design-D.md`를 그대로 승계한다(재설계 금지).

## 범위
- 포함: R1~R6 이관 6단계(순차) + 횡단 안전장치(R7) + 개발계 구동검증(R8) + 설정·스텁(R9). 단계마다 커밋 1개 + 셀프리뷰 기록(Codex 불가, 싱글 진행).
- 제외: 운영 배포·운영DB 적용, `booster-app/modules/post/.../client/mldelegator/` WIP 구역, 배치 잡 신규작성(이미 존재), adminweb Thymeleaf/static JS 자산(→R5에서 서버측만, 화면은 별도 태스크로 분리), 실제 알림톡 발송 활성화.

## 기능 요구사항 / 수용 기준 (표 순서 = 실행 순서. 앞 단계 그린 없이 다음 단계 착수 금지)
| R# | 단계 | 구 소스 → 신 목적지 모듈 | AC# 수용 기준 |
|---|---|---|---|
| R1 | 1. product v2 스택 + **callbackOrder 돈버그 가드** | product-service v2(엔티티6·repo7·`ReviewBoostService`/`ReviewPresentService`/`BoostPresentOrderTransactionService`/`BoostStorePresentSettingService`/`SubscribeProductRegistrationService`·검수API·`StorePresentValidator`) → `booster-app/modules/product/` 동일경로 덮어쓰기 병합 | AC1 `./gradlew :app:test` 그린 + `ReviewBoostServiceCallbackOrderTest` 동반 이식·통과(중복·역순 콜백 no-op, 원장 이중기록 0건) |
| R2 | 2. 발송본체(수동) | medilawyer-boot `apis/reviewBoost/v2/` 서비스12+converter3+dto10, 엔티티3(messageDelivery·alimTalkBlocklist·reviewRequestSetting)+repo3, `module-client/bizmessage/ReviewBoostV2BizMessageService`(SENS HMAC)+enum7, `ReviewBoostV2Controller` → `modules/notification/.../notification/reviewboost/` (Kotlin→Java 1:1) | AC2 POST `/api/v2/boost1/organizations/{o}/teams/{t}/messages` 경로 **불변**으로 게이트웨이 경유 정상응답 + 발송 어댑터는 스텁 호출(외부 실발송 0) |
| R3 | 3. notification 조회측 | notification-service `ReviewBoostTemplateService`·`ReviewBoostV2Controller`(`/apis/v2/review-boost/...`)·예외4·에러코드 08004~08007 → `modules/notification`(엔티티는 R2로 통합, 읽기/쓰기 이중엔티티 금지) | AC3 `/notification/apis/v2/review-boost/...` 조회 정상응답 + 기동 시 JPA 엔티티/빈 이름충돌 0 |
| R4 | 4. crawling 자동발송 전환 | crawling-service `ReviewBoostAutoSendService`·`client/legacy/ReviewBoostAutoMessageRequest`·`ReviewBoosterService`(59bf680→843aba2 순효과) → `modules/crawling/` + legacy Feign 제거, `ReviewBoostAutoSendClient`(crawling 인터페이스) ← app `CrawlingNotificationLocalAdapter` 구현 | AC4 동일 입력 자동발송 2회 트리거 시 `uq_kash_idempotency_key`로 1건만 기록·2번째 중복거절, 환불 0원건 제외 확인 |
| R5 | 5. adminweb 배선(서버측 한정) | reviewed-admin-web 병원템플릿 전환 배선 + `1ed9ef3` legacy 알림톡 클라이언트층 → `apps/reviewed-app/app/.../api/domain/adminweb/`, Feign→RestClient, booster internal 2엔드포인트 HTTP 호출 | AC5 reviewed-app 빌드·기동 그린 + 호출층 계약 검증. Thymeleaf/JS 자산은 미변경(후순위 분리) |
| R6 | 6. 게이트웨이 전환(최종) | gateway `Routes.java:35` `legacy-v2` 목적지 `ServiceId.LEGACY` → `ServiceId.BOOSTER` | AC6 RouteTable 테스트 `/api/v2/boost1/...`→BOOSTER 매칭 + 개발계 게이트웨이 경유 v2 호출이 booster-web에 도달 |
| R7 | 횡단: 돈·멱등·차단 3종 동반이관 | ①callbackOrder 비관적잠금 가드(`findByUuidForUpdate`, 최종상태 중복·역순 no-op + send_request 완료처리) ②SHA256 멱등키 `review-boost-auto-<SHA256>`: 생성=crawling, 소비=notification `findByIdempotencyKey`, DB 최후방어=UNIQUE ③수신차단 `ReviewBoostRecipientBlockService`를 send()/sendAuto() **최앞단**(메시지·멱등키 생성 이전), 차단 시 BLOCKED 감사이력만 | AC7 3종 각각 테스트 존재·통과. 차단 대상은 발송 어댑터 호출 0회, `KakaoAlimTalkSendHistory` BLOCKED 1건 |
| R8 | 개발계 구동검증(=이번 완료 정의) | 개발계 스택 DELL `legalcare-local`(pg·gateway·booster-web·booster-worker·stub) | AC8 ①`:app:test` 그린 ②개발계 PG에 `reviewboost-v2-*.sql` 5종 적용 + 백필 완료(재실행 멱등) ③booster-web·worker 컨테이너 healthy, 기동 예외 0 ④v2 4계약 응답: 수동발송·템플릿조회·internal request-setting(PUT)·자동발송 트리거 ⑤`kakao_alimtalk_send_history` 행 생성 확인 ⑥외부 SENS 실호출 0건 |
| R9 | 설정·스텁(R2 착수 전 선행) | `review-boost.common-message-template-id` env 주입, SENS/알림톡 base URL = `stub`(`STUB_BASE`), 발송 기본 OFF 플래그 | AC9 키 미설정 시 부팅실패 재현→설정으로 해소 기록 + 개발계 실발송 차단 근거(스텁 URL·플래그 값) 문서화 |

## 기술 제약
- 적응이식 3건(문자 그대로 복사 금지): ①`NaverBlogHandler` — 신 모듈은 de-Feign이라 `catch(FeignException)`·JSON 정규식이 안 맞음 → `LegalCareException` 기반 재작성(미적응 시 429가 전부 EXCEPTION 기록) ②notification 템플릿류 `@Entity(name="NotificationBoostMessageTemplate")`·`@Repository("notificationBoostMessageTemplateRepository")` 리네임(user 모듈 충돌) ③`review-boost.common-message-template-id`는 기본값 없음 → 미주입 시 부팅실패.
- 모듈 의존은 `notification→product`, `product→crawling`만 존재 → 역방향 직접의존 금지. 인터페이스는 호출 모듈에, 구현은 app 계층 `*LocalAdapter`(@Primary) — `product/client/notification/service/ProductNotificationLocalAdapter.java` 선례 준수.
- booster-app은 Java 전용(kotlin 플러그인 없음) → Kotlin→Java는 클래스·메서드·필드명 및 로직 1:1 보존 변환. `REQUIRES_NEW`는 runCatching 안이 아니라 `TransactionTemplate`으로, jsonb 저장 전 NUL 제거(선행 리뷰 교훈 승계).
- 작업 위치는 worktree `/Users/steve/steve/legalcare-renew-prodsync-wt`(branch `feature/prod-sync-candidates`)에 한정, push·머지 금지. 금액은 BigDecimal, 로그에 수신번호·개인정보 원문 금지(마스킹).

## 가정
- A1. 개발계 PG에 선행 DDL 5종을 적용할 권한과 상태가 확보돼 있다(선행 태스크에서 psql 실적용 검증 완료).
- A2. WAIT 상태 행을 소비하는 자동발송 배치는 신 legalcare-batch에 이미 존재(사용자 확인) — 배치 동작 검증은 이번 범위 밖.
- A3. adminweb 화면 자산 없이도 서버측 호출층 검증만으로 개발계 "동작 확인"이 성립한다.

## 리스크
RISK: HIGH — 알림톡 발송(비용)·주문 콜백 원장(돈)·신규 DDL 의존·5서비스→1앱 대형 이관.
- 1순위 개발계 실발송 사고 → R9(스텁·OFF)를 R2 착수 **전에** 완료. 2순위 callbackOrder 가드 누락 이중기록 → R1 최우선. 3순위 Kotlin→Java 변환 중 로직 드리프트 → 단계별 셀프리뷰에 구/신 파일 대조 기록 필수.

## 관련 파일
| 경로 | 설명 |
|---|---|
| `/Users/steve/steve/mocking-box/review/20260808-2258-prod-sync-followups/03-design-D.md` | 파일단위 매핑·모듈배치·7단계 의존순서 원본(승계). 개발자 필독 |
| `/private/tmp/claude-501/-Users-steve-steve-mocking-box/8f23c7f7-7c11-47e6-b28c-5c139f6de797/scratchpad/synthesis.md` | 구 커밋해시·경로·이슈 총람(재조사 금지) |
| `/Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app/modules/{product,notification,crawling}/` | 목적지 3모듈. product에 구버전 `service/ReviewBoostService.java`(callbackOrder 가드 없음), notification에 v1 `ReviewBoost*` 5서비스 기존재 → 병합 대상 |
| `/Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app/app/src/main/java/legalcare/medilawyer/product/client/notification/service/ProductNotificationLocalAdapter.java` | 역방향 접점 어댑터 선례(app 계층 @Primary 구현) |
| `/Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app/app/src/main/resources/db/reviewboost-v2-{product,alimtalk,order-alter,backfill}.sql`, `crawling-fail-log.sql` | 선행 DDL·백필(691046c). 개발계 적용 대상, 적용순서 DDL→백필→R1 |
| `/Users/steve/steve/legalcare-renew-prodsync-wt/apps/gateway-app/src/main/java/legalcare/medilawyer/gateway/route/Routes.java` (35행 `legacy-v2`) | R6 전환지점(현재 LEGACY) |
| `/Users/steve/steve/legalcare-renew-prodsync-wt/docker-compose.yml` (stub:80, gateway:86, booster-web:103, booster-worker:143) | 개발계 스택 구성·스텁 주입 지점(`STUB_BASE`, `TRANSITION_LEGACY_URL`) |

## 의도 확인
재진술: "새로 들어간 알림톡 v2 PR을 그대로 통합앱에 옮기고, 개발계에서 도는 것까지 확인해라(싱글 진행)." 이 지시서는 승계 계획의 7단계를 R1~R6 실행순서로 고정하고, 사용자가 "이미 되어 있다"고 한 배치·DDL·라우트는 범위에서 빼 순수 이관 작업만 남겼다. "돈다"의 증거는 AC8 6항목으로 못박았고, 사고 방지를 위해 실발송은 R9 스텁/OFF로 봉인해 계약 검증까지만 수행한다.

## OPEN QUESTIONS
없음
