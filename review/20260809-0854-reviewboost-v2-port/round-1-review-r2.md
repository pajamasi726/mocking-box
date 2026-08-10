# 라운드 1 교차검토 — R2(발송본체) · R3(조회측) · R7②③

검토 대상은 worktree `/Users/steve/steve/legalcare-renew-prodsync-wt` 의 `5575042` 이후 uncommitted 82파일이다.
`git diff HEAD` + untracked 전량을 직접 읽었고, 구 원본은 `medilawyer-boot`/`notification-service` 의 `origin/main` 에서 꺼내 대조했다.
수정 권한 없음 — 코드는 한 줄도 건드리지 않았다.

먼저 잘된 것부터. **개발계 실발송 봉인은 진짜로 성립한다.** 게이트가 닫히면 `RestClientSupport` 자체를 한 번도 부르지 않고(`ReviewBoostV2BizMessageService.java:93-102`), 테스트가 `verifyNoInteractions` 로 그걸 못박아 뒀다(`ReviewBoostV2BizMessageServiceTest.java:70`). URL 인디렉션도 살아 있다 — `docker-compose.yml:137-139`, `deploy/dev/docker-compose.yml:84-89` 둘 다 SES base 를 stub 로 고정하고 `REVIEW_BOOST_SEND_ENABLED=false` 를 명시 주입한다. 기반 기본값도 `config/notification.yml:72` 에서 false 다. 수신번호는 로그·예외 어디에도 원문이 안 남고(`MaskingUtil` 의 "패턴 불일치 시 원문 반환" 구멍을 `maskPhone` 이 `***` 로 막았다), 본문 로깅 필터도 이 앱엔 없다. 발송 오케스트레이션·페이로드 렌더러·HMAC 서명 원문은 구 Kotlin 과 바이트 단위로 같다. 엔티티 24컬럼이 DDL 과 nullable·길이까지 정확히 일치하고 절단 상수(100/100/2000)도 컬럼 폭과 맞다. 82파일 전량이 `apps/booster-app` 안이고 mldelegator·gateway·reviewed·product·crawling·DDL·compose 는 손대지 않았다 — **범위 밖 변경 0건**이다.

그런데 아래 다섯 건이 걸린다. 1·2번은 개발계 검증(R8)을 그대로 막고, 3·4번은 각각 권한과 프론트 계약이 어긋난 것이라 R4 로 넘어가기 전에 정리돼야 한다.

---

## 1. [치명] 게이트 OFF 상태에서 **두 번째 발송부터 DB UNIQUE 위반으로 죽는다**

게이트가 닫히면 합성 응답의 messageId 가 상수 `"SEND_DISABLED"` 다(`ReviewBoostV2BizMessageService.java:186-194`). 이 값이 그대로 `markAccepted` 를 타고(`ReviewBoostMessageSendService.java:175-179`) `provider_message_id` 컬럼에 들어간다(`KakaoAlimTalkSendHistory.java:189-192`). 그런데 DDL 에 이런 게 있다:

```sql
-- app/src/main/resources/db/reviewboost-v2-alimtalk.sql:69-72
CREATE UNIQUE INDEX IF NOT EXISTS uq_kash_provider_message
    ON kakao_alimtalk_send_history (provider, provider_message_id)
    WHERE provider_message_id IS NOT NULL;
```

`provider` 는 항상 `NAVER_SENS` 로 고정이다(`KakaoAlimTalkSendHistory.java:147`). 즉 게이트가 닫힌 환경에서는 **DB 전체를 통틀어 딱 한 건만** 발송 흐름을 끝까지 통과하고, 두 번째부터 `(NAVER_SENS, 'SEND_DISABLED')` 중복으로 `DataIntegrityViolationException` 이 난다. 이 예외는 `sendPrepared` 의 try 밖(`markAccepted` 호출 지점)에서 터지므로 아무도 안 잡는다. 게다가 그 롤백으로 이력은 `PROCESSING` 에 고착되는데 `RETRYABLE_STATUSES = Set.of(PENDING)`(`KakaoAlimTalkSendStatus.java:22`)이라 `claimPending` 이 다시는 선점하지 못한다 — 같은 멱등키로 재요청해도 영원히 안 나간다.

이게 정확히 AC8 이 돌릴 환경이다. compose 두 곳 다 `REVIEW_BOOST_SEND_ENABLED=false` 이고, `ReviewBoostV2DdlTest.java:44-50` 이 이 SQL 을 적용 순서에 넣어두었으니 개발계 PG 에도 이 인덱스가 있다. AC8 ④(4계약 응답)·⑤(이력 행 생성)는 발송을 2건 이상 넣는 순간 깨진다.

테스트로 안 잡힌 이유도 분명하다. 신규 테스트는 전부 repository 를 mock 하고, 실제 PG 를 쓰는 `ReviewBoostV2DdlTest` 는 `uq_kash_idempotency_key` 만 확인한다(`ReviewBoostV2DdlTest.java:139`). 그 하네스가 이미 있으니 몇 줄이면 재현·고정된다.

고치는 방향은 여러 가지겠지만, 표식을 행마다 유일하게 만드는 게 제일 싸 보인다(`SEND_DISABLED-<ULID>` 같은). `provider_message_id LIKE 'SEND_DISABLED%'` 로 SQL 구분 가능성은 그대로 유지된다.

## 2. [치명] 게이트 OFF 는 "일시 정지"가 아니라 "메시지 영구 폐기"다 — release 기본값 true 의 근거가 여기서 무너진다

1번을 고쳐서 중복 충돌을 없애도 더 큰 문제가 남는다. 게이트가 닫힌 채 들어온 요청은 `ACCEPTED` 로 확정되고 멱등키를 소비한다. 나중에 게이트를 켜도 `claimPending` 이 `ACCEPTED` 를 재선점하지 못하니(`KakaoAlimTalkSendStatus.java:22`) 그 메시지는 **영구 미발송인데 성공으로 기록된 상태**가 된다. 같은 멱등키로 재시도해도 `findByIdempotencyKey` 가 기존 건을 돌려주고 끝난다.

그런데 `application-release.yml:200` 은 운영자에게 이렇게 안내한다:

> 운영에서 일시 차단이 필요하면 `REVIEW_BOOST_SEND_ENABLED=false` 를 명시 주입하면 된다.

이 안내대로 하면 차단 구간 동안 들어온 알림톡이 전부 조용히 사라진다. 킬스위치가 아니라 메시지 파쇄기다. 그리고 이 "OFF 는 조용히 멈춘다"는 성질이 바로 release 기본값을 `false → true` 로 뒤집은 근거였는데(`application-release.yml:194-202`, `04-changes-r2.md:31`), 조용한 것은 기본값 탓이 아니라 **OFF 를 ACCEPTED 로 기록하는 설계 탓**이다. 원인을 그대로 두고 기본값만 위험한 쪽으로 뒤집은 셈이라 순서가 거꾸로다.

기본값 뒤집기 자체도 짚어야겠다. R9 는 "발송 기본 OFF 플래그"라고 못박았고 범위 제외 항목에 "실제 알림톡 발송 활성화"가 명시돼 있다(`01-requirements.md:10,23`). 운영 기본값을 ON 으로 바꾸는 건 이관 작업이 단독으로 내릴 결정이 아니라 **사용자 승인이 필요한 운영 정책 변경**이다. 지금 당장 실발송이 새지는 않는다 — 게이트웨이 `Routes.java:35` 의 `legacy-v2` 가 아직 `ServiceId.LEGACY` 를 가리키고 `sendAuto` 는 호출자가 없어서, 운영에서 이 코드에 도달할 경로가 없다. 그래서 "지금은 무해하지만 승인 없이 남기면 안 되는 변경"으로 본다.

덧붙여 문서 불일치도 있다. `config/notification.yml:66-68` 은 아직 "false 면 SENS 호출 없이 **차단 이력만 남긴다**"고 적혀 있는데 구현은 ACCEPTED 이력을 남긴다. 돈이 걸린 게이트의 운영자용 설명이라 그냥 두면 안 된다.

## 3. [치명] 권한 판정이 구 대비 **양방향으로** 어긋났다 — "통과/거절 집합 동일"은 성립하지 않는다

`04-changes-r2.md:24` 는 구 `validateRoleOwnerAndAmdinAndPermissionWriting` 과 1:1 이라 단언했다. 실제로 대조해 보니 두 방향 모두 어긋난다.

**(a) 권한 축소 — 팀에 소속 안 된 조직 OWNER/ADMIN 이 거절된다.**
신 검증기는 판정을 `userHandler.getUserStoreAuthority(appUserId, storeId)` 하나에 맡긴다(`ReviewBoostSendAuthorizationValidator.java:55,66-69`). 그 계산 본체가 이렇다:

```java
// modules/user/.../service/AuthorityService.java:42-45
TeamAppUserEntity teamAppUser = teamAppUserRepository.findByTeamIdAndAppUserId(team.getId(), userId);
if (teamAppUser == null) {
    return userStoreAuthorityConverter.toVisitorModel(userId, storeId);
}
```

팀 멤버십이 없으면 **조직 역할을 보기도 전에** VISITOR 로 확정된다. 반면 구 코드는 조직 역할을 먼저 본다:

```kotlin
// ReviewBoostSendAuthorizationValidator.kt:32-38
if (organizationAppUser.organizationRole in setOf(ORGANIZATION_OWNER, ORGANIZATION_ADMIN)) { return }
val teamAppUser = teamAppUserService.getTeamAppUser(...)   // 여기까지 오지도 않는다
```

즉 팀에 직접 소속되지 않은 조직 소유자/관리자는 구에선 통과, 신에선 403 이다.

**(b) 권한 확대 — 팀에서 해제된 사용자가 통과한다.** 이쪽이 더 위험하다.
`findByTeamIdAndAppUserId` 에는 `usable` 조건이 없다(`modules/user/.../TeamAppUserRepository.java:20`). 같은 파일 15행의 `findAllByAppUser_IdAndUsable` 은 필터를 거는 걸 보면 누락이 맞다. 구는 `TeamAppUserService` 가 `getByTeamIdAndAppUserIdAndUsable(..., true)` 로 조회하고 없으면 예외로 거절했다. 결과적으로 팀에서 soft-delete 된 사용자가 `permission_writing=true` 를 남긴 채면 신 코드는 **알림톡 발송을 허용한다.**

**(c) 판정 축이 (organizationId, teamId) → storeId 로 바뀌었다.**
`ReviewBoostMessageCreationService.java:78` 이 요청받은 팀을 버리고 storeId 만 넘기면, `AuthorityService.java:35` 가 `getByStoreIdAndUsable(storeId, true)` 로 팀을 **다시** 찾는다. 구는 `team` 객체 자체를 검증기에 넘겼다. store 에 usable 팀이 둘이면 요청한 팀이 아닌 팀 기준으로 권한이 평가되거나, `Optional` 반환이라 `IncorrectResultSizeDataAccessException` 으로 500 이 난다. 같은 축의 문제로 `ReviewBoostTeamService.java:40` 의 `findFirstByStore_IdAndUsable` 도 구 `fetchOne()`(`TeamRepositoryImpl.kt:21-28`) 대비 완화다 — 구는 팀이 2건이면 `NonUniqueResultException` 으로 즉시 실패했는데 신은 ORDER BY 없이 아무거나 골라 발송을 진행한다.

권한 테스트 7건(`ReviewBoostSendAuthorizationValidatorTest.java:42-92`) 중 (a)(b)(c) 어느 것도 다루는 케이스가 없다. 검증기가 teamId 를 아예 안 받으니 테스트로 잡힐 수가 없는 구조다.

## 4. [치명] **에러 응답 봉투는 보존되지 않았다** — 성공 봉투를 재현한 논리가 여기서 뒤집힌다

`ReviewBoostV2ApiResponse.java:10-14` 의 논거는 "경로만 지키고 봉투를 바꾸면 R6 전환 순간 프론트가 깨진다"였다. 맞는 말인데, 그 논리가 에러 경로에는 적용되지 않았다.

이관된 예외는 전부 `LegalCareException` 을 상속한다(`BlockedReviewBoostRecipientException.java:12`, `ReviewBoostSendFailedException.java:8` 등). 그리고 booster 의 공통 어드바이스가 이렇다:

```java
// core/src/main/java/legalcare/medilawyer/core/advice/ControllerExceptionAdvice.java:70-78
@ExceptionHandler(LegalCareException.class)
@ResponseStatus(HttpStatus.OK)                       // ← 모든 비즈니스 오류가 HTTP 200
public ResultResponse<?> handleException(LegalCareException e) {
    ResultResponse<?> resultResponse = new ResultResponse<>(e.getHttpStatus());
    ...
```

구는 `GlobalExceptionHandler.handleCustomException` 이 `ErrorType.httpStatus`(400/409/500…)를 **실제 HTTP 상태로** 내보내고 본문은 `ErrorResponse{ errors:[{code,message}], errorMessage, path, timeStamp }` 였다(`ErrorResponse.kt:5`). 신은 HTTP 200 + `ResultResponse{ status:409, code:"08008", message, result }` 다(`ResultResponse.java:12-15`).

즉 수신차단 응답이 **HTTP 409 + `errors[0].code`** 에서 **HTTP 200 + `status:409`** 로 바뀐다. 각 예외 생성자에 애써 보존해 둔 `HttpStatus.CONFLICT`/`BAD_REQUEST` 가 와이어에는 안 나간다. 프론트가 HTTP 상태나 `errors[]` 로 분기하고 있으면 R6 컷오버 순간 깨지는데, 이건 성공 봉투를 지킨 이유와 정확히 같은 종류의 위험이다. 성공만 지키고 에러는 안 지킨 근거가 문서에 없다. (구 `ReviewBoostV2ControllerTest.kt` 가 이관되지 않아 직렬화된 JSON 을 단언하는 테스트가 성공·실패 양쪽 다 없다.)

## 5. [중대] R7② 멱등 **소비측이 테스트 0건**이다 — 구 테스트 8클래스 43케이스가 조용히 빠졌다

`findByIdempotencyKey` 를 단언하는 테스트가 신규 테스트 전체(app + notification)에 **한 건도 없다**. grep 결과 0줄이다. 실제 방어의 1차 관문은 `ReviewBoostMessageCreationService` 의 세 지점(`:81, :107, :170`)인데, 그걸 덮던 구 `ReviewBoostMessageCreationServiceTest.kt`(16 @Test)가 통째로 안 넘어왔다. 지금 있는 멱등 테스트는 `send_idempotencyKeyConflict_reusesExistingDelivery` 하나뿐인데(`ReviewBoostMessageSendServiceTest.java:236`), 이건 creation service 를 mock 한 상태에서 `DataIntegrityViolationException` 흡수 분기만 본다. AC7② "테스트 존재·통과"의 근거로는 약하다.

빠진 목록을 세어 봤다: `MessageCreationServiceTest` 16, `RequestSettingServiceTest` 7, `HospitalTemplatePolicyServiceTest` 6, `SendModelValidationTest` 5, `AlimTalkServiceTest` 3, `MessageConverterTest` 3, `CustomerCreationServiceTest` 2, `V2ControllerTest` 1 — 8클래스 43케이스다. `ReviewBoostDeliveryServiceTest` 도 구 11 → 신 7 로 줄었고, 사라진 것 중에 하필 `백필된 과거 이력을 멱등키로 조회한다` 가 있다.

`04-changes-r2.md:38` 은 이관분 세 개에 "(전량)"을 붙여 놨는데, 그 표기가 **이관하지 않은 8클래스를 가린다.** 신규 서비스 12개 중 7개(`AlimTalkService`, `CustomerCreationService`, `HospitalTemplatePolicyService`, `MessageCreationService`, `MessageTemplateService`, `RequestSettingService`, `TeamService`)가 무테스트다. 분량 문제로 뒤로 미루는 건 받아들일 수 있지만, 미뤘다는 사실 자체는 문서에 적어야 한다.

참고로 동일 멱등키 동시요청의 이중발송 창은 없다고 판단했다. `claimPending` 이 `UPDATE ... WHERE id=? AND status IN (PENDING)` 단일 원자 갱신이고(`KakaoAlimTalkSendHistoryRepository.java:28-38`) 1을 받은 하나만 진행한다. PG 에서 UNIQUE 충돌은 선행 트랜잭션 커밋까지 블록되므로 패자가 `getExisting` 할 때 승자 행이 반드시 보인다. `uq_kash_idempotency_key`(`reviewboost-v2-alimtalk.sql:52`)도 부분 인덱스가 아닌 전역 UNIQUE 라 최후 방어가 확실하다. 이 부분 **설계는 문제없다 — 없는 건 테스트지 방어가 아니다.**

---

## 그 밖에 (중 · 하)

**6. [중] `Set.of(ALL, messagePurpose)` 는 Kotlin `setOf` 와 다르다** — `ReviewBoostRecipientBlockService.java:33-34`. `messagePurpose == ALL` 이 오면 `IllegalArgumentException: duplicate element` 로 즉사한다. 구 `.kt:21` 의 `setOf` 는 조용히 중복 제거했다. 지금 호출자가 `REVIEW_REQUEST` 하나뿐이라 잠복 상태지만(`ReviewBoostMessageSendService.java:121`), 호출자가 하나만 늘어도 터진다. `EnumSet.of(...)` 나 `Set.copyOf(List.of(...))` 면 구 의미가 복원된다.

**7. [중] `markFailed` 의 error_message 가 null 로 저장될 수 있다** — `ReviewBoostMessageSendService.java:216` 이 `exception.getMessage()` 를 넘기고 `ReviewBoostDeliveryService.java:161-165` 의 `truncate` 가 null 을 통과시킨다. 구는 `exception.ErrorMessage.toString()`(non-null 보장, `.kt:216`)이었다. 실패 사유가 빈 이력은 재시도 분류·감사에서 곧바로 손해다.

**8. [중] SENS 응답 바인딩이 느슨해져 UNKNOWN 이던 경로가 ACCEPTED 로 넘어갈 수 있다** — 구 `AlimTalkResDto.kt:3-19` 는 data class 라 `requestId`/`messageId` 가 non-null 이었고 null 이 오면 역직렬화에서 터져 `markUnknown` 으로 갔다. 신 `AlimTalkResponseModel` 은 전 필드 nullable 이라 `statusCode="202"`·`requestStatusCode="A000"` 이면서 id 가 null 인 응답이 오면 `markAccepted(null, null)` 로 진행한다. 4xx/5xx → UNKNOWN 이라는 주 분기와 `builderWithoutStatusHandler` 선택 자체는 양쪽 동일함을 확인했다 — 그 판단은 맞다.

**9. [하] 경로 계약 게이트는 프리픽스만 고정하고 URL 자체는 안 고정한다** — `ReviewBoostV2PathPrefixTest` 는 `WebConfig` 의 predicate 를 직접 평가하니 `.and(...)` 제외 조건을 지우면 진짜로 깨진다(동어반복 아님). 컨트롤러를 다른 패키지로 옮기는 것도 FQN 하드참조라 컴파일 에러다. 다만 `@PostMapping("/api/v2/boost1/...")`(`ReviewBoostV2Controller.java:41`) 문자열이 바뀌거나 클래스 레벨 `@RequestMapping` 이 붙어도 테스트는 초록이다. R6 전환의 유일한 계약이니 `RequestMappingHandlerMapping`/MockMvc 로 최종 URL 을 못박는 편이 안전하다. 세 번째 케이스(`otherModulesAreUnaffected`)는 `forBasePackage` 가 이미 걸러내므로 사실상 동어반복이다. 경로·요청/응답 필드 자체는 구와 완전 일치를 확인했고(5필드·12필드, `@Pattern` 정규식까지 동일), 양쪽 다 Jackson 네이밍 전략 미설정이라 camelCase 로 같다. R3 경로 `/notification/apis/v2/review-boost/stores/{id}/message-template` 도 프리픽스 정상 적용 + 봉투 무변경으로 **R3 는 깨끗하다.**

**10. [하] `ReviewBoostRequestSettingService` 서브트리가 무호출·무테스트인데, 하필 구 대비 의도적 발산이 있는 자리다** — 서비스 + `ReviewBoostRequestSettingConverter` + DTO 2 + `ReviewBoostStoreRepository` 가 프로덕션 참조 0이다(구 호출자였던 `InternalAdminWebController.kt`/`InternalProductReviewBoostController.kt` 는 R4·R5 몫). 요구사항 R2 가 "서비스12"로 지정했으니 고아 변경은 아니지만, `boost_review_request_setting` 에 **쓰는** 경로를 테스트 없이 넣는 셈이다. `insertCommonIfAbsent` 에 `created_at/updated_at` 을 추가한 판단(`BoostReviewRequestSettingRepository.java:24-52`)은 DDL(`reviewboost-v2-alimtalk.sql:140-141`, NOT NULL·DEFAULT 없음)을 보면 맞다 — 다만 네이티브 쿼리라 컴파일이 안 잡아주고 검증 수단(`ReviewBoostV2DdlTest`)이 이미 있는데 안 썼다.

**11. [하] 병원 표시명 공백 처리 차이** — `ReviewBoostHospitalTemplatePolicyService.java:92` 의 `String.trim()` 은 U+0020 이하만 제거하고, Kotlin `trim()`(`.kt:100`)은 NBSP 등 유니코드 공백까지 제거한다. `hospital_display_name` 이 NBSP 로만 채워지면 신은 그걸 병원명으로 쓰고 구는 `store.storeName` 으로 폴백한다.

**12. [정보] 부팅·고아 확인 결과** — 82파일 전량이 R2/R3/R7 에 매핑되고 **UNMAPPED 0건**이다. 엔티티 이름충돌은 `@Entity(name="NotificationBoostMessageTemplate")` 로 정확히 회피했고, 빈 이름도 `reviewBoostV2SendController`/`notificationReviewBoostV2Controller` 로 분리했다(안 하면 기본 빈 이름이 겹친다). `ddl-auto: none` + 128개 엔티티 전수 확인 결과 중복 엔티티명 0, 중복 빈 이름 0. team/store/boost_customer/boost_message_template 다중 `@Table` 매핑은 post·user·product·crawling 선례가 있는 관례라 신규 위반이 아니다. `@SpringBootTest` 전체 컨텍스트 기동이 통과하니 AC3 의 "이름충돌 0"은 실증됐다. `modules:user` 는 안 건드렸다. 다만 `boost_message`/`boost_message_template` 가 이번에 notification 쪽에서 **쓰기** 대상이 되었는데 다른 4모듈이 같은 행의 부분 컬럼을 매핑하고 있다 — 한 요청 안에서 영속성 컨텍스트가 별개 인스턴스를 들고 있을 수 있어 R4·R5 붙일 때 stale read 를 한 번 점검하면 좋겠다.

---

## 빌드 재검증 (작성자 보고와 대조)

`cd apps/booster-app && ./gradlew cleanTest test` 를 직접 돌렸다. **BUILD SUCCESSFUL in 3m 50s.** JUnit XML 을 모듈별로 합산한 실측치는 다음과 같다.

| 모듈 | tests | fail | err | skip |
|---|---|---|---|---|
| core 44 · app 39 · chrono 7 · medicontents 18 · post 67 · external 40 · crawling 68 · notification 68 · user 86 · product 71 | **508** | **0** | **0** | **0** |

`04-changes-r2.md:36-37` 의 508/0/0/0 및 모듈별 분포와 정확히 일치한다. 테스트 결과 보고는 사실이다 — 문제는 통과 여부가 아니라 5번에서 지적한 **덮지 않은 범위**다.

---

## 정리

돌아가는 코드이고 이관 충실도도 대체로 높다. 상태전이·페이로드 렌더링·서명·병원 커스텀 템플릿 분기·멱등 3중 방어는 구와 1:1 이 맞고, 개발계 봉인과 개인정보 마스킹도 실제로 작동한다.

다만 1·2번은 AC8(개발계 구동검증)에 착수하는 순간 바로 부딪히는 결함이고, 3번은 돈·환자 발송이 걸린 엔드포인트의 권한 집합이 양방향으로 어긋난 것이며, 4번은 R6 컷오버에서 프론트가 깨지는 계약 문제다. 5번은 그 넷을 잡고 나면 자연스럽게 따라오는 숙제다. 2번의 release 기본값 건은 코드 문제가 아니라 **사용자 승인이 필요한 결정**이니, 고치기 전에 먼저 물어보는 게 맞다.

VERDICT: REQUEST_CHANGES
