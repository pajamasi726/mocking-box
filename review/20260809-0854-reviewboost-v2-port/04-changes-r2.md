# 변경 요약: R2 발송본체(Kotlin→Java) + R3 조회측 + R7②③

92파일(신규 86 = main 74·test 12, 수정 6). 커밋하지 않음. **라운드1 지적 5건 반영 후 갱신본** — 대응 요약은 `round-1-claude.md`.

## 변경 파일
| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `modules/notification/.../notification/reviewboost/**` (신규 27) | v2 발송본체 — 서비스 12·converter 3·DTO 9·컨트롤러/모델 3 | R2 |
| `.../notification/core/code/*.java` (신규 7) | enum 7종(`KakaoAlimTalk*` 6 + `ReviewBoostDeliveryMode`) | R2 |
| `.../notification/persistence/postgresql/entity/*` (신규 6·수정 1) | 지시서 엔티티3 + 발송 경로가 읽는 team·store·customer 3 + `BoostMessageEntity` 에 생성 경로 추가 | R2 |
| `.../persistence/postgresql/repository/*` (신규 6) | 위 엔티티 리포지토리. 템플릿은 `@Repository("notificationBoostMessageTemplateRepository")` | R2·R3 |
| `.../client/alimtalk/service/ReviewBoostV2BizMessageService.java` | SENS HMAC 직접발송 어댑터(마스킹·발송게이트 포함) | R2·R9 |
| `.../notification/core/exception/*` (신규 13·`NotificationErrorCode` 수정) | 구 전역 `ErrorType` 재사용분을 모듈 예외로. 08004~08007 은 구 코드 그대로, 08008~08016 신설 | R2·R3 |
| `.../notification/service/ReviewBoostTemplateService.java`·`converter/`·`controller/api/front/ReviewBoostV2Controller.java`·`controller/model/response/` | 조회측 `/apis/v2/review-boost/stores/{id}/message-template` | R3 |
| `app/.../WebConfig.java` | v2 발송 컨트롤러만 `/notification` 프리픽스에서 제외(경로 불변) | R2 |
| `app/src/main/resources/application-release.yml` | `review-boost.send.enabled` 운영 기본값 false→**true** | R2 |
| `modules/notification/build.gradle` | `ulid-creator`(메시지·템플릿 ID 생성) | R2 |
| `app/src/test/.../ReviewBoostV2PathPrefixTest.java`·`ExternalSendSealingTest.java` | 경로 불변 게이트 3케이스 + 운영 발송 기본값 게이트 2케이스 | R2 |

## Kotlin→Java 변환 — 드리프트 없음 근거
클래스·메서드·필드명, 호출 순서, 상수(`202`/`A000`/`REVIEW_BOOST_AUTO`/`blocked-` prefix/절단 길이)를 전부 보존했다. Kotlin 문법 때문에 형태만 바뀐 것: `data class`→Lombok 값 클래스(+`@EqualsAndHashCode` — 구 테스트가 값 동등성으로 스텁을 매칭한다), `var` 프로퍼티 직접 대입→엔티티 상태전이 메서드(`markAccepted/markFailed/markUnknown/switchToHospitalCustom`), `?:`/`?.`→`Optional`·명시 분기.

**의미가 바뀔 수밖에 없던 3곳**(전부 코드 주석에 대조표 포함):
1. `ReviewBoostSendAuthorizationValidator` — 구는 `AppUser` 시큐리티 주체 + `OrganizationAppUserService`/`TeamAppUserService`. booster엔 셋 다 없다(스프링 시큐리티 미사용). **초안은 `UserStoreAuthorityModel`(authority+canWrite)로 수렴시켰는데 그게 틀렸다**(라운드1 #3): `AuthorityService:42-45`는 팀 멤버십이 없으면 조직 역할을 보기도 전에 VISITOR로 확정하고(→팀 미소속 조직 OWNER/ADMIN이 구에선 통과, 신에선 403), `TeamAppUserRepository:20`의 `findByTeamIdAndAppUserId`에는 `usable` 필터가 없다(→팀에서 해제된 사용자가 구에선 거절, 신에선 통과 = 권한 완화). 지금은 `organization_app_user`·`team_app_user`를 이 모듈이 직접 읽어 구 조건을 **단계별로 재현**한다. 판정 축도 storeId → 요청이 지정한 팀 엔티티로 되돌렸다(한 병원에 팀이 둘 이상이면 어긋난다). 8케이스로 고정, 그중 2개는 위 두 어긋남의 `[회귀]` 테스트.
2. 응답 봉투 — 구 `ApiResponse{status:"200",message,data}` vs booster `ResultResponse{status:200,code,message,result}`. **경로만 지키고 봉투를 바꾸면 R6 전환 순간 프론트가 깨진다**(같은 프론트·같은 URL). 이 엔드포인트 전용 `ReviewBoostV2ApiResponse`로 구 봉투를 재현했고, **에러 봉투도 같이 재현했다**(라운드1 #4): 전역 `ControllerExceptionAdvice`는 `LegalCareException`을 `@ResponseStatus(OK)`로 처리해 409가 `200+status:409`로 나간다. 구 `GlobalExceptionHandler` 실측은 업무 예외 전부 **HTTP 400 + `{errors,errorMessage,path,timeStamp}`**(`ErrorType.httpStatus`는 실제로 쓰이지 않았다). 컨트롤러 로컬 `@ExceptionHandler` 2개로 재현 — 전역 advice는 손대지 않았다.
3. `RestTemplate`→`RestClient` 는 `builderWithoutStatusHandler` 로 붙였다. 기본 `builder` 는 SENS 4xx를 `LegalCareException` 으로 바꿔 호출측의 "거절→FAILED / 그 외→UNKNOWN" 분기를 뒤집는다(구 RestTemplate은 HTTP 오류를 순수 RuntimeException으로 던졌고 그 경로는 UNKNOWN이 맞다).

## 마스킹 · 발송게이트 · 차단 · 멱등 위치
- **마스킹**: `ReviewBoostV2BizMessageService` 의 log 3곳 전부 `maskPhone()` 경유. `MaskingUtil.maskPhoneNumber` 가 패턴 불일치 시 **원문을 그대로 돌려주는** 구멍이 있어 마스킹이 안 일어나면 `***` 로 통째 가린다. 원문 미출력을 테스트 4건이 직접 단언.
- **`review-boost.send.enabled` 소비 = "보류(hold)"** (라운드1 #1#2로 전면 재설계): 플래그 소유자는 어댑터(`isSendEnabled()`)지만, 판정은 오케스트레이터가 **`claimPending` 이전에** 한다. 이유가 셋이다 — ①이력이 PENDING으로 남아 `RETRYABLE_STATUSES`에 걸리므로 게이트를 켜면 그대로 재개된다 ②provider id를 건드리지 않아 NULL이라 부분 UNIQUE `uq_kash_provider_message(WHERE provider_message_id IS NOT NULL)`를 비켜 가 보류가 몇 건 쌓여도 충돌하지 않는다 ③멱등키를 소진하지 않는다. 어댑터 `send()`에 도달하면 조용히 성공시키지 않고 `IllegalStateException`으로 실패시킨다(도달 불가 방어선 — 새 호출자가 확인을 빠뜨려도 외부로 안 나간다).
  - **폐기한 초안**: `ACCEPTED` 확정 + `provider_*_id='SEND_DISABLED'`. 2건째부터 UNIQUE 위반이라 DB당 1건만 통과했고(AC8 ④⑤를 그대로 막는다), 게이트를 켜도 재발송되지 않는 **영구 폐기**였다. 전 테스트가 repo mock이라 그린이었다 → 그래서 실 DB 테스트를 새로 넣었다(아래).
- **기본값 설계**: 기반(`config/notification.yml`)·local·dev = `false`, **release = `true`**. 근거는 재검토 후 갱신했다(초안의 "조용히 멈춘다"는 보류 설계에선 약해진다): ①R6는 레거시를 그대로 대체하는 컷오버인데 레거시엔 게이트가 없었고 항상 발송했다 — env 누락으로 동작이 달라지면 안 된다 ②보류도 여전히 장애다(고객 미수신). 쌓인 PENDING을 자동 재개하는 재조정 배치는 이 저장소에 없다 ③뒤집어 말하면 OFF가 이제서야 **복구 가능한 킬스위치**가 됐다 — release yml이 광고하는 그대로.
- **R7③ 차단**: `ReviewBoostMessageSendService.rejectBlockedRecipient` 를 `send()`(권한확인 직후)·`sendAuto()` 최앞단 = 메시지·고객·멱등키 생성 **이전**. 차단 시 어댑터 호출 0회 + `KakaoAlimTalkSendHistory.blocked(...)` 1건(감사 전용 키 `blocked-<ULID>`, `errorCode=RECIPIENT_BLOCKED`).
- **R7② 멱등**: 소비측 `ReviewBoostDeliveryService.findByIdempotencyKey` (creation 3지점에서 조회) + `DataIntegrityViolationException` 흡수 후 기존 건 재사용(send/sendAuto) + DB `uq_kash_idempotency_key` 최후방어. 생성측(SHA256)은 R4.

## 테스트 결과
`cd apps/booster-app && ./gradlew cleanTest test` → **BUILD SUCCESSFUL (3m 52s), 531 tests / 0 fail / 0 err / 0 skipped**
(app 43, core 44, chrono 7, crawling 68, external 40, medicontents 18, notification 87, post 67, product 71, user 86 — 이관 전 459 대비 +72)

**구 v2 테스트 79케이스 중 46 이관**(라운드1 지적대로 "(전량)" 표기 정정):
| 구 테스트 | 구/이관 | 비고 |
|---|---|---|
| MessageSendService | 9 / 9 | 전량 + 게이트 보류·재개 2 신설 = 11 |
| RecipientBlock | 4 / 4 | 전량 |
| AlimTalkPayloadRenderer | 6 / 6 | 전량 |
| MessageConverter | 3 / 3 | 전량(정규화는 멱등키·차단조회의 입력이라 이관) |
| MessageCreationService | 16 / 10 | 멱등 소비 6 + 권한·팀·환자 4 |
| DeliveryService | 11 / 7 | 선점·차단이력·상태가드·절단 |
| SendAuthorizationValidator | 6 / 8 | 구 boot 서비스 대상이라 복사 불가 → 조건 단위 재작성(회귀 2 포함) |
| V2Controller | 1 / 2 | 와이어 계약(성공·에러 봉투)으로 대체 |

**미이관 33건과 사유**: RequestSettingService 7(소비처가 R4·R5에서 붙는다 — 서비스 자체도 아직 호출자 없음) · HospitalTemplatePolicy 6(정책 해석은 R3 `ReviewBoostTemplateService` 테스트 7이 같은 테이블·같은 분기를 덮는다) · MessageCreation 잔여 6(템플릿 비활성·정책 위반 등 — `NotAllowed`/`NotFound` 예외 경로는 각 서비스 테스트에 존재) · SendModelValidation 5(jakarta Bean Validation 자체 동작, 애너테이션은 1:1 복사) · Delivery 잔여 4 · AlimTalkService 3(템플릿 재확인/로깅) · CustomerCreation 2(`REQUIRES_NEW` 동작은 실 트랜잭션 없이는 무의미).

**신규(구에 없던 것) 25**: SENS 어댑터 마스킹·게이트 6 · **게이트 보류 실 DB 4** · 경로 불변 3 · 와이어 계약 2 · R3 템플릿 조회 7 · 운영 발송 기본값 2(봉인 테스트 16→17 포함) · 게이트 보류/재개 오케스트레이션 2.

`@SpringBootTest` 전체 컨텍스트 기동 테스트 통과 = **AC3 "기동 시 JPA 엔티티/빈 이름충돌 0"** 실증(신규 엔티티 8·리포지토리 8·컨트롤러 2 포함).

**mock으로는 못 잡는 것 전담**: `ReviewBoostSendGateHoldDbTest`(Testcontainers PG + 배포 DDL 원본) 4케이스 — 보류 3건이 UNIQUE 충돌 없이 쌓임 / 게이트 ON 시 3건 전부 재선점 / **[회귀] 폐기한 초안(고정 표식)은 2건째에서 `uq_kash_provider_message` 위반** / 멱등키 UNIQUE 최후방어.

## AC 자가 점검
- **AC2** ⚠️ 부분 — 경로 `/api/v2/boost1/organizations/{o}/teams/{t}/messages` 불변을 `ReviewBoostV2PathPrefixTest` 가 고정, 발송 어댑터 실호출 0(게이트 OFF 시 `verifyNoInteractions`). **게이트웨이 경유 실호출은 미검증**(R8 범위, 게이트웨이는 아직 LEGACY 라우팅).
- **AC3** ✅ 조회 경로 구현 + 전체 컨텍스트 기동 충돌 0. 실 HTTP 응답 확인은 R8.
- **AC7 ②③** ✅ 차단·멱등 각각 테스트 존재·통과(차단 시 어댑터 0회 + BLOCKED 1건 단언).

## 알려진 한계 / 리뷰어에게
- **⚠️ `@Valid` 검증 실패가 HTTP 500 으로 나간다.** 구 `GlobalExceptionHandler` 에 `MethodArgumentNotValidException` 핸들러가 없어 catch-all(500)로 떨어지던 동작을 그대로 재현한 것이다(계약 보존 우선). 상식과 충돌하는 자리라 코드 주석에도 개선 대상으로 적어 뒀다 — **컷오버 후 프론트와 합의해 400 으로 바꾸는 것을 권한다.**
- **보류 건을 자동 재개하는 주체가 없다.** 게이트를 끄면 PENDING 이 쌓이고, 켜도 같은 멱등키로 재요청이 와야 나간다(`RETRYABLE_STATUSES`·`claimPending` 은 배치를 받을 준비만 돼 있다). 재조정 배치는 이 저장소 범위 밖이다 — release 기본값을 ON 으로 둔 근거 ②가 이것이다.
- **권한 판정을 위해 `organization_app_user`·`team_app_user` 읽기 매핑을 추가했다.** 근본 원인은 `modules:user` 쪽(`AuthorityService` 의 VISITOR 조기 확정, `TeamAppUserRepository` 의 `usable` 누락)이지만, 거기를 고치면 post·product·notification 조회측의 권한 판정이 함께 바뀐다 — 이번 범위에서 검증할 수 없어 건드리지 않았다. **`usable` 누락은 이 모듈 밖에도 남아 있는 문제다(후속 태스크 권장).**
- **`ReviewBoostRequestSettingService` 3진입점은 아직 호출자가 없다.** 소비처(product 구독등록 internal 접점 / adminweb 템플릿 전환)가 R4·R5에서 붙는다. 같은 테이블·같은 정책을 `HospitalTemplatePolicyService` 와 나눠 갖는 구조라 쪼개면 규칙이 두 군데로 갈려 함께 옮겼다. 같은 이유로 `ReviewBoostRequestSettingConverter` 는 구 4메서드 중 internal 모델을 쓰는 2개를 뺐다.
- **team·store·customer 읽기 엔티티 3개를 notification 모듈에 새로 만들었다.** 이 저장소는 모듈마다 자기 `team`/`store` 매핑을 두는 게 확립된 관례(user·product·post·crawling 4곳 선례)라 그걸 따랐고, `modules:user` 를 건드리지 않는다. 클래스명에 `ReviewBoost` 접두가 있어 `@Entity(name=)` 재명명은 불필요(템플릿만 필요해서 지시대로 부여).
- **`insertCommonIfAbsent` 네이티브 INSERT 에 `created_at/updated_at` 을 추가**했다(구는 3컬럼). 우리 DDL의 두 컬럼은 `NOT NULL` + DEFAULT 없음이고 네이티브 경로는 JPA 감사를 안 타서, 문자 그대로 옮기면 NOT NULL 위반으로 즉사한다. DEFAULT 가 있는 스키마에서도 안전한 방향이다. **DDL은 손대지 않았다.**
- `BoostMessageEntity` 를 읽기전용→생성 가능으로 확장했다(같은 테이블 이중 엔티티 금지 원칙). `Persistable` 부여로 save 가 merge(SELECT 선행) 대신 persist 로 간다 — 기존 조회 경로의 매핑·필드는 무변경.
- jsonb 저장 경로는 R2에 없고(본문·에러메시지 모두 text/varchar) 금액도 없다. `REQUIRES_NEW` 는 구 설계대로 별도 빈(`ReviewBoostCustomerCreationService`)에 남겼다 — 프록시 경계가 있어 `TransactionTemplate` 이 필요한 "runCatching 안" 함정이 아니다(근거는 클래스 주석).
- **R1 인수인계 처리 완료**: `send.enabled` 소비 코드 없음 → 이번에 어댑터가 소비. 구 `phone=$phone` 원문 로깅 → 마스킹.

STATUS: DONE
