# 운영 프로모션 계획: crawler-worker 3역할 → ECS Fargate 통일

사용자 결정(2026-08-09): **①UA 는 운영 그대로(모바일 135) ②운영 배포는 ECS Fargate 로 통일 ③이름은 새 이름 + 예전 이름 병행**

## 1. 현황 — 왜 "이미지만 올리면 끝"이 아닌가

**(a) 배포 경로가 3갈래다**

| 역할 | 원본 | 운영 배포 방식 | 자원 |
|---|---|---|---|
| thread | crawler | **EC2 에 `docker run`** (GitHub Actions `appleboy/ssh-action`) | EC2 52.78.94.62(사설 172.31.40.145), ECR `crawler-prod` |
| delegate | ai-delegate-crawler | **ECS Fargate** | 클러스터 `medilawyer-prod` · 서비스 `svc-ai-delegate-crawler-prod` · 태스크 `task-ai-delegate-crawler-prod` · 컨테이너 `container-ai-delegate-crawler-prod` · ECR `ai-delegate-crawler-prod` · 2048cpu/4096mem/awsvpc · role `ecsTaskExecutionRole` |
| receipt | pg-receipt-crawler | **배포 워크플로 없음** (`.github/` 부재). 운영 존재 여부 미확인 | — |

**(b) 설정이 이미지에 구워져 있다 — 이게 진짜 걸림돌이다**

운영 워크플로는 빌드 직전 `.env.secret` 을 파일로 만들고, **git HEAD 의 Dockerfile 이 `COPY . .`** 로 그걸 통째로 이미지에 넣는다
(`crawler/Dockerfile:25`·`ai-delegate-crawler/Dockerfile:29` — git HEAD 기준). 앱은 `load_dotenv()` 로 그 파일을 읽는다.

**통합 이미지는 allowlist COPY 라 `.env` 계열이 0건이다**(실측 확인). 그대로 운영에 올리면 `load_dotenv()` 가
읽을 파일이 없고, **예외 없이 조용히 코드 기본값으로 떨어진다.** thread 의 코드 기본값은
`ENABLE_EXTERNAL_ACTIONS=true` · `ELASTIC_APM_ENABLED=true`(주소 하드코딩) 이라 그대로 뜨면 의도치 않게 동작한다.

→ **설정을 ECS 태스크 정의의 `environment`/`secrets` 로 옮기는 것이 프로모션의 필수 선행 작업이다.**

**(c) 운영은 8/6 안전화 작업분을 아직 못 받았다**

운영 빌드는 `actions/checkout` → **git HEAD** 기준인데, 원본 저장소의 8/6 안전화 변경(crawler 11건·ai-delegate 34건·
pg-receipt 9건)이 **미커밋**이다. 통합본은 그 작업트리에서 복사됐다. 즉 **통합본 = 안전화 반영본**,
**운영 현재 = 안전화 이전본**이다. 프로모션은 통합만이 아니라 안전화 반영도 동시에 나가는 일이 된다.

## 2. 목표 상태

```
ECR: crawler-worker-prod  (저장소 1개, 이미지 1개)
   └ ECS 클러스터 medilawyer-prod
        ├ svc-crawler-worker-thread    CRAWLER_ROLE=thread    :42030  → ALB 타깃그룹
        ├ svc-crawler-worker-delegate  CRAWLER_ROLE=delegate  :42010  → 기존 prod-delegate ALB 타깃그룹
        └ svc-crawler-worker-receipt   CRAWLER_ROLE=receipt   포트없음 (ALB 불필요)
```
- 이미지·태그는 **셋이 동일**. 다른 건 `CRAWLER_ROLE` 과 역할별 env 뿐이다.
- 예전 이름은 **DNS·타깃그룹 수준에서만** 유지한다(`prod-delegate.medilawyer.co.kr` 그대로).
- thread 는 지금 이름이 없다(`52.78.94.62:42030` 하드코딩) → **이름을 새로 붙인다**(예: `prod-crawler.medilawyer.co.kr`).
  붙인 뒤 `booster-app` 의 `config/crawling.yml:103`·`config/user.yml:19` 를 env 인디렉션으로 바꾼다.

## 3. 필요한 산출물

| # | 산출물 | 비고 |
|---|---|---|
| P1 | ECR 저장소 `crawler-worker-prod` | `infra/ecr` 테라폼에 추가. 수명주기 정책은 기존 prod 규칙(keep 20) 따름 |
| P2 | 태스크 정의 3개 | 이미지 동일, `CRAWLER_ROLE` 과 env 만 다름. `ecsTaskExecutionRole` 재사용 |
| P3 | **설정 이관표** | `.env`/`.env.prod` 의 평문 키(thread 22 · delegate 25 · receipt 4)는 `environment` 로, `.env.secret`·Vault 토큰은 **Secrets Manager → `secrets`** 로 |
| P4 | ECS 서비스 3개 | delegate 는 기존 타깃그룹 승계, thread 는 신규 타깃그룹, receipt 는 LB 없음 |
| P5 | ALB 리스너 규칙 · 타깃그룹 | thread 신규. delegate 는 기존 것을 새 서비스로 갈아끼움 |
| P6 | Route53 레코드 | thread 용 신규 이름. `infra/dns` 테라폼에 추가(이미 있는 스택) |
| P7 | 배포 워크플로 1개 | 저장소 1개 → 이미지 1개 빌드 → 서비스 3개 갱신. 기존 워크플로 3개(중 2개)는 폐기 |
| P8 | 원본 저장소 선커밋 | 8/6 안전화 미커밋분을 커밋해야 "운영과 같은 코드"가 성립 |

## 4. 순서 (되돌릴 수 있는 단위로)

1. **선행 조사** — AWS 자격정보 갱신 후 실측: receipt 가 운영에 떠 있나 · 현 ECS 서비스/타깃그룹 상태 ·
   `.env.secret` 실제 키 목록(현 태스크는 `environment` 0건 `secrets` 0건이라 전부 이미지 안에 있다)
2. **P8 원본 선커밋** — 코드 기준선을 고정한다. 이걸 안 하면 "무엇을 배포했는지" 재현이 안 된다
3. **P3 설정 이관표 작성 → Secrets Manager 등록** — 값 이동만, 배포 없음
4. **P1 ECR + P2 태스크정의** — 만들기만 하고 서비스에 안 붙임
5. **receipt 부터 올린다** — LB 가 없어 되돌리기가 가장 쉽고, 실사용량이 가장 적다(개발계 누적 KCS 24·KSTA 1)
6. **thread** — 신규 타깃그룹·DNS 를 먼저 만들고, EC2 와 **병행 가동**하다 호출자(`booster-app` 설정)를 옮긴다
7. **delegate** — 기존 타깃그룹을 새 서비스로 교체. 운영 HTTP 90일 1,014건이라 영향 창이 좁다
8. **EC2 정리** — thread 안정 확인 후

각 단계는 **직전 상태로 되돌릴 수 있어야 한다**. ECS 는 이전 태스크 정의 리비전으로 서비스 업데이트하면 롤백된다.

## 5. 검증

개발계에서 이미 확인된 것(라우트 동등·테스트·카프카 왕복·크로미움 실구동)은 재사용한다. 운영에서 추가로 볼 것:
- **설정이 실제로 주입됐는지** — 각 역할 기동 직후 `/health` 응답의 플래그가 의도한 값인지
  (thread `{kafkaConsumerEnabled, externalActionsEnabled}` · delegate `{..., eurekaEnabled, enabledCrawlingChannels}`)
- **카프카 그룹 참여** — `dev-` 접두사가 `prod-` 로 바뀐 그룹에 정상 참여하는지. **오프셋 초기화 주의**
- **receipt 는 healthcheck 로 컨슈머 생존을 못 본다** — 브로커에서 그룹 상태를 직접 확인(README 절차)
- APM 에 `crawler` 서비스 트랜잭션이 계속 들어오는지(현재 90일 75건이라 표본이 적다)

## 6. 위험

| 위험 | 완화 |
|---|---|
| 설정 누락 → 코드 기본값으로 조용히 동작 | P3 이관표를 **키 단위로 대조**. 기동 후 `/health` 플래그 확인 |
| 카프카 그룹명 변경 시 오프셋 초기화 → 재처리 | 그룹명 **변경 금지**. 현행 값 그대로 승계 |
| 8/6 안전화분이 운영에 처음 나감 | 통합과 안전화를 **한 번에** 내보내지 말고, P8 선커밋 후 안전화만 먼저 현 방식으로 배포하는 선택지 검토 |
| receipt 가 운영에 없는데 새로 띄움 | 선행 조사에서 확정. 없었다면 "신규 기동"이지 "교체"가 아니다 |
| 이미지 3.14GB → Fargate pull 시간 | `--only-shell` 로 594MB 감축 가능(재검증 필요) |
