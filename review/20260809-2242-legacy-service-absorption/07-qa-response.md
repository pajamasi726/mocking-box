# P5 QA 결함 대응 — D2 수정 · D1 분석

worktree `/Users/steve/steve/legalcare-renew-prodsync-wt` (HEAD `4df03196` 위, **커밋 0**).
QA 리포트의 두 결함 중 **D2 는 고쳤고 D1 은 지시대로 분석만** 했다. 아래 수치는 전부 명령과 원문 출력이 붙어 있다.

| 결함 | 처리 | 근거 |
|---|---|---|
| D2 [CRITICAL] `post-reply-ai-recommend` 100% 500 | **반영** — 엔티티 20개 `@Id` 를 구와 같은 JVM 원시형으로 복귀 | 라이브 500→200 · 골든 24/24 · 회귀 0 |
| D1 [MAJOR] pre-handler 에러 봉투 이탈 | **분석만**(코드 무수정) | §D1 |

---

## D2 — 무엇이 틀렸고 무엇을 고쳤나

QA 의 기전 분석이 맞다. 다만 **개수가 다르다.**

- QA: "같은 패턴이 legacy 엔티티 **16개**" — QA 의 grep(`private Long .* = 0`)이 **초기값이 있는 것만** 잡았다.
- 지시서: "`private Long id` 가 **21개 파일**" — 맞지만 그중 1개는 **구도 래퍼**다.
- **실측 = 20개.** 신 legacy 엔티티 **46개 전수**를 구 `module-storage` 와 javap 로 대조했다.
  `BoostReceiptPaymentsEntity` 만 구가 `val id: Long? = null`(nullable)이라 래퍼가 맞고, 나머지 25개는 문자열 식별자다.

대조 방법(선언이 아니라 컴파일 결과를 봤다):
```
구 : javap -p -cp medilawyer-boot/module-storage/core-storage/build/classes/kotlin/main <FQCN>
신 : javap -p -cp booster-app/modules/legacy/build/classes/java/main <FQCN>
검증: 구 module-storage 소스가 전부 빌드산출물보다 오래됐음을 확인(find -newer → 0건), 구 레포 git status clean
```

### 구/신 `@Id` JVM 타입 대조표 — 수정한 20개

| # | 엔티티 (테이블) | 구 선언 (`module-storage/.../db/domain`) | 구 javap | 신 javap (수정 전 → 후) |
|---|---|---|---|---|
| 1 | `AppUser` (`app_user`) | `val id: Long = 0` :20 | `long` | `java.lang.Long` → `long` |
| 2 | `AppUserChannelAccount` (`app_user_channel_account`) | `val id: Long = 0` :19 | `long` | `java.lang.Long` → `long` |
| 3 | `AppUserStore` (`app_user_store`) | `val id: Long = 0` :18 | `long` | `java.lang.Long` → `long` |
| 4 | `BoostMessageDelivery` (`boost_message_delivery`) | `val id: Long = 0L` :36 | `long` | `java.lang.Long` → `long` |
| 5 | `BoostPresent` (`boost_present`) | `val id: Long` :16 | `long` | `java.lang.Long` → `long` |
| 6 | `BoostPresentLedger` (`boost_present_ledger`) | `val id: Long` :16 | `long` | `java.lang.Long` → `long` |
| 7 | `BoostPresentOrder` (`boost_present_order`) | `val id: Long` :20 | `long` | `java.lang.Long` → `long` |
| 8 | `Otp` (`otp`) | `val id: Long = 0` :13 | `long` | `java.lang.Long` → `long` |
| 9 | `PostReply` (`post_reply`) | `val id: Long = 0` :18 | `long` | `java.lang.Long` → `long` |
| 10 | `PostReplyAiRecommend` (`post_reply_ai_recommend`) | `val id: Long = 0` :13 | `long` | `java.lang.Long` → `long` |
| 11 | `PostReplyChild` (`post_reply_child`) | `val id: Long = 0` :23 | `long` | `java.lang.Long` → `long` |
| 12 | `PostReplyRegistrationRequest` (`post_reply_registration_request`) | `val id: Long = 0` :17 | `long` | `java.lang.Long` → `long` |
| 13 | `PostWord` (`post_word`) | `val id: Long = 0` :15 | `long` | `java.lang.Long` → `long` |
| 14 | `Product` (`product`) | `val id: Long = 0` :13 | `long` | `java.lang.Long` → `long` |
| 15 | `ProductPost` (`product_post`) | `val id: Long = 0` :24 | `long` | `java.lang.Long` → `long` |
| 16 | `ReviewRequestSetting` (`boost_review_request_setting`) | `val id: Long = 0L` :21 | `long` | `java.lang.Long` → `long` |
| 17 | `Store` (`store`) | `val id: Long = 0` :19 | `long` | `java.lang.Long` → `long` |
| 18 | `StoreChannel` (`store_channel`) | `val id: Long = 0` :22 | `long` | `java.lang.Long` → `long` |
| 19 | `StoreChannelAttribute` (`store_channel_attribute`) | `val id: Long = 0` :16 | `long` | `java.lang.Long` → `long` |
| 20 | `TeamAuthBypass` (`team_auth_bypass`) | `val id: Long` :12 | `long` | `java.lang.Long` → `long` |

**손대지 않은 26개** (구와 이미 같아서):

| 구분 | 수 | 구 선언 | 구=신 javap |
|---|---|---|---|
| nullable 숫자 PK | 1 | `BoostReceiptPaymentsEntity.id: Long? = null` | `java.lang.Long` (래퍼가 맞다) |
| ULID 식별자(기본값 있음) | 19 | `val identifier: String = UlidCreator.getUlid().toString()` | `java.lang.String` |
| 문자열 식별자(기본값 없음) | 4 | `val identifier: String` (`Customer`·`Threads`·`ThreadComment`·`ThreadReply`) | `java.lang.String` |
| 그 외 문자열 PK | 2 | `Post.id: String` · `Token.refreshTokenKey: String` | `java.lang.String` |

수정 후 **46/46 JVM 타입 일치, 46/46 게터 반환타입 일치**(`getId()`/`getIdentifier()` 도 javap 로 대조).
남는 차이는 `final` 하나뿐이다 — 구는 Kotlin `val` 이라 `private final long`, 신은 Hibernate 필드접근·`@NoArgsConstructor`·Lombok 게터 때문에 `private long`.
`isNew()` 는 `Class#isPrimitive()` 만 보므로 이 차이는 판정에 관여하지 않는다.

### 지시받은 다른 식별자 축(`identifier: String` 19곳·4곳) — 문제 없음, 다만 축이 하나 더 있었다

문자열 PK 25개는 구·신 모두 `java.lang.String` 이라 `isNew()` 의 원시형 갈래를 타지 않는다.
대신 이 엔티티들은 **`Persistable<String>` 오버라이드가 `isNew()` 를 직접 결정**한다. 그쪽도 전수 대조했다:

```
신 엔티티 48 | Persistable 23 | isNew 오버라이드 23
구 엔티티 51 | Persistable 23 | isNew 오버라이드 23
Persistable/isNew 축 불일치: 0   (본문 전부 createdAt == null 로 동일)
```
구에만 있고 신에 없는 엔티티 3개(`admin`·`channel`·`post_reply_history`)는 이번 이식 범위 밖이라 대조 대상이 아니다.

### 컴파일러가 강제한 호출부 2곳 — 둘 다 구와 **더** 같아졌다

| 파일:라인 | 수정 전 | 수정 후 | 구 원본 |
|---|---|---|---|
| `AdminService.java:247` | `!team.getStore().getId().equals(store.getId())` | `team.getStore().getId() != store.getId()` | `AdminService.kt:179` `if (team.store.id != store.id)` |
| `BizMessageService.java:113` | `appUser.getId() != null && appUser.getId() == 316L` | `appUser.getId() == 316L` | `BizMessageService.kt:60` `if (appUser.id == 316L)` |

널 검사는 **박싱 때문에 생겼던 군더더기**였고 원시형 복귀로 불가능해지는 동시에 불필요해졌다.
`TeamAuthBypass.of(Long id, …)` → `of(long id, …)` 도 같은 이유(구 주생성자가 `id: Long` non-null).

### 변경 파일

| 파일 | 요약 |
|---|---|
| `modules/legacy/.../db/domain/**` 20개 | `@Id` 를 `private long id;` 로. 각 파일에 구 선언·기전 4줄 주석 |
| `modules/legacy/.../admin/service/AdminService.java` | 원시형 값 비교로 (구 1:1) |
| `modules/legacy/.../client/bizmessage/service/BizMessageService.java` | 널 검사 제거 (구 1:1) |
| `modules/legacy/src/test/.../db/domain/LegacyEntityIdJvmTypeTest.java` **(신규)** | 46엔티티 `@Id` 타입 표 고정 + 표/실제 집합 일치 강제 |
| `app/src/test/.../entityid/LegacyIdIsNewPersistTest.java` **(신규)** | 실 PG(Testcontainers)에서 D2 경로 재현 |

`ProductPost.java` 의 "지금은 손대지 않는다 — P4/P5 백로그" 주석(라운드1-P2 지적 9)도 갱신했다.
그 주석이 적은 **"둘 다 INSERT 로 끝난다"가 틀렸다** — `merge()` 는 관리 사본을 만들기 때문에
원본을 참조 중인 관리 엔티티가 flush 되면 `TransientPropertyValueException` 이 난다. 그 오판이 D2 를 P5 까지 끌고 왔다.

---

## D2 검증

### ① 뮤테이션 — 새 테스트가 실제로 무는가

`PostReplyAiRecommend.id` 만 `private Long id = 0L;` 로 되돌리고 두 테스트를 돌렸다.

```
:modules:legacy:test  LegacyEntityIdJvmTypeTest  FAILED
  [구/신 @Id JVM 타입 불일치 …] Expecting empty but was:
  ["PostReplyAiRecommend.id : java.lang.Long — 구는 `val id: Long = 0` 라 JVM 타입이 long 여야 한다"]

:app:test  LegacyIdIsNewPersistTest  3 tests completed, 3 failed
  isNew 기전    : expected: long / but was: java.lang.Long
  본문 2건      : org.hibernate.TransientPropertyValueException: Persistent instance of
                  '…post.Post' references an unsaved transient instance of '…PostReplyAiRecommend'
```
QA 가 실측한 500 바디와 **같은 예외·같은 문구**다. 뮤턴트 제거 후 둘 다 그린.

이 뮤테이션이 부수적으로 증명한 것 하나: **Boot 4.0.7 / Hibernate 7 에서도 원시형 `long` @Id 는
`JpaMetamodelEntityInformation.getIdType()` 이 `long.class` 로 보고한다.** (수정본에서 `isEqualTo(long.class)` 가 통과)
"Kotlin 시절 Hibernate 에서만 그랬을 수 있다"는 의심을 여기서 닫았다.

### ② 로컬 전체 스위트

```
booster : ./gradlew test --rerun-tasks → BUILD SUCCESSFUL in 5m 32s
          classes=236 tests=906 failures=0 errors=0 skipped=8
          (기준선 234/901/0/0/8 — 신규 테스트 2클래스·5건만 증가, 스킵 불변)
gateway : ./gradlew test --rerun-tasks → BUILD SUCCESSFUL, classes=9 tests=84 failures=0 errors=0 skipped=0
Routes.java:34 = new Route("legacy", MEDILAWYER, "/api/v1", ServiceId.LEGACY)   ← 무변경
git diff HEAD --numstat -- apps/gateway-app → 0줄
```

### ③ 라이브 — "수정 전 500 이 나던 요청이 200 이 되는가"

DELL 에 **새 병행 컨테이너 2개**만 띄웠다(기존 94개 무접촉, compose 미사용).
`legalcare/booster:p5qa`(미수정, QA 가 만든 그 이미지) 와 `legalcare/booster:p5d2`(수정본)에
**같은 env 파일**(`~/p5-qa/p5-booster.env`)로 A/B 를 했다.

```
[수정 전 p5qa] POST /api/v1/post-reply-ai-recommend?postId=P5D2_PROBE_20260811                → HTTP=500
               POST .../post-reply-ai-recommend?postId=NAVER_MAP_6a4dbc82ce3d590ee12032bc     → HTTP=500
   바디: {"errors":[{"code":"SERVER-001","message":"org.hibernate.TransientPropertyValueException: …"}]}
   두 번 호출 후 행수 불변(82269 / 81873) — 롤백 확인

[수정 후 p5d2] 같은 요청 2건                                                                   → HTTP=200
   바디: {"status":"200","message":"AI 추천 답글 조회가 완료되었습니다.",
          "data":{"result":"SUCCESS","status":"OK","replyAi":"dev-contract-fixture"}}
   골든(구 운영) 응답 봉투와 동일
```
ML 은 개발계 스텁이 `{"status":"OK","result":"SUCCESS","replyAi":"dev-contract-fixture"}` 를 준다 —
즉 QA 가 본 500 은 ML 실패가 아니라 D2 경로가 맞다(스텁 응답을 직접 찍어 확인).

라이브에 올린 jar 는 위 소스 그대로다. 이후 **주석 위치만** `@Id` 위로 옮겼고(19파일),
그 뒤 javap 재대조에서 46/46 타입이 그대로임을 다시 확인했다 — 바이트코드 의미는 동일하다.

**골든셋 24건 전수 재발사**(QA 하니스의 `replay-write.jsonl` 을 그대로 읽어 postId 를 재사용):
```
골든 post-reply-ai-recommend 요청: 24
골든 상태코드 분포 : {'200': 24}
신(수정본) 상태분포 : {'200': 24}
distinct postId    : 12
골든 대비 불일치    : 0
```

### ④ 라이브 회귀 — 20개 엔티티를 건드렸으니 나머지도 봤다

기준선은 **QA 가 미수정 빌드로 기록한 값**이고, 구(legacy-service)로는 **한 발도 쏘지 않았다**.

```
읽기 리플레이 119건 (replay-read.jsonl 의 new_status/new_body 대조)
  → {'SAME': 119}                      (상태·바디 전부 동일)

계약 스모크 309프로브 (smoke-report.jsonl 의 new_status/new_shape 대조)
  → {'SAME': 308, 'STATUS_DIFF': 1}
     차이 1건 = POST /api/v1/schedule/alimtalk/new-negative-post · D_415 · 500 → 200
```
그 1건은 내 수정과 무관하다 — **대조군으로 증명했다**. 같은 시각에 **미수정 이미지**(`p5qa`)로 같은 프로브를 쏘니
역시 `HTTP=200`("알림톡 부정리뷰 생성 스케줄링 (테스트용)을 완료했습니다."). QA 가 D3 로 기록한 **시각 의존 결함**
(00:30 발화만 `HourOfDay -1` 로 죽는다)이고, QA 는 00:32 KST 에, 나는 01:43 KST 에 쟀다.

읽기 리플레이는 **처음에 2건이 달랐다**(`/api/v1/stores/261/posts/statistics/period-count`, `positiveCounts 0→1`).
원인을 재지 않고 넘기지 않았다 — `store_channel.id=676` 의 `store_id` 가 261 이라 **내가 넣은 합성 probe post 1행** 때문이었고,
그 행을 지운 뒤 다시 돌리니 `{'SAME': 119}` 가 됐다.

---

## 안전조건 준수와 내가 만든 부작용 (자진 신고)

```
기존 컨테이너 정지·삭제·재생성 0 · compose 미사용 · 실행 중 env 변경 0
운영 RDS 접근 0 · 게이트웨이 라우트/코드 변경 0 · medilawyer-boot 수정 0 · 커밋 0
legacy-service                  RestartCount=0 StartedAt=2026-08-10T03:46:38Z running (불변)
legalcare-local-legacy-bridge-1 RestartCount=0 StartedAt=2026-08-06T08:26:44Z running (불변)
컨테이너: 회수 후 running 94 / 전체 99 — QA baseline-containers.tsv(99행, created 4·exited 1 포함)와 이름집합 diff 0
외부 실호출 0 — p5d2-after 의 /proc/net/route 기본 게이트웨이 0개(network internal=true),
               ESTABLISHED 41개 전부 172.28.0.0/16 (5432×20·3306×10·9092×6·27017×3·6379×1·5672×1), 공인 대상 0
               앱 로그에 ntruss/sens.apigw 시도 0건
```

| 무엇 | 상세 | 처리 |
|---|---|---|
| 병행 컨테이너 3개 | `p5d2-before` · `p5d2-after` · `p5d2-before2`(대조군) | **전부 회수** |
| 이미지 `legalcare/booster:p5d2` | 재현용으로 남김(QA 의 `p5qa` 와 같은 취급) | 잔존 — 필요 없으면 `docker rmi` |
| 신 PG 쓰기 | 합성 post 1행 + `post_reply_ai_recommend` 13행 + 스모크 D_415 의 `boost_review_request_setting` 1행 | **전부 삭제 원복.** 최종 `tables=144 rows_total=45,845,365` — QA `rows-final.tsv` 와 **diff 0줄** |
| 구 PG | 접근 0 (구로는 한 발도 쏘지 않았다) | 쓰기 0건 |
| 부정리뷰 스케줄러 1회 실행 | 스모크의 D_415 프로브가 `@RequestBody` 없는 이 핸들러를 태웠다(QA 때와 같은 프로브). 대상 155명 로깅까지 감 | 발송 0 — `serviceMode=dev` 라 `sendAlimTalkSvc` 즉시 return, `sesBaseUrl` 도 블랙홀. 로그에 발송 시도 0건 |
| Docker Hub 에서 `curlimages/curl:8.11.1` 1회 pull | 앱 egress 가 아니라 데몬의 이미지 취득. 이후에는 로컬 이미지 재사용 | 자진 신고 |

**원복하지 못한 것 하나 — 정직하게 남긴다.** 골든 postId 12건 중 **11건의 `post.updated_at` 이 오늘로 앞당겨졌다**.
성공 호출이 `post.post_reply_ai_recommend_id` 를 채우면서 JPA 감사가 `updated_at` 을 갱신했고, 나는 **사전 스냅샷을 1건만 떴다.**
FK 는 전부 NULL 로 되돌렸고 추천 답글 13행도 지웠지만, 원래 `updated_at` 값은 복원할 근거가 없어 **추정으로 채우지 않았다.**

- 원복 완료: `NAVER_MAP_6a4dbc82ce3d590ee12032bc` → `2026-07-08 07:54:38.112672+00`(사전 스냅샷 보유)
- 원복 불가 11건: `NAVER_MAP_6a4dbbf98a313423a1e67185` · `NAVER_MAP_6a4dba167fe436c9f4c6d224` ·
  `NAVER_MAP_6a4dbc72c697b8bb16a053aa` · `NAVER_MAP_6a4dabf72851ca4afd41e02d` · `NAVER_MAP_6a4f6239c753c2e7fd526bb5` ·
  `GOOGLE_MAP_…` 6건 → `updated_at` 이 `2026-08-10 16:36~16:37+00` 로 남아 있다(원래는 07-08/07-10/07-11).
  행수·FK·다른 컬럼은 전부 기준선이다.

---

## D1 — 고치는 방법과 영향 범위 (코드 무수정)

### 왜 `basePackages` 로는 구조적으로 못 잡는가

`LegacyExceptionAdvice`(`modules/legacy/.../LegacyExceptionAdvice.java:41`)가 못 잡는 5축은 **생산 지점이 legacy 모듈 밖**이다.

| 축 | 생산 지점 | 왜 advice 가 개입 못 하나 |
|---|---|---|
| 무토큰 401 `90008` | `core/.../filter/TokenVerificationFilter.java:116` throw → 같은 파일 `:129-139` 에서 `mapper.writeValue` 로 **응답을 직접 쓴다** | DispatcherServlet 에 도달하지 않는다. `@ControllerAdvice` 는 어떤 설정으로도 못 낀다 |
| 잘못된 토큰 400→**401** | 같은 필터 `:97-107`(`rejectsInvalidToken`) + `:131` 이 상태를 **401 로 고정** | 동일 |
| `/internal` 미매핑 메서드 401→**403** `90403` | `app/.../security/InternalAccessBlockFilter.java:246` (필터) | 동일 |
| 미매핑 경로 → 404 `40400` | `core/.../advice/ControllerExceptionAdvice.java:112` (`NoResourceFoundException`) | 핸들러가 선택되지 않아 "컨트롤러 빈의 패키지"가 없다 → `basePackages` 필터가 후보에서 뺀다 |
| 미지원 메서드 → 500 `50000` | 같은 파일 `:138` catch-all | `HttpRequestMethodNotSupportedException` 도 핸들러 선택 전 예외라 동일 |

즉 **필터 단계**와 **DispatcherServlet 단계** 두 곳이 필요하고, 둘 다 `:core`/`:app` 에 있다.

### 파급이 큰 이유 — 숫자로

| 공용 지점 | 이 코드가 책임지는 표면 |
|---|---|
| `TokenVerificationFilter` (`:core`) | booster 전 HTTP 표면. 컨트롤러 매핑 실측 **548개** 중 legacy 는 155개(28%) — 나머지 **393개**(user 94·post 61·crawling 51·external 51·product 48·notification 46·certification 15·medicontents 12·chrono 4·app 11)가 같은 필터를 탄다 |
| `ControllerExceptionAdvice` (`:core`) | 위와 동일 + `ControllerExceptionAdviceErrorContractTest` 가 404=`40400`·500=`50000` 을 **테스트로 못박아** 두었다 |

### 선택지

| 안 | 방법 | 평가 |
|---|---|---|
| A | `LegacyExceptionAdvice` 의 `basePackages` 제거(전역 승격) | **불가.** 9모듈 393매핑의 에러 봉투가 통째로 구 형식이 된다. 애초에 `basePackages` 를 둔 이유가 이것(해당 파일 주석) |
| B | **소유 정책에 "에러 렌더러"를 얹는다** | **권장.** 아래 |
| C | 상태코드만 맞추고 바디는 둔다 | R1 부분 충족. 다만 400→401·401→403 두 축은 상태가 바뀌므로 어차피 필터를 건드려야 한다 |
| D | 게이트웨이에서 봉투 변환 | 라우트 변경 금지에 저촉하고, 게이트웨이가 백엔드 에러 스키마를 알게 되는 층 위반 |

### 권장안 B 를 구체적으로

이 저장소에는 이미 **정확히 이 목적의 확장점**이 있다. `RestrictedPathPolicy`(`:core`)는
"경로 소유권을 선언한 정책이 있으면 그 정책이 판정하고, 없으면 기존 방식 그대로 — **기존 경로의 판정 경로는 한 줄도 바뀌지 않는다**"
라는 원칙으로 설계됐고, 구현체 `LegacyPathAuthPolicy` 는 `:app` 에 있다. `TokenVerificationFilter` 는 이미
`owningPolicy` 를 들고 다닌다(`:102`, `:115`).

1. **필터 축** — `RestrictedPathPolicy` 에 `default` 메서드로 에러 렌더러 훅을 하나 추가하고
   (`TokenVerificationFilter:129-139` 의 catch 와 `InternalAccessBlockFilter:246` 이 그것을 통해 바디·상태를 쓰게),
   `LegacyPathAuthPolicy` 만 구 봉투(`{errors:[{code,message}],errorMessage,path,timeStamp}`)와 구 상태코드를 반환한다.
   기본 구현은 현행 `ResultResponse` 그대로 → **정책이 없는 393매핑은 바이트 단위로 불변.**
2. **DispatcherServlet 축** — 404/405 는 `ControllerExceptionAdvice` 에서 `request.getRequestURI()` 로 갈라야 한다.
   `@ControllerAdvice` 는 "이 예외는 안 잡겠다"를 표현할 수 없으므로(다시 던지면 500), **advice 를 하나 더 얹는 방식은 성립하지 않는다.**
   같은 소유 판정(`policy.owns(uri)`)을 그 클래스 안에서 부르는 형태가 된다.
3. **경계 조건** — 구 `AUTH-006`(잘못된 토큰)은 **400**, 구 무토큰은 **401**, 구 미매핑 경로는 필터가 먼저 사용자 조회를 해서 **400**.
   즉 "legacy 면 무조건 401" 이 아니라 **구의 상태코드 표를 그대로 옮겨야** 한다. 이 표는 QA 가 미리보기로 실측해 뒀다(07-qa-report §D1).

### 작업량·리스크 추정

- 손대는 파일: `RestrictedPathPolicy`(`:core`) · `TokenVerificationFilter`(`:core`) · `ControllerExceptionAdvice`(`:core`) ·
  `InternalAccessBlockFilter`(`:app`) · `LegacyPathAuthPolicy`(`:app`) + 새 렌더러 1개 = **6개 안팎**.
- 회귀 위험의 핵심은 "**정책이 없는 경로에서 한 글자도 안 바뀐다**"를 증명하는 것이다.
  `ControllerExceptionAdviceErrorContractTest` 에 **legacy 경로 축**을 더하고, 나머지 8모듈 대표 경로의 봉투를 고정하는 테스트가 같이 필요하다.
- **검증은 P6 미리보기 게이트웨이로만 확인 가능하다** — QA 가 쓴 방식(`GATEWAY_SERVICES_LEGACY_URL` 만 booster 로 돌린 별도 인스턴스)
  그대로 구/신을 나란히 찍어 5축을 재야 한다. 지시대로 **P6 는 사용자 확인 전까지 하지 않으므로 여기서 멈춘다.**

---

## 리뷰어에게

- **D2 의 발현 조건은 여전히 좁다.** "다른 관리 엔티티가 새 인스턴스를 참조한 채 flush" 뿐이라, 20개 중 나머지 19개가
  지금 깨져 있었다는 뜻은 아니다. 이번 수정은 **타입을 구와 같게 만든 것**이지 "19곳의 버그를 고친 것"이 아니다.
  다만 `Otp`(`POST /api/v1/otp/send` 가 여전히 BLOCKED)처럼 **리플레이가 못 덮은 쓰기 경로는 이번에도 미검증**이다.
  QA 총평의 "쓰기 경로를 실제로 태우는 리플레이(구·신 DB 정렬 + 발송 스텁 확보)"는 여전히 남아 있다.
- **비-`@Id` 필드의 박싱은 손대지 않았다.** 구는 non-null Kotlin `Long`/`Int`/`Boolean` 이 전부 원시형인데
  신은 `TeamAuthBypass.appUserId` 처럼 래퍼인 자리가 많다. `isNew()` 축과 무관하고, 널 계약은 직전 커밋(`4df03196`)이
  다룬 영역이라 D2 범위로 끌어들이지 않았다. 별도 백로그로 볼 것.
- **회귀 검증의 기준선이 QA 의 미수정 빌드 기록이다.** 즉 "QA 가 잰 것과 같다"는 것이지 "구와 같다"는 것이 아니다.
  구와의 대조는 QA 리포트가 이미 한 것을 신뢰했다(구로 다시 쏘지 않았다 — 안전조건).
- 골든 post 11건의 `updated_at` 드리프트는 위에 적은 대로 복원하지 못했다. 개발계 미러라 기능 영향은 없다고 보지만,
  숨기지 않고 ID 를 전부 남겼다.

STATUS: REVISED
