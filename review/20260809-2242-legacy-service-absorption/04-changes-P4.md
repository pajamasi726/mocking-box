# 변경 요약: P4 부스트·관리자·비HTTP 이식 (R6 — 61경로 + `@KafkaListener` 1 + `@Scheduled` 3)

브랜치 `feature/prod-sync-candidates` @ `/Users/steve/steve/legalcare-renew-prodsync-wt` · **커밋 0**

## 결과 요약

| | 결과 |
|---|---|
| 130경로 인벤토리 — 빠짐0·유령0 | ✅ `P4-inventory-130.tsv` |
| booster 전체 테스트 | ✅ **864 / 실패 0 / 에러 0 / 스킵 8** (클래스 225. 기준선 850·스킵8 → **+14**) |
| 게이트웨이 | ✅ 84 / 0 / 0 · `Routes.java:34` = `ServiceId.LEGACY` 불변 · diff 0줄 |
| `@LegacyNonNull` 초과0·누락0 | ✅ 기계 전수(구 non-null 보유 531클래스) |
| DTO 필드명·선언순서·`@JsonPropertyOrder` | ✅ 246쌍 불일치 0 |
| `/internal` 예외구멍 ↔ 구 SecurityConfig | ✅ 양방향 차집합 0 |
| 접촉 테이블 ↔ `P-1-endpoint-table-map.tsv` | ✅ 35/35 + 신규 PG 실측 23/23 존재 |
| 외부 실호출·실발송 | ✅ 0건 (앱 미기동 · 구 41010 은 health 읽기 프로브 1회) |
| 커밋·푸시 | ✅ 0 |

**AC6 은 여전히 판정 불가**다 — 리플레이가 P5 소관이라 실행하지 않았다. 코드·계약 수준의 준비와
그것을 지키는 게이트 테스트까지가 이 단계의 산출물이다(아래 "AC 자가 점검").

## 변경 파일

| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `L/apis/reviewBoost/**` (8) | `BoostController` 16경로 + `BoostFacade` + 하위 6서비스 | R6 |
| `L/admin/**` (5) | `AdminController` 9 + `AdminSeoController` 2 + 서비스 3 | R6 |
| `L/airtable/**` (2) · `L/client/ml/controller` (1) | Airtable 3(반환 `void`) · `GET /api/v1/ml/capture-image` | R6 |
| `L/client/kafka/{controller,config}` (4) | Kafka 테스트 3경로 + `KafkaTopicConfig` + `LegacyKafkaConsumerConfig` | R6 |
| `L/eventScheduler/controller` (1) | `EventSchedulerController` 8경로 | R6 |
| `L/schedule/**` (5) | `ScheduledController` 5 + `ScheduledService`(`@Scheduled` 제거) + **`LegacyScheduledJobs`(신설)** + Helper/BatchConfig | R6·R10 |
| `L/kafka/KafkaConsumer.java` (1) | `@KafkaListener` 1건 — `autoStartup` 기본 false | R6·R10 |
| `L/LegacyRuntimeToggles.java` (1) | 스위치 프로퍼티 키·리스너 id 단일 출처 | R10 |
| `L/apis/internal/**` (22) | `/internal` 4컨트롤러 14경로 + 모델·서비스 + **`LegacyInternalPaths`(예외 목록)** | R6 |
| `L/apis/http/**` (13) | 컨슈머·Admin 이 부르는 보조 서비스(community·keyword·thread 계열) | R6 |
| `L/db/domain/**` (46) · `L/db/dto/**` (66) · `L/enums` (11) · `L/db/vo` · `L/code` | 엔티티 18 + 리포지토리 58 + DTO 79 + enum 11 | R6 |
| `L/client/{external,bizmessage}` (2) | `makeBoosterReview`(Gemini) · `sendBoostAlimTalkSvc` 추가 | R6 |
| `L/security/LegacyWebMvcConfig.java` | `SchedulerServiceId` 쿼리파라미터 컨버터(아래 "리뷰어에게" 6) | R6 |
| `L/client/rabbit/config/LegacyRabbitProducerConfig.java` (신설) | legacy 전용 `RabbitTemplate` — 기본 빈이 없어 컨텍스트가 죽던 것 해소 | R6 |
| `L/client/kafka/config/LegacyKafkaProducerConfig.java` | `TYPE_MAPPINGS` 에 `db.dto.kafka` 최상위 27종 전수 등록(구 FQCN) | R6 |
| `app/.../client/legacy/CrawlingLegacyLocalAdapter.java` (신설) | **AC6** — crawling→legacy HTTP 를 인프로세스로 | R6 |
| `app/.../product/client/legacy/ProductLegacyLocalAdapter.java` (신설) | **AC6** — product→legacy HTTP 를 인프로세스로 | R6 |
| `modules/{crawling,product}/.../LegacyService.java` | `@FeignClient(primary = false)` **1줄씩** (어댑터가 이기게) | R6 |
| `app/.../security/InternalAccessBlockFilter.java` | `/internal` 14경로 **열거형 예외**(아래 ②) | R6 |
| `app/src/main/resources/config/legacy.yml` | 스위치 2키(기본 false) + `naverCloud.reviewBoostFramerUrl` | R6·R10 |
| `modules/legacy/build.gradle` | `spring-boot-starter-amqp` · `google-genai:1.0.0` · `project(':modules:notification')` | R6 |
| 테스트 4 | 엔드포인트 마운트 게이트 129 · `/internal` 예외 게이트 · AC6 인프로세스 게이트 · 런타임 스위치 게이트 | R6·R10 |

`L` = `apps/booster-app/modules/legacy/src/main/java/legalcare/medilawyer/legacy`.
**R# 미매핑 변경 0.** 게이트웨이·`medilawyer-boot`·SQL·다른 8개 모듈 무수정.
```
$ git status --porcelain -uall | (영역별 집계)
  281  modules/legacy      (main java 신규 225 · 수정 54 · 12,669줄 + build.gradle + 테스트)
    7  app                 (LocalAdapter 2 · InternalAccessBlockFilter · config/legacy.yml · 테스트 3)
    1  modules/crawling    (@FeignClient primary=false 1줄)
    1  modules/product     (@FeignClient primary=false 1줄)
$ git -C /Users/steve/steve/legal-care/medilawyer-boot status --short     0줄
$ git diff -- apps/gateway-app                                            0줄
```

## 인벤토리 — **130/130, 빠짐 0 · 유령 0**

```
$ python3 P4-scripts/p4_extract_old.py && python3 P4-scripts/p4_extract_new.py && python3 P4-scripts/p4_inventory.py
구 엔드포인트 총 130  (컨트롤러 129 + HealthCheck 1)
이식완료 130 / 미이식 0
유령(신 modules/legacy 에만 있는 매핑) 0
→ P4-inventory-130.tsv 130 행
```
`P4-inventory-130.tsv` = 130행 × (단계·HTTP·경로·구 파일:라인·신 파일:라인·판정).
구 추출은 **괄호균형 파서**다(다중행 애노테이션 포함). 130 의 내역:
P0 1(`/api/v1/health`) + P1 11 + P2 51 + P3 5 + **P4 61** + 선행 태스크 1(`/api/v2/boost1/...`, `modules:notification` 소속) = 130.
`LegacyP1EndpointMountTest` 가 실제 조립된 앱의 매핑 129개(=130 − v2)와 이 목록을 **양방향**으로 비교한다.

## ① 스케줄러·컨슈머는 기본 꺼짐 (R10)

**스위치 2개, 둘 다 기본 false** (`legacy.scheduling.enabled` · `legacy.kafka.consumer.enabled`,
키 문자열의 단일 출처는 `LegacyRuntimeToggles`).

- **`@Scheduled` 3건**은 `ScheduledService` 에서 떼어 **`LegacyScheduledJobs`** 로 옮기고 그 빈에
  `@ConditionalOnProperty(havingValue="true")`(matchIfMissing 미지정=false)를 걸었다.
  미설정이면 **빈 자체가 안 생긴다.** `ScheduledService` 는 `ScheduledController` 가 쓰므로 항상 등록되는데,
  거기에 `@Scheduled` 가 다시 붙으면 게이트가 무력화되므로 그 부재를 테스트가 못박는다.
  앱 전역 `booster.scheduling.enabled`(기존, 기본 false)와 **AND** 조건이다 —
  다른 모듈 스케줄러를 켜려고 전역 스위치를 올리는 순간 legacy 3건이 같이 켜지는 것을 막는 것이 이 2번째 게이트의 존재 이유다.
- **`@KafkaListener` 1건**은 `@ConditionalOnProperty` 를 쓸 수 없다 — 구 `KafkaController` 가 `KafkaConsumer` 를
  생성자로 주입받아(쓰진 않는다) 빈이 없으면 기동이 깨진다. 그래서 **빈은 만들고 컨테이너를 기동하지 않는다**:
  `autoStartup = "${legacy.kafka.consumer.enabled:false}"`. 구와 다른 것은 `id = "legacyEventMessageHandler"` 하나이고
  (검증에서 컨테이너를 지목하려면 안정적인 id 가 필요하다) 와이어에는 영향이 없다.

**켜는 절차 — 구를 먼저 내려야 한다.**
1. 스케줄러: 구 `MedilawyerApplication` 은 `@EnableScheduling` 이 무조건 붙어 있어 **프로퍼티로 못 끈다.**
   구를 내리지 않고 켜면 같은 크론이 양쪽에서 돌아 **알림톡이 두 번 나간다.**
   순서: 구 중지 → `booster.scheduling.enabled=true` **AND** `legacy.scheduling.enabled=true`.
   배치(`batchAppUserStoreSvc`)만은 3번째 게이트가 더 있다 — `POST /api/v1/schedule/status?newState=true`.
2. 컨슈머: `groupId` 가 **구와 완전히 같은 `medilawyer-spring-boot`** 다. 구가 살아 있는 채로 켜면
   이중 실행이 아니라 **컨슈머 그룹 리밸런스**가 나서 파티션이 구·신으로 쪼개진다 —
   구가 받던 메시지 일부를 신이 가로채고 **어느 쪽에도 전량이 없다**(중복보다 나쁘다).
   반드시 구 중지 → 신 기동.

**이중 실행 0건 실증 방법 (P5 에서 쓸 것)**

| 대상 | 방법 | 오늘 실측한 기준선 |
|---|---|---|
| 카프카 컨슈머 | `docker exec kafka1 kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group medilawyer-spring-boot --members` → **멤버 수와 HOST 가 변하지 않아야 한다** | **3멤버 전부 `/192.168.1.55`(구), 파티션 17.** 신이 붙으면 4번째 멤버가 즉시 보인다 |
| 〃 (앱 안에서) | `KafkaListenerEndpointRegistry.getListenerContainer("legacyEventMessageHandler").isRunning() == false` | — |
| `@Scheduled` 3건 | `:app` 의 `SurfaceDumpEndpoint.schedules()` 덤프에 `LegacyScheduledJobs` 항목 **0건**(빈이 없으므로) | — |
| 〃 (부작용 쪽) | 크론 발화 시각(매시 30분·10:45·매 30분)을 지난 뒤 **신규 PG** `store_channel.crawling_start_at` 변화 0 · 신 앱 로그에 job 진입 로그 0줄 | — |
| 알림톡 실발송 | `BizMessageService.sendAlimTalkSvc` 첫 줄 `serviceMode != prod → return` + `naverCloud.sesBaseUrl` 기본값 스텁(`http://localhost:19999`) | 개발계 `serviceMode=dev` |

⚠ **주의 1**: `SurfaceDumpEndpoint.kafkaListeners()` 는 id/groupId/topics 만 찍고 `isRunning` 을 안 찍는다 —
**꺼져 있어도 목록에는 나온다.** 목록 존재를 "켜졌다"로 오독하지 말 것.

⚠ **주의 2 — 접두사 없는 토픽 1개.** 컨슈머가 부르는 `PostFacade.addPostSvc` 는 마지막에
`produceMessageWithoutProfile(MEDILAWYER_NEW_POST_ANALYSIS_REQUEST, ...)` 로 발행하는데, 이 메서드만
**`serviceMode` 접두사를 안 붙인다** — 즉 `medilawyer.new-post.analysis-request` 는 dev·prod 가 **같은 토픽**이다
(구 `PostFacade.kt:104-106` 그대로다. 신이 만든 성질이 아니다).
컨슈머가 꺼져 있으면 이 경로에 도달할 수 없지만, **켤 때는 이 한 토픽이 환경 경계를 넘는다**는 것을 알고 켜야 한다.

## ② `/internal` 14경로 — **구 동작을 보존하기로 했다**(와일드카드 아닌 열거형 예외)

**결정**: `InternalAccessBlockFilter` 에 **(HTTP메서드, 경로패턴) 14쌍**만 통과시키는 열거형 예외를 넣었다.
목록은 컨트롤러 옆(`LegacyInternalPaths.ENTRIES`)에 산다.

**근거 (전부 실측)**
1. 구 `security/SecurityConfig.kt:64-79` 가 이 14경로를 **메서드별로 열거해 `permitAll`** 하고 있었다.
2. 게이트웨이 라우트표(`gateway-app/.../route/Routes.java`)에 **`/internal` 프리픽스 라우트가 없다** →
   구에서도 신에서도 **인터넷에서 이 경로에 도달할 수 없다.** 예외를 열어도 외부 노출은 늘지 않는다.
3. 호출자 중 **`apps/reviewed-app`(다른 앱·다른 JVM)** 이 9경로를 HTTP 로 부른다
   (`.../adminweb/client/legacy/service/LegacyServiceRestClientImpl`, base URL `TRANSITION_LEGACY_URL`,
   현재 값 `http://192.168.1.55:41010` = 구). **인프로세스 전환이 원리적으로 불가능한 호출자**다 —
   booster 자신의 `/internal` 37개(호출자가 전부 같은 프로세스로 흡수돼 `*LocalAdapter` 가 된 것)와 성격이 다르다.
4. 막아 두면 P5 계약 스모크에서 **"구 200 → 신 403" 14건**이 그대로 회귀로 뜬다(판정 규칙상 불합격).
5. P0 라운드1 리뷰어도 같은 결론이었다 — "P4 는 컨트롤러만 옮겨선 안 되고 **예외구멍 설계가 따로 필요하다**".

**기계 대조 (빠짐0·유령0)**
```
구 SecurityConfig /internal permitAll 규칙 11개(그중 2개가 /internal/admin/web/** 와일드카드)
  → 구 정확일치 규칙 중 신에 없는 것: []
  → 신 14개 중 구가 열지 않은 것: 없음
```
와일드카드 2개는 **5개 실경로로 좁혔다**(구보다 넓어지지 않는 방향).
`LegacyInternalAllowlistTest` 가 컨트롤러 매핑을 리플렉션으로 전수 열거해 목록과 양방향 비교하고,
목록을 비우면 P4 이전 동작(전부 403)으로 돌아가는 것까지 단언한다.

**남는 위험(문서화 대상)**: `POST /internal/reviewed/auth-bypass` 는 **인증 우회 레코드를 만든다.**
사설망 안에서 무인증 호출이 가능해지는데 **구와 정확히 같은 노출 수준**이다.
공유 시크릿 헤더로 승격하려면 구·신 양쪽 호출자를 동시에 고쳐야 하므로 **P6 이후 결정 항목**으로 올린다.

## ③ boost1(돈) — 발송 봉인 상태와 AC6 인프로세스 전환

**발송·과금 경로 전수** (`BoostFacade` 기준, 전부 구에 가드가 **없다** — 새로 만들지 않았다)

| 지점 | 무엇을 | 실제로 막는 것 |
|---|---|---|
| `BoostFacade:339` | NCP 알림톡(`sendBoostAlimTalkSvc`) | `naverCloud.sesBaseUrl` 기본값 **스텁**(`config/notification.yml:55` → `http://localhost:19999`) |
| `BoostFacade:356` | NCP SENS SMS | 동일 |
| `BoostFacade:497` | Gemini(`makeBoosterReview`) | `google.api.api-key` 기본값 **자리표시자**(`config/external.yml:40`) |

> `sendAlimTalkSvc`(스케줄러용)에는 `serviceMode != prod → return` 가드가 있는데
> **`sendBoostAlimTalkSvc`(부스트용)에는 없다.** 구가 그렇다. 즉 boost1 발송을 막는 것은
> **URL 기본값 하나뿐**이고, `NAVER_CLOUD_SES_BASE_URL` 을 실주소로 주는 순간 진짜 나간다. **P5 진입 조건.**

**원장 쓰기 지점과 멱등** — `boost_message`·`boost_customer_gift`·`boost_team_gift`·`boost_gift` 전부 **멱등키가 없다**.
같은 요청 2회 = 행 2개(+알림톡 2통). 구 그대로 두었다 — 멱등을 넣으면 AC4/AC6 의 "행 변화 일치"가 깨진다. **판단 항목으로 올린다.**
(기프티콘 원장 `boost_present*` 3테이블은 이 슬라이스가 **읽지도 쓰지도 않는다** — 구 전역 grep 확인.)

**AC6 인프로세스 전환**: booster→legacy Feign 은 둘이었고 **둘 다** `@Primary` 로컬 어댑터로 대체했다.

| 구 호출 | 새 구현 |
|---|---|
| `modules:crawling` → `POST /api/v1/boost1/organizations/{o}/teams/{t}/messages` | `CrawlingLegacyLocalAdapter` → `BoostFacade.sendMessagesFcd` |
| `modules:product` → `PUT /internal/product/review-boost/stores/{storeId}/request-setting` | `ProductLegacyLocalAdapter` → **notification 의** `ReviewBoostRequestSettingService.initializeCommonSetting` |

Feign 인터페이스는 **지우지 않고** `primary = false` 만 붙였다(계약이자 P6 롤백 지점).
`LegacyInProcessConversionTest` 가 ①주입 대상이 어댑터인지 ②`primary=false` 가 유지되는지
③`LEGACY-SERVICE` 를 가리키는 Feign 이 2개를 넘지 않는지를 단언한다 —
어댑터가 사라지면 **예외 없이 조용히 HTTP 로 되돌아가고**, 개발계에선 구가 살아 있어 응답까지 정상이라 아무도 눈치채지 못한다.

> **발송 주체가 바뀐다(이중발송은 아니다).** 전환 전에도 booster 크롤링 이벤트당 발송은 1회였고(구가 대신 쐈다),
> 전환 후에도 1회다(신이 쏜다). 달라지는 것은 **`boost_message` 원장이 구 PG 대신 신 PG 에 쌓인다**는 점이다 —
> R10 이 경고한 병행기간 데이터 이원화가 이 경로에서 실현된다. P5 판정 시 양쪽 PG 를 각각 세야 한다.

## 기계 검증 · 테스트 (전부 실행함)

```
booster  $ rm -rf app/build/test-results core/build/test-results modules/*/build/test-results
         $ ./gradlew test duplicateClassCheck --rerun-tasks --continue
           BUILD SUCCESSFUL in 4m 25s   (61 actionable tasks: 61 executed)
           클래스 225  tests=864 failures=0 errors=0 skipped=8       기준선 850 → +14
           app 259 / core 48 / chrono 7 / crawling 68 / external 40 / legacy 113 /
           medicontents 18 / notification 87 / post 67 / product 71 / user 86

핵심 게이트(전부 그린)
  LegacyP1EndpointMountTest        3   조립된 앱 매핑 129 ↔ 구 목록 양방향(빠짐0·유령0)
  LegacyInProcessConversionTest    4   AC6 — LegacyService 주입 대상이 LocalAdapter · Feign primary=false · LEGACY 대상 2개뿐
  LegacyInternalAllowlistTest      2+2 예외목록 ↔ 컨트롤러 매핑 양방향 · 메서드 불일치·우회 시도 차단 · 목록 비우면 전부 403
  LegacyRuntimeTogglesTest         6   @Scheduled 3건 게이트 · ScheduledService 에 @Scheduled 0 · 크론 구 값 · 컨슈머 autoStartup
  LegacyKafkaProducerConfigTest    7   __TypeId__ 구 FQCN · db.dto.kafka 최상위 전수 등록 · max.request.size 500MB
  JsonPropertyOrderCoverageTest    1   직렬화 대상 타입 전수 키순서 핀
  BoosterApplicationContextTest    3   전 모듈 컨텍스트 로드(빈 충돌·누락 0)

gateway  $ ./gradlew test --rerun-tasks   -> 클래스 9 tests=84 failures=0 errors=0   (기준선 불변)
         $ sed -n '34p' Routes.java  -> new Route("legacy", MEDILAWYER, "/api/v1", ServiceId.LEGACY)
         $ git diff -- apps/gateway-app  -> 0줄

$ python3 P4-scripts/p4_extract_old.py && p4_extract_new.py && p4_inventory.py
  구 엔드포인트 총 130 (컨트롤러 129 + HealthCheck 1) / 이식완료 130 / 미이식 0 / 유령 0

$ python3 P4-scripts/p4_nonnull.py
  구 '기본값 없는 non-null' 보유 531클래스 · 신 @LegacyNonNull 83클래스 · 초과 0 / 누락 0

$ python3 P4-scripts/p4_dto_parity.py
  짝지어진 클래스 246쌍 · 불일치 0건        # 필드명·선언순서 + @JsonPropertyOrder

$ (신규 PG 읽기 프로브)  public 144테이블 · P4 접촉 23테이블 전수 존재
  (P-1 표의 "부재" 2건(boost_review_request_setting·boost_message_delivery)은 선행 v2 DDL 로 이미 해소 — P-1 은 그 시점 스냅샷)
$ P-1-endpoint-table-map.tsv 의 P4 접촉 35테이블 → 신 legacy 엔티티 @Table 매핑 35/35

$ (구 41010 읽기 프로브) legacy-service Up · RestartCount=0 · GET /api/v1/health 200
  legacy-bridge / deploy-postgres-new-1 / booster-web 전부 RestartCount=0 · StartedAt 불변
  개발계 컨테이너 정지·삭제·env 변경·compose 재생성 0
$ git -C medilawyer-boot status --short   -> 0줄
```

**테스트를 그린으로 만들며 고친 것 2건**(둘 다 이 단계가 만든 결함이다)
1. **`RabbitTemplate` 빈 모호성** — `KafkaConsumer` 가 타입만으로 주입받게 두었더니 전체 컨텍스트 게이트가
   `NoUniqueBeanDefinitionException: found 2: crawlingRabbitTemplate, chronoRabbitTemplate` 로 죽었다.
   booster 엔 모듈별 템플릿이 둘 있고 어느 쪽도 `@Primary` 가 아니라 Boot 기본 빈이 아예 없다.
   남의 모듈 것을 빌리지 않고 **legacy 전용 `legacyRabbitTemplate`** 을 만들어(형제 설정과 같은 모양) `@Qualifier` 로 못박았다.
   구는 `send()`(컨버터 미경유)라 와이어 바이트는 동일하다.
2. **Kafka `TYPE_MAPPINGS` 누락 20종** — `LegacyKafkaProducerConfigTest` 의 "`db.dto.kafka` 최상위 전수" 게이트가 잡았다.
   등록이 빠지면 `__TypeId__` 헤더에 **신 패키지(`...legacy.db.dto.kafka.*`)가 새어 나가는데 상태코드는 200 그대로**라
   리플레이가 침묵한다. 27종 전부 구 FQCN 으로 등록했고, 구에서 중첩이던 2종
   (`ProducerNewThreadData$ProducerNewThreadCommentData`/`...$ProducerNewThreadReplyData`)은 `$` 까지 살렸다.

**테스트 실행 환경 메모** — `local` 프로파일 게이트 DB 3개를 **내 로컬에** 띄워서 돌렸다
(`booster-pg-gate`(15544)·`booster-mysql-gate`(13307)·`booster-redis-gate`(6379) — `application-local.yml:32` 의 레시피 그대로).
**개발계 박스와 무관하다.** 한 번의 클린 회차에서 `modules:post` 의
`MatchReviewAnalysisDataQueryMapperTest` 2건이 Testcontainers PG `Read timed out` 으로 떨어졌는데,
같은 테스트만 재실행하면 통과했고 최종 전량 회차는 실패 0이다 — 코드가 아니라 환경 경합이다(P3 이 겪은 것과 같은 종류).

## AC 자가 점검

- **AC6 ⏸ 판정 불가(리플레이 미실행 — P5 소관).** 이 단계가 낸 것은 코드·계약과 그것을 지키는 게이트다:
  boost1 16경로 이식 · booster→legacy Feign **2개 전부** 인프로세스 전환 · `LegacyInProcessConversionTest` 4건 그린.
  **"boost1 리플레이 회귀 0"과 "기프트·발송 원장 이중기록 0건"은 아직 아무것도 실측하지 못했다.**
  후자는 원장에 멱등키가 없어(아래 ③) 리플레이 재생 횟수만큼 행이 늘어난다는 점을 P5 판정 규칙에 반영해야 한다.
- **R9(전량 이식) ✅** — 130/130, 빠짐0·유령0(기계 · 괄호균형 파서). 대조표 = `P4-inventory-130.tsv`.
  조립된 앱에서도 129개(=130 − notification 소속 v2)가 그대로 매핑됨을 `LegacyP1EndpointMountTest` 가 확인.
- **R10(병행·롤백) ✅** — 스케줄러·컨슈머 기본 꺼짐(테스트로 고정) + 켜는 절차·선행조건 + 이중 실행 0건 실증법(위 ①).
  롤백: `legacy.scheduling.enabled`/`legacy.kafka.consumer.enabled` 를 지우면 즉시 원복(미설정=꺼짐),
  `/internal` 예외는 `LegacyInternalPaths.ENTRIES` 를 비우면 P4 이전 동작으로 복귀(그 동치를 테스트가 단언).
  AC10 의 "구·신 동시 가동 중 이중 실행 0건 **실측**"은 P5 에서 위 ① 표대로 재야 한다.
- **외부 실호출·실발송 0건 ✅** — 앱을 기동하지 않았고, 구 41010 은 `GET /api/v1/health` 읽기 프로브뿐(구 DB 쓰기 0).
  신규 PG 는 `pg_tables` 조회만. 개발계 컨테이너 조작 0.
- **커밋·푸시 0 ✅** — `git status` 만 남겼다(오케스트레이터가 단계 종료 후 커밋).

## 남은 일 (다음 단계로 넘김)

1. **뮤테이션 검증 미실시.** 이번에 추가한 게이트가 실제로 무는지(예: `LegacyInternalPaths` 항목 1개 제거 →
   `LegacyInternalAllowlistTest` 가 깨지는가, `@Primary` 제거 → `LegacyInProcessConversionTest` 가 깨지는가)를
   아직 안 돌렸다. 지금까지 각 단계가 해 온 검증이라 **다음 라운드의 1순위**로 남긴다.
2. **P5 진입 조건 3가지**(전부 아래 "알려진 한계"에 근거 있음):
   `NAVER_CLOUD_SES_BASE_URL` 이 스텁으로 고정돼 있는지 · `medilawyer.jwt.*` 실키 주입 ·
   OSIV 차이(1번 항목)를 어떻게 잴지 결정.
3. **미이식으로 남긴 것**(부르는 경로가 0이라 의도적으로 뺀 것 — 필요해지면 그때 옮긴다):
   `StoreChannelService.search*ForMigration` 4종(구 주석 "Todo. 삭제 예정") ·
   `getStoreChannel(channelId, storeName, storeDetail, usable)` ·
   `StoreChannelRepositoryCustom.getByChannelIdListAndUsable` · `StoreRepositoryCustom.getByStoreIdListAndUsable` ·
   `AppUserStoreRepositoryCustom.{deleteByAppUserId,getAllByUsable,getAllByUsableAndIntervalHour}` ·
   `StoreChannelHelper.parseAddressSimple`(구 호출부 0건).

**이어받는 사람이 재현할 명령**
```bash
cd /Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app
# local 게이트 DB 3개가 떠 있어야 :app 의 전체 컨텍스트 테스트가 돈다(application-local.yml:32 레시피)
#   docker run -d --name booster-pg-gate    -e POSTGRES_PASSWORD=gate -p 15544:5432 postgres:16
#     -> createdb medilawyer, medicontents
#   docker run -d --name booster-mysql-gate -e MYSQL_ROOT_PASSWORD=gate -e MYSQL_DATABASE=lawkit -p 13307:3306 mysql:8
#   docker run -d --name booster-redis-gate -p 6379:6379 redis:7
rm -rf app/build/test-results core/build/test-results; for d in modules/*/build/test-results; do rm -rf "$d"; done
./gradlew test duplicateClassCheck --rerun-tasks --continue
python3 <TASK_DIR>/P4-scripts/p4_test_summary.py          # 모듈별·총계 + 실패 목록
python3 <TASK_DIR>/P4-scripts/p4_extract_old.py && python3 <TASK_DIR>/P4-scripts/p4_extract_new.py
python3 <TASK_DIR>/P4-scripts/p4_inventory.py             # 130/130 · 빠짐0 · 유령0
python3 <TASK_DIR>/P4-scripts/p4_dto_parity.py && python3 <TASK_DIR>/P4-scripts/p4_nonnull.py
```
⚠ **Gradle 은 한 번에 하나만 돌려라** — 같은 머신에서 워크트리 2개가 동시에 Gradle 을 돌려
`modules:post` 의 Testcontainers PG 연결이 45분간 블록된 전례가 있다(P3 기록).
이번에도 `--rerun-tasks` 한 회차에서 같은 테스트 2건이 `Read timed out` 으로 떨어졌다가 재실행에서 통과했다.

## 알려진 한계 / 리뷰어에게

1. **OSIV 가 구와 다르다 — 이번 단계에서 고치지 않았다(횡단 결함).**
   구 `core-storage.yml:10` = `open-in-view: false`, booster `application.yml:120` = Boot 기본값 **true**
   (그 주석은 user-service 기준으로 "옛 값이 없다"고 판단한 것인데, **legacy-service 에는 옛 값이 있었다**).
   구체적 영향: `AirtableFacade` 의 "이미 있으면 갈아끼운다" 세 갈래는 `@Transactional` 없이 setter 만 호출하고 `save` 를 안 부른다.
   구는 EM 이 닫혀 **UPDATE 가 0건**이었는데, 신은 요청 스코프 EM 이 살아 있어 뒤이은 커밋에 **더티체킹 UPDATE 가 실린다.**
   방향이 "신이 구보다 더 많이 한다"라 상태코드 리플레이로는 안 잡힌다.
   전역 `open-in-view=false` 는 10개 모듈 전체 동작을 바꾸므로 R6 슬라이스에서 결정할 일이 아니다 →
   **P5 에서 `thread`/`thread_comment`/`thread_reply` 행 단위 차등으로 실측**하고, 그 결과로 판단하자.
2. **`sendBoostAlimTalkSvc` 에 `serviceMode` 가드가 없다**(위 ③). 실발송을 막는 것이 URL 기본값 하나뿐이다.
   P5 에서 boost1 을 리플레이하려면 `NAVER_CLOUD_SES_BASE_URL` 이 스텁으로 고정돼 있는지 **먼저** 확인해야 한다.
3. **boost1 전 경로가 구에서 무인증(`permitAll`)이다** — `SecurityConfig.kt:56-58` 의 `GET/POST/DELETE /api/v1/boost1/**`.
   `LegacyPathAuthPolicy`(P0)가 이미 같은 규칙을 갖고 있어 이식으로 인가가 넓어지거나 좁아지지 않는다.
   다만 **기프트 선택·메시지 발송이 무인증**이라는 사실 자체는 보안 백로그로 남긴다(구 계약이라 이번엔 안 건드렸다).
4. **`ScheduledHelper.negativeTimeRange` 에 구 버그가 있다** — `hour==0` 이면 `withHour(-1)` 로 `DateTimeException`.
   크론이 매시 30분이라 **0시 30분마다 이 경로를 탄다.** 고치지 않고 주석으로 남겼다(1:1).
5. **코루틴 → Java 전환에서 트랜잭션 수명이 길어졌다.** 구는 `@Transactional suspend fun` 이라 첫 중단점에서
   트랜잭션이 끝났는데, Java 이식본은 메서드 끝까지 열려 있다. 결과 집합·발송 대상은 같고(필요한 연관을 구가 전부 fetchJoin)
   커넥션 점유 시간만 길어진다. job 으로의 트랜잭션 전파는 구와 같이 "안 됨"을 유지했다(전용 Executor).
6. **`SchedulerServiceId` 쿼리파라미터 바인딩을 새로 고쳤다(구 200 → 신 400 방지).**
   구 상수명이 backtick `` `medilawyer-boot` `` 인데 Java 는 하이픈 식별자가 불가능해 `MEDILAWYER_BOOT` 다.
   `@RequestParam` 은 Jackson 이 아니라 `Enum.valueOf` 를 타므로 `@JsonProperty` 가 안 먹는다 →
   `LegacyWebMvcConfig` 에 **하이픈 이름 하나만** 받아 주는 컨버터를 달았다(대소문자 무시 같은 확장은 안 넣었다 — 구보다 넓어진다).
   **이 경로는 리플레이로 반드시 태워야 한다**(`POST /api/v1/admin/stores/{id}/communities-keywords?serviceId=medilawyer-boot`).
7. **`ReviewBoostAutoMessageSendModel.idempotencyKey` 의 `@Size` 가 구와 다르다.** 구 `max=200`,
   재사용한 notification 이관분 `max=100`. 101~200자 멱등키가 구에선 통과하고 신에선 400 이다.
   notification 을 고치는 것은 v2 계약 변경이라 손대지 않았다 — **결정 항목**.
8. **`legacyKafkaListenerContainerFactory` 를 새로 만들었다.** booster `:core` 의 `kafkaListenerContainerFactory` 는
   구와 다르다(earliest→**latest**, ackMode **MANUAL_IMMEDIATE**). 특히 MANUAL_IMMEDIATE 인데 구 리스너 시그니처에
   `Acknowledgment` 파라미터가 없어 **오프셋이 영영 커밋되지 않는다**(재기동마다 재처리). 그래서 구 설정 그대로인
   전용 팩토리를 만들었다 — 구와 다른 것은 빈 이름 문자열 하나뿐이다.
9. **`BatchConfig.batchEnabled` 에 `volatile` 을 붙였다**(구엔 없음). 쓰는 쪽이 톰캣 요청 스레드, 읽는 쪽이 스케줄러
   스레드라 가시성 보장이 없었고, 없을 때의 사고 방향이 "끄려고 눌렀는데 안 꺼진다"라 이 단계 목적과 정면 충돌한다.
   구가 의도한 동작을 관측 가능하게 만들 뿐 응답·상태코드에 영향 0.
10. **P1/P2 이식분의 `@LegacyNonNull` 누락 2건을 이번에 보강했다** — `GoogleSearchResultItem`(+중첩 `DisplayName`·`GoogleMapsLinks`)과
    `NaverPlaceItem`. 둘 다 살아 있는 외부응답 역직렬화 경로인데 최상위 타입에서 축이 끊겨 있었다
    (라운드1-P2 BLOCKER 와 같은 축). 방향은 "구와 같아지는" 쪽이고, 지시서의 "실패 유형 ①(요청·응답 양쪽, 초과·누락 0 실증)"이
    P4 의 명시 과제라 슬라이스 밖이지만 고쳤다. **P1/P2 경로의 동작이 바뀌는 변경이므로 리뷰에서 확인해 달라.**
11. **`InternalAdminWebController` 의 컨버터 필드명만 구와 다르다**(`reviewBoostRequestSettingConverter` →
    `legacyInternalReviewBoostConverter`). notification 쪽 동명 클래스와 단순명이 겹쳐서다. 생성자 주입 필드명이라 계약이 아니다.
12. **`Organization` 엔티티에 `@OneToMany teamList` 를 되살렸다** — `/internal/reviewed/customers` 의
    `join(organization.teamList, team)` 이 그 연관 없이는 QueryDSL 로 표현되지 않는다.
    구의 `cascade = [CascadeType.PERSIST]` 는 **일부러 뺐다**: 목록에 넣는 코드(구 `addTeam`)가 신에 없어 발화 지점이 0인데,
    붙이면 이 슬라이스가 손대지 않은 저장 경로에 영향을 줄 수 있다. **의도적 축소이므로 확인해 달라.**
13. **`com.google.genai:google-genai:1.0.0` 을 `modules/legacy/build.gradle` 에 추가했다** —
    `makeBoosterReview` 가 이 SDK 를 직접 쓴다(구 `build.gradle.kts:85` 와 같은 좌표·같은 버전).
    `modules/external` 이 이미 같은 좌표를 갖고 있어 **앱에 새로 들어오는 외부 SDK 는 아니다.**
    다만 구가 요청마다 `Client` 를 새로 만들고 `close` 하지 않는 **커넥션 누수**가 그대로 옮겨졌다
    (`Client` 는 `AutoCloseable`). 1:1 이라 고치지 않았다 — **판단 항목.**
14. **`StoreChannelService.getStoreChannelOptional` 에 `@Transactional` 을 붙였다.** 내가 보조 담당에게
    "구에 없다"고 잘못 지시했는데 구 `:298` 에 실제로 있었고, 담당이 구 원본을 확인해 1:1 로 되돌렸다(옳은 판단).
    호출부가 이미 트랜잭션이라 REQUIRED 합류일 뿐 동작 차이는 없다.

STATUS: DONE
