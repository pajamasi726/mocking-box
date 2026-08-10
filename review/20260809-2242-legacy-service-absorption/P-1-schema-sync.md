# P-1 스키마 동기화 — 실측 기록

대상: 신규 PG `deploy-postgres-new-1` / DB `medilawyer`
원본 코드: `/Users/steve/steve/legal-care/medilawyer-boot` (git `930c83cf`, **읽기만 함**)
DDL 산출물: `legalcare-renew-prodsync-wt/apps/booster-app/app/src/main/resources/db/legacy-absorption-p1-schema.sql`
매핑 원본: `P-1-endpoint-table-map.tsv` (150행)

---

## 0. 전제 정정 — "59개 부족"은 폐기했다

착수 지시서(`01-requirements.md` R11, `00-log.md` 23:15)는 **개발계 `postgresql` 컨테이너(153) − 신규 PG(94) = 59 부족**을 전제로 삼았다. 오케스트레이터의 중간 정정대로 **이 뺄셈은 짝이 틀렸다.** 실측으로 확인한 계보는 이렇다.

| | 개발계 `postgresql` | 신규 `deploy-postgres-new-1` |
|---|---|---|
| 계보 | 개발용 별도 인스턴스 | **2026-07-14 운영 RDS(`medilawyer-prod`) 복사본** (`mocking-box/deploy/config.yaml`) |
| public 테이블(BASE TABLE) | **152** | **93** (적용 전) |
| `store` / `app_user` / `post` 행수 | 0 / 2 / 18 | **79,584 / 858 / 9,852,411** |

개발계 컨테이너는 코어 테이블이 사실상 비어 있는 **스크래치 DB** 다. 신규 PG 는 운영 실데이터를 담고 있다. 두 DB 의 테이블 목록 차집합(62건)은 "부족분"이 아니라 **서로 다른 두 계보의 차이**일 뿐이므로 판정 근거로 쓰지 않았다.

복사 도구가 무엇을 가져왔는지도 확인했다 — `mocking-box/internal/pg/copy.go:155` 이 `WHERE n.nspname=$1 AND c.relkind='r'` 로 **일반 테이블 전수**를 열거한다. 필터 목록이 아니라 전수이므로, 신규 PG 에 없는 테이블은 **2026-07-14 시점 운영에 없었다**고 읽는 것이 타당하다. (`relkind='p'` 인 파티션 부모는 복사 대상에서 빠지지만, 자식 파티션(`'r'`)도 하나도 없으므로 파티션 테이블 자체가 없었다고 판단했다.)

그래서 판정 기준을 이렇게 바꿨다: **구 소스를 읽어 접촉 테이블을 알아낸 뒤, 그 각각이 신규 PG 에 있는지만 본다.**

---

## 1. 접촉 매핑 — 방법과 결과

### 1.1 방법

1. 구 저장소 `@RestController` 38개(= `core-api` 35 + `core-client` 1 + `core-common` 2)에서 `@*Mapping` 을 전수 추출
2. 컨트롤러 → Service/Facade → Repository(QueryDSL 포함) → 엔티티 → `@Table(name=…)` 로 테이블명 확정
3. 네이티브 SQL(`nativeQuery=true` 5곳, `JdbcTemplate` 1곳)은 SQL 문자열에서 직접 테이블명 추출
4. JPA 지연로딩으로만 읽히는 연관 테이블도 접촉으로 계산(응답 DTO 조립 중 SELECT 발생)
5. 5개 병렬 에이전트로 분할 추적 후 교차 검증, 판정이 갈린 항목은 직접 소스로 재확인

### 1.2 엔드포인트 개수 — 147 이 아니라 130 이다

| 구분 | 수 |
|---|---|
| `@*Mapping` 어노테이션 총계 | 149 |
| − 클래스 레벨 `@RequestMapping` 접두 (엔드포인트 아님) | 19 |
| **= 실제 핸들러 엔드포인트** | **130** |
| ├ `/api/v1` | **115** |
| ├ `/internal/**` | 14 |
| └ `/api/v2` (범위 밖) | 1 |

지시서의 "147"은 **149 − HealthCheck 2건**으로 보인다. 클래스 레벨 접두 19건이 엔드포인트로 중복 계상된 수치다. TSV 에는 130개 핸들러 행 + 19개 접두 행 = 149행(+헤더) 을 모두 남겼고, 접두 행은 `판정 = 엔드포인트 아님`으로 표시했다. **어느 행도 접촉 테이블 칸이 비어 있지 않다** (DB 무접촉은 `(DB 무접촉)` 로 명시).

참고: `CommunityController` 는 `@Deprecated("사용안함")` 이고 클래스 본문이 비어 있어 핸들러가 0개다. `TeamChannelAccountController` · `TeamAuthController` · `TeamAppUserController` · `OrganizationAppUserController` 4개도 매핑 0건이다.

### 1.3 구 앱이 만지는 테이블 = 50개

| 출처 | 수 |
|---|---|
| JPA 엔티티 `@Table` | 49 |
| 네이티브 SQL 전용 (`crawling_fail_log`, `CrawlingFailLogRecorder.kt:101`) | 1 |
| **합계** | **50** |

이 중 **44개는 신규 PG 에 운영 데이터와 함께 이미 있다.** 부재는 6개다.

### 1.4 부재 6개 — 접촉 여부와 처분

| 테이블 | 접촉 엔드포인트 | 개발계 PG | 신규 PG(적용 전) | 처분 |
|---|---|---|---|---|
| `crawling_fail_log` | **POST `/api/v1/store/store-channel/completed`** + Kafka 리스너 | 있음(`p`) | 없음 | **생성** |
| `boost_review_request_setting` | `/internal/**` 4경로 (+`/api/v2` 1) | 있음(3행) | 없음 | **생성** |
| `boost_message_delivery` | `/internal/crawling/review-boost/stores/{storeId}/messages` (+`/api/v2` 1) | 있음(8,592행) | 없음 | **생성** |
| `app_user_channel_account` | **`/api/v1` 5경로** | **없음** | 없음 | **생성 안 함** (아래 3절) |
| `admin` | 없음 (엔티티만) | **없음** | 없음 | 생성 안 함 |
| `post_reply_history` | 없음 (엔티티·리포지토리 고아) | **없음** | 없음 | 생성 안 함 |

`crawling_fail_log` 의 HTTP 도달은 판정이 갈렸던 항목이라 직접 소스로 확인했다:

```
StoreController.kt:93   @PostMapping("/api/v1/store/store-channel/completed")
  → StoreFacade.kt:117    storeChannelService.updateStoreChannel(dto)
    → StoreChannelService.kt:257  crawlingFailLogRecorder.recordTargetNotFound(...)
    → StoreChannelService.kt:281  crawlingFailLogRecorder.recordCrawlFail(...)
      → CrawlingFailLogRecorder.kt:101  INSERT INTO crawling_fail_log ...
```

카프카 리스너(`KafkaConsumer.kt:166`)와 **같은 파사드 메서드를 공유**하므로 HTTP 경로만으로도 INSERT 가 난다. 기록기는 `TransactionTemplate(REQUIRES_NEW)` 를 `runCatching` 안에서 돌려 테이블이 없어도 WARN 만 남기고 조용히 실패한다 — HTTP 상태코드는 200 그대로라 **응답 비교로는 안 잡히고 write-set 비교에서만 드러난다.** 이것이 이 테이블을 반드시 만들어야 하는 이유다.

### 1.5 알림톡 발송이력은 구 앱이 안 쓴다

`kakao_alimtalk_send_history` / `alimtalk_send_history` / `SendHistory` — 저장소 전체 grep **0건**. `BizMessageService`(`module-client/core-client/.../bizmessage/service/BizMessageService.kt`)는 생성자에 Repository 를 하나도 받지 않고 `RestTemplate` 으로 NCP SENS 를 호출할 뿐이다. `/api/v1/notice` · `/api/v1/negative-review-alimtalk` · `/api/v1/schedule/alimtalk/*` 의 접촉 테이블은 전부 **발송 대상 조회용 SELECT** 뿐이고 발송 결과 쓰기는 없다. v1 은 발송 원장이 없고 `boost_message_delivery` 는 v2 경로에서만 쓰인다 — **v1/v2 비대칭**이다.

---

## 2. DDL 적용

### 2.1 스키마 원본을 어디서 가져왔나

운영 DB 는 읽지 않았다. 세 후보 중 **저장소 정본 + 구 엔티티**를 근거로 삼았다.

**`crawling_fail_log`** — 정본은 이미 저장소에 있다: `apps/booster-app/app/src/main/resources/db/crawling-fail-log.sql`.
개발계 PG 의 실제 구조와 대조한 결과 **컬럼 20개·타입·기본값·인덱스명 4종(`idx_cfl_occurred`/`_channel_day`/`_status_occurred`/`_pipeline_occurred`)·테이블 코멘트 문자열까지 일치**했다. 파티션 경계도 동치다(저장소 `'2026-08-01 00:00:00+09'` = 개발계 `'2026-07-31 15:00:00+00'`). 즉 개발계 구조는 이 파일을 그대로 실행한 결과이고, **파일이 정본**이다. 새 파일에 복제하지 않고 그대로 적용했다.

**`boost_review_request_setting`** — 구 엔티티 `ReviewRequestSetting.kt:15` 가 계약이다. 저장소 `db/reviewboost-v2-alimtalk.sql` §3 에도 CREATE 가 있으나 CHECK 제약 2종과 부분 인덱스가 빠진 축소판이라, P-1 파일에서 전체 제약을 갖춰 정의했다. 두 파일은 **어느 순서로 돌려도 서로 no-op** 이 되도록 작성했다.

**`boost_message_delivery`** — 구 엔티티 `BoostMessageDelivery.kt:18` 이 계약이다(컬럼명·길이·nullable·유니크 2종이 1:1 대응). 저장소 main 트리에 DDL 이 없고 `app/src/test/resources/db/reviewboost-v2-postv2-fixture.sql` 에만 있는데, 그 파일은 스스로 *"스키마 진실이 아니다 — 실제 개발계/운영 DB 에는 절대 적용하지 말 것"* 이라고 선언하고 실제로 `hospital_display_name` 의 NOT NULL 이 빠져 있다. 따라서 **엔티티를 근거로 P-1 파일에 새로 작성**했다.

세 테이블 모두 **2026-07-14 시점 운영에 없던 것들**이다(1.4 표). 운영 현재 구조와 같은지는 확인하지 않았다 — 4절 미판정 참조.

### 2.2 적용 명령과 결과

```
$ docker cp .../crawling-fail-log.sql deploy-postgres-new-1:/tmp/
$ docker cp .../legacy-absorption-p1-schema.sql deploy-postgres-new-1:/tmp/
$ docker exec deploy-postgres-new-1 psql -U postgres -d medilawyer -v ON_ERROR_STOP=1 -f /tmp/crawling-fail-log.sql
CREATE TABLE ×7 / CREATE INDEX ×4 / COMMENT          exit=0
$ docker exec deploy-postgres-new-1 psql -U postgres -d medilawyer -v ON_ERROR_STOP=1 -f /tmp/legacy-absorption-p1-schema.sql
CREATE TABLE ×2 / ALTER TABLE ×3 / DO / CREATE INDEX / COMMENT ×4   exit=0
```

**적용 순서는 두 파일 모두 필요하다** — P-1 파일은 `crawling_fail_log` 를 정의하지 않는다(드리프트 방지).

생성된 객체:

```
boost_message_delivery        | r | rows=0
boost_review_request_setting  | r | rows=0
crawling_fail_log             | p | rows=0        ← 파티션 부모
crawling_fail_log_2026m08..12 | r | 각 rows=0     ← 5개
crawling_fail_log_default     | r | rows=0
```

**파티션 범위 판단**: 저장소 정본이 만드는 당월(2026m08)~2026m12 5개 + DEFAULT 로 충분하다. 골든셋 캡처는 2026-07-16~17 이지만 `occurred_at DEFAULT now()` 이므로 리플레이 INSERT 는 **실행 시각(2026-08) 파티션**으로 간다. 경계 밖은 DEFAULT 가 받는다. 개발계 PG 에는 2027m07 까지 있으나 리플레이 판정에 불필요해 따라가지 않았다(과잉 생성 회피).

실동작 검증(행은 남기지 않도록 ROLLBACK):
```
BEGIN; INSERT INTO crawling_fail_log (source,pipeline,stage,reason,payload) VALUES ('p1-verify','MAP','CRAWL_FAIL','partition routing check','{"k":1}'::jsonb);
routed_to=crawling_fail_log_2026m08          ← 파티션 라우팅 정상
status_default=FAILED                        ← 기본값 정상
ROLLBACK;   → 잔여 행 0

BEGIN; INSERT INTO boost_review_request_setting (store_id,usable,delivery_mode) VALUES (999999999,true,'HOSPITAL_CUSTOM');
ERROR: violates check constraint "ck_review_request_template_by_mode"   ← 제약 정상 작동
```

### 2.3 멱등 — 두 번 돌려 확인

```
RUN 2: crawling-fail-log.sql            exit=0, ERROR 0건
RUN 2: legacy-absorption-p1-schema.sql  exit=0, ERROR 0건, NOTICE(skip) 6건
  NOTICE: relation "crawling_fail_log" already exists, skipping
  NOTICE: relation "crawling_fail_log_2026m08..12" / "_default" already exists, skipping
  NOTICE: relation "idx_cfl_occurred" already exists, skipping
  NOTICE: relation "boost_message_delivery" already exists, skipping
  NOTICE: relation "boost_review_request_setting" already exists, skipping
  NOTICE: column "delivery_mode"/"hospital_message_template_id"/"hospital_display_name" already exists, skipping
  NOTICE: relation "idx_review_request_setting_template" already exists, skipping
```

PG 는 `ADD CONSTRAINT IF NOT EXISTS` 를 지원하지 않아 CHECK 제약은 `pg_constraint` 존재검사 DO 블록으로 감쌌다.

### 2.4 기존 테이블 무손상 — 전후 대조

적용 전후로 `information_schema.columns` 전체를 덤프해 신규 3테이블만 제외하고 비교했다.

```
before:  814 lines  md5 837fa4a03cb9f3466cc05f08a72d6408
after :  814 lines  md5 837fa4a03cb9f3466cc05f08a72d6408   ← diff 0
```

**기존 94테이블의 컬럼·타입·nullable 이 한 글자도 바뀌지 않았다.** 테이블 수는 94 → 103 (`information_schema.tables`), 순증 9 = 파티션 부모1 + 파티션6 + 일반2.

구 PG(`postgresql` 컨테이너)는 `pg_dump -s` 와 SELECT 만 했고 적용 후에도 **152 테이블 그대로**다.

### 2.5 신규 스택 정상 확인

```
deploy-postgres-new-1  healthy / running
legalcare-local-*      28개 전부 Up (booster-web / booster-worker / gateway 포함)
booster-web 로그 (최근 10분): "does not exist" 0건, SQLException 0건
booster-worker 로그: 0건
:10000 /actuator/health → 200
```

booster-web 로그의 `404 Not Found: GET actuator/metrics/*` WARN 은 대시보드 폴링이 만드는 기존 소음으로 DDL 과 무관하다.

**관측 1건(P-1 범위 밖)**: 게이트웨이 경유 `/api/v1/health` 가 404 다. `legacy-bridge` 직결은 200 이므로 게이트웨이 라우트 쪽 사안이며, DDL 은 HTTP 라우팅에 영향을 줄 수 없다. P0/P6 에서 확인할 것.

---

## 3. 데이터 시드 — **하지 않았다**

지시서는 "빈 표는 조회 응답이 구와 달라져 불일치 노이즈가 된다"는 이유로 접촉 테이블 시드를 요구했다. 실측 결과 **이 전제가 이번 대상에는 성립하지 않는다.**

1. **신규 PG 는 이미 운영 데이터를 갖고 있다.** 접촉 50개 중 44개가 실데이터와 함께 존재한다 — `post` 9,852,411 / `store` 79,584 / `boost_message` 31,001 / `boost_customer` 18,508 / `store_channel` 218,810 / `post_word` 19,730,591 등. "빈 표" 문제 자체가 없다.
2. **새로 만든 3개는 조회 응답에 실리지 않는다.** `crawling_fail_log` 는 append-only 실패 원장이라 읽는 엔드포인트가 없다(개발계 PG 에서도 0행). `boost_review_request_setting` · `boost_message_delivery` 를 읽는 경로는 `/internal/**` 과 `/api/v2` 뿐인데, **골든셋 35파일에 그 경로 트래픽이 없다.**
3. **개발계 PG 에서 복사하면 오히려 오염된다.** 리플레이의 old 기준은 `deploy/config.yaml` 상 **운영 RDS** 다. 개발계 `boost_review_request_setting` 3행 · `boost_message_delivery` 8,592행은 개발 스크래치 데이터이고, 참조하는 `store_id`/`message_id` 가 운영 계보 신규 PG 의 행과 맞지 않는다. 넣는 순간 판정 기준이 깨진다.

**개인정보·자격정보 확인**: 시드하지 않기로 했지만 후보 테이블의 내용은 확인했다. `boost_message_delivery` 에 `hospital_display_name`(병원 표시명) · `provider_message_id` 가 있고 수신자 전화번호 컬럼은 없다. `boost_review_request_setting` 은 설정값만 담는다. 어느 쪽도 복사하지 않았으므로 개인정보 이동 0건이다.

**골든셋 실측** — 35파일 118,268 레코드 중 `/api/v1` 은 **21개 경로**(OPTIONS 포함 42조합)뿐이다. 상위: `boost1/.../messages/{id}/posts`(34) · `post-reply-ai-recommend`(24) · `boost1/.../message-templates`(24) · `users/me/stores`(19) · `organizations/{id}/teams`(16) · `stores/{id}/posts/statistics/period-count`(12) · `boost1/.../messages`(10) · `post-word/statistics`(9) · `team/presentHistories/{id}`(7). 이 21개가 읽는 테이블은 전부 운영 데이터가 채워져 있다.

리플레이 판정 결과 시드가 필요해지면 **운영에서 캡처 시점 기준으로 뽑아야 한다** — 그때는 승인을 받겠다.

---

## 4. 만들지 않은 것

### 4.1 접촉하지만 만들지 않은 것 — `app_user_channel_account` (판단 필요)

`/api/v1` **5개 경로**가 접촉한다:

| 경로 | 메서드 | 위치 |
|---|---|---|
| `/api/v1/app-user-store/match` | PATCH | `AppUserStoreController.kt:56` |
| `/api/v1/app-user-channel-account/list` | GET | `AppUserChannelAccountController.kt:31` |
| `/api/v1/app-user-channel-account/login` | POST | `:45` |
| `/api/v1/app-user-channel-account/login/kakao/sms` | POST | `:67` |
| `/api/v1/app-user-channel-account/relogin` | POST | `:90` |

그런데 이 테이블은 **양쪽 DB 어디에도 없다.**

```
docker exec postgresql psql -U legalcare -d medilawyer -Atc "select count(*) from app_user_channel_account"
ERROR:  relation "app_user_channel_account" does not exist
```

구 앱의 `ddl-auto` 는 `none` 이다(`core-storage.yml:14` 외 6곳) — Hibernate 가 만들어 주지도, 부팅 시 검증에 실패하지도 않는다. 즉 **구 legacy-service 는 지금 이 경로들에서 런타임 에러를 내고 있다.**

지금 신규 PG 에만 이 테이블을 만들면 **신이 구보다 잘 도는 상태**가 된다(구 500 → 신 200). 리플레이 판정 기준("200 나와야 할 게 다른 에러면 불합격, 원래 나던 에러는 그대로면 통과")에서 이는 동등 위반이다. 그래서 만들지 않았다.

**뒤집힐 조건**: 운영 DB 에 이 테이블이 실재한다면 판단이 바뀐다. 신규 PG 가 운영 전수 복사본이라는 점에서 *없었다*고 보는 것이 합리적이지만, 확정하려면 운영 조회가 필요하고 **지시대로 하지 않았다.** 4.3 미판정 ①.

`AppUserChannelAccountController` 클래스에는 `@Deprecated("NEVER USED")` 가 붙어 있다. 다만 **`PATCH /api/v1/app-user-store/match` 는 별개 컨트롤러 경로인데도 이 테이블을 쓴다** — 이식 때 놓치기 쉬운 지점이라 P2(R4)에 전달한다.

### 4.2 접촉 자체가 없어 만들지 않은 것

| 테이블 | 근거 |
|---|---|
| `admin` | `db/domain/admin/Admin.kt` 를 import 하는 파일이 저장소 전체에 **0건**. `AdminRepository` 조차 없다. 엔티티만 남은 사문(死文) |
| `post_reply_history` | 엔티티·`Repository`·`RepositoryCustom`(선언 비어 있음)·`RepositoryImpl` 4파일이 있으나 주입하는 서비스가 0건. `Post`/`PostReply` 에 역방향 매핑도 없다. 답글 등록 이력은 실제로 **`post_reply_registration_request`** 에 남는다(신규 PG 에 4,741행 존재) |

두 테이블 모두 개발계·신규 PG 양쪽에 없고, 구 앱이 지금 이 상태로 정상 기동 중이다.

### 4.3 개발계에만 있는 45테이블 — 전부 범위 밖

개발계 PG 에만 있는 62개 중, 접촉분 17개(`crawling_fail_log` 계열 15 + boost 2)를 뺀 **45개는 구 소스에 문자열로도 등장하지 않는다.** 엔티티 `@Table` 검색과 확장자 무제한 전체 grep 양쪽 모두 0건이다.

| 묶음 | 테이블 | 수 | 범위 밖 근거 |
|---|---|---|---|
| 알림톡 발송이력 | `kakao_alimtalk_send_history` + 파티션 28 | 29 | grep 0건. 구 `BizMessageService` 는 Repository 미주입, 외부 API 전용 |
| 리뷰부스터 v2 잔여 | `batch_alimtalk_send_log` · `boost_present_send_request` · `boost_review_match_analysis` · `boost_review_platform_click` · `boost_store_present_setting` | 5 | grep 0건. 신규 booster 의 v2 이관분 소유이며 구 앱은 안 씀 |
| GEO | `geo_analysis_request` · `geo_collection_status` · `geo_hospital_source` · `geo_sov` · `geo_wup_collection` | 5 | grep 0건 |
| 병원 공식 | `hospital_official` · `_category` · `_category_taxonomy` | 3 | grep 0건 |
| 기타 | `crawling_daily_stats` · `kakao_alimtalk_blocklist` · `stores_updated_1` | 3 | grep 0건 |

이 45개는 **구 legacy-service 이식과 무관**하다. 신규 booster 의 v2 이관분(`kakao_alimtalk_*`, boost v2 4종)이 필요로 하는 것은 별건이며, 그쪽 DDL(`db/reviewboost-v2-alimtalk.sql`, `db/reviewboost-v2-product.sql`)이 이미 저장소에 있으나 **신규 PG 에 아직 적용되지 않았다.** P-1 은 "접촉분만" 규칙에 따라 적용하지 않았다 — 필요 여부는 해당 태스크가 판단할 몫이다.

---

## 5. 미판정

**① `app_user_channel_account` 의 운영 실재 여부 — 미판정, 승인 필요**
신규 PG 가 운영 전수 복사본이라는 근거(`copy.go:155` 전수 열거)로 "운영에도 없다"고 추정했으나 확정하지 못했다. 운영 DB 읽기가 금지되어 있어 **추측으로 채우지 않았다.** 있는 것으로 밝혀지면 4.1 판단이 뒤집히고 `/api/v1` 5경로의 DDL 이 추가로 필요하다.

**② 신규 3테이블의 운영 현재 구조 일치 여부 — 미판정**
2026-07-14 시점 운영에 없던 테이블들이라 그 이후 운영에 생겼다면 구조가 다를 수 있다. 근거로 삼은 것은 저장소 정본 DDL 과 구 엔티티이지 운영 조회가 아니다. 운영 배포 시점에는 재대조가 필요하다.

**③ 지연로딩 프록시의 SELECT 발생 여부 — 4곳 미판정 (DDL 영향 없음)**
`/api/v1/auth/login` · `/api/v1/jwt/create` · `/api/v1/jwt/refresh` 에서 `organizationAppUser.organization.identifier` 접근이 프록시 초기화를 유발하는지, `/api/v1/organizations/{id}/users/{id}/role` 등에서 `.identifier`/`.id` 역참조가 SELECT 를 내는지는 Hibernate 접근전략에 달려 정적으로 확정 불가하다. TSV 에 `organization(미판정)` 으로 표기했다. **해당 테이블은 신규 PG 에 이미 있으므로 DDL 판정에는 영향이 없다.**

**④ `boost_customer` 조인 여부 1건 — 미판정 (DDL 영향 없음)**
`/api/v1/team/presentHistories/{teamId}` 의 QueryDSL 프로젝션에 `message.customer.identifier` 가 들어가는데, Hibernate 가 FK 컬럼으로 최적화해 조인을 생략하는지 SQL 로그로 확인하지 않았다. 보수적으로 접촉 테이블에서 제외했다. 테이블은 신규 PG 에 18,508행으로 존재한다.

**⑤ `/api/v1/stores/{storeId}/posts/statistics` 의 `team` 조인 — 런타임 파라미터 의존**
`teamId` + (`productPostStatus` 또는 `hasProduct`) 가 함께 올 때만 조인이 붙는다(`QuerydslFilter.kt:121,127,148,149`). `team(조건부)` 로 표기. 테이블 존재하므로 DDL 영향 없음.

**⑥ 정적탐색의 한계**
AC11 은 "정적탐색 + 리플레이 로그 `relation does not exist` 0건"의 이중 확인을 요구한다. **P-1 단계에서는 전자만 끝났다.** 후자는 P1~P4 리플레이 실행 시 확인해야 하며, 그때 새 테이블이 튀어나오면 이 문서를 갱신해야 한다. 현재 구 legacy-service 로그(최근 72시간)에는 `relation does not exist` 가 0건이다.

---

## 6. 부수 발견 (P2~P4 에 전달)

1. **`GET /api/v1/stores/{storeId}/channels`(`StoreController.kt:163`)는 `storeId` PathVariable 을 쓰지 않는다.** `userDetails.appUser` 로만 조회해 `/api/v1/app-user-store/list` 와 완전히 같은 결과를 낸다. 버그로 보이나 **1:1 이식 원칙상 그대로 옮겨야 한다.**
2. **`PostService.deletePostListAndRelatedObjects`(`PostService.kt:78-84`)가 `post_reply` 는 지우고 `post_reply_child` 는 안 지운다.** `PATCH /api/v1/admin/store-channel-match` 에서 고아 행이 남는다. 역시 이식 대상 그대로.
3. **`@KafkaListener` 는 저장소 전체에 1개**(`KafkaConsumer.kt:82`). 20개 토픽을 구독하지만 실제 분기는 6개뿐이고 나머지 14개는 no-op 이다.
4. **`@Scheduled` 3건 모두 `serviceMode ∉ {prod, dev}` 이면 조기 종료**한다(`ScheduledService.kt:73,212,304`). R10 의 "기본 OFF 플래그" 설계 시 이 게이트를 참고할 것.

---

## 7. AC11 자가 점검

| 항목 | 결과 |
|---|---|
| 접촉 매핑표 공란 0 | ✅ TSV 150행(헤더+130 핸들러+19 접두), 접촉 테이블 칸 공란 0. DB 무접촉은 `(DB 무접촉)` 명시 |
| 접촉 테이블이 신규 PG 에 존재 | ✅ 접촉 50개 중 44개 기존 + 3개 생성 = 47개 존재. 미생성 3개는 근거와 함께 4절에 기록 |
| DDL 재실행 멱등 | ✅ 두 파일 각각 2회 실행, 양쪽 `exit=0` / ERROR 0건 / NOTICE skip 으로 확인 |
| 미접촉 목록 + 범위 밖 근거 | ✅ 4.3 에 45개를 5묶음으로, grep 0건 근거와 함께 기록 |
| 기존 94테이블 무손상 | ✅ `information_schema.columns` 덤프 md5 전후 동일(`837fa4a0…`) |
| 신규 스택 정상 | ✅ 컨테이너 28개 Up, booster 로그 DB 에러 0건, `:10000/actuator/health` 200 |
| 정적탐색 + 런타임 교차확인 | ⚠️ 정적탐색만 완료. `relation does not exist` 0건 확인은 P1~P4 리플레이에서 수행 |

## 8. 준수 확인

`medilawyer-boot` 무수정(`git status` 클린, 읽기만) · 구 PG 는 `pg_dump -s`/SELECT 만(적용 후 152테이블 그대로) · 신규 PG 기존 테이블 구조 무변경 · 컨테이너 정지·삭제·compose 재생성 없음 · **운영 DB 접근 0건(읽기 포함)** · booster 코드 무수정 · 커밋 없음.

변경된 파일은 하나다: `legalcare-renew-prodsync-wt/apps/booster-app/app/src/main/resources/db/legacy-absorption-p1-schema.sql` (신규, 미커밋).

---
---

# 완결분 — 운영 대비 부족 42테이블 전량 생성 (P-1b)

기준이 **운영(`medilawyer-prod`)으로 확정**된 뒤 이어서 한 작업이다. 위 1~8절은 당시 기록으로 그대로 두고,
그때의 판단이 뒤집힌 곳은 이 절에서 명시적으로 정정한다.

DDL 산출물: `legalcare-renew-prodsync-wt/apps/booster-app/app/src/main/resources/db/legacy-absorption-p1b-schema.sql` (신규, 미커밋)
작업 로그·중간 산물: DELL `/mnt/ex_disk1/p1b-schema-sync/`

## 9. 기준 재확인 — 계수 방식을 하나로 통일했다

먼저 세 DB 를 **같은 쿼리**로 세었다. 이전 기록들이 서로 다른 계수법을 섞어 쓴 탓에 숫자가 어긋났기 때문이다.

```sql
SELECT c.relname, c.relkind::text FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
WHERE n.nspname='public' AND c.relkind IN ('r','p') ORDER BY 1;
```

| 계수법 | 운영 | 신규(작업 전) | 개발계 |
|---|---|---|---|
| `pg_class` relkind r+p | **144** | **102** | 152 |
| `pg_class` relkind r만 | 142 | 101 | 150 |
| `information_schema.tables` BASE TABLE | 144 | 102 | 152 |
| `information_schema.tables` 전체 | 145 | 103 | 153 |

**마지막 행이 +1 인 이유는 각 DB 에 뷰가 1개씩 있기 때문이다.** 착수 지시서의 "개발계 153", 내가 P-1 에서 쓴
"신규 94→103" 은 이 전체 계수(뷰 포함)이고, 오케스트레이터의 144/102 는 BASE TABLE 계수다. **숫자가 달랐던 게
아니라 세는 대상이 달랐다.** 이후 이 문서의 모든 테이블 수는 `pg_class relkind IN ('r','p')` 기준이다.

차집합도 직접 확인했다 — **운영에만 42개, 신규에만 0개**로 오케스트레이터 실측과 일치한다.

| 묶음 | 개수 |
|---|---|
| `kakao_alimtalk_send_history` 파티션 부모 1 + `_default` 1 + 월파티션 27 (`y2025m05`~`y2027m07`) | **29** |
| `crawling_fail_log_2027m01`~`_2027m07` (부모는 신규 PG 에 이미 있음) | 7 |
| 리뷰부스터 v2 4 (`boost_present_send_request`·`boost_review_match_analysis`·`boost_review_platform_click`·`boost_store_present_setting`) | 4 |
| `crawling_daily_stats` · `kakao_alimtalk_blocklist` | 2 |
| **합계** | **42** |

지시서의 묶음표는 알림톡을 27로 적었는데 **실제로는 29**다(부모·`_default` 포함 시 2+8+12+7). 총합 42 는 같다.

## 10. 스키마 원본 — 운영에서 구조만 뽑아 그대로 재현했다

`pg_dump -s` (구조 전용, `-a`·데이터 옵션 없음) 한 번으로 운영 public 스키마 13,113줄을 받았다.
`COPY`·`INSERT` 구문 **0건**을 확인해 데이터가 섞이지 않았음을 검증했다.

파일은 이 덤프에서 42개 관련 구문 **612개**를 뽑아 만들었다. 누락 검증은 반대 방향으로 했다 —
전체 덤프에서 타깃 테이블명 문자열을 포함하는데 선택되지 않은 구문이 **0건**임을 확인했다.
(첫 시도에서 `ALTER INDEX … ATTACH PARTITION` 203건을 통째로 놓쳤다. 인덱스명은 테이블명과 정확히 일치하지
않아 exact-match 필터에 걸리지 않았기 때문이다. 접두 매칭으로 바꾸고 이 역방향 검증을 추가해 잡았다.)

외부 의존도 확인했다: **타깃을 참조하는 FK 0건, 트리거·룰·정책 0건, 신규 PG 와의 객체명 충돌 0건**(테이블·인덱스·제약·시퀀스 352개 전수 대조).

**pg_dump 원문을 그대로 쓰지 않고 바꾼 곳이 두 군데 있다. 둘 다 "바꿔야 운영과 같아지는" 경우다.**

- **`crawling_fail_log` 2027 파티션은 `PARTITION OF` 로 붙였다.** 덤프 원문은 "빈 테이블 생성 → ATTACH →
  인덱스 개별 생성" 순인데, 신규 PG 의 부모는 이미 파티션 인덱스가 유효 상태라 ATTACH 시점에 PG 가 자식
  인덱스를 자동 생성해 뒤따르는 구문과 충돌한다. `PARTITION OF` 는 저장소 `crawling-fail-log.sql`·운영 배치가
  쓰는 방식이고, 같은 방식으로 만든 신규 PG 의 2026 파티션이 운영과 구조가 완전히 일치함을 먼저 확인한 뒤 택했다.
- **CHECK·부분인덱스의 값목록은 `IN (…)` 형으로 적었다.** pg_dump 가 역파싱한 형태
  `((status)::text = ANY ((ARRAY[…])::text[]))` 를 그대로 되먹이면 PG 가 배열 캐스트를 원소별로 풀어
  `ANY (ARRAY[(…)::text, …])` 로 굳는다. 판정 결과는 같지만 카탈로그 표현이 운영과 달라진다.
  같은 서버에서 두 형태를 만들어 `pg_get_constraintdef` 를 비교해 **`IN` 형만 운영과 일치**함을 실증하고 60곳을 치환했다.

## 11. 적용과 검증 — 명령과 실제 출력

적용 전 **트랜잭션 드라이런**으로 먼저 확인했다(PG 는 DDL 이 트랜잭션 대상이다).

```
BEGIN; \i legacy-absorption-p1b-schema.sql
  DRYRUN 테이블수 = 144
ROLLBACK;
  ROLLBACK 후 테이블수 = 102        ← 흔적 0
```

이후 본 적용도 단일 트랜잭션(`BEGIN … COMMIT`, `lock_timeout=15s`)으로 했다 — 실패 시 부분 적용이 남지 않도록.

| # | 검증 | 명령 | 결과 |
|---|---|---|---|
| ① | 테이블명 집합 | `comm -23/-13` 양방향 | 운영 144 = 신규 144, **양방향 차집합 0**. `relkind` 까지 포함해 완전 일치 |
| ② | **구조 전수 대조** | 컬럼(타입·nullable·기본값·identity)·제약·인덱스·파티션경계·코멘트를 정규화해 diff | **42개 전부 차이 0**. P-1 이 만든 8개도 차이 0 (12절에서 고침) |
| ③ | DDL 멱등 | 전체 스크립트 3회 실행 | `exit=0`, **ERROR 0건**, WARNING 0건, NOTICE(skip) 284건 |
| ④ | 기존 테이블 무손상 | 컬럼·제약·인덱스·상속 1,143행 덤프 md5 전후 비교 | `233444ef…` **전후 동일** |
| ⑤ | 파티션 라우팅 | INSERT → `tableoid` 확인 → ROLLBACK | 아래 표. **잔여행 0** |
| ⑥ | 제약 실동작 | 위반 INSERT 6종 | 5종 ERROR·1종 통과(의도) — 아래 |
| ⑦ | 스택 무손상 | `docker ps` 전후, booster 로그, 헬스 | 91개 전후 동일·소실 0·신규 0·재시작 0, DB 에러 **0건**, `:10000/actuator/health` 200 |
| ⑧ | **운영 무변경** | `pg_dump -s` 재취득 후 diff | 테이블수 144 동일. 덤프 **실질 diff 0줄** (차이는 pg_dump 가 매 실행 새로 만드는 `\restrict` 난수 토큰 2줄뿐) |
| ⑨ | 개발계 무변경 | 같은 계수 쿼리 | 152 그대로. 읽기만 함 |

**⑤ 파티션 라우팅 실측** — 타임존 표기 동치도 함께 증명했다(`'2027-01-01 00:00:00+09'::timestamptz = '2026-12-31 15:00:00+00'::timestamptz` → `t`, 서버 TimeZone=`Etc/UTC`).

| 넣은 값 | 착지 파티션 | 의미 |
|---|---|---|
| `occurred_at` 기본값(now) | `crawling_fail_log_2026m08` | 기본값 경로 정상 |
| `2026-12-31 23:59:59+09` | `_2026m12` | KST 월경계 직전 |
| `2027-01-01 00:00:00+09` | `_2027m01` | **경계가 정확히 KST 월초** |
| `2027-03-15 12:00+09` | `_2027m03` | 신규 파티션 정상 |
| `2027-09-01`·`2020-01-01` | `_default` | 범위 밖 수용 |
| kakao `created_at 2025-05-15+09` | `…_y2025m05` | |
| kakao `created_at 2026-08-09+09` | `…_y2026m08` | |
| kakao `created_at 2030-01-01+09` | `…_default` | 범위 밖 수용 |

**⑥ 제약 실동작** — 잘못된 status·잘못된 scope_type·같은 달 `idempotency_key` 중복·
`provider_message_id` 중복·`DELIVERED` 인데 `completed_at` NULL → **전부 ERROR**.
다른 달의 같은 `idempotency_key` 는 **통과**했는데, 이는 운영 구조를 그대로 따른 결과다(다음 절).

## 12. 알아둘 것 — 운영 구조에서 그대로 승계한 두 가지

- **`kakao_alimtalk_send_history` 의 `idempotency_key` 유일성은 "월 단위"다.** 운영은 파티션 부모가 아니라
  각 월파티션에 개별 UNIQUE 인덱스(`…_uq_idempotency`)를 걸고 있다. 파티션 키(`created_at`)를 포함하지 않는
  전역 UNIQUE 를 PG 가 허용하지 않기 때문이다. 즉 **월이 바뀌면 같은 멱등키가 다시 들어간다.** 운영에 있는
  성질이므로 그대로 재현했고, 판정 기준상 이게 맞다. 다만 알림톡 이관 담당이 알아야 할 사실이라 남긴다.
- **`boost_message_delivery` 에 `idempotency_key` UNIQUE 가 이름만 다른 채 2개 있다**
  (`uq_boost_message_delivery_idempotency`, `…_idempotency_key`). 개명 흔적으로 보이나 운영 현행이라 둘 다 만들었다.

## 13. 저장소 DDL 대 운영 실구조 — 차이 전량

**저장소 DDL 은 정본이 아니었다.** 특히 알림톡은 구조 자체가 다르다.

| 저장소 파일 / 대상 | 저장소 | 운영(정본) |
|---|---|---|
| `reviewboost-v2-alimtalk.sql` §1 `kakao_alimtalk_send_history` | **일반 테이블** | **`PARTITION BY RANGE (created_at)` + 월파티션 27 + `_default`** |
| 〃 PK | `id` 단독 | **`pk_…` (id, created_at) 복합** (파티션 키 포함 필수) |
| 〃 `idempotency_key` UNIQUE | 테이블 전역 1개 | **파티션마다 1개**(유일성 범위가 월 단위 — 12절) |
| 〃 `created_at`·`updated_at` | 기본값 없음 | **`DEFAULT CURRENT_TIMESTAMP`** |
| 〃 인덱스 이름 | `idx_kash_*` / `uq_kash_*` | **`ix_kakao_alimtalk_send_history_*`** |
| 〃 CHECK 이름 | `ck_kash_attempt_count` | **`ck_kakao_alimtalk_send_history_attempt_count`** (`ck_kash_status` 는 동일) |
| 〃 컬럼 코멘트 | 없음 | **5개** |
| `reviewboost-v2-alimtalk.sql` §2 `kakao_alimtalk_blocklist` | CHECK 없음, 인덱스 없음(주석으로 보류) | **CHECK 2종**(`scope_type`·`message_purpose`) + **부분 인덱스 2종**(`ix_…_active_global_phone`·`_active_store_phone`) |
| `reviewboost-v2-product.sql` `boost_review_match_analysis` | 인덱스 `idx_brma_status_id` | 운영엔 **`idx_brma_status` 도 함께** 존재 |
| `crawling-fail-log.sql` | 컬럼 코멘트 없음 | **`payload`·`status` 컬럼 코멘트 2개** |
| 〃 파티션 범위 | 2026m08~12 + DEFAULT | **2027m07 까지** |

컬럼 구성·타입·nullable 은 대체로 맞았다. 틀린 것은 **파티션 여부·PK 구성·유일성 범위·객체 이름·코멘트**다.
알림톡을 저장소 DDL 로 운영에 적용했다면 파티션 없는 테이블이 만들어져 **운영과 다른 DB 가 됐을 것이다.**
저장소 파일들은 이 사실을 반영해 갱신하는 게 맞지만, 이번 지시 범위 밖이라 **손대지 않았다**(P0 이후 판단 몫).

## 14. P-1 산출물 자체의 결함 — 이번에 고쳤다 (12→②의 "P-1 신규 8도 차이 0")

운영과 구조를 대조하니 **P-1 이 만든 3테이블에도 차이가 있었다.** P-1 은 운영 조회가 금지된 상태에서
저장소 DDL·구 엔티티를 근거로 만들었기 때문이다. 이번 파일 D절에서 운영에 맞췄다.

| 테이블 | 차이 | 심각도 |
|---|---|---|
| `boost_message_delivery` | **인덱스 3종 누락. 그중 `uq_boost_message_delivery_provider_message` 는 UNIQUE** — 운영은 막는 `(provider, provider_message_id)` 중복을 신규는 허용하고 있었다 | **정합성 구멍** |
| 〃 | 컬럼 순서 상이 — `hospital_display_name` 이 운영 16번째, P-1 6번째 | `SELECT *` 순서 차이 |
| 〃 | 컬럼 코멘트 11개 누락 | 경미 |
| `boost_review_request_setting` | 컬럼 코멘트 2개 누락 | 경미 |
| `crawling_fail_log` | 컬럼 코멘트 2개 누락 | 경미 |

코멘트는 `COMMENT ON` 으로 더했다. 컬럼 순서는 ALTER 로 고칠 수 없어 **`boost_message_delivery` 만 재생성**했다 —
행 0건·FK 0건·의존 객체 0건·시퀀스 미사용(`last_value=1, is_called=false`)을 먼저 확인하고 단일 트랜잭션에서
DROP→CREATE 했다. **P-1 이 몇 시간 전에 만든 빈 테이블이며, 운영 복사본 94테이블은 건드리지 않았다.**
D절 DROP 은 `"컬럼 순서가 운영과 다르고 AND 행이 0건"` 일 때만 실행되고, 행이 있으면 `RAISE WARNING` 만 남기고
아무것도 하지 않는다 — 재실행해도 안전하다.

## 15. 범위 밖에서 발견한 것 — 기존 94테이블이 운영과 어긋나 있다 (미조치·승인 필요)

**42개를 다 만든 뒤에도 운영과 신규가 같아지지 않는다.** 테이블 *이름* 집합은 이제 144=144 로 같지만,
**구조**를 대조하니 2026-07-14 복사본인 기존 94개 중 **13개가 운영과 다르다.** 지금까지의 "부족 42개" 논의는
이름 집합만 비교한 것이어서 이 층을 통째로 못 봤다.

지시가 "기존 테이블 ALTER·DROP 금지"이므로 **하나도 손대지 않았다.** 아래는 실측 결과와 조치안이다.

**(가) 순번만 다름 — 기능 영향 없음, 조치 불필요 (8개)**
`boost_dental_patient_info` · `post` · `post_reply_registration_request` · `product_post` ·
`store_channel_attribute` · `team` · `team_app_user` · `team_auth` · `team_channel_account`
→ 운영에 삭제된 컬럼(`attisdropped`)이 18개 있어 `attnum` 에 구멍이 나 있고, 복사본은 새로 만들어져 구멍이 없다.
살아 있는 컬럼의 이름·타입·순서는 같다.

**(나) 실질 차이 — 리플레이 판정을 오염시킨다 (5개)**

| 테이블 | 차이 | 왜 위험한가 |
|---|---|---|
| `boost_post_similarity` | `post_id`·`enhanced_post_id` 가 신규 **varchar(50)**, 운영 **varchar(255)** | **신규 PG 의 `post` 중 id 가 50자를 넘는 행이 175,256건**(최대 79자)이다. 그 글에 대한 INSERT 는 운영에선 되고 신규에선 `value too long` 으로 실패한다 → 리플레이 불합격이 코드 탓으로 오판된다 |
| `similarity_score_analysis_data` | `target_post_id`·`compare_target_post_id` 동일 문제 (50 vs 255) | 〃 (현재 159,792행) |
| `boost_present_order` | 컬럼 **2개 누락**(`source varchar(10)`, `created_by bigint`) + 코멘트 2 + 인덱스 2(**`UNIQUE (uuid)` 포함**) | 해당 컬럼을 쓰는 쿼리는 `column does not exist` 로 즉사. UNIQUE 누락은 운영이 막는 중복을 신규가 허용 |
| `boost_message` | 인덱스 `(team_id, usable, id)` 누락 | 성능만 |
| `post` | 인덱스 `(created_at)`·`(updated_at)` 누락 | 성능만 (900만행 테이블) |

**조치안(미적용 — 승인 필요).** 앞 3건은 정합성에 직접 걸리므로 리플레이 전에 처리해야 한다.

```sql
-- (1) 컬럼 폭 확장 — 확장 방향이라 데이터 손실 없음. ACCESS EXCLUSIVE 락, 테이블 재작성 없음
ALTER TABLE public.boost_post_similarity          ALTER COLUMN post_id                TYPE varchar(255);
ALTER TABLE public.boost_post_similarity          ALTER COLUMN enhanced_post_id       TYPE varchar(255);
ALTER TABLE public.similarity_score_analysis_data ALTER COLUMN target_post_id         TYPE varchar(255);
ALTER TABLE public.similarity_score_analysis_data ALTER COLUMN compare_target_post_id TYPE varchar(255);

-- (2) 누락 컬럼 — 둘 다 nullable 이라 기존 698행에 영향 없음
ALTER TABLE public.boost_present_order ADD COLUMN IF NOT EXISTS source     varchar(10);
ALTER TABLE public.boost_present_order ADD COLUMN IF NOT EXISTS created_by bigint;
COMMENT ON COLUMN public.boost_present_order.source     IS '선물 발송 방식: MANUAL(수동), AUTO(자동) — 기존 주문은 NULL';
COMMENT ON COLUMN public.boost_present_order.created_by IS '수동 선물 발송을 실행한 사용자 ID — 자동·기존 주문은 NULL';

-- (3) 누락 인덱스 — uuid 중복 0건 확인 완료라 UNIQUE 생성 가능
CREATE UNIQUE INDEX IF NOT EXISTS uq_boost_present_order_uuid ON public.boost_present_order (uuid);
CREATE INDEX IF NOT EXISTS idx_boost_present_order_message ON public.boost_present_order (message_id, result, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_boost_message_team          ON public.boost_message (team_id, usable, id);
CREATE INDEX IF NOT EXISTS idx_post_created_at             ON public.post (created_at);   -- 900만행, CONCURRENTLY 권장
CREATE INDEX IF NOT EXISTS idx_post_updated_at             ON public.post (updated_at);   -- 〃
```
인덱스 이름은 운영 실제 이름을 아직 못 맞췄다(운영은 자동생성명·중복명이 섞여 있다). 승인 시 운영 이름으로 다시 뽑겠다.

## 16. 이전 보고 오류 추적 결과 — 근거 없는 수치였다

P-1 3절에 쓴 "개발계 `postgresql` 은 store 0·app_user 2·post 18 인 빈 스크래치 DB"는 **틀렸다.**
실측은 오케스트레이터 값과 같은 **store 64,644 · app_user 786 · post 2,119,442** 다.

어떤 쿼리에서 나온 값인지 추적하려고, 실행 중인 **모든 PG 컨테이너의 모든 데이터베이스**를 훑어
`public.store`/`app_user`/`post` 를 세는 스크립트를 돌렸다. 세 테이블이 존재하는 DB 는 딱 둘이다.

```
deploy-postgres-new-1 / medilawyer : store=79584  app_user=858 post=9964949
postgresql            / medilawyer : store=64644  app_user=786 post=2119442
```

`0/2/18` 을 내는 DB 는 이 환경에 없다. `legalcare`·`medicontents`·`postgres`·`legalcare-local-pg-1` 어디에도
그 테이블이 없어 오류조차 나지 않는다. **즉 그 수치는 어떤 쿼리 결과도 아니다 — 재는 대신 단정한 값이다.**
결론(판정 기준을 소스 기반으로 바꾼 것)은 맞았지만 근거로 내세운 숫자가 거짓이었다.

같은 착오가 이번 작업에 남아 있는지 점검한 결과, **성격이 다른 계수 오류를 하나 더 찾아 고쳤다** — 9절의
`information_schema.tables` 전체 계수(뷰 포함, +1) 대 BASE TABLE 계수 혼용이다. "94→103" 은 뷰를 포함한 값이라
BASE TABLE 기준 102 와 어긋났다.

재발 방지로 이번 절의 **모든 수치는 실행한 명령과 원문 출력을 남겼고**, 세 DB 를 **같은 스크립트 하나로** 셌다.
DB 식별도 다시 확인했다 — `docker ps` 상 호스트 `0.0.0.0:5432` 를 퍼블리시하는 컨테이너는 `postgresql` 하나뿐이라
"구 legacy-service 의 `127.0.0.1:5432` = `postgresql` 컨테이너" 라는 P-1 의 식별 자체는 옳았다.

## 17. 미판정 갱신

| P-1 미판정 | 현재 |
|---|---|
| ① `app_user_channel_account` 운영 실재 여부 | **해소** — 운영에 없다(오케스트레이터 확인). 만들지 않은 판단 유지 |
| ② 신규 3테이블의 운영 구조 일치 여부 | **해소** — 대조 결과 차이가 있었고 14절에서 전부 맞췄다. 현재 차이 0 |
| ③ 지연로딩 프록시 SELECT 4곳 | 미판정 유지. 해당 테이블은 존재하므로 DDL 영향 없음 |
| ④ `boost_customer` 조인 1건 | 미판정 유지. DDL 영향 없음 |
| ⑤ `/stores/{id}/posts/statistics` 의 `team` 조건부 조인 | 런타임 의존. DDL 영향 없음 |
| ⑥ 런타임 `relation does not exist` 0건 확인 | **미완** — P1~P4 리플레이에서 확인해야 한다 |
| **⑦ (신규) 기존 94테이블 중 5개의 운영 대비 실질 드리프트** | **미조치·승인 필요** (15절). 리플레이 판정을 오염시킨다 |
| **⑧ (신규) 인덱스 이름 대조** | 15절 조치안의 인덱스 이름은 임시명이다. 승인 시 운영 실제 이름으로 재취득 필요 |

## 18. 준수 확인

**운영**: `pg_dump -s` 2회 + SELECT 만. 세션에 `default_transaction_read_only=on` 을 걸어 실수로도 못 쓰게 했다.
작업 전후 스키마 덤프 실질 diff **0줄**, 테이블 수 144 동일 — **쓰기·DDL·데이터 이동 0건**.
**데이터 반출 0건** (덤프에 `COPY`/`INSERT` 0건 확인, 운영 행 데이터를 신규 PG 로 옮긴 적 없음).
**신규 PG**: 42개 신규 생성 + P-1 이 만든 빈 테이블 1개 재생성(14절) + 코멘트 15개. **운영 복사본 94테이블은 무손상**(md5 전후 동일).
**개발계 `postgresql`**: 읽기만, 152 그대로.
**컨테이너**: 91개 전후 동일, 정지·삭제·compose 재생성 **0건**, 재시작 흔적 0.
`medilawyer-boot` 무수정(`git status` 클린, `930c83cf`). booster 코드 무수정. **커밋 0건.**

저장소 변경은 미추적 신규 파일 2개뿐이다:
`db/legacy-absorption-p1-schema.sql`(P-1) · `db/legacy-absorption-p1b-schema.sql`(이번).

---

## 19. 기존 테이블 정합 (P-1c) — 15절 조치안을 승인받아 적용했다

DDL 산출물: `legalcare-renew-prodsync-wt/apps/booster-app/app/src/main/resources/db/legacy-absorption-p1c-existing-align.sql` (신규, 미커밋)
작업 로그: DELL `/mnt/ex_disk1/p0-work/` · 적용 2026-08-09 15:08~15:17 UTC

### 19.1 먼저 15절을 다시 쟀다 — 조치안 두 곳이 틀렸었다

15절은 인덱스 이름을 임시로 지어 뒀고(미판정 ⑧), 나는 그걸 쓰지 않고 **운영 `pg_class` 에서 실제 이름을 뽑았다.**
네 개가 달랐다. 임시명으로 만들었으면 이름만 다른 인덱스가 영구히 남았을 것이다.

| 15절 임시명 | 운영 실제 이름 |
|---|---|
| `uq_boost_present_order_uuid` | **`boost_present_order_uuid_uk`** |
| `idx_boost_present_order_message` | **`idx_boost_present_order_message_result_created_at`** |
| `idx_boost_message_team` | **`idx_boost_message_team_usable_id`** |
| `idx_post_created_at` · `idx_post_updated_at` | 동일 (맞았다) |

그리고 **운영의 `idx_boost_message_team_usable_id` 는 `indisvalid=false`** 다 — CREATE INDEX CONCURRENTLY
가 중간에 실패해 남은 껍데기다. 플래너가 쓰지 않으므로 **운영은 이 인덱스가 있다고 믿지만 실제로는 없는 상태**다.
15절은 인덱스 *정의*만 봐서 이 층을 못 봤다. 시그니처에 `indisvalid`/`indisready` 를 추가해 다시 쟀다.

### 19.2 구조 전수 대조 — 이름 집합이 아니라 구조로

`struct_sig2.sql`(작업 디렉토리)로 **public 스키마 테이블·파티션 전량**의 구조를 한 줄씩 정규화해 뽑고 정렬 후 diff 했다.
포함 항목: 컬럼(조밀순번·이름·`format_type`(=타입+길이)·notnull·기본값·identity·generated·collation) ·
제약(이름+`pg_get_constraintdef`) · 인덱스(이름+정의+`indisvalid`+`indisready`) ·
파티션(relkind·전략·**경계**·부모·persistence·reloptions) · 테이블/컬럼 코멘트.

`attisdropped` 로 생긴 `attnum` 구멍은 **조밀 순번**으로 메웠다. 15절 (가)의 8개는 살아 있는 컬럼의
이름·타입·순서가 같고 삭제 흔적만 다른 것이라, 원시 `attnum` 으로 비교하면 의미 없는 차이가 8건 뜬다.

| | 조치 전 | 조치 후 |
|---|---|---|
| 시그니처 줄수 (운영 / 신규) | 4,903 / 4,894 | **4,903 / 4,903** |
| 테이블 수 (운영 / 신규) | 144 / 144, 양방향 차집합 0 | 동일 |
| **전체 diff** | **36줄** | **12줄 (= 차이 3종)** |
| **대상 5개 테이블의 구조 차이** | 컬럼2·컬럼2·컬럼2+코멘트2+인덱스2·인덱스1·인덱스2 | **0** |

**남은 차이 3종 — 전부 "내가 고칠 수 없거나 고쳐선 안 되는" 것이고, 정의는 일치한다.**

| # | 차이 | 왜 남겼나 |
|---|---|---|
| 1 | `boost_message.idx_boost_message_team_usable_id` — 운영 `valid=false`, 신규 `valid=true` | 이름·정의·컬럼 전부 동일하고 **유효성 플래그만** 다르다. INVALID 인덱스는 일부러 만들 수 없고 만들어서도 안 된다. 차이의 방향이 **신규가 운영보다 정상**이며, 질의 *결과*는 달라지지 않는다(플래너 선택지만 는다). 운영 쪽 결함으로 보고한다 |
| 2·3 | `crawling_fail_log` PK 제약·인덱스 **이름** — 운영 `crawling_fail_log_pkey1`, 신규 `crawling_fail_log_pkey` | **승인 범위 5개 테이블 밖**이라 손대지 않았다. PK 컬럼 구성 `(id, occurred_at)` 은 동일. 운영 이름에 `1` 이 붙은 것은 생성 시 이름 충돌로 PG 가 자동 개명한 흔적이다. 두 저장소 전체에서 이 이름을 참조하는 코드 **0건**, `ON CONFLICT ON CONSTRAINT` 사용 **0건**을 확인해 기능 영향이 없음을 확인했다 |

**뷰·시퀀스도 함께 봤다** (시그니처는 테이블만 다루므로 별도 확인):
- `public` 유일한 뷰 `post_thread_union_view` → `pg_get_viewdef` **바이트 단위 동일**(diff 0). `pg_depend` 의 컬럼 참조 목록이
  운영 25개 / 신규 2개로 다른데, 정의가 같으므로 카탈로그 기록 방식 차이일 뿐 기능 차이가 아니다.
- 시퀀스 101개 파라미터 대조 → 차이 1건, `crawling_fail_log_id_seq1`(운영) vs `crawling_fail_log_id_seq`(신규).
  위 2·3 과 같은 뿌리이고 같은 이유로 범위 밖.

### 19.3 적용 — 명령과 실제 출력

A절(소형 테이블)은 **단일 트랜잭션**(`lock_timeout=10s`), B절(`post` 9,964,949행·6.1GB)은
**`CREATE INDEX CONCURRENTLY`** 로 나눴다. 일반 `CREATE INDEX` 는 SHARE 락으로 `post` 쓰기를 빌드 내내 막는데,
크롤러가 계속 `post` 에 쓰는 개발계를 아침까지 살려 둬야 하므로 쓰기를 막지 않는 쪽을 택했다.
CONCURRENTLY 는 트랜잭션 블록 안에서 실행할 수 없어 절을 분리한 것이다.

```
[드라이런] A절 BEGIN … ROLLBACK  → exit=0, 잔여 컬럼 0·잔여 인덱스 0·post_id 타입 varchar(50) 그대로
[A절 본적용]  real 0m3.408s, exit=0
   NOTICE: [P1C] widened boost_post_similarity.post_id -> varchar(255)
   NOTICE: [P1C] widened boost_post_similarity.enhanced_post_id -> varchar(255)
   NOTICE: [P1C] widened similarity_score_analysis_data.target_post_id -> varchar(255)
   NOTICE: [P1C] widened similarity_score_analysis_data.compare_target_post_id -> varchar(255)
[B절] CREATE INDEX CONCURRENTLY ×2 — 약 9분 (688,072 블록 스캔 ×2)
   출력 0바이트(에러 없음) · 종료 후 pg_stat_progress_create_index 비어 있음
   idx_post_created_at:true , idx_post_updated_at:true  ← 둘 다 valid
   신규 PG 전체 INVALID 인덱스 = 0
```

### 19.4 검증 — 요구된 5가지 전부

| 검증 | 명령/방법 | 결과 |
|---|---|---|
| ① **행수 불변** | 5개 테이블 `count(*)` 전후 | 4,866 / 159,792 / 698 / 31,001 / 9,964,949 → **전부 동일** |
| ② **테이블 재작성 없음** | `pg_relation_filenode` 전후 | 39553 / 39423 / 39557 → **동일**. varchar 확대가 실제로 메타데이터만 바꿨음을 실증 |
| ③ **멱등** | 전체 스크립트 **총 3회** 실행 | run#2·#3 **exit=0, ERROR 0건**. NOTICE(skip) 11건씩. 3회 후 구조 시그니처 md5 `3d78c2db…` **동일** |
| ④ **스택 무손상** | `docker ps` · `RestartCount` · 앱 로그 · 헬스 | 컨테이너 **91→91**, 소실 0·신규 0, **재시작 카운트 변화 0**, booster-web/worker·legacy-bridge·legacy-service **DB 에러 0건**(15:08Z 이후), 게이트웨이 `/actuator/health` 200, legacy `/api/v1/health` 200 |
| ⑤ **79자 post.id INSERT 실증** | 아래 | **성공** |

**⑤ 상세** — 실제 79자 값 `GOOGLE_MAP_Ci9DQUlRQUNvZENodHljRjlvT200d00wMHljalZVWkVGdFIyMVJOR2MxWjIxR1IwRRAB` 로 시험했다.

```
[B] varchar(50) 컬럼에 대입 → SQLSTATE=22001 value too long for type character varying(50)   ← 조치 전이었다면 이렇게 죽었다
[B] varchar(255) 로 넓힌 실제 컬럼에 대입 → 성공
[C] boost_post_similarity          INSERT 1건 → post_id_len=79 · enhanced_len=79, 행수 4866→4867
    similarity_score_analysis_data INSERT 1건 → target_len=79 · compare_len=79, 행수 159792→159793
    ROLLBACK
[D] 롤백 후 4866 / 159792  ← 원복 확인
```

처음에 `v::varchar(50)` 로 실패를 증명하려다 **그게 성공해 버렸다.** PG 의 명시적 CAST 는 초과분을 조용히 자르고
에러를 내지 않는다 — 실패는 *대입*에서만 난다. 잘못된 증명이라 임시 테이블 대입으로 다시 짰다. 위 [B] 가 그 결과다.

**추가 실동작 확인** (조치가 "있기만 한" 게 아니라 "먹히는지"):
- `boost_present_order_uuid_uk` — 기존 행을 통째로 복제해 같은 uuid 로 INSERT →
  `duplicate key value violates unique constraint "boost_present_order_uuid_uk"` **차단 확인**
- `source`/`created_by` — `MANUAL`/`12345` 로 INSERT 후 재조회 성공 → ROLLBACK, 행수 698 유지

### 19.5 준수 확인

**운영**: `pg_dump -s` 1회 + SELECT 만, 세션에 `default_transaction_read_only=on` 강제
(실제로 `CREATE TEMP TABLE` 시도가 `cannot execute CREATE TABLE in a read-only transaction` 으로 막히는 것을 확인).
**내 작업 전/후 운영 구조 시그니처 md5 `3c56728a…` 동일, diff 0줄** — 쓰기·DDL·데이터 이동 **0건**.
덤프에 `COPY`/`INSERT` **0건**(데이터 반출 0). 테이블 수 144 그대로.
(P-1b 덤프와의 텍스트 diff 1,857줄은 전량 `GRANT`/ACL 구문 — P-1b 가 `-O` 로 뽑아 소유권·권한을 뺐던 차이다.
운영에서 사라진 줄 `^<` 은 **0건**.)

**신규 PG**: 승인된 **5개 테이블만** 변경(컬럼폭 4 · 컬럼추가 2 · 코멘트 2 · 인덱스 5). 그 외 기존 테이블 ALTER/DROP **0건** —
전수 시그니처 diff 가 이를 뒷받침한다(5개 외 테이블의 차이는 조치 전과 동일한 `crawling_fail_log` 이름 2줄뿐이고 이건 원래부터 있던 것이다).
**개발계 `postgresql`**: 읽기만, 152 그대로.
**컨테이너**: 91개 전후 동일, 정지·삭제·compose 재생성 0건, 재시작 0.
`medilawyer-boot` 무수정. **커밋 0건.** 게이트웨이 라우트 무변경.

### 19.6 미판정 갱신

| 항목 | 현재 |
|---|---|
| ⑦ 기존 94테이블 중 5개 드리프트 | **해소** — 5개 전부 운영에 맞췄고 구조 차이 0 |
| ⑧ 인덱스 이름 대조 | **해소** — 운영 실제 이름으로 재취득해 적용(19.1) |
| **⑨ (신규) 운영 `idx_boost_message_team_usable_id` 가 INVALID** | **운영 측 결함으로 보고.** 운영은 이 인덱스를 못 쓰는 상태다. 이번 작업 범위 밖이며 운영 DDL 소유권 절차로 처리할 일 |
| **⑩ (신규) `crawling_fail_log` PK·시퀀스 이름 불일치** | **미조치.** 승인 범위 5개 밖. 코드 참조 0건이라 기능 영향 없음. 이름까지 맞추려면 `ALTER TABLE … RENAME CONSTRAINT` + `ALTER SEQUENCE … RENAME` 2줄이면 되나 **승인 없이는 하지 않는다** |
| ⑥ 런타임 `relation does not exist` 0건 | 미완 — P1~P4 리플레이에서 확인 |
