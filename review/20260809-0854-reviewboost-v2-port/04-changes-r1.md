# 변경 요약: R9 발송 봉인 + R1 product v2 스택 + callbackOrder 돈버그 가드

82파일 (+3,613 / −466). 구 `product-service` 548019c→origin/main 차분을 패키지 리매핑해 적용했다(수작업 재작성 아님).

## 변경 파일
| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `app/src/main/resources/config/notification.yml` | `review-boost.common-message-template-id` + `send.enabled`(기본 false) + `naverCloud.{sesBaseUrl,bizMessageServiceId,naverCloudSecretKey,reviewBoostBaseUrl}` env화 | R9 |
| `app/src/main/resources/application-dev.yml` | `naver-cloud.url` 기본값을 실 SENS → 도달불가 주소로 하향 | R9 |
| `docker-compose.yml` | booster-web/worker 에 발신 3경로 `${STUB_BASE}` 고정 + 발송 OFF | R9 |
| `stubs/stub_server.py` | `/alimtalk` canned 에 `messages[].requestStatusCode=A000` 추가(v2 수락판정 대응) | R9 |
| `app/src/test/.../BoosterApplicationContextTest.java` | 5개 키 해석 + 발송 OFF + URL 비-ntruss 게이트 3종 추가 | R9 |
| `modules/product/**` (신규 27·수정 22·삭제 3) | v2 엔티티4·repo4·서비스4(`BoostPresentOrderTransactionService`/`BoostStorePresentSettingService`/`SubscribeProduct{,Registration}Service`)·검수API·`StorePresentValidator`·`ReviewBoostService` v2 최종상태 | R1 |
| `modules/product/.../product/client/legacy/**` | 구독 시 공통 알림톡 설정 초기화 Feign(신규) | R1 |
| `app/src/main/resources/config/product.yml` | 위 Feign 의 `productLegacyService` transition url | R1 |
| `app/.../ProductPostLocalAdapter.java` | `getMessageAiPosts` 구현(검수 통합조회가 원문 조인에 사용) | R1 |
| `app/.../UserProductLocalAdapter.java` | `getSubscriptionList` 소유자 이동(`ProductService`→`SubscribeProductService`) 반영 | R1 |
| `modules/product/src/test/**` (신규 15) | 구 v2 테스트 전량 동반 이식 + 역순 콜백 케이스 1건 추가 | R1 |

## 테스트 결과
`cd apps/booster-app && ./gradlew cleanTest test` → **BUILD SUCCESSFUL (3m 43s), 459 tests / 0 fail / 0 err / 0 skipped**
(app 35, core 44, chrono 7, crawling 68, external 40, medicontents 18, notification 23, post 67, product 71, user 86)
`cleanTest` 로 전량 강제 재실행했다 — `:app:test` 단독은 UP-TO-DATE 로 스킵될 수 있고, 콜백 가드 테스트는 `modules/product` 소스셋이라 `:app:test` 만으로는 애초에 돌지 않는다(라운드1 지적).

중간에 실제로 잡힌 결함 2건(둘 다 수정 후 그린): ① `:app:test` 부팅 게이트가 `ConflictingBeanDefinitionException: 'legacyHandler'` 로 실패 — crawling 의 동명 `LegacyHandler` 와 충돌, product 쪽에 `@Component("productLegacyHandler")` 부여로 해소. ② `UserProductLocalAdapter` 컴파일 실패 — 구독 로직이 `SubscribeProductService` 로 분리되며 호출부가 깨짐.

## AC 자가 점검
- **AC1** ✅ `:app:test`·전체 스위트 그린. `ReviewBoostServiceCallbackOrderTest` 동반 이식 후 3케이스 통과(중복 콜백 시 `ledger.save` 1회·`increaseSendingCount` 1회 = 원장 이중기록 0건 / 실패 콜백 / 역순 콜백 no-op).
- **AC9** ✅ 부팅실패 재현: 기본값 없는 `@Value` 프로브로 `Could not resolve placeholder 'review-boost.common-message-template-id...'` 실측(프로브는 제거). 해소 근거는 아래.

**callbackOrder 가드 위치**: `modules/product/.../product/service/ReviewBoostService.java:164-171` — `findByUuidForUpdate`(`BoostPresentOrderRepository.java:24` `@Lock(PESSIMISTIC_WRITE)`) 로 잠근 뒤 `order.getResult() != WAIT` 이면 즉시 return. 원장은 `existsByOrderId` 로 2차 방어(:184), send_request 완료처리는 :196.

## 개발계에서 돈 나가는 외부 호출 경로 — 전수 목록과 봉인 상태

booster-app 설정의 외부 주소·자격증명을 전수 조사(`config/*.yml` + `application-{local,dev,release}.yml`)한 결과다.
"봉인" = ①비운영 프로파일 기본값이 실 provider 가 아님 ②로컬·개발계 compose 가 스텁/도달불가로 명시 주입 ③운영은 env 미주입 시 부팅 실패.

| # | 경로 | 비용 성격 | 로컬 스택 | 개발계 스택 | 운영 fail-fast | 게이트 |
|---|---|---|---|---|---|---|
| 1 | NCP SENS 알림톡 `naver-cloud.url` | 건당 과금 | `stub:8099/alimtalk` | `:19999/alimtalk`(fail-closed) | ✅ | ✅ |
| 2 | NCP SENS SMS/OTP `naver-cloud.sms.sms-url` | 건당 과금 | `stub:8099/sms` | `:19999/sms` | ✅ | ✅ |
| 3 | NCP SENS v2 리뷰부스터 `naverCloud.sesBaseUrl` | 건당 과금(R2 이관 예정) | `stub:8099` | `:19999` | ✅ | ✅ |
| 4 | **카카오 선물하기 비즈 `kakao-biz.api.url`** | **기프티콘 실구매** | `stub:8099/giftbiz` | `:19999/giftbiz` | ✅ | ✅ |
| 5 | **카카오 선물 콜백 `kakao-biz.api.present-call-back-url`** | **운영 원장 오염** | booster 자신 | booster 자신(:18081) | ✅ | ✅ |
| 6 | NCP 자격증명 `naver-cloud.*Key`, `naverCloud.*Key` | 위 1~3 의 서명 성립 조건 | placeholder | placeholder | ✅ | — |
| 7 | 카카오 선물 인증 `kakao-biz.authorization` | 위 4 의 성립 조건 | `stub` | `stub-dev-not-a-real-token` | ✅ | — |
| 8 | v2 발송 게이트 `review-boost.send.enabled` | 논리 차단(2중 잠금) | false | false | 기본 false 유지 | ✅ |
| 9 | 알림톡 본문 링크 `naverCloud.reviewBoostBaseUrl` | 발신 아님(본문 URL) | 비운영 | 비운영 | ✅ | ✅ |
| 10 | AWS SES 메일 `aws.client.*` | 건당 과금(소액) | placeholder 키 | placeholder 키 | ❌ 미적용 | ❌ |
| 11 | Slack webhook·bot 6종 | 무과금, 운영 채널 오염 | placeholder | dev 전용 webhook | ❌ 미적용 | ❌ |
| 12 | Google Gemini 2종 / YouTube / 네이버 검색 / **네이버 지도 `maps.apigw.ntruss.com`**(`external.yml:54`) / 공공데이터 / Airtable / 카카오REST | 토큰·쿼터 과금 | placeholder 키 | placeholder 키 | ❌ 미적용 | ❌ |
| 13 | legalcare-batch 의 SENS·카카오선물 | 과금 | `${STUB_BASE}` | 스택에 batch 없음 | ❌ 미적용 | ❌ |

**10~13 미적용 사유(의도적)**: 이번 작업(R1·R9)이 건드린 경로가 아니다. `dev-placeholder` 기본값 키는 총 30개인데, 전부를 한 번에 운영 fail-fast 로 바꾸면 이번 범위 밖의 운영 부팅을 깨뜨린다. 10~12 는 자격증명이 placeholder 라 호출이 인증 단계에서 실패하고(1~5 와 달리 "토큰만 채우면 나가는" 구조가 아님), 13 은 로컬에서 이미 스텁 고정 + 잡 비활성 + 개발계 미배포다. **후속 태스크 권장**: 30개 전수에 대해 "운영 fail-fast / 비운영 스텁 기본값" 방침을 일괄 적용.

**R9 실발송 차단 근거**: 두 스택 모두 `docker compose --env-file .env.example config` 로 실제 렌더값을 확인했다.
- 로컬(`docker-compose.yml`): `NAVER_CLOUD_URL=http://stub:8099/alimtalk`, `NAVER_CLOUD_SMS_URL=http://stub:8099/sms`, `NAVER_CLOUD_SES_BASE_URL=http://stub:8099`, `KAKAO_BIZ_URL=http://stub:8099/giftbiz`, `KAKAO_BIZ_PRESENT_CALLBACK_URL=http://booster-web:18080/product/...`, `REVIEW_BOOST_SEND_ENABLED=false`
- 개발계(`deploy/dev/docker-compose.yml`): 같은 6개가 `127.0.0.1:19999`(리스너 없음 = fail-closed)와 자기 자신(:18081)으로. 개발계는 스텁 컨테이너가 없어 "조용한 성공"보다 "즉시 connection refused"가 안전하다고 판단했다.
- 프로파일 기본값에 남아 있던 실 provider 주소 3건 제거(dev 의 실 SENS 주소, product.yml 의 카카오 선물 API·운영 콜백).
- 위 전부를 `ExternalSendSealingTest`(16케이스, local/dev/release)가 매 빌드 검증. 하드코딩으로 되돌리면 실제로 FAIL 하는 것까지 뮤테이션으로 확인했다.

## 알려진 한계 / 리뷰어에게
- **⚠️ 배포 전 필수 확인 (라운드1 #3 반영으로 신설된 위험)**: `application-release.yml` 이 발송·결제 키 13개를 기본값 없는 `${VAR}` 로 재정의했다. **운영 태스크 정의/Secrets Manager 에 이 13개가 없으면 release 부팅이 실패한다**(의도된 fail-fast). 뒤집어 말하면 지금까지는 없어도 조용히 떴다는 뜻이라, 운영에서 알림톡이 placeholder 주소로 나가고 있었는지 현황 확인을 권한다.
- **⚠️ 키 유출**: `application-dev.yml` 에서 제거한 NCP access/secret 키는 git 이력에 그대로 남는다. 이 커밋으로 무효화되지 않으므로 NCP 콘솔에서 **폐기·재발급**이 필요하다.
- **`review-boost.send.enabled` 는 아직 읽는 코드가 없다.** R2 발송 어댑터가 소비할 자리다(그래서 지금의 실효 차단은 URL 쪽). **R4 발송 ON 시 release 에서 `REVIEW_BOOST_SEND_ENABLED=true` 승격이 필요하다 — 안 하면 운영이 조용히 발송 정지된다.** 인수인계 필수 항목.
- **개발계 스택은 스텁 컨테이너가 없다.** 봉인은 fail-closed(리스너 없는 포트)로 걸었으므로, AC8 의 v2 계약 검증(스텁 응답 필요)은 스텁을 포함한 로컬 스택에서 하거나 DELL 에 스텁을 띄우고 `DEV_STUB_BASE` 를 채워야 한다. R2/R8 착수 전 결정 필요.
- **갈림길 1 — 구 `LegacyService` 를 LocalAdapter 로 안 바꾸고 Feign 유지**: 목적지(medilawyer-boot v2 요청설정)가 R2 에서 notification 모듈로 들어온다. 지금 어댑터로 만들면 R2 에서 다시 뜯어야 해 Feign(+transition url=stub) 으로 뒀다. R2 에서 `ProductNotificationLocalAdapter` 로 흡수하는 게 최종형.
- **갈림길 2 — MyBatis `BoostPresentOrderQueryMapper` 3종 + 테스트 삭제**: 운영 82a6fcd 가 이 조회를 JPQL 프로젝션(`BoostPresentOrderSentListProjection`)으로 대체하며 구 jOOQ 커스텀 리포지토리를 지웠다. 신 앱의 MyBatis 판은 그 jOOQ 의 후신이라 남기면 이중 구현이 된다. 매퍼 테스트 1개가 같이 사라진 대신 JPQL 경로는 `ReviewBoostServiceSentPresentListTest` 가 덮는다.
- **함정 하나 실측 기록**: `@Query` 안의 FQN enum(`...core.code.BoostPresentOrderAttributeKey.CHANNEL`)도 리매핑 대상이다. 컴파일이 안 잡아주고 런타임 쿼리 파싱에서 터지는 자리라 `BoostPresentOrderRepository.java:41,44,47,67,70,73` 을 눈으로 확인했다.
- **R2 로 넘길 보안 지적(지금 범위 아님)**: 구 `ReviewBoostV2BizMessageService.kt` 는 `log.info`/`log.error` 에 `phone=$phone` 원문을 찍는다. 이관 시 마스킹 필수 — 문자 그대로 옮기면 개인정보 로깅 위반이 된다.
- 엔티티↔DDL 대조는 컬럼명·길이·UNIQUE 제약까지 확인했다(`uk_bpsr_post`, `uk_brpc_message_platform`, `boost_present_order.source/created_by`). DDL 은 손대지 않았다.
- 변경 82파일 전부 R1/R9 에 매핑된다(고아 변경 0).

STATUS: DONE
