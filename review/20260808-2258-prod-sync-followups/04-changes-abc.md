# 변경 요약: prod-sync 후속 A(HP시크릿)·B(ES이미지)·C(스케줄러 off)

## 변경 파일
| 파일 | 한 줄 요약 | R# |
|---|---|---|
| infra/onprem/providers.tf (신규) | batch 관례 복사, S3 backend key `onprem/terraform.tfstate` | R1 |
| infra/onprem/variables.tf (신규) | region 변수(batch와 동일) — providers.tf가 var.region 참조하므로 추가 | R1 |
| infra/onprem/secrets.tf (신규) | `legalcare/onprem/hp-ssh` 시크릿 **컨테이너만** 정의(실값·version 리소스 없음) | R1 |
| infra/onprem/README.md (신규) | 시크릿명·"실값은 콘솔/CLI 주입"(put-secret-value 예시)·apply 수동 명시 | R2 |
| infra/onprem/.terraform.lock.hcl (신규) | init 산출 lock — batch도 커밋돼 있는 관례라 유지 | R1 |
| deploy/es/Dockerfile (신규, worktree) | ES 8.15.0 + nori·kuromoji·smartcn 설치 + dict/ COPY(chown elasticsearch) | R3 |
| docker-compose.yml (worktree) | es 서비스 `build: ./deploy/es` + `legalcare/elasticsearch:8.15.0-nori` 태그, env·healthcheck 유지 | R3 |
| chrono/.../EnterpriseScheduler.java :28 | @Scheduled 주석처리 + 배치 이관 사유 주석 | R4 |
| chrono/.../HospitalRankingScheduler.java :27 | 〃 | R4 |
| chrono/.../ContractScheduler.java :37·:59 | 〃 (2지점) | R4 |
| product/.../ReviewBoostScheduler.java :28 | 〃 | R4 |

## 검증 결과 (전체 gradle 빌드는 이 환경 불가 — 하단 한계)
- C: `grep -rn "@Scheduled" apps/booster-app/modules/*/src/main/java | grep -v "//"` → **0건**. 주석처리 5건, diff는 4파일+compose뿐.
- B: `docker build` 성공 → 이미지 내 `elasticsearch-plugin list` = 3종 8.15.0, dict 4파일 elasticsearch 소유 확인.
  단독 기동 후 `_cat/plugins` 3종 표시 + `dict/userdictionary.txt` 참조 인덱스 생성·nori 분석 쿼리 정상(사전 로드 실증). `docker compose config` 파싱 정상.
- A: `terraform init -backend=false && validate` **통과**(TF 1.15.7), `fmt -check` 클린.
  `aws secretsmanager describe-secret`(profile legalcare, 계정 074185958044) → **ResourceNotFoundException = 미존재 확인**(가정 A3 실측 일치, 정의 진행 타당).
  `grep -rn password infra/onprem/` → README의 `<실값>` 플레이스홀더 1건뿐, 실값 0건.

## AC 자가 점검
- AC1 ✅ validate 통과 + 미존재 확인 기록 + 실값 grep 0건 (apply는 범위 밖, 운영자 수동)
- AC2 ✅ infra/onprem/README.md에 시크릿명·주입 절차 절 존재
- AC3 ✅(로컬 실증) 재생성 이미지에서 _cat/plugins 3종 + nori 사전 분석 쿼리 정상 — compose 환경 재검증은 QA 몫
- AC4 ✅(정적) 활성 @Scheduled 0건 → 기동 시 등록 잡 0건. 빌드 통과는 CI 확인 필요(주석만이라 컴파일 안전, 린트 플러그인 부재 확인)

## 알려진 한계 / 리뷰어에게
- 색인 데이터(index_autocomplete/index_total)는 이미지에 넣지 않음 — **별도 복원 대상**.
- secret_version 리소스는 의도적으로 미사용: 쓰면 실값이 tfstate(S3)에 남아 기술 제약("git/tfstate 실값 금지") 위반. 컨테이너만 정의가 batch 관례("실값은 SM 소유")와 일치.
- C에서 @SchedulerLock·import는 존치(빈 유지, 재활성화 시 @Scheduled 한 줄 해제만) — unused import는 warning뿐(checkstyle/spotless 없음).

STATUS: DONE
