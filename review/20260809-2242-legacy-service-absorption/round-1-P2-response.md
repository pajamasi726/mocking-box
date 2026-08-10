# 라운드1 P2 리뷰 대응 — BLOCKER 1 + 지적 13건

브랜치 `feature/prod-sync-candidates` @ `/Users/steve/steve/legalcare-renew-prodsync-wt` ·
**커밋 0건**(HEAD `b42d9b6` 불변) · 게이트웨이 무수정 · `medilawyer-boot` clean(`git status` 0줄) ·
컨테이너 조작 0 · 운영 RDS 0 · **외부 실호출 0**(구 41010 프로브도 0건).

BLOCKER 는 반박하지 않는다. 확인한 그대로다. 다만 **DTO 를 하나씩 고치지 말라**는 지시가 이 라운드의
중심이라, 먼저 그 축의 복원 방침부터 정하고 그 방침으로 일괄 적용했다. 나머지는 그 뒤에 붙인다.

---

## 1. non-null 축 — 방침을 먼저 정했다 (BLOCKER=지적 11 + 지적 3)

### 무엇이 사라졌었나

구는 DTO 가 전부 Kotlin `data class` 였고 매퍼에 `KotlinModule` 이 붙어 있었다
(`common/jackson/ObjectConfig.kt:16`). 그래서 **기본값 없는 non-null 파라미터**는
*키 누락*과 *명시적 null* **양쪽에서** `MissingKotlinParameterException`(=`MismatchedInputException` 하위)을
던졌다. Java 로 옮기며 `@NoArgsConstructor`+`@Setter` POJO 가 되면서 요청·응답 양쪽에서 그 계약이 통째로
증발했다. 지적 3 과 지적 11 은 같은 뿌리의 앞뒤다.

### 복원 수단 선정 — 판정 기준은 "구의 예외 타입·에러코드·HTTP 상태까지 같아지는가"

| 수단 | 키 누락 | 명시적 null | 예외 타입 | 비용 |
|---|---|---|---|---|
| `@JsonProperty(required=true)` | 잡음 | **못 잡음** | MismatchedInput | creator 필수 → DTO 전량에 `@JsonCreator` 생성자 신설 |
| `@JsonSetter(nulls=Nulls.FAIL)` | **못 잡음** | 잡음 | InvalidNull(하위) | 낮음 |
| **역직렬화 후 검사(채택)** | 잡음 | 잡음 | MismatchedInput | 애노테이션 1줄/필드 |

앞의 둘은 **각각 절반씩만** 덮는다. 둘을 합치면 DTO 마다 두 종류의 애노테이션 + creator 생성자가 필요하고
`@JsonPropertyOrder`·positional 생성자와 얽혀 오히려 계약이 흔들린다. 그래서 무인자 생성자 + setter 로
채워진 뒤 **"여전히 null 인 필드"** 를 보는 방식을 골랐다 — 구 `KotlinValueInstantiator` 와 같은 성질이고
키 누락·명시적 null 을 같은 자리에서 같은 예외로 처리한다.

구현: `common/jackson/LegacyNonNull.java`(표시) + `LegacyNonNullModule.java`(강제) +
`LegacyJacksonConfig.java`(앱 매퍼 등록). 두 진입점에 얹었다 —
**요청 바디**는 앱 공용 `JsonMapper`(Boot 4 가 컨텍스트의 `JacksonModule` 빈을 모아
`MapperBuilder.addModules` 로 넣는다, `JacksonAutoConfiguration$AbstractMapperBuilderCustomizer` 바이트코드 실측),
**외부 응답**은 `LegacyRestClients.java:99` 의 자체 매퍼.

### 그래서 구와 같아지는 것

예외 **계열**이 같아지므로 그 위에 얹힌 매핑이 전부 따라온다.

- 요청 바디: Spring 이 `HttpMessageNotReadableException` 으로 감싸고, 구·신 모두
  `@ExceptionHandler(Exception)` 이 받아 **500 `SERVER-001`**.
- ML 응답: `RestClient` 가 `RestClientException("Error while extracting response for type […]")` 로 감싼다
  (`DefaultRestClient.java:273-274` 실측 — 구 `RestTemplate` 의 `HttpMessageConverterExtractor` 와 같은 모양).
  호출부 catch 가 구와 같은 `CustomException` 을 만든다 — `authenticateChannelAccount` 는 **400 `REQUEST-003`**,
  `kakaoSms`/`naverCaptcha` 는 catch 가 없어 **500 `SERVER-001`**.

**남는 차이는 메시지 문자열 하나다.** 문장 자체는 `jackson-module-kotlin` 것을 그대로 옮겼지만
(`LegacyNonNullModule.MESSAGE_FORMAT`) 그 안에 박히는 FQCN 이 `legalcare.medilawyer.db.dto…` →
`…legacy.db.dto…` 로 다르다. 패키지 차이는 이식 전반에 이미 있는 차이라 남긴다.

### 전수 추출과 적용 결과

구 Kotlin 주생성자를 괄호균형 파서로 훑어 **기본값 없는 non-null 파라미터**를 기계 추출했다
(`db/dto` 대응 111파일 + 중첩 클래스): **112클래스 / 420필드**.
그중 신에서 실제로 **null 이 들어올 수 있는 자리**(= 역직렬화 경로)를 진입점 기준으로 좁혔다 —
`@RequestBody` 바인딩 19종(21곳) + `retrieve().body(X.class)` 대상 10종, 그리고 거기서 도달하는 중첩·원소 타입.

동명 `.kt` 가 둘인 `TeamChannelAccountLoginReq` 는 신 Java 패키지(`db/dto/teamAppUser`) 기준으로 골랐다.
전수 대조에서 동명 충돌은 이 하나뿐이었다.

| 그룹 | 클래스 | 필드 | 대표 |
|---|---|---|---|
| 요청 바디 — 조직/팀 (R4) | 8 | 24 | `OrganizationRegisterationReq`(3) `TeamChannelAccountLoginReq`(5) `InviteOrganizationAppUserReq`(2+3) |
| 요청 바디 — 스토어/채널 (R4) | 5 | 15 | `StoreRegistrationReq`(2) `NewStoreChannelReq`(2) `ReviewCompletionRes`(3+4) |
| 요청 바디 — ML/채널계정 (R4) | 6 | 26 | `MlKakaoSmsLoginReq`(10+5) `AppUserChannelAccountLoginReq`(5) `MlNaverCaptchaReq`(4) |
| 외부 응답 — ML (R4) | 4 | 9 | `MlAppUserChannelAccountLoginRes`(2) `MlKakaoSmsLoginRes`(2) |
| 외부 응답 — NCP 알림톡 (R4) | 2 | 13 | `AlimTalkResDto`(5+8) |
| 외부 응답 — 검색 4종 (**R3**) | 8 | 26 | `NaverSearhResultItem`(9) `KakaoBusinessSearchResultMeta`(3) |
| 기타 요청 바디 (R4) | 2 | 4 | `MatchStoreReq`(2) `PostReplyAddReq`(6) 등 |
| **합계 부착** | **35** | **117** | 기계 대조 **불일치 0** |
| 부착 안 함(직렬화 전용) | 77 | 303 | 응답 DTO — 역직렬화 경로가 없어 이 축의 영향이 없다 |

기계 대조 결과(구 Kotlin 집합 == 신 `@LegacyNonNull` 집합, 35클래스 전부 `OK`, 초과·누락 0)는
`scratchpad/verify.py` 출력에 있고 요약은 위 표의 마지막 두 줄이다.

**primitive 는 박싱했다(11클래스 19필드).** Java primitive 는 "키 누락"과 "0/false 를 실제로 보냄"을
구별할 수 없어 검사가 성립하지 않는다. `LegacyNonNullModule.scan()` 이 primitive 에 표시가 붙으면
기동 시점에 터뜨려 "표시만 하고 검사 못 하는 상태"를 원천 차단한다.
박싱으로 lombok 게터 이름이 바뀌는 자리(`isPermissionWriting()` → `getPermissionWriting()`)는 호출부 3파일을
같이 고쳤고, `KakaoBusinessSearchResultMeta.isEnd` 는 수동 게터 핀을 유지해 JSON 키가 안 바뀌게 했다.

### 이 축을 P1 검색 DTO 까지 넓힌 이유와 그 결과 — **먼저 읽어 달라**

지적 3·11 은 P2 만 짚었지만 같은 방침을 P1(R3) 검색 DTO 8개에도 적용했다. 1:1 이 기준이고
"구가 던지던 것을 신이 안 던지는" 방향은 같은 구멍이기 때문이다. 그런데 적용하면서 **구의 카카오 상호검색이
사실은 항상 400 이었다**는 것이 드러났다.

- 구 `KakaoBusinessSearchResultMeta.kt` 는 `val isEnd: Boolean` 등 **기본값 없는 non-null** 3개인데
  카카오 응답 키는 `is_end`/`pageable_count`/`total_count` 스네이크다 → 키가 안 맞아 `MissingKotlinParameterException`.
- 구 매퍼에 네이밍 전략이 없다. **구 저장소 전수 grep 0건 + cloud config 저장소
  (`/Users/steve/steve/legal-care/cloud_config_repository`) 전수 grep 0건** — `property-naming-strategy`/`SNAKE_CASE` 어디에도 없다.
- 구 `ExternalClientService.kt:229-231`(`catch (e: Exception)` → `:231`) 이 받아 `CustomException(REQUEST_FAIL_SEARCH)` → **400**.

앞 판 주석은 이 자리에 "구에서도 바인딩되지 않는다(**기본값**)"라고 적었는데 그게 틀렸다 — 기본값이 없으니
기본값으로 떨어지지 않고 **예외**다. 주석을 근거와 함께 고쳤다(`KakaoBusinessSearchResultMeta.java:13-27`).
**결과적으로 이 변경은 카카오 검색 경로를 신에서 200 → 400 으로 되돌린다.** AC3("구 200 → 신 비200 0건")은
구가 애초에 400 이라 위반이 아니지만, 리플레이에서 눈에 띌 것이라 여기 먼저 적어 둔다.
네이버·모두닥·구글은 키가 맞아 정상 동작이 유지된다.

### 뮤테이션 — BLOCKER 재현 확인

`MlAppUserChannelAccountLoginRes.result` 의 `@LegacyNonNull` 한 줄만 지우고 재실행:

```
$ ./gradlew :app:test --tests '*LegacyNonNullContractTest*'
legacy non-null 계약 배선 > ML 이 result 를 빼먹으면 응답 파싱에서 터진다 … FAILED
  java.lang.AssertionError: Expecting code to raise a throwable.
```

즉 되돌리면 파싱이 통과하고 `result == null` 로 바인딩된다. 그 상태에서 컨트롤러 판정식이 무엇을 내는지는
같은 파일의 대조군 테스트가 값으로 못박는다 — `null != FAIL` → **"1차 로그인 시도에 성공했습니다."**.
원복 후 재실행 `BUILD SUCCESSFUL`.

---

## 2. 나머지 지적 — 반영/반박

| # | 등급 | 처리 | 근거 |
|---|---|---|---|
| 11 | BLOCKER | **반영** | 위 1절. `MlAppUserChannelAccountLoginRes`·`MlKakaoSmsLoginRes` 등 4클래스 9필드 + 배선 2곳 |
| 3 | BLOCKER | **반영** | 위 1절. 요청 바디 진입점 19종 전수 |
| 2 | BLOCKER | **반영** | `LegacyKafkaProducerConfig.java:134-140` 에 구 FQCN 3개 추가(`ProducerPostExtractCompleted`·`ProducerTargetUpdate`·`MlReviewAnalysisReq`). 개별로 적는 방식이 또 빠질 수 있어 **빠짐 자체를 막는 게이트**를 새로 넣었다 — `db.dto.kafka` 최상위 클래스를 스캔해 전부 매핑돼 있는지 확인(`LegacyKafkaProducerConfigTest.everyKafkaDtoHasLegacyMapping`). `MlReviewAnalysisReq` 는 패키지가 `db.dto.ml` 이라 스캔에 안 걸린다는 것까지 테스트에 적었다 |
| 1 | BLOCKER | **부분 반영 + 부분 반박** | 아래 3절 |
| 4 | HIGH | **반영** | `NoticeController.java:117` `Objects.requireNonNull(storeChannel)` · `BizMessageService.java:266` `Objects.requireNonNull(appUser.getPhone())`. Kotlin `!!` 는 `Intrinsics.checkNotNull` → **메시지 없는** NPE 라 1-arg `requireNonNull` 과 같다. 이식 범위의 `!!` 를 전수 grep 한 결과 실제 단언은 5건이고 그중 2건이 이것, `NoticeController.kt:73`(`appUser!!.id`)은 Java 에서 `.getId()` 가 자동 NPE 라 동등, 나머지 2건(`PostReplyService.kt:171` `saveReply`, `ReviewBoostV2BizMessageService.kt:52`)은 이 슬라이스 밖이다 |
| 5 | HIGH | **범위 밖으로 명시**(코드 무변경) | 아래 4절 |
| 6 | HIGH | **범위 밖으로 명시**(코드 무변경) | 아래 4절 |
| 7 | MEDIUM | **반영** | `ProducerUserPostDeleteRequest`·`ProducerPostPunishRequest` 에 형제와 같은 마스킹 `toString()` 추가 + 규칙을 테스트로 못박음(`KafkaDtoLoggingMaskTest`). **구는 전부 data class 라 자동 `toString()` 이 email·실명·phone·본문을 평문으로 찍었다** — 마스킹은 구에 없던 것이고 라운드1 B3 판정에 따른 **의도된 차이**다. 그 사실을 테스트 Javadoc 에 명시했다 |
| 8 | MEDIUM | **반영** | 주석만 고치는 게 아니라 **애노테이션이 실제로 빠져 있었다**. `PostService.java:65` `@Transactional`, `:72` `@Transactional(readOnly = true)` 추가(구 `PostService.kt:60,87`) |
| 9 | MEDIUM | **기록만**(지적 취지대로) | `ProductPost.java:47-64` 에 primitive→boxed 로 `persist`→`merge` 가 되는 메커니즘, 지금 손실이 없는 이유, 언제 드러나는지를 적었다. 11개 엔티티를 한꺼번에 바꾸면 게터 반환 타입이 바뀌어 슬라이스 밖을 흔든다 — P4/P5 백로그 |
| 10 | LOW | **5건 반영 / 1건 반박** | `CrawlingFailLogRecorder` `catch (Exception)`→`catch (Throwable)`(구 `runCatching` 범위) · `StoreChannelHelper` `toLowerCase()` 3곳 → `toLowerCase(Locale.ROOT)` · `MlAppUserChannelAccountLoginRes` `TUID`/`TSID`/`UUID` 에 `@JsonProperty` 핀 · `StoreChannelService.java:229` `log.error` 복원(구 `.also` 는 throw 직전 실행이라 순서까지 동일) · `spring.kafka.bootstrap-servers` 의 빈 기본값 제거(구의 fail-fast 복원, 세 프로파일 모두 값이 있어 기동 무영향). **로컬 `JsonMapper` 는 유지** — 앱 공용 매퍼에는 구 `ObjectConfig` 와 다른 전역 설정이 얹혀 있어 주입하면 오히려 구와 멀어진다. 대신 "스칼라만 넣는다"는 전제를 클래스 주석에 못박았다. `CommunityController` 는 구도 핸들러 0개라 무변경 |
| 12 | MEDIUM | **반영** | `LegacyAppUserResolutionInterceptor.java:120-139` — 항목별 try/catch 를 **전체 try** 로 되돌렸다. 구 `associate{}` 는 원자적이라 한 항목만 깨져도 맵 전체가 비고 조직 엔드포인트 7개가 `AUTH-002` 가 된다. 견고성은 항목별이 낫지만 인가가 넓어지는 방향이라 구 의미를 따랐다. `null` 값이 섞인 경우까지 구(`:56` `key.toString()` NPE → `:66` catch)와 같게 맞췄다 |
| 13 | LOW | **반영 + 메커니즘 반박** | 아래 5절 |
| 14 | LOW | **(a) 반영 / (b) 보류** | 아래 6절 |

---

## 3. 타임아웃 — (b)는 했고 (a)는 근거를 대고 안 했다

**(b) legacy 프로파일별 복원 — 반영.** `LegacyRestClients.Profile` 4종을 만들고 단일 인자 오버로드를
없앴다. 새 호출 지점이 생기면 컴파일 단계에서 프로파일을 고르게 된다.

| 프로파일 | connect / read | 구 출처 | 신 호출 지점 |
|---|---|---|---|
| `REST_TEMPLATE` | 10s / 60s | `RestTemplateConfig.kt:12-13` | ML 4 + NCP 알림톡 1 |
| `WEB_CLIENT` | 5s / 10s | `WebClientConfig.kt:18-21` | 검색 5 + 메일 1 |
| `ML_KAKAO_MAP` | 30s / 30s | `MlService.kt:78-79` 인라인 | `check_id` 의 KAKAO_MAP |
| `ML_GOOGLE_MAP` | 30s / **무제한** | `MlService.kt:102-108` 인라인 | `check_id` 의 GOOGLE_MAP |

리뷰어가 지적한 **채널 단위 분기** 문제가 핵심이라 `MlService.channelProfile(ChannelId)` 로 갈랐다 —
메서드 단위로 하면 여섯 채널 중 둘이 틀린다. 네 번째 갈래(구 `GOOGLE_MAP` 도 read 무제한)를 알려준 건
맞는 지적이고, 그래서 **거기만 일부러 무제한으로 뒀다**(값을 넣는 게 오히려 구와 다른 동작이다).
네 값을 각각 못박는 테스트를 넣었다(`LegacyRestClientTimeoutTest`) — "무제한만 아니면 된다"로 두면
다시 하나로 덮는 실수를 못 막는다.

**(a) `:core` 공유 팩토리 10s/60s "원복" — 반박.** 전제가 틀렸다.

- 구 마이크로서비스의 Feign 실값은 저장소가 아니라 config-server 조각에 있었다:
  `cloud_config_repository/feign-release.yml:7-8` → `connectTimeout: 200000000 / readTimeout: 200000000`
  (≈55시간 = 사실상 무제한). 그 조각이 실제로 물렸다는 근거는
  `user-service/src/main/resources/application-release.yaml:7` 의 composite name list 에 `feign` 이 들어 있는 것.
- 같은 모노레포 `lawkit-app` 은 그 200,000s 블록을 **명시 이식했다가**(커밋 `8821175`) 별도 커밋(`0bb3c7e`)에서 지웠다.
- 즉 **10s/60s 는 원복이 아니라 운영이 가진 적 없는 새 정책**이다. Feign 라이브러리 기본값이 10s/60s 인 것은 맞지만
  운영은 그 기본값으로 돈 적이 없다.
- 덧붙여 리뷰어가 근거로 든 커밋 `025c270` 은 **현 워크트리 HEAD 의 조상이 아니다**(`git merge-base --is-ancestor` → false).
  이 워크트리 yml 7개에는 `spring.cloud.openfeign.client.config` 블록이 아직 살아 있다.

안전성 자체는 리뷰어 우려대로 확인했다 — **공유 팩토리를 타는 54개 호출 지점(8모듈) 중 SSE·스트리밍·
long-polling·대용량 다운로드는 0건**이다(`text/event-stream`/`SseEmitter`/`Flux`/`BodyHandlers` 전수 grep 0,
응답 추출이 전량 `.body(X.class)`/`.toBodilessEntity()`). 리뷰어가 "확인하지 않았다"고 남긴 항목이라 대신 봤다.
그러니 값을 넣는 것 자체는 기술적으로 안전하다. **다만 R3/R4 어디에도 매핑되지 않는 8모듈 54지점의 정책 변경**이라
1:1 이관 단계에서 넣을 것이 아니다 → **백로그**(근거는 `LegacyRestClients` 클래스 주석에 남겼다).

**게이트웨이 무한 대기**는 리뷰어 분석이 맞다(`GatewayConfig.java:52` connect 10s 만, `ProxyFilter.java:164-173`
에 `HttpRequest.timeout()` 없음, 주석 `:54-55` 의 "헤더 수신까지의 지연은 커넥트 타임아웃이 잡는다"는 사실이 아니다).
다만 게이트웨이는 SSE 를 실제로 릴레이하므로(`ProxyFilter.java:39,272,281`) 바디 무제한이 **의도된 설계**이고,
`Routes.java:34` 가 아직 `LEGACY` 라 이번 라운드에서 게이트웨이를 건드리는 것은 불변조건 위반이다. 백로그.

---

## 4. 지적 5·6 — R# 없는 변경. 되돌리지 않고 범위 밖으로 명시한다

둘 다 이번 라운드에서 **코드를 건드리지 않았다**(지시: "R# 매핑을 명시하든지, 범위 밖이면 그렇게 적어라").

**게이트웨이 신뢰헤더 9파일** — R1~R11 어디에도 매핑되지 않는다. R2 는 "구 인증 필터의 **경로별 인증 요구**를
booster 필터로 이설"이지 게이트웨이의 토큰 출처를 쿠키→`Authorization` 으로 바꾸는 것이 아니다.
특히 `Routes.java:34` 가 아직 `ServiceId.LEGACY` 이므로 **그 변경은 지금 서비스 중인 구 스택에 그대로 적용된다** —
리뷰어 지적이 맞다. 되돌리면 P0 의 AC2(무토큰 10경로 응답코드 대조)가 함께 깨지므로 임의로 되돌리지 않았다.
**요구사항으로 승격할지 되돌릴지 결정을 요청한다.** 방어 자체는 촘촘하다는 리뷰어 평가에 동의한다
(시크릿 공백 시 null 반환, `MessageDigest.isEqual` 상수시간, `ProxyFilter.java:221-222` 인바운드 `X-Auth-*` 접두 전량 제거).

**SQL 3개** — R11 은 "**접촉분만** DDL 선행 적용"인데 `legacy-absorption-p1b-schema.sql:2` 는 스스로
"운영 대비 부족 42테이블 전량 생성"이라 적고 있고, `p1c` 는 기존 테이블 5개를 변경한다(`post` 9.96M행에
`CREATE INDEX CONCURRENTLY` 2개 포함). **R11 범위 초과가 맞다.** 부팅 시 실행되는 Flyway/Liquibase 가
저장소에 없어 런타임 위험은 0이고, 순수하게 범위 문제다. 신규 PG 에 실제로 적용됐는지는 확인하지 않았다
(psql 세션을 열지 않았다 — 리뷰어도 같다).

---

## 5. 지적 13 — 반영했지만 지적의 **메커니즘 설명은 틀렸다**

지적은 "구 `NotImplementedError` 는 `Error` 라 `@ExceptionHandler(Exception)` 에 안 잡히고 컨테이너
`/error` 로 떨어져 부트 기본 바디(`timestamp/status/error/path`)가 나간다"고 했다. 그렇지 않다.
Spring 4.3 부터 `DispatcherServlet.doDispatch` 가 핸들러에서 나온 `Throwable` 을 감싸 예외 처리 흐름에 태운다:

```java
// spring-webmvc-7.0.8 소스 DispatcherServlet.java:975-979 (실측)
catch (Throwable err) {
    // As of 4.3, we're processing Errors thrown from handler methods as well,
    // making them available for @ExceptionHandler methods and other scenarios.
    dispatchException = new ServletException("Handler dispatch failed: " + err, err);
}
```

그래서 구도 **500 + legacy `ErrorResponse` 봉투**였다. 실제로 갈리는 건 봉투가 아니라
`errors[0].message` **문자열**이다. 그 값을 맞추려면 (1) `Error` 여야 한 겹 감싸이고 (2) `toString()` 이
구 클래스명을 내야 한다 — `common/exception/LegacyNotImplementedError.java` 가 그 둘을 한다.
구는 인자 없는 `TODO()` 라 메시지도 `"An operation is not implemented."` 고정이다.
`LegacyExceptionAdviceTest` 에서 MockMvc 로 실측한 최종 바이트:

```
errors[0].message = "Handler dispatch failed: kotlin.NotImplementedError: An operation is not implemented."
```

클래스명을 `toString()` 으로 흉내 내는 건 보기 좋은 짓이 아니다. Kotlin 런타임을 끌어오는 것보다 대가가
작다고 판단했고 파일 주석에 그 판단을 적었다. 다르게 봐도 좋다.

---

## 6. 지적 14 — (a) 반영, (b) 보류하고 이유를 적는다

**(a) 복호화 실패 래핑 — 반영.** `new IllegalStateException(e)` 는 `getMessage()` 가 `cause.toString()` 이라
`errorMessage` 가 `"javax.crypto.BadPaddingException: …"` 로 갈렸다. `@SneakyThrows` 로 바꿔 원 예외를 그대로
전파한다(Kotlin 전파와 동일) — `TeamChannelAccountFacade.java:264,269` · `AppUserChannelAccountService.java:216,221`.
`legacy.yml:27-28` 의 crypt 키·IV 가 자리표시자라 relogin 이 매번 이 경로를 탄다는 지적이 맞고,
덧붙이면 자리표시자 IV(`dev-placeholder-iv`)는 **18바이트**라 AES 블록 16바이트가 아니어서
`IvParameterSpec` 단계에서 먼저 터진다. P3 진입 조건에 실키 주입이 이미 등재돼 있다.

**(b) ML·메일 non-2xx 메시지 — 보류.** 두 가지 때문이다.

1. `defaultStatusHandler` 는 **append** 이고 적용은 **first-match-wins** 다(`DefaultRestClient.java:941-946` 실측).
   나중에 더해도 `RestClientSupport` 의 핸들러를 이길 수 없다. 이기려면 `RestClientSupport.builder` 사용을
   그만두고 조립을 통째로 복제해야 한다.
2. 복제하더라도 **맞춰야 할 문자열이 구 스택마다 다르고**(`RestTemplate` → `DefaultResponseErrorHandler`,
   `WebClient` → `WebClientResponseException`) 그 포맷은 구가 돌던 Spring 버전에 딸려 있다.
   ML 에서 non-2xx 를 실제로 받아 보지 않고는 맞았는지 확인할 방법이 없는데 이번 라운드는 외부 실호출 금지다.
   **추정으로 맞추면 틀린 채로 굳는다.** → P5(값 단위 대조)로 넘긴다.
   영향 범위: 채널계정 4경로 + `POST …/invitations` 의 `errorMessage` 문자열. 근거는 `LegacyRestClients` 주석에 기록.

---

## 7. `AppUserService` 삭제 사고 후속

리뷰어가 구 Kotlin 과 4메서드 전수 대조로 **동작 동일**을 확인해 줬다(애노테이션·readOnly·에러코드·메시지,
구가 인자 `usable` 대신 리터럴 `true` 를 넘기는 버그, `.also` 의 로그 선행 순서, `AppUserInfoRes` 의
positional 전환 시 필드 순서까지). 그 결과를 반영해, **코드만 봐서는 안 보이는 판단**을 파일 상단에 다시 적었다
(`AppUserService.java:16-49`) — ① `usable` 대신 `true` 를 넘기는 구 버그를 일부러 남긴 이유,
② `.also { log.error }` 가 `throw` 가 아니라 `CustomException(...)` 표현식에 붙어 "로그 → throw" 순서라는 것,
③ named→positional 전환 시 `AppUserInfoRes.kt:6-10` 순서와 대조해야 한다는 것, ④ `@Transactional` 의
`readOnly` 가 메서드마다 다른 것이 구의 비일관이지 실수가 아니라는 것.
P2-B 원본 주석은 복구 불가가 맞다(git 미추적, `build/classes` 의 `.class` 가 재작성 이후 타임스탬프).

---

## 8. 수정 파일

`modules/legacy` main **56파일**(신규 4: `LegacyNonNull`·`LegacyNonNullModule`·`LegacyJacksonConfig`·
`LegacyNotImplementedError`) + test **7파일**(신규 3: `LegacyNonNullModuleTest`·`LegacyRestClientTimeoutTest`·
`KafkaDtoLoggingMaskTest`) + `app` test 신규 1(`LegacyNonNullContractTest`).
**`:core` main·게이트웨이·SQL·`medilawyer-boot` 무수정.** 전부 R4 매핑, 검색 DTO 8개만 R3.

| 묶음 | 파일 | R# |
|---|---|---|
| non-null 축 — 기반 3 + DTO 31 + 호출부 3 | `common/jackson/**` `db/dto/**` `apis/http/{organization,organizationAppUser,teamAppUser}/…Facade.java` | R4(검색 DTO 8은 R3) |
| 카프카 매핑 3 + 마스킹 2 + fail-fast | `client/kafka/config/LegacyKafkaProducerConfig.java` `db/dto/kafka/Producer{UserPostDelete,PostPunish}Request.java` | R4 |
| 타임아웃 프로파일 | `client/LegacyRestClients.java` `client/{ml,external,email,bizmessage}/…` 4 | R4(검색·메일은 R3/R4 공용 팩토리) |
| 널 단언·트랜잭션·인가·예외 | `notice/…/NoticeController.java` `client/bizmessage/…` `apis/http/post/service/PostService.java` `security/LegacyAppUserResolutionInterceptor.java` `apis/http/organization/service/OrganizationFacade.java` `common/exception/LegacyNotImplementedError.java` | R4 |
| 잔가지 5 + 주석 복원 | `apis/http/storeChannel/service/{CrawlingFailLogRecorder,StoreChannelHelper,StoreChannelService}.java` `db/domain/productPost/ProductPost.java` `apis/http/appUser/service/AppUserService.java` | R4 |

## 9. 테스트 재실행

```
$ cd apps/booster-app && rm -rf */build/test-results && ./gradlew test duplicateClassCheck --rerun-tasks
BUILD SUCCESSFUL in 3m 44s
  app 241 / core 48 / chrono 7 / crawling 68 / external 40 / legacy 64 /
  medicontents 18 / notification 87 / post 67 / product 71 / user 86
  tests=797 failures=0 errors=0        (기준선 776 → +21)
$ cd ../gateway-app && rm -rf build/test-results && ./gradlew test --rerun-tasks
  tests=84 failures=0 errors=0         (기준선 84 불변)
$ sed -n '34p' .../route/Routes.java
      new Route("legacy", MEDILAWYER, "/api/v1", ServiceId.LEGACY),   <- 불변
$ git -C .../legalcare-renew-prodsync-wt log --oneline -1   -> b42d9b6 (커밋 0)
$ git -C .../medilawyer-boot status --short | wc -l         -> 0
```

**새 테스트 21건이 보장하는 것**(전부 뮤테이션으로 확인, 8/8 문다 — 되돌리면 깨지고 원복하면 그린):

| 뮤테이션 | 깨지는 테스트 |
|---|---|
| ML 응답 `result` 의 `@LegacyNonNull` 제거 | `LegacyNonNullContractTest` + `LegacyNonNullModuleTest` |
| 요청 바디 `storeName` 의 `@LegacyNonNull` 제거 | `LegacyNonNullContractTest` |
| `MatchStoreReq.storeId` 박싱 되돌리기(`Long`→`long`) | `LegacyNonNullContractTest` |
| `MlReviewAnalysisReq` 매핑 제거 | `LegacyKafkaProducerConfigTest` |
| 조직권한 맵을 항목별 try/catch 로 되돌리기 | `LegacyAppUserResolutionInterceptorTest` |
| `ML_GOOGLE_MAP` 의 무제한 read 를 60s 로 덮기 | `LegacyRestClientTimeoutTest` |
| `LegacyNotImplementedError` → `RuntimeException` | `LegacyExceptionAdviceTest` |
| Delete DTO 의 마스킹 `toString()` 제거 | `KafkaDtoLoggingMaskTest` |

`LegacyNonNullModuleTest.gate_onlyMarkedLegacyTypes` 가 **다른 9개 모듈에 영향이 없다**는 것을 직접 확인한다 —
모듈이 앱 공용 매퍼에 얹히므로 이게 이 변경의 최대 위험이었다. legacy 밖 타입과 표시 없는 legacy 타입은
게이트를 통과하지 못한다(deserializer 를 감싸지조차 않는다).

## 10. 남은 것 (정직 신고)

1. **지적 14(b)** — 위 6절. 외부 실호출 없이는 검증 불가라 P5 로 넘겼다.
2. **지적 1(a)** — 반박했지만 `:core` 54지점이 무제한인 상태 자체는 사실이다. 요구사항으로 승격 필요(백로그).
3. **지적 5·6** — 코드를 안 건드렸다. 승격/되돌리기 결정 요청.
4. **직렬화 전용 DTO 77클래스 303필드** — 구 Kotlin non-null 이었지만 신에서 검사가 없다. 여기서 null 이
   들어가면 구는 생성자에서 NPE(500), 신은 `"x": null` 로 200 이다. 지적 4 가 그 사례 2건이었고 둘 다 고쳤으며,
   구 `!!` 전수 grep 으로 슬라이스 내 잔여 0건을 확인했다. **다만 `!!` 없이도 nullable 값이 흘러드는 경로까지는
   훑지 않았다** — 그건 호출부 데이터플로 분석이 필요하고 이번 라운드에 하지 않았다.
5. **카카오 검색 200→400** — 위 1절 말미. 의도한 1:1 복원이지만 리플레이에서 눈에 띌 변화라 재확인을 요청한다.
6. **리플레이 미실행** — 지시대로 P5 소관. AC4 는 여전히 미판정이다.
7. **p1/p1b/p1c SQL 의 실제 적용 여부** — psql 세션을 열지 않았다.

STATUS: REVISED
