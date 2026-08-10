# 최종 보고: 운영 동기화 후속 A~D (prod-sync-followups)

**상태: MERGE_READY (A·B·C·D-DDL)** · **D 코드본체 = STAGED(계획완료)**
구성: writer=claude · reviewer=claude(셀프리뷰 단독) · mode=review · rounds=2 · Codex 미사용 · RISK: HIGH · 개발계
빌드 환경: 이 맥에서 gradle 빌드 가능 확인(`./gradlew :app:test` BUILD SUCCESSFUL) — 애초 "빌드 불가" 전제는 오판.

## 변경(커밋, push 안 함)
| 워크스트림 | 파일 | 커밋 | 검증 |
|---|---|---|---|
| C 스케줄러 off | booster chrono 3 + product 1 (@Scheduled 5지점) | 324f077 | 활성 @Scheduled 0 · :app:test 그린 |
| B ES 이미지 | deploy/es/Dockerfile+dict · docker-compose.yml | 51b364c | 실빌드 _cat/plugins 3종·nori 분석 정상 |
| A HP 시크릿 | infra/onprem/{providers,variables,secrets,README} | 96067b3(infra) | terraform validate·fmt · describe-secret=미존재 · 실값 0 |
| D DDL(R6) | booster db/reviewboost-v2-*.sql(5) + 스모크 테스트·픽스처2 | 691046c | psql(pg16) 실적용 · 부분UNIQUE·CHECK·파티션·백필멱등 · gradle 그린 |
| E 반영 6건 | (앞선 저위험 동기화) | 76894ce~ae59565 | 사용자 승인 |

## 리뷰 지적 → 처리
| # | 지적(심각도) | 처리 |
|---|---|---|
| ABC-1,2 | 미참조 stopword·secret recovery_window(MINOR) | 비차단, 유지 |
| D-1 | 백필 §2가 boost_message_delivery를 v1로 오인 → 실DB 실패(HIGH) | DO 가드(to_regclass 스킵)+픽스처 분리로 거짓그린 제거, 재검증 |
| D-2 | §1 근거 주석 오류(LOW) | 실동작 정정 + 다중ACTIVE 노출쿼리 추가 |

## 이행 점검 (R#/AC#)
| R# | 상태 | 구현/검증 |
|---|---|---|
| R1 HP시크릿 정의 | ✅ | infra/onprem/secrets.tf(컨테이너만) · validate · 미존재 실측 |
| R2 README | ✅ | infra/onprem/README.md 주입절차·용도 명시 |
| R3 ES 커스텀이미지 | ✅ | Dockerfile+dict · 실빌드 플러그인 3종 |
| R4 스케줄러 off | ✅ | 5지점 주석 · 활성 0 · 배치 대응잡 실재 |
| R5 v2 코드 이관 | 🟠 STAGED | 03-design-D.md 파일단위 계획(7단계·의존순서). 본체 미착수 |
| R6 신규 DDL | ✅ | 5 SQL + 스모크, psql·gradle 검증 |
| R7 돈/멱등 가드 | 🟠 STAGED | DDL 지원(send_history 멱등 UNIQUE·blocklist). callbackOrder 가드=이관 1순위(계획) |
| R8 백필 | ✅(DDL) / 🟠 원자적용 | 백필 SQL 멱등·안전스킵 검증. R5와 원자 적용은 코드이관 시 |
| R9 /api/v2 라우트 | ✅(현행) | legacy-v2 라우트+테스트(현재 LEGACY 목적지=정합). 이관완료 시 BOOSTER 전환(최후) |

## QA / 영역 검토
- 별도 team-qa 미기동: 구현+독립 셀프리뷰 두 단계가 각각 **실제 명령 실행 검증**(terraform validate, docker build+_cat/plugins, gradle :app:test, psql DDL 실적용) 수행 → QA급 커버. 미구현 D 본체는 QA 대상 없음.

## 남은 리스크 / 판단 필요
- **D 코드본체 이관(R5/R7)**: 대형·머니(알림톡 발송)·5서비스→1앱. 빌드 가능 확인됐으니 03-design-D.md 순서대로 별도 실행 권장. 방식 승인 필요(블라인드 dev 이관 vs 단계별 검증 이관).
- A: terraform apply·실값 주입은 운영자 수동(의도).
- 커밋은 격리 브랜치(worktree feature/prod-sync-candidates, infra feature/onprem-hp-ssh-secret). push·머지 안 함.

상세: {TASK_DIR}/00-log.md · 04-changes-abc.md · 04-changes-D-ddl.md · 03-design-D.md · round-1-review-*.md
