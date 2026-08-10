# 변경 요약: P2 코어 쓰기(R4/AC4) — 51경로 **전량 이식 완료**

브랜치 `feature/prod-sync-candidates` @ `/Users/steve/steve/legalcare-renew-prodsync-wt` · **커밋 0건**(HEAD `b42d9b6` 불변)

P2-A/B/C 가 29경로를 이식했고, **P2-D(이번)가 잔여 22경로를 이식해 인벤토리 미이식 0** 이 됐다.
컨트롤러 단위로 통째 이식 — 반쯤 이식된 경로 0. `StoreController` 12 · `TeamController` 7 ·
`NoticeController` 2 · `ProductController` 1.

## 변경 파일

이번 세션(P2-D) 변경은 **`modules/legacy` 안**과 `app` 테스트 2개 · `config/legacy.yml` 1키 ·
`modules/legacy/build.gradle` 1줄뿐이다. `:core` main·게이트웨이·`medilawyer-boot` 무수정(`git status` 실측).

| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `apis/http/store/controller/StoreController.java` + `store/service/{StoreFacade,StoreService}.java` | 구 12경로 1:1 **신규** | R4 |
| `apis/http/storeChannel/service/{StoreChannelHelper,CrawlingFailLogRecorder}.java` · `StoreChannelService`(+5메서드) | 주소파싱·매칭점수·채널저장 4종·크롤결과 반영 | R4 |
| `apis/http/team/controller/TeamController.java` · `team/service/TeamService`(+6메서드) · `team/service/excel/*` 4개 | 구 7경로 1:1 **신규**(POI 엑셀 포함) | R4 |
| `notice/api/controller/NoticeController.java` · `client/bizmessage/service/BizMessageService.java` | 구 2경로 1:1 **신규**(알림톡 발송 본체) | R4 |
| `apis/http/productPost/{controller/ProductController,facade/ProductPostFacade}.java` · `product/service/ProductService` · `thread/service/ThreadService` | 구 1경로 1:1 **신규**(Kafka 발행 2종) | R4 |
| `apis/http/post/service/PostFacade.java` **신규** · `PostService`(+2메서드) · `apis/http/appUser/service/AppUserService`(+1) | 통계·기간카운트·회원조회 | R4 |
| `eventScheduler/service/EventSchedulerService.java` **신규** | 스토어 등록 시 스케줄러 타깃 추가(카프카 1건) | R4 |
| `db/domain/{message,customer,boostPresent,boostPresentLedger,boostPresentOrder,thread,community,teamAuth/TeamAuthBypass}/**` 14개 | 엔티티 8 + 리포지토리 6 **신규** | R4 |
| `db/domain/{post,store,storeChannel,appUser,productPost,product}/**` 리포지토리·엔티티 확장 | 파생/커스텀 쿼리 8개 + `Post.message`·`ProductPost.threads`·팩토리 3개 | R4 |
| `db/dto/**` 27개 · `enums/**` 7개 **신규** | 요청·응답·카프카 DTO 전량 | R4 |
| `client/kafka/{KafkaProducer(+1),config/LegacyKafkaProducerConfig}` · `client/email/PushEmailService`(+1분기) | 리뷰분석 요청 발행 · `__TypeId__` 매핑 2종 추가 · 분석완료 메일 | R4 |
| `app/.../LegacyP1EndpointMountTest.java`(기대 40→63, 스캔범위 확대) · `LegacyP2dContractTest.java` **신규** | 매핑·바디 계약 게이트 | R4 |
| `modules/legacy/.../{TeamServiceStatisticsTest}.java` **신규** · `LegacyKafkaProducerConfigTest`(+2) | 계산식·카프카 와이어 게이트 | R4 |
| `app/src/main/resources/config/legacy.yml` | `legacy.external.application-name` 1키(구 `spring.application.name`=`legacy-service`) | R4 |
| `modules/legacy/build.gradle` | `poi-ooxml:5.4.1` 1줄(`:core` 가 implementation 이라 컴파일 경로에 없다) | R4 |

`modules/legacy` main java 223 → **308**(+85), test 12 → **13**(+1). R4 에 매핑되지 않는 변경은 없다.

## 테스트 결과

```
$ cd /Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app && ./gradlew test duplicateClassCheck
BUILD SUCCESSFUL
  app 237 / core 48 / legacy 47 / chrono 7 / crawling 68 / external 40 /
  medicontents 18 / notification 87 / post 67 / product 71 / user 86
  tests=776 failures=0 errors=0       (내 실측 기준선 764 → +12)
$ ./gradlew duplicateClassCheck                 -> BUILD SUCCESSFUL
$ cd ../gateway-app && ./gradlew test           -> tests=84 failures=0 errors=0 (기준선 84 불변)
$ sed -n '34p' .../route/Routes.java
      new Route("legacy", MEDILAWYER, "/api/v1", ServiceId.LEGACY),   <- 불변
$ cd /Users/steve/steve/legal-care/medilawyer-boot && git status --short  -> (빈 출력)
$ git -C .../legalcare-renew-prodsync-wt log --oneline -1  -> b42d9b6 (커밋 0)
```

**리플레이는 실행하지 않았다**(지시 · P5 소관). **외부 실호출 0건**(전부 소스 대조 + 로컬 테스트).

**대조 — 빠짐 0 · 유령 0.** 괄호균형 파서로 구 Kotlin 4컨트롤러에서 뽑은 매핑 vs 매핑 게이트 기대목록:
```
구 소스 22  /  기대목록 63(= P0 health 1 + P1 11 + P2-A 2 + P2-B 21 + P2-C 6 + 이번 22)
구 22 중 기대목록에 없음: []      기대목록의 나머지 41 = 이전 슬라이스분(정상)
```
매핑 게이트는 `containsExactlyInAnyOrder` 라 유령 경로도 같이 막는다. 스캔 범위를
`legacy.apis.` → `legacy.` 로 넓혔다 — 구 `NoticeController` 의 패키지가 `apis` 가 아니라 `notice` 다
(안 넓혔으면 그 2경로가 게이트를 그냥 통과했다).

**DTO 필드명·선언순서 기계 대조 32쌍 — 불일치 0.** 엔티티 애노테이션 대조 9쌍도 일치
(차이는 문서화한 미이식 연관 3개와 `@UniqueConstraint` 중첩 문법뿐).
**접촉 테이블**: `P-1-endpoint-table-map.tsv` 의 이번 22경로 요구 테이블 21종 전부 매핑됨
(`crawling_fail_log` 만 엔티티 없이 JDBC 직접 INSERT — 구 그대로).

**뮤테이션 9건 전부 문다**(각 1회 되돌린 뒤 재실행, 전부 원복 후 776/0 재확인):

| 뮤테이션 | 잡은 테스트 |
|---|---|
| 새 DTO 의 `@JsonPropertyOrder` 제거 | `JsonPropertyOrderCoverageTest` |
| 카프카 DTO 의 `@JsonProperty("isInsulting")` 게터 제거 | `LegacyKafkaProducerConfigTest`(바디 바이트) |
| `TYPE_MAPPINGS` 에서 delete DTO 매핑 제거 | `LegacyKafkaProducerConfigTest`(`__TypeId__`) |
| `purchases` 요청의 `@JsonSubTypes` 제거 | `LegacyP2dContractTest`(Id.DEDUCTION 2건) |
| `StoreController` 경로 1개를 유령 경로로 | `LegacyP1EndpointMountTest` |
| `PostResponse` 키 순서 뒤집기 | `LegacyP2dContractTest` |
| `getPresentHistories` 의 `masking` 을 true 로 | `TeamServiceStatisticsTest` **(처음엔 아무도 안 물었다 — 아래 참조)** |
| 키워드 랭킹을 30일 창으로 "고침" | `TeamServiceStatisticsTest` |
| 작성률 정수 나눗셈을 반올림으로 "고침" | `TeamServiceStatisticsTest` |

## AC 자가 점검

- **AC4(write 샤드 리플레이 + DB 행 변화 일치)** ❌ **미판정** — 리플레이 미실행(지시). 이번 22경로 중 쓰기는 6개.
- **AC4 선행(스코프)** ✅ 인벤토리 51행 = 이식완료 51 / **미이식 0**, 공란 0.
- **AC9 선행(이식 경계)** ✅ 51행 전부 구 파일:라인 + 이식상태(P2-A/B/C/D)가 있다.
- **AC1 무회귀** ✅ 776/0 · `duplicateClassCheck` 통과 · gateway 84/0 · `Routes.java:34` 불변.
- **AC2 무회귀** ✅ 새 22경로의 인증 경계는 구 `SecurityConfig` 그대로 — `/api/v1/team/**` 7개와
  `/stores/check`·`/store/*`·`/store/register`·`posts/statistics` 2개가 permitAll, 나머지 11개가
  `authenticated()`. 인벤토리 5열과 같고 **정책표(`LegacyPathAuthPolicy`) 무수정**.
- **야간 불변조건** ✅ 컨테이너 정지·삭제·compose 0 · 실행 중 env 변경 0 · 운영 RDS 0 · 구 서비스 정지 0 ·
  **외부 실호출 0**(구 41010 프로브도 0건) · 커밋·푸시 0 · 게이트웨이 무수정 · `medilawyer-boot` clean.

## 알려진 한계 / 리뷰어에게

**1. 타임아웃 — 지시대로 실측했고, 구와 다르다(고치지 않았다).**
구는 클라이언트 스택마다 값이 다르다(실측):

| 구 클라이언트 | connect | read/response | 이번 슬라이스 사용처 |
|---|---|---|---|
| `RestTemplateConfig.kt:12-13` | 10s | read **60s** | `BizMessageService` 알림톡, `MlService` 대부분 |
| `WebClientConfig.kt:18-21` | 5s | response/read/write **10s** | `ExternalClientService` 검색 4종, `PushEmailService` 메일 |
| `MlService.kt:79` 인라인 WebClient | (기본) | response **30s** | `authenticateChannelAccount(KAKAO_MAP)` |

신은 전부 `RestClientSupport.HTTP_1_1_REQUEST_FACTORY`(`RestClientSupport.java:102-104`)를 타는데
**connect/read 타임아웃을 설정하지 않는다 → JDK `HttpClient` 기본 = 무한**. 즉 느린 상대에서
구는 10~60초에 끊고 신은 안 끊는다. 블랭킷 값 하나로 덮으면 위 세 값 중 둘이 틀리고,
그 팩토리는 P1(검색 5경로)·P2-B(채널계정 5경로)에도 이미 얹혀 있어 이번 슬라이스 밖을 건드리게 된다.
그래서 **값만 확정해 남기고 고치지 않았다** — 호출지점별 표가 필요한 작업이고, P2-B 가 남긴
"카카오맵 30초" 미검증 건도 같은 뿌리다. **다음 라운드 1순위로 등재.**

**2. 뮤테이션이 실제로 구멍을 하나 찾았다.** `getPresentHistories` 의 `masking` 을 `false`→`true` 로
바꿔도 **처음엔 아무 테스트도 깨지지 않았다**. DTO 키 순서 핀은 "자리"만 보고 값의 출처를 안 본다.
`masking=false` 는 **환자 연락처를 마스킹 없이 응답에 싣는다**는 뜻이라 값이 뒤집히면 계약이 바뀐다.
`TeamServiceStatisticsTest` 를 새로 만들어 `masking`·`total`/`filteredTotal` 분리·페이징 보정·
`endDate` 다음날 0시 변환·정수 나눗셈·평균 반올림을 서비스 수준에서 못박았다.

**3. 개인정보 — 응답에는 그대로 두고 로그에서만 가렸다.**
`GET /api/v1/team/presentHistories/{teamId}`(+엑셀)의 `customerPhone` 은
`boost_customer.id`(= 정규화된 전화번호, `ReviewBoostCustomerCreationService` 실측)이고
`masking` 은 항상 `false` 다 — **구가 평문으로 내보낸다**. 바디를 바꾸면 1:1 위반이라 그대로 뒀다.
로그 쪽은 라운드1 B3 판정대로 마스킹했다(전수 grep 결과 미마스킹 0건):
- `BizMessageService`: 구는 회원 실명·수신 전화번호를 그대로 찍는다 → `LegacyLogMasking.name/phone`
- `StoreFacade.authenticateTeamAuthFcd`: 구는 `mlLoginRes` 객체 전체(세션 쿠키·토큰·전화번호)를 찍는다 → 제거

**4. 밑줄 필드 전수 재점검 — 이번 22경로에는 없다.** P2-B 가 잡은 `_kau` 계열은 ML 채널계정 DTO 4개에만
있고 전부 `@JsonProperty` 로 못박혀 있다(전수 grep). 이번 신규 DTO 27개에는 밑줄 시작 필드가 0건이다.
대신 같은 종류의 함정이 **셋** 있었고 전부 처리했다:
`isInsulting`/`isDefamatory`(카프카 바디) · `isAlreay`(`TeamCheckRes`, 구 오탈자 그대로) — lombok 게터만
두면 Jackson 이 "is" 를 떼어 `insulting`/`alreay` 로 내보낸다. 게터 이름을 `@JsonProperty` 로 못박았다.

**5. Kafka — 구 설정과 대조했다.** 이번에 발행이 2종 늘었다
(`ProducerUserPostDeleteRequest`·`ProducerPostPunishRequest`). P1 에서 잡힌 `__TypeId__`/`MAX_REQUEST_SIZE`
문제와 같은 함정이라 **`TYPE_MAPPINGS` 에 구 FQCN 2개를 추가**했다(안 하면 그 둘만
`legalcare.medilawyer.legacy.db.dto.kafka.*` 가 헤더에 실려 나가고 상태코드는 200 그대로다).
`MAX_REQUEST_SIZE`(500MB)·`ADD_TYPE_INFO_HEADERS`(true)·`security.protocol`(PLAINTEXT)은 P1 그대로다.
바디 바이트도 테스트로 고정했다.

**6. 구의 결함을 그대로 옮긴 것 7건**(전부 응답 바디·DB 행·카프카 바디에 나가므로 고치면 1:1 위반).
- `TeamCheckRes.isAlreay` 오탈자 · `TeamController.addPostReplyReq`(키워드 랭킹인데 메서드명이 답글)
- `getReviewWriteRate` 가 분모를 **다시 질의**한다(같은 조건 2회 — 분자/분모 스냅샷이 다를 수 있다)
- `getPostKeywordRankingByTeam` 만 **기간 제한이 없다**(나머지 통계는 30일 창)
- `getPeriodCountList` 는 `period != MONTH` 면 **항상 빈 배열**(`?period=DAY` → `[]`)
- `POST /store/store-channel` 이 "리뷰분석 완료 … 전송을 완료했습니다" 라는 **무관한 메시지**를 돌려준다
- `GET /stores/{storeId}/channels` 가 **`storeId` 를 안 쓴다**(로그인 사용자 전체 목록을 준다)
- `ProductController` 가 바디엔 `"201"`, HTTP 상태엔 200 을 싣는다 · `ThreadService` 가 스레드 미발견에
  `REQUEST_FAIL_ML`("ML서버 요청 실패")을 던진다 · 네 채널 저장 중 **네이버만** 매칭 실패에 400 을 던지고
  나머지 셋은 조용히 건너뛴다

**7. 구의 미도달 코드는 옮기지 않았다(문서화 완료).** `Message.messageTemplate`·`Message.post` 역참조,
`Threads.threadKeywordList/productPostList`, `StoreChannelService.search*` 10개,
`ProductPostFacade` 의 미사용 의존 3개, `MlReviewAnalysisReq.forRequest` 의 강남언니/여신티켓 분기
(`addNewStoreChannel` 이 채널을 지도 4종으로 이미 제한한다 — 도달 불가 증명), `BizMessageService.sendSmsSvc`·
`sendBoostAlimTalkSvc`. 각 파일 주석에 "왜 안 옮겼는지 + 무엇이 딸려오는지"를 남겼다.

**8. 확신 없는 것 3개(정직 신고).**
- **`boost_customer` 를 실제로 조인하지 않는다는 것은 추론이다.** 선물 내역 쿼리는
  `message.customer.identifier` 만 읽고, 그 경로는 `boost_message.customer_id` FK 라 Hibernate 가
  조인 없이 푼다 — **생성 SQL 을 실측하지 않았다**(DB 없는 단위 테스트라). P-1 표도 이 경로에
  `boost_customer` 를 넣지 않았으므로 표와는 일치한다.
- **작업 중 실수 1건**: `apis/http/appUser/service/AppUserService.java`(P2-B 산출물, 미커밋)를 새 파일로
  덮어써 지웠고 구 Kotlin 원본에서 재작성했다. 컴파일 오류로 즉시 드러나 4개 호출부 기준으로 복원했고
  776/0 으로 확인했지만, **P2-B 가 그 파일에 달아 둔 주석은 복원되지 않았다.** git 미추적이라 원문 대조 불가.
- **구 서버 실응답과 값 단위로 대조하지 않았다.** 키 이름·순서·계산식은 구 소스에서 기계적으로 옮기고
  바이트로 고정했지만, 실데이터 1건 프로브는 P5 진입 조건으로 남긴다.

**9. P3 진입 조건(P2-B 것 유지 + 1건 추가).** `medilawyer.jwt.*` 실키 미주입 시 초대 토큰 검증 불가(P2-B).
추가: `naverCloud.sesBaseUrl` 기본값이 `http://localhost:19999` 이고 `medilawyer.serviceMode` 기본값이
`dev` 다 — **알림톡은 `serviceMode == "prod"` 에서만 나간다**(구 가드). 리플레이에서 알림톡 발송을
재현하려면 그 둘을 함께 주입해야 하고, 안 하면 `POST /api/v1/notice` 는 DB 만 읽고 200 을 준다(구와 동일).

STATUS: DONE
