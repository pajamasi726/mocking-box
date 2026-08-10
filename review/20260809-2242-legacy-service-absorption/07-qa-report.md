# QA 리포트: P5 개발계 전수 테스트 — legacy-service → booster-app 흡수

대상 worktree `/Users/steve/steve/legalcare-renew-prodsync-wt` HEAD `4df03196` (코드 무수정·커밋 0).
판정 근거는 전부 아래에 명령과 원문 출력을 붙였다. 재지 않은 수치는 쓰지 않았다.

## 테스트 환경

| 항목 | 값 |
|---|---|
| 빌드 | 로컬 `./gradlew test bootJar --rerun-tasks` → jar 를 DELL 로 전송 → `eclipse-temurin:25-jre` 위에 이미지 생성(`legalcare/booster:p5qa`, `legalcare/gateway:p5qa`) |
| 병행 인스턴스 | `p5-booster`(172.28.0.33:18080) · `p5-gateway`(172.28.0.34:10000) · `p5-gateway-p6`(172.28.0.35:10000, P6 미리보기) |
| 네트워크 | `legalcare-local_legalcare` (**`internal=true`** — 기본 라우트 없음) |
| 신 DB | `deploy-postgres-new-1` / `medilawyer` (144테이블) — 실행 중 `booster-web-5` 와 공유 |
| 구(기준) | 라이브 `legacy-service` `127.0.0.1:41010`, 구 DB = `postgresql` 컨테이너(`-U legalcare`) |
| JWT | 병행 인스턴스에만 구 개발계 키 주입(`LegalcareSecretKeyMadeByJay…1`, `application-workstation.yml:17`). 실행 중 컨테이너 무접촉 |
| 프로파일 | `SPRING_PROFILES_ACTIVE=local,web`, `LEGACY_KAFKA_CONSUMER_ENABLED=true`, `LEGACY_SCHEDULING_ENABLED=true`, `BOOSTER_SCHEDULING_ENABLED=false` |

**골든셋 토큰을 그대로 못 쓴 이유(방식 변경, 반드시 읽을 것).** 골든셋은 **운영** 캡처라 토큰이
① 운영 키로 HS384 서명돼 있고(구 개발계 키·부스터 키 어느 것으로도 검증 실패 — 5개 후보 전수 대조) ②
`exp`가 2026-07-17 로 **이미 만료**다. 그래서 **클레임(`appUserId`·`email`·`organizationIdAndRole`)은
그대로 보존하고 서명·`exp`만 개발계 키로 다시 만든 토큰**으로 리플레이했다. 인가 의미는 보존되고
서명 주체만 바뀐다. 구·신에 **같은 토큰**을 쏘므로 차등검증의 대칭성은 유지된다.

### 독립 재실행 (개발자 보고 불신, 내가 직접)
```
booster : ./gradlew test bootJar --rerun-tasks  → BUILD SUCCESSFUL in 5m 12s
          classes=234 tests=901 failures=0 errors=0 skipped=8
gateway : ./gradlew test bootJar --rerun-tasks  → BUILD SUCCESSFUL
          classes=9 tests=84 failures=0 errors=0 skipped=0
Routes.java:34 = new Route("legacy", MEDILAWYER, "/api/v1", ServiceId.LEGACY)   ← 무변경
```
첫 실행은 18초에 끝나 캐시였다 — `--rerun-tasks` 로 다시 돌린 값이 위 수치다.

## 커버리지 (뭉뚱그리지 않음)

인벤토리 130행 = `/api/v1` **115** + `/internal` **14** + `/api/v2` **1**(선행 태스크분, 이번 검증 제외).

| 구분 | 경로 수 | 방법 |
|---|---|---|
| 골든셋 리플레이로 **실트래픽** 검증 | **20** | 구·신 이중발사(읽기) / 골든 기준(쓰기). 골든 `/api/v1` 엔트리 369건 중 OPTIONS 179 제외 **190건** |
| 계약 스모크로 검증 | **129** (리플레이 20 포함) | 경로·메서드·인증경계·바디형태 4프로브 × 129 = **309 프로브** |
| 두 방법 모두 미적용 | **0** | — |
| 구측을 못 찌른 프로브가 있는 경로 | **6** | 아래 BLOCKED |

골든셋 커버리지 20/129 = **15.5%**. (지시서의 "15경로"보다 5개 많다 — 내 실측은 20이고,
추가로 `GET .../message-templates/{id}` 는 구에 매핑이 없어 **405 축**으로만 잡힌다.)

### 검증 깊이 — "129경로 검증"을 그대로 읽으면 안 된다

계약 스모크의 비GET 프로브는 **핸들러를 실행시키지 않도록 일부러 설계**했다(415·미매핑 메서드).
발송·기프트·쓰기 경로를 구로 쏠 수 없기 때문이다. 그래서 경로마다 **어디까지 대조됐는지**가 다르다.

| 깊이 | 경로 수 | 무엇이 대조됐나 |
|---|---|---|
| **업무 로직 본문까지** 구·신 실행·대조 | **50** | 응답 데이터·에러 분기까지. (스모크 본문 대조 45 ∪ 리플레이 실트래픽 20) |
| **경계까지만** (401 / 405 / 415) | **79** | 경로 존재·메서드 매핑·인증 요구·에러 봉투. **핸들러 본문은 구·신 어느 쪽도 안 돌았다** |

즉 **130경로 이식의 "동작 동등"이 입증된 것은 50경로**이고, 나머지 79경로는 **계약 표면만** 같다는
것까지 확인했다. R9 가 정의한 스모크 축(경로·메서드·인증경계)은 충족하지만,
"미사용 71경로가 실제로 구와 같게 동작한다"는 이번 검증으로 보증되지 않는다.

## 시나리오 결과

| # | 시나리오 | 관련 AC | 기대 | 실제 | 판정 |
|---|---|---|---|---|---|
| 1 | 골든셋 읽기 샤드 119건 구·신 이중발사 | AC7① | 회귀 0 | 불일치 108건 → **원인별 분해: 데이터차이 102 / 코드결함 4(D1) / 구측미실측 2** | **FAIL**(D1) |
| 2 | 골든셋 쓰기 샤드 71건 | AC7① | 회귀 0 | 골든 200 → 신 비200 **71/71**. 분해: 데이터 30 / **코드결함 24(D2)** / 환경 17 | **FAIL**(D2) |
| 3 | 계약 스모크 129경로 309프로브 | AC9 | 구=신 | 상태 일치 283 · 상태 불일치 14 · 바디형태 불일치 176 — **전부 D1 한 축**. D1 제외 시 **129/129 완전 일치** | **FAIL**(D1) |
| 4 | 카프카 컨슈머 그룹 참여 + 토픽 왕복(정상1·실패2) | AC7② | 참여·왕복 성공, 구 무영향 | 구독 17토픽(구와 동일 목록) · 정상1 처리 · 실패2 graceful · 구 브로커 그룹 **무변화** | **PASS** |
| 5 | `@Scheduled` 3건 수동 트리거 | AC7③ | 3건 성공 | batch 200 · 긍정리뷰 200(155건 처리) · 부정리뷰 **00:44 에 500 → 01:00 재시도 200**(155건 처리). 500은 구에도 있는 시각 의존 결함(D3) | **PASS** |
| 6 | 신규 스택 전 컨테이너 healthy · 기동 예외 0 | AC7④ | 예외 0 | booster/gateway **기동 구간 ERROR 0**, 기존 94컨테이너 무손상(96 running = 94+내 2) | **PASS** |
| 7 | 외부 실호출 0건 | AC7⑤ | 0건 | 4계층 실증(라우트 없음·DNS 불가·소켓 36개 전부 사설·앱 게이트 차단) | **PASS** |
| 8 | 구 legacy-service·bridge 무손상 | R10 | 정지·재시작 0 | `RestartCount=0`, `StartedAt` 불변 | **PASS** |
| 9 | 게이트웨이 라우트 무변경 | R8 전 | LEGACY 유지 | 병행 게이트웨이 `/api/v1` → 구 봉투 응답 = LEGACY 로 감 | **PASS** |
| 10 | 구 DB 쓰기 | 안전조건 | 최소 | **0건**(대조표 아래) | **PASS** |
| 11 | 외부검색 5 + ML 1 경로 계약 대조 | AC9 | 구=신 | 구측을 찌를 수 없음(외부 실호출 금지) + 신측은 스텁 | **BLOCKED** |
| 12 | ML 로그인 3건 · 알림톡 발송 10건 판정 | AC7① | 구=신 | 신측 스텁/블랙홀이라 구와 동일 조건 재현 불가 | **BLOCKED** |

## 발견 결함

### D1. [MAJOR] 핸들러 진입 전(pre-handler) 오류 응답이 구 봉투를 벗어난다 — **129/129 경로 전면**

`LegacyExceptionAdvice` 는 `@RestControllerAdvice(basePackages = "legalcare.medilawyer.legacy")`
(`modules/legacy/src/main/java/legalcare/medilawyer/legacy/common/exception/LegacyExceptionAdvice.java:41`).
`basePackages` 는 **핸들러가 선택된 뒤**에만 적용된다. 필터 401·HandlerMapping 405/404 처럼
**컨트롤러에 닿기 전에 끝나는 응답**은 이 advice 를 못 타고 booster 전역 봉투로 나간다.

**게이트웨이 경유 실측(= 실사용자가 보는 것).** `Routes.java` 는 손대지 않고 `GATEWAY_SERVICES_LEGACY_URL`
환경변수만 booster 로 돌린 미리보기 인스턴스(`p5-gateway-p6`)와 현행을 나란히 찍었다.

| 축 | 현행(P6 전, 구) | P6 이후(신) | 상태코드 |
|---|---|---|---|
| 무토큰 | `{"errors":[{"code":"AUTH-001","message":"Full authentication is required to access this resource"}],"errorMessage":null,"path":"uri=…","timeStamp":"…"}` | `{"status":401,"code":"90008","message":"토큰이 필요합니다."}` | 401 → 401 (바디만) |
| **잘못된 서명 토큰** | `400` `AUTH-006 유효하지 않은 토큰입니다.` | `401` `90038 서명이 올바르지 않습니다.` | **400 → 401** |
| 미지원 메서드 | `500` `SERVER-001 Request method 'GET' is not supported` | `500` `{"code":"50000",…}` | 500 → 500 (바디만) |
| 매핑 없는 경로(토큰 동반) | `400`(구는 필터가 먼저 사용자 조회) | `404` `40400` | **400 → 404** |
| `/internal/**` 미매핑 메서드 | `401` `AUTH-001` | `403` `90403 해당 엔드포인트는 외부에서 접근할 수 없습니다.` | **401 → 403** (14경로) |

**재현 절차**
1. `docker run -d --name g6 --network legalcare-local_legalcare -e GATEWAY_SERVICES_LEGACY_URL=http://<booster>:18080 … legalcare/gateway:p5qa`
2. `curl -H 'Host: gateway.dev.revieworks.com' http://<g6>:10000/api/v1/app-users/me`
3. 같은 요청을 현행 게이트웨이(`legalcare-local-gateway-1` 또는 dev-edge `:19000`)로 반복해 대조

**영향** — R1 이 못 박은 "경로·요청/응답 바디·HTTP 상태·**에러코드** 무변경 틀"의 정면 위반이다.
`errors[0].code` 를 보고 분기하는 클라이언트는 전부 깨지고, **401/400 축은 상태코드까지 바뀌어**
토큰 만료 시 자동 로그아웃/재발급 인터셉터의 동작이 달라진다.
AC2 는 "무토큰 10경로 응답코드 동일"만 봤기 때문에 **바디 축과 잘못된-토큰 축이 검증 구멍으로 남아 있었다.**

**규모** — 계약 스모크 309프로브 중 **176프로브(바디)+14프로브(상태)**가 이 한 축이고,
**이 축 하나를 빼면 129/129 경로가 전 프로브 완전 일치**한다. 즉 결함은 산발이 아니라 단일 지점이다.

---

### D2. [CRITICAL] `POST /api/v1/post-reply-ai-recommend` 가 100% 500 — Kotlin→Java ID 타입 변환이 `persist`를 `merge`로 뒤집는다

골든셋에서 **`/api/v1` 최다 POST**(쓰기 71건 중 24건, 운영 응답 전부 200)인데 신에서 **24/24 전부 500**.

```
신 응답: {"errors":[{"code":"SERVER-001","message":
  "org.hibernate.TransientPropertyValueException: Persistent instance of
   '…legacy.db.domain.post.Post' references an unsaved transient instance of
   '…legacy.db.domain.postReplyAiRecommend.PostReplyAiRecommend'
   (persist the transient instance before flushing)
   [Post.postReplyAiRecommend -> PostReplyAiRecommend]"}]}
골든(구 운영) 응답: {"status":"200","message":"AI 추천 답글 조회가 완료되었습니다.", …}
```

**근본 원인 — 서비스 코드는 1:1 이 맞다. 엔티티 ID 필드의 JVM 타입이 갈렸다.**

```
구(Kotlin)  PostReplyAiRecommend.kt:10-13   val id: Long = 0
  javap →   private final long id;                     ← JVM 원시형
신(Java)    PostReplyAiRecommend.java:25-28  @Id @GeneratedValue(IDENTITY) private Long id = 0L;
  javap →   private java.lang.Long id;                 ← 래퍼, 그리고 null 이 아님
```
Spring Data `AbstractEntityInformation.isNew()` 는 **ID 타입이 원시형일 때만** `0`을 "새 엔티티"로 본다.
래퍼면 `id == null` 로만 판정하므로 `0L`은 **기존 엔티티**가 되어 `SimpleJpaRepository.save()` 가
`persist()` 대신 **`merge()`** 를 탄다. `merge()` 는 관리 사본을 새로 만들어 INSERT 하지만,
`newReply.addPost(post)`(`PostReplyAiRecommend.java:49-52`)가 이미 `post.postReplyAiRecommend` 에 꽂아 둔
것은 **원본(비관리) 인스턴스**다 → `post` flush 시점에 위 예외.

SQL 로그로 확인(내가 `SPRING_JPA_SHOW_SQL=true` 로 병행 인스턴스만 재기동해 관찰):
```
Hibernate: select p1_0.id,… from post …
Hibernate: insert into post_reply_ai_recommend (ai_recommendation_reply,created_at,post_id,updated_at,usable) values (?,?,?,?,?)
→ 사본은 INSERT 되지만 post 가 참조하는 원본은 비관리 → 예외 → 트랜잭션 롤백(행 증가 0 실측)
```

**재현 절차**
1. `p5-booster` 기동(위 환경)
2. `curl -X POST -H 'Content-Type: application/json' 'http://<booster>:18080/api/v1/post-reply-ai-recommend?postId=NAVER_MAP_6a4dbc82ce3d590ee12032bc'`
3. 500 + `TransientPropertyValueException` (100% 재현, 25/25)

**관련 파일:라인**
- `modules/legacy/src/main/java/legalcare/medilawyer/legacy/db/domain/postReplyAiRecommend/PostReplyAiRecommend.java:25-28` (ID 타입)
- 같은 파일 `:49-52` (`addPost`)
- `modules/legacy/src/main/java/legalcare/medilawyer/legacy/apis/http/postReplyAiRecommend/service/PostReplyAiRecommendService.java:41-46`
- 구 대응 `medilawyer-boot/.../postReplyAiRecommend/PostReplyAiRecommend.kt:10-13`

**확산 위험 — 같은 패턴이 legacy 모듈 엔티티 16개에 있다.**
`Store` · `AppUser` · `AppUserStore` · `AppUserChannelAccount` · `StoreChannel` · `StoreChannelAttribute` ·
`PostReply` · `PostReplyChild` · `PostReplyRegistrationRequest` · `PostReplyAiRecommend` · `PostWord` ·
`Product` · `ProductPost` · `Otp` · `BoostMessageDelivery` · `ReviewRequestSetting`
(`grep -A3 GenerationType.IDENTITY … | grep "private Long .* = 0"`).
증상이 드러나는 조건은 "다른 관리 엔티티가 새 인스턴스를 참조한 채 flush"뿐이라 나머지 15개가
지금 전부 깨진다는 뜻은 아니다 — **다만 신규 INSERT 경로 전수 재점검이 필요하다.** 이번 리플레이가
덮은 쓰기 경로는 4종뿐이라 나머지는 확인하지 못했다.

**제안** — `= 0L` 초기값을 지우고 `private Long id;` 로 두면 `isNew()` 가 `id == null` 로 true 가 된다.

---

### D3. [MINOR·이식된 기존 결함] 부정리뷰 `@Scheduled` 이 매일 00:30 에 죽는다

```
POST /api/v1/schedule/alimtalk/new-negative-post  (00:44 KST 실행)
→ 500 {"errors":[{"code":"SERVER-001","message":"Invalid value for HourOfDay (valid values 0 - 23): -1"}]}
```
`ScheduledHelper.negativeTimeRange` 의 `currentTime.withHour(currentTime.getHour() - intervalTime)` 가
0시에 `withHour(-1)` 이 된다. `NEGATIVE_REVIEW_CRON = "0 30 * * * *"`(매시 30분) · `NEGATIVE_INTERVAL_HOUR = 1` ·
`NEGATIVE_REVIEW_START_HOUR = 10` 이라 **00:30 발화는 반드시 실패**한다.

**구와 신이 완전히 같다** — 상수 4개(10/19/30/1)와 분기 구조가 동일(`ScheduledHelper.kt:10-30`
vs `ScheduledHelper.java:23-42`). **이번 이식이 만든 회귀가 아니라 구에 있던 결함을 그대로 옮긴 것**이므로
동등성(AC) 위반이 아니다. 다만 운영에서 매일 00:30 에 실패하는 잡이므로 별도 백로그로 남긴다.

**시각 의존이라는 것도 실증했다** — 같은 요청을 **01:00 KST 에 재시도하니 200**:
```
[부정리뷰] 시간 = 2026-08-11T00:30+09:00 ~ 2026-08-11T01:30+09:00 필터링된 teamAppUser 개수: 155
{"status":"200","message":"알림톡 부정리뷰 생성 스케줄링 (테스트용)을 완료했습니다.","data":{}}
```
→ AC7③ 의 "3건 수동 트리거 성공"은 충족된다(3/3). 발송은 일어나지 않았다(대상 없음 + 블랙홀).

---

## 내가 만든 부작용 (자진 신고)

| 무엇 | 상세 | 처리 |
|---|---|---|
| 신규 PG 에 1행 INSERT | 계약 스모크의 415 프로브가 `PUT /internal/product/review-boost/stores/164/request-setting` 에서 **핸들러까지 진입**했다(이 경로는 `Content-Type` 을 안 가린다 — 내 "415 는 핸들러를 안 태운다" 가정이 이 경로에서만 깨졌다). `boost_review_request_setting` 에 `id=2, store_id=164` 생성 | **삭제로 원복.** 최종 대조 `tables=144 rows_total=45,845,365` 로 **기준선과 완전 동일, 변화 테이블 0** |
| 구 PG 쓰기 | 같은 프로브가 구에도 200 을 냈으나 구는 이미 store 164 행 보유 → **행수 3 불변, `updated_at` 도 2026-08-06 그대로** | **쓰기 0건** |
| 신 브로커에 테스트 메시지 4건 | `dev.medilawyer.event.system.post.insult-defamation-generation.result` 에 정상1·실패3. **신 브로커(`legalcare-local-kafka-1`)에만** 발행 | 개발계 신 스택 전용 토픽, 구 브로커 무접촉 |
| 병행 컨테이너 3개 | `p5-booster` · `p5-gateway` · `p5-gateway-p6` | **회수 완료** — `docker ps -q` 94개로 복귀, 기준선 대비 diff 0줄 |

## 엣지 케이스 검증 내역

**카프카 격리 판단과 근거 (지시받은 "판단하고 근거를 남겨라").**
groupId `medilawyer-spring-boot` 는 `KafkaConsumer.java:191` **하드코딩 리터럴**이라 env 로 못 바꾼다.
그런데 **구·신이 서로 다른 브로커 클러스터를 본다** — 구는 `192.168.1.55:9092,9093`(`kafka1`/`kafka2`,
`cloud_config_repository/kafka-workstation.yml:3`), 신은 `kafka:9092` = `legalcare-local-kafka-1`(172.28.0.3).
컨슈머 그룹은 클러스터별 네임스페이스라 **같은 이름이어도 리밸런스가 교차하지 않는다.**
그래서 별도 그룹을 만들 필요 없이 그대로 켰고, 실측으로 확인했다.

```
[기동 전] 구 브로커 medilawyer-spring-boot : consumer-id 1개(/192.168.1.55), 파티션 17
          신 브로커 medilawyer-spring-boot : 없음
[기동 후] 구 브로커 : consumer-id 1개(/192.168.1.55), 파티션 17   ← 변화 없음
          신 브로커 : consumer-…-b8aee6f3 (/172.28.0.33), 17토픽 구독
```
(지시서에 적힌 "그룹 멤버 3개"와 다르다 — 내 실측은 **distinct consumer-id 1개**, 파티션 17이다.
구독 토픽 목록 17개는 구와 문자열 단위로 동일.)

토픽 왕복: 정상 1건 → `Sent PostAddReq to RabbitMQ …: newPostAddReqList 개수 = 0` / `hasCarrotMarket:false`
(핸들러 본문 도달), 스키마 위반 1건·깨진 JSON 1건 → `Failed to convert message to PostAddReq: …` 로
잡고 **컨테이너 생존·오프셋 정상 진행(4/4, lag 0)**. 포이즌필 루프 없음.

**외부 실호출 0건 — 4계층 실증.**
```
① 라우트   : /proc/net/route 에 기본 게이트웨이 없음 (network internal=true)
② DNS      : sens.apigw.ntruss.com / generativelanguage.googleapis.com / hooks.slack.com /
             openapi.naver.com / s3.ap-northeast-2.amazonaws.com  → 전부 (해석불가)
③ 소켓     : ESTABLISHED 36개 = 172.28.0.10:5432(20) · 172.28.0.3:9092(9) ·
             172.28.0.5:3306(5) · 172.28.0.6:27017(2)  → 사설/루프백 밖 0
④ 앱 게이트: [egress] 외부 송신 차단(기본값) / 실제 발화 "외부 송신이 차단됐다
             (external.egress.enabled=false): vendor=GEMINI, at=ExternalClientService.makeBoosterReview"
```
※ `/proc/net/tcp` 만 보면 ESTABLISHED 가 0으로 나온다(JVM 은 IPv4-mapped IPv6 소켓을 쓴다).
처음 그렇게 측정했다가 파서를 고쳐 `/proc/net/tcp6` 까지 판 값이 위 36개다.

**반대로 구에서는 돈이 실제로 나간다 — 확인만 하고 찌르지 않았다.**
`BizMessageService.kt:95` `sendBoostAlimTalkSvc` 에는 `serviceMode` 가드가 **없고**(같은 파일 `:54,56`
의 `sendAlimTalkSvc` 에는 있다), 구 개발계 `application-workstation.yml:36` 의
`sesBaseUrl = https://sens.apigw.ntruss.com` + 실 액세스키를 든 채 **host 네트워크**에서 돌아
`sens.apigw.ntruss.com → 110.165.28.15` 로 실제 해석된다.
→ **`POST /api/v1/boost1/**` · `schedule/alimtalk/**` · `otp/**` 등 발송·기프트 경로는 구로 단 한 건도
쏘지 않았다.** 리플레이 하니스에 `OLD_DENY` 목록으로 못 박아 두었고(사유 문자열 포함),
계약 스모크의 비GET 프로브도 구·신 양쪽에서 **핸들러 미실행 프로브(415 / 미매핑 메서드)** 로만 구성했다.
미매핑 메서드는 경로별 매핑 집합을 계산해서 골랐다 — 안 그러면 `…/gifts/{giftId}` 처럼 GET·POST·DELETE
가 함께 걸린 경로에서 **실제 삭제 핸들러가 돈다.**

**불일치의 회귀/데이터차이 판정 (원인마다).**
- 읽기 85건 `구 400 USER-002 → 신 200` : 구 응답 본문의 이메일을 뽑아 양쪽 `app_user` 를 조회 →
  `hanhohee@gmail.com`(구 0/신 1) 53건 + `cityamc@naver.com`(구 0/신 1) 32건 = **85/85 데이터차이**
- 읽기 17건 `200/200 바디 상이` : 정렬만 다른 것 2 · 키집합 차이 6 · 값 차이 9.
  표본 검증 — `organization 01JVPHVB9D71YS0PB8FT6KZWTD` 이름이 구 `솔동물병원` / 신 `24시 솔동물의료센터`,
  `team_channel_account(01JT022SVDQFZQ3GRTQKSE7FJA, NAVER_MAP).usable` 이 구 `t` / 신 `f`,
  `강남구립행복요양병원` store id 가 구/신 상이 → **전부 DB 내용 차이**
- 쓰기 30건 : `BoostMessageSend 정보를 찾을 수 없습니다` — 운영 messageId 가 신 DB 에 없음 = 데이터차이
- 쓰기 17건 : 알림톡 블랙홀 10 · egress 게이트 4 · ML 스텁 3 = **환경 인공물**(판정 불가 → BLOCKED)
- 두 DB 는 실제로 다르다: `app_user` 구 786 / 신 858(교집합 783), `store` 64,644 / 79,584,
  `team` 125 / 107 — 그래서 인증 경로 비교에는 **양쪽에 동일하게 존재하는 신원**(121조합 중
  `1116bk@naver.com`/795/org `01KF2KE1RMP98Q04YJHDGK6EGV`/team `01KF2KKQKPMVVCST250RP6QJV8`)을 썼다

**기존 개발계 무손상 (검증 중 · 회수 후 2회 확인).**
```
기준선 94 running → 검증 중 최대 97 running (= 94 + 내 3개) → 회수 후 94 running
회수 후 baseline 대비 diff: 0줄 ("차이 없음 — 개발계 기준선 복귀 확인")
legacy-service                  RestartCount=0  StartedAt=2026-08-10T03:46:38Z (불변)
legalcare-local-legacy-bridge-1 RestartCount=0  StartedAt=2026-08-06T08:26:44Z (불변)
신 DB 최종: tables=144 rows_total=45,845,365 — 기준선과 동일, 변화 테이블 0
구 DB: boost_review_request_setting 3행, updated_at 최신값 2026-08-06 (오늘 쓰기 흔적 0)
docker compose 미사용 · 기존 컨테이너 정지/삭제/재생성 0 · 운영 RDS 접근 0 · 커밋 0
```
`legalcare-local-reviewed-web-1 Exited(143) 3 days ago` 는 **기준선에 이미 있던 상태**로 내 작업과 무관.

**스케줄러 킬스위치 실증.** `LEGACY_SCHEDULING_ENABLED=true` + `BOOSTER_SCHEDULING_ENABLED=false` 에서
`GET /actuator/scheduledtasks` = `{"cron":[],"custom":[],"fixedDelay":[],"fixedRate":[]}` —
빈은 등록되되 **크론은 한 건도 걸리지 않는다**(AC10 의 "이중 실행 0건"을 구조로 보장).

## BLOCKED (실행 불가 항목과 사유)

| 항목 | 사유 |
|---|---|
| `store-channel/search/{naver,unofficial/naver,kakao,modoodoc,google}` 5경로 · `GET /api/v1/ml/capture-image` — 인증 통과 후 동작 대조 | 구는 네이버·카카오·구글·모두닥 **실 API 를 호출**한다(안전조건 "외부 실호출 0건"). 신은 스텁으로 돌려놨으므로 동일 조건 재현 불가. **경로·메서드·미매핑메서드 축은 검증됨**, 응답 본문 축만 미검증 |
| `channel-accounts/login` 3건 | 신이 ML 스텁을 보고 400 `REQUEST-003`, 골든(구)은 200 + `result:FAIL`. 실 ML 서버 없이는 P2 라운드1 BLOCKER 수정의 정상 동작을 확인할 수 없다 |
| `POST /api/v1/boost1/**/messages` 10건 | 신은 블랙홀(`127.0.0.1:19999`) I/O 오류로 500. 구는 실발송이라 대조 불가 |
| `POST /api/v1/post-reply-ai-recommend` 구측 라이브 대조 | 구 ML `reply_gpt_return` 이 상위 LLM 과금을 유발할 수 있어 미발사. **구 기준값은 골든셋 200 × 24건을 사용** |
| `POST /api/v1/otp/send` | 신에서 `500 SERVER-001` + **빈 메시지**(`"message":""`)로 실패하고 `otp` 행 변화 0(508→508). 원인이 스텁 메일서버 응답인지 이식 결함인지 가릴 수 없다 — 구는 실 메일서버(`https://mail.legalcare.ai`)라 대조 불가. **D2 의 `TransientPropertyValueException` 은 여기서 재현되지 않았다**(같은 `Long id = 0L` 패턴인 `Otp` 엔티티인데도) → D2 의 발현 조건이 "다른 관리 엔티티가 새 인스턴스를 참조한 채 flush"라는 분석과 일치 |
| 카프카 컨슈머 40개 분기 중 1개만 왕복 | P4 응답의 "뮤턴트 40/40 생존" 구간. 토픽별 페이로드 픽스처가 없어 나머지 16토픽 핸들러는 미검증 |

## 총평

**단일 지점 결함 2건이 AC7 을 막는다.**

계약 스모크가 보여준 그림은 오히려 좋은 소식이다 — **D1(에러 봉투) 한 축을 제외하면 129경로 309프로브가
전부 구와 일치**한다. 130경로 이식 자체의 정확도(URL·메서드·경로변수·인증경계·정상 응답 형태)는
매우 높고, 불일치는 산발이 아니라 **두 곳에 몰려 있다.**

- **D2 가 배포 차단 사유다.** 운영 트래픽에서 `/api/v1` 최다 POST 인 AI 추천 답글이 100% 500 이다.
  원인이 "Kotlin 원시형 → Java 래퍼" 라는 **1:1 이식이 놓치기 쉬운 축**이고 같은 패턴이 엔티티 16개에
  있어, 고치더라도 **신규 INSERT 경로 전수 재점검 없이는 다른 곳에서 같은 방식으로 또 터진다.**
  리플레이가 덮은 쓰기 경로가 4종뿐이라 이번 검증으로는 나머지를 보증할 수 없다.
- **D1 은 P6 라우트 전환의 전제 조건이다.** 상태코드까지 바뀌는 축(400→401, 401→403)이 있어
  R1 "에러코드 무변경"과 정면으로 어긋난다. `LegacyExceptionAdvice` 를 패키지 한정 advice 로 둔 이상
  필터·HandlerMapping 단계는 구조적으로 못 잡으므로, 봉투를 맞추려면 **경로 기준(`/api/v1`, `/internal`)
  으로 봉투를 고르는 전역 지점**이 필요하다. 지금 상태로 라우트를 돌리면 전 클라이언트의 에러 처리가 바뀐다.

방법론적으로 정직하게 남길 것 세 가지.
① **골든셋 토큰은 재서명해서 썼다**(운영 키 미보유 + 만료). 인가 클레임은 보존했다.
② **구·신 DB 내용이 다르다**(app_user 786 vs 858 등). 그래서 상태코드 불일치의 다수(102/108)가
데이터차이였고, 이를 요청 단위로 증명하는 데 검증 시간의 상당 부분을 썼다. 진짜 동등성 판정을 원하면
**구 DB 를 신 DB 로 정렬한 뒤 다시 돌려야** 판정 신뢰도가 올라간다.
③ **AC7 ⑤(외부 0)는 통과했지만 그 이유의 절반이 "네트워크가 봉인돼 있어서"다.** 앱 레벨 게이트가
Gemini 를 실제로 막는 것은 실측했으나(egress 게이트 발화), 알림톡·SMS 축은 게이트가 아니라
`sesBaseUrl` 기본값이라는 **설정 한 줄**에 걸려 있다. 운영 프로파일에서 그 값이 실주소가 되는 순간
`sendBoostAlimTalkSvc` 에는 `serviceMode` 가드가 없다 — **구와 동일한 위험을 그대로 안고 간다.**

그리고 **커버리지를 뭉뚱그리지 말 것** — 이번 검증으로 "구와 같게 **동작한다**"가 입증된 것은
**50/129 경로**다. 나머지 79경로는 계약 표면(경로·메서드·인증경계·에러 봉투)만 같다는 것까지다.
D2 를 고친 뒤에는 **쓰기 경로를 실제로 태우는 리플레이**(구·신 DB 정렬 + 발송 스텁 확보)가 한 번 더 필요하다.

CRITICAL 1건 · MAJOR 1건 → 판정 규칙에 따라 FAIL.

## 재현·회수 상태

병행 셋트는 **회수 완료**(개발계는 기준선 94컨테이너로 복귀). 재검증 자산은 DELL `~/p5-qa/` 에 남겼다 —
`p5_replay.py`(OLD_DENY 안전목록 포함) · `p5_smoke.py` · `cluster.py`/`deepdiff.py`/`summary.py` ·
`egress_proof2.sh` · `kafka_roundtrip.sh` · `sched_trigger.sh` · `rowcount.sh` ·
`replay-{read,write}.jsonl` · `smoke-report.jsonl` · `rows-{before,final}.tsv` · `p5-booster.env`.
이미지 `legalcare/booster:p5qa` · `legalcare/gateway:p5qa` 도 남아 있어 아래 한 줄로 즉시 재기동된다.
```
docker run -d --name p5-booster --network legalcare-local_legalcare \
  --env-file ~/p5-qa/p5-booster.env legalcare/booster:p5qa
```
**주의**: 재현 시에도 `p5-booster.env` 의 `NAVER_CLOUD_SES_BASE_URL`·`LEGACY_*_URL` 스텁 값을
절대 실주소로 바꾸지 말 것. 그 한 줄이 알림톡 실발송을 가르는 유일한 스위치다.

QA_VERDICT: FAIL
