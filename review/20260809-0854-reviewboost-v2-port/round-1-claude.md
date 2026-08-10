# 라운드1 대응: 치명 4 + 중대 1

전부 반영. 반박 0건 — 지적 5개 모두 실제 결함이었고, 확인 과정에서 원인이 지적보다 더 깊은 곳(#1·#4)이라 설계 자체를 바꿨다.

| # | 판정 | 처리 |
|---|---|---|
| #1+#2 게이트 OFF 설계 | **반영(전면 재설계)** | `ACCEPTED+표식` 폐기 → **보류(hold)**. 게이트 판정을 `claimPending` **이전**으로 옮겨(`ReviewBoostMessageSendService.sendPrepared`) 이력을 PENDING·provider id NULL로 남긴다. 부분 UNIQUE의 `WHERE provider_message_id IS NOT NULL` 을 비켜 가 다건 충돌 0, RETRYABLE_STATUSES에 남아 재개 가능, 멱등키 미소진. 어댑터는 플래그 소유자로 남아 `isSendEnabled()` 를 제공하고, `send()` 에 도달하면 조용히 성공시키지 않고 `IllegalStateException` 으로 실패시킨다(도달 불가 방어선). |
| #3 권한 판정 | **반영(3건 모두)** | (a)(b) `UserStoreAuthorityModel` 경로 폐기 — `AuthorityService:42-45`는 팀 미소속 시 조직역할을 보기 전에 VISITOR 확정, `TeamAppUserRepository:20`은 `usable` 필터 부재를 실측 확인. 구 로직을 조건 단위로 재현하도록 `organization_app_user`·`team_app_user` 읽기 전용 매핑 2개를 추가해 직접 판정. (c) 판정 축을 storeId → **요청이 지정한 팀 엔티티**로 되돌렸다. |
| #4 에러 봉투 | **반영** | 구 `GlobalExceptionHandler` 실측: 업무 예외는 **HTTP 400 + `{errors,errorMessage,path,timeStamp}`**(`ErrorType.httpStatus`는 쓰이지 않았다). 컨트롤러 로컬 `@ExceptionHandler` 2개로 재현 — 전역 advice 무변경. |
| #5 멱등 소비측 테스트 | **반영** | 신규 `ReviewBoostMessageCreationServiceTest` 10케이스(멱등 소비 6 포함) + 정규화 3케이스. 미이관분 목록·사유는 `04-changes-r2.md`. |

## 정정한 주장 2개
- "통과/거절 집합 동일"(초안) → **틀렸다.** 팀 미소속 조직 OWNER/ADMIN이 신에선 403, 팀에서 해제된 사용자가 신에선 통과였다. 지금은 조건 단위 재현이고 두 케이스를 `[회귀]` 표시 테스트로 못박았다.
- "(전량)" 표기 → 구 79케이스 중 이관 46. 미이관 33건은 문서에 목록·사유로 남겼다.

## 검증 (강제 재실행)
`cd apps/booster-app && ./gradlew cleanTest test` → **BUILD SUCCESSFUL (3m 52s), 531 tests / 0 failures / 0 errors / 0 skipped** (라운드1 제출본 508 → +23).

수정 중 실제로 잡힌 자체 결함 3건(전부 수정 후 그린): ①`ReviewBoostSendGateHoldDbTest` 가 static 컨테이너를 메서드 간 공유해 건수 단언이 흔들림 → 테스트마다 TRUNCATE ②`verifyNoInteractions(alimTalkService)` 가 게이트 조회까지 금지해 실패 → `never()).send(...)` 로 축소 ③Mockito 중첩 스터빙(`when(...).thenReturn(mockFactory())`) 4곳 → 지역변수로 hoist.

## 남긴 판단 1개 (검토 바람)
`@Valid` 실패가 **HTTP 500** 으로 나간다. 구 `GlobalExceptionHandler` 에 `MethodArgumentNotValidException` 핸들러가 없어 catch-all(500)로 떨어지던 동작을 그대로 재현한 것이다. 계약 보존과 상식이 충돌하는 자리라 일단 보존을 택하고 코드 주석에 개선 대상으로 남겼다 — 컷오버 후 프론트 합의하에 400으로 바꾸는 게 맞다고 본다.

STATUS: REVISED
