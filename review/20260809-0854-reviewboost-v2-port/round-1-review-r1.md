# Round 1 리뷰 — R9 발송 봉인 + R1 product v2 스택 (검토자: Claude, 수정 권한 없음)

문서를 믿지 않고 `git diff HEAD` / `git status`(82파일, 스테이징됨) 로 실물을 봤고, 구 `product-service origin/main` 과 파일 단위로 기계 대조했으며, 빌드는 작성자 보고를 무시하고 직접 다시 돌렸다.

## 먼저 검증된 것 (이건 진짜다)

**빌드.** `./gradlew :app:test` 는 UP-TO-DATE 로 스킵돼서 아무것도 안 돈다 — `cleanTest test` 로 강제 재실행했다. `BUILD SUCCESSFUL in 3m 49s`, JUnit XML 155개 집계 결과 **443 tests / fail 0 / err 0 / skip 0** (app 19·core 44·chrono 7·crawling 68·external 40·medicontents 18·notification 23·post 67·product 71·user 86). 04-changes 의 숫자와 모듈별 분포까지 정확히 일치한다. 참고로 콜백 가드 테스트는 `modules/product` 소스셋이라 지시서의 게이트인 `:app:test` 만으로는 **안 돈다** — 작성자가 전체 스위트로 돌린 판단이 맞았다.

**R7① callbackOrder 가드 — 요구한 성질을 실제로 만족한다.** `ReviewBoostService.java:156` `@Transactional`, `:165` `findByUuidForUpdate`(`BoostPresentOrderRepository.java:23` `@Lock(PESSIMISTIC_WRITE)`), `:167` 최종상태 즉시 return, `:184` `existsByOrderId` 2차 방어, `:196/:198` send_request 종결. 순서가 **잠금 → 검사 → 갱신** 으로 맞다. 중요한 건 이 트랜잭션에서 `findByUuidForUpdate` 가 **첫 DB 접근**이라는 점이다 — 앞에 `log.info` 밖에 없어서 영속성 컨텍스트가 비어 있고, 따라서 PG READ COMMITTED 에서 `FOR UPDATE` 해제 후 최신 커밋본을 읽는다(1차 캐시 stale 함정에 안 걸린다). TX2 는 TX1 커밋 후 `SUCCESS` 를 보고 return 하므로 원장 이중기록은 구조적으로 불가능하다. 원장·`increaseSendingCount` 의 쓰기 지점이 앱 전체에서 `:184~:188` 하나뿐인 것도 grep 으로 확인했다. 그리고 이 메서드는 구 원본과 **한 글자도 다르지 않다**.

**이관 충실도 — 예상보다 훨씬 좋다.** 주석/import/package 를 제외한 로직 기준으로 product 모듈 변경 파일 60여 개 중 **차이 0** 이 대부분이었다. 0 이 아닌 것도 전부 설명이 붙는 통합앱 적응분이다: `BoostPresentOrderRepository.java:41,44,47,67,70,73` FQN enum 리매핑, `ProductService`→`@Service("productDomainService")`, `InternalUserController`→`productInternalUserController`, `PostService` Feign `contextId`, `ReviewBoostScheduler` `@Profile("worker")`+`@SchedulerLock`. 구 `core.code`/`core.exception` 에서 `product.core.*` 로 옮긴 7개 파일도 구 원본 대비 차이 0. `LegacyHandler`/`LegacyService` 도 빈 이름·contextId 한 줄만 다르다. **구에는 있는데 신에 없는 파일 0건**, 그리고 손대지 않은 파일 중 v2 로직 누락 후보도 없었다(`ReviewPresentService` 포함 — 상류에서 안 바뀐 게 맞다). MyBatis 매퍼 3종 삭제도 정당하다: 구 저장소 `origin/main` 에 `BoostPresentOrderRepositoryCustom/Impl/QueryMapper` 가 **전부 없다**(상류가 지웠다). 검수 API 잠금(`BoostReviewMatchAnalysisRepository` `findByIdForUpdate`/`findByMessageIdForUpdateOrderByIdAsc`)·`StorePresentValidator`·상태 전이 모두 원본 그대로다.

**고아 변경 0.** 82파일이 전부 product 모듈 / app 설정 3종 / 어댑터 2종 / 부팅게이트 테스트 / compose / stub 에 들어간다. `mldelegator`·gateway `Routes.java`·notification·crawling 모듈 전부 무접촉. 모듈 의존도 깨끗하다 — product 소스가 참조하는 타 모듈은 `crawling` 1건뿐이고 역방향 없음. `productLegacyHandler`/`productLegacyService` 빈 이름 처리도 기존 선례와 같은 방식이라 적절하다.

**로그 PII.** 새로 추가된 log 문 10개를 다 봤는데 수신번호·고객명 원문이 들어간 자리가 없다. `receiver`/`customerId` 는 엔티티 필드로만 흐른다. 04-changes 가 R2 용으로 남긴 `phone=$phone` 경고도 정확한 지적이다.

이하 지적. 심각도 순.

---

## 1. [CRITICAL] 개발계 선물발송이 **카카오 운영 giftbiz 로 실호출**되고, 콜백은 **운영 게이트웨이**로 돌아간다 — 스텁으로 돌릴 방법이 아예 없다

`app/src/main/resources/config/product.yml:29-30`

```yaml
kakao-biz:
  api:
    url: https://gateway-giftbiz.kakao.com/openapi/giftbiz          # ← ${} 없음
    present-call-back-url: https://gateway.medilawyer.co.kr/product/apis/review-boost/unrestricted/callback/order
```

R9 는 SENS 3경로를 `${STUB_BASE}` 로 env 화해 잘 봉인했는데, **정작 R1 이 이관한 코드가 실제로 돈을 쓰는 경로는 이쪽**이다. `KakaoBizApiRestClient.java:84` 가 `${kakao-biz.api.url}` 을, `KakaoBizPresentTemplate.java:35` 가 `${kakao-biz.api.present-call-back-url}` 을 읽는데 둘 다 **env 인디렉션이 없어서 compose 로 덮을 수가 없다**. `docker-compose.yml` booster-web/worker 환경에 `KAKAO_BIZ_*` 가 없는 것도 확인했다(있는 건 batch 서비스 블록의 `docker-compose.yml:262-263` `KAKAO_BIZ_URL: ${STUB_BASE}/giftbiz` 뿐 — 배치 앱은 이미 env 로 풀어놨다. booster 만 안 됐다).

결과적으로 R8 에서 `POST /apis/review-boost/send/present` 를 한 번이라도 때리면 개발계에서 카카오 운영 API 로 실제 HTTPS 요청이 나간다. 지금 당장 과금이 안 되는 유일한 이유는 `kakao-biz.authorization` 이 `dev-placeholder-kakao-biz-auth`(product.yml:27) 로 남아 401 을 맞기 때문이고, 이건 **설정으로 막은 게 아니라 값을 안 채워서 우연히 막힌 상태**다. 누가 R8 에서 "발송 테스트가 401 나네" 하고 `KAKAO_BIZ_AUTHORIZATION` 에 실 토큰을 넣는 순간:

1. 실제 선물이 구매된다(진짜 돈).
2. 페이로드의 `success_callback_url`/`fail_callback_url`(`KakaoBizPresentTemplate.java:50-51`)이 **운영 게이트웨이**라, 카카오가 **운영 booster 로 콜백**한다 → 개발계에서 만든 주문의 결과가 **운영 `boost_present_order`/`boost_present_ledger` 에 기록되거나, 운영에 없는 uuid 라 `NotFoundException` 으로 조용히 유실된다. 어느 쪽이든 원장 오염이다.

`stubs/stub_server.py:15` 에 `/giftbiz` canned 응답이 **이미 있다**는 게 정황상 결정적이다 — 스텁으로 보낼 의도는 있었는데 URL 이 하드코딩이라 아무것도 그리로 안 간다.

요청: `kakao-biz.api.url` / `present-call-back-url` 을 `${KAKAO_BIZ_URL:...}` / `${KAKAO_BIZ_PRESENT_CALLBACK_URL:...}` 로 env 화하고, `docker-compose.yml` booster-web/worker 에 `${STUB_BASE}/giftbiz` 와 개발계 자기 게이트웨이 주소를 주입. **R8 착수 전에 닫아야 한다.**

## 2. [MAJOR] 정작 `dev` 프로파일로 뜨는 스택(`deploy/dev`)은 R9 봉인을 하나도 못 받았고, 그 프로파일엔 실 NCP 키가 박혀 있다

`deploy/dev/docker-compose.yml:46,69` / `app/src/main/resources/application-dev.yml:180-181`

`application-dev.yml:179` 의 기본값을 실 SENS 에서 도달불가 주소로 내린 건 정확히 옳은 수정이다. 그런데 그 프로파일을 실제로 켜는 스택은 root `docker-compose.yml`(`local,web`)이 아니라 `deploy/dev/docker-compose.yml:46`(`SPRING_PROFILES_ACTIVE: dev,web`)이고, 이쪽은 `NAVER_CLOUD_URL: ${DEV_NAVER_CLOUD_URL:-}` (`:69`) 한 줄만 있고 `NAVER_CLOUD_SMS_URL`·`NAVER_CLOUD_SES_BASE_URL`·`REVIEW_BOOST_SEND_ENABLED`·`REVIEW_BOOST_COMMON_MESSAGE_TEMPLATE_ID` 를 **하나도 주입하지 않는다**. root compose 가 받은 봉인 5줄이 여기엔 없다.

그리고 같은 프로파일 파일에 실 자격증명이 그대로다:

```yaml
application-dev.yml:180  naver-cloud-accessKey: ${NAVER_CLOUD_ACCESS_KEY:Cnky7OVxbJoMHBYlpM0q}
application-dev.yml:181  naver-cloud-secretKey: ${NAVER_CLOUD_SECRET_KEY:9tqCv...}   # 40자 실 시크릿 평문
```

작성자 본인이 `:173` 에 "이 블록의 자격증명은 실제 NCP SENS 키다" 라고 적어놨다. 즉 **URL 한 칸(`DEV_NAVER_CLOUD_URL`)만 실 SENS 로 채우면 서명이 유효해서 진짜로 접수·과금된다.** 봉인이 URL 단일 축에만 걸려 있고, 그 축이 `deploy/dev` 에선 빈 문자열로 방치돼 있다.

지시서가 정의한 "개발계"는 root compose 스택이니 AC9 자체를 못 지켰다고 하진 않겠다. 다만 04-changes 의 "개발계 실발송 차단 근거" 서술은 root compose 에만 해당한다는 단서가 필요하고, `deploy/dev` 에도 같은 5줄을 넣는 게 R9 의 원래 취지에 맞다. 별건으로 이 두 키는 **회수·로테이션 대상**이다(fintech-safety §5.4 — 자격증명은 Secrets Manager, 하드코딩 금지). 이번 diff 가 만든 문제는 아니지만, R9 가 손댄 바로 그 블록이고 diff 가 이제 "여기 진짜 키 있음" 이정표까지 달아놨다.

## 3. [MAJOR] `dev-placeholder` 기본값이 기술제약 ③ 이 안전장치로 지목한 **부팅 실패(fail-fast)를 없앴다** — 그리고 인수인계 목록에 1개만 적혀 있다

`config/notification.yml:55,56,58,70`

기술제약 ③ 은 "`review-boost.common-message-template-id` 는 기본값 없음 → 미주입 시 부팅실패"를 **성질로** 명시했다. AC9 도 그 실패를 재현하라고 했고 작성자는 실제로 재현했다(좋다). 그런데 최종 형태가 `${REVIEW_BOOST_COMMON_MESSAGE_TEMPLATE_ID:dev-placeholder-common-message-template-id}` 라서, 이제 **env 를 안 넣어도 앱이 멀쩡히 뜬다**. `application-release.yml` 에 이 키들 재정의가 **하나도 없는 것**도 확인했다 — 즉 운영에서 env 를 빠뜨리면 배포 시점에 시끄럽게 죽는 대신, `bizMessageServiceId=dev-placeholder-...` 로 조용히 떠서 R2 이후 v2 알림톡이 전량 벤더 거절로 실패한다. 돈 경로에서는 "안 뜨는 것"이 "잘못된 값으로 뜨는 것"보다 안전하다.

저장소 전체 관례(`notification.yml:14-20` 헤더의 PORTING-GUIDE rule #3)가 `dev-placeholder-*` 인 건 알겠고 그 일관성 자체는 존중한다. 그래서 반박 여지가 있는 지적이다. 다만 최소한 둘 중 하나는 필요하다 — (a) release 프로파일에서 placeholder 값이면 부팅 거부하는 `ApplicationRunner`/`@Profile("release")` 검증 빈, 또는 (b) 04-changes 의 인수인계 항목 확장.

지금 04-changes "리뷰어에게" 에는 `REVIEW_BOOST_SEND_ENABLED` **하나만** 승격 필수로 적혀 있는데, 같은 처지의 키가 5개 더 있다: `REVIEW_BOOST_COMMON_MESSAGE_TEMPLATE_ID`, `NAVER_CLOUD_SES_BASE_URL`, `NAVER_CLOUD_BIZ_MESSAGE_SERVICE_ID`, `NAVER_CLOUD_SECRET_KEY`, `REVIEW_BOOST_BASE_URL`. 특히 `reviewBoostBaseUrl`(`:58`)은 알림톡 본문에 심는 리뷰 페이지 링크라, 놓치면 운영 고객에게 `http://localhost:19999/...` 링크가 나간다. 인수인계 문서에 6개 전부 올려달라.

## 4. [MAJOR] 새 부팅 게이트가 `local` 프로파일만 검증한다 — 정작 버그가 있던 `dev` 프로파일은 안 탄다

`app/src/test/java/legalcare/medilawyer/BoosterApplicationContextTest.java:28` / `:71-74`

04-changes 는 "위 3가지를 `BoosterApplicationContextTest` 가 매 빌드 검증한다"고 적었는데, 이 테스트는 `@SpringBootTest(properties = "spring.profiles.active=local")` 이라 `application-local.yml` + `config/*.yml` 만 탄다. `application.yml:35` 의 주석대로 프로파일 그룹에 환경 축이 없어서 `local` 이 `dev` 를 끌어오지도 않는다. 즉 `doesNotContain("ntruss.com")` 어서션(`:71-74`)은 **R9 가 실제로 고친 그 줄(`application-dev.yml:179`)을 검사하지 않는다.** 누가 내일 dev 기본값을 실 SENS 로 되돌려도 이 게이트는 초록이다.

검사 범위도 좁다 — `naverCloud.sesBaseUrl` 과 `naver-cloud.url` 만 보고 `naver-cloud.sms.sms-url` 은 안 본다. 프로파일별로 같은 어서션을 도는 파라미터라이즈드 테스트거나, 아니면 yml 을 텍스트로 스캔하는 린트가 이 목적엔 더 맞다(`smoke/env_lint.sh` 가 이미 있는데 스캔 대상이 `.env`/root compose 뿐이라 `apps/**/application-*.yml` 을 안 본다).

## 5. [MINOR] 배치 앱에는 같은 `ntruss` 기본값이 그대로 남아 있다

`apps/legalcare-batch/src/main/resources/application.yml:58` (`ses-base-url: ${NCP_SES_BASE_URL:https://sens.apigw.ntruss.com}`), `apps/legalcare-batch/.../batch/job/review/NcpAlimtalkClient.java:57` (같은 문자열이 `@Value` 기본값으로 한 번 더).

`application-dev.yml:179` 에서 없앤 것과 **정확히 같은 패턴**이고, 하필 A2 가 "이미 존재한다"고 한 WAIT 소비 배치다. 지금은 `application.yml:39` `app.notify.enabled: false`(하드코딩) + `app.jobs.alimtalk.enabled` 게이트 + compose 3중 주입으로 막혀 있어서 즉시 위험은 아니다. 그래도 R9 를 한 번 도는 김에 같이 내리는 게 맞다.

## 6. [MINOR] `updatePresentOrder` 의 무잠금 read-modify-write 가 콜백과 경합한다 (원본 승계)

`service/handler/PresentTemplateHandler.java:68-69`

```java
BoostPresentOrderEntity orderEntity = boostPresentOrderRepository.findById(...)   // 잠금 없음
boostPresentOrderRepository.save(...toEntity(orderEntity, SEND_FAIL));
```

수동 발송에서 벤더가 이미 접수했는데 우리 쪽이 read timeout 으로 실패 처리하는 창이 있다. 그 사이 카카오 콜백이 먼저 도착하면 `callbackOrder` 가 SUCCESS + 원장 1건을 쓰고 커밋하는데, 뒤이어 이 무잠금 save 가 `SEND_FAIL` 로 덮는다 → **원장에는 과금 1건, 주문 상태는 SEND_FAIL**. 원장 이중기록은 아니라서 R7① 의 AC 는 지키지만 정합성 구멍은 맞다.

구 원본과 로직 차이 0 이라 **이번 이관이 만든 결함은 아니다.** R1 을 막을 사안은 아니고, `findByUuidForUpdate` 로 바꾸는 후속 티켓으로 남기자.

## 7. [MINOR] `getAnalysisForReview` 가 잠그기 전에 먼저 조회한다 (원본 승계)

`service/ReviewBoostService.java:396-405`

`:397` 에서 잠금 없이 `findById(id)` 로 엔티티를 1차 캐시에 올린 뒤 `:400`/`:404` 에서 `FOR UPDATE` 로 다시 읽는다. Hibernate 는 이미 세션에 있는 엔티티를 재조회해도 필드 상태를 갱신하지 않으므로, 잠금 획득 후 `:375` 에서 보는 `analysis.getMatchStatus()` 가 **잠금 이전 값**일 수 있다. 동시 검수 두 건이 둘 다 `NEED_HUMAN` 통과 가능. 4번 항목의 `callbackOrder` 가 이 함정을 안 밟은 것과 대조된다(거기선 잠금 조회가 첫 접근).

역시 원본과 차이 0 이라 이관 결함은 아니다. 다만 지시서가 "검수 API 잠금(PESSIMISTIC_WRITE)"을 충실도 점검 항목으로 찍었으니, **잠금은 이관됐지만 그 잠금이 보장하는 범위가 겉보기보다 좁다**는 사실은 기록해두는 게 낫다.

## 8. [MINOR] 콜백 테스트가 가드의 절반만 덮는다

`modules/product/src/test/.../ReviewBoostServiceCallbackOrderTest.java:63`

3케이스(중복·실패·역순) 모두 실제로 유효한 검증이고, 특히 역순 케이스가 `ledger.save` 0회 + `history.save` 0회 + `findByOrderId` 0회 + `publishEvent` 0회까지 잡는 건 형식적이지 않다. 다만 두 가지가 비어 있다.

첫째, `existsByOrderId` 는 `false` 로만 스텁돼 있어서(`:63`) **2차 방어선이 참일 때의 경로가 어느 테스트에도 없다.** 중복 콜백 케이스는 1차 가드에서 return 하므로 `existsByOrderId` 를 두 번째로 호출조차 안 한다. 원장 가드가 실제로 막는지 보려면 `existsByOrderId=true` + `order.result=WAIT` 조합이 필요하다.

둘째, 순수 Mockito 라 `@Lock(PESSIMISTIC_WRITE)` 가 걸리는지, 동시 콜백이 실제로 직렬화되는지는 **아무것도 검증하지 않는다.** "원장 이중기록 0건" 은 지금 로직 수준 주장이다. 이미 `crawling` 등에 Testcontainers 판이 있으니, 같은 주문 uuid 로 두 스레드가 `callbackOrder` 를 때리는 통합 테스트 1개면 이 주장이 실측이 된다. R7① 이 이번 작업 최우선 항목이었던 만큼 그만한 값어치는 있다.

## 9. [MINOR] `naver-cloud.url` 이 두 파일에 서로 다른 기본값으로 이중 정의됐다

`config/notification.yml:38` (`dev-placeholder-naver-cloud-url`) vs `application-dev.yml:179` (`http://localhost:19999/alimtalk`). 둘 다 같은 env 를 먼저 보므로 실동작 차이는 없고 둘 다 도달불가라 안전하지만, 한쪽만 고치면 프로파일에 따라 다르게 도는 전형적인 유지보수 함정이다. 어느 쪽이 정본인지 주석 한 줄이라도 있으면 좋겠다.

## 10. [MINOR] 원장 금액 scale 이 KRW 규칙과 다르다 (스키마 승계)

`persistence/repository/entity/BoostPresentLedgerEntity.java:42` — `precision = 13, scale = 4`. 테스트도 `new BigDecimal("4500.0000")` 를 쓴다. BigDecimal 인 건 맞지만 프로젝트 금융 규칙의 "KRW 는 scale 0" 과 어긋난다. 구 원본·DDL 승계라 이번에 바꿀 일은 아니고 바꿔서도 안 된다. 다만 R2 이후 금액 비교/합산이 들어올 때 `equals` 가 scale 을 보는 함정(`4500` vs `4500.0000`)이 있으니, 비교는 `compareTo` 로 한다는 원칙만 어딘가에 못박아두자.

---

## 정리

R7① 돈버그 가드는 요구한 성질을 실제로 만족하고, 이관 충실도는 기계 대조 기준으로 사실상 완벽하다(로직 차이 0 이 대부분, 누락 파일 0, 고아 변경 0). 테스트 443개 그린도 직접 재확인했다. 이 부분은 그대로 가면 된다.

막는 건 **1번** 하나다. R9 가 SENS 를 잘 봉인해놓고 정작 R1 이 이관한 선물발송이 카카오 운영 API 로 나가고 콜백이 운영 게이트웨이로 돌아가는 상태에서 R8(개발계 구동검증)에 들어가는 건 지시서 리스크 1순위 "개발계 실발송 사고" 를 그대로 밟는 그림이다. 스텁에 `/giftbiz` 응답이 이미 준비돼 있으니 env 화 2줄 + compose 2줄이면 닫힌다.

2·3·4 는 R2 착수 전까지, 5~10 은 후속으로 처리하면 된다. 6·7 은 원본 승계라 **이번 라운드에서 코드를 고치라는 요구가 아니다** — 후속 티켓으로 남겨달라는 뜻이다.

VERDICT: REQUEST_CHANGES
