# Round 1 — P4 리뷰 대응

대상 worktree `/Users/steve/steve/legalcare-renew-prodsync-wt` (기준 `8fb454f2`). 커밋하지 않았다.

먼저 B1 부터. **반박하지 않는다 — 다만 리뷰어의 근거보다 더 강한 근거를 만들어서 확인했다.**
리뷰어는 구조가 같은 look-alike 클래스를 새로 컴파일해 재현했는데, 나는 **운영에 배포된 것과 같은
`medilawyer-boot` 컴파일 산출물**(`module-storage/core-storage/build/classes/kotlin/main`, kotlinc
1.9.25)에 jackson-module-kotlin 2.17.2 를 붙여, 부착 지점 **323곳을 한 곳씩 격리해** 쟀다
(대상 필드 하나만 빼고 나머지는 전부 채운 JSON — 다른 필드가 먼저 터져서 결론이 오염되는 걸 막으려고).

```
원시형 38곳 → 38/38 통과 (키 누락·명시적 null 모두 0/false)
참조형 285곳 → 285/285 MismatchedInputException
ERROR 0
```

가르는 선이 정확히 "구 Kotlin 타입이 JVM 원시형이 되는가"라는 게 실측으로 확정됐다. 기전도 붙는다 —
Kotlin non-null `Long` 은 주생성자에서 `long` 이 되고, Jackson 의 `PropertyValueBuffer._findMissing`
이 원시형 creator 파라미터에 기본값을 채워 넣기 때문에 `KotlinValueInstantiator` 의 null 검사에
**애초에 도달하지 않는다**. 재현 코드는 `P4-scripts/OldPrimitiveProbe.java`, 결과는 `probe38.txt` 다.

실제로 이 라운드에서 가장 값나가는 발견은 B1 이 아니라 **OSIV 쪽**이었다. 리뷰어가 정직하게
"실행으로 확인 못 했다"고 남긴 그 한 단계를 실행해 보니, 지적의 **읽기 축(L1·L2)은 성립하지 않고
쓰기 축(N1(a))만 실재**했다. 아래 OSIV 절에 측정값을 적었다.

---

## BLOCKER 1 — `@LegacyNonNull` 원시형/참조형 분류

**반영.** 마커를 둘로 갈랐다. 원시형 자리는 `@LegacyNonNullPrimitive` 로 바꿔 **JVM 기본값으로
통과**시키고(구와 동일), 참조형 285곳은 `@LegacyNonNull` 그대로 던진다.

| 항목 | 처리 |
|---|---|
| ① 원시형 38곳에서 `@LegacyNonNull` 제거 | 38/38 `@LegacyNonNullPrimitive` 로 치환. `LegacyNonNullModule` 이 null 이면 0/false 를 채운다 |
| ② 게이트 타입 인지화 | `p4_nonnull_typed.py`(오프라인) + **`LegacyNonNullInventoryTest`(빌드 안)**. 원 스크립트는 "대체됨" 헤더를 붙여 남겼다 |
| ③ `LegacyNonNullContractTest:60-72` 반전 | 구 동작(0 으로 진입)을 못박도록 뒤집고, "같은 DTO 의 참조형은 여전히 거부"를 짝으로 추가 |
| ④ P1/P2 AC3/AC4 재실행 | **못 했다.** 리플레이는 개발계 스택이 필요하고 이번 라운드는 코드 수정만이다 — 아래 "남은 것" 참조 |

기동 시점 방어도 넣었다. 박싱 원시형에 `@LegacyNonNull` 이 붙으면 `LegacyNonNullModule.scanOnce`
가 예외를 던지고 앱이 뜨지 않는다. 같은 필드에 두 마커를 같이 붙여도 마찬가지다.

### 원시형 38곳 — 구 실측 전수

값은 `키누락/명시적null` 두 경우의 결과다. 전부 통과(예외 없음)이며 신도 이제 같다.

| # | 신 파일:라인 | 필드 | 구 Kotlin | 구 실측 |
|---|---|---|---|---|
| 1 | `db/dto/admin/OrganizationAppUserAddReq.java:26` | `appUserId` | `Long` | 0/0 |
| 2 | `db/dto/admin/SeoHospitalUpdateReq.java:23` | `publicSeoEnabled` | `Boolean` | false/false |
| 3 | `db/dto/airtable/SaveThreadCommentFromAirtableRequest.java:34` | `dbSave` | `Boolean` | false/false |
| 4 | `db/dto/airtable/SaveThreadFromAirtableRequest.java:29` | `storeId` | `Long` | 0/0 |
| 5 | `db/dto/airtable/SaveThreadFromAirtableRequest.java:47` | `dbSave` | `Boolean` | false/false |
| 6 | `db/dto/airtable/SaveThreadReplyFromAirtableRequest.java:36` | `dbSave` | `Boolean` | false/false |
| 7 | `db/dto/appUserChannelAccount/AppUserChannelAccountLoginReq.java:26` | `storeId` | `Long` | 0/0 |
| 8 | `db/dto/appUserStore/MatchStoreReq.java:18` | `storeId` | `Long` | 0/0 |
| 9 | `db/dto/bizmessage/AlimTalkResDto.java:57` | `useSmsFailover` | `Boolean` | false/false |
| 10 | `db/dto/external/AiBoostCreationResult.java:28` | `score` | `Long` | 0/0 |
| 11 | `db/dto/external/KakaoBusinessSearchResultMeta.java:38` | `isEnd` | `Boolean` | false/false |
| 12 | `db/dto/external/KakaoBusinessSearchResultMeta.java:41` | `pageableCount` | `Int` | 0/0 |
| 13 | `db/dto/external/KakaoBusinessSearchResultMeta.java:43` | `totalCount` | `Int` | 0/0 |
| 14 | `db/dto/external/NaverSearchResult.java:23` | `total` | `Int` | 0/0 |
| 15 | `db/dto/external/NaverSearchResult.java:25` | `start` | `Int` | 0/0 |
| 16 | `db/dto/external/NaverSearchResult.java:27` | `display` | `Int` | 0/0 |
| 17 | `db/dto/kafka/ConsumerScheduleCrawlingTaskWithTargets.java:44` | `timestamp` | `Long` | 0/0 |
| 18 | `db/dto/kafka/ConsumerThreadDisplayStatusModified.java:29` | `storeId` | `Long` | 0/0 |
| 19 | `db/dto/kafka/NewPostAddReq.java:36` | `score` | `Long` | 0/0 |
| 20 | `db/dto/kafka/NewPostAddReq.java:59` | `isInsulting` | `Boolean` | false/false |
| 21 | `db/dto/kafka/NewPostAddReq.java:62` | `isDefamatory` | `Boolean` | false/false |
| 22 | `db/dto/kafka/PostWordAddReq.java:29` | `contentFirstIndex` | `Int` | 0/0 |
| 23 | `db/dto/kafka/PostWordAddReq.java:31` | `contentLastIndex` | `Int` | 0/0 |
| 24 | `db/dto/kafka/ProducerTaskUpdate.java:43` | `batchSize` | `Int` | 0/0 |
| 25 | `db/dto/kafka/ProducerTaskUpdate.java:45` | `batchInterval` | `Int` | 0/0 |
| 26 | `db/dto/organization/dto/OrganizationRegisterationReq.java:19` | `storeId` | `Long` | 0/0 |
| 27 | `db/dto/organizationAppUser/InviteOrganizationAppUserReq.java:34` | `permissionWriting` | `Boolean` | false/false |
| 28 | `db/dto/organizationAppUser/InviteOrganizationAppUserReq.java:36` | `permissionReading` | `Boolean` | false/false |
| 29 | `db/dto/reviewBoost/GiftSaveReq.java:28` | `isValid` | `Boolean` | false/false |
| 30 | `db/dto/reviewBoost/MessageButtonReq.java:26` | `order` | `Int` | 0/0 |
| 31 | `db/dto/store/dto/InternalStoreRegistrationRequestModel.java:24` | `hospitalOriginId` | `Long` | 0/0 |
| 32 | `db/dto/store/dto/NewStoreChannelReq.java:18` | `storeId` | `Long` | 0/0 |
| 33 | `db/dto/team/TeamRegistrationReq.java:19` | `storeId` | `Long` | 0/0 |
| 34 | `db/dto/teamAppUser/TeamAppUserPermissionUpdateReq.java:19` | `permissionWriting` | `Boolean` | false/false |
| 35 | `db/dto/teamAppUser/TeamAppUserPermissionUpdateReq.java:21` | `permissionReading` | `Boolean` | false/false |
| 36 | `db/dto/teamAppUser/TeamAppUserRegistrationReq.java:17` | `permissionWriting` | `Boolean` | false/false |
| 37 | `db/dto/teamAppUser/TeamAppUserRegistrationReq.java:19` | `permissionReading` | `Boolean` | false/false |
| 38 | `db/dto/teamAppUser/TeamChannelAccountLoginReq.java:26` | `storeId` | `Long` | 0/0 |

**참조형 285곳**은 전건 THROW 로 실측돼 `@LegacyNonNull` 을 그대로 뒀다. 파일별 분포는
`modules/legacy/src/test/resources/legacy/nonnull-inventory.tsv` 323행이 전부 갖고 있다.

리뷰어 표와의 차이 두 가지: `AlimTalkResDto.java:57 useSmsFailover`(리뷰어 표에 없음)가 원시형에
포함되고, `SaveThread{Comment,Reply}` 는 각 파일 1건씩이다. 총계는 38 로 같다.

---

## BLOCKER 2 — SDK 직접 호출 egress

**반영.** 다만 Gemini 하나가 아니었다. `apps/booster-app` 전 모듈을 훑은 결과 **RestClient 를
거치지 않고 소켓을 직접 여는 자리가 6종**이고, 그중 셋은 엔드포인트를 바꿀 수단이 코드에도
프로퍼티에도 없다.

| 벤더 | 자리 | 오버라이드 | 처리 |
|---|---|---|---|
| Gemini | `legacy ExternalClientService.java:414`(신 `:422`), `external GeminiAiService.java:32,42` | **불가** | 호출 직전 `egressGate.check` 3곳 |
| Slack SDK | `notification SlackMessageSender.java:24,37` | **불가** | 호출 직전 2곳 |
| AWS S3 | `core AwsConfig.s3Client` | **불가** | `ExecutionInterceptor` 로 전송 직전 1곳(현재·미래 호출부 전부) |
| AWS SES | `notification AwsSESConfiguration.sesClient` | **불가** | 동일 |
| Slack raw HttpClient | `core SlackErrorNotifier`·`RenewSlackNotifier`·`SlackBootFailureListener` | 일부 가능 | 전송 조건에 게이트 추가(무동작, 예외 아님) |
| Google OAuth | `user GoogleOauth2Service.java:37,63` | **불가** | 2곳. 현재 도달 경로 없음(latent) |

게이트는 `core/.../egress/ExternalEgressGate` 이고 **기본값이 차단**이다(`external.egress.enabled`).
`application-release.yml` 에서만 연다 — 즉 local·dev·리플레이 스택은 설정을 건드리지 않아도 봉인된다.
반대로 두면(기본 허용) 플래그를 빠뜨린 한 번의 기동이 곧 실송신이고, 나가는 것이 환자 리뷰 원문·
고객 이메일·알림톡이라 되돌릴 수 없다.

**차단 시 응답과 리플레이 판정.** 송신 직전에 `ExternalEgressBlockedException`(unchecked)을 던진다.
벤더가 401 을 돌려줄 때 SDK 가 던지는 자리와 같은 자리라 호출부 `try/catch` 와 에러코드 매핑이
그대로 따라온다 — 예를 들어 `makeBoosterReview` 는 구와 같은
`CustomException(REQUEST_FAIL_SEARCH)` 로, 그 위는 500 `SERVER-001` 로 간다. **즉 개발계의 현재
동작(자리표시자 키 → 인증 실패)과 상태코드가 같고, 다른 것은 예외 메시지 문자열뿐이다.**
리플레이 판정에서는 "구·신 모두 외부 호출이 실패한 경로"로 같은 자리에 떨어지고 상태코드 비교는
움직이지 않는다. Slack 알림 3곳만 예외적으로 "무동작"인데, 원래도 실패해도 무해한 부수 경로라
예외를 던지면 오히려 없던 5xx 가 생긴다.

`ExternalSendSealingTest` 에 Gemini 키 2개(`google.api.api-key`,
`google.reviewed.translation.api-key`)의 자리표시자 검사와 프로파일별 게이트 상태 검사를 추가했다.

---

## BLOCKER 3 — 뮤테이션

**반영.** 리뷰어와 같은 실험을 격리 사본에서 그대로 재현한 뒤 사살률을 실측했다.

### ① 리뷰어 재현 — 98개 전부 한 번에

대상 9파일에서 **본문이 있는 메서드 98개**를 전부
`throw new UnsupportedOperationException("MUTANT")` 로 바꾸고 전체 스위트를 돌렸다.
(리뷰어는 99로 셌다. 파서 경계 1건 차이이고 파일 구성은 같다.)

```
리뷰어 측정(수정 전):  BUILD SUCCESSFUL   864 tests, 0 failed   ← 99개 중 0개 잡힘
이번 측정(수정 후):    BUILD FAILED       :modules:legacy:test 에서 129 tests, 10 failed
```

전부 지웠을 때 **아무도 모르던 상태**는 끝났다. 다만 이건 "무언가는 잡는다"까지만 말한다.

### ② 개별 사살률 — **98개 중 6개 (6.1%)**

한 번에 하나씩 넣고 `:modules:legacy:test :core:test` 를 98회 돌렸다.
(①의 전체 스위트 실행이 `:modules:legacy:test` 에서 멈춰 `:app:test` 까지 가지 않았다 —
`:app` 쪽이 추가로 무는지는 표본으로 따로 확인했고, `AirtableFacade` 뮤턴트에 대해
`:app:test` 단독은 0건이었다. `:app` 은 대부분 매핑·킬스위치 같은 구조 게이트다.)

| 파일 | 사살/전체 | 비고 |
|---|---|---|
| `AirtableFacade` | **3/3** | 이번에 붙인 `AirtableFacadeDetachTest` |
| `InternalStoreService` | **2/13** | `saveAuthBypass`·`searchAuthBypassStores` — 인가 축 |
| `BoostFacade` | **1/17** | `sendMessagesFcd` — 발송 축 |
| `KafkaConsumer` | 0/40 | **미커버** |
| `EventSchedulerService` | 0/9 | 미커버 |
| `ScheduledService` | 0/5 | 미커버 |
| `AdminService` | 0/5 | 미커버 |
| `AdminFacade` | 0/4 | 미커버 |
| `PostFacade` | 0/2 | 미커버 |
| **합계** | **6/98 (6.1%)** | 수정 전 0/99 |

### ③ "덮었다고 주장한 곳은 실제로 무는가" — 8/8

전수 6.1% 는 낮다. 그래서 <b>주장한 커버리지</b>만 따로 쟀다. 위험도 순으로 고른 지점에
같은 뮤테이션을 넣었을 때 전부 죽는다.

| 뮤테이션 대상 | 결과 | 잡은 테스트 |
|---|---|---|
| `BoostFacade#sendMessagesFcd` | KILLED | `BoostSendBehaviorTest` |
| `StoreFacade#authenticateTeamAuthFcd` | KILLED | `StoreAuthBypassBehaviorTest` |
| `InternalStoreService#saveAuthBypass` | KILLED | `InternalAuthBypassGrantTest` |
| `InternalStoreService#searchAuthBypassStores` | KILLED | `InternalAuthBypassGrantTest` |
| `AirtableFacade#saveThreadFromAirtable` | KILLED | `AirtableFacadeDetachTest` |
| `TeamAppUserFacade#registerAppUserFcd` | KILLED | `LegacyOsivLazyLoadingTest` |
| `LegacyNonNullModule#enforce` | KILLED | `LegacyNonNullModuleTest` 외 |
| `ExternalEgressGate#check` | KILLED | `ExternalEgressGateTest` |
| `MessagesSendReq` 에서 `@LegacyNonNull` 1개 제거 | **KILLED** | `LegacyNonNullInventoryTest` (리뷰어 측정에선 SURVIVED) |

> 이 표를 만들다 내 하네스에서 오탐을 하나 잡았다. zsh 에서 `./gradlew $tasks` 로 여러 태스크를
> 한 문자열로 넘기면 단어 분리가 안 돼 gradle 이 실패하고, 그게 KILLED 로 기록된다. 그래서
> `AirtableFacade` 는 처음에 KILLED 로 나왔지만 실제로는 **SURVIVED** 였다 — 그때 방어가
> 없다는 걸 알고 `AirtableFacadeDetachTest` 를 붙였고, 위 3/3 은 그 뒤의 재측정값이다.

### 무엇을 덮었고 무엇을 안 덮었나

| 위험 축 | 상태 |
|---|---|
| 돈·발송 (boost1 발송) | `sendMessagesFcd` 커버. **기프트 5경로·`customerSelectGiftFcd` 미커버** |
| 인가 (`/internal/reviewed/auth-bypass`) | 부여·소비 양쪽 커버 |
| 개인정보 | 로깅 마스킹은 기존 `LegacyLogMaskingTest`·`KafkaDtoLoggingMaskTest`. **네트워크 축은 egress 게이트로 이동** |
| 상태변경 (Airtable 수신) | 3경로 전부 커버 |
| **카프카 컨슈머 분기** | **미커버 0/40** — 지시받은 축인데 못 했다. 아래 "남은 것" |
| 스케줄러·관리자 | 미커버 |

지시받은 축 중 **카프카 컨슈머 분기가 비어 있다**. 40개 뮤턴트 전부 생존이고, 이 구간은
`KafkaConsumer` 652줄에 핸들러가 몰려 있어 행위 테스트를 붙이려면 토픽별 페이로드 픽스처가
필요하다. 이번 라운드 안에 정직하게 넣을 수 있는 분량이 아니어서 손대지 않았다.

---

## OSIV — 실행으로 확정. **읽기 축은 반박, 쓰기 축은 반영**

지시받은 대로 L1 에 통합테스트를 붙였다(`app/src/test/.../osiv/LegacyOsivLazyLoadingTest`,
Testcontainers PostgreSQL 에 실제 행을 넣고 legacy 도메인만 올린 최소 컨텍스트). 결과가 지적과 갈린다.

| 축 | 실측 | 판정 |
|---|---|---|
| 프록시 **식별자** 게터 | 초기화 없이 값을 준다 | 구·신 차이 **없음** |
| 프록시 그 외 속성 게터 | `LazyInitializationException` | 세션 유무로 갈림(기전 확인) |
| **L1 `registerAppUserFcd`** | 트랜잭션·OSIV 없이 **성공**, 행 1건 기록 | **반박** — "구 500 → 신 200" 아님 |
| **`save` 없는 setter** | EM 이 바인딩돼 있으면 뒤 트랜잭션 커밋에 실려 **UPDATE**. 없으면 no-op | **반영** — N1(a) 실재 |

L1 이 프록시에서 꺼내는 것은 전부 식별자다 — `TeamAppUserRegistrationRes.from` 의
`getIdentifier()`/`getId()`, 그리고 연관을 새 엔티티에 넘겨 저장하는 것은 FK 값만 필요해 초기화를
요구하지 않는다. L2(`JwtProvider.java:184` `getOrganization().getIdentifier()`)도 같은 부류라 같은
이유로 성립하지 않는다. 리뷰어가 미확인으로 남긴 바로 그 단계가 결론을 뒤집었다.

쓰기 축은 실재하므로 고쳤다. 전역 `open-in-view: false` 는 쓰지 않았고(지시대로 —
`modules/user` 9경로가 깨진다), `AirtableFacade` 세 갈래에서 조회 직후 `entityManager.detach` 로
구와 같은 준영속 상태를 만든다. `@Transactional` 이나 `save` 를 새로 넣는 쪽은 오히려 구엔 없던
UPDATE 를 확정하므로 택하지 않았다.

**M7(AC 양방향)은 반영에 동의하되 이번 라운드에서 코드로 할 일은 없다.** AC3/AC6/AC7 판정 기준
개정은 P5 실행 절차 문서의 몫이라 아래 "오케스트레이터에게"로 넘긴다. 다만 근거가 하나 줄었다 —
"구 5xx → 신 2xx" 방향의 실물로 제시된 2건(L1·L2)은 위 실측으로 **divergence 가 아니다**.
양방향 계수 자체는 여전히 필요하다(AirtableFacade 3경로가 행 단위 차등 대상으로 남는다).

---

## 나머지 지적

| # | 지적 | 판정 | 처리 |
|---|---|---|---|
| M4 | `/internal` permitAll 근거 (a) 가 사실이 아니다 | **반영(부분)** | `application.yml` 주석 정정 — 필터는 출처를 보지 않고, 14건은 포트에 닿는 누구에게나 무인증이라고 명시. 로컬 어댑터 2건 **제거는 하지 않았다**: booster 밖 호출자가 없다는 것을 증명하지 못했고(저장소 grep 0건은 "이 저장소 안에 없다"까지만), 구도 permitAll 이었다. 실해법은 코드가 아니라 포트 배치라고 적었다 |
| M5 | 스케줄러·컨슈머 기본 꺼짐을 런타임으로 실증 불가 | **동의, 처리 없음** | 배포 이미지에 이식분이 없다는 리뷰어 실측이 맞다. AC10 그린 처리는 P5 몫 |
| M6 | 멱등키 `@Size` 200 복원 | **반영(+확장)** | `@Size` 만 되돌리면 400 이 500(DB 제약)으로 바뀔 뿐이다 — 신 저장소가 `kakao_alimtalk_send_history.idempotency_key VARCHAR(100)` 이고 구 저장소(`boost_message_delivery`)는 200 이었다. 모델·엔티티·DDL 셋을 같이 200 으로 맞췄다. DDL 은 조건부 `ALTER`(이미 200 이면 no-op — 무조건 걸면 UNIQUE 인덱스가 재생성돼 제약 검사 순서가 바뀐다. 실측으로 잡았다) |
| M7 | AC 판정 양방향화 | **동의, 문서 몫** | 위 OSIV 절 |
| N1 | AirtableFacade 쓰기 축 | **반영** | 위 OSIV 절 |
| N2 | 실 외부 URL 기본값 5개 | **부분 처리** | 이 5개는 전부 `RestClient` 경유라 base-url 스텁으로 막힌다. 이번에 넣은 게이트는 스텁이 안 통하는 SDK 축 전용이다. "4xx 도 실호출"이라는 지적 자체는 맞고, 그건 P5 실행 스택 선택(replay compose)의 문제라 아래로 넘긴다 |
| N3 | `@LegacyNonNull` 보강이 2건이 아니라 4클래스 14필드 | **반영** | 축소 보고였던 것이 맞다. 이제 부착 전수 323행이 `nonnull-inventory.tsv` 로 저장소 안에 있고 빌드가 대조한다 |
| N4·N5·N6·N7·N8 | 문제 없음 판정 | 수용 | — |
| N9 | boost1 은 15가 아니라 16경로 | **동의** | 지시서 숫자 정정 대상. 코드는 16개 전부 이식돼 있어 변경 없음 |
| AC6 "이중기록 0건" 재정의 | 델타 비교로 바꿀 것 | **동의, 문서 몫** | 원장 멱등키 부재는 구 그대로 — P6 백로그로 남길 것 |

---

## 수정 파일 (49 수정 + 12 신규, +348/−71)

| 지적 | 파일 |
|---|---|
| B1 | `LegacyNonNullPrimitive.java`(신) · `LegacyNonNullModule.java` · **DTO 26개**(38곳 마커 치환) · `LegacyNonNullInventoryTest.java`+`nonnull-inventory.tsv`(신) · `LegacyNonNullContractTest.java` |
| B2 | `core/egress/{ExternalEgressGate,ExternalEgressBlockedException,AwsEgressGateInterceptor}.java`(신) · `ExternalClientService`·`GeminiAiService`·`SlackMessageSender`·`AwsConfig`·`AwsSESConfiguration`·`GoogleOauth2Service`·`SlackErrorNotifier`·`RenewSlackNotifier`·`SlackBootFailureListener` · `application-release.yml` · `ExternalSendSealingTest`·`ExternalEgressGateTest`(신) + 기존 Slack 테스트 4개 |
| B3 | `BoostSendBehaviorTest`·`StoreAuthBypassBehaviorTest`·`InternalAuthBypassGrantTest`·`AirtableFacadeDetachTest`(전부 신규) |
| N1(a) | `AirtableFacade.java`(detach 3곳) |
| N1(b) | `osiv/LegacyOsivLazyLoadingTest.java`(신) |
| M4 | `application.yml`(주석 정정) |
| M6 | `ReviewBoostAutoMessageSendModel`·`KakaoAlimTalkSendHistory`·`db/reviewboost-v2-alimtalk.sql`·`InternalCrawlingReviewBoostController`(javadoc) |
| 도구 | `{TASK_DIR}/P4-scripts/`: `p4_nonnull_typed.py`·`OldPrimitiveProbe.java`·`targets*.tsv`·`probe38.txt`(신), `p4_nonnull.py`(대체됨 표기) |

---

## 검증

| 항목 | 결과 |
|---|---|
| booster 전체 스위트 | `classes=234 tests=901 failures=0 errors=0 skipped=8` (기준선 864/8 → +37) |
| gateway | `tests=84 failures=0 errors=0 skipped=0` |
| `Routes.java:34` | `new Route("legacy", MEDILAWYER, "/api/v1", ServiceId.LEGACY)` — 무변경(`git status apps/gateway-app` 클린) |
| 구 계약 실측 | 원시형 38/38 통과 · 참조형 285/285 THROW · ERROR 0 |
| 뮤테이션 | 98개 중 **6개 사살(6.1%)**. 수정 전 0/99. 전부 한 번에 넣은 대조군은 BUILD FAILED(수정 전 SUCCESSFUL). 주장 커버리지 8/8 KILLED |

불변 조건 준수: 컨테이너 조작 0 · 운영 RDS 접근 0 · **외부 실호출 0**(구 계약 실측은 로컬 JVM 에서
이미 빌드된 `.class` 를 읽은 것이고 네트워크를 쓰지 않는다) · 게이트웨이 라우트 무변경 ·
`medilawyer-boot` 무변경(읽기만) · 커밋 없음.

---

## 남은 것 (정직하게)

1. **P1/P2 AC3/AC4 재실행** — B1 요구 ④. 개발계 리플레이가 필요해 이번 라운드에서 못 했다.
   38곳이 P1~P4 에 걸쳐 있으므로 P5 전량 리플레이 전에 해당 샤드를 다시 돌려야 한다.
2. **AC3/AC6/AC7 판정 기준 개정**(M7) — 문서 작업이라 코드로 하지 않았다. "구 5xx → 신 2xx" 별도
   계수와 `AirtableFacade` 3경로 행 단위 차등이 필요하다.
3. **P5 실행 스택 확정**(B2 후속 / N2) — 게이트로 SDK 축은 막았지만 `RestClient` 축의 실 URL
   기본값 5개는 스텁 주입에 의존한다. `docker-compose.replay.yml`(`networks.internal: true`) 사용을
   못박을지 결정이 필요하다. 리플레이 스택 기동은 금지 조건이라 확인하지 못했다.
4. **카프카 컨슈머 분기 행위 테스트** — 지시받은 4개 위험 축 중 유일하게 못 채운 것이다.
   뮤턴트 40개 전부 생존이고, 토픽별 페이로드 픽스처가 필요해 분량이 크다. 다음 순위 1번.
5. **N6 잔여 본문 대조** — `AdminService`·`ScheduledService`·`EventSchedulerService`·`AdminFacade`·
   `PostFacade` 는 라인 단위 대조도 행위 테스트도 없다. 위 뮤테이션 표의 SURVIVED 나머지가
   이 구간이다.
6. **`/internal` 포트 노출**(M4) — 코드로 할 수 있는 것은 다 했고, 남은 것은 booster
   `0.0.0.0:18081` 을 게이트웨이 뒤로 넣거나 방화벽으로 닫는 배치 작업이다.

STATUS: REVISED
