# 라운드1 P2 교차검토 — 코어 쓰기 51경로 (R4/AC4)

수정 권한 `review` 라 코드는 한 줄도 건드리지 않았다. 커밋·푸시 0, 게이트웨이 라우트 무수정,
컨테이너·운영 RDS·외부 실호출 0건. 구 41010 프로브도 하지 않았다(소스 대조로 충분했다).

먼저 인정할 것부터. 51경로 인벤토리가 실제로 51개 매핑과 일치하고, `@JsonPropertyOrder` 는
111개 DTO 전수에서 구 Kotlin 주생성자 순서와 이름·순서 모두 어긋난 곳이 없다. P2-B 가 잡았던
밑줄 필드 문제는 `CookieDict` 4개 25필드 전부 `@JsonProperty` 로 못박혀 닫혔고, `isX` 계열 13개도
게터에 핀이 붙어 있다. 밑줄은 필드에, `isX` 는 게터에 붙여야 한다는 것까지 구분해 쓴 흔적이 보인다
(`NaverChannelRes.java:29-30` 에 잘못 붙였을 때 `{"isEnd":true,"end":true}` 가 된다는 실측까지 남겼다).
로그 마스킹도 `MaskingUtil` 이 형식 불일치 입력을 원문 그대로 돌려주는 성질까지 알고
`LegacyLogMasking.java:40,49,58` 에서 "마스킹이 안 일어났으면 통째로 가린다"로 감쌌다. 이 부분은 좋다.

그런데 **응답 바디 바깥**에서 구와 갈리는 것이 여럿 나왔다. 리플레이가 P2 의 11.5% 밖에 못 덮는
상황이라 이 축이 중요한데, 아래 1~4번은 전부 그 축에 있다.

---

## 지적

### 1. [BLOCKER] 이식된 외부호출 전량이 타임아웃 무한 + `@Transactional` 안 — DB 커넥션이 영구히 묶인다

작성자가 "값 하나로 덮으면 셋 중 둘이 틀린다"고 한 판단은 맞다. 다만 실측하니 **셋이 아니라 넷**이고,
더 중요한 건 이게 스레드 문제가 아니라 **커넥션 풀 문제**라는 점이다.

구의 프로파일(내가 구 소스에서 직접 확인):

| 구 스택 | connect | read/response | 이식된 사용처 |
|---|---|---|---|
| `RestTemplateConfig.kt:12-13` | 10s | **60s** | `MlService` 6메서드 중 5, `BizMessageService` 알림톡, `PushEmailService.sendOtpEmail` |
| `WebClientConfig.kt:18-21` | 5s | **10s** | `ExternalClientService` 검색 5종, `PushEmailService.requestBaseEmail` |
| `MlService.kt:78-79` 인라인 | (Netty 기본 30s) | **30s** | `authenticateChannelAccount` 의 `KAKAO_MAP` |
| `MlService.kt:102-108` 인라인 | (Netty 기본 30s) | **없음** | 같은 메서드의 `GOOGLE_MAP` |

네 번째는 작성자 표에 없다. 구 `GOOGLE_MAP` 분기는 `WebClient.builder()` 를 커넥터 없이 쓴다 —
즉 **구도 이 한 갈래는 read 무한**이었다. 이건 작성자에게 유리한 사실인데 놓쳤다.

신은 `RestClientSupport.java:102-104` 의 `HTTP_1_1_REQUEST_FACTORY` 하나로 전부 보낸다.
`JdkClientHttpRequestFactory.readTimeout` 은 `@Nullable` 이고 기본 null 이며
(`spring-web-7.0.8` 소스 `:44`), `JdkClientHttpRequest.java:119` 가 `if (this.timeout != null)` 로
감싸므로 **null = 타임아웃 자체를 안 건다**. `HttpClient.newBuilder()` 에도 `.connectTimeout()` 이 없다.
무한이 맞다.

**왜 스레드 고갈보다 나쁜가.** 이식본은 외부호출이 전부 `@Transactional` 안에 있다(구 그대로):

- `StoreFacade.java:121` `registerStoreFcd` → `:131-134` 에서 `ExternalClientService` **4회**
  (네이버·카카오·모두닥·구글). 그 앞 `:123` 에서 이미 store 를 INSERT 했다.
- `StoreFacade.java:154` `completedReviewAnalysisFcd` → `:158` DB UPDATE 후 `:171` 메일 + `:172` 알림톡
- `StoreFacade.java:231` `authenticateTeamAuthFcd` → `:250` ML 로그인
- `AppUserChannelAccountService.java:70,116,153` / `TeamChannelAccountFacade.java:73,131,170,219` → ML
- `OrganizationAppUserFacade.java:59` → `:126` 초대 메일

Hibernate 의 기본 커넥션 핸들링은 지연 획득·트랜잭션 종료 후 반납이다. 위 경로들은 **DB 를 먼저 만지고
그다음 외부를 부르므로**, 외부가 안 끊기면 커넥션이 커밋까지 반납되지 않는다. 즉 느린 상대 하나가
Tomcat 스레드(기본 200)와 Hikari 커넥션을 **동시에** 영구 점유한다. 구는 최악 60초면 풀렸다.

**게이트웨이가 먼저 끊어주지 않는다.** `GatewayConfig.java:52` 는 `connectTimeout(10s)` 만 두고,
주석 `:53-55` 는 "헤더 수신까지의 지연은 커넥트 타임아웃이 잡는다"고 적었는데 **이건 사실이 아니다**.
`HttpClient.connectTimeout` 은 TCP 연결까지만 덮는다. 최초 바이트까지의 지연은 `HttpRequest.timeout()`
소관인데 `ProxyFilter.java:164-173` 은 그걸 설정하지 않고 `send()` 한다. 백엔드가 연결은 받고
응답을 안 주면 게이트웨이도 같이 무한 대기한다. **체인 어디에도 끊는 지점이 없다.**

**영향 범위**(요청받은 대로 실측). 이번 슬라이스에서 외부를 실제로 타는 경로는 최소 14개다 —
P2-D 6(store 4 + notice 2), P2-C 3, P2-B 5. 여기에 P1 검색 5경로가 같은 팩토리를 탄다.
그리고 그 `static final` 팩토리는 legacy 밖 **9개 모듈 약 35개 RestClient** 가 공유한다
(`modules/{chrono,crawling,external,medicontents,notification,product,user}` 전수).

**고칠 방향.** 값 하나로 덮지 말라는 건 맞는데, "그래서 안 고친다"는 결론은 과하다. 둘로 나뉜다.

(a) **`:core` 공유 팩토리에는 안전망 기본값 10s/60s 를 넣는 게 맞다.** 근거는 이 클래스가
스스로 "Feign 패리티"를 표방한다는 점이다. OpenFeign 의 `Request.Options` 기본값이 정확히
connect 10s / read 60s 인데, 마이그레이션 커밋 `025c270` 이 제거한 `spring.cloud.openfeign.client.config`
블록에는 `url` 키밖에 없었다(`user.yml` diff 실측). 즉 **35개 클라이언트 전부가 암묵적 10s/60s 로
돌다가 이 이관에서 조용히 무한이 됐다.** 10s/60s 를 넣는 건 변경이 아니라 **원복**이다 —
HTTP/1.1 고정을 정당화한 그 파일 자신의 논리와 같다. (다만 SSE·대용량 스트리밍 클라이언트가
섞여 있지 않은지는 확인이 필요하다. 나는 확인하지 않았다.)

(b) **legacy 는 `LegacyRestClients` 에서 프로파일별로 덮으면 된다.** 덮어쓰기 지점이 이미 있다 —
`DefaultRestClientBuilder.requestFactory` 는 단순 last-wins 세터다(`:360-363` 실측). 따라서
`LegacyRestClients.builder(baseUrl, Profile)` 로 (10s,60s)/(5s,10s)/(30s,30s) 팩토리 3개를
프로파일당 **하나씩** 만들어 두고 골라 끼우면 `:core` 를 안 건드리고 구 값을 그대로 재현한다.
`HttpClient` 는 스레드 안전하고 커넥션을 풀링하므로 프로파일당 1개가 맞다(호출마다 만들면 풀이 샌다).

여기서 하나 걸리는 게 있다. `MlService.java:127` 이 구의 **세 갈래를 한 줄로 합쳤다**:

```java
case NAVER_MAP, MODOODOC, GANGNAM_SISTER_MAP, YEOSHIN_TICKET, KAKAO_MAP, GOOGLE_MAP -> {
```

구는 이 여섯이 스택도 타임아웃도 파싱 방식도 달랐다. 그래서 **메서드 단위 프로파일로는 부족하고
이 메서드 안에서 채널별로 갈라야 한다.** 합친 것 자체는 저장소 규약(RestClient 단일) 때문에
불가피하지만, 그 대가가 문서화되지 않았다.

덧붙여 `MlService.java:42-45` 주석의 "남는 차이는 카카오맵의 응답 타임아웃 30초뿐"은 **틀렸다.**
(i) `GOOGLE_MAP` 의 커넥트 30초도 사라졌고, (ii) 구 `MlService.kt:95` 의
`log.info("bodyStr: $bodyStr")` — ML 로그인 원문 바디(세션 쿠키 포함) 통째 로깅 — 도 함께 없어졌다.
(ii)는 보안상 잘한 제거지만 "차이는 30초뿐"이라고 적어두면 다음 사람이 속는다.

---

### 2. [BLOCKER] Kafka `__TypeId__` — 발행 DTO 6개 중 **3개가 여전히 신 패키지로 나간다**

작성자 문서 5번은 "`TYPE_MAPPINGS` 에 구 FQCN 2개를 추가했다"로 끝난다. 그런데 이 모듈이 발행하는
DTO 는 6종이고, 매핑은 3개뿐이다(`LegacyKafkaProducerConfig.java:105-108`). 남은 셋을 직접 확인했다:

| 발행 지점 | DTO | 구 FQCN | 신 `__TypeId__` |
|---|---|---|---|
| `EventSchedulerService.java:40` | `ProducerTargetUpdate` | `legalcare.medilawyer.db.dto.kafka.…` | `legalcare.medilawyer.**legacy**.db.dto.kafka.…` |
| `StoreFacade.java:180` | `ProducerPostExtractCompleted` | `legalcare.medilawyer.db.dto.kafka.…` | `…**legacy**.db.dto.kafka.…` |
| `KafkaProducer.java:76` (← `StoreFacade.java:139,213`) | `MlReviewAnalysisReq` | `legalcare.medilawyer.db.dto.ml.…` | `…**legacy**.db.dto.ml.…` |

패키지 선언을 양쪽에서 대조해 확인했다. `ADD_TYPE_INFO_HEADERS=true` 라 헤더는 실제로 실린다.
바디 바이트는 구와 같으니 **소비자가 타입 헤더로 역직렬화하는 순간에만 깨지고, HTTP 는 200 그대로**다.
P1 에서 잡힌 것과 완전히 같은 함정인데, 가드를 만들어 놓고 이번에 늘어난 3개에 적용하지 않았다.
`LegacyKafkaProducerConfig.java:52-56` 주석과 `LegacyKafkaProducerConfigTest` 가 바로 이 실패 모드를
경고하고 있는데 정작 그 셋에 대한 테스트가 없다.

이건 인벤토리 51경로 중 `POST /store/register`·`/store/store-channel`·`/store/store-channel/completed`
세 개의 쓰기 경로에 직접 걸린다.

---

### 3. [BLOCKER] 요청 DTO의 non-null 계약이 사라져 **구가 거부하던 요청이 신에서는 DB 를 쓴다**

구 요청 DTO 는 Kotlin `data class` 의 non-null 주생성자 파라미터이고, 구 매퍼는
`ObjectConfig.kt:16` 에서 `KotlinModule` 을 등록한다. 그래서 필드가 빠진 바디는 **역직렬화 단계에서
터지고 500** 으로 끝난다 — 컨트롤러 본문에 진입하지 못한다.

이식본은 `@NoArgsConstructor @Setter` POJO 라 그냥 null/0 으로 바인딩된다. 예:

```java
// StoreRegistrationReq.java:9-16   (구 StoreRegistrationReq.kt:5-8 은 val storeName: String)
@Getter @Setter @NoArgsConstructor
public class StoreRegistrationReq { private String storeName; private String storeDetail; }
```

같은 패턴이 `NewStoreChannelReq.java:15-18`, `StoreAuthenticationReq.java:13-17`,
`ReviewCompletionRes.java:22-45` 에도 있다. 결과가 경로마다 다르다:

- `POST /api/v1/store/store-channel` 에서 `storeId` 누락 → 구 **500**, 신은 `storeId=0` 으로
  진입해 `StoreFacade.java:185-190` 에서 **400 `COMMON-001`**. 상태코드와 바디가 둘 다 다르다.
- `POST /api/v1/store/store-channel/completed` 에서 최상위 `result` 누락 → 구 **500, DB 무변화**.
  신은 `result` 를 읽지도 않아 **200** 을 주고 `store_channel` UPDATE + `crawling_fail_log` INSERT +
  Kafka 발행까지 **전부 수행**한다.
- `POST /api/v1/store/register` 에서 `storeName` 누락 → 구는 바인딩에서 죽어 DB 무접촉,
  신은 `StoreFacade.java:123` 에서 `saveStoreSvc(null, …)` 로 INSERT 를 시도한다(롤백되지만
  IDENTITY 시퀀스는 이미 전진).
- `POST /api/v1/stores/{storeId}/channels/{channelId}/auth` 에서 계정정보 누락 → 신은
  `accountId=null` 로 **실제 ML 외부호출을 발사**한다.

**그리고 이건 P2-D 만의 문제가 아니다.** `@NoArgsConstructor`+`@Setter` 조합을 `db/dto` 전수로
훑으니 P2-A/B/C 요청 DTO 에도 그대로 있다 — `OrganizationRegisterationReq`,
`UpdateOrganizationNameReq`, `TeamRegistrationReq`, `TeamAppUserRegistrationReq`,
`TeamAppUserPermissionUpdateReq`, `OrganizationRoleUpdateReq`, `InviteOrganizationAppUserReq`,
`AppUserChannelAccountLoginReq`, `TeamChannelAccountLoginReq`, `MlNaverCaptchaReq`,
`MlReLoginReq`, `MatchStoreReq`. 두 개를 표본으로 구와 직접 대조했다:

```
구 OrganizationRegisterationReq.kt : val teamAuthId: String (non-null), val storeId: Long, val organizationName: String
신 OrganizationRegisterationReq.java:13-15 : String / long / String  ← 전부 널 허용, storeId 는 누락 시 0
구 MatchStoreReq.kt : val storeId: Long, val matchType: AppUserStoreMatchRequestType (non-null)
신 MatchStoreReq.java:14-15 : long / AppUserStoreMatchRequestType  ← matchType 누락 시 null 로 진입
```

즉 **51경로 중 쓰기 요청 바디를 받는 것 거의 전부**가 같은 계약 완화를 겪었다. 개별 경로마다
결과가 다르므로(위 네 사례처럼 500→400, 500→200+DB쓰기, 500→외부호출) 일괄 판정이 안 되고
경로별로 봐야 한다.

리플레이 코퍼스가 정상 바디만 담고 있으면 이것도 안 잡힌다. AC4 의 "행 변화 일치"를 정면으로 흔든다.

---

### 4. [HIGH] `!!` 널 단언 2곳이 사라져 500 이 200 이 되고, 운영에서 `to: null` 알림톡이 나간다

- 구 `NoticeController.kt:85` 는 `storeChannel!!` 이다. 네이버맵 채널이 없는 스토어면 즉시 NPE → 500.
  신 `NoticeController.java:111` 은 그냥 넘긴다. `Post.from` 이 필드에 담기만 하고 역참조하지 않아
  흐름이 계속되고, 비운영에서는 `serviceMode` 가드에 막혀 **빈 바디 200** 이 나간다.
- 구 `BizMessageService.kt:340` 은 `to = appUser.phone!!` 이다. 전화번호 없는 회원이면 NPE
  (→ `/notice` 는 사용자별 try/catch 로 건너뜀, NCP 미호출).
  신 `BizMessageService.java:261` 은 `appUser.getPhone()` 그대로라 **`to: null` 로 NCP 요청을 실제로 보낸다.**
  운영(`serviceMode == prod`)에서만 드러나므로 개발계 리플레이로는 영원히 안 보인다.

---

### 5. [HIGH] 추적 대상 13파일 중 **9개가 어떤 R# 에도 매핑되지 않는다** — 게이트웨이 신뢰헤더

`git diff HEAD` 로 잡히는 변경 중 R4 에 매핑되는 건 **0건**이다(R4 는 전부 미추적 `modules/legacy` 안에 있다).
나머지가 R1/R2 인 건 맞는데, 아래 묶음은 R1~R11 어디에도 없다:

- `gateway/auth/IdentityHeaders.java:38,47-58` — `X-Auth-Trust` 헤더 신설
- `gateway/config/GatewayProperties.java:18-24`, `GatewayConfig.java:79-89` — `trust-secret` 배선
- `gateway/auth/IdentityHeaderFilter.java:70-78` — **`/api/v1` 의 토큰 출처를 쿠키 → `Authorization` 헤더로 변경**
- `core/filter/TokenVerificationFilter.java:68-75` — `TrustedIdentityResolver` 가 non-null 이면
  **JWT 검증을 통째로 건너뛰고** `chain.doFilter` 로 직행하는 분기
- `app/.../LegacyGatewayIdentityResolver.java`(미추적) — 그 리졸버

두 가지가 걸린다. 첫째, `Routes.java:34` 가 아직 `ServiceId.LEGACY` 이므로 `/api/v1` 토큰 출처 변경은
**지금 서비스 중인 구 스택에 그대로 적용된다.** 게이트웨이 무수정이 AC 인 단계에서 라이브 경로를 건드린 셈이다.
둘째, 우회 분기 자체는 방어가 촘촘하다 — 시크릿 비면 첫 줄에서 null 반환, `MessageDigest.isEqual` 상수시간 비교
(`LegacyGatewayIdentityResolver.java:151-157`), `ProxyFilter.java:221-222` 가 인바운드 `X-Auth-*` 를 접두 전량 제거.
설계는 일관된다. 다만 만료도 바인딩도 없는 평문 내부 헤더 베어러 시크릿이고, 그 파일 주석 스스로
booster 가 `127.0.0.1:18081` 로 직접 닿는다고 적고 있다. **이건 요구사항과 리뷰를 거쳐야 할 결정이지
P0 의 부수효과로 들어올 성질이 아니다.**

곁가지로 `gateway/application-dev.yml:21` 에 구 legacy-service 액세스 키와 같은 값이라는
JWT 서명키 리터럴이 하나 더 늘었다. 이미 `HEAD` 의 8개 파일에 있는 값이라 새로 만든 죄는 아니고
`application-release.yml` 에는 안 넣었지만, 노출면이 넓어지는 건 맞다.

---

### 6. [HIGH] DDL 이 "접촉분만"이라는 R11 을 넘는다

`legacy-absorption-p1b-schema.sql:2` 는 헤더에서 **"운영 대비 부족 42테이블 전량 생성"** 이라고
스스로 선언한다(`CREATE TABLE` 43회). 그 대상 중 `kakao_alimtalk_send_history`(+27 파티션),
`kakao_alimtalk_blocklist`, `boost_present_send_request`, `boost_review_match_analysis`,
`boost_review_platform_click`, `boost_store_present_setting`, `crawling_daily_stats` 는
`modules/legacy` 전체에서 **참조 0건**이다. `p1-schema.sql:35,83` 의 `boost_message_delivery`·
`boost_review_request_setting` 도 마찬가지고, 그 파일 자신의 주석이 이들을 P4 `/internal/**` 와
`/api/v2` 에 묶는다 — `/api/v2` 는 요구사항 11행 제외 목록에 있다.

`legacy-absorption-p1c-existing-align.sql` 은 더 나아가 **기존 테이블 5개를 변경**한다.
`:61-62` 컬럼 추가, `:71` `boost_present_order.uuid` 에 **UNIQUE 인덱스**(이미 커밋된 `modules/product`
v2 주문 경로의 쓰기 의미가 바뀐다), `:110-111` 은 스스로 9,964,949행·6.1GB 라고 적은 `post` 테이블에
`CREATE INDEX CONCURRENTLY` 2개. 트랜잭션 처리와 INVALID 인덱스 정리(`:90-105`)는 잘 짜여 있지만,
인덱스 추가는 요구사항 11행이 제외한 "성능개선"이고 51경로 중 어느 것도 이 5개 테이블에 쓰지 않는다.

부팅 시 실행되는 Flyway/Liquibase 가 저장소에 아예 없어서 런타임 위험은 없다. 순수하게 **범위** 문제다.

---

### 7. [MEDIUM] Kafka DTO 2개에 마스킹 `toString()` 이 빠졌다 — 애노테이션 한 줄이면 PII 가 로그로 샌다

`KafkaProducer.java:55` 는 `log.info("Data = {}", data)` 로 무타입 `Object` 를 찍는다.
`ProducerUserPostReplyCreate.java:63` 은 마스킹 `toString()` 을 갖고 있고, 그 주석 `:19-20` 이
"다른 DTO 가 같은 자리를 지나가도 각자 자기 `toString()` 으로 책임지게 한다"고 규칙을 명시한다.
그런데 이번에 늘어난 `ProducerUserPostDeleteRequest`·`ProducerPostPunishRequest` 는 둘 다
`email`/`appUserName`/`phone` 3종을 들고 있으면서(각 `:25-27`) `toString()` 이 없다.

지금 안 새는 이유는 `@Getter` 만 있어서 `Object.toString()` 이 해시를 찍기 때문 — **우연이다.**
누가 `@ToString`·`@Data` 를 붙이거나 `record` 로 바꾸는 순간
`POST /api/v1/products/{productName}/purchases` 가 고객 이메일·실명·전화번호를 로그에 쓰기 시작한다.
규칙을 문서에 적었으면 테스트로 못박거나, 마스킹을 `KafkaProducer` 쪽으로 올려야 한다.

---

### 8. [MEDIUM] `@Transactional` 이 빠졌는데 주석은 "구 그대로 없다"고 적혀 있다

- `PostService.java:64-65` `getPeriodCountsSvc` — 구 `PostService.kt:60-61` 에는 `@Transactional` 이 **있다.**
- `PostService.java:70-71` `getPost` — 구 `PostService.kt:87-88` 은 `@Transactional(readOnly = true)` 다.

둘 다 Javadoc 이 "구 그대로 `@Transactional` 이 없다"고 적었는데 사실이 아니다. 지금은 호출부가
상위 트랜잭션 안이라 `PROPAGATION_REQUIRED` 로 합류해 동작이 같고 OSIV 도 켜져 있어 응답 바이트는
동일하다. 문제는 **틀린 주석이 다음 수정자를 오도한다**는 것이고, 이 메서드가 트랜잭션 밖에서
호출되는 순간 조용히 달라진다.

---

### 9. [MEDIUM] 엔티티 ID 가 primitive → boxed 로 바뀌어 `persist` 가 `merge` 로 간다

구 `ProductPost.kt:24` 의 `val id: Long = 0` 은 JVM 에서 **primitive `long`** 이다.
신 `ProductPost.java:50` 은 `private Long id = 0L` — 박싱됐다.
Spring Data 의 `AbstractEntityInformation.isNew()` 는 `idType.isPrimitive()` 로 갈린다:
primitive 면 `id == 0` → `true` → `em.persist()`, 박싱이면 `id == null` → `0L` 이라 `false` → `em.merge()`.
둘 다 INSERT 로 끝나지만 merge 는 SELECT 를 한 번 더 하고 **다른 인스턴스**를 돌려준다.

이 경로는 `ProductPostFacade.java:124,226` 이 반환값(`savedProductPost`)을 쓰고 있어 기능상 손실은 없다.
다만 같은 패턴이 `Store`·`AppUser`·`StoreChannel`·`AppUserStore`·`Product`·`PostReply` 등
**11개 엔티티**에 있어서(`private Long id = 0L` 전수 grep), 반환값을 안 쓰는 저장 경로가 하나라도
생기면 그때 드러난다. 지금 고칠 것은 아니고 **알고는 있어야 한다**.

---

### 10. [LOW] 잔가지 묶음

- `CrawlingFailLogRecorder.java:65,93` 이 구 `runCatching`(`Throwable` 포획)을 `catch (Exception)` 으로
  좁혔다. 클래스 계약이 "기록 실패가 본 흐름을 절대 깨지 않는다"인데 `Error` 가 새어나가면
  `/store/store-channel/completed` 가 500 이 된다.
- `StoreChannelHelper.java:138-139,167` 의 `toLowerCase()` — 구 Kotlin `lowercase()` 는 `Locale.ROOT` 고정인데
  인자 없는 Java 판은 JVM 기본 로케일을 탄다. `store_channel.match_score` 에 흘러든다. 터키어형
  로케일에서만 갈리므로 실질 위험은 낮지만 `Locale.ROOT` 를 넘기면 끝나는 일이다.
- `CrawlingFailLogRecorder.java:54` 가 앱 공용 매퍼 대신 `JsonMapper.builder().build()` 로 로컬 매퍼를
  만든다. 현재 페이로드가 스칼라뿐이라 바이트는 같지만, 날짜·enum 이 들어오는 순간 조용히 갈린다.
- `LegacyKafkaProducerConfig.java:79` 의 `${spring.kafka.bootstrap-servers:}` 빈 기본값 — 구
  `KafkaProducerConfig.kt:15` 는 기본값이 없어 **부팅이 실패**했다. 신은 부팅에 성공하고 첫 발행에서,
  그것도 `@Transactional` 안에서 터진다.
- `MlAppUserChannelAccountLoginRes.java:95-97` 의 `TUID`/`TSID`/`UUID` 는 이 모듈에서 유일하게
  **핀 없이 라이브러리 기본값에 기대는 키**다. Jackson 3 가 구식 mangling 을 버려서 지금은 맞지만
  (Jackson 2 였다면 `tuid`), `@JsonProperty` 한 줄씩이면 끝난다. `LegacyP2ResponseShapeTest:154-160`
  이 바이트를 잡고 있어 구멍은 아니고 견고성 문제다.
- `StoreChannelService.java:227-231` 이 구 `StoreChannelService.kt:261-262` 의 `log.error` 를 잃었다(로그만).
- `CommunityController.java:19-21` 은 핸들러가 0개인 `@RestController` 다. 구도 같아서 1:1 로는 맞지만
  아무것도 서빙하지 않는 빈이다.

---

## 지적 (P2-A/B/C 29경로 — 대조 완료분)

문서 작성 중 29경로 대조가 끝나 아래를 추가한다. 응답 DTO 키·순서·`@JsonPropertyOrder`(25개 전수),
밑줄 쿠키 키 핀, 에러코드·메시지 문자열, 열거형 12개, `@Transactional` 배치, **positional 생성자
전환 지점 전수**(`AppUser`·`TeamChannelAccount.from`·`PushEmail` 10인자·`MlReLoginRes` 10인자 — 구가
named argument 를 쓰던 자리까지)는 어긋난 곳이 없었다. 남은 것이 아래 넷이다.

### 11. [BLOCKER] ML **응답** DTO 의 non-null 계약도 사라졌다 — 로그인 실패가 "성공"으로 나간다

지적 3번이 요청 바디 쪽이라면 이건 응답 역직렬화 쪽이고, 결과가 더 나쁘다.

구 `MlAppUserChannelAccountLoginRes.kt:14,17` 의 `result`·`status` 는 기본값 없는 non-null 이다
(`MlKakaoSmsLoginRes.kt:11,14` 도 같다). ML 이 그 키를 빼먹거나 `null` 로 주면 `KotlinModule` 이
`MissingKotlinParameterException` 을 던지고, 그게 HTTP 호출 안에서 터진다:
- `authenticateChannelAccount` → 구 `MlService.kt:116-120` 의 catch → **400 `REQUEST-003`**
- `kakaoSms`·`naverCaptcha` → 구 `MlService.kt:139-141,160-162` 에 catch 가 없다 → **500 `SERVER-001`**

신 `MlAppUserChannelAccountLoginRes.java:29-30`·`MlKakaoSmsLoginRes.java:13-15` 는
`@Setter @NoArgsConstructor` 라 전 필드가 널 허용이다. 그래서 `result` 가 null 로 채워지고,
컨트롤러가 이렇게 판정한다:

```java
// AppUserChannelAccountController.java:72-74
String message = mlResponse.getResult() == ChannelAccountLoginResult.FAIL
        ? "1차 로그인 시도에 실패했습니다."
        : "1차 로그인 시도에 성공했습니다.";
```

`null != FAIL` 이므로 **200 + "1차 로그인 시도에 성공했습니다." + `{"result":null,"status":null,…}`**.
같은 패턴이 `:94-96`(2차 카카오)과 `OrganizationController` 채널계정 4경로에 있다.
**ML 로그인 실패가 클라이언트에는 성공으로 보인다.** 방향이 구 4xx/5xx → 신 200 이라
AC3 판정("구 200 → 신 비200 0건")이 그대로 통과시킨다.

(미지 enum 값은 양쪽 다 던진다 — `LegacyRestClients` 가 구 `ObjectConfig` 동등 매퍼로 덮어서
`READ_UNKNOWN_ENUM_VALUES_AS_NULL` 이 꺼져 있다. 갈리는 건 **키 누락·명시적 null** 뿐이다.)

### 12. [MEDIUM] 토큰의 조직권한 맵 — 부분 실패 의미가 뒤집혀 **권한이 넓어진다**

구 `JwtAuthenticationFilter.kt:46-69` 는 `associate{}` 전체(`:55-57`)를 **하나의 try** 로 감싼다.
엔트리 하나라도 `OrganizationRole.valueOf()` 가 실패하면 `:66-68` 이 받아 **맵 전체를 비운다**.
신 `LegacyAppUserResolutionInterceptor.java:114-130` 은 엔트리별 try/catch 라 **나쁜 항목만 버린다.**

즉 알 수 없는 역할명이 든 토큰에서 구는 역할 게이트가 걸린 조직 엔드포인트 7개를 전부
`400 AUTH-002`("해당 조직에 권한이 없습니다.")로 막았는데, 신은 **나머지 조직을 통과시킨다.**
역할 enum 에 값을 추가·개명하는 순간(구토큰이 살아 있는 배포 창) 현실이 되는 시나리오다.
견고성으로는 신이 낫지만 **인가 판정이 완화되는 방향**이라 1:1 이관에서는 구 의미를 따라야 한다.

### 13. [LOW] `TODO()` 를 `UnsupportedOperationException` 으로 바꾸면 500 바디가 달라진다

구 `OrganizationFacade.kt:99` 의 `ORGANIZATION_SUPERVISOR -> TODO()` 는
`kotlin.NotImplementedError` 를 던지는데 이건 `Error` 하위다. 구 핸들러에는
`@ExceptionHandler(Exception::class)`(`GlobalExceptionHandler.kt:41`)뿐이라 **안 잡히고**
컨테이너 `/error` 로 떨어져 부트 기본 바디(`timestamp/status/error/path`)가 나간다.
신 `OrganizationFacade.java:115-116` 은 `UnsupportedOperationException`(=`Exception`)이라
`LegacyExceptionAdvice.java:59` 가 잡아 legacy `ErrorResponse` 모양으로 응답한다.
둘 다 500 이지만 **바이트가 다르다.** `PATCH /organizations/{id}/users/{targetUserId}/role` 에
`"organizationRole":"ORGANIZATION_SUPERVISOR"` 로 도달한다.

### 14. [LOW] 예외 메시지 문자열이 `errorMessage` 로 흘러 나가는데 두 군데서 달라졌다

구 응답 바디의 `errorMessage` 는 예외 메시지를 그대로 싣는다. 그래서 아래 둘은 바이트 차이다.
- `TeamChannelAccountFacade.java:253-266`·`AppUserChannelAccountService.java:205-218` 이
  복호화 실패를 `new IllegalStateException(e)` 로 감싼다. `getMessage()` 가 `cause.toString()` 이라
  `"javax.crypto.BadPaddingException: …"` 가 되는데, 구(`TeamChannelAccountFacade.kt:46-47` 등)는
  원본 메시지를 그대로 흘린다. `legacy.yml:27-28` 의 crypt 키·IV 가 자리표시자라
  `…/channel-accounts/relogin` 은 **지금 상태로는 매번 이 경로를 탄다.**
- ML·메일 non-2xx 시 구는 `RestTemplate`/`WebClient` 의 `"500 Internal Server Error: [<body>]"` 를
  `"1차 로그인에 실패했습니다. 원인 : …"`(`MlService.kt:119`)·`"Email 전송에 실패했습니다. = …"`
  (`PushEmailService.kt:100`)에 접어 넣는데, 신은 `FeignParityStatusHandler.java:56,60` 이 만든
  다른 문구가 들어간다. 채널계정 4경로와 `POST …/invitations` 에 걸린다.

---

## 지시 8항목에 대한 답

**1. 타임아웃** → 지적 1번. 외부호출 경로 전수·구 프로파일 매핑(넷)·운영 영향 판정(스레드가 아니라
커넥션 풀, 게이트웨이도 안 끊음)·고칠 방향(`:core` 안전망 10s/60s 원복 + `LegacyRestClients` 프로파일별
오버라이드, 오버라이드 지점은 이미 존재)·영향 범위(이번 14 + P1 5 + 타 모듈 35 클라이언트) 모두 위에 적었다.
채널 분기 합침 때문에 **메서드 단위가 아니라 채널 단위 프로파일**이 필요하다는 게 핵심이다.

**2. `AppUserService` 삭제 사고** → **동작 동일 맞다. 내가 구 Kotlin 과 4메서드 전수 대조했다.**
`getAppuserInfoSvc`(`@Deprecated`+`@Transactional`), `getAppUser(email,usable)`,
`getAppUserOptional`, `getAppUser(appUserId,status,usable)` — 애노테이션·readOnly·에러코드·메시지 문자열,
그리고 구가 인자 `usable` 대신 리터럴 `true` 를 넘기는 버그(`AppUserService.kt:54` → `.java:62`)까지 일치한다.
구의 `.also { log.error }` 가 `throw` 가 아니라 `CustomException` 표현식에 붙어 **로그가 먼저 실행**되는데,
신도 로그 → throw 순서라 관찰 순서가 같다. 구에만 있는 5번째 메서드
`getAppUser(email,status,usable)` 는 호출부가 없어 미이식이 맞다.
특히 위험했던 건 구가 **named argument** 로 만들던 `AppUserInfoRes` 를 신이 **positional** 로 바꾼 것인데,
`AppUserInfoRes.java:18` 의 파라미터 순서가 구 `AppUserInfoRes.kt:6-10`(appUserId, email, appUserName,
phone, role)과 같아서 필드가 뒤바뀌지 않았다. 확인했다.
복구 불가 잔여분: `build/classes` 의 `.class` 는 16:45 로 **재작성 이후**고 `compileTransaction/stash-dir`
에도 없어서 P2-B 원본은 남아 있지 않다. 따라서 "P2-B 판에 다른 애노테이션이 있었는가"는 **확인 못 했다.**
컴파일은 메서드 시그니처만 잡지 애노테이션 차이는 안 잡는다.

**3. 밑줄 필드** → **이번 22경로에는 없다는 주장이 맞다.** 다만 슬라이스 경계를 넓혀 8개 컨트롤러에서
도달 가능한 구 Kotlin DTO 전수를 훑으면 밑줄 프로퍼티가 `CookieDict` 4개에 **25개** 있고
(`MlAppUserChannelAccountLoginRes.kt:37-46` 외), 전부 `@JsonProperty` 로 핀이 박혀 있다.
`_kau` 류 구멍은 닫혔다. `eMail` 형 0건, `isX` 13개 전부 게터 핀.

**4. 마스킹 계약** → **구와 값이 같다.** 구 `TeamService.kt:163` `masking = false`,
신 `TeamService.java:229` 6번째 인자 `false`. DTO 도 필드명·선언순서·타입이 완전 일치
(`PresentHistoryPageByTeamRes.kt:3-10` ↔ `.java:13-18`, `PresentHistoryByTeamRes.kt:5-10` ↔ `.java:22-26`).
**같은 성격의 플래그는 양쪽 통틀어 이 하나뿐이다** — `masking|anonymi|blind|hidePhone|hideName|isMasked`
전수 grep 결과 다른 곳엔 없다. `customerPhone`(= `boost_customer.id` = 전화번호)이 JSON 과 xlsx 양쪽에
평문으로 나가는 것도 구와 동일하다. 바꾸면 1:1 위반이 맞으니 현행 유지 판단에 동의한다.

**5. Kafka** → 지적 2번. `ProductPostFacade` 쪽 2종은 제대로 매핑됐지만 `StoreFacade`·
`EventSchedulerService` 쪽 3종이 빠졌다. 그 외 설정은 `max.request.size` 524288000,
`security.protocol` PLAINTEXT, `ADD_TYPE_INFO_HEADERS` true 까지 구와 일치하고, 토픽 23개 문자열,
키 없는 2-arg `send`, 트랜잭션 안 발행(구도 동일)도 일치한다. `linger.ms` 가 kafka-clients
3.7.1→4.1.2 버전 차로 0→5ms 바뀌지만 와이어 포맷과 무관하다.

**6. 1:1 정확도** → **51경로 전량 대조했다.** 지적 3·4·8·9·10·11·12·13·14번이 결과물이다.

P2-D 22경로는 URL·메서드·파라미터·응답 키/순서·상태코드·에러코드·통계 계산식(30일 창 4/5,
키워드 랭킹만 무제한, 정수 나눗셈, `endDate.plusDays(1)` 배타 상한)·엑셀(파일명·
`ContentDisposition…filename(name, UTF_8)`·시트명·5컬럼 순서·`yyyy-MM-dd HH:mm`)까지 대조했다.
엑셀 응답 조립은 구 Kotlin 과 구문 단위로 동일함을 확인했다.

P2-A/B/C 29경로는 URL·메서드·경로변수 전 쌍 일치(`@RequestParam` 은 이 29개에 없다), 응답/요청 DTO
25개의 필드명·선언순서·중첩타입까지 일치, `ApiResponse`/`ErrorResponse`/`ErrorType`/`ResponseType`
및 열거형 12개 코드·메시지·상태·순서 동일, `CustomException` 분기와 메시지 문자열 전량 일치
(의도적으로 보존한 구 버그 — `OrganizationService.java:33` 의 `REQUEST_FAIL_MAIL` 오용,
`OrganizationAppUserFacade.java:172-173` 의 삭제 카운트 뒤바뀜, `teamappUserInfoList` 오탈자,
`TeamChannelAccountService` 의 `usable` 무시 — 포함). `@Transactional`/`readOnly` 는 구가 빠뜨린
세 군데까지 메서드 단위로 같다. **positional 생성자 전환 지점 전수에서 필드 뒤바뀜 0건.**
남은 차이가 지적 11~14번이다.

**7. 범위 이탈** → 지적 5·6번. `modules/legacy` main 308개 자체는 깨끗하다: 273개가 51경로에서 도달,
26개가 P1(R3) 전용, 1개가 P0, 8개가 스프링 진입점 인프라, **완전 사장 코드 0개**.
`@Scheduled`·`@EventListener`·`ApplicationRunner`·`@KafkaListener`·`@Async` 전부 0건이고
`build/` 는 `.gitignore` 이중 커버라 `git add` 해도 안 딸려온다(322개 중 build 0개, 실측).
범위 이탈은 모듈 안이 아니라 **게이트웨이·`:core` 추적 파일 9개와 SQL 3개**에 있다.

**8. fintech-safety** → **"과금 경로 없다"는 판정은 맞다. 직접 확인했다.**
`getPrice()`·`getItemPrice()` 호출부가 신 모듈 전체에서 **0건**이고, 구에서도 `Product` 는
`product_id` FK 공급용으로만 조회된다. `boost_present_ledger` 는 이 슬라이스에서 읽기 전용이다 —
유일한 사용처가 `BoostPresentOrderRepositoryImpl.java:115` 의 `leftJoin` 이고, `price`/`itemCount` 는
select 도 안 된다. 엔티티가 `@Getter` + `protected` 무인자 생성자뿐이라 세터가 없어 dirty checking
UPDATE 도 불가능하고 리포지토리 자체가 없다. `Id.DEDUCTION` 은 금융 차감이 아니라 Jackson 의
**구조 기반 서브타입 추론**이다(`PurchaseProductRequest.java:17`, 구 `PurchaseProductPostReq.kt:6` 동일).
금액 타입도 구와 일치한다(`BoostPresent.itemPrice`·`BoostPresentLedger.price` 는 양쪽 `BigDecimal`,
`Product.price` 는 양쪽 `Int`/`int`).

다만 두 가지는 기록해 둔다. 첫째, `POST /products/{productName}/purchases` 의 `product_post` INSERT 에
락이 전혀 없다 — 모듈 전체에 `@Lock`·`LockModeType`·`@Version` **0건**(구도 동일). 유일한 방어인
유니크 제약 `(app_user_id, post_id, product_id)`(`ProductPost.java:41-44`)이 `ThreadRequest` 분기를
못 덮는다. 그 분기는 `post_id` 가 NULL 인데 PostgreSQL 은 NULL 을 매번 다른 값으로 보므로
**스레드 구매는 무제한 중복 생성된다.** 멱등키(`x-transaction-id`)도 traceId 도 없다. 구와 같아서
1:1 로는 정당하지만, "구도 racy 했다"는 변론은 트래픽을 넘기는 순간 끝난다 — P4/P5 백로그로 올려야 한다.
둘째, 같은 트랜잭션 안에서 Kafka 를 발행하므로(지적 2번 표) 롤백 시 이벤트만 남는 dual-write 가 있다.
이것도 구와 동일하고 작성자가 `ProductPostFacade.java:58-59` 에 적어 뒀다.

---

## 확인 못 한 것 (정직하게)

- P2-A/B/C 29경로는 이번 라운드에서 **처음으로** 교차검토됐다(그전까지 task dir 에 P0 2건·P1 1건뿐).
  전수 대조는 마쳤지만 **한 번도 검토된 적 없던 코드를 한 라운드에 몰아서 본 것**이라,
  P2-D 만큼 여러 각도로 뒤집어보지는 못했다. 지적 11번이 마지막에 나온 것도 그래서다.
- p1/p1b/p1c SQL 이 실제로 `deploy-postgres-new-1` 에 적용됐는지, `p1c` 가 주장하는 사전점검
  (`boost_present_order.uuid` 중복 0건 등)이 성립하는지 — 파일만 읽었고 psql 세션은 열지 않았다.
- 776/0·84/0·`Routes.java:34` 는 오케스트레이터 재실측을 신뢰하고 다시 돌리지 않았다.
- 지적 1번의 (a) 안(공유 팩토리 10s/60s)이 안전한지 — SSE·스트리밍을 쓰는 클라이언트가 35개 중에
  섞여 있는지 확인하지 않았다. 적용 전에 그건 봐야 한다.
- 구 서버 실응답과의 값 단위 대조는 하지 않았다(외부 실호출 금지 · P5 소관).

---

승인하지 못하는 이유는 단순하다. 1·2·3·4·11번이 전부 **HTTP 200 을 유지한 채** 구와 갈리는 것들이고,
리플레이가 P2 의 11.5% 밖에 못 덮는 상황에서는 지금 코드에서 잡는 것 말고 잡을 기회가 없다.

우선순위를 매기면 **11번이 제일 급하다** — ML 로그인 실패가 클라이언트에 "성공했습니다"로 나가고,
구가 4xx/5xx 로 막던 것이 신에서 200 이 되므로 AC3 판정 방향("구 200 → 신 비200")을 정확히 빠져나간다.
3번과 뿌리가 같다(Kotlin non-null → Java 널 허용). 요청·응답 양쪽에서 같은 계약이 통째로 증발했으니
**DTO 단위로 하나씩 고치기보다 이 축을 어떻게 복원할지 방침을 먼저 정하는 게 낫다** —
`@NonNull` 검증이든, `@JsonCreator` + 필수 파라미터든, 역직렬화 후 공통 검증이든.
2번은 P1 에서 이미 한 번 잡힌 함정에 가드까지 만들어 놓고 새로 늘어난 3개에 적용하지 않은 것이라
빠르게 닫힌다. 1번은 이번 슬라이스보다 큰 횡단 과제라 (a)/(b) 를 나눠 판단하면 된다.
5·6번은 결함이라기보다 **R# 없는 변경**이라 승인 전에 요구사항으로 승격하든 되돌리든 정리가 필요하다.

VERDICT: REQUEST_CHANGES
