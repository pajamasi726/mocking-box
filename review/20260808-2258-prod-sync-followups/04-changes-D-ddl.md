# 변경 요약: D — 리뷰부스터 v2 신규 DDL + 이관 계획 (코드 본체 이관 제외)

> **[리뷰 라운드1 반영]** #1(HIGH): 백필 §2의 "boost_message_delivery=v1 테이블" 전제가 거짓(실측 07-30 신설 469f7ba4, 스냅샷에 없음) → §2를 DO 블록 존재 가드로 조건화(부재 시 NOTICE 스킵), 픽스처를 스냅샷분/post-v2분으로 분리해 거짓 그린 제거, 스냅샷 상태 psql 재현으로 스킵 실증.
> **#2(LOW)**: "구 코드도 getFirst()로 1건만" 주석은 오류(구 코드는 size()!=1이면 예외, 20b3181~1 실측) → 실제 구 동작으로 정정 + 다중 ACTIVE 병원 사전 노출 쿼리를 §1 앞에 추가(동작 변화를 운영 판단 사항으로 명시).

## 변경 파일 (전부 apps/booster-app 하위, worktree feature/prod-sync-candidates, 미커밋)

| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `app/src/main/resources/db/reviewboost-v2-product.sql` | match_analysis·send_request·platform_click·store_present_setting 4종 CREATE(운영 확장분 통합본, 부분 UNIQUE=PG 문법) | R6 |
| `app/src/main/resources/db/reviewboost-v2-alimtalk.sql` | kakao_alimtalk_send_history(+CHECK에 BLOCKED)·blocklist·boost_review_request_setting(+방어적 3컬럼 ALTER) | R6 |
| `app/src/main/resources/db/reviewboost-v2-order-alter.sql` | boost_present_order ADD source/created_by(스냅샷 기존재 테이블이라 유일한 ALTER) | R6 |
| `app/src/main/resources/db/crawling-fail-log.sql` | 월별 RANGE 파티션 현행형 신설(PK(id,occurred_at)·인덱스4·DEFAULT 파티션) | R6 |
| `app/src/main/resources/db/reviewboost-v2-backfill.sql` | §1 B2C-345 시드(다중 ACTIVE 노출 쿼리 포함) + §2 B2C-343 이관(delivery 부재 시 DO 가드 스킵, 존재 시 이관·건수 NOTICE), 양쪽 멱등 | R8 |
| `app/src/test/resources/db/reviewboost-v2-snapshot-fixture.sql` | 스냅샷 "실존" 5테이블만의 최소 대역(delivery 제외 사유 명기, 실DB 적용 금지) | R6 |
| `app/src/test/resources/db/reviewboost-v2-postv2-fixture.sql` | §2 이관 경로 검증용 delivery 대역(post-v2 상태, 실DB 적용 금지) | R6 |
| `app/src/test/java/legalcare/medilawyer/ReviewBoostV2DdlTest.java` | 배포 파일 원본을 Testcontainers postgres:16에 적용하는 스모크(shedlock 관례) — §2 스킵/이관 양경로 단언 포함 | R6 |
| `{TASK_DIR}/03-design-D.md` | 코드 이관 파일 매핑·의존 순서 계획서 | R5·R7·R9 |

## 테스트 결과

- `./gradlew :app:test --tests ReviewBoostV2DdlTest` — **BUILD SUCCESSFUL** (tests=1 failures=0 errors=0). 수정 전 라운드에서 :app 전 6클래스 스위트도 통과(모듈 코드 미변경 — 리소스·테스트만).
- psql 직접 검증(일회용 postgres:16 = dev compose 동일 버전), **스냅샷 상태(= delivery 없음) 재현**: 5파일 순서 적용 무오류 → 백필 실행 시 §1 시드 2건(101=최신, 103 EXPIRED·store 0 제외) + 다중 ACTIVE 노출 쿼리가 store 101 {2,1} 표시 + **§2 "relation does not exist" 없이 NOTICE 스킵**. 이어서 postv2 픽스처 생성 후 재실행 → §2 legacy=2/migrated=2 이관 수행, §1 행수 불변(멱등). 부분 UNIQUE 중복거절·멱등키 중복거절·BLOCKED INSERT·파티션 라우팅은 1라운드에서 검증 완료(해당 파일 무변경).

## AC 자가 점검 (이번 산출물 범위)

- AC6 전반부(DDL 작성+testcontainers 스키마) ✅ — 상기 테스트. 후반부(개발계 PG 적용)는 미실행 — 오케스트레이터 지시로 파일 산출까지만, 적용은 운영자/후속.
- AC8 준비물 ✅ — 백필 스크립트 멱등+스냅샷 안전(§2 가드) 검증 완료. 원자 적용(코드 배포와 묶기)은 이관 단계 1에서.
- AC5/7/9 — 코드 미이관(스코프 제외), 03-design-D.md 계획으로 대체.

## 알려진 한계 / 리뷰어에게

1. **추정 1건**: `kakao_alimtalk_blocklist.created_at/updated_at NOT NULL` — 엔티티(BaseEntity)는 nullable 매핑, 자매 테이블 운영 실측(NOT NULL)을 따름. 그 외 전 컬럼은 엔티티/운영 DDL 문서 실측(파일 헤더에 근거 커밋·문서 명기).
2. `uq_bsp_setting_active_store`·`uk_brrs_store`는 운영 DDL 원문 미보존 — 정의는 코드 주석·읽기측 unique=true에서 복원, 제약명은 개발계 신규 부여라 운영과 이름이 다를 수 있음(기능 동일).
3. 운영 `ck_kash_status`에 BLOCKED가 실제 추가돼 있는지 미실측(문서 공백) — 개발계는 포함이 정답(없으면 차단 감사 INSERT 즉사), 운영 대사 시 확인 권장.
4. blocklist 조회 인덱스는 운영 실존 미확인이라 정의 보류(파일 내 주석 제안만). crawling_fail_log는 빈 DB 신설이라 운영의 pre2608 과거분 파티션 미생성(데이터 없음).
5. 백필 §1의 다중 ACTIVE 병원 처리: 구 동작(예외로 발송 중단)과 달리 최신 1건을 시드한다 — 사전 노출 쿼리 결과가 있으면 운영 판단 후 진행하도록 파일에 명시. §2는 개발계에 delivery가 생기는 시점(운영 데이터 복사 등)에 재실행하면 실제 이관을 수행한다(멱등).

STATUS: DONE
