# Round 1 리뷰: 워크스트림 A·B·C (독립 검토, 수정 권한 없음)

문서만 믿지 않고 diff·신규 파일을 직접 읽었고, 핵심 주장은 독립 재실행으로 재검증했다
(terraform validate/fmt, describe-secret, 활성 @Scheduled grep, 로컬 이미지 존재 확인).

## A — HP 시크릿 (R1·R2): 통과

보안 최우선 항목이라 가장 깐깐하게 봤는데 흠잡을 데가 없다. secrets.tf:5-9는
`aws_secretsmanager_secret` 컨테이너만 정의하고 `aws_secretsmanager_secret_version`을 안 썼다 —
version 리소스를 썼다면 실값이 S3 tfstate에 평문으로 남았을 텐데 그 함정을 정확히 피했고,
그 이유를 secrets.tf:1-3 주석과 04-changes에 명시까지 했다. 실값 grep은 내가 다시 돌려도
README.md:21의 `<실값>` 플레이스홀더 1건뿐이다. host/port/user 메타는 git에 있지만 이건
비밀이 아니라 R1이 요구한 메타데이터고 내부망 IP다.

- 시크릿명 `legalcare/onprem/hp-ssh`: batch/shared.tf:23·27·32·36·41의 `legalcare/*` 관례와 일치.
- providers.tf:14 backend key `onprem/terraform.tfstate`: batch(:15)·ecr(:13)·teamcity(:12)·medicontents(:12)와 충돌 없음. 버킷·use_lockfile·encrypt 구성은 batch/providers.tf와 동일.
- README.md: 시크릿명, put-secret-value 주입 절차(쉘 히스토리 대안 포함), apply 수동 — R2 충족.
- `terraform validate` Success + `fmt -check` 클린 (내가 재실행). `describe-secret` → ResourceNotFoundException (내가 재실행 — A3 미존재 실측 재확인, 정의 진행 타당·apply 미실행도 확인됨).
- 로컬 `.terraform/`은 infra/.gitignore:2로 커밋 안 됨. `.terraform.lock.hcl` 커밋은 batch/ecr/teamcity와 같은 관례.

## B — ES 커스텀 이미지 (R3): 통과

Dockerfile:4 `FROM ...elasticsearch:8.15.0`(버전 유지), :7 plugin 3종 `--batch` 설치, :11
`COPY --chown=elasticsearch:root dict/ /usr/share/elasticsearch/config/dict/`. 핵심인 경로 정합을
교차 확인했다: 인덱스 설정의 상대경로는 ES config 디렉토리 기준으로 해석되므로
index_total.json:74·113, index_reviewed.json:74·113, index_autocomplete.json:26의
`dict/userdictionary*.txt`·`dict/synonyms.txt` 참조가 COPY 목적지와 정확히 맞는다.
이미지에는 dict 텍스트 4파일뿐 색인 데이터 없음(별도 복원 방침 준수, 이미지 836MB로도 방증).
`legalcare/elasticsearch:8.15.0-nori` 이미지가 로컬에 실재함도 확인(빌드 주장 물증).
compose es 블록은 env 3종·healthcheck·networks 전부 보존 — 원래 데이터 볼륨은 없던
서비스라 누락된 보존 항목도 없다.

## C — 스케줄러 off (R4): 통과

엄밀 grep(`^\s*@Scheduled`)으로 booster-app 활성 어노테이션 0건. 주석처리는 정확히 5지점 —
EnterpriseScheduler.java:29, HospitalRankingScheduler.java:28, ContractScheduler.java:38·61,
ReviewBoostScheduler.java:29 — 각각 "배치 이관(legalcare-batch 담당)" 사유 주석 동반. diff에
로직 변경은 0(어노테이션 줄 교체+주석 삽입뿐), 메서드·빈·@SchedulerLock 존치, 명시적 제외인
CrawlingScheduler는 미변경. 덤으로 교차 검증: 꺼진 5개 잡 모두 legalcare-batch에 대응
JobConfig가 실재한다(HolidayFetch·DueDateIncrease·EnterpriseDailyReport·HospitalRanking·
BoostPresentStatus) — "배치가 담당" 사유가 실증된다. 주석 전용 변경이라 컴파일 리스크는
사실상 0(unused import는 warning)이고, CI 빌드 미실행 한계는 04-changes가 이미 공개했다.

## orphan_check: 클린

worktree 변경 = java 4 + compose(R3·R4) + 신규 deploy/es(R3), infra 변경 = onprem/ 5파일(R1·R2).
전부 R#에 매핑되고, mldelegator WIP·medicontents/(사용자 소유·untracked) 미접촉.

## 지적사항 (전부 MINOR, 비차단)

1. **MINOR** deploy/es/dict/stopword.txt — 현 인덱스 JSON 어디서도 참조 안 함(불용어는 index_total.json:79 `"stopwords": "_english_"` 인라인). 구 배포 동일 구성 유지 취지면 무해하니 존치 동의, 죽은 파일임만 인지해 둘 것.
2. **MINOR** infra/onprem/secrets.tf:6 — `recovery_window_in_days` 미설정(기본 30일). destroy 후 같은 이름 재생성이 30일 막히니 운영자 참고. apply 범위 밖이라 수정 요구 아님.

두 건 모두 정보성이고 수용 기준·보안·범위 위반은 발견하지 못했다. AC1~AC4 자가 점검 주장을
전수 재검증했고 전부 사실과 일치한다.

VERDICT: APPROVE
