# legacy-service → booster-app 흡수

**사용자 요청 원문**
> 다 하고 나면 레거시 서비스도 부스터 api 로 합쳐버려 그리고 개발계에서 전체 테스트해 자체 리뷰로
> (방식) 기존에 테스트한 방식있을꺼야 그대로 셀프 리뷰로 해서 진행해 진행한 방식 따로 기록도 해두고
> (목표) 구는 그대로 두고 신규를 시스템 전체 한셋트를 옮겨서 100% 맞추는게 목표야
> 응 진행해, 그리고 개발계에서 전체 테스트 다해

**구성**: writer=claude · reviewer=claude(단독 셀프리뷰) · mode=review · rounds=2 · quality off

## 22:45 오케스트레이터 → 착수 전 범위 실측

**대상 확정** — `legacy-service` = `/Users/steve/steve/legal-care/medilawyer-boot` (구 코틀린 모놀리스)
- `module-core/core-api/src/main/resources/application.yml:11` → `spring.application.name: legacy-service`
- 배포 이미지명 `legacy-service-{profile}` (`build.gradle.kts:109`), git HEAD `930c83cf`
- 규모: **638 파일 / 28,861 줄 / `@RestController` 38개**. 모듈 4개(module-client·core·storage·support)

**신규 스택이 레거시에 묶여 있는 지점 — 딱 하나다**
```
게이트웨이 라우트 19개 중 LEGACY 행선지 = "/api/v1" 1건 (Routes.java:34)
  gateway → GATEWAY_SERVICES_LEGACY_URL=http://legacy-bridge:8080
          → nginx upstream legacy_upstream = host.docker.internal:41010
          → 구 legacy-service 컨테이너(network_mode: host, 포트 41010)
```
- 실통 확인: `legacy-bridge:8080` 응답 403(인증 게이트, 경로 자체는 살아 있음)
- 그 외 신규 앱의 `TRANSITION_LEGACY_URL` 은 **전부 `http://stub:8099`(스텁)** — booster·reviewed 둘 다.
  즉 **실제 레거시 의존은 게이트웨이 `/api/v1` 경로 하나뿐이다.**
- (이전 세션에 추가된 `legacy-v2` = `/api/v2` → LEGACY 도 같은 성격)

**옮길 분량 — 147개 전부가 아니다**
- APM 3개월 실측: `/api/v1` **147개 중 76개 실사용**, 71개 미사용
- 76개 중 **20개는 3개월 통틀어 1~9회**(관리자·초기설정 기능). 짧은 관측 창에서는 안 보이는 것들이라
  "안 쓰니까 버린다"는 판단을 26시간 창으로 하면 안 된다(사용자가 이미 반려한 근거)

**booster-app 현황** — 흡수 받는 쪽
- 모듈 9개: certification · chrono · crawling · external · medicontents · notification · post · product · user
- `@RestController` **62개**. `InternalLegacyController.java` 가 이미 있다(레거시 접점 존재)

**검증 방식** — `legalcare-consulting/_analysis/legalcare-map/08-신개발계/검증방식-리플레이-차등검증.md`
골든셋 `/mnt/ex_disk1/prod-capture/*.golden.jsonl`(35파일·2.4GB) · 작업 디렉토리 `/mnt/ex_disk1/replay-20260807/`
판정: "200 나와야 할 게 다른 에러면 불합격, 원래 나던 에러는 그대로면 통과"
**리플레이가 못 보는 것**(카프카·스케줄러·외부 API)은 별도 확인 필요

## 23:05 team-pm → 오케스트레이터 (1차)
- R1~R10 / AC1~AC10, 단계 P0~P6 + 횡단 2개. RISK: HIGH. STATUS: READY
- PM 이 스스로 찾은 제약 4건: ①booster `bypass.url` substring 매칭 → `/api/v1` 이식 즉시 401 회귀 위험
  ②구·신이 **같은 PG(`medilawyer`)** 를 본다 → 데이터 이관·DDL 불필요 ③`LegacyService.java:16` 이 boost1 을 HTTP 로 호출
  ④구에 `@Scheduled` 3건 + `@KafkaListener` 1건 → 병행 시 이중 실행

## 23:15 오케스트레이터 → 지시서 검수: **②가 틀렸다. 반려 사유**
PM 은 저장소의 `core-storage.yml`(AWS RDS 주소)을 읽고 판단했는데, 구 legacy-service 는 그 파일을 안 쓴다.
실행 인자가 `-Dspring.profiles.active=workstation,git,vault` — **config-server + Vault 에서 받는다.**
그래서 **실제 연결처를 프로세스에서 직접 추적**했다.

| | 구 legacy-service | 신 booster |
|---|---|---|
| 연결처 | `127.0.0.1:5432` → **`postgresql` 컨테이너** | **`deploy-postgres-new-1:5432`** |
| DB | medilawyer | medilawyer |
| **public 테이블 수** | **153** | **94** |

**서로 다른 PG 인스턴스이고, 신규 쪽에 테이블이 59개 적다.** (신에만 있는 것은 3개)

**신규 DB 에 없는 59개 내역** (파티션을 접으면 논리 테이블 약 18개)
| 묶음 | 테이블 | 개수 |
|---|---|---|
| 알림톡 발송이력 | `kakao_alimtalk_send_history` + 월파티션 25 | 26 |
| 크롤 실패로그 | `crawling_fail_log` + 파티션 14 | 15 |
| 리뷰부스터 v2 | `boost_message_delivery` · `boost_present_send_request` · `boost_review_match_analysis` · `boost_review_platform_click` · `boost_review_request_setting` · `boost_store_present_setting` · `batch_alimtalk_send_log` | 7 |
| GEO | `geo_analysis_request` · `geo_collection_status` · `geo_hospital_source` · `geo_sov` · `geo_wup_collection` | 5 |
| 병원 공식 | `hospital_official` · `_category` · `_category_taxonomy` | 3 |
| 기타 | `crawling_daily_stats` · `kakao_alimtalk_blocklist` · `stores_updated_1` | 3 |

전부 **최근 운영에 추가된 것들**이다(리뷰부스터 v2·크롤 실패로그는 이번 프로젝트에서 다룬 바로 그 스키마).
즉 **신규 스택 DB 가 운영 스키마를 아직 못 따라잡은 상태**다.
→ "데이터 이관·DDL 불필요"는 성립하지 않는다. **스키마 동기화가 선행 단계로 들어가야 한다.**
PM 재호출 1회로 정정 지시.

## 23:25 team-pm(v2) → 오케스트레이터. 지시서 승인
- 반려 사유 정정 완료: "같은 DB" 전제 철회 → **P-1 스키마 동기화 단계 신설**(R11/AC11)
- P-1 핵심 산출물 = **147 엔드포인트 × 59테이블 접촉 매핑**. 접촉분만 DDL, 미접촉분은 범위 밖 명시
- 데이터 방침: **전량 복사 금지**, 리플레이 판정에 필요한 접촉 테이블만 캡처 시점 시드
  (빈 표로 두면 조회 응답이 구와 달라져 불일치가 데이터 노이즈로 오염된다 — 타당)
- 접촉 판정을 **정적탐색 + 리플레이 런타임 `relation does not exist` 0건**으로 이중화 — 정적탐색이 놓친 걸 런타임이 잡는다
- 이중매핑 제약을 **병행기간 데이터 이원화** 위험으로 재작성. 롤백 시 신규 PG 잔여 행 처리 방침을 AC10 필수 항목화
- "저장소 yml 은 실환경 근거가 아니다"를 제약에 못박음(v1 오류 재발 방지)
- 단계: P-1 → P0 기반·인증 → P1 읽기 → P2 코어쓰기 → P3 인증발급 → P4 부스트·관리자·비HTTP → P5 전체검증 → P6 라우트전환 + 횡단 A(147 전량이식) / B(병행·롤백)
- RISK: HIGH. STATUS: READY → **오케스트레이터 승인. P-1 착수.**

## 2026-08-10 00:20 오케스트레이터 → P-1 결과 검수 + 기준 재확립
**사용자 지시**: "운영에 나가 있는건 그대로 가져가야해 지금 개발중인것들이라서" / "전수 테스트 다되어져야 나갈수 있으니까
셀프 리뷰모드로 진행해 오래 걸려도돼 **개발계 죽이지만마 운영이랑**"

**운영 RDS 읽기전용 스키마 대조 실시**(사용자 승인 하에, SELECT only)
| | 테이블 수 |
|---|---|
| 운영 `medilawyer-prod` | **144** |
| 신규 PG `deploy-postgres-new-1` | **102** (P-1 이 3개 만든 뒤) |
| 부족 | **42** · 신규에만 있는 것 **0** (깨끗한 부분집합) |

부족 42개 = 알림톡 발송이력 본체+파티션 26(27) · 크롤 실패로그 2027 파티션(7) ·
리뷰부스터 v2 4(`boost_present_send_request`·`boost_review_match_analysis`·`boost_review_platform_click`·`boost_store_present_setting`) ·
`crawling_daily_stats`(1) · `kakao_alimtalk_blocklist`(1) → **논리 8개**
전부 **2026-07-14 복사 시점 이후 운영에 추가된 것**. 사용자가 말한 "지금 개발중인 것들"이 이것.

**담당이 보류한 판단 3건, 운영 조회로 전부 해결** — `app_user_channel_account`·`admin`·`post_reply_history`
**셋 다 운영에도 없다.** 안 만든 판단이 맞았다. 특히 `app_user_channel_account` 는 `/api/v1` 5경로가 쓰는데
운영에도 없어 구가 그 경로에서 런타임 에러를 낸다 — 신규에만 만들면 "신이 구보다 잘 도는" 상태가 되어 동등 판정이 깨진다는
담당 논리가 정확했다.

**담당 보고 오류 1건 정정** — "개발계 `postgresql` 은 사실상 빈 스크래치(store 0·app_user 2·post 18)"라 했으나
내 실측은 **store 64,644 · app_user 786 · post 2,119,442**. 빈 DB 가 아니다.
개발계를 배제한 *근거*는 틀렸으나, 판정 기준을 "구 소스가 만지는 테이블인가"로 바꾼 *결론*은 맞다.

**엔드포인트 수 정정 수용** — 147 이 아니라 **130**(`/api/v1` 115 + `/internal` 14 + `/api/v2` 1).
147 은 클래스 레벨 `@RequestMapping` 접두 19건을 중복 계상한 값.

**결정**: 부족 42개 **전부 신규 PG 에 생성**(사용자 승인). 스키마가 있다고 코드가 도는 것은 아니므로 위험이 없고,
"테이블 없음"으로 인한 조용한 유실을 미리 막는다(`crawling_fail_log` 가 그 실례였다 —
`runCatching` 이 예외를 삼켜 응답은 200 인데 write-set 에서만 드러남).

## 2026-08-10 00:35 사용자 → 야간 자율 진행 지시
> "나는 자러 갈거니까, 개발계에서 전체 테스트 해보고 아침에 확인할게 천천히 진행해"
> (직전) "전수 테스트 다되어져야 나갈수 있으니까 셀프 리뷰모드로 진행해 오래 걸려도돼 **개발계 죽이지만마 운영이랑**"

**야간 실행 계획** — 단계는 순차다(앞 단계 AC 그린 없이 다음 착수 금지).
| 순서 | 단계 | 상태 |
|---|---|---|
| 1 | P-1 완결: 운영 기준 42테이블 동기화 | **진행 중** |
| 2 | P0 기반·인증 (`modules/legacy` 신설 + 인증경계 이설) | 대기 |
| 3 | P1 읽기 → P2 코어쓰기 → P3 인증발급 → P4 부스트·관리자·비HTTP | 대기 |
| 4 | **P5 개발계 전수 테스트**(골든셋 35파일 전량 리플레이 + 카프카 + 스케줄러 + 전체기동) | 대기 |
| 5 | P6 라우트 전환 | **사용자 확인 후** — 야간에 하지 않는다 |

**야간 불변 조건 (어기면 즉시 중단)**
1. 개발계 컨테이너 정지·삭제·compose 재생성 **금지**. 스택은 아침까지 계속 떠 있어야 한다
2. 운영 RDS 는 **읽기만**. 쓰기·DDL·데이터 이동 0건
3. 신규 PG 는 **테이블 추가만**. 기존 것 ALTER/DROP/TRUNCATE 금지
4. 구 legacy-service·legacy-bridge 정지 금지(병행 유지)
5. 외부 실호출(국세청·네이버·카카오·SMS·기프트) 0건
6. 커밋·푸시 금지
7. **P6(게이트웨이 라우트 전환)은 사용자 확인 전까지 실행하지 않는다** — 이게 트래픽 방향을 바꾸는 유일한 단계다

각 단계는 셀프 리뷰(독립 컨텍스트) → 오케스트레이터 재실측 → 다음 단계 순으로 간다.
단계마다 이 로그에 append 하고, 아침에 `09-final-report.md` 로 종합한다.

## 2026-08-10 01:10 team-dev → P-1 완결 (STATUS: DONE, `P-1-schema-sync.md` 9~18절)
부족 42개 전량 생성. 운영 `pg_dump -s` 재현. 산출물 `db/legacy-absorption-p1b-schema.sql`(3,751줄, 미커밋)

## 01:20 오케스트레이터 → 독립 재검증
| 항목 | 결과 |
|---|---|
| 테이블명 집합 | **운영 144 = 신규 144**, 양방향 차집합 **0** ✅ |
| 컬럼 차이(기존 94 중) | **확인됨** — 아래 |
| 영향 규모 | **확인됨** — 아래 |

## 01:25 **P-1 이 더 깊은 층을 찾아냈다 — 이름만 맞춰선 안 되는 것이었다**

담당 지적 3건 전부 실측으로 사실 확인:

**① 저장소 DDL 이 정본이 아니었다**
`db/reviewboost-v2-alimtalk.sql` 의 `kakao_alimtalk_send_history` 는 **일반 테이블**인데
운영은 **`PARTITION BY RANGE (created_at)` + 파티션 28개**다. PK 도 `id` 단독 vs `(id, created_at)`,
`idempotency_key` UNIQUE 가 전역 1개 vs **파티션마다 1개**(유일성 범위가 **월 단위**로 좁다).
→ 저장소 파일로 운영에 적용했으면 다른 DB 가 만들어졌다. 파일은 손대지 않음(범위 밖).

**② P-1 1차 산출물 자체의 구멍 — 담당이 스스로 찾아 고침**
`boost_message_delivery` 에 운영의 `uq_boost_message_delivery_provider_message`(UNIQUE)가 없어
**운영은 막는 중복을 신규는 허용**하고 있었다. 재생성(행 0·FK 0·의존객체 0 확인 후 단일 트랜잭션).

**③ 42개를 다 만들어도 운영과 같아지지 않는다 — 기존 94개 중 5개가 구조 상이 (미조치)**
지금까지 "부족 42개" 논의가 **이름만 비교**해서 이 층을 통째로 못 봤다. 내 실측:
```
운영 varchar(255) → 신규 varchar(50)  4컬럼
  boost_post_similarity.post_id / .enhanced_post_id
  similarity_score_analysis_data.target_post_id / .compare_target_post_id
boost_present_order  컬럼 2개 누락: source(varchar10) · created_by
boost_message·post 인덱스 3종 누락(성능만)
```
**영향 규모 실측**: 신규 PG 의 `post` 중 **id 가 50자 초과인 행 175,256건**(최대 79자).
그 글과 엮인 INSERT 는 운영에선 성공하고 신규에선 `value too long` 으로 실패한다
→ **리플레이 불합격이 "이식 코드가 틀렸다"로 오판된다.** 반드시 선조치.

**ALTER 안전성 실측** (내가 직접 확인)
| 테이블 | 행수 | 판정 |
|---|---|---|
| `boost_post_similarity` | 4,866 | varchar 확대는 PG 에서 테이블 재작성 없음(메타데이터만) |
| `similarity_score_analysis_data` | 159,792 | 동일 |
| `boost_present_order` | 698 | nullable 컬럼 추가 안전 · **uuid 중복 0건** → UNIQUE 추가 가능 |

**결정**: 기존 테이블 ALTER 를 **이번 5개에 한해 허용**한다.
사용자 지시가 "운영에 나가 있는건 그대로 가져가야해"이고, 이 조치는 **운영과 어긋난 복사본을 운영에 맞추는 것**이다.
"ALTER 금지"는 파손 방지 목적이었고, 이건 파손이 아니라 정합이다. 범위를 이 5개로 못박고 진행.

## 2026-08-10 02:15 team-dev → 기존테이블 정합 + P0 완료 (STATUS: DONE)
**작업①** `db/legacy-absorption-p1c-existing-align.sql` 적용. 작성자가 적용 전 재실측해 **15절 조치안의 오류 2건을 스스로 정정**:
인덱스 이름 4개가 임시명이었음 → 운영 `pg_class` 실명으로 교체 · 운영의 `idx_boost_message_team_usable_id` 가
**`indisvalid=false`**(실패한 CONCURRENTLY 잔재)임을 발견 — 15절은 인덱스 *정의*만 봐서 이 층을 못 봤다.
- 구조 시그니처 운영/신규 **4,903 = 4,903**, 대상 5개 테이블 차이 **9종 → 0**
- 잔여 diff 3종은 전부 손댈 수 없거나 손대선 안 되는 것(운영 INVALID 인덱스 · `crawling_fail_log` 제약 **이름** `_pkey1` vs `_pkey`, 승인범위 밖이라 미조치, 코드 참조 0건)
- **filenode 불변 = 테이블 재작성 없음** · 79자 `post.id` INSERT 성공 실증(varchar(50) 대입은 SQLSTATE 22001 로 실패함도 함께 실증)

**작업② P0** — `modules/legacy` 신설 + `HealthCheck` 이식 + `bypass.url` 함정 해결
- 해법: `bypass.url` 규칙을 **안 바꿨다**(9모듈 수백 경로가 얹혀 있다). `RestrictedPathPolicy` 확장점을 만들어
  **소유를 주장한 정책이 있을 때만 분기**, 없으면 종전 판정 폴백. `LegacyPathAuthPolicy` 가 구 `SecurityConfig.kt:36-62`
  permitAll **26규칙을 선언 순서 그대로** 들고 `PathPatternParser` 로 판정
- AC2 를 10경로가 아니라 **67경로**로 실측(AUTHENTICATED 66 = 401/401, `/api/v1/health` 200/200).
  permitAll 48개는 구에 찌르면 카카오·네이버·알림톡·기프트가 실제 발사되므로 불변조건 5 때문에 미실행 —
  대신 핸들러 없는 하위경로 프로브로 게이트 통과만 실증(프로브 창 3,248줄에 컨트롤러/서비스 로그 0건, 외부 송신 0건)

## 02:25 오케스트레이터 → 독립 재검증
| 항목 | 작성자 주장 | 내 재실측 | 판정 |
|---|---|---|---|
| 운영 vs 신규 5개 테이블 구조 | 차이 0 | **67컬럼 전수 대조 diff 0** | 확인 |
| 행수 불변 | 4,866/159,792/698 | **동일** | 확인 |
| booster 테스트 | 672개 0실패 | 전 모듈 강제 재실행 → **클래스 174 · 672 테스트 · 실패 0 · 오류 0 · 스킵 0** | 확인 |
| 스택 무손상 | 91개 유지 | **91컨테이너 · 공용망 31 · 전부 running** | 확인 |
| 구 병행 유지 | 정지 안 함 | `legacy-service` · `legacy-bridge` **3일째 Up** | 확인 |
| 변경 규모 | — | 수정 3파일 **35줄** + 신규(모듈·정책·테스트·DDL). P0 치고 작다 | 양호 |

작성자가 올린 판단 필요 4건(`/internal/**` 403 불일치 · 401 바디 상이 · 핸들러없는경로 구500/신404 ·
`ApiV1Filter` 미이식)은 **셀프 리뷰어에게 심각도 판정을 맡겼다**(특히 P5 전수 리플레이에서 대량 불일치를 만들 것이 있는지).

## 2026-08-10 02:45 리뷰어(독립 컨텍스트) → **VERDICT: REQUEST_CHANGES**
상세 `round-1-P0-review.md`

**CRITICAL — 인증 경계가 절반만 이식됐다**
"**어떤 경로가 토큰을 요구하는가**"는 옮겼는데 "**토큰이 어디에 실려 오는가**"는 안 옮겼다.
- `HeaderTokenExtractor.supports()` = `ApiPathMatcher.isApiPath(uri)` · `CookieTokenExtractor.supports()` = 그 부정
- `ApiPathMatcher.isApiPath("/api/v1/...")` → **false** (그 매처는 `/apis` 와 `/{module}/apis` 만 안다)
- → `/api/v1` 요청은 **쿠키 `accessToken`** 을 읽는다. 구는 `JwtAuthenticationFilter.kt:39` 에서 **`Authorization` 헤더**를 읽는다

**리뷰어 측정**: 골든셋 `/api/v1` 369건 중 Authorization 152 · 쿠키 0. 그중 92건이 토큰필수 경로, **87건이 200 기록** → 전부 401 로 뒤집힌다

**내 독립 재실측(더 큰 규모)**: 35파일 전수 스캔
```
/api/v1 포함 레코드 4,186건
  authorization 키 등장 3,963
  cookie 키 등장            0   ← 쿠키는 단 한 건도 없다
```
**코드 확인**: `ApiPathMatcher.isApiPath` 를 직접 읽어 `/api/v1` 에 false 반환함을 확인 ·
`HeaderTokenExtractor:19` / `CookieTokenExtractor:23` 의 supports 확인 ·
구 `medilawyer-boot/.../security/JwtAuthenticationFilter.kt:39` = `request.getHeader("Authorization")` 확인

**→ P1 에서 즉시 대량 불일치.** P5 까지 못 가고 첫 읽기 단계에서 터진다. 반드시 선조치.

**AC2 가 왜 못 봤나** (내가 물었던 질문): 66개 프로브가 **전부 무토큰**이었다.
무토큰이면 게이트가 제대로 막든, 추출기가 헤더를 아예 안 읽든 **둘 다 401** 이다 → 66건 일치는 공허하다.
유효 토큰 1개로 `GET /api/v1/users/me/stores` 를 한 번만 찔렀으면 P0 에서 잡혔다.

**리뷰어가 검증한 것(반박 불가 수준)**: 26규칙 이식이 선언 순서까지 정확 · 폴백 안전
(booster 어떤 컨트롤러도 `/api/v1` 을 서빙하지 않아 `owns()` 가 다른 9모듈을 가로챌 수 없음) ·
`PathPatternParser` 버전차 문제 없음(spring-web 6.1.13 vs 7.0.8 로 **44개 엣지케이스 실행**, 출력 차이 0)

**추가 MAJOR**
- permitAll 경로가 200 은 나오지만 **익명으로 실행**된다(`appUserId`·`email`·`organizationRole` 미설정).
  `SecurityConfig.kt:48` 이 "회원용 + 비회원용"이라 명시 → **상태코드만 비교하는 AC3 규칙으로는 못 잡는다**
- 만료 토큰: 구는 permitAll 에서도 400 거부(`GlobalExceptionHandler.kt:38`), booster 는 삼키고 진행
  (`TokenVerificationFilter.java:95-102`) — 49경로에 새로 활성화된 분기. 오늘은 CRITICAL 에 가려 안 보임
- `p1b-schema.sql:3643` 에 트랜잭션 없는 실 `DROP TABLE`(가드는 건전, 자동 실행 안 됨)
- 변경표가 SQL 2파일 누락 · TSV 15행 자기모순 · AC2 표가 130 중 115만 계수
- **`GET /api/v1/team/presentHistories/{teamId}/excel` 이 표·픽스처 양쪽에서 누락**되고 대신 유령 `GET /api/v1/team` 이 들어감
  (여러 줄 `@GetMapping` 을 추출기가 못 파싱) → **전수 재점검 필요**

**작성자 자가보고 4건 판정**: 전부 MINOR 이고 이연 타당. 단 2건 정정 —
①401 바디 차이는 **리플레이가 영원히 못 잡는다**(판정이 기능 한정) → **P5 가 아니라 P6 전에** 결정해야 한다
②500→404 는 작성자가 쓴 것과 **반대 방향으로 위험**하다: 판정 기준이 그걸 **조용히 통과**시킨다 —
누락된 excel 엔드포인트가 실제로 그렇게 새어나갔다

**범위 이탈 0**: 모든 변경이 R1/R2/R11 매핑 · 게이트웨이 `Routes.java:34` 여전히 `ServiceId.LEGACY` ·
`medilawyer-boot` 무수정 · P-1c 는 승인된 5테이블만

## 2026-08-10 03:50 team-dev(RESPOND) → 지적 11건 일괄 처리 (STATUS: REVISED, `round-1-P0-response.md`)
**CRITICAL 닫힘** — `LegacyHeaderTokenExtractor` 신설. `/api/v1` 만 소유(`LegacyPathAuthPolicy.owns()` 위임으로 소유권 단일화).
`ApiPathMatcher` 무수정 → 기존 9모듈 추출 경로 불변. 구 저장소 전체에 `getCookies` 참조 **0건** 확인(쿠키 폴백 없음).

**결정적 실증 — 같은 유효 토큰으로 130경로 전후 스윕**
| | 무토큰 | 유효토큰(헤더) |
|---|---|---|
| 수정 전 booster, AUTHENTICATED 66경로 | 401×66 | **401×66** |
| 수정 후 같은 66경로 | 401×66 | **404×66** (게이트 통과) |
| 구 41010 `GET /api/v1/app-users/me` | 401 | **200** |
쿠키에만 실으면: 구 **401** / 수정전 booster **404** — 두 서버가 정반대를 읽고 있었음이 실증됐다.
무토큰 열이 전후 바이트 동일 = 리뷰어 지적("무토큰 프로브는 공허")이 수치로 확인됨.

**MAJOR 처리**
- permitAll 익명: 같은 수정으로 닫힘. 49경로 전수 조사 → 신원 미사용 25 · 선언만 20 · **실사용 4**.
  그중 2건이 실제 동작 변화(`OrganizationController.kt:249` 신원 없으면 400 · `StoreFacade.kt:100` 이 `appUser?.id` 를 카프카 메시지에 실어 write-set 상이).
  리뷰어가 든 예시 `GET /api/v1/posts/**` 는 **신원 무관**임을 밝히고 과장하지 않음(`PostRepositoryImpl.kt:50`)
- 만료토큰: `rejectsInvalidToken()` 신설. 구 **400 `AUTH-006`** vs 수정전 **200** → 현재 **401 `90006`**.
  400↔401 격차는 **안 닫힘**(공용 catch 가 9모듈 전체를 401 로 고정) → P6 전 결정 항목으로 명시
- excel 누락: 추출기를 괄호균형 파서로 재작성, 전 저장소에서 여러줄 애노테이션 **정확히 1건** 발견.
  독립 산출물 `P-1-endpoint-table-map.tsv` 와 **130↔130 양방향 차집합 0**
- 문서: TSV 130×16 재생성 · 변경표를 `git status` 와 일치(16파일) · `p1b:3637~3769` 를 BEGIN/COMMIT 으로 감쌈

**뮤테이션 검증**: 추출기 off + `@Order` 제거 + `rejectsInvalidToken=false` → 신규 테스트 **6개 실패**. 장식용 아님.

**작성자가 못 닫았다고 밝힌 2건 (정직)**
1. **구·신 JWT 서명키 대조 불가** — config-server 가 `legacy-service` 에 `propertySources: []` 반환.
   각자 자기 서명 토큰으로만 검증 → "각 서버가 헤더를 읽는다"는 증명, "**구 토큰이 신에서 검증된다**"는 미증명
2. P5 판정 규칙(`구 500 → 신 404` 불합격 · 바디 비교 · 게이트웨이 진입) 을 `04-changes-P0.md` 에 기록했으나
   공유 절차 문서(`검증방식-리플레이-차등검증.md`)는 **수정 안 함**(다른 태스크가 의존) → P5 진입 조건으로 등재

## 04:00 오케스트레이터 → 재검증 + **내 판단: ①번이 P3 가 아니라 P1 선행이다**
| 항목 | 재실측 |
|---|---|
| 새 추출기 범위 | 코드 확인 — `/api/v1` 만 소유, `ApiPathMatcher` 무수정 ✅ |
| 테스트 | 전 모듈 강제 재실행 **클래스 180 · 689 테스트 · 실패 0** ✅ (672 → +17) |
| 야간 불변조건 | 91컨테이너 전부 running · 공용망 31 · 구 legacy-service·legacy-bridge **3일째 Up** · 최근 종료는 2일 전(무관) ✅ |

**내 판단 — JWT 서명키는 P3 로 미룰 수 없다.**
리플레이는 **캡처된 요청을 그대로** 보낸다. 골든셋 `/api/v1` 4,186건 중 3,963건이 Authorization 을 달고 있고
그 토큰은 **운영 키로 서명된 것**이다. booster 가 그 키로 검증하지 못하면 **P1 첫 읽기 단계에서 전부 401** —
방금 닫은 CRITICAL 과 정확히 같은 실패 형태다. 2라운드 리뷰어에게 이 판정을 명시적으로 물었다.

**부수 기록**: 프로브 토큰 발급을 위해 `POST /api/v1/jwt/refresh` 로 구 쪽에 `token` 행 1건 upsert 됨.
정식 API 경유이고 DB 직접 쓰기가 아니다. 기록만 남긴다.

## 2026-08-10 04:20 리뷰어(2라운드) → **VERDICT 는 APPROVE 성격이나 새 지적 1건** (`round-2-P0-review.md`)
- **CRITICAL 진짜로 닫힘**: `@Order(HIGHEST_PRECEDENCE)` 로 주입 순서가 결정적임을 확인
  (Spring `AnnotationAwareOrderComparator` 정렬 → 등록 순서 무관, 프록시 안전). 실 조립 컨텍스트에서
  주입 필드를 리플렉션으로 읽는 테스트가 가드. 기존 9모듈 추출 경로 불변 재확인
- **7번(excel) 순환논증 아님**: 작성자 말을 안 믿고 구 소스에서 다시 셈 —
  다중행 애노테이션 `TeamController.kt:70` 1건, 메서드 레벨 매핑 131 − Feign 아웃바운드 1 = **130**. 양방향 차집합 직접 `comm` 으로 0/0

- **새 지적 — 서명키.** 작성자의 "config-server 가 `propertySources: []` 라 대조 불가" **전제가 틀렸다.**
  구는 `/app/resources/` 의 yml 4개만 쓰고(bootstrap 없음·configserver import 없음) 키는 저장소 자신의
  `application-workstation.yml` 에 평문 리터럴이다. config-server 무관.

## 04:30 오케스트레이터 → 키 불일치 독립 확인. **P1 차단 사유 확정**
| 대상 | sha256 앞16 | 길이 |
|---|---|---|
| 구 legacy-service 저장소(`application-workstation.yml:18` `accesstokenSecretKey`) | `a1fa3d59…` | **55** |
| 신 booster 저장소 기본값(`application-dev.yml`) | `ba5da75e…` | **55** |
| **신 실행 중 `legalcare-local-booster-web-5` 의 `JWT_SECRET`** | **`bd18abff…`** | **44** |

**길이부터 다르다(44 vs 55) — 명백히 다른 키다.** 실행 중 컨테이너의 env 가 저장소 기본값을 덮고 있다.
(내 1차 추출은 base64 패딩 `=` 에서 잘려 43자로 나왔다. `${line#*=}` 로 정확 재추출해 44자 확정.)

**영향**: 리플레이는 캡처된 요청을 **그대로** 보낸다. 골든셋 `/api/v1` 요청 중 Authorization 을 단 것들의 토큰은
구 키로 서명돼 있다. 신이 다른 키로 검증하면 **P1 첫 읽기 단계에서 전부 401** — 방금 닫은 CRITICAL 과 같은 실패 형태.
게다가 이번 라운드에 켠 `rejectsInvalidToken()=true` 가 위험을 키웠다: 키가 어긋나면 **permitAll 49경로까지 401**
(수정 전이었으면 200 익명). 옳은 수정이지만 **키 정합이 선행돼야 한다.**

**야간 조치 결정**: 키를 맞추려면 실행 중 `booster-web` 컨테이너를 재생성해야 한다 →
사용자 조건("개발계 죽이지만마")과 2026-08-07 실사고 전례 때문에 **야간에 하지 않는다.**
→ **P1 리플레이 실행의 진입 조건**으로 등재하고 아침 판단에 올린다.
대신 **P1 코드 이식은 키와 무관**하므로 계속 진행한다(리플레이 실행만 보류).

## 04:35 내 수치 정정 (리뷰어 지적 수용)
내가 04:00 에 쓴 골든셋 "4,186건 중 authorization 3,963" 은 **요청 수가 아니라 문자열 출현 줄 수**다.
`json.dumps` 문자열에 `/api/v1` 이 포함되면 세는 방식이라 과대계상됐다.
**정확한 값은 리뷰어의 369건 / Authorization 152 / cookie 0** 이다. 방향(쿠키 0)은 동일하나 규모는 내 쪽이 틀렸다.

## 2026-08-10 04:45 오케스트레이터 → **P0 종료(MERGE 상태)**, P1 착수
2라운드 지적 4건은 전부 **실측 1회 + 문서 정정**이고 코드 수정 불필요 → 핑퐁 종료.
새 지적(서명키)은 P0 산출물의 결함이 아니라 **개발계 배포 env 문제**라 P1 진입 조건으로 분리한다.

**P1 진입 조건 (리플레이 실행분에만 해당)**
1. `legalcare-local-booster-web-5` 의 `JWT_SECRET`(44자)이 구 키(55자)와 다르다 → 리플레이 대상 booster 는
   **구 키로 검증**할 수 있어야 한다
2. 실행 중 컨테이너를 건드리지 않는다(사용자 조건 + 2026-08-07 실사고) →
   **리플레이 전용 booster 를 옆에 세우는 방식**을 우선 검토(병행 원칙과 동일)
3. 게이트웨이는 `/api/v1` 을 LEGACY 로 보내므로 **리플레이는 booster 직결**로 쏜다(P6 라우트 전환 금지)

**P1 코드 이식은 키와 무관** → 지금 진행한다.

## 2026-08-10 10:55 team-dev → P1 이식 완료 (STATUS: DONE, `04-changes-P1.md`)
읽기 7컨트롤러 → `modules/legacy` 신규 123파일. 이식 경로 구 **11** ↔ 신 **11**(`comm -3` 0줄).
booster 전체 **703 / 실패 0**(689 → +14) · gateway 76/0 · `Routes.java:34` 불변 · 커밋 0 · 불변조건 전부 유지.

**테스트가 실제로 잡은 결함 2건** (작성 중 실패 → 코드 수정)
1. **Jackson 키 중복 생성** — lombok `isFirst()` 게터 때문에 `{"isFirst":true,...,"first":true,"last":true}` 로 나감.
   필드에 `@JsonProperty` 를 달아도 게터 프로퍼티는 별개다. **상태코드 200 그대로라 리플레이로는 영원히 못 잡는 종류.**
2. 에러 봉투 키 순서 역전(`errorMessage` 가 앞으로) → `@JsonPropertyOrder` 고정
기대값은 상상이 아니라 **구 개발계 41010 직접 실측**(읽기 프로브 6건, 외부 실호출 0)

## 11:05 오케스트레이터 → **골든셋 커버리지 실측. 검증 전략을 바꿔야 한다**

담당이 "P1 11경로 중 8경로가 골든셋에 0건"이라 보고 → 전체 규모를 내가 직접 쟀다.

```
골든셋 35파일 전수 스캔
  /api/v1 경로 인스턴스 총 257건
  경로 템플릿으로 접으면 → 15개 엔드포인트
```
**이식 대상 130개 중 골든셋이 덮는 건 15개(11.5%)뿐이다.**

전체 목록(빈도순): `boost1/.../messages/{id}/posts` 68 · `boost1/.../message-templates` 43 ·
`users/me/stores` 35 · `organizations/{id}/teams` 29 · `boost1/.../messages` 20 ·
`message-templates/{id}` 8 · `app-users/me` 6 · `app-user-store/list` 6 ·
`channel-accounts` 6 · `channel-accounts/login` 6 · `team/{averageScore,presentCount,messageCount,keyword,reviewWriteRate}/{id}` 각 6

**왜 이렇게 좁은가**: 골든셋은 2026-07-16~17 **약 하루치 캡처**다. APM 3개월 기준 실사용은 76개인데
하루 창에는 15개만 잡혔다. 사용자가 이미 반려한 "26시간 창으로 판단하면 안 된다"의 같은 현상이다.

**→ "전수 테스트 = 골든셋 리플레이"는 성립하지 않는다.** 리플레이는 130개 중 15개만 판정한다.
나머지 115개는 다른 검증이 필요하다. **아침 보고의 최우선 항목.**

## 11:20 사용자 → "응 이어서 하고 다 되면 보고서로 보고해"
검증 전략은 오케스트레이터 제안 **(다)** 로 확정하고 진행:
- 골든셋이 덮는 **15개** → 리플레이 정밀 판정(응답 + DB 행 변화)
- 나머지 **115개** → **계약 스모크**(경로·메서드·인증경계·응답 바디 형태를 구/신 대조)
- **커버리지를 최종 보고에 명시**한다. "전수 테스트 통과"라고 뭉뚱그리지 않는다
- 추가 캡처(운영 트래픽 더 뜨기)는 운영 접근이 필요하므로 **보고서의 판단 항목**으로 올린다

**계약 스모크가 필요한 근거가 P1 에서 이미 나왔다**: Jackson 키 중복(`isFirst`+`first` 동시 출력)은
상태코드 200 이라 리플레이 판정으로는 영원히 안 잡힌다. 바디를 봐야 잡힌다.

## 11:45 리뷰어(P1 1라운드) → **VERDICT: REQUEST_CHANGES** (`round-1-P1-review.md`)
이식 품질 자체는 높다고 평가 — Kotlin `let`/엘비스 반환값 의미론, `kotlin.math.round`=`Math.rint`,
`trim()`↔`strip()`, lombok `isXxx` 게터의 Jackson 명명 등 조용히 틀리는 지점을 대부분 정확히 짚었고
11경로를 양쪽 소스에서 독립 재열거해 일치 확인.

**지적 5건**
1. **CRITICAL** `DisplayName.java:13-16`·`GoogleMapsLinks.java:13-19` `@JsonPropertyOrder` 누락 + 생성자 2개
   → Jackson 알파벳 정렬. 구글 검색 응답에서 5키 중 4개 이동. 작성자가 잡은 `isFirst` 중복키와 **같은 부류인데 전수가 아니었다**.
   바깥 타입의 핀은 **중첩 타입에 전파되지 않는다**
2. **MAJOR 인증 사용자 조회 축소** — 구 `JwtAuthenticationFilter.kt:36-80` 은 Authorization 만 있으면
   **전 경로에서** `loadUserByUsername` 호출(없으면 400). 신 `LegacyUserDetailsArgumentResolver` 는
   `CustomUserDetails` 파라미터를 선언한 **6개 메서드에서만**. 나머지는 구 400 → 신 200
3. **MAJOR 카프카 와이어 포맷 상이** — 구 `ADD_TYPE_INFO_HEADERS=true`(`__TypeId__` 발행) + `MAX_REQUEST_SIZE=524288000`,
   신은 app `QueueConfig.java:66-81` 을 빌려 써서 `false` + 기본 1MB. 문서에 한 줄도 없음
4. **MAJOR 모듈→app 역방향 의존** — `KafkaProducer.java:31` 의 `@Qualifier("queueKafkaTemplate")`. 기술 제약 위반
5. **MAJOR** `RestClientSupport.java:163` 의 `READ_UNKNOWN_ENUM_VALUES_AS_NULL=true` 가 구 `ObjectConfig.kt` 엔 없음
   → ML 이 미지 status 를 주면 구 400 → 신 200 + `status:null`

**작성자 판단 3건 처리**: ①1:1 경계 **타당**(도달가능 메서드·연관 누락 0, EAGER·orphanRemoval 0으로 검증).
단 미이식 개수가 문서엔 1/1/2 인데 실제 **18/12/4** 이고 `Post.postReplyRegistrationRequestList` 가 목록에서 누락
(코드 javadoc 은 정확, 문서로 옮기며 뒤집힘) ②구 버그 2개 보존 **맞음** ③**PII 는 "1:1이라 못 고친다"를 기각** —
1:1 이 보호하는 건 URL·바디·상태·에러코드이고 **로그는 거기 없다**. 마스킹해도 AC3 영향 0.
추가로 `PostReplyService.java:145-147` 도 실명+답글 전문을 찍는데 미공개

## 11:55 오케스트레이터 → #2 독립 확인
구 `JwtAuthenticationFilter.kt:36-80` 실물 확인: `if (accessToken != null)` 안에서 경로 무관하게
`customUserDetailsService.loadUserByUsername(email, organizationIdAndRole)` 호출 → 사용자 부재 시 전파되어 400.
신 `LegacyUserDetailsArgumentResolver.resolveArgument` 는 ArgumentResolver 라 **파라미터 선언 시에만** 동작,
`CustomUserDetails` 선언 지점 **6곳**. → **지적 사실이고, 골든 35건 중 33건이 그 위에 있다.**
게다가 방향이 **구 400 → 신 200** 이라 AC3("구 200 → 신 비200 이면 불합격")이 **조용히 통과시킨다.**

## 12:30 사용자 질문 → **JWT 키 구조 실측. P3 지시서에 반영할 발견 1건**
> "기존에 회원은 하나 아니였어? jwt가 여러군데 나뉘어져 있나?"

| 대상 | sha256 앞16 | 길이 |
|---|---|---|
| 구 legacy-service **access**(`medilawyer.jwt.accesstokenSecretKey`) | `ba5da75e…` | 55 |
| 구 legacy-service **refresh**(`refreshtokenSecretKey`) | `a1fa3d59…` | 55 |
| **신규 4앱 저장소 기본값**(`jwt.key`, booster·reviewed·lawkit 공통) | **`ba5da75e…`** | 55 |
| 실행 중 booster-web·booster-worker·reviewed-web·lawkit-web | `bd18abff…` | **44** |
| 실행 중 gateway | (JWT_SECRET 없음 — 라우팅 전용) | — |

**답**: 회원(사용자 저장소)은 하나가 맞다. **키가 나뉘어 있다.**
1. **구는 access·refresh 가 서로 다른 키** 2개
2. **신규 저장소 기본값 = 구의 access 키와 정확히 동일**(`ba5da75e…`) → 코드는 이미 맞춰져 있다.
   누군가 의도적으로 구 access 키를 신규 기본값으로 박아뒀다
3. **개발계 배포 env 가 44자 키로 덮고 있고, 신규 3앱이 전부 같은 값**을 쓴다
   → **신규끼리는 일관, 구와만 끊어짐.** 코드 문제가 아니라 배포 설정 문제

**새로 드러난 과제 (P3 반영 필요)**
**신규에는 refresh 전용 키가 없다.** 구는 access/refresh 를 다른 키로 서명하는데 신규는 `jwt.key` 하나뿐이다.
→ **구가 발급한 refresh 토큰을 신규가 검증할 수 없다.** 흡수 후 기존 세션의 토큰 갱신 불가로 이어진다.
지시서 AC5 는 "구/신 × 발급/검증 4조합"으로만 적혀 있고 **refresh 키 부재는 안 잡혀 있었다.**
→ P3 착수 지시에 명시한다.

**P1 리플레이 차단 요인의 정확한 성격도 이걸로 확정**: "키를 골라야 한다"가 아니라
**"개발계 env 가 저장소 기본값을 덮고 있으니 되돌리면 된다"**. 단 3앱이 같은 env 를 쓰므로 함께 바꿔야 하고
컨테이너 재기동이 필요하다 → 사용자 확인 항목 유지.

## 12:45 사용자 결정 → **인증을 게이트웨이로 집중. 이번 흡수 범위에 포함**
> "어차피 인증은 gw지나오면서 처리 할건데 분리 될 이유 없어, 어차피 신서버는 인증은 gw가 다 할꺼니까"
> (선택) "이번에 같이 옮긴다"

**실측 — 구조는 이미 만들어져 있다**
| | 인증 코드 |
|---|---|
| 구 gateway(`legal-care/gateway`) | **없음** |
| 신 gateway-app | **있음** — `auth/JwtVerifier.java` · `auth/IdentityHeaderFilter.java` · `auth/IdentityHeaders.java` · `proxy/ProxyFilter.java` |

동작: `IdentityHeaderFilter:34` 가 `verifier.enabled()` 일 때만 검증하고 `X-Auth-*` 를 attribute 로 남김 →
`ProxyFilter` 가 다운스트림 주입 + **인바운드 `X-Auth-*` 스푸핑 스트립**.
`JwtVerifier:45` `enabled() = key != null`.

**그런데 지금 꺼져 있다** — 실행 중 gateway 컨테이너에 `JWT_SECRET` 이 **없다**(실측).
키가 없어 `enabled()` false → 신원 헤더 미주입 → 현재는 **각 앱이 자체 검증** 중.

**목표 구조 (사용자 확정)**
```
클라이언트 --Authorization--> 게이트웨이
                              ├ JwtVerifier 검증 (키 1개)
                              └ X-Auth-* 신원헤더 주입
                                    ↓
                    booster / reviewed / lawkit  (헤더를 믿음 · 자체 검증 불필요)
```
→ **키는 게이트웨이에만. refresh 도 게이트웨이 담당.**

**이 결정이 해소하는 것**
- 신규 3앱의 `JWT_SECRET` 44자 vs 구 55자 불일치 → 게이트웨이 하나만 맞추면 된다
- **refresh 전용 키 부재**(12:30 발견) → 앱마다 들 이유가 없어진다
- P0 의 `LegacyPathAuthPolicy`·`LegacyHeaderTokenExtractor` → **과도기 폴백**으로 성격 전환(삭제 아님)

**P2 에 미치는 영향은 없다** — `LegacyUserDetailsArgumentResolver` 는 request attribute
(`CommonConstants.EMAIL`·`ORGANIZATION_ROLE`)를 읽는 구조라, 그 attribute 를 채우는 주체가
토큰 파싱이든 `X-Auth-*` 헤더 매핑이든 **리졸버 코드는 안 바뀐다.** 진행 중인 P1수정+P2 는 그대로 둔다.

**신설 단계 P0b(게이트웨이 인증 집중)** — P2 완료 후 착수:
1. 게이트웨이에 키 주입(저장소 기본값 = 구 access 키) + `enabled()` 활성 확인
2. `X-Auth-*` → 다운스트림 attribute 매핑 경로 확인(booster 쪽 수신부)
3. **refresh 토큰 경로** 설계 — 구는 refresh 전용 키가 따로 있다. 게이트웨이가 갱신을 담당할 때 그 키를 어떻게 다룰지
4. **스푸핑 방어 실증** — `ProxyFilter` 의 인바운드 `X-Auth-*` 스트립이 실제로 동작하는지(외부에서 헤더 위조 시도)
5. 다운스트림 자체 검증을 끌지 남길지 판정

## 13:30 team-dev → P1 수정 완료 + P2 착수 (STATUS: DONE)
**P1 대응** (`round-1-P1-response.md`) — 지적 5건 + PII 처리. 클린빌드 **729/0**, gateway 76/0, 커밋 0.
- **#1 은 반박이 붙었다**: "생성자 2개면 알파벳 정렬"이 실측에서 재현 안 됨. 실제 규칙은
  **"전필드 생성자가 유일한 non-default 생성자가 아니면 알파벳"**. 다만 순서가 `-parameters` 플래그와
  생성자 개수라는 두 우연 위에 선다는 우려는 옳아서 **핀 없던 13개 타입 전부에 부착** + `JsonPropertyOrderCoverageTest` 로 영구화
- **#2** 인터셉터로 되살림. 필터가 아닌 이유는 제약 — 필터에서 던지면 `handler==null` 이라
  `basePackages` 걸린 `LegacyExceptionAdvice` 미적용 → 구 봉투가 안 나온다. 뮤테이션에서 `expected:<400> but was:<200>` 재현
- **#3** 소비자 특정: `legal-care/event-coordinator`(kafkajs) 가 `consumer.ts:1016-1021` 에서 headers 를
  구조분해만 하고 안 씀, `__TypeId__` grep 0건. 그래도 1:1 기준이라 타입매핑으로 **구 FQCN** 이 실리게 함
- **#5 에서 자기 테스트의 허점을 발견**: 처음엔 한 테스트로 묶었는데 **컨버터 교체를 통째로 지워도 7건 전부 통과** —
  DTO 방어가 컨버터 겹을 가리고 있었다. 겹을 나눠 각각 뮤테이션하고, 겹2의 오늘 효과가 0이라는 것도 기재
- **PII 3번째 건 발견**(리뷰어·작성자 둘 다 놓쳤던 것): `MlService.java:54` 가 **복호화된 채널 계정 ID** 를 로깅

**P2 착수** (`04-changes-P2.md`, `P2-endpoint-inventory.tsv` 51행) — **정직한 부분 이식**
R4 는 **51 엔드포인트 · 3,850줄**로 P1 의 5배. 한 라운드에 다 하는 건 불가능하다고 판단해
①스코프 인벤토리 ②과금 경로 실측 ③외부호출 폐포 0인 최하단 슬라이스만 이식. **49개 미이식이라고 인벤토리에 명시.**

**과금 판정(중요)**: `grep -rn "\.price\b"` → **빈 출력**. `Product.price` 는 구 저장소 전체에서 읽히는 곳 0건.
purchases 는 `product_post` 1행 INSERT + Kafka 1건이 전부 → 잔액·원장·금액 산술 없음 →
비관적 락·BigDecimal 대상 아님. **단 구에 traceId 도 멱등도 없어 같은 요청 2번이면 행이 2개** 생긴다.
AC4 가 "요청별 DB 행 변화 일치"를 보므로 **멱등을 넣는 순간 리플레이 불합격** → 보존이 맞고, 판단 항목으로 올림

## 13:40 오케스트레이터 → 재검증 + 담당이 못 푼 DB 확인
- **테스트 729/0 확인**(클래스 191, 강제 재실행)
- **담당이 불변조건 때문에 확인 못 한 P2 대상 테이블 존재 여부를 내가 조회**:
  `store`·`app_user`·`app_user_store`·`team`·`team_app_user`·`team_auth`·`team_channel_account`·
  `organization`·`organization_app_user`·`product`·`product_post` → **전부 있음**
  **`app_user_channel_account` 없음**(기지) · **`notice` 없음** ← **새로 발견**
→ `notice` 는 P-1 의 "운영 144 = 신규 144" 와 모순되지 않는지 확인 필요(운영에도 없으면 정합).
  3개 DB 대조 조회 실행 중.

## 13:55 오케스트레이터 → `notice` 건 해소
3개 DB 대조 결과: `notice` 와 `app_user_channel_account` **모두 운영·신규PG·개발계구DB 전부에 없다**(f/f/f).
→ P-1 의 "운영 144 = 신규 144" 와 **모순 없음**. 컨트롤러는 있는데 테이블이 없어 **구도 그 경로에서 실패**한다.
신규에 만들지 않는 것이 맞다(만들면 "신이 구보다 잘 도는" 상태가 되어 동등 판정이 깨진다).
P-1 담당의 `app_user_channel_account` 판단 논리가 `notice` 에도 그대로 적용된다.

## 14:00 오케스트레이터 → **P0b(게이트웨이 인증 집중) 착수**
사용자 결정(12:45)에 따라 신설한 단계. P2 나머지(49경로)보다 먼저 하는 이유:
- **P1 리플레이 차단 요인(JWT 키)을 푸는 경로**가 이쪽이다. 앱마다 키를 맞추는 대신 게이트웨이 하나로 모은다
- refresh 전용 키 부재 문제도 같이 해소된다
- P2 나머지 이식은 이 구조가 정해진 뒤에 해야 신원 주입 경로를 두 번 안 고친다

## 14:20 사용자 → P5 방식 확정
> "다 되고 나면 신규 스택들 다 띄워서 전수 테스트 한번 진행해, 다진행하고 보고만해"

**P5 실행 계획 (병행 셋트 방식 — 기존 개발계 무손상 유지)**
크롤러 통합 때 쓴 것과 같은 원칙: **기존을 내리지 않고 신규 셋트를 옆에 세운다.**

| # | 할 일 | 근거 |
|---|---|---|
| 1 | 이식분이 들어간 **신규 이미지 빌드**(booster·gateway 최소, 필요시 reviewed·lawkit) | 현재 개발계 이미지엔 P0~P4 코드가 없다 |
| 2 | **병행 셋트로 기동** — 자체 이름·자체 포트, 기존 컨테이너 무접촉 | "개발계 죽이지마" 조건 유지 |
| 3 | 병행 게이트웨이에 **JWT 키 주입**(구 access 키) + 인증 활성화 | P1 리플레이 차단 요인 해소. **실행 중 컨테이너는 안 건드린다** |
| 4 | 병행 게이트웨이 라우트에서 `/api/v1` → **BOOSTER** | 원본 `Routes.java` 는 `LEGACY` 유지. 병행본에서만 전환 |
| 5 | **리플레이 전수 실행** — 골든셋 35파일, 읽기/쓰기 샤드 분리 | 검증방식 문서 절차 그대로 |
| 6 | **계약 스모크** — 골든셋이 안 덮는 나머지 경로 | 커버리지 15/130 이라 리플레이만으론 부족 |
| 7 | 리플레이가 못 보는 것: **카프카 왕복 · 스케줄러 · 전체 기동** | 검증방식 문서의 "못 보는 것" 절 |
| 8 | 정리 — 병행 셋트 회수, 기존 스택 무손상 재확인 | |

**판정 규칙 (P0·P1 에서 보강된 것 반영)**
- 기본: "200 나와야 할 게 다른 에러면 불합격, 원래 나던 에러 그대로면 통과"
- **추가**: 구 500 → 신 404 도 **불합격**(P0 리뷰 지적 — 이식 누락이 조용히 통과되는 구멍)
- **추가**: 상태코드만 보지 말고 **응답 바디 비교**(P1 에서 Jackson 키 중복·순서 결함이 200 뒤에 숨어 있었다)
- **추가**: 구 400 → 신 200 도 불합격(P1 인증 조회 축소 건이 이 방향이었다)

**보고**: 전 단계 종합 `09-final-report.md` + 사용자 보고. **커버리지를 뭉뚱그리지 않는다** —
리플레이 15경로 / 계약 스모크 나머지로 명시.

## 15:10 team-dev → P0b 완료 (STATUS: DONE, `04-changes-P0b.md`) — **내 전제 2건이 틀렸다**

**① 정정: 게이트웨이 인증은 켜져 있다.** 내가 12:45·14:00 에 "실행 중 gateway 에 `JWT_SECRET` 없음 → `enabled()` false"
라고 쓴 건 **프로퍼티 이름을 잘못 짚은 것**이다. 게이트웨이가 읽는 건 **`JWT_TOKEN_KEY`**(`gateway-app/application.yml:27`).
내 재실측:
```
gateway  JWT_TOKEN_KEY  sha=bd18abfff82b7904  len=44
booster  JWT_SECRET     sha=bd18abfff82b7904  len=44   ← 동일 키
로그 "JWT_TOKEN_KEY 미설정" 경고 0건
```
→ **게이트웨이는 지금도 검증하고 `X-Auth-*` 를 주입 중**이고, 앱들과 같은 키를 쓴다.

**② 진짜 원인은 키가 아니라 토큰 출처였다.** `IdentityHeaderFilter` 가 `ApiPathMatcher.isApiPath()` 로
헤더/쿠키를 가르는데 `isApiPath("/api/v1/...")`=**false** → 쿠키를 찾는다. 구 클라이언트는 쿠키를 안 보낸다.
**P0 라운드1 CRITICAL 과 똑같은 결함이 게이트웨이에도 그대로 있었다**(실행 중 이미지 소스 `8b1f7ec` 로 확인).
→ 그래서 `/api/v1` 은 **한 번도 신원 주입을 받은 적이 없다.**

**주요 판정 3건 (작성자, 근거 타당)**
- **refresh 는 게이트웨이가 담당할 수 없다** — `JwtProvider.kt:81-98` 4단계 중 순수 암호연산은 1개뿐,
  나머지는 `token` 테이블·`organization_app_user` DB 상태다. 게이트웨이엔 데이터소스가 없다(`build.gradle:14-28`).
  결정적으로 **구는 refresh 를 헤더가 아니라 바디로 받는다**(`RefreshReqDto`) — 무변형 스트리밍 프록시는 바디를 안 읽는다
- **"키는 게이트웨이에만"은 발급을 옮기지 않는 한 성립 안 함** — `createToken` 호출부가 로그인·초대수락·JwtController 로
  전부 앱에 있고 그것들은 위 이유로 앱에 남는다 → 앱이 서명키를 계속 든다 → **다운스트림 자체 검증 유지**로 판정
- **스푸핑**: 헤더를 그냥 믿으면 인증 무력화. 헤더 소비 경로는 만들되 공유 시크릿(`X-Auth-Trust`) 뒤에 두고
  **기본 꺼짐**. 시크릿이 비면 오늘 동작이 한 줄도 안 바뀐다

**검증**: booster **744/0**(+15) · gateway **84/0**(+8) · `Routes.java:34` 불변 · 개발계 93 running ·
`legacy-service` RestartCount=0 · 컨테이너 조작 0 · 커밋 0
**뮤테이션 5건 전부 문다.** 그중 하나는 테스트 작성 중 발견한 구멍 —
기능 스위치를 빼면 `MessageDigest.isEqual([],[])==true` 라 **`X-Auth-Trust: ""` 하나로 뚫린다**

**미검증(작성자 자진 신고)**: 실 게이트웨이 ↔ 실 booster 로 신뢰 경로를 **통과시켜 보지 못함**(컨테이너 재생성 금지).
헤더 이름이 어긋나면 **에러 없이 신원이 조용히 사라진다** → P5 진입 조건에 등재.
`X-Auth-Trust` 가 정적 시크릿이라 사설망 관찰자에 취약(HMAC 승격은 결정 항목).

## 15:30 사용자 → 인증 확정 + 다음 단계 확인
> "다음 뭐할 차례야 / 인증은 그렇다치자 기존대로간다고하고"

**인증 구조 확정**: P0b 판정대로 **기존 유지**.
- 게이트웨이는 지금처럼 검증 + `X-Auth-*` 주입(이미 켜져 있음, 키 `bd18abff…` 44자)
- **다운스트림 자체 검증도 유지** — 토큰 발급(로그인·초대수락·JwtController)과 refresh 가 앱에 남아야 하므로
  앱이 서명키를 계속 든다. "키는 게이트웨이에만"은 성립하지 않음
- `X-Auth-Trust` 신뢰 경로는 **기본 꺼짐** 유지(시크릿 비면 오늘 동작 무변화)
- HMAC 승격·헤더 이름 정합 실증은 P5/후속

**진행률**: 130경로 중 **13 이식**(P1 11 + P2-A 2), **117 남음**
**다음**: P2 나머지 **49경로**(약 3,500줄) 착수

## 15:55 team-dev → P2-B (STATUS: DONE) — **49 중 27 이식, 22 잔여**(정직 신고)
`OrganizationController` 21 + `AppUserStoreController` 2 + `AppUserChannelAccountController` 4 = **27경로**.
컨트롤러 단위로 통째 이식, **반쯤 이식된 경로 0**. `modules/legacy` main java 136 → 223.
잔여 22 = `StoreController`(12) · `TeamController`(7) · `NoticeController`(2) · `ProductController`(1) —
각각 `StoreChannelHelper`(502줄) · 리뷰부스트v2 엔티티 6종+POI · `BizMessageService` · `ProductPostFacade`(375줄)+Kafka 폐포가 딸려온다.

**이번 라운드 핵심 — 테스트가 잡은 결함 1건**
`MlAppUserChannelAccountLoginRes.CookieDict` 의 **밑줄로 시작하는 필드 7개가 응답에서 통째로 증발**했다.
```
핀 없이 직렬화 실측 → {"TUID":null,"TSID":null,"UUID":null}   // 밑줄 7개 증발
```
원인: 구는 Kotlin data class 라 `jackson-module-kotlin` 이 **주생성자 파라미터**로 프로퍼티를 찾는데,
Java 는 **게터 이름**을 본다. `get_kau()` 는 접두사 뒤가 대문자가 아니라 유효 게터로 인정되지 않는다.
증발한 것이 **카카오 2차 SMS 인증 세션 쿠키**다 — 비면 2차 인증이 항상 실패하는데 **HTTP 상태는 200** 이라
**리플레이 상태코드 비교로는 절대 안 잡힌다.** `@JsonProperty` 로 못박고 양방향 테스트, ML DTO 4개 전수 적용.
(내 확인: 구 소스에 `_kau` 실재 — `AppUserChannelAccountService.kt:168` · `TeamChannelAccountFacade.kt:176`)

**검증**: booster **763/0**(작성자) → **내 재실행 764/0**(클래스 199) · gateway 84/0 · `Routes.java:34` 불변 ·
뮤테이션 6건 전부 뭄 · 괄호균형 대조 빠짐0·유령0 · 접촉테이블 27/27 일치 · 외부 실호출 0 · 커밋 0

**작성자가 올린 판단 3건**
1. `app_user_channel_account` 미생성 → 5경로가 구와 똑같이 런타임 실패. **AC4 "행 변화 일치"가 그 위에서 무의미** → 리플레이 판정에서 별도 분류 필요
2. **P3 진입 조건 추가**: 조직 초대가 **초대 링크 토큰을 발급**하므로 `medilawyer.jwt.{access,refresh}tokenSecretKey` 를 P2 에서 열었고 기본값이 자리표시자다. 실키 없이는 구 발급 초대토큰을 신이 검증 못 한다
3. **미검증**: `authenticateChannelAccount` 의 카카오맵 **30초 타임아웃**이 `RestClientSupport` 공통값으로 바뀜 — 느린 응답에서 갈릴 수 있는데 값을 재보지 않음

## 16:40 team-dev → **P2 완료. 인벤토리 미이식 0 (51/51)**
`StoreController` 12 + `TeamController` 7 + `NoticeController` 2 + `ProductController` 1 = 22경로 추가.
`modules/legacy` main **223 → 308**. `:core`·게이트웨이·`medilawyer-boot` 무수정, 커밋 0.
검증: booster **776/0** · gateway 84/0 · 빠짐0·유령0(괄호균형, 게이트 `containsExactlyInAnyOrder` 63경로) ·
DTO 필드명·선언순서 기계대조 **32쌍 불일치 0** · 엔티티 애노테이션 9쌍 일치 · 뮤테이션 **9건 전부 뭄** ·
개인정보 미마스킹 로그 0 · 접촉테이블 P-1 표 일치

**① 타임아웃 — 실측했고 구와 다르다. 안 고쳤다(다음 라운드 1순위)**
내 독립 확인:
| | 연결 | 읽기/응답 |
|---|---|---|
| 구 `RestTemplateConfig.kt:12-13,22-23` | 10s | **60s** |
| 구 `WebClientConfig.kt:18,27-28` | 5s | **10s** |
| 구 카카오맵 인라인 | — | **30s** |
| **신 `RestClientSupport.java:102-104`** | **없음** | **없음 → 무한** |

`JdkClientHttpRequestFactory` 에 타임아웃을 **하나도 설정하지 않는다.**
→ 외부가 응답을 안 주면 **스레드가 영원히 붙잡힌다.** 값 하나로 덮으면 셋 중 둘이 틀리고,
그 팩토리는 **P1·P2-B 경로에도 이미 얹혀 있어** 이번 슬라이스 밖을 건드린다. **횡단 과제로 승격.**

**② 뮤테이션이 실제 구멍을 찾았다** — `getPresentHistories` 의 `masking` 을 `false`→`true` 로 뒤집어도
**처음엔 아무 테스트도 안 깨졌다.** `masking=false` 는 **환자 연락처(`boost_customer.id`, 전화번호)를 평문으로
응답에 싣는다**는 뜻이라 값이 뒤집히면 계약이 바뀐다. `TeamServiceStatisticsTest` 신설로 못박음
(마스킹·페이징 보정·정수 나눗셈·평균 반올림·기간창)

**③ 작업 중 실수 1건 (자진 신고)** — P2-B 산출물 `AppUserService.java`(git 미추적)를 새 파일로 덮어써 삭제.
컴파일 오류로 즉시 드러나 구 Kotlin 원본 + 4개 호출부 기준 재작성, 776/0 확인.
**단 P2-B 가 달아둔 주석은 복원 안 됨.** 동작은 동일

## 16:50 오케스트레이터 → 재검증
타임아웃 격차 **독립 확인**(위 표는 내가 구 소스에서 직접 뽑은 값).
`RestClientSupport` 에 Timeout 설정 줄 **0개** 확인 → 무한 맞다.
→ **P2 셀프 리뷰에 타임아웃을 명시 항목으로 넣는다.** 이식 슬라이스가 아니라 **횡단 결함**이다.

## 17:35 리뷰어(P2 1라운드) → **VERDICT: REQUEST_CHANGES** (지적 14건, `round-1-P2-review.md`)
51경로 전량을 구 소스와 대조. 코드 무수정(`git status` 32건 그대로, HEAD `b42d9b6` 불변).

**[BLOCKER] 지적 11 — ML 로그인 실패가 클라이언트에 "성공"으로 나간다**
구 `MlAppUserChannelAccountLoginRes.kt` 의 `result`·`status` 는 **기본값 없는 non-null** →
ML 이 키를 빼먹으면 `KotlinModule` 이 던져서 `authenticateChannelAccount` 400 `REQUEST-003`,
catch 없는 `kakaoSms`·`naverCaptcha` 는 500 `SERVER-001`.
신은 `@Setter @NoArgsConstructor` 라 `result=null` 로 채워지고:
```java
// AppUserChannelAccountController.java:72-74
mlResponse.getResult() == ChannelAccountLoginResult.FAIL ? "실패했습니다" : "성공했습니다"
```
`null != FAIL` → **200 + "성공했습니다" + `{"result":null,"status":null,…}`**. 채널계정 **6경로**.
방향이 **구 4xx/5xx → 신 200** 이라 **AC3 판정을 정확히 빠져나간다.**

**내 독립 확인**: 구 DTO 의 `result: ChannelAccountLoginResult` / `status: ChannelAccountLoginDescription`
기본값 없음 확인 · 신 DTO `@Setter @NoArgsConstructor` + 평범한 필드 확인 · 컨트롤러 72-74행 실물 확인. **사실이다.**

**리뷰어 핵심 통찰**: 지적 3(요청 바디)과 **뿌리가 같다** — Kotlin non-null → Java 널 허용이
**요청·응답 양쪽에서** 증발했다. → "DTO 하나씩 고치기보다 **이 축의 복원 방침을 먼저 정하라**"

**[MEDIUM] 지적 12 — 인가가 넓어지는 방향으로 갈렸다**
구 `JwtAuthenticationFilter.kt:46-69` 는 `associate{}` 전체가 한 try 라 역할 하나만 깨져도 **맵 전체를 비운다**
(→ 조직 엔드포인트 7개 전부 `AUTH-002`). 신 `LegacyAppUserResolutionInterceptor.java:114-130` 은
엔트리별 try/catch 라 **나머지 조직을 통과시킨다**. 견고성은 신이 낫지만 1:1 에서는 구 의미를 따라야 한다.

**[LOW] 13·14** — `TODO()`(`NotImplementedError` 는 `Error` 라 구 `@ExceptionHandler(Exception)` 에 안 잡힘)를
`UnsupportedOperationException` 으로 바꿔 500 바디 상이 · 복호화 실패를 `IllegalStateException(e)` 로 감싸
`errorMessage` 가 `cause.toString()` 이 됨. 후자는 `legacy.yml:27-28` crypt 키가 자리표시자라 **relogin 이 지금 매번 이 경로를 탄다**

**[정리 필요] 5·6** — 결함이 아니라 **R# 없는 변경**(게이트웨이 신뢰헤더 9파일 + SQL 3개) → 승인 전 정리

**깨끗했던 것(29경로)**: URL·메서드·경로변수 전 쌍 · DTO 25개 필드명·선언순서·중첩타입 · 열거형 12개 ·
에러코드·메시지 문자열(의도 보존한 구 버그 4종 포함) · `@Transactional` 배치(구가 빠뜨린 3곳까지) ·
**positional 생성자 전환 지점 전수에서 필드 뒤바뀜 0건**

## 18:35 team-dev(RESPOND) → P2 지적 처리 완료 (STATUS: REVISED, `round-1-P2-response.md`)

**BLOCKER + 지적3 — 축 전체를 복원했다(DTO 하나씩이 아니라)**
복원 수단 선정 기준을 "구의 예외 타입·에러코드·HTTP 상태까지 같아지는가"로 잡고 셋을 비교:
- `@JsonProperty(required=true)` → **명시적 null 을 못 잡는다**
- `@JsonSetter(nulls=Nulls.FAIL)` → **키 누락을 못 잡는다**
- 구 `KotlinModule` 은 **둘 다 던진다** → 위 둘은 각각 절반짜리
→ 무인자 생성자로 채워진 뒤 "여전히 null 인 필드"를 보는 **`LegacyNonNullModule`** 신설.
예외 계열이 `MismatchedInputException` 으로 같아져 그 위 매핑이 전부 따라온다
(요청 바디 500 `SERVER-001` / ML 응답 400 `REQUEST-003`·500)

기계 전수: **112클래스 420필드** 추출 → 역직렬화 경로 **35클래스 117필드**에 표시, 구 Kotlin 집합과 **불일치 0**.
primitive 19필드는 박싱(primitive 는 "키 누락"과 "0 을 실제로 보냄"을 구별 못 해 검사가 성립 안 함 —
표시가 primitive 에 붙으면 기동 시 터지게 함)

**리뷰어 주장 3건 반박 (근거 있음, 내가 1건 재확인)**
- **지적1(a) 타임아웃 "10s/60s 원복"** → **틀렸다.** 구 운영 실값은
  `cloud_config_repository/feign-release.yml:7-8` 의 **`connectTimeout: 200000000` / `readTimeout: 200000000`**
  = **200,000초(≈55시간, 사실상 무제한)**. 10s/60s 는 원복이 아니라 **새 정책**이고,
  리뷰어가 근거로 든 커밋 `025c270` 은 현 HEAD 의 조상도 아니다.
  **← 내가 직접 파일 확인. 사실이다.** 대신 (b) legacy 프로파일 4종은 채널 단위로 복원
  **→ 내가 16:50 에 "구 60초 vs 신 무한" 이라 쓴 것은 절반만 맞다.** `RestTemplate`·`WebClient` 는 실값이 있지만
  **운영에서 실제로 타는 Feign 경로는 구도 무제한**이었다. 정정한다
- **지적13 "Error 는 `@ExceptionHandler` 에 안 잡힌다"** → Spring 4.3+ 는 `DispatcherServlet.java:975-979` 가
  `ServletException` 으로 감싸 태운다. 갈리는 건 봉투가 아니라 **메시지 문자열**이고 MockMvc 실측으로 구와 동일 바이트
- **카카오 상호검색은 구에서 항상 400 이었다**(스네이크 키 불일치 + non-null). 복원으로 신도 200→400.
  AC3 위반 아니고 리플레이에서 눈에 띌 것이라 문서 앞에 명시

**검증**: booster **797/0**(776 → +21, `--rerun-tasks` 클린) ← **내 재실행도 797/0(클래스 205)** ·
gateway 84/0 · `Routes.java:34` 불변 · **뮤테이션 8/8 뭄**(BLOCKER 되돌리면 재현) · 커밋 0 · 외부 실호출 0

**남은 것 3건**
1. 지적14(b) `defaultStatusHandler` — append + first-match-wins 라 이기려면 조립 복제 필요,
   맞출 문자열이 구 스택마다 달라 외부 실호출 없이 검증 불가 → **P5**
2. 지적5·6 — 코드 무변경, 범위 밖 명시. 단 게이트웨이 신뢰헤더는 `Routes.java:34` 가 아직 LEGACY 라
   **라이브 구 스택에 적용된다**는 지적이 맞다 → **승격/되돌리기 결정 필요**
3. 직렬화 전용 77클래스 303필드 — 구 `!!` 전수 grep 으로 슬라이스 내 잔여 0 확인,
   단 `!!` 없이 nullable 이 흘러드는 경로의 데이터플로 분석은 미실시
