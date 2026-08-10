# 팀 작업 로그 — 리뷰부스터 v2 코드 이관 (릴리님 PR → 통합앱)

**사용자 원문 요청**
> 음? 코드가 대형 알림톡? 인건가? 알림톡 부분만 새로들어간거 아니야? 그거 PR만 가져와서 옮기면 되잖아. 검토하고 바로 착수해서, 개발계테스트까지 진행해. 배치는 이미 되어져 있는거로 알아. 코덱스 없으니 동일하게 싱글로.

**구성**: writer=claude · reviewer=claude(셀프리뷰 단독) · mode=review · rounds=2 · Codex 미사용 · 개발계 대상

**실측 규모 (스냅샷 2026-07-13 → origin/main)**
| 서비스 | prod(java/kotlin) | test | other |
|---|---|---|---|
| medilawyer-boot | 58f +2787 | 20f +3241 | 4f +90 |
| product-service | 56f +1694 | 15f +1743 | 1f +1 |
| crawling-service | 10f +558 | 4f +551 | 1f +0 |
| notification-service | 14f +463 | 1f +237 | - |
| reviewed-admin-web-service | 41f +852 | 5f +1072 | 18f +2656 |
| **합계** | **179f +6354** | 45f +6844 | 24f +2747 |

→ 사용자 판단 타당: 프로덕션 6.3천줄 규모, "PR 옮기기"로 착수 가능. 이전 턴의 "대형" 표현은 과했음(오케스트레이터 정정).

**선행 완료**: DDL/백필/스모크 = 이전 태스크(20260808-2258)에서 커밋 691046c. 이관계획 03-design-D.md 존재.
**배치**: 사용자 확인 — 이미 되어 있음(신 legalcare-batch). 배치 잡 신규 작성 불필요.

## $(date +%H:%M) 오케스트레이터 → 준비
- 규모 실측 후 착수 결정. PM 호출.

## 09:00 team-pm → 오케스트레이터
- 01-requirements.md: R1~R9/AC1~AC9. RISK HIGH. OPEN QUESTIONS 없음.
- 이관 6단계(R1 product+callbackOrder가드 → R2 발송본체(notification) → R3 조회측 → R4 crawling 자동발송 → R5 adminweb 서버측 → R6 게이트웨이 BOOSTER 전환).
- R7 돈/안전 횡단(멱등가드·SHA256키·수신차단). R8 개발계 증거 6항목. R9 스텁·발송OFF를 R2 선행으로 강제.
- adminweb 화면자산(Thymeleaf/JS)은 별도 태스크 분리. 배치 신규작성 제외(이미 있음).
- 오케스트레이터 자가점검: 사용자 의도("PR 옮기고 개발계 확인, 발송사고 없이")와 일치 OK.

## 09:02 오케스트레이터 → team-dev
- R9(스텁 선행) + R1(product v2 + callbackOrder 가드) 구현 지시. 단계별 빌드 검증.

## 09:35 team-dev(IMPLEMENT) → R9+R1 완료
- R9: 개발계 발송 봉인 — compose 렌더로 NAVER_CLOUD_* → stub:8099 확인. **application-dev.yml 이 실 SENS 엔드포인트를 기본값으로 갖고 있던 것 발견·도달불가로 변경**(env 없이 뜨면 과금발송 가능했음). common-message-template-id 부팅실패 재현 후 해소.
- R1: product v2 스택 82파일(+3613/-466) 이관. 차분 패키지 리매퍼로 변환, 거부 훅 5개만 수동 병합(신앱 적응 보존: productDomainService·SchedulerLock·deleted=false).
- R7①: callbackOrder 가드 ReviewBoostService.java:164-171 findByUuidForUpdate(PESSIMISTIC_WRITE)+최종상태 no-op, 원장 existsByOrderId 2차방어. **역순 콜백 테스트 추가**(구 테스트는 중복만 커버).
- 빌드 중 실결함 2건 수정: legacyHandler 빈 충돌(crawling과 동명)→productLegacyHandler, UserProductLocalAdapter 컴파일 깨짐.
- 검증: `./gradlew clean test` BUILD SUCCESSFUL, **443 tests / 0 fail**.
- 인수인계: ①review-boost.send.enabled 는 아직 소비 코드 없음(R2 어댑터가 읽을 자리) — R4 발송 ON 시 release 승격 필요, 누락 시 운영 조용히 발송정지 ②R2 이관 시 구 Kotlin이 phone 원문 로깅 → 마스킹 필수.
- STATUS: DONE. 산출물 04-changes-r1.md

## 10:05 team-dev(CROSS_REVIEW) → R9+R1 셀프리뷰
- VERDICT: REQUEST_CHANGES. **[CRITICAL] 카카오 선물발송(진짜 돈)이 운영 API 하드코딩 + 콜백 운영 게이트웨이** — R9는 SENS만 봉인. 지금 안 나가는 이유가 auth placeholder 401뿐이라 R8 중 토큰 채우면 실구매.
- MAJOR3: deploy/dev compose 봉인 미적용·application-dev.yml 평문 NCP키, dev-placeholder가 fail-fast 무력화(release 재정의 0), 부팅게이트가 local만 검증.
- 빌드 재검증: :app:test 가 UP-TO-DATE 스킵되던 것 발견 → cleanTest 강제 443 그린 확인.

## 10:25 team-dev(수정) → R9+R1 대응
- #1 반영: 카카오 선물 API·콜백 env 인디렉션화, 기본값 도달불가. env명을 legalcare-batch 의 KAKAO_BIZ_* 와 통일. 콜백은 booster 자신.
- #2 반영: deploy/dev compose 봉인 8줄(스텁 부재라 fail-closed :19999), application-dev.yml 평문 NCP키 제거·DEV_STUB_BASE 단일파생.
- #3 반영+부분반박: dev-placeholder 30개 전수조사 → 발송·결제 13개만 release fail-fast, 나머지 17개(Slack/SES/Gemini/Airtable)는 범위 밖이라 후속 권고.
- #4 반영: ExternalSendSealingTest 16케이스(local/dev/release 파라미터), release는 env 없으면 IllegalArgumentException 단언.
- MINOR 배치 ntruss 반박(이미 env 인디렉션+STUB_BASE, deploy/dev에 batch 없음, 잡 OFF).
- 검증: cleanTest test **459 tests / 0 fail**. **뮤테이션 확인**: product.yml 하드코딩 복원 시 local·dev FAILED, 복구 시 SUCCESSFUL.
- 되돌림 2건: ①release fail-fast 가 배포 깨뜨릴 수 있음 — 운영 태스크에 13 env 실재 확인 필요. 지금껏 조용히 떴다면 **운영 알림톡이 placeholder 주소로 나갔을 가능성 미확인** ②제거한 NCP 키는 git 이력에 남음 → 콘솔 폐기·재발급 필요.
- STATUS: REVISED

## 10:40 오케스트레이터 → 되돌림 검증 (직접)
- 구 운영 config-server release 조회: naverCloud.url / sms.smsUrl 모두 **실 SENS 주소로 정상 주입 중** → "운영이 placeholder로 발송했을 가능성"은 기우로 확인(운영 정상). fail-fast는 신 앱 전용 안전장치이며 배포 시 13 env 준비 필요.
- NCP 키 git 이력 잔존은 사실 → 키 회전 권고(기존 P0 목록에 이미 있음).

## 11:00 오케스트레이터 → 커밋 R9+R1 (5575042)
- 봉인 델타 육안확인(실주소 주석만·값 env화·평문키 제거) 후 커밋. 459 tests 그린.

## 12:10 team-dev(IMPLEMENT) → R2+R3+R7②③ 완료
- 82파일(신규76/수정6). cleanTest test **508 tests / 0 fail** (+49).
- 구현 중 함정 4건 포착·해결: ①WebConfig 프리픽스로 /notification/api/v2 가 될 뻔(R6 전환 즉시 404) → 해당 컨트롤러만 프리픽스 제외 + 경로 테스트 3케이스 ②응답봉투 계약(구 ApiResponse{data} vs 신 ResultResponse{result}) 구 봉투 재현 ③insertCommonIfAbsent 네이티브 INSERT가 NOT NULL 감사컬럼 누락 → CURRENT_TIMESTAMP 추가(DDL 불변) ④builder() 가 SENS 4xx를 예외화해 FAILED/UNKNOWN 분기 역전 → builderWithoutStatusHandler.
- 마스킹: log 3곳, MaskingUtil 패턴 불일치 시 원문반환 구멍 발견 → *** 통째가림 + 원문 미출력 테스트 단언.
- send.enabled 소비 구현(OFF면 RestClient 호출 0, verifyNoInteractions). 기본값 비운영 false / release true(운영 조용한 정지 방지).
- 차단(R7③) 최앞단·BLOCKED 이력, 멱등(R7②) findByIdempotencyKey 3지점+DIVE 흡수+DB UNIQUE.
- 권한 적응이관(AppUser 부재 → UserStoreAuthorityModel), 통과/거절 집합 동일 실측·7케이스 고정.
- 확인요청 1건: 발송 OFF 시 이력 ACCEPTED + provider_id='SEND_DISABLED'(ck_kash_status 8종 고정이라 SKIPPED 불가).
- STATUS: DONE. 산출물 04-changes-r2.md

## 13:30 team-dev(CROSS_REVIEW) → R2+R3 셀프리뷰
- VERDICT: REQUEST_CHANGES. 치명4: ①게이트OFF 2건째 uq_kash_provider_message 위반(AC8 차단) ②게이트OFF=영구폐기(ACCEPTED확정+멱등키소진, 켜도 재발송 불가) ③권한 판정 어긋남(팀미소속 조직ADMIN 거절 / **해제된 팀원 발송 허용** / 판정축 변경) ④에러봉투 미보존(409→200+status:409). 중대1: 멱등 소비측 테스트 0.
- 빌드 재검증 508 그린 일치. 강점: 개발계 SENS 실발송 경로 0, 수신번호 미로깅, 범위밖 변경 0.

## 14:10 team-dev(수정) → R2+R3 대응
- #1#2 재설계: 게이트 판정을 **선점 이전(sendPrepared 최상단)** 으로 이동 → 이력 PENDING·provider id NULL(부분UNIQUE 회피)·RETRYABLE 재개·멱등키 미소진. 실DB 테스트 ReviewBoostSendGateHoldDbTest(보류3건 무충돌/ON시 3건 재선점/**폐기안은 2건째 UNIQUE 위반 회귀고정**).
- #3 권한: UserStoreAuthorityModel 경로 폐기, organization_app_user·team_app_user 직접 조회로 구 로직 조건단위 재현, 판정축 팀 엔티티 복원. 초안의 "집합 동일" 주장 **틀렸다고 정정**.
- #4 에러봉투: 구 GlobalExceptionHandler 실측(업무예외 전부 HTTP 400 + {errors,errorMessage,path,timeStamp}) → 컨트롤러 로컬 @ExceptionHandler 재현, 전역 advice 무변경.
- #5 멱등 소비측 테스트 10케이스 신규, 구 79 중 46 이관·33 미이관 사유표.
- 검증: cleanTest test **531 tests / 0 fail** (+23). 수정 중 자체 테스트 3건 실파손 발견·수정.
- 검토요청: @Valid 실패가 HTTP 500(구 동작 보존). 컷오버 후 400 전환 권고.
- STATUS: REVISED

## 13:10 사용자 → 오케스트레이터 (중요 정정 2건)
- "기존에 이렇게 해서 처리했었는데?" — GW 404/매핑없음 알림 실수신 샘플 제시. "develop 은 env 가 아니라 profile 에 하드코딩이야".
- 조사: 원본 커밋 549f9dd(앱)·569f91a(게이트웨이 RouteMissAlerter)는 **구 서비스별 레포(booster-api/legalcare-gateway develop)에만** 존재, renew 모노레포엔 미이관. 에이전트가 다른 리모트를 안 찾고 새로 구현한 것.
- 오케스트레이터 조치: dev 프로파일 웹훅 하드코딩 3앱 즉시 복원(관례), 에이전트에 원본 이식 지시.

## 13:20 team-dev(수정) → 404 알림 원본 이식
- 자체 구현 16파일 전량 삭제, 원본 이식. RouteMissAlerter **byte-identical**, 앱 3종 isNotifiable404/dedupeKey404 문자단위 동일. 중복 4벌 문제 소멸(앱은 기존 RenewSlackNotifier 사용).
- gateway dev 웹훅 하드코딩(앱 3종과 동일), application.yml 은 빈 기본값(운영 env).
- 실발송 실증(가짜 웹훅): GW 404 1건(id 3건 dedupe)·노이즈 0건, 앱 404 1건·노이즈 0건. 실제 수신 텍스트 확인.
- 테스트: booster 535 / reviewed 239 / lawkit 182 / gateway 76, fail 0.
- 확인필요: 이모지 🔎(소스) vs 🔍(사용자 샘플) 1자 차이, reviewed 라벨 404=reviewed-api / 예외=reviewed-app.
- STATUS: REVISED
