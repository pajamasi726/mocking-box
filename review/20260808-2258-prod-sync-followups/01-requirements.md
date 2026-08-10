# 작업지시서: 운영→신개발계 동기화 후속 4종 (prod-sync-followups)

## 배경 및 목표
> HP 접속정보를 시크릿매니저에 없으면 추가해줘(infra 레포의 테라폼으로 작업). 시크릿매니저에 있다고 README 등 레포 어딘가에 적어둬. 2번은 해. 3번은 다 주석처리 해놔 — 신규 서버는 스케줄 서버에서 도는 것 없다(배치가 담당). 4번은 최근 릴리님이 작업한 건데 코드 그대로 옮겨와 — 지금은 마이크로서비스라 여러 개인데 통합앱 1개로 되면 되니까 어렵지 않을 것. 6번은 넣어도 됨. codex 못 쓰니 셀프모드로 리뷰하며 진행. 개발계 대상.

보고서 결정항목 = 워크스트림 A(HP시크릿)·B(ES이미지)·C(스케줄러)·D(v2이관). 상세 분석은 synthesis.md 참조(재탐색 금지). 대상: 개발계.

## 범위
| WS | 포함 | 명시적 제외 |
|---|---|---|
| A | HP SSH 접속정보 시크릿 테라폼 정의(infra) + 문서화 | terraform apply, 비밀번호 실값 커밋 |
| B | ES 커스텀 이미지(플러그인 3종+한국어 사전) + compose 교체 | ES 버전 변경(8.15.0 유지) |
| C | booster-app 활성 @Scheduled 5지점 주석처리 | CrawlingScheduler(@Scheduled 아님·동적 디스패처), 스케줄러 클래스 삭제 |
| D | 리뷰부스터 알림톡 v2 전량을 통합앱 이관(발송본체 포함) | 운영 배포, mldelegator WIP 구역(booster/post/.../client/mldelegator/) 접근, 저위험 6종 기적용분 재작업 |

## 기능 요구사항 / 수용 기준
| R# | 요구사항 | AC# | 수용 기준 |
|---|---|---|---|
| R1 | HP(host 192.168.1.12, port 9955, user deployer, password) SSH 접속정보를 `legalcare/*` 네이밍 시크릿으로 infra 테라폼에 정의(실값은 TF 밖 주입) | AC1 | `terraform validate` 통과 + `aws secretsmanager describe-secret`으로 기존재 여부 확인 기록 + 레포 grep에 비밀번호 실값 0건 |
| R2 | infra README에 시크릿명과 "실값은 콘솔/CLI 주입" 명시 | AC2 | README에 해당 절 존재 |
| R3 | es 서비스가 nori+kuromoji+smartcn 플러그인·한국어 사전을 이미지에 구운 커스텀 이미지 사용(Dockerfile: FROM elasticsearch:8.15.0 + plugin install + dict COPY) | AC3 | 컨테이너 삭제·재생성 후 `_cat/plugins`에 3종 표시 + nori 사전 분석 쿼리 정상 |
| R4 | @Scheduled 5지점(EnterpriseScheduler:28, HospitalRankingScheduler:27, ContractScheduler:37·59, ReviewBoostScheduler:28) 주석처리 + "배치 이관" 사유 주석 | AC4 | 빌드 통과 + booster-app 기동 시 @Scheduled 등록 잡 0건 |
| R5 | v2 전량(medilawyer-boot 발송본체 15커밋: 멱등키·차단목록·SENS발송·공통이력 / crawling 체인A / product v2 스택 / notification d40664f+a6d5156 / adminweb 배선)을 운영 코드 그대로, booster/reviewed 통합앱 내부 구현으로 이관 — synthesis의 "legacy 존속·4접점만" 전제는 본 지시로 폐기, legacy Feign 경유 제거 | AC5 | 수동(/api/v2/boost1)·자동(크롤 트리거) 발송 플로우가 legacy 서버 없이 게이트웨이+통합앱만으로 개발계 end-to-end 동작(실발송은 A2 게이트 하에) |
| R6 | 신규 DDL 작성·개발계 적용: boost_review_match_analysis, boost_present_send_request, boost_review_platform_click, boost_store_present_setting, kakao_alimtalk_send_history, kakao_alimtalk_blocklist, boost_review_request_setting(+3컬럼), boost_present_order ALTER (부분 UNIQUE=PG 문법) | AC6 | 개발계 PG 적용 + testcontainers 스키마 동봉으로 관련 테스트 유효 |
| R7 | 돈·멱등 안전장치 동반 이관: SHA256 멱등키(review-boost-auto-*), callbackOrder 멱등가드(82a6fcd), 수신차단목록 조회 | AC7 | 중복/역순 콜백·중복 자동발송 테스트에서 원장 이중기록 0건 |
| R8 | B2C-345 데이터 백필을 개발계 DB에 R5와 원자 적용 | AC8 | 백필 후 기존 병원의 선물발송·조회 정상(미백필 시 전면중단 회귀 없음) |
| R9 | 게이트웨이 /api/v2 라우트를 이관된 통합앱 목적지로 연결 + RouteTable 테스트 | AC9 | /api/v2/boost1 매칭 테스트 통과 |

## 기술 제약
- infra 관례: 시크릿명 `legalcare/*`, 계정 074185958044, ap-northeast-2, S3 백엔드(batch/providers.tf) — "실값은 SM 소유, TF는 정의/참조만"(batch/shared.tf:20,43 주석). git/tfstate에 실값 금지.
- D 적응이식 필수 3건(synthesis): NaverBlogHandler 문자복사 금지(신 모듈 de-Feign → LegalCareException 기반), notification 엔티티/빈 리네임(user 모듈 충돌), `review-boost.common-message-template-id` @Value 기본값 없음 → config env 필수(미설정 시 부팅실패).
- B/C/D는 worktree `/Users/steve/steve/legalcare-renew-prodsync-wt`(branch feature/prod-sync-candidates)에서만, A는 `/Users/steve/steve/infra`. codex 불가 → 워크스트림별 커밋 전 셀프 리뷰 기록 남길 것.

## 가정
- A1. 한국어 사전 파일은 DELL 기동 중 ES 컨테이너 또는 구 배포 구성에서 추출 가능.
- A2. 개발계 알림톡/SENS는 실사용자 발송 차단 설정(스텁·테스트 채널)으로 구동하며 QA가 게이트를 검증한다.
- A3. AWS SM에 HP 시크릿 미존재(infra TF 내 흔적 0건 실측) — AC1 확인 결과 기존재 시 정의 생략하고 R2 문서화만 수행.

## 리스크
RISK: HIGH — D가 알림톡 발송(비용)·결제 콜백 멱등·신규 DDL·백필을 동반하고 A가 인증정보를 취급.
- 1순위 개발계 실발송 사고 → A2 게이트를 QA 최우선 시나리오로. 2순위 미백필 배포(선물발송 전면중단) → R8 원자 적용. 3순위 실값 유출 → AC1 grep 검증.

## 관련 파일
| 경로 | 설명 |
|---|---|
| /private/tmp/claude-501/-Users-steve-steve-mocking-box/8f23c7f7-7c11-47e6-b28c-5c139f6de797/scratchpad/synthesis.md | D 커밋해시·경로·이슈 총람(medilawyer-boot/product/crawling/notification/reviewed-admin-web 섹션) — 개발자 필독 |
| /Users/steve/steve/infra/batch/shared.tf, providers.tf, README.md | A: 시크릿 네이밍·백엔드·문서 관례 원본 |
| /Users/steve/steve/legalcare-renew-prodsync-wt/docker-compose.yml:70-74 | B: es 서비스(바닐라 8.15.0) 교체 지점 |
| /Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app/modules/chrono/src/main/java/legalcare/medilawyer/chrono/scheduler/{Enterprise,HospitalRanking,Contract}Scheduler.java, .../modules/product/src/main/java/legalcare/medilawyer/product/scheduler/ReviewBoostScheduler.java | C: 주석처리 대상 4파일 5지점(실측 일치) |
| /Users/steve/steve/legalcare-consulting/_analysis/legalcare-map/08-신개발계/prod-code-sync-report-2026-08-08.html | 결정항목 원 근거 보고서 |

## 의도 확인
재진술: "보고서 결정항목 중 HP시크릿(테라폼 정의+문서화)·ES커스텀이미지·스케줄러 전면 off·리뷰부스터 v2 통합앱 이관을 개발계에 셀프리뷰로 적용하라." 이 지시서는 이를 A~D/R1~R9로 계약화했고, 원문 제약(apply 없이 정의만·주석처리 방식·"코드 그대로+통합앱 1개 구조")을 각 R에 그대로 반영했다. 특히 R5는 보고서의 "legacy 존속" 원계획을 사용자 지시대로 뒤집었음을 명시해 개발자 혼선을 차단한다.

## OPEN QUESTIONS
없음
