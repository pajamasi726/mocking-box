# 변경 요약: P1 읽기(저위험) — R3 읽기 계열 7컨트롤러 이식

브랜치 `feature/prod-sync-candidates` @ `/Users/steve/steve/legalcare-renew-prodsync-wt` · **커밋 0건**(HEAD `b42d9b6` 불변)
`medilawyer-boot` **무수정**(`git status --short` 빈 출력, HEAD `930c83cf`) · 게이트웨이 **무수정**(`Routes.java:34` 여전히 `ServiceId.LEGACY`)
**리플레이는 실행하지 않았다**(지시대로 — 진입 조건인 JWT 서명키 불일치 미해소). 준비 상태 점검만 아래 4절에 실측으로 남긴다.

## 변경 파일
| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `modules/legacy/build.gradle` | QueryDSL APT·ulid·spring-kafka 추가(P1 이식분이 쓰는 것) | R3 |
| `modules/legacy/.../legacy/apis/**` (컨트롤러 7 + 서비스 8, 15파일) | 구 7컨트롤러와 그 도달 서비스 1:1 이식 | R3 |
| `modules/legacy/.../legacy/db/domain/**` (엔티티 13 + 리포지토리 20, 33파일) | 구 엔티티·리포지토리 이식(테이블 13종) | R3 |
| `modules/legacy/.../legacy/db/dto/**` (34파일) | 요청·응답·ML·카프카·외부검색 DTO 이식 | R3 |
| `modules/legacy/.../legacy/db/querydsl/QuerydslFilter.java` | 구 QueryDSL 필터 이식(빈 이름 `legacyQuerydslFilter`) | R3 |
| `modules/legacy/.../legacy/enums/**` (21파일) | 구 enum 상수표 이식 | R3 |
| `modules/legacy/.../legacy/client/{external,ml,kafka}/**` (4파일) | 외부검색·ML·카프카 클라이언트 이식(WebClient/RestTemplate → **RestClient**) | R3 |
| `modules/legacy/.../legacy/common/**` (6파일) | `ApiResponse`·`CustomException`·에러봉투 + **legacy 한정 `@RestControllerAdvice`** + `CryptConfig` | R3 |
| `modules/legacy/.../legacy/security/**` (4파일) | 구 `@AuthenticationPrincipal CustomUserDetails` 자리를 booster 필터 애트리뷰트로 재현 | R3·R2 |
| `modules/legacy/src/test/.../{PublicHospitalServiceTest,LegacyExceptionAdviceTest}.java` | 반올림 규칙·구 에러봉투(400/500) 고정 | R3 |
| `app/src/test/.../LegacyP1EndpointMountTest.java` | 실조립 컨텍스트의 매핑 11개를 구 목록과 **양방향** 대조 | R3 |
| `app/src/test/.../LegacyP1ResponseShapeTest.java` | 응답 바디 키 이름·순서를 구 실측 문자열로 고정 | R3 |
| `app/src/main/resources/config/legacy.yml` | 구 설정키(`ml.review.server`·`medilawyer.crypt.*`·검색 API 키) 이식, 비밀값은 env 자리표시자 | R3 |
| `app/src/main/resources/application.yml` | 위 yml 1개를 `spring.config.import` 에 등재(1줄) | R3 |

신규 123파일(`modules/legacy` 아래, P0 의 2파일 제외) + `app` 테스트 2 + `config/legacy.yml` 1, 수정 2줄 단위 1파일(`application.yml`).
**R3 에 매핑되지 않는 변경 파일 없음.** `git status --short` 의 나머지 항목은 전부 P0 산출물이다.

## 이식 경로 목록 (구 소스 ↔ 신 소스 양방향 차집합 0)
구 소스에서 괄호균형 파서로 재추출(P0 라운드1 의 "여러 줄 애노테이션 누락" 재발 방지) → **11개**, 신 Java 소스 재추출 → **11개**, `comm -3` 출력 **0줄**.

| # | 메서드 · 경로 | 구 인증경계 | P-1 접촉 테이블 대조 |
|---|---|---|---|
| 1 | `GET /api/v1/public/hospitals` | PERMIT_ALL | store ✅ |
| 2 | `GET /api/v1/public/hospitals/{storeId}` | PERMIT_ALL | store·store_channel·post·post_word ✅ |
| 3 | `GET /api/v1/post-word/statistics` | PERMIT_ALL | store_channel·post_word·post·store ✅ |
| 4 | `GET /api/v1/store-channel/search/naver` | PERMIT_ALL | DB 무접촉 ✅ |
| 5 | `GET /api/v1/store-channel/search/unofficial/naver` | AUTHENTICATED | DB 무접촉 ✅ |
| 6 | `GET /api/v1/store-channel/search/kakao` | AUTHENTICATED | DB 무접촉 ✅ |
| 7 | `GET /api/v1/store-channel/search/modoodoc` | AUTHENTICATED | DB 무접촉 ✅ |
| 8 | `GET /api/v1/store-channel/search/google` | AUTHENTICATED | DB 무접촉 ✅ |
| 9 | `GET /api/v1/posts` | PERMIT_ALL | post·store_channel·store·post_reply_ai_recommend·post_word·post_reply·product_post·app_user·product·team ✅ |
| 10 | `POST /api/v1/post-reply` | AUTHENTICATED | app_user·team_channel_account·team·store·post·store_channel·post_reply_registration_request·post_reply ✅ |
| 11 | `POST /api/v1/post-reply-ai-recommend` | PERMIT_ALL | post·store_channel·post_reply_ai_recommend ✅ |

`CommunityController` 는 구도 핸들러 0개(클래스 레벨 접두만) — 이식했으나 매핑 0개가 정상이고 P-1 매핑표 135행의 "엔드포인트 아님"과 일치한다.
인증경계는 P0 `LegacyPathAuthPolicy` 판정과 11/11 일치(`P0-auth-boundary.tsv` 대조). 접촉 테이블은 11경로 모두 "신규PG 부재 테이블 없음" 행이라 P-1 DDL 추가분과 무관하다.

## 테스트 결과
```
$ ./gradlew test --rerun-tasks            (apps/booster-app, 전체 스위트)   BUILD SUCCESSFUL in 4m 20s
  클래스 184 · TESTS=703 · FAILURES=0 · ERRORS=0 · SKIPPED=0
  app 203 · core 48 · legacy 8 · chrono 7 · crawling 68 · external 40 · medicontents 18
  · notification 87 · post 67 · product 71 · user 86        (기준선 689 → +14)
$ ./gradlew duplicateClassCheck                                            BUILD SUCCESSFUL
$ ./gradlew test --rerun-tasks            (apps/gateway-app)               BUILD SUCCESSFUL
  TESTS=76 · FAILURES=0 · ERRORS=0        · Routes.java:34 = ServiceId.LEGACY 그대로
```
중간에 한 번 `notification` 의 testcontainers 테스트가 `ContainerFetchException: Can't get Docker image: postgres:16` 로 1건 실패했다.
이식분과 무관한 로컬 Docker 일시 문제이고(같은 테스트가 직전·직후 실행에서 통과, `docker images postgres:16` 존재 확인)
재실행 결과가 위 703/0 이다. 숨기지 않고 적는다.

**새 테스트 14개가 실제로 보장하는 것**(장식 아님 — 이 중 둘은 작성 도중 실제로 실패해서 코드를 고쳤다):
- `LegacyP1EndpointMountTest`(3): 실조립 컨텍스트에 등록된 legacy 매핑 집합 = 구 11개. **빠짐·유령 양쪽**을 잡는다. `private` 핸들러(`post-reply-ai-recommend`)가 매핑되는지도 별도 단언 — 이 경로가 빠지면 구 400 → 신 404 가 되고 리플레이 판정 규칙은 그것을 **통과시킨다**(P0 라운드1 에서 excel 엔드포인트가 실제로 그렇게 샜다).
- `LegacyP1ResponseShapeTest`(5): 응답 바디를 **앱이 실제로 쓰는 `HttpMessageConverter`** 로 직렬화해 구 실측 문자열과 비교. 최초 실행에서 `{"isFirst":true,...,"first":true,"last":true}` 처럼 **키가 중복 생성**되는 것을 잡아냈다(lombok 게터 `isFirst()` → Jackson 이 "is" 를 떼어 별도 프로퍼티를 만든다). 게터에 `@JsonProperty` 를 직접 달아 해소. 상태코드는 200 그대로라 리플레이로는 영원히 안 잡히는 종류다.
  또 `LegacyExceptionAdvice` 의 `basePackages` 가 지워지거나 넓어지는 것을 단언한다 — 그러면 나머지 9모듈의 에러 봉투가 통째로 구 형식으로 바뀌는데 기존 테스트 중 그걸 잡는 것이 없다.
- `LegacyExceptionAdviceTest`(2): `CustomException` → 400 구 봉투, **필수 파라미터 누락 → 400 이 아니라 500 `SERVER-001`**. 이 테스트가 `ErrorResponse` 키 순서 역전(`errorMessage` 가 `errors` 앞)도 잡아 `@JsonPropertyOrder` 로 고정했다. Spring 7 이 만든 문구가 구(Spring 6)와 **글자까지 동일**함도 실측으로 확인.
- `PublicHospitalServiceTest`(4): 평균 별점이 `Math.round`(half-up)가 아니라 `Math.rint`(ties-to-even, 구 `kotlin.math.round` 의 JVM 구현)임을 4.25/4.35 경계로 고정 + 키워드 안정정렬·스니펫 160자 규칙.

## 리플레이 준비 점검 (실행 안 함 — 점검만)
**① 골든셋 내 P1 경로 건수** — `/mnt/ex_disk1/prod-capture/*.golden.jsonl` 35파일 전수, `path` 의 쿼리스트링을 떼고 집계
(첫 집계에서 쿼리스트링을 안 떼 전 경로 0건으로 나왔던 것을 재측정해 정정했다).

| 경로 | 건수 | Authorization 有/無 | 골든 상태코드 |
|---|---|---|---|
| `GET /api/v1/post-word/statistics` | **9** | 9 / 0 | 200 ×9 |
| `POST /api/v1/post-reply-ai-recommend` | **24** | 24 / 0 | 200 ×24 |
| `GET /api/v1/store-channel/search/naver` | **2** | 0 / 2 | 200 ×2 |
| 나머지 8경로 | **0** | — | — |
| 합계 | **35** | 33 / 2 | 전건 200 |

골든셋 전체의 `/api/v1` 엔트리는 369건(OPTIONS 179 · GET 119 · POST 71)이고 그중 P1 몫이 35건이다.

**①-2 실제 리플레이가 먹는 코퍼스(`/mnt/ex_disk1/replay-20260807/`)에서의 분포** — AC3 이 말하는 "read 샤드"는 원본 캡처가 아니라
이 준비된 코퍼스다. 여기서는 건수가 다르다(`prep_corpus.py` 의 정리 규칙 때문으로 보이나 규칙 자체는 확인하지 않았다 — **미판정**).

| 코퍼스 | P1 해당분 |
|---|---|
| `read.golden.jsonl` (`/api/v1` 202건) | `GET post-word/statistics` **5** · `GET store-channel/search/naver` **1** · OPTIONS 18 |
| `write.golden.jsonl` = `write-w1.golden.jsonl` (`/api/v1` 39/37건) | `POST post-reply-ai-recommend` **12** (w1 샤드 0~3 에 5·1·4·2 로 분산) |
| `retry-0.golden.jsonl` | `GET store-channel/search/naver` 1 |

→ **`post-reply-ai-recommend` 는 이미 write 코퍼스로 분류돼 있다.** "읽기 단계인데 쓰기가 섞인다"는 우려는 기존 분류가 이미 해결해 뒀고,
read 샤드로 판정할 것은 사실상 `post-word/statistics` 5건과 `search/naver` 1건뿐이다.

**② 판정 가능 범위** — 11경로 중 리플레이로 판정 가능한 것은 **3개뿐**이다. 나머지 8개는 골든 0건이라
AC3 의 "구 200 → 신 비200 0건"이 공집합 위에서 성립해 버린다. R9 의 **계약 스모크**(경로·메서드·인증경계) 대상으로 넘겨야 한다.

**③ 진입 조건(키)이 이 35건에 직접 걸린다** — 33건이 Authorization 을 달고 있고 그 토큰은 구 키(55자) 서명이다.
실행 중 `legalcare-local-booster-web-5` 의 `JWT_SECRET` 은 여전히 **44자**(env 재확인). P0 라운드2 에서 켠
`rejectsInvalidToken()` 때문에 permitAll 인 `post-reply-ai-recommend` 24건까지 401 이 된다.

**④ 리플레이 전용 booster 를 옆에 세우는 방안 — 성립하나 조건 3개**(전부 읽기 점검, 실행 0)
- 네트워크·DB: 실행 중 컨테이너는 `legalcare-local_legalcare` 네트워크에 붙어 있고 `CORE_DB_URL=jdbc:postgresql://deploy-postgres-new-1:5432/medilawyer`. 같은 네트워크에 이름만 다른 컨테이너를 추가로 붙이는 것은 가능하고, **읽기 3경로 중 2개(post-word/statistics · search/naver)는 DB 를 공유해도 안전**하다.
- **`docker compose up` 으로 하면 안 된다.** 실행 중 `booster-web-5` 에 compose 라벨이 **하나도 없다**(`com.docker.compose.project` 빈 값 — 수동 생성분). compose 가 이 컨테이너를 자기 것으로 인식하지 못해 재생성·이름충돌로 이어진다(메모리 `dell-compose-single-service-hazard` 와 같은 사고 형태). `docker run --network legalcare-local_legalcare --name booster-replay ...` 형태여야 한다.
- 이미지: 현재 떠 있는 것은 `legalcare/booster:dev-20260806-delegate-contract` 로 **P1 코드가 없다**. 리플레이용 이미지를 새로 빌드해야 하고, 그때 `JWT_SECRET`=구 키, `ml.review.server`=실 ML 또는 녹화 스텁, `medilawyer.crypt.*`=실키를 주입해야 한다(현재 기본값은 스텁/자리표시자).
- 저장소의 `docker-compose.replay.yml` 은 `internal: true` 로 egress 를 막는 좋은 장치지만 **서비스 이름이 `booster-web` 그대로**라 그대로 쓰면 실행 중 컨테이너를 건드린다. 별도 프로젝트명(`-p`)과 컨테이너명 분리가 선행 조건이다.

**⑤ 리플레이 시 외부 실호출 위험 1건(선결)** — `GET /api/v1/store-channel/search/naver` 2건은 **네이버 공식 API 실호출** 경로다.
`internal: true` 를 쓰면 호출이 막혀 `REQUEST-002` 가 되고 골든의 200 과 불일치로 찍힌다. 스텁 응답을 준비하든, 이 2건을 판정 제외로 명시하든 **P5 전에 결정**해야 한다.

**⑥ `post-reply-ai-recommend` 는 사실상 쓰기** — 성공 시 `post_reply_ai_recommend` 에 insert/update 하고 ML 서버를 호출한다.
R3 가 "읽기 계열"로 분류했지만 리플레이 코퍼스는 이미 이것을 **write 로 분류**해 뒀다(위 ①-2). 그대로 write 절차(캡처 시점 복원)를 타면 된다.
다만 ML 응답이 없으면(`ml.review.server` 기본값=스텁) 구의 200 과 달라지므로 실 ML 또는 녹화 스텁이 필요하다.

## AC 자가 점검
- **AC3(리플레이 판정)** ❌ **미실행** — 진입 조건(JWT 서명키) 미해소. 지시대로 코드 이식과 테스트까지만 했고 준비 상태는 위 4절에 실측으로 남겼다. 판정 자체는 아침 결정 후 별도 라운드.
- **AC3 선행(이식 완전성)** ✅ 구 11경로 = 신 11경로, 양방향 차집합 0(명령·출력 위). 실조립 컨텍스트 매핑도 동일(테스트 2개).
- **AC1 무회귀** ✅ booster **703/0**(라운드1 대응 후 **723/0**), gateway 76/0, `duplicateClassCheck` 통과, `Routes.java:34` 불변. (초안의 "702"는 재실측에서 703 이었다 — `./gradlew test` 후 `build/test-results/test/*.xml` 의 `tests` 속성 합계.)
- **AC2 무회귀** ✅ P0 인증경계 테스트(`LegacyPathAuthPolicyTest` 132케이스 · `LegacyTokenSourceTest` 16케이스) 전부 통과 — 11경로의 인증경계가 P0 정책표와 11/11 일치.
- **야간 불변조건** ✅ 컨테이너 정지·삭제·compose 실행 0 · 운영 RDS 접근 0 · 구 서비스 정지 0 · **외부 실호출 0**(구 개발계 41010 에 대한 읽기 프로브 6건은 내부 호출이며 그중 4건은 존재하지 않는 ID 를 써 컨트롤러 직전에 끊긴다) · 커밋 0 · 게이트웨이 무수정 · `medilawyer-boot` 무수정 · 실행 중 booster-web env 무변경.

## 알려진 한계 / 리뷰어에게

**1. "1:1"의 경계를 어디에 그었는지** — 클래스·메서드·필드명·URL·로직·응답 키는 전부 보존했다. 대신 두 가지를 의도적으로 줄였다.
① **P1 경로에서 도달 불가능한 메서드는 안 옮겼다.** (라운드1 B1 지적 반영 — 아래 수치는 괄호균형 파서로 재추출한 실측이고, 이전 표기는 "이식한 개수"를 "미이식 개수"로 잘못 옮긴 것이었다.) `StoreChannelService.kt` 19개 중 **1개 이식 / 18개 미이식**, `TeamService.kt` 13개 중 **1개 / 12개**, `MlService.kt` 6개 중 **2개 / 4개**. 옮기면 `StoreChannelHelper`(502줄)·`CrawlingFailLogRecorder`·엑셀 생성기까지 딸려와 P1 범위를 벗어난다. 각 클래스 javadoc 에 "무엇을 왜 뺐는지"를 적어 뒀다.
② **P1 경로가 순회하지 않는 엔티티 연관은 뺐다 — 총 13개.** (라운드1 B1 지적 반영: `Post.postReplyRegistrationRequestList` 가 빠져 있었고 `StoreChannel` 쪽 이름이 축약돼 있었다. 구 `db/domain` 전 엔티티의 연관 애노테이션을 신 이식본과 기계 대조해 재작성했다.) `Post.postReplyRegistrationRequestList`·`Post.message` / `Store.appUserStoreList` / `StoreChannel.postList`·`StoreChannel.storeChannelAttributeList` / `AppUser.appUserStoreList`·`organizationAppUserList`·`teamAppUserList` / `Team.teamAppUserList`·`teamChannelAccountList` / `ProductPost.threads` / `Organization.organizationAppUserList`·`teamList`. 그대로 옮기면 Message·MessageTemplate·Threads·Keyword·Customer·AppUserStore·TeamAppUser·OrganizationAppUser 등 **15개 엔티티가 추가로** 딸려온다. 응답에 영향 0이고 필요한 단계에서 붙이면 된다. **이 둘이 "1:1 위반"으로 보인다면 지적해 달라 — 되돌리는 것도 가능하다.**

**2. 적응 이식 3건**(문자 그대로 복사가 불가능했던 것)
- `WebClient`/`RestTemplate` → **`RestClient`**(저장소 규약). URL·헤더·쿼리·바디·응답 DTO·예외 처리 등가.
- `@AuthenticationPrincipal CustomUserDetails` → `LegacyUserDetailsArgumentResolver`. **booster 에는 Spring Security 자체가 없다**(build.gradle 전수 grep 0건). P0 필터가 심는 `email`/`organizationRole` 애트리뷰트로 구와 같은 객체를 만든다. 차이 1: 구는 Authorization 헤더만 있으면 **모든 요청**에서 app_user 를 조회하는데 여기서는 그 파라미터를 선언한 경로에서만 조회한다(P-1 매핑표의 `app_user(조건부)` 가 그 자리다). 응답이 갈리는 경우는 "토큰은 유효한데 그 이메일의 app_user 가 없는" 극단뿐.
- 구 `QuerydslConfig` 의 `@Bean JPAQueryFactory` 는 이식하지 않고 각 `*RepositoryImpl` 이 `new JPAQueryFactory(em)` 를 만든다. booster 의 동명 설정은 `@UseDatasource` 로 컴포넌트 스캔에서 제외돼 **컨텍스트에 그 빈이 0개**다(이걸 모르고 주입하려다 컨텍스트 로딩이 깨졌고, 그래서 이 사실은 추측이 아니라 실측이다). `modules/post` 의 기존 사용처와 같은 방식.

**3. 구의 버그로 보이는 것 2개를 그대로 뒀다** — 고치면 응답이 구와 달라진다.
- `PostRepositoryImpl.getPostList` 의 `teamId?.let{...} ?: let{...}` 는 Kotlin `let` 이 마지막 식만 반환해서 **`productPostStatus` 필터가 어떤 경우에도 where 절에 안 들어간다**. Java 이식본이 그 분기 구조를 그대로 재현하고 근거를 javadoc 에 남겼다.
- `StoreChannelService.getStoreChannel` / `TeamChannelAccountService.getTeamChannelAccount` 는 `usable` 인자를 무시하고 항상 `true` 로 조회한다.

**4. 미판정 3건 (정직하게)**
- **구/신 응답 동등을 실제로 대조한 것은 3경로뿐이다**(`/api/v1/posts` 빈결과·데이터1건, `/api/v1/public/hospitals` 빈결과, 3개 에러 케이스). 나머지는 테스트로 계약만 고정했고 값 동등은 리플레이의 몫이다.
- `KakaoBusinessSearchResultMeta` 는 구에서도 카카오 응답 키(`is_end`)와 필드명(`isEnd`)이 어긋나 바인딩되지 않는다. 그 성질을 그대로 뒀다(고치면 구와 달라진다). 카카오 검색 경로는 골든 0건이라 확인할 방법도 없다.
- 타임존은 위험이 아님을 확인했다: 구·신 컨테이너 모두 `TZ=Asia/Seoul`(실측), 관련 컬럼 4종 전부 `timestamp with time zone`이라 booster 의 `hibernate.jdbc.time_zone=UTC` 가 절대시각을 바꾸지 않는다.

**5. 개인정보 로깅 — 라운드1 판정에 따라 전수 마스킹함(→ `round-1-P1-response.md` §5).**
초안은 `KafkaProducer.produceMessage` 1건만 올리고 판단을 구했는데, 리뷰어 지적대로 "하나만 올리면 나머지는 검토됐다"로 읽힌다.
로그 라인은 1:1 제약(클래스·메서드·필드명·로직·URL)에도 AC 판정 대상(바디·상태·에러코드)에도 없으므로 마스킹의 AC3 영향은 0 이다.
전수 결과 순수 로그 4곳을 가렸고, 응답 바디로 나가는 3곳은 AC3 때문에 그대로 뒀다(목록은 응답 문서 §5).

**6. 설정 3종이 자리표시자다** — `medilawyer.crypt.decrypt.{key,iv}`(비밀값이라 저장소에 안 넣었다), 네이버/카카오/구글 검색 키(빈 문자열), `ml.review.server`(기본값 스텁). 이 상태로도 **부팅은 되고** 나머지 9모듈에 영향이 없지만, `POST /api/v1/post-reply`(복호화 실패 → 500)와 검색 5경로·AI추천 1경로는 실키/실 서버 없이는 구와 같은 응답을 낼 수 없다. P5 전에 env 주입 계획이 필요하다.

**7. 이번 단계가 건드린 공용 지점 2개** — `TokenVerificationFilter` 는 **추가 수정 없음**(P0 상태 그대로). 새로 생긴 공용 영향은 ①`application.yml` 의 `config.import` 1줄 ②`LegacyWebMvcConfig` 가 등록하는 인자 리졸버 하나뿐이고, 후자는 `supportsParameter` 가 `CustomUserDetails` 타입에만 참이라 다른 모듈 컨트롤러를 건드리지 않는다. `LegacyExceptionAdvice` 도 `basePackages = "legalcare.medilawyer.legacy"` 로 가뒀다.

STATUS: DONE
