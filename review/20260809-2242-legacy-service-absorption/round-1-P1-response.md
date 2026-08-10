# Round 1 대응 — P1(읽기, R3)

리뷰 잘 읽었다. A2·A3·A4·B3 은 전부 반영했고 실제로 코드가 바뀌었다. A1 은 **한 군데 반박이 있다** — 지적한
"지금 키 순서가 알파벳으로 깨져 있다"는 실측에서 재현되지 않았다. 다만 그 아래 깔린 우려(순서가 못 박혀 있지
않다)는 옳고, 요구한 대로 전수를 했다. 아래에 재현 명령과 원문 출력을 남긴다.

기준선 재실측: `./gradlew test` 후 `**/build/test-results/test/*.xml` 의 `tests` 합계 = **703**(초안의 702는
틀렸다). 대응 후 **723/0**. gateway 76/0, `Routes.java:34` = `ServiceId.LEGACY` 불변.

---

## 1. A1 (CRITICAL) 중첩 DTO 키 순서 — **반박 1 + 반영(전수)**

### 반박: "지금 알파벳순으로 나간다"는 재현되지 않았다

앱의 실제 `HttpMessageConverter` 로 측정했다(`LegacyP1ResponseShapeTest` 와 같은 방식, 임시 테스트):

```
MEASURE-DisplayName     = {"text":"텍스트","languageCode":"ko"}
MEASURE-GoogleMapsLinks = {"directionsUri":"d","placeUri":"p","writeAReviewUri":"w","reviewsUri":"r","photosUri":"ph"}
MEASURE-ApiResponse     = {"status":"200","message":"메시지","data":"데이터"}
```

셋 다 **선언 순서**다. 즉 `GET /api/v1/store-channel/search/google` 응답은 현재도 구와 같은 키 순서로 나간다.

기제를 따로 재 봤다. 같은 컨버터로 생성자 모양만 바꾼 세 클래스:

```
MEASURE2-TwoLombokCtor   = {"text":"t","languageCode":"ko"}        무인자 + 전필드 생성자   -> 선언순
MEASURE2-TwoExplicitCtor = {"languageCode":"ko","text":"t"}        명시 생성자 2개(부분+전체) -> 알파벳순
MEASURE2-OneExplicitCtor = {"status":"200","message":"m","data":"d"}  명시 생성자 1개(전필드)  -> 선언순
```

규칙은 "생성자가 2개면 알파벳"이 아니라 **"전필드 생성자가 유일한 non-default 생성자가 아니면 알파벳"** 이다.
`@NoArgsConstructor` + `@AllArgsConstructor` 는 무인자가 default 로 빠지고 전필드 하나만 남아 암묵 creator 로
잡힌다 — `DisplayName`/`GoogleMapsLinks` 가 그 경우다. `ErrorResponse`(`:13-15` 실측 기록)가 알파벳이었던 이유도
생성자가 2개여서가 아니라 **4-arg 전필드와 3-arg 부분 생성자가 같이 있어 creator 를 고르지 못한** 것이다.
`SORT_PROPERTIES_ALPHABETICALLY` 기본값 이야기는 이 관측을 설명하지 못한다.

**그래서 "리플레이가 못 잡는 200 회귀가 지금 있다"는 부분은 사실이 아니다.** 다만 —

### 반영: 순서가 두 개의 우연 위에 서 있다는 지적은 옳다 → 전수로 못박고 게이트를 세웠다

지금 맞는 이유는 (a) `-parameters` 컴파일 옵션(`build.gradle:23`)과 (b) 생성자 개수, 둘 다 우연이다.
C3 에서 리뷰어 본인이 `ApiResponse` 를 두고 같은 지적을 했는데, 그건 **모든 핀 없는 타입에 해당한다.**

요구한 대로 눈이 아니라 기계로 훑었다. `modules/legacy` 전 소스를 괄호균형 파서로 파싱해 타입 85개를 뽑고,
①생성자 개수(명시+lombok 유도) ②`@JsonPropertyOrder` 유무 ③필드 선언순 ④boolean `isXxx` + `@JsonProperty`
위치를 전수 대조했다. 결과 핀이 없던 13개 전부에 붙였다:

| 타입 | 위치 | 응답 도달 여부 |
|---|---|---|
| `DisplayName` | `db/dto/external/DisplayName.java:9` | ✅ google 검색 `data.list[].displayName` |
| `GoogleMapsLinks` | `db/dto/external/GoogleMapsLinks.java:9` | ✅ 같은 응답의 `googleMapsLinks` |
| `ApiResponse` | `common/response/ApiResponse.java:16` | ✅ **11경로 전부의 봉투** |
| `StatisticsRes` | `db/dto/postWord/StatisticsRes.java:9` | ✅ post-word 통계 |
| `StatisticsWordCountRes` | `db/dto/postWord/StatisticsWordCountRes.java:12` | ❌ QueryDSL 프로젝션 전용 |
| `GoogleSearchResult`·`KakaoBusinessSearchResult`·`KakaoBusinessSearchResultMeta`·`ModoodocBusinessSearchResult`·`NaverSearchResult` | `db/dto/external/*` | ❌ 외부 API 인바운드 |
| `PostReplyAddReq`·`PostListReq`·`StoreChannelReq` | `db/dto/{postReply,post/dto,storeChannel}` | ❌ 요청측 |

핀 목록은 구 Kotlin 선언 순서와 대조해 넣었다(`GoogleSearhResultItem.kt:3-6,8-14`, `NaverSearchResult.kt`,
`ApiResponse.kt` 등 원문 확인).

**같은 실수가 다시 나지 않게 게이트를 만들었다** — `JsonPropertyOrderCoverageTest`
(`modules/legacy/src/test/.../db/dto/`). `db/dto`·`common/response`·`common/exception` 을 클래스패스 스캔해
(중첩 타입 포함) 모든 직렬화 대상에 대해 "핀이 있고 그 목록이 필드 선언순과 정확히 같다"를 단언한다.
표본이 0건이면 위장 통과가 되므로 `hasSizeGreaterThanOrEqualTo(30)` 도 함께 건다.

뮤테이션으로 무는지 확인했다:

```
DisplayName 의 핀 제거 + PostListData 핀에서 isInsulting/isDefamatory 순서 뒤집기
-> MUTATION 결과: tests= 1 failures= 1
   {"...DisplayName"="@JsonPropertyOrder 없음. 기대 목록 = [text, languageCode]",
    "...PostListData"="핀=[...isDefamatory, isInsulting...] / 필드선언순=[...isInsulting, isDefamatory...]"}
```

바이트 단언도 추가했다 — `LegacyP1ResponseShapeTest.googleChannelRes_nestedTypes_keepLegacyKeyOrder`
(중첩 2개까지 문자 단위 고정) · `apiResponseEnvelope_keyOrderPinned`.

---

## 2. A2 (MAJOR) 인증 사용자 조회 범위 — **반영**

지적이 맞다. 구 `JwtAuthenticationFilter.kt:41-83` 은 `accessToken != null` 이면 경로 불문 조회하고,
실패하면 `resolver.resolveException` 으로 빠져 **`filterChain.doFilter` 에 도달하지 않는다**(:94-97).
인자 리졸버는 `CustomUserDetails` 파라미터를 선언한 2경로에서만 도는 게 맞다.

`LegacyAppUserResolutionInterceptor` 를 만들어 `/api/v1/**` 전체에서 조회하도록 되돌렸다.
결과를 요청 애트리뷰트에 담고 리졸버가 재사용하므로 **조회는 요청당 1회**(구와 동일).

**필터가 아니라 인터셉터인 이유**를 남겨 둔다. 구처럼 필터에서 던지면 `resolveException(req, res, null, e)` 가
되는데, `ExceptionHandlerExceptionResolver` 는 `handlerType == null` 일 때
`HandlerTypePredicate.test(null)` 로 묻고 셀렉터가 걸린 advice 는 거기서 false 를 돌려준다. `LegacyExceptionAdvice`
는 `basePackages = "legalcare.medilawyer.legacy"` 로 가둬 놨으므로 **필터에서 던지면 구 봉투가 안 나온다.**
구가 통했던 건 전역 `GlobalExceptionHandler`(셀렉터 없음)였기 때문이다. 인터셉터의 `preHandle` 은 핸들러가
매핑된 뒤라 예외가 그 핸들러와 함께 `processHandlerException` 으로 가고 advice 가 정상적으로 붙는다.

남는 차이 1개는 정직하게 적는다: 구는 핸들러 매핑 **이전**이라 존재하지 않는 경로에서도 400 을 냈고, 신은
404 다. P1 이식 11경로에서는 차이가 없고(아직 안 옮긴 경로는 어차피 404), P4 에서 147경로가 다 들어오면 사라진다.

게이트는 `LegacyAppUserResolutionInterceptorTest` 4케이스. 뮤테이션(인터셉터 무력화)으로 확인:

```
MUTATION: tests= 4 failures= 1
  CustomUserDetails 파라미터가 없는 경로도 app_user 부재면 구처럼 400 USER-002 다
    -> java.lang.AssertionError: Status expected:<400> but was:<200>
```

지적이 말한 그 200 이 그대로 재현됐다.

---

## 3. A3 (MAJOR) 카프카 와이어 포맷 — **반영. 소비자도 실측했다**

리뷰어가 미확정으로 남긴 소비자를 찾았다. 이 토픽의 소비자는 구 레포 밖의
`/Users/steve/steve/legal-care/event-coordinator`(Node.js/kafkajs, `src/kafka/topics.ts:9` 에 토픽 등록)다.

```
consumer.ts:1016-1021
  const { offset, headers, value } = message;
  const messageValue = JSON.parse(value.toString());
  await eventMessageHandler(topic, messageValue);
```

`headers` 를 구조분해만 하고 **쓰지 않는다**. 레포 전수 grep 에서 `__TypeId__` **0건**(다른 `headers:` 출현
8곳은 전부 Airtable 로 보내는 axios 요청 헤더다). 즉 헤더 유무가 이 소비자를 깨지 않는다.

그럼에도 **①이 아니라 ②(타입 매핑)를 골랐다.** 근거는 기준이 1:1 이라는 것 — 구는 헤더를 붙여 보냈고,
나중에 타입 매핑으로 소비하는 소비자가 붙어도 구와 같은 값을 봐야 한다. `LegacyKafkaProducerConfig` 를 모듈
안에 세우고 구 `KafkaProducerConfig.kt:19-34` 를 그대로 재현했다:

| 항목 | 구 | 신(초안, app QueueConfig 차용) | 신(지금) |
|---|---|---|---|
| `__TypeId__` | 켜짐(기본값) · 구 FQCN | **꺼짐** | 켜짐 + `TYPE_MAPPINGS` 로 **구 FQCN** |
| `MAX_REQUEST_SIZE` | 524,288,000 | **기본 1MB** | 524,288,000 |
| `security.protocol` | PLAINTEXT | 미설정 | PLAINTEXT |

브로커 없이 바이트를 실측하는 게이트를 붙였다(`LegacyKafkaProducerConfigTest`, 3케이스). 프로듀서 팩토리에
넘기는 **같은 설정 맵**으로 `JsonSerializer` 를 세워 `serialize(topic, headers, value)` 를 직접 부른다:

- `__TypeId__` == `legalcare.medilawyer.db.dto.kafka.ProducerUserPostReplyCreate` (이식으로 바뀐 패키지가 새지 않는다)
- 바디 12키 이름·순서 문자 단위 고정
- `max.request.size == 524288000`

참고로 `org.springframework.kafka.support.serializer.JsonSerializer` 는 spring-kafka 4.0.6 에서
deprecated-for-removal 이다. 구가 이 클래스(Jackson 2)를 썼고 `app/QueueConfig`·`ExternalKafkaProducerConfig`
도 아직 이걸 쓰므로 이번엔 맞췄다. 교체는 저장소 전체가 한 번에 움직여야 하는 별건이다.

---

## 4. A4 (MAJOR) 역방향 의존 — **반영**

`KafkaProducer.java:31` 의 `@Qualifier("queueKafkaTemplate")`(app 계층 빈) → `@Qualifier("legacyKafkaTemplate")`
(모듈 자체 빈)로 바꿨다. 지적한 대로 `modules/post/.../PostKafkaProducerConfig` 가 선례이고 그 클래스 주석의
결정 규칙("`QueueConfig` 의 기존 팩토리와 다르면 새 프로듀서 팩토리+KafkaTemplate 빈을 추가")을 그대로 탔다.
`ProducerFactory<String,Object>` 가 3개가 되지만 기존 두 개와 같은 **파라미터명 기반 해석**을 쓰므로 모호성이
없다 — 전체 스위트 723/0 과 `duplicateClassCheck` 통과로 확인했다.

---

## 5. A5 (MAJOR) ML 응답 파싱 관대함 — **반영. 두 겹으로 닫고 각각 뮤테이션했다**

지적대로 `RestClientSupport.java:164` 의 `READ_UNKNOWN_ENUM_VALUES_AS_NULL=true` 는 Feign 패리티용이고
구 legacy 의 `RestTemplate`/`WebClient` 가 쓰던 `ObjectConfig.kt:12-19` 에는 없다. 그런데 파고 보니 **enum 만이
아니었다** — `MlAiRecommendRes.replyAi` 처럼 구 Kotlin 에서 non-null 이던 필드가 신에서는 조용히 null 이 된다는
쪽(리뷰어가 §A5 두 번째 문단에 적은 것)이 오히려 더 넓다. 둘 다 닫았다.

- **겹1(DTO)** — `MlAiRecommendRes`·`MlPostReplyRegistraionRes` 를 `@NoArgsConstructor`+setter 에서
  `@JsonCreator` + `@JsonProperty(required=true)` + `@JsonSetter(nulls=FAIL)` 로 바꿨다. 전자가 "키 누락",
  후자가 "명시적 null" 을 막는다(Kotlin non-null 파라미터는 둘 다 한 번에 막는다).
- **겹2(컨버터)** — `LegacyRestClients` 가 `RestClientSupport.builder()` 의 JSON 슬롯만 구 매퍼로 덮는다.
  `:core` 를 고치지 않은 이유는 거기에 다른 9모듈의 외부 클라이언트가 전부 얹혀 있어서다. 상태 핸들러·요청
  팩토리·재시도는 그대로 물려받는다 — 그쪽을 바꾸면 예외 타입이 달라지고 그 메시지가 구 응답 바디
  (`errorMessage`)로 흘러 들어간다.

처음엔 한 테스트로 묶었는데 **컨버터 교체를 통째로 지워도 7건이 다 통과했다** — 겹1이 먼저 걸려 겹2를 가리고
있었다. 그래서 겹을 나누고 각각 뮤테이션했다:

```
MUTATION(컨버터 교체 제거):  tests=7 failures=1
  FAIL: LegacyRestClients 로 만들면 같은 응답이 예외가 된다(= 구 동작)
MUTATION(DTO 를 지적 전 상태로): tests=7 failures=2
  FAIL: 구에서 non-null 이던 필드가 응답에 없으면 예외다
  FAIL: 구에서 non-null 이던 필드가 명시적 null 이어도 예외다
```

겹2를 재려면 nullable enum 필드가 필요한데 legacy 인바운드 DTO 에는 그런 필드가 **0개**라(전부 non-null)
테스트 안에 프로브 타입을 만들어 잰다. 정직하게 적으면 **겹2의 오늘 관측 가능한 효과는 0 이다** — 겹1이 항상
먼저 걸린다. 그래도 두는 이유는 이것이 모듈의 정책 차이이고, P2~P4 에서 nullable enum DTO 가 들어오는 순간
같은 회귀가 조용히 다시 열리기 때문이다.

`ExternalClientService` 도 같은 진입점으로 옮겼다. 그쪽 응답 DTO 에는 enum 이 없어 동작 차이는 없지만,
모듈 안에서 HTTP 클라이언트 조립 방식이 갈라지지 않게 하는 편이 낫다고 봤다.

---

## 6. B3 개인정보 로깅 — **판정 수용. 전수로 훑고 마스킹**

"1:1 이라 못 고친다"는 논리를 접는다. 리뷰어 말이 맞다 — 1:1 이 보호하는 것은 클래스·메서드·필드명·로직·URL 이고
AC 가 판정하는 것은 바디·상태·에러코드이며, **로그는 그 어디에도 없다.** 마스킹의 AC3 영향은 0 이다.

"하나만 올리면 나머지는 검토됐다로 읽힌다"도 맞다. `modules/legacy` 의 `log.*` 호출 **전수**(grep 25건)를
훑어 분류했다.

**가린 것 — 순수 로그 4곳**

| 위치 | 원문에 실리던 것 | 조치 |
|---|---|---|
| `ProducerUserPostReplyCreate.toString()` → `KafkaProducer.java:43` | email · appUserName(실명) · **phone** · 답글 본문 | 타입의 `toString()` 을 마스킹판으로 교체 |
| `PostReplyService.java:145-147` | appUserName(실명) · 답글 전문 | `name()` · `text()` |
| `MlService.java:54` | **복호화된 채널 계정 ID** + 답글 본문 | `accountId()` · `text()` |
| `MlService.java:95-96` | 리뷰 작성자 · 리뷰 본문 | `text()` |

3번은 리뷰어도 나도 처음 목록에 없던 것이다. `PostReplyService` 가 `cryptConfig` 로 **복호화한** 채널 로그인
계정 ID 를 ML 요청 DTO 에 실어 보내는데 그게 로그로 그대로 나가고 있었다.

마스킹 지점을 로거가 아니라 **DTO 의 `toString()`** 에 둔 것은 `produceMessage(KafkaProducerTopics, Object)`
시그니처 때문이다 — 호출부에서 타입별로 가릴 수가 없고, P2~P4 에서 다른 DTO 가 같은 자리를 지나가도 각자
책임지게 하는 편이 빠뜨림이 없다. 카프카 페이로드는 게터로 직렬화되므로 영향 없고, 그것을
`LegacyKafkaProducerConfigTest` 가 바이트로 고정한다.

**안 가린 것 — 응답 바디로 나가는 3곳(AC3 때문)**

`CustomUserDetailsService.java:32`(email), `MlService.java:76-78`·`:111`(계정ID·답글 본문)은 구에서
`ErrorResponse.errorMessage` 로 **바디에 실려 나간다.** 여기를 고치면 AC3 이 깨진다. `MlService.java:82`·`:115`
의 `log.error(..., e.getMessage())` 는 그 바디와 같은 문자열을 찍는 것이라 로그만 가려도 실익이 없어 뒀다.
**바디에서 개인정보를 빼는 것은 응답 계약 변경이므로 P6 전 별도 결정 사항으로 올린다.**

게이트는 `LegacyLogMaskingTest` 3케이스. "가려진 문자열이 나오는가"가 아니라 **"원문이 부분문자열로도 나오지
않는가"** 를 본다 — `MaskingUtil.maskPhoneNumber` 는 형식이 안 맞으면 원문을 그대로 돌려주므로 그 경로까지 덮었다
(`ReviewBoostV2BizMessageService.maskPhone:200-203` 이 같은 이유로 쓰는 방식).

---

## 7. B1 숫자 정정 — **반영**

지적대로 문서가 "이식한 개수"를 "미이식 개수"로 뒤집어 적었다. 괄호균형 파서로 재추출했다(depth==1 인
`fun` 선언만 카운트):

```
StoreChannelService.kt   depth==1 fun = 19   이식 1(getStoreChannel :289)          -> 미이식 18
TeamService.kt           depth==1 fun = 13   이식 1(getTeam :65)                   -> 미이식 12
MlService.kt             depth==1 fun =  6   이식 2(addReplyRequest, aiRecommend)  -> 미이식  4
```

신 이식본 쪽도 세어 대조했다(`StoreChannelService.java` 1 · `TeamService.java` 1 · `MlService.java` 2).

연관 목록도 눈이 아니라 기계로 다시 뽑았다 — 구 `db/domain` 전 엔티티의 `@OneToMany/@ManyToOne/@OneToOne/
@ManyToMany` 필드를 신 이식본과 차집합했다. **미이식 13개**이고 문서에는 12개만 있었다:

```
AppUser        미이식=['appUserStoreList','organizationAppUserList','teamAppUserList']
Organization   미이식=['organizationAppUserList','teamList']
Post           미이식=['postReplyRegistrationRequestList','message']     <- 앞의 것이 빠져 있었다
ProductPost    미이식=['threads']
Store          미이식=['appUserStoreList']
StoreChannel   미이식=['postList','storeChannelAttributeList']           <- 문서엔 'attributeList' 로 축약
Team           미이식=['teamAppUserList','teamChannelAccountList']
(나머지 6개 엔티티는 차이 0)
```

`04-changes-P1.md` §1-①②와 AC1 줄(702→703)을 이 실측으로 고쳤다.

---

## 8. B2 · C1~C7 — 판정 수용, 처리 내역

- **B2**(구 버그 2개 보존 확인) — 동의. 변경 없음.
- **C1**(바디 실측 3경로뿐) — 사실이다. 다만 제안한 "구 41010 에 실데이터 1건 프로브"는 이번에 하지 않았다.
  이번 라운드의 지시가 P1 대응 + P2 착수이고, 프로브는 개발계 데이터 상태에 의존해 여기서 재면 다음 라운드에
  또 재야 한다. **P5 진입 조건에 넣는다**(`publicSeoEnabled` storeId 1건으로 2경로 바디 고정).
- **C2**(검색 키 fail-fast 소실) — 사실이다. 코드로 되돌리지 않았다. 구는 키 없으면 **부팅 실패**인데 booster 는
  9모듈이 한 프로세스라 그 방향으로 되돌리면 legacy 키 하나가 없어서 전 모듈이 안 뜬다. 대신 P5 필수 env
  체크리스트 항목으로 승격한다. 새로 열린 설정키 4개(`legacy.external.*-url`)도 같이 기록한다.
- **C3**(`ApiResponse` 핀 없음) — 반영. §1 표에 포함, 바이트 단언도 추가.
- **C4**(`required=false` 9개 계약이 안 잠김) — **P2 착수를 우선해 이번엔 안 했다.** 지적은 옳고 다음 라운드에서
  `MethodParameter` 훑는 단언 한 개로 닫는다.
- **C5**(`AccessDeniedException` 핸들러 미이식 미공개) — `LegacyExceptionAdvice` javadoc 에 한 줄 남기는 것이
  맞다. **이번엔 안 했다** — AC9 대조표 자체가 아직 없어 그 자리를 만들 때 같이 넣는 편이 낫다고 봤다.
- **C6**(`PostRepository` 제네릭 `<Post,String>`) — 동의. AC9 대조표에 남긴다.
- **C7**(`legacy-absorption-p1{,b,c}*.sql` 은 P-1 산출물) — 사실이다. 귀속 정정은 AC9 대조표와 함께.

C4·C5·C6·C7 은 전부 **AC9 이식 대조표**에 붙어야 할 것들이라, 표를 만드는 라운드에서 묶어 처리하는 게
흩어 놓는 것보다 낫다고 판단했다. 지금 라운드에서 닫으라고 하면 닫겠다.

---

## 9. 변경 파일

| 파일 | 한 줄 | 지적 |
|---|---|---|
| `modules/legacy/.../db/dto/**` 12개 + `common/response/ApiResponse.java` | `@JsonPropertyOrder` 핀 | A1 |
| `modules/legacy/src/test/.../db/dto/JsonPropertyOrderCoverageTest.java` | 핀 전수 기계 게이트(신규) | A1 |
| `app/src/test/.../LegacyP1ResponseShapeTest.java` | 중첩 DTO·봉투 바이트 단언 2건 추가 | A1·C3 |
| `modules/legacy/.../security/LegacyAppUserResolutionInterceptor.java` | 경로 불문 app_user 조회(신규) | A2 |
| `modules/legacy/.../security/LegacyUserDetailsArgumentResolver.java` | 인터셉터 결과 재사용(조회 1회) | A2 |
| `modules/legacy/.../security/LegacyWebMvcConfig.java` | 인터셉터 등록(`/api/v1/**`) | A2 |
| `modules/legacy/src/test/.../security/LegacyAppUserResolutionInterceptorTest.java` | 400 회귀 게이트(신규) | A2 |
| `modules/legacy/.../client/kafka/config/LegacyKafkaProducerConfig.java` | 구 프로듀서 설정 1:1 재현(신규) | A3·A4 |
| `modules/legacy/.../client/kafka/KafkaProducer.java` | 모듈 로컬 템플릿으로 교체 | A4 |
| `modules/legacy/src/test/.../client/kafka/config/LegacyKafkaProducerConfigTest.java` | 헤더·바디 바이트 게이트(신규) | A3 |
| `modules/legacy/.../client/LegacyRestClients.java` | 구 매퍼로 JSON 슬롯 교체(신규) | A5 |
| `modules/legacy/.../db/dto/ml/MlAiRecommendRes.java`·`MlPostReplyRegistraionRes.java` | 엄격 creator 로 교체 | A5 |
| `modules/legacy/.../client/ml/service/MlService.java` | 클라이언트 교체 + 로그 마스킹 | A5·B3 |
| `modules/legacy/.../client/external/ExternalClientService.java` | 클라이언트 교체 | A5 |
| `modules/legacy/src/test/.../client/LegacyRestClientsTest.java` | 두 겹 각각 게이트(신규) | A5 |
| `modules/legacy/.../common/log/LegacyLogMasking.java` | 로그 전용 마스킹(신규) | B3 |
| `modules/legacy/.../db/dto/kafka/ProducerUserPostReplyCreate.java` | `@ToString` → 마스킹 `toString()` | B3 |
| `modules/legacy/.../apis/http/postReply/service/PostReplyService.java` | 실명·답글 마스킹 | B3 |
| `modules/legacy/src/test/.../common/log/LegacyLogMaskingTest.java` | 원문 미노출 게이트(신규) | B3 |
| `review/.../04-changes-P1.md` | 미이식 수·연관 목록·기준선 정정 | B1 |

`:core` 와 `app` 의 main 소스는 이번 라운드에서 **한 줄도 안 건드렸다.** 변경은 전부 `modules/legacy` 안이고
예외는 `app` 테스트 1개(단언 추가)뿐이다.

## 10. 테스트 재실행

```
$ cd /Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app && ./gradlew test
EXIT=0
  app 205 / core 48 / chrono 7 / crawling 68 / external 40 / legacy 26 /
  medicontents 18 / notification 87 / post 67 / product 71 / user 86
합계 tests=723 failures=0 errors=0        (기준선 703 → +20: legacy +18, app +2)

$ ./gradlew duplicateClassCheck        -> DUP_EXIT=0
$ cd ../gateway-app && ./gradlew test  -> tests=76 failures=0 errors=0
$ sed -n '34p' .../route/Routes.java   -> new Route("legacy", MEDILAWYER, "/api/v1", ServiceId.LEGACY)
$ cd /Users/steve/steve/legal-care/medilawyer-boot && git status --short  -> (빈 출력)
```

야간 불변조건: 컨테이너 정지·삭제·compose 0 · 운영 RDS 0 · 구 서비스 정지 0 · **외부 실호출 0**(이번 라운드는
구 41010 프로브도 하지 않았다 — 근거는 전부 소스 대조와 로컬 테스트다) · 커밋·푸시 0 · 게이트웨이 무수정 ·
`medilawyer-boot` 무수정 · booster-web env 무변경.

STATUS: REVISED
