# Round 1 — P4 교차 검토 (독립 리뷰어)

대상: `8fb454f2` (worktree `/Users/steve/steve/legalcare-renew-prodsync-wt`, 291파일 +16,047/−136)
권한: `review` — 코드는 한 줄도 고치지 않았다. 검토 종료 시점 두 저장소 모두 `git status` 클린 확인.
뮤테이션은 scratchpad 격리 사본(`booster-app` 15M 복사본)에서만 돌렸다.

먼저 결론부터. **P4 는 "이식 자체"의 완성도가 낮아서 막는 게 아니다.** 61경로 컨트롤러 계약은
실제로 잘 맞고, 문서는 이 파이프라인에서 본 것 중 가장 정직하다 — 작성자가 스스로 올린 OSIV·
`sesBaseUrl`·멱등키 3건은 전부 실재하는 문제였다. 막는 이유는 다른 데 있다. **이 변경을 지켜 줄
그물이 없다는 것이 측정으로 확정됐고, 그물이 없는 상태에서 이미 교차 단계(P1~P4) 전역에 걸친
계약 역전이 하나 들어가 있다.** 아래 B1~B3 이 그것이다.

---

## BLOCKER

### B1. `@LegacyNonNull` 을 원시형에 붙여 구 계약을 **뒤집었다** — 38곳, P1~P4 전 단계

가장 중요한 지적이다. 작성자의 헤드라인 검증 항목(`04-changes-P4.md:12` "초과0·누락0")이
**전제부터 틀렸다.**

Kotlin 의 기본값 없는 non-null 은 두 종류인데 Jackson 에서 동작이 다르다. 참조형은 던지지만
**원시형은 던지지 않고 JVM 기본값(0/false)이 들어간다.** 구 스택 그대로(kotlinc 1.9.25,
jackson-module-kotlin 2.17.2) 직접 컴파일해 재현했다:

```
MatchStoreReq: missing Long storeId       -> OK    MatchStoreReqLike(storeId=0, matchType=REGISTRATION)
MatchStoreReq: explicit null Long         -> OK    MatchStoreReqLike(storeId=0, matchType=R)
MatchStoreReq: missing String matchType   -> THROW MissingKotlinParameterException
GiftSaveReq:   missing Boolean isValid    -> OK    GiftSaveReqLike(isValid=false, name=x)
GiftSaveReq:   missing String name        -> THROW MissingKotlinParameterException
Score:         missing Int score          -> OK    ScoreLike(score=0)
```

그런데 이식본은 원시형을 **박싱해서** `@LegacyNonNull` 을 붙였고
(`common/jackson/LegacyNonNull.java:24-26`, `LegacyNonNullModule.java:124-129,190-206`),
`LegacyJacksonConfig.java:24-27` 이 이를 앱 전역 `JsonMapper` 에 등록한다.
결과는 **구 200 → 신 500 `SERVER-001`**. 방향이 AC3/AC4 가 잡는 방향이라 리플레이가 잡을 수는
있지만, 그 필드를 생략한 요청이 골든셋에 없으면 조용히 통과한다.

기계 전수로 센 결과 **박싱된 구 원시형에 붙은 `@LegacyNonNull` 이 38곳**이고, P4 단독이 아니라
**P1·P2·P3 이식분에 이미 깔려 있다**:

| 파일:라인 | 타입·필드 | 영향 경로 |
|---|---|---|
| `db/dto/appUserStore/MatchStoreReq.java:18` | `Long storeId` | P2 스토어 매칭 |
| `db/dto/team/TeamRegistrationReq.java:19` | `Long storeId` | P2 팀 등록 |
| `db/dto/organization/dto/OrganizationRegisterationReq.java:19` | `Long storeId` | P2 조직 등록 |
| `db/dto/teamAppUser/TeamAppUserRegistrationReq.java:17,19` | `Boolean permission*` | P2 |
| `db/dto/teamAppUser/TeamAppUserPermissionUpdateReq.java:19,21` | `Boolean permission*` | P2 |
| `db/dto/organizationAppUser/InviteOrganizationAppUserReq.java:34,36` | `Boolean permission*` | P2 |
| `db/dto/appUserChannelAccount/AppUserChannelAccountLoginReq.java:26` | `Long storeId` | P2 |
| `db/dto/store/dto/NewStoreChannelReq.java:18` | `Long storeId` | P2 |
| `db/dto/reviewBoost/GiftSaveReq.java:28` | `Boolean isValid` | **P4 boost1 기프트** |
| `db/dto/reviewBoost/MessageButtonReq.java:26` | `Integer order` | **P4 boost1 발송** |
| `db/dto/admin/OrganizationAppUserAddReq.java:26` | `Long appUserId` | P4 admin |
| `db/dto/admin/SeoHospitalUpdateReq.java:23` | `Boolean publicSeoEnabled` | P4 admin/seo |
| `db/dto/airtable/SaveThreadFromAirtableRequest.java:29,47` | `Long storeId`·`Boolean dbSave` | P4 airtable |
| `db/dto/airtable/SaveThread{Comment,Reply}FromAirtableRequest.java:34,36` | `Boolean dbSave` | P4 airtable |
| `db/dto/store/dto/InternalStoreRegistrationRequestModel.java:24` | `Long hospitalOriginId` | P4 `/internal` |
| `db/dto/kafka/*` (`NewPostAddReq:36,59,62`, `PostWordAddReq:29,31`, `ProducerTaskUpdate:43,45`, `ConsumerScheduleCrawlingTaskWithTargets:44`, `ConsumerThreadDisplayStatusModified:29`) | 카프카 페이로드 | P4 — 컨슈머가 메시지를 **버린다** |
| `db/dto/external/*` (`AiBoostCreationResult:28`, `NaverSearchResult:23,25,27`, `KakaoBusinessSearchResultMeta:38,41,43`) | 외부 응답 파싱 | P1/P2 — 구는 0 으로 진행, 신은 예외 |

가장 나쁜 것은 **테스트가 이 역전을 정답으로 못박아 놨다**는 점이다:

> `app/src/test/java/legalcare/medilawyer/LegacyNonNullContractTest.java:60-72`
> `@DisplayName("구가 Long 이던 자리(storeId)도 키가 없으면 거부한다 — 0 으로 진입하지 않는다")`

위 재현 결과가 보여주듯 구는 정확히 **0 으로 진입했다.** 이 테스트는 구 동작을 보존하는 게 아니라
구와 달라진 동작을 고정한다. 누가 나중에 고치려 하면 이 테스트가 막는다.

요구: ① Kotlin 원시형이었던 자리에서 `@LegacyNonNull` 을 걷어내고 ② `p4_nonnull.py` 게이트를
**타입 인지형**으로 고치고(참조형만 대상) ③ `LegacyNonNullContractTest:60-72` 를 구 동작 기준으로
뒤집고 ④ **P1/P2 의 AC3/AC4 그린 판정을 재실행**해야 한다 — 그 판정들은 이 38곳이 들어가기
전/후가 섞여 있어 현재 유효하지 않다.

### B2. boost1 실발송 차단이 `sesBaseUrl` 하나로 안 된다 — Gemini 로 **환자 리뷰 원문이 나간다**

작성자 주장("막는 건 `sesBaseUrl` 기본값 스텁 하나")의 앞부분은 맞다. `serviceMode` 가드 유무는
구와 정확히 일치한다(구 `BizMessageService.kt:54-56` 가드 있음 / `:95` `sendBoostAlimTalkSvc` 없음 /
`:154` `sendSmsSvc` 없음 → 신 `BizMessageService.java:108-110` 만 가드). **가드를 빠뜨린 게 아니다.**

문제는 "하나뿐"이 틀렸다는 것이다. 카테고리별로 추적한 결과:

| 경로 | 판정 | 근거 |
|---|---|---|
| 기프트 | 차단 — 호출부 자체가 없음 | boost1 기프트 5경로는 전부 DB. 벤더 클라이언트는 `modules/product` 에 있고 boost1 에서 도달 불가 |
| 알림톡 | 차단 | `BizMessageService.java:183-184` → `naverCloud.sesBaseUrl` → `config/notification.yml:55` 기본 `http://localhost:19999`, compose 가 stub 로 덮음 |
| SMS | 차단 | 동일 프로퍼티 |
| 이메일(SES) | 차단 — 호출부 없음 | `BoostFacade` 생성자에 메일 서비스 주입 없음 |
| **기타 아웃바운드** | **차단 안 됨** | 아래 |

`POST /api/v1/boost1/organizations/{o}/teams/{t}/messages/{messageId}/posts`
→ `BoostFacade.savePostFcd` → `ExternalClientService.makeBoosterReview`:

```java
// modules/legacy/.../client/external/ExternalClientService.java:414
Client geminiClient = Client.builder().apiKey(apiKey).build();
// :419
geminiClient.models.generateContent(aiModel, userInstruction, generateContentConfig);
```

**base URL 프로퍼티가 없다.** SDK(`google-genai:1.0.0`) 내부에 `https://generativelanguage.googleapis.com`
이 박혀 있고, SDK 가 읽는 유일한 오버라이드 `GOOGLE_GEMINI_BASE_URL` 은 이 저장소 어디에도
설정돼 있지 않다(전수 grep 0건 — 직접 확인). API 키가 자리표시자여도 **TLS 연결 + POST 는 나간다**;
Google 이 401 을 돌려줄 뿐이다. 그리고 그 바디에는 `ExternalClientService.java:461-467` 이
**병원명·진료내역·환자가 쓴 리뷰 원문**을 싣는다. 로그는 `LegacyLogMasking.text` 로 잘 가렸는데
정작 네트워크로는 원문이 나간다.

구도 같은 코드라 **이식 버그는 아니다**(구 `ExternalClientService.kt:378` 동일). 하지만 AC7⑤
"외부 실호출 0건"과 지시서의 봉인 규약을 P5 에서 깨는 것은 확실하다. 실제로 egress 를 막는 것은
`docker-compose.replay.yml:16-18` 의 `networks.legalcare.internal: true` **한 줄뿐이고**,
`deploy/dev/docker-compose.yml` 에는 `networks:` 블록이 아예 없다(직접 확인). P5 를 DELL 배포
스택에서 돌리면 그대로 나간다.

같은 축의 부수 지적: `ExternalSendSealingTest` 가 `sesBaseUrl`/`kakao-biz`/`naver-cloud` 는 게이트하는데
**Gemini 키가 목록에 없다** — 그래서 이게 새어 나왔다. 그리고 `application-dev.yml:196` 의
Slack 웹훅 기본값이 **실 웹훅**이라 `SPRING_PROFILES_ACTIVE=dev` 로 env 없이 뜨면 스택트레이스가
실제 Slack 으로 나간다(두 compose 모두 빈 값으로 덮고는 있다).

P5 진입 전 요구: `GOOGLE_GEMINI_BASE_URL=http://stub:8099` 주입 **또는** replay compose 사용을
확정하고, `ExternalSendSealingTest` 에 Gemini 키를 추가할 것.

### B3. 뮤테이션 실측 — **신규 테스트가 이식 로직을 전혀 물지 않는다**

작성자가 "다음 라운드 1순위"로 남긴 항목을 직접 돌렸다. 격리 사본에서 기준선부터 재현했다:

```
기준선:  classes=225  tests=864  failures=0  errors=0  skipped=8      (오케스트레이터 실측과 일치)
```

**음성 대조군** — P4 이식분에서 가장 큰 9개 파일의 **모든 메서드 본문 99개**를
`throw new UnsupportedOperationException("MUTANT")` 로 바꿨다. 대상은
`BoostFacade`(691줄, `sendMessagesFcd` 포함) · `KafkaConsumer`(652) · `ScheduledService`(576) ·
`AdminService`(314) · `InternalStoreService`(260) · `EventSchedulerService`(232) ·
`AdminFacade`(219) · `AirtableFacade`(216) · `PostFacade`.

```
뮤턴트:  classes=225  tests=864  failures=0  errors=0  skipped=8      ← 기준선과 동일. BUILD SUCCESSFUL
```

**99개 중 0개가 잡혔다.** 알림톡·SMS 발송 본체를 통째로 지워도, 카프카 컨슈머 652줄을 다 지워도,
스케줄러 576줄을 다 지워도 864개가 전부 통과한다.

**양성 대조군** — 테스트가 "덮는다고 주장하는 것"은 실제로 잡는지 따로 확인했다(각각 전체 스위트 실행):

| 뮤테이션 | 결과 | 잡은 테스트 |
|---|---|---|
| `/internal` 예외목록에 유령 항목 1개 추가 | **KILLED** | `LegacyInternalAllowlistTest` |
| `KAFKA_CONSUMER_AUTO_STARTUP` 기본값 `false`→`true` | **KILLED** | `LegacyRuntimeTogglesTest` |
| `LegacyScheduledJobs` 에 `matchIfMissing = true` 삽입 | **KILLED** | `LegacyRuntimeTogglesTest` |
| `BATCH_APP_USER_STORE_CRON` 30분→15분 | **KILLED** | `LegacyRuntimeTogglesTest` |
| `POST /api/v1/boost1/gifts` → `/giftsX` | **KILLED** | `LegacyP1EndpointMountTest` |
| **`MessagesSendReq` 에서 `@LegacyNonNull` 1개 제거** | **SURVIVED** | — |

즉 **+14 테스트는 구조 게이트로서는 제대로 작동한다**(킬스위치·경로문자열·예외목록은 확실히 문다).
문제는 그 바깥이다. 16,047줄의 **행위**에 대한 커버리지가 0 이고, 골든셋은 130 중 15개(11.5%)만
덮는다. 코드 대조가 유일한 방어선이라는 오케스트레이터의 판단이 측정으로 확인됐다 — 그런데 그
유일한 방어선이 B1 을 놓쳤다.

부수 지적: `@LegacyNonNull` **커버리지 게이트가 빌드에 없다.** `LegacyNonNullContractTest` 는 표본
몇 개의 배선만 보고, 전수 대조는 `P4-scripts/p4_nonnull.py` 가 하는데 이 스크립트는 저장소에 없고
`{TASK_DIR}` 에만 있다(직접 확인). CI 에서 안 돈다. 위 표의 SURVIVED 가 그 결과다.

---

## MAJOR

### M4. `/internal` permitAll — 판단은 옳으나 **근거 (a) 가 사실이 아니다**

작성자 근거 3개 중 (c)(막으면 P5 에 14건 회귀)는 맞고, 목록 자체도 정확하다. 컨트롤러 4개를
리플렉션 전수한 결과 **(메서드,경로) 14쌍이 빠짐 0·유령 0**, 와일드카드 없음 —
`LegacyInternalPaths.java:36-55` ↔ `Internal*Controller`. 필터도 fail-closed 로 잘 짰다
(`InternalAccessBlockFilter.java:267-320`, 디코드·슬래시정규화·매트릭스제거·대소문자).

문제는 (a) "게이트웨이에 `/internal` 라우트가 없어 외부 노출이 안 는다"이다. 게이트웨이 경유가
막히는 건 맞다(실측 `GET :18082/internal/reviewed/customers` → 404). 그런데 **booster 는
게이트웨이를 거치지 않고도 닿는다** — 개발계 실측:

```
docker inspect booster-api-dev → {"8080/tcp":[{"HostIp":"0.0.0.0","HostPort":"18081"}]}
ss -ltnp                       → LISTEN 0.0.0.0:18081 (booster) / *:41010 (구 legacy-service)
curl http://192.168.1.55:18081/actuator/health → 200
deploy/dev/docker-compose.yml:44-50 → booster-web: network_mode: host, SERVER_PORT 18081
```

그리고 **구 :41010 은 지금 이 순간 무인증으로 열려 있다**(읽기 프로브만 수행):

```
GET :41010/internal/reviewed/customers          → 200   (무인증)
GET :41010/internal/reviewed/auth-bypass/stores → 200   (무인증)
GET :41010/api/v1/stores                        → 401
```

그러니 "구와 같다"는 참이지만, 그 구의 자세가 이미 틀렸다. 사무실 LAN·VPN 사용자는 게이트웨이를
건너뛰고 직접 닿는다. **공인망 도달 여부는 확인 못 했다**(라우터에 접근하지 않았고, 저장소 문서에는
18081/41010 포워딩 근거가 없다).

특히 `POST /internal/reviewed/auth-bypass` 는 단순 쓰기가 아니라 **인가 부여 프리미티브**다.
`InternalStoreService.java:80-89` 가 `team_auth_bypass(appUserId, storeId)` 행을 넣고,
`StoreFacade.java:241-248` 이 그 행이 있으면 ML 인증을 **건너뛰고** SUCCESS 를 돌려준다.
무인증 호출자가 임의 사용자에게 임의 스토어의 채널인증 우회를 영구 부여할 수 있다. 멱등키도
행위자 기록도 없고 감사는 `log.info` 한 줄(`:82`)이다. 구도 같으므로 P4 회귀는 아니지만,
이 포트 뒤에서 가장 값나가는 표적이다.

또 하나 실행 가능한 지적: 근거 (b)("reviewed-app 이 9경로를 HTTP 로 부른다")가 양방향으로 부정확하다.
실제 호출은 **10경로**(`reviewed-app/.../LegacyServiceRestClientImpl.java:31` 1건 +
`.../adminweb/.../LegacyServiceRestClientImpl.java:44,53,62,71,80,90,100,109,119` 9건)이고,
반대로 **나머지 중 2개는 이미 인프로세스 `@Primary` 어댑터가 있어 HTTP 가 필요 없다** —
`ProductLegacyLocalAdapter`(`PUT .../request-setting`)와 `CrawlingLegacyLocalAdapter:55`
(`POST /internal/crawling/review-boost/stores/{storeId}/messages`). 후자는 **알림톡 발송 트리거**다.
목록에서 뺄 수 있는지 확인할 것.

### M5. 스케줄러·컨슈머 "기본 꺼짐"을 **런타임으로 실증할 수 없다** — 코드가 배포된 적이 없다

지시받은 "기동 로그·빈 목록으로 실증"을 시도했고, 실증이 불가능하다는 것을 실측으로 확인했다:

```
docker inspect legalcare-local-booster-web-5
  → ImageName=legalcare/booster:dev-20260806-delegate-contract
  → Image Created=2026-08-06T09:36:58Z   (P4 커밋은 2026-08-10 22:09)
docker exec … unzip -l /app.jar | grep -c "medilawyer/legacy/"   → 0
  (LegacyScheduledJobs·LegacyInternalPaths·legacy/kafka/KafkaConsumer·InternalAccessBlockFilter 전부 부재)
```

**개발계에 떠 있는 booster 는 P0~P4 이식분을 단 한 클래스도 갖고 있지 않다.** 따라서 "기본 꺼짐"의
근거는 현재 `LegacyRuntimeTogglesTest` 의 애노테이션 리플렉션뿐이다. 그 게이트가 **실제로 문다는
것은 위 B3 양성 대조군에서 내가 확인했으므로**(3개 뮤테이션 전부 KILLED) 선언 수준의 신뢰는 있다.
다만 `KafkaListenerEndpointRegistry.getListenerContainer(...).isRunning() == false` 와
`ScheduledTaskHolder` 실측은 **못 했다** — 테스트 자신도 그건 P5 몫이라고 적어 뒀다
(`LegacyRuntimeTogglesTest.java:28-31`). AC10 을 이 상태로 그린 처리하면 안 된다.

"구와 groupId 가 같아 켜면 파티션이 쪼개진다"는 주장은 **확인됨**:

```
kafka-consumer-groups --describe --group medilawyer-spring-boot --members
  consumer-…-1  /192.168.1.55  17 partitions
  consumer-…-2  /192.168.1.55   0
  consumer-…-3  /192.168.1.55   0
```
멤버 3개 전부 구(192.168.1.55), 17 토픽×1파티션이 1번 멤버에 몰려 있다. 신이 합류하면 리밸런스로
일부를 가져간다. 다만 표현을 정확히 하자면 "쪼개진다"보다 **"신이 가져간 토픽의 메시지는 신 PG 에
기록되어 원장이 이원화된다"**가 실제 피해다. 커밋 오프셋이 없는 토픽들은 LOG-END 가 0 이라
리플레이 폭주 위험은 없다(확인함).

### M6. 멱등키 길이 200 → 100 축소 — 살아 있는 계약 변경

`apis/internal/controller/InternalCrawlingReviewBoostController.java:41` 이
`modules/notification/.../ReviewBoostAutoMessageSendModel.java:27`(`@Size(max = 100)`)를 재사용하는데,
구 `apis/internal/controller/request/ReviewBoostAutoMessageSendModel.kt:9` 는 `@Size(max = 200)` 이다.
길이 101~200 멱등키는 **구 200 → 신 400**. 작성자가 javadoc(25-29줄)에 적어는 뒀지만 고치지 않았다.
크롤러가 실제로 부르는 경로다.

### M7. AC 판정 기준이 **한 방향**이라 OSIV 류를 구조적으로 못 잡는다

AC3 이 "구 200 → 신 비200 0건"으로 정의돼 있다. 그런데 OSIV 차이가 만드는 divergence는
"구 500 → 신 200"(구는 `LazyInitializationException`, 신은 조용히 성공) 방향으로도 나온다.
이 방향은 AC3 문구상 회귀가 아니다.

**이건 이론이 아니라 실물이 2건 있다** — N1(b) 의 L1·L2. 특히 L1
(`POST /api/v1/organizations/{o}/teams/{t}/users/{targetUserId}`)은 실 쓰기 경로인데, 구는 500,
신은 200 이다. P5 전량 리플레이는 이걸 **회귀로 세지 않는다.** 그리고 세지 않는 정도가 아니라,
`err_clusters.py` 원인별 판정에서 "구가 원래 내던 에러"로 분류돼 오히려 정상 취급될 수 있다.

B1 의 원시형 문제는 다행히 잡히는 방향이지만, AC 문구가 한 방향인 한 P5 를 그린으로 통과해도
동등이 입증되지 않는다. **AC3/AC6/AC7 을 양방향(상태코드 불일치 전건 + 쓰기 행 차등)으로
고쳐야 한다.** 최소한 "구 5xx → 신 2xx" 는 별도 카운트로 뽑아 눈으로 보게 할 것.

---

## MEDIUM

### N1. OSIV — **쓰기 축은 작성자 신고대로 P4 한정. 그러나 읽기 축에 P2/P3 라이브 divergence 2건이 더 있다**

지시받은 세 질문에 답한다.

**설정 사실**: 구 `medilawyer-boot/module-storage/core-storage/src/main/resources/core-storage.yml:10`
= `open-in-view: false`이고 `module-core/core-api/src/main/resources/application.yml:16` 이 이 파일을
import 한다. 프로파일 오버라이드 없음(전수 grep 1건). 신은 `application.yml:120` 주석대로 Boot
기본값 유지(설정 부재). Spring Boot **4.0.7**(`apps/booster-app/build.gradle:2`).

**(a) 더티체킹 쓰기 축 — P1·P2·P3 에는 없다.** `modules/legacy` 전체에서 "`@Transactional` 없는
메서드가 리포지토리로 읽은 엔티티에 setter 를 호출" 패턴을 기계 전수한 결과 후보 9건 중 실제 해당은
**`AirtableFacade` 3건뿐**이고, 나머지는 엔티티가 아니거나(`request`/`factory`/`headers`)
트랜잭션 안에서 불리는 도메인 헬퍼(`Store.addStoreChannel`, `OrganizationAppUser.addTeamAppUser`)다.
작성자가 3갈래라고 한 것이 맞다.

**(b) 지연로딩 읽기 축 — 여기에 작성자가 신고하지 않은 것이 2건 있다.** 그리고 이건 "플래그를
끄면 깨진다"가 아니라 **지금 구와 신이 이미 다르게 동작한다**는 뜻이다. 구는 OSIV=false 라서
트랜잭션 밖 지연 traversal 이 `LazyInitializationException` → 500 이고, 신은 OSIV=true 라
조용히 200 을 낸다.

| # | 경로 | 신 코드 | 구 원본 — `@Transactional` 유무 | 쿼리 fetch join |
|---|---|---|---|---|
| L1 | `POST /api/v1/organizations/{o}/teams/{t}/users/{targetUserId}` (P2, 실 쓰기 경로) | `TeamAppUserFacade.java:35` `registerAppUserFcd` — **없음**, `:43-44` 가 LAZY `getOrganization()`·`getAppUser()` traversal, `TeamAppUserRegistrationRes.java:32-37` 이 `getTeam().getIdentifier()` | 구 `TeamAppUserFacade.kt:23-24` — `@Deprecated` 만, **`@Transactional` 없음**(직접 확인) | 구 `OrganizationRepositoryImpl.kt:48-53` `getByOrganizationIdAndAppUserIdAndUsable` — **fetch join 0개**(직접 확인) |
| L2 | `POST /api/v1/jwt/create` (P3) | `JwtController.java:59` `createJwtReq` — **없음**, `JwtProvider.java:184` 가 `getOrganization().getIdentifier()` | 구 `JwtController.kt:23-25` — `//Todo. 삭제` + `@PostMapping` 만, **`@Transactional` 없음**(직접 확인) | 구 `OrganizationRepositoryImpl.kt:23-29` — `appUser`·`teamAppUserList`·`team`·`store` 는 fetch join 하는데 **`organization` 은 안 한다**(직접 확인) |

즉 구는 이 두 경로에서 500 을 냈을 것이고 신은 200 을 낸다. 방향이 **"구 500 → 신 200"** 이라
아래 M7 이 지적한 대로 AC3 문구("구 200 → 신 비200 0건")로는 **구조적으로 잡히지 않는다.**
L2 는 `//Todo. 삭제` 가 붙은 개발용 토큰 발급 훅(`JwtController.java:51` 에 하드코딩
`FIXED_EMAIL = "jay@legalcare.ai"`)이라 긴급도가 낮지만, **L1 은 팀에 사용자를 등록하는 실 쓰기 경로**다.

한 가지 유보: 체인 중 "필드 접근(`@Id` 가 필드에 있고 Lombok `@Getter`) 엔티티는 프록시 getter
호출 시 초기화된다"는 마지막 단계는 **정적 추론이고 실행으로 확인하지 못했다**
(`Organization.java:34-36`, `Team.java:37-39`, `AppUser.java:43-46` 에서 접근 타입만 확인).
Hibernate 의 표준 동작이지만, L1 에 통합테스트 하나를 붙이면 바로 결판난다 — 그게 요구사항이다.

**(c) 전역 설정을 바꾸면**: `open-in-view: false` 로 구와 맞추려는 시도는 **하지 말 것.**
`modules/user` 에서 **9개 엔드포인트(공개 6개 포함, `/user/`·`/user/stores` 등)가 확실히 깨지고**,
`modules/legacy` 자신도 위 L1·L2 가 그제서야 구와 같은 500 을 내기 시작한다. 나머지 8모듈
(post·crawling·notification·medicontents·external·product·certification·chrono·build)은 확인된
파손 0건이다. 이 조사는 정적 분석이며 **`open-in-view: false` 로 실제 기동해 요청을 태워 보지는
않았다.** 권고는 그대로 범위를 좁히는 쪽이다 — `AirtableFacade` 3메서드는 구처럼 명시적 no-op
으로 만들고(1:1 원칙), L1 에는 `@Transactional` 을 붙이거나 구 동작 그대로 500 을 유지할지 결정하고,
P5 에서 이 경로들만 **행 단위 + 양방향 상태코드** 로 측정할 것.

**어느 테이블에 몇 행인가**:

| 메서드 | 테이블 | 갈아끼우는 컬럼 | 요청당 |
|---|---|---|---|
| `AirtableFacade.java:75` `saveThreadFromAirtable` | `thread` (`Threads.java:34`) | title, content, author, threadUrl, authorAt, captureImg (`:98-103`) | 1행 |
| `AirtableFacade.java:134` `saveThreadCommentFromAirtable` | `thread_comment` (`ThreadComment.java:30`) | content, author, authorAt (`:156-158`) | 1행 |
| `AirtableFacade.java:175` `saveThreadReplyFromAirtable` | `thread_reply` (`ThreadReply.java:28`) | content, author, authorAt (`:199-201`) | 1행 |

구는 UPDATE 0건, 신은 기존 행이 있을 때 1행 UPDATE. 경로는 `POST /api/v1/airtable/save*` 3개.

**전역 설정을 바꾸면**: `open-in-view: false` 를 전역에 주면 legacy 뿐 아니라 기존 9모듈이 전부
영향을 받는다. 이 축의 정밀 조사는 **완료하지 못했다**(별도 조사를 걸었으나 회신 전에 검토를 마감했다).
그래서 **전역 설정 변경은 권하지 않는다.** 권고는 범위를 좁힌 쪽이다 — `AirtableFacade` 3메서드에
`@Transactional(readOnly = true)` 로 읽기 경계를 닫거나, 세 갈래를 구처럼 명시적 no-op 으로
만들거나(1:1 원칙상 후자가 맞다), 최소한 P5 에서 이 3경로만 **행 단위 차등**으로 측정할 것.
심각도는 HIGH 가 아니라 **MEDIUM** 으로 본다 — 쓰는 값이 요청 바디와 같은 값이라 데이터가
오염되는 게 아니라 "구엔 없던 갱신이 생긴다"에 그치고, 돈·발송과 무관한 콘텐츠 테이블이다.

### N2. 지시서가 봉인이라 부르는 것 중 **실 외부 URL 이 기본값인 게 5개** (P1/P2 유래, P4 아님)

`config/legacy.yml:104-109` — `openapi.naver.com`, `dapi.kakao.com`, `www.apis.modoodoc.com`,
`places.googleapis.com`, 그리고 실 AWS Lambda URL. 자격증명이 비어 있어 4xx 로 끝난다는 게
작성자 논리인데(`:66-67`), **4xx 도 실호출이다.** AC7⑤ "외부 실호출 0건"을 문자 그대로 지키려면
B2 와 같이 다뤄야 한다. P4 도입은 아니므로 P4 승인 조건으로 걸지는 않는다.

---

## MINOR / 확인 결과 문제 없음

- **N3. 범위 이탈 판정 — 이탈 아님, 다만 문서가 축소 보고했다.** `@LegacyNonNull` 보강은
  "2건"이 아니라 **4클래스 14필드**다(`DisplayName` 2, `GoogleMapsLinks` 5, `GoogleSearchResultItem` 5,
  `NaverPlaceItem` 2). 구 원본(`GoogleSearhResultItem.kt:3-25`, `NaverSearchResponse.kt:22-28`)과
  대조한 결과 **14곳 전부 기본값 없는 non-null 참조형이 맞다** — 즉 보강 자체는 정확하고
  구에 가까워지는 방향이다. 다만 P1/P2 경로의 역직렬화 엄격도를 바꾸는 변경이므로
  **P1/P2 AC 재실행 대상**이다(B1 과 같은 이유). 그 외 P1/P2 파일 수정분(
  `AppUserStoreService.java:118`, `TeamAppUserService.java:79`, `PostWordService.java:57` 등)은
  P4 가 필요로 하는 메서드 추가라 정상이다.
- **N4. 개인정보 로깅 — 문제 없음.** P4 신규 로그는 `LegacyLogMasking`(`common/log/LegacyLogMasking.java`)을
  일관되게 탄다(`ScheduledService.java:380,395,427,495,509`, `BizMessageService.java:288,382`,
  `KafkaConsumer.java:243,264`). 구가 원문을 찍던 자리를 가린 것이라 방향도 안전하다.
  단 B2 대로 **네트워크로는 원문이 나간다** — 로그만 가린 셈이다.
- **N5. 밑줄 필드·Jackson 키 순서·`is` 접두 불리언 — 전수 확인, 이상 없음.** 구 전체에서 밑줄
  프로퍼티는 `MlScreenshotRes.kt:13 s3_url` 하나뿐이고 `db/dto/ml/MlScreenshotRes.java:40` 에
  `@JsonProperty("s3_url")` 로 보존됐다. `@JsonPropertyOrder` 는 이식 DTO 전건이 Kotlin 생성자
  선언 순서와 일치. `isValid`/`isUsed`/`isInsulting` 등은 `@JsonProperty` 로 고정돼 있고,
  구 `KotlinModule` 기본 설정에서 `{"isValid":...}` 로 나가는 것이 맞다.
- **N6. 컨트롤러 계약 61경로 — 불일치 0.** URL·HTTP 메서드·`@RequestParam` 이름/required/기본값·
  경로변수명·에러코드·`@Transactional`(`BoostFacade` 18/18, `readOnly` 포함)·카프카 토픽/직렬화/
  `groupId`/`TYPE_MAPPINGS`/컨슈머 `earliest`·concurrency 3·`FixedBackOff(1000,1)` 전부 일치.
  다만 아래 파일들은 **본문 라인 단위 대조를 끝내지 못했다**(→ 잔여 리스크로 남긴다):
  `AdminService`(314) · `AdminFacade`(219) · `AirtableFacade`(216) · `InternalStoreService`(260) ·
  `ScheduledService`(576) · `EventSchedulerService`(240) · `KafkaConsumer` 핸들러 3~6 ·
  `ExternalClientService` · `StoreChannelHelper/Service` · `BizMessageService` 본문.
  **B3 때문에 이 잔여분에는 어떤 자동 방어도 없다.**
- **N7. `e.printStackTrace()` → `log.error`** (`EventSchedulerController.java:143,161,179`).
  응답·상태·메시지 동일, 라우팅만 다름. 수용.
- **N8. `{"storeIds": null}` 에러 바디 차이** — `GooglePlaceIdStatusRequestModel.java:32`.
  둘 다 500 `SERVER-001`, `errors[0].message` 만 다름. 작성자가 이미 문서화. 수용.
- **N9. boost1 은 15경로가 아니라 16경로**다(구 `BoostController.kt` 의 `@*Mapping` 16개).
  16개 전부 이식됐다. 지시서 숫자 쪽을 고칠 것.

---

## AC6 판정 — 요구받은 대로 답한다

**"booster→legacy HTTP 호출 0건"**: 코드상 달성됐다. Feign 2개가 `@Primary` 로컬 어댑터로
대체됐고(`CrawlingLegacyLocalAdapter`, `ProductLegacyLocalAdapter`), `LegacyInProcessConversionTest`
가 `LEGACY-SERVICE` Feign 이 정확히 2개이고 둘 다 `primary=false` 임을 컨텍스트에서 센다.
**단 이건 booster 프로세스 안의 이야기다.** 다른 JVM(`reviewed-app` 10경로)은 여전히 HTTP 로
`/internal` 을 부르고, 그건 AC6 범위 밖이다. 실측 검증은 M5 대로 **불가능하다** — 코드가 배포된 적이 없다.

**"기프트·발송 원장 이중기록 0건"**: 이 상태로는 **판정할 수 없다.** 원장에 멱등키가 없는 것은
구 그대로가 맞다(`BoostFacade.sendMessagesFcd:291-334` — `Message` 행을 `identifier = ULID` 로
매번 새로 만든다. 요청 2회 = `boost_message` 2행 + 알림톡 2통). 그래서 "이중기록 0건"은
**코드 속성이 아니라 트래픽 속성**이다. 리플레이가 같은 요청을 두 번 쏘거나 재시도하면 무조건 2행이
쌓이고, 그건 회귀가 아니라 설계다. 판정 방법을 이렇게 바꿔야 한다:

1. AC6 의 "이중기록 0건"을 **"구 PG 행 증가분 == 신 PG 행 증가분"** 으로 재정의한다
   (0건이 아니라 *일치*). 절대량 0 은 구에서도 성립하지 않는다.
2. 병행 가동 중이므로 `boost_message`·`boost_message_delivery`·`customer_gift` 를 리플레이
   전/후로 **양쪽 PG 에서 각각 카운트**해 델타를 비교한다(AC10 이 요구하는 것과 같은 방식).
3. 멱등키 부재 자체는 **P4 결함이 아니므로 이번 승인 조건에서 뺀다.** 다만
   `review/LESSONS.md` 와 P6 백로그에 "원장 멱등키 없음 — 재시도 안전하지 않음"으로 남길 것.
   fintech-safety 기준으로는 결함이지만 1:1 이식 제약이 우선한다.

---

## 요구 사항 (P5 진입 전)

1. **B1** — 원시형 38곳 `@LegacyNonNull` 제거 · 게이트 타입 인지화 ·
   `LegacyNonNullContractTest:60-72` 반전 · **P1/P2 AC3/AC4 재실행**
2. **B2** — Gemini egress 봉인(`GOOGLE_GEMINI_BASE_URL` 주입 또는 replay compose 확정) ·
   `ExternalSendSealingTest` 에 Gemini 키 추가 · P5 실행 스택을 문서에 못박을 것
3. **B3** — 이식 로직에 대한 행위 테스트를 최소한 **돈·발송·인가 경로**(boost1 발송/기프트,
   `/internal/reviewed/auth-bypass`, 카프카 컨슈머 분기)에는 붙일 것. 전량은 요구하지 않는다.
   그리고 `p4_nonnull.py` 를 빌드 안으로 들일 것
4. **M4** — `LegacyInternalPaths` 에서 로컬 어댑터가 있는 2건 제거 가능 여부 확인 ·
   booster 포트 바인딩(0.0.0.0:18081)을 게이트웨이 뒤로 넣거나 방화벽 처리 ·
   `application.yml:317-321` 의 "두 필터가 충돌하지 않는다" 주석 정정
5. **M6** — 멱등키 `@Size` 200 복원
6. **M7** — AC3/AC6/AC7 판정을 양방향 + 쓰기 행 차등으로 개정. "구 5xx → 신 2xx" 별도 카운트
7. **N1** — `AirtableFacade` 3경로 행 단위 차등 + **L1(`TeamAppUserFacade.registerAppUserFcd`)에
   통합테스트 1개**로 지연로딩 divergence 를 실행 확인. **전역 OSIV 변경은 하지 말 것**
   (`modules/user` 9경로가 깨진다)

## 확인하지 못한 것

- 18081/41010 의 **공인망 도달 여부** (라우터 미접근, 저장소 문서에 포워딩 근거 없음)
- `docker-compose.replay.yml` 의 `internal: true` 가 베이스 파일에서 상속한 `18081:18080` 퍼블리시를
  실제로 무력화하는지 (replay 스택 미기동 — 기동 금지 규칙 준수)
- N1(b) 의 마지막 단계 — **필드 접근 엔티티의 프록시 getter 가 실제로 초기화를 트리거하는지**.
  접근 타입은 확인했으나 실행으로 확인하지 못했다. L1 통합테스트 1개로 결판난다
- OSIV 조사 전체가 **정적 분석**이다. `open-in-view: false` 로 기동해 요청을 태워 보지 않았다
- `AdminService`·`ScheduledService`·`InternalStoreService`·`EventSchedulerService`·`KafkaConsumer`
  핸들러 3~6 · `ExternalClientService` · `StoreChannelHelper/Service` · `BizMessageService` 의
  **본문 라인 단위 구/신 대조** (N6)
- 필터의 런타임 동작 — 배포 이미지에 `InternalAccessBlockFilter` 자체가 없어 실환경 검증 불가

---

P4 는 이 태스크에서 가장 잘 쓰인 단계다. 61경로 계약이 정확하고, 킬스위치는 실제로 물며(직접 검증),
작성자가 자진 신고한 3건은 전부 진짜였다. 그럼에도 막는 이유는 B1 하나만으로 충분하다 —
**구 계약을 뒤집는 변경이 P1~P4 전역 38곳에 들어가 있고, 그것을 정답으로 못박는 테스트까지 있으며,
그 사이 유일한 방어선이었어야 할 테스트 스위트는 이식 로직을 전혀 물지 않는다는 것이 측정으로
확정됐다.** B1·B2 를 안 고치고 P5 로 가면 전량 리플레이는 "그린"을 내면서 실제로는 계약 역전을
통과시키고 환자 리뷰 원문을 Google 로 흘린다.

VERDICT: REQUEST_CHANGES
