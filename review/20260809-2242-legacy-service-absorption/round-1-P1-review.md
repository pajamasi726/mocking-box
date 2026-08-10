# Round 1 — P1(읽기, R3) 교차 검토

수정 권한 `review` — **코드는 한 줄도 건드리지 않았다.** 개발계는 읽기만 했고 컨테이너·구 서비스·게이트웨이·운영 RDS 에 접근하지 않았다.
근거는 두 소스 트리 직접 대조뿐이다: 구 `/Users/steve/steve/legal-care/medilawyer-boot`(HEAD `930c83cf`, clean) vs
신 `/Users/steve/steve/legalcare-renew-prodsync-wt`(HEAD `b42d9b6`, 커밋 0 — 확인함).

리플레이가 P1 을 거의 못 받쳐준다는 전제(11경로 중 골든 있는 것 3개, 그중 read 샤드는 2개)를 그대로 받아들이고,
**코드 대조를 유일한 방어선으로 삼아** 11경로를 구 원본과 한 줄씩 맞춰봤다. 결론부터: 이식의 골격은 견고하고
작성자가 스스로 올린 판단 3건 중 2건은 내 검증에서도 옳다. 다만 **리플레이로는 영원히 안 잡히는 결함 2건이 더 남아 있고**,
문서가 실제보다 안전해 보이게 쓰인 곳이 몇 군데 있다. 그래서 REQUEST_CHANGES 다.

---

## A. 반드시 고쳐야 하는 것

### A1. (CRITICAL) 구글 검색 응답의 중첩 DTO 2개가 키 순서를 잃는다 — 200 이라 리플레이가 못 잡는다

작성자가 `isFirst` 중복키를 잡아낸 것과 **정확히 같은 부류**인데, 두 건이 남아 있다.

- `modules/legacy/.../db/dto/external/DisplayName.java:13-16` — `@JsonPropertyOrder` 없음
- `modules/legacy/.../db/dto/external/GoogleMapsLinks.java:13-19` — `@JsonPropertyOrder` 없음

둘 다 `@NoArgsConstructor` + `@AllArgsConstructor` 라 **생성자가 2개 → Jackson 이 암묵 creator 를 못 고르고 알파벳순으로 떨어진다.**
(이 저장소가 올라탄 Jackson 3 은 `SORT_PROPERTIES_ALPHABETICALLY` 기본값이 `true` 다. 작성자 본인이
`ErrorResponse.java:13-15` 에 "붙이기 전엔 `errorMessage` 가 `errors` 앞으로 나왔다"고 실측을 남겼는데, 그게 바로 이 알파벳 정렬이다.)

구 원본 `module-storage/.../db/dto/external/GoogleSearhResultItem.kt:3-6, 8-14` 와 대조한 실제 차이:

| DTO | 구 키 순서 | 신 키 순서(알파벳) |
|---|---|---|
| `DisplayName` | `text, languageCode` | `languageCode, text` |
| `GoogleMapsLinks` | `directionsUri, placeUri, writeAReviewUri, reviewsUri, photosUri` | `directionsUri, photosUri, placeUri, reviewsUri, writeAReviewUri` (5개 중 4개 이동) |

이 둘은 `GET /api/v1/store-channel/search/google` 응답 `data.list[]` 안에 그대로 실려 나간다(`ExternalClientService.java:307-312`).
바깥쪽 `GoogleSearchResultItem.java:19-20` 은 핀이 박혀 있는데 **`@JsonPropertyOrder` 는 중첩 타입으로 전파되지 않는다** — 그래서 놓친 것이다.
상태코드 200, 골든 0건. 리플레이로도 상태코드 비교로도 영원히 안 잡히고 전환 당일 클라이언트에서 처음 터진다.

`@JsonPropertyOrder({"text","languageCode"})` / `@JsonPropertyOrder({"directionsUri","placeUri","writeAReviewUri","reviewsUri","photosUri"})` 를 달면 끝난다.
**같은 김에 `db/dto` 전수 재확인을 권한다** — 나는 "핀이 있거나, 생성자가 정확히 하나이거나" 규칙으로 34개 DTO 를 훑었고
나머지 21개는 전 필드가 `@JsonPropertyOrder` 에 열거돼 있음을 확인했다(`PostListData` 21/21, 오타키 `reviewDeleteProducutPostStatus` 포함).
직렬화되지 않는 요청측 DTO 9개(`PostListReq`·`StoreChannelReq`·`NaverSearchResult` 등)는 지금은 무해하지만 같은 지뢰다.

### A2. (MAJOR) 인증 사용자 조회 범위가 좁아져 "구 400 → 신 200" 이 생긴다 — AC3 의 판정 규칙이 못 잡는 방향이다

구 `security/JwtAuthenticationFilter.kt:38-73` 은 **Authorization 헤더가 있으면 경로를 가리지 않고** 모든 요청에서
`customUserDetailsService.loadUserByUsername(email, ...)` 을 호출한다. 그 이메일의 `app_user` 가 없으면 거기서 `USER_NOT_FOUND` 로 끊긴다(400 `USER-002`).

신 `LegacyUserDetailsArgumentResolver.java:59-68` 은 **컨트롤러가 `CustomUserDetails` 파라미터를 선언한 경로에서만** 조회한다.
11경로 중 그런 경로는 `GET /api/v1/posts` 와 `POST /api/v1/post-reply` 둘뿐이다.

작성자도 이 차이를 `04-changes-P1.md:130` 에 적었지만 "응답이 갈리는 경우는 극단뿐이고 그때도 봉투는 같은 `USER-002` 400" 이라고 썼다.
**그 문장은 파라미터를 선언한 2경로에 대해서만 참이다.** 나머지 9경로에서는 구 400 / 신 200 으로 갈린다. 그리고 하필,

- `GET /api/v1/post-word/statistics` — 골든 9건, **9건 전부 Authorization 有**
- `POST /api/v1/post-reply-ai-recommend` — 골든 24건, **24건 전부 Authorization 有**

즉 **P1 골든 35건 중 33건이 정확히 이 갈림길 위에 있다.** 게다가 신규 PG 는 별도 인스턴스이고 P-1 에서 "접촉 테이블만" 시드했으므로
`app_user` 행이 비어 있는 것은 희귀 케이스가 아니라 **오히려 기본 상태**다.

방향이 `구 400 → 신 200` 이라는 게 핵심이다. AC3 의 판정 문구는 "구 200 → 신 비200 0건" 이라 **이 방향을 통과시킨다.**
작성자가 P0 라운드1 에서 excel 엔드포인트를 두고 스스로 지적했던 바로 그 구멍이다(`04-changes-P1.md:63`).

요청: ① 리졸버 호출 여부와 무관하게 구처럼 필터 단계에서 조회하도록 맞추든지, ② 못 맞추겠으면
**AC3 판정 규칙에 "구 비200 → 신 200" 도 회귀로 센다**를 명시하고 33건에 대해 `app_user` 존재를 선행 시드 조건으로 못박든지 — 둘 중 하나는 해야 한다.
지금 문서 상태로는 리플레이를 돌려도 초록이 나오고, 그 초록이 아무것도 보증하지 않는다.

### A3. (MAJOR) 카프카 와이어 포맷이 구와 다르다 — 문서에 한 줄도 없다

`POST /api/v1/post-reply` 성공 시 `medilawyer.event.user.post.reply.created` 를 발행한다(`PostReplyService.java:139`).

구 `module-core/core-api/.../kafka/config/KafkaProducerConfig.kt:20-29`:
- `JsonSerializer`, `ADD_TYPE_INFO_HEADERS` **미설정 → 기본값 `true`** → `__TypeId__: legalcare.medilawyer.db.dto.kafka.ProducerUserPostReplyCreate` 헤더가 붙는다
- `MAX_REQUEST_SIZE_CONFIG = 524288000`
- `security.protocol = PLAINTEXT`

신 `KafkaProducer.java:31` 은 app 계층의 `queueKafkaTemplate` 을 그대로 빌려 쓰는데, 그 팩토리(`app/.../QueueConfig.java:66-81`)는
- `JsonSerializer.ADD_TYPE_INFO_HEADERS = **false**` (해당 주석에 "notification-service 소비자가 `__TypeId__` 를 기대하지 않아 일부러 껐다"고 적혀 있다 — **다른 토픽 이야기다**)
- `MAX_REQUEST_SIZE` 미설정(기본 1MB)
- `security.protocol` 미설정

`04-changes-P1.md` 는 이 지점을 "JSON value serializer 를 쓴다" 로만 적고 넘어갔다. 헤더·요청크기·프로토콜 3개가 다르다.

주의할 점은 **"헤더를 다시 켜면 된다"가 답이 아니라는 것**이다. 신 DTO 는 패키지가 `legalcare.medilawyer.legacy.db.dto.kafka` 로 바뀌었으므로
헤더를 켜면 구와 **다른 FQCN** 이 실린다. 타입 매핑으로 소비하는 소비자가 있다면 오히려 그때 깨진다.
정답은 셋 중 하나다 — ① 소비자가 `__TypeId__` 를 안 본다는 것을 실측으로 확정하고 문서에 남기거나,
② `JsonSerializer` 타입 매핑으로 구 FQCN 을 그대로 실어 보내거나, ③ legacy 전용 프로듀서 팩토리를 따로 두고 구 설정을 그대로 재현하거나.
나는 소비자를 확인하지 못했다 — 구 레포 안에는 이 토픽의 컨슈머가 없고(레포 내 컨슈머는 `StringDeserializer` 를 쓴다) 외부 서비스가 받는다.
**P2/AC4 착수 전에 결론이 나 있어야 한다.**

### A4. (MAJOR) 모듈 → app 역방향 의존 — 기술 제약 위반이고, A3 의 원인이기도 하다

`modules/legacy/.../client/kafka/KafkaProducer.java:31` 이 `@Qualifier("queueKafkaTemplate")` 로
**app 계층에 정의된 빈**(`app/src/main/java/legalcare/medilawyer/QueueConfig.java:83`)을 직접 끌어온다.

`01-requirements.md:29` 의 기술 제약은 "모듈 간 역방향 의존 금지(인터페이스는 호출 모듈, 구현은 app 계층 `*LocalAdapter` @Primary)" 다.
컴파일 의존은 아니지만 런타임 의존이고, 인터페이스 우회 없이 이름으로 직결한 형태라 규약이 막으려는 것을 그대로 한다.

게다가 이 저장소에는 **이미 정답 선례가 있다**: `modules/post/.../queue/config/PostKafkaProducerConfig.java:59-65` 는
모듈 안에 자기 프로듀서 팩토리 + `KafkaTemplate` + `QueuePublisher` 를 두고, 그 클래스 주석에
"`QueueConfig` 의 기존 팩토리와 다르면 새 프로듀서 팩토리+KafkaTemplate 빈을 추가" 라는 규칙까지 명시돼 있다.
legacy 는 구 설정이 `QueueConfig` 와 **실제로 다른데**(A3) 그 규칙을 타지 않았다.

`build.gradle` 자체는 깨끗하다 — `api project(':core')` 하나뿐이고 다른 모듈 의존 0, **Feign 0**(전수 grep), 외부 HTTP 는 전부 `RestClient`. 그 부분은 규약대로다.

### A5. (MAJOR) ML 응답 파싱 관대함이 구보다 커졌다 — 구 400 이 신 200 이 된다

`ExternalClientService`·`MlService` 가 쓰는 `RestClientSupport.builder()`(`core/.../rest/RestClientSupport.java:110-121`)는
JSON 슬롯에 "Feign 패리티" 컨버터를 끼운다. 그 컨버터가 켜는 것(`:163`):

```java
.configure(EnumFeature.READ_UNKNOWN_ENUM_VALUES_AS_NULL, true)
```

구는 `RestTemplate`/`WebClient` 가 앱 `ObjectMapper`(`module-core/core-common/.../common/jackson/ObjectConfig.kt:12-19`)를 쓰는데,
거기엔 `FAIL_ON_UNKNOWN_PROPERTIES=false` 만 있고 **`READ_UNKNOWN_ENUM_VALUES_AS_NULL` 은 없다.**

결과: ML 이 `MlResponseStatus`/`PostReplyRegistrationResult` 에 없는 `status` 를 돌려주면
- 구: 역직렬화 예외 → catch → `CustomException(REQUEST_FAIL_ML)` → **400 `REQUEST-003`**
- 신: **200** + `"status": null`

`POST /api/v1/post-reply-ai-recommend`(골든 24건)와 `POST /api/v1/post-reply` 둘 다 해당한다.
같은 결의 두 번째 항목: 구는 `KotlinModule` 이라 non-null 생성자 파라미터가 응답에 없으면 던지는데
(`MlAiRecommendRes.kt:8-16` 의 `val replyAi: String`), 신 `MlAiRecommendRes.java:19-28` 은 `@NoArgsConstructor`+setter 라 조용히 null 이 된다.

**단, 이건 ML 이 실제로 그런 페이로드를 주는지에 달린 조건부 결함이다.** 지금은 `ml.review.server` 기본값이 스텁이라 확인할 방법도 없다.
최소한 "알려진 차이"로 문서에 올리고, P5 에서 실 ML 붙일 때 판정 항목에 넣어야 한다.

---

## B. 작성자가 판단을 구한 3건 — 내 판정

### B1. "1:1 의 경계" — **타당하다. 다만 보고된 숫자가 틀렸다.**

도달 불가 주장을 독립적으로 검증했다. 11경로의 구 컨트롤러에서 전체 호출 그래프를 따라가 클래스별 메서드를 전수 대조했고,
**빠뜨린 도달 가능 메서드는 하나도 없었다.** 간접 도달 경로(인터페이스 구현, Kotlin 확장함수, 메서드 참조, `@EventListener`,
`@Scheduled`, `@Transactional` 셀프인보케이션)도 전부 0건으로 확인했다. 오버로드 함정도 확인했다 —
구 `getStoreChannel` 은 3개 오버로드(`StoreChannelService.kt:289, 304, 561`)인데 `PostWordController.kt:32` 의 인자 형태는 `:289` 로만 해석된다.

엔티티 연관도 마찬가지다. 특히 **EAGER 함정이 없다**는 것을 확인했다 — 구 `db/domain` 전체에서 `EAGER` 0건이고,
`fetch = FetchType.LAZY` 가 안 붙은 연관 애노테이션도 0건이다. `orphanRemoval` 도 레포 전체 0건이고,
남은 `cascade = [PERSIST]` 는 P1 이 해당 엔티티를 persist 하지 않아 무관하다.
`Post` 에 대한 쓰기(`post.postReply = …`, `post.postReplyAiRecommend = …`)는 dirty-check UPDATE 이고 양쪽 다 `@DynamicUpdate` 라
**빠진 `message_id` 가 null 로 밀리지 않는다.**

**그런데 `04-changes-P1.md:125` 의 숫자가 실제와 다르다.**

| 구 클래스 | 구 메서드 수 | 이식 | 실제 미이식 | 문서 표기 |
|---|---|---|---|---|
| `StoreChannelService.kt` (575줄) | 19 | 1 | **18** | "1개" |
| `TeamService.kt` (190줄) | 13 | 1 | **12** | "1개" |
| `MlService.kt` (248줄) | 6 | 2 | **4** | "2개" |

코드 안 javadoc 은 "**P1 도달 1개**만 이식" 으로 정확하다 — 문서로 옮기면서 "도달 1개"가 "미이식 1개"로 뒤집혔다.
결론은 안 바뀌지만 **AC9 의 이식 대조표 147행이 이 숫자 위에 서 있어서** 그대로 두면 안 된다.
같은 맥락으로 연관 목록도 하나 빠졌다 — `Post.postReplyRegistrationRequestList`(`Post.kt:96-98`)도 미이식인데 §1-② 목록에 없다.
(레포 전체 사용처 0건이라 가장 안전한 축이지만, 목록이 불완전하면 진짜 누락이 거기 숨는다.)

### B2. 구의 버그 2개 보존 — **둘 다 사실이고, 그대로 두는 것이 맞다.**

**`getPostList` 의 `productPostStatus`** — 구 `PostRepositoryImpl.kt:67-74` 를 직접 읽어 확인했다.
`teamId?.let { A; B } ?: let { C; D }` 에서 Kotlin `let` 은 마지막 식만 반환하므로 `productPostStatusIn(...)` 결과는 버려지고,
where 에 들어가는 것은 `hasProduct(...)` 뿐이다. 주장대로다.

더 중요한 건 작성자가 **엘비스의 2차 효과까지 재현했다**는 점이다 — `teamId != null` 이라도 `hasProduct == null` 이면
첫 블록의 반환이 null 이라 엘비스가 두 번째 블록을 실행한다. `PostRepositoryImpl.java:155-173` 의
`if (condition == null) { ... }` 가 정확히 그 자리다. 이건 Kotlin→Java 이식에서 가장 놓치기 쉬운 종류인데 제대로 잡았다.

**`usable` 무시** — 구 `StoreChannelService.kt:290`(`..., true` 하드코딩)과
`TeamChannelAccountService.kt:27`(같음)에서 확인했다. 사실이다.
덧붙이면 **P1 의 두 호출부가 어차피 `true` 를 넘기므로 이 버그는 P1 에서 휴면 상태다** — 보존 비용이 0 이라는 뜻이고, 그래서 더더욱 보존이 맞다.

### B3. 개인정보 로깅 — **"1:1 이라서 못 고친다"는 논리를 기각한다. 마스킹이 맞다.**

`01-requirements.md:26` 의 1:1 제약이 보호하는 대상은 명시적으로 **"클래스·메서드·필드명·로직·URL 경로"** 이고,
AC 들이 판정하는 것은 **요청/응답 바디·HTTP 상태·에러코드**다. **로그 라인은 그 어디에도 없다.**
리플레이 차등검증도 로그를 읽지 않는다. 즉 `phone` 을 마스킹해도 **AC3 판정에 미치는 영향이 0** 이다.
반면 민감정보 로깅 금지는 조건부 규칙이 아니다. 트레이드오프가 성립하지 않으므로 "판단을 구한다"의 답은 명확하다 — 마스킹하라.
그게 싫으면 1:1 을 근거로 들 게 아니라 **명시적 예외 승인**을 받아 문서에 남겨야 한다.

그리고 **공개된 건이 하나 더 있다.** 작성자는 `KafkaProducer.produceMessage` 만 올렸는데,
`PostReplyService.java:145-147` 도 `appUserName`(실명)과 **답글 본문 전체**를 로그에 찍는다(구도 동일).
지적 목록을 스스로 올릴 거면 전수여야 한다 — 하나만 올리면 "나머지는 검토했고 문제없다"로 읽힌다.

---

## C. 그 외 지적

### C1. (MEDIUM) 응답 바디 동등을 실측한 것은 11경로 중 3개뿐이고, 가장 위험한 경로가 그 밖에 있다

`04-changes-P1.md:138` 이 정직하게 적은 대로 실측 대조는 `/api/v1/posts`(빈결과·1건), `/api/v1/public/hospitals`(**빈결과만**), 에러 3케이스가 전부다.
문서 머리말의 "읽기 프로브 6건" 중 4건은 존재하지 않는 ID 라 컨트롤러 직전에 끊겼으므로 바디 대조 근거가 아니다.

문제는 **`GET /api/v1/public/hospitals/{storeId}` 가 그 밖에 있다는 것**이다. 이 경로가 P1 에서 로직 밀도가 가장 높다 —
`Math.rint` 반올림, 키워드 동점 시 안정정렬, 스니펫 160자 절단, 채널 distinct 가 전부 여기 모여 있다.
그런데 **골든 0건 + 실측 0건**이고, 방어선은 `PublicHospitalServiceTest` 하나뿐인데
그 테스트는 "신 구현이 리뷰어가 읽은 구 사양과 맞는가"를 볼 뿐 **구의 실제 출력과 맞는가는 보지 않는다.**

`/api/v1/public/hospitals` 도 마찬가지다. 실측한 것은 `"data":[]` 인 빈 목록이라 **`updatedAt` 이 한 번도 바이트로 비교된 적이 없다.**
`LegacyP1ResponseShapeTest.publicHospitalSitemap_keyOrder` 도 `startsWith("{\"storeId\":7,\"updatedAt\":")` 로 **날짜 형식을 일부러 비껴간다.**

다만 이건 내가 확인한 범위에서는 위험이 낮다 — 날짜 렌더링을 가를 수 있는 입력 두 개를 양쪽에서 대조했는데 둘 다 같다:
구·신 모두 `hibernate.jdbc.time_zone: UTC`(구 `core-storage.yml:13` / 신 `application.yml:119`)이고,
양쪽 다 `spring.jackson` 설정이 없어 ISO-8601 기본값이다. 작성자의 §4 타임존 판단은 유효하고 근거도 그가 쓴 것보다 튼튼하다.
**그래도 실데이터 1건 프로브면 끝나는 일이다** — 구 41010 에 `publicSeoEnabled` 인 storeId 하나로 두 경로를 찔러 바디를 고정하라.

### C2. (MEDIUM) 외부 검색 자격증명의 fail-fast 가 사라졌다

구 `ExternalClientService.kt:30-43` 은 `@Value("${naver.naverClientId}")` — **기본값이 없어 키가 없으면 부팅이 실패한다.**
신 `ExternalClientService.java:78-84` 은 `@Value("${naver.naverClientId:}")` — 빈 문자열로 뜨고, 검색 5경로가 호출 시점에 400 `REQUEST-002` 를 낸다.

`config/legacy.yml:31-32` 에 이 차이를 적어 둔 건 좋다. 하지만 **기동 실패(즉시 발각)를 런타임 오답(조용한 회귀)으로 바꾼 변경**이고,
하필 그 5경로가 골든 0건이라 리플레이도 못 잡는다. P5 에서 env 주입을 잊으면 "테스트 통과 + 전 검색 400" 상태로 전환된다.
최소한 P5 진입 전 필수 env 체크리스트로 승격하거나, 검색 경로에 한해 기동 시 키 존재를 검증하는 편이 낫다.

한편 새로 열린 설정키 4개(`legacy.external.*-url`, `legacy.yml:43-48`)는 구에 없던 것이다. 기본값이 구 리터럴과 동일해 동작 차이는 없고
리플레이 스텁 주입에 오히려 유용하지만, **구에 없던 설정 표면이 늘었다는 사실이 변경 파일 표에 없다.**

### C3. (MINOR) `ApiResponse` 만 키 순서가 핀 없이 서 있다 — 11경로 전부의 봉투다

`common/response/ApiResponse.java:15-26` 에는 `@JsonPropertyOrder` 가 없다. 형제 DTO 들은 전부 붙어 있는데 여기만 없다.
지금 순서가 맞는 이유는 **생성자가 정확히 하나이고 `-parameters` 컴파일 옵션이 켜져 있어서**(`build.gradle:23`) creator 순서를 타기 때문이다.
그 플래그가 빠지는 순간 `{data, message, status}` 알파벳순으로 뒤집히고, **11경로 성공 응답이 한꺼번에 깨진다.**
`LegacyP1ResponseShapeTest` 가 문자열 비교로 잡아주긴 하지만, 타입 자체에 못박아 두는 편이 이 파급력에 맞다.

### C4. (MINOR) 매핑 게이트 테스트가 경로·메서드만 고정한다 — 필수/선택 파라미터 계약은 안 잡힌다

`LegacyP1EndpointMountTest` 는 훌륭하다. 양방향 집합 비교로 빠짐·유령을 둘 다 잡고,
`private` 핸들러 매핑과 `LegacyExceptionAdvice` 의 `basePackages` 축소까지 못박는다. 이런 게이트가 있어야 맞다.

다만 고정 대상이 `METHOD + path` 뿐이다. `PostController.java:50-64` 의 `@RequestParam(required=false)` 9개는
작성자가 구 서버에 직접 찔러 확정한 계약인데(`PostController.java:27-32` javadoc), **그 측정을 잠가 두는 단언이 없다.**
누가 `required=false` 하나를 지우면 구 200 이 신 500 이 되고, 골든이 0건이라 리플레이도 침묵한다.
`MethodParameter` 의 required 플래그를 훑는 단언 한 개면 된다.

### C5. (MINOR) 구 `AccessDeniedException` 핸들러 미이식이 공개되지 않았다

구 `GlobalExceptionHandler.kt` 는 핸들러가 4개다 — `CustomException`(:22), `Exception`(:41), `AuthenticationException`(:62), **`AccessDeniedException`(:83)**.
신 `LegacyExceptionAdvice.java` 는 앞의 2개만 옮겼고, javadoc(`:33-35`)은 `AuthenticationException` 만 언급한다.

P1 경로에는 메서드 시큐리티가 없어 `AccessDeniedException` 이 도달할 일이 없으므로 **동작 영향은 없다.** 판단 자체는 옳다.
하지만 미이식 목록이 불완전하면 AC9 대조표에서 그 자리가 빈다. 한 줄 추가하면 된다.

### C6. (MINOR) `PostRepository` 제네릭 변경은 받아들이되 대조표에 남겨야 한다

`PostRepository.java:28` 이 구 `JpaRepository<Post, Long>` 를 `<Post, String>` 으로 고쳤다(구의 선언 오류 교정).
`@Id` 가 실제로 `String` 이고 P1 이 `findById` 계열을 안 쓰므로 **동작 차이 0** 이다. javadoc 도 이유를 정확히 적었다. 동의한다.
다만 "1:1 이식"을 표방하는 단계에서 시그니처를 바꾼 유일한 지점이므로 AC9 대조표에 명시적으로 남겨라.

### C7. (MINOR) `git status` 잔여 항목이 "전부 P0 산출물" 이라는 서술이 부정확하다

`04-changes-P1.md:26` 의 문장과 달리, 미추적 파일 중 `app/src/main/resources/db/legacy-absorption-p1{,b,c}*.sql` 3개는
**P-1(R11) 스키마 동기화 산출물**이다(각 파일 헤더가 "P-1 스키마 동기화", 근거 문서로 `P-1-schema-sync.md` 를 지목한다).
"P-1" 과 "P1" 이 파일명에서 구별이 안 돼서 생긴 혼동으로 보인다.
셋 다 Flyway·`spring.sql.init` 어디에도 연결돼 있지 않아(전수 grep) 빌드·기동에 영향은 없다. 귀속만 정정하면 된다.

---

## D. 내가 확인해서 문제없다고 판정한 것 (다음 라운드에서 다시 하지 마라)

- **11경로 = 11경로.** 구 7컨트롤러를 직접 읽어 매핑을 독립 재열거했다(PublicHospital 2 · Community 0 · PostWord 1 · StoreChannel 5 · Post 1 · PostReply 1 · AiRecommend 1). 신도 동일. URL·HTTP 메서드·파라미터 이름/필수여부·`defaultValue` 까지 일치.
- **`ErrorType` 27/27 바이트 일치**(공백 제외 diff 0). **`KafkaProducerTopics` 23/23 일치**, 중복 code 21/22 보존 포함.
- **`QuerydslFilter`** 전 메서드 등가. `sortedByDescending`↔`Comparator.reversed()` 안정성, `when(filter){ALL,null}`↔`if(null||ALL)`, `between` 오버로드까지 확인.
- **`PostListData`** 21필드 전 항목 등가. `authorDtm`/`replyAt` 은 양쪽 다 `toEpochMilli()` — 초/밀리 혼동 없음. `find{}`↔`findFirst()`, `it.team?.identifier == teamId`↔`getTeam()!=null && equals` 등가.
- **`PublicHospitalService`**: `kotlin.math.round` = `Math.rint`(ties-to-even) 판단이 옳다 — 진짜 catch 다. Kotlin `trim()`↔Java `strip()` 도 옳다(`trim()` 이었으면 틀렸다). `groupingBy/eachCount`↔`LinkedHashMap`, `mapNotNull().take()`↔`filter().limit()` 등가.
- **`PostWordService`·`PostReplyService`·`PostReplyAiRecommendService`** 등가. 특히 `post.postReplyAiRecommend?.let{ 대입 } ?: run{}` 이 대입식의 반환값 `Unit` 때문에 엘비스를 타지 않는다는 점까지 신 `if/else` 가 옳게 재현했다.
- **`isXxx` 중복키 전수 확인 완료.** `SliceValue:42,47` · `PostListData:109,114` · `NaverChannelRes:33` · `KakaoBusinessSearchRes:29` · `ModoodocBusinessSearchRes:29` · `KakaoBusinessSearchResultMeta:31,36` — 전부 게터에 `@JsonProperty`. **필드에 붙인 것은 0건**(그 자리에 붙이면 중복이 생긴다). `getXxx()`/`isXxx()` 동시 존재도 0건. `canReply`·`usable`·`modify` 같은 비-`is` 불리언도 양쪽 키가 같다. **A1 의 2건 외에 남은 것은 없다.**
- **`@JsonInclude`** 양쪽 모두 미적용, `"errorMessage":null` 방출 동일. **enum** 은 양쪽 다 `name()`. **날짜/숫자** 렌더링 동일(C1 참조).
- **모듈 의존**: `modules/legacy/build.gradle` 은 `api project(':core')` 하나. 다른 모듈 의존 0, Feign 0, 외부 HTTP 전부 RestClient. ✅ (런타임 예외 1건은 A4)
- **범위**: `modules/legacy` 소스 126파일 + app 테스트 2 + `config/legacy.yml` 1 + `application.yml` 1줄. **전부 R3 에 매핑되고 R3 밖 변경은 없다.** 커스텀 리포지토리 인터페이스 6개가 전부 P1 도달 메서드 1개씩으로 깎여 있는 것도 확인했다. 게이트웨이 무수정(`Routes.java:34` = `ServiceId.LEGACY`), `medilawyer-boot` 무수정(clean) 확인.
- **`ExternalClientService`** 5경로: URL 조립(base+query), 헤더, 바디, `<b>` 태그 제거, 카카오/모두닥 루프 종료조건(`5/5=1`, `5/20=0`), `NaverPlaceItem` 폴백 인자 순서 모두 등가. `.uri()` 생략 시 baseUrl 로 POST 되는 것도 확인.

---

## E. 확인 못 한 것 (정직하게)

- **아무것도 실행하지 않았다.** 컴파일·테스트·리플레이 0. 보고된 703/0 · 76/0 은 오케스트레이터 재실측을 신뢰했고 내가 다시 돌리지 않았다.
- **구 서버 실응답을 직접 받지 않았다.** 구 41010 프로브를 하지 않았으므로, 구측 직렬화 결과는 Kotlin/`jackson-module-kotlin` 의미론 + 작성자가 코드 주석에 남긴 실측 기록(`SliceValue.java:11-15`, `ErrorResponse.java:13-15`)에 의존한다. 특히 **구가 `isFirst` 만 내보내고 `first` 는 안 내보낸다는 것을 독립 확인하지 못했다.**
- **A3 의 카프카 소비자 설정을 확인하지 못했다.** 이 토픽의 컨슈머는 구 레포 밖에 있다. 그래서 A3 의 심각도는 "다르다(확정) + 깨진다(미확정)" 이다.
- **A5 는 조건부다.** ML 이 실제로 미지 enum·누락 필드를 돌려주는지 확인하지 못했다(현재 스텁).
- **런타임 동적 참조.** 미이식 필드명과 Q타입을 구 트리 전수 grep 했으나, 문자열로 조립되는 JPQL 이나 리플렉션 참조는 grep 으로 증명되지 않는다. 그런 구성은 발견하지 못했다.

---

## 총평

이식 자체의 품질은 높다. Kotlin `let`/엘비스의 반환값 의미론, `kotlin.math.round`=`Math.rint`, `trim()`↔`strip()`,
`private` 핸들러 매핑, lombok `isXxx` 게터의 Jackson 명명 — **Kotlin→Java 이식에서 조용히 틀리는 지점들을 대부분 정확히 짚었다.**
게이트 테스트 2개도 장식이 아니라 실제로 회귀를 막는 물건이다. 리플레이가 못 받쳐주는 단계에서 이 정도 자기검증은 적절한 대응이다.

승인하지 않는 이유는 세 가지다.
첫째, **작성자가 자기 방법론으로 잡아낸 결함과 정확히 같은 부류가 두 건 더 남아 있다**(A1) — 방법은 옳았는데 전수가 아니었다.
둘째, **AC3 판정 규칙이 통과시켜 버리는 방향의 회귀가 골든 33/35 건 위에 놓여 있다**(A2). 이 상태로 리플레이를 돌리면 초록이 나오는데 그 초록이 거짓이다.
셋째, **카프카 와이어 포맷이 구와 다른데 문서에 없다**(A3/A4). 돈·발송이 걸린 P2/P4 로 넘어가기 전에 닫혀야 하는 종류다.

A1 은 애노테이션 두 줄이면 끝난다. A2 는 코드 수정이든 판정 규칙 개정이든 택일하면 된다.
A3/A4 는 소비자 실측 한 번 + 모듈 로컬 프로듀서 설정으로 정리된다. B1 의 숫자와 B3 의 마스킹은 문서·한 줄 수정이다.
전부 이번 라운드 안에 닫을 수 있는 크기라고 본다.

VERDICT: REQUEST_CHANGES
