# Round 1 리뷰: D — 리뷰부스터 v2 DDL + 이관계획 (교차 검토, 수정 없음)

문서만 믿지 않고 전부 다시 돌려봤다. 일회용 postgres:16에서 5개 SQL을 헤더 순서대로 적용(무오류·9객체), 가드 전부 찔러봄(활성 2건 거부/비활성 다건 허용, 멱등키 중복 거부, BLOCKED 통과, enum에 없는 'SENT'는 CHECK 거부, KST 경계 8/31 23:59:59→m08·9/1 00:00→m09·2027-05→default), 백필 2회 실행 멱등(INSERT 0 0)과 최신 id 시드·EXPIRED/store 0 제외, 구버전 기본형 테이블에 방어 ALTER 경로까지 별도 DB로 재현했고, gradle 테스트도 재실행했다(tests=1 failures=0). 스키마 대조도 운영 DDL 문서 3종·구 엔티티 6종·enum 7종을 origin/main에서 실측했는데 전 컬럼/타입/nullable/인덱스가 일치한다. CHECK의 BLOCKED 추가는 blocked() 팩토리(KakaoAlimTalkSendHistory.kt)가 실제로 BLOCKED+`blocked-<ULID>`를 기록하므로 필수였음이 실증됐고, 배치 위치·Testcontainers가 배포 원본을 classpath로 읽는 관례(shedlock 선례)도 맞다. 그 위에서 지적 하나가 크다.

## 지적 1 — [높음] 백필 §2의 전제가 틀렸다: boost_message_delivery는 07-13 스냅샷에 없다

`reviewboost-v2-backfill.sql:63`은 "boost_message_delivery 는 v1 테이블이라 07-13 스냅샷에 존재한다"고 단언하는데, medilawyer-boot 실측으로는 이 테이블의 엔티티가 **2026-07-30 신설**이다(커밋 469f7ba4 "boost_message_delivery 엔티티 및 Repository 추가" — v2 발송본체 작업의 일부다). 스냅샷(07-13)보다 17일 뒤다. 로컬의 스냅샷 계열 DB(mb-booster-pg-new-1)에 to_regclass로 확인해도 `boost_present=true, boost_present_order=true, boost_message_delivery=false`로 파일들 자신의 전제("07-30~08-05 신설분은 없다")와 정확히 같은 패턴이다.

결과: 실제 개발계 PG에서 이 스크립트를 돌리면 §2 사전확인(:67)에서 `relation does not exist`로 죽는다. 파일에 트랜잭션 래핑이 없어 §1(AC8 핵심)은 자동커밋으로 이미 적용된 뒤라 데이터 사고는 아니지만, "R8 원자 적용" 절차서가 에러로 종료되는 건 운영자 입장에서 사고다. 더 나쁜 건 **픽스처가 이 오류를 가린다**는 점 — `reviewboost-v2-snapshot-fixture.sql:34-35`가 이 테이블을 "스냅샷 존재" 명목으로 만들어주기 때문에 테스트는 통과하고 실DB만 실패하는, 대역이 현실과 어긋난 전형적 패턴이다. 참고로 스냅샷에 원천 테이블이 없다는 건 개발계엔 이관할 v1 이력 자체가 없다는 뜻이라, §2는 개발계에서 어차피 no-op이어야 한다.

수정 제안: §2를 `DO $$ ... IF to_regclass('public.boost_message_delivery') IS NOT NULL THEN ... $$` 조건부로 감싸거나 별도 파일로 분리하고, :63 전제 문구·픽스처 헤더·04-changes.md:11 서술을 정정할 것. 덧붙이면 §2(B2C-343 이관)는 R8 원문("B2C-345 데이터 백필")에도 없는 범위 추가라, 남긴다면 그 근거(AC7 멱등 재현용)도 파일에 명시하는 게 맞다.

## 지적 2 — [낮음] 백필 §1의 근거 주석이 사실과 다르다

`reviewboost-v2-backfill.sql:17` "(구 코드도 List.getFirst() 로 1건만 쓰던 동작이라 정보 손실 아님)" — 실측(20b3181 직전 ReviewBoostService.enqueueManualMatchedRequest)은 getFirst()가 아니라 **`activePresents.size() != 1`이면 예외**(13020)였다. 즉 ACTIVE 여러 개인 병원은 구 코드에선 자동발송이 에러(수동 개입 대기)였는데, 이 백필은 그 병원들을 조용히 "최신 선물로 자동발송 가능" 상태로 전환한다. 최신 1건 선택 자체는 합리적 해소라고 보지만(부분 UNIQUE 불변식 준수, 운영 백필 원문도 미보존이라 정답 부재), 돈 경로의 결정이 틀린 근거 위에 서 있으면 안 된다. 주석을 사실대로 고치고, 검증쿼리에 "ACTIVE 복수 보유 병원 목록"을 한 줄 추가해 운영자가 자동 해소된 병원을 눈으로 확인하게 하길 권한다.

## 지적 3 — [낮음] crawling-fail-log.sql의 R# 매핑이 헐겁다

R6의 테이블 열거(01-requirements.md:24)에 crawling_fail_log가 없다. 실제로는 R5 crawling 이관의 선행물이 맞고(구 crawling-service origin/main에 CrawlingFailLogRecorder 실존, 신 booster-app엔 참조 0건 실측 — 코드가 오면 테이블이 필요하다), 04-changes가 R6으로 묶은 것도 이해는 되지만, 고아 변경으로 오독되지 않게 04-changes에 "R5 이관 선행물"임을 한 줄 명시하는 게 좋겠다.

## 지적 4 — [정보] 파티션 자동생성 전제

`crawling-fail-log.sql:17` "crawlDailyReportJob이 다음 3개월치 보장"은 legalcare-batch가 신개발계 booster PG에 붙어 있을 때 얘기다. 미가동이면 2027-01부터 전부 DEFAULT로 수렴한다 — INSERT는 안 깨지니 무장애지만 파티션 DROP 보존 모델은 퇴화한다. 헤더에 "batch 미가동 환경은 월 파티션 수동 추가" 한 줄이면 충분하다. (5개월분만 만든 것 자체는 DEFAULT 안전망이 있어 문제없다고 본다. 파티션 부모의 identity 컬럼도 PG16에서 정상 동작함을 실측했다.)

## 지적 5 — [정보] 사소 2건

`reviewboost-v2-alimtalk.sql:146-151` 방어 ALTER 경로는 delivery_mode에 DEFAULT가 남고 신규 CREATE 경로(:137)는 없다 — 두 경로 end-state가 미세하게 다르다(기능 영향 없음, 실측). 그리고 ReviewBoostV2DdlTest는 §2 멱등을 데이터 있는 상태로는 안 태운다(테스트에선 빈 테이블이라 INSERT 0 0) — 지적 1 수정 시 시드 2행을 넣어 같이 증명하면 좋다.

## 03-design-D.md 판정

이관 계획은 타당하다. 모듈 의존 엣지 주장(notification→product:15, product→crawling:20)은 build.gradle 실측과 일치해 사이클 논리가 성립하고, callbackOrder 멱등가드 부재 주장도 실측과 일치한다(신 앱 ReviewBoostService:119는 잠금 없는 findByUuid, 구 :153은 findByUuidForUpdate — "돈버그라 #6 최우선"이 정확한 우선순위다). R9 게이트웨이 전환을 마지막에 두는 이유("먼저 바꾸면 미구현 booster로 라우팅돼 즉사")와 DDL→백필→배포 순서 강제도 옳다.

정리하면: DDL 5종과 테스트의 품질은 높고 스키마 정합은 전 항목 검증됐다. 다만 지적 1은 R8 절차가 실제 대상 DB에서 에러로 끝나는 문제라 수정 없이 승인할 수 없다. 지적 1(+2의 주석 정정)만 반영되면 바로 승인이다.

VERDICT: REQUEST_CHANGES
