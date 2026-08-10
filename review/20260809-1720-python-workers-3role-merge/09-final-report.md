# 최종 보고: 파이썬 크롤러 워커 3역할 통합 (브라우저 상향 + 개발계 전면 재기동)

**상태: MERGE_READY** (핑퐁 2라운드 · QA 1사이클 · 이행점검 OK)
구성: writer=claude · reviewer=claude(단독 셀프리뷰) · mode=review · rounds=2 · RISK: HIGH
결과물: `/Users/steve/steve/legal-care/crawler-worker` — **커밋 안 함**(사용자 확인 후 결정)
개발계: **이미 신규 스펙으로 교체 완료**, 3역할 healthy, 공용망 28컨테이너 전부 running

## 무엇이 달라졌나

| | 전 | 후 |
|---|---|---|
| 저장소 | 3개 | **1개** (`crawler-worker`) |
| 이미지 | 3개 | **1개** (`crawler-worker:dev-3role`, 역할 3개) |
| playwright | 1.42.0 / 1.42.0 / 1.55.0 | **1.55.0 통일** |
| pydantic | 2.9.2 / 2.9.2 / 2.12.0 | **2.12.0 통일** |
| 이미지 크기 | (합계) | 3.14GB (상향 전 2.51GB) |

## 변경 파일
| 파일 | 변경 내용 | R# |
|---|---|---|
| `roles/receipt/` (17파일 신규) | pg-receipt-crawler 무수정 복제(sha256 동일) | R1 |
| `docker/entrypoint.sh` | receipt 분기 → `python worker.py`(포트 없음). 오타 역할 exit 64 유지 | R2 |
| `Dockerfile` | receipt allowlist COPY · 베이스 다이제스트 고정(`@sha256:229a2c5b…`) | R1·R2·R3 |
| `requirements.txt` | 3-way union. 충돌 18건 해소(상향 7 · 하위핀 유지 12) | R3 |
| `docker-compose.dev.yml` | receipt 서비스(격리 토픽·그룹) + 3역할 healthcheck | R5 |
| `docker-compose.cutover.yml` | receipt 서비스 · 별칭 `pg-receipt-worker` · 교체 대상 env 승계 | R5·R7 |
| `README.md` · `.env.example` | 3역할 구조 · 교체/롤백 절차 · 의존성 결정 근거 | R7 |

**R4 발동 없음** — playwright 1.55 에서 세 역할 소스가 한 줄도 깨지지 않아 `roles/*` 무수정 원칙 유지.

## 리뷰 지적 → 처리 (1라운드 11건 · 2라운드 2건)
| # | 지적 (심각도) | 처리 |
|---|---|---|
| 1 | **receipt healthcheck 가 항상 통과** — 검사 프로세스 자기 cmdline 에 `worker.py` 문자열이 있어 자기를 발견 (MAJOR) | 반영. 이름매칭 폐기 → `ppid==1`(tini 가 exec 한 자식) + 브로커 ESTABLISHED 소켓. 워커 죽인 상태에서 실제 실패 실증 |
| 2 | 낮춘 의존성 12건이 한 줄도 안 돌았다 (MAJOR) | 반영. stdlib 스텁 A/B → **"표면 동일" 주장이 3건에서 틀림**(boto3 무결성헤더 · UA · certifi 루트 7개) |
| 3 | receipt 가 Debian 12→13 이동인데 미기재 (MAJOR) | 반영 + 범위 정정(receipt 한정). 베이스 다이제스트 고정 |
| 4~11 | 문서 근거 오류·OpenAPI 한 줄 차이·DLQ 왕복 근거 등 (MINOR 8) | 반영 6 · 부분반박 2(둘 다 근거 타당 확인) |
| 2R-F1 | 브로커 45초+ 중단 시 워커가 멀쩡해도 unhealthy 로 보이는 점 미기재 (MINOR) | 반영(README) |
| 2R-F2 | `04-changes.md` 표에 반증된 문구 잔존 (MINOR) | 반영 |

**2라운드 VERDICT: APPROVE** — healthcheck 새 구현에 구멍 없음(검사 프로세스가 `ppid==1` 필터에 구조적으로 안 걸림), 다이제스트 실재 확인, 범위 이탈 0건.

## QA
**29 시나리오 / 약 150 체크 — QA_VERDICT: PASS.** AC1~AC7 전부 PASS.
- QA 가 크롤러 소스를 읽고 **실제 쓰는 playwright API 24종만 골라 하니스를 새로 작성** → 3역할×24 = 72/72 PASS
- AC5 는 정상 1건 + 위반 1건 양쪽으로 돌려 탐지력 확보(completed+1/DLQ+0, DLQ+1/completed+0)
- 결함 3건 전부 MINOR (D1 healthcheck 망단절 시 60초+ 지연 · D2 UA · D3 덤프 로그 혼입)

## 이행 점검
| R#/AC# | 구현 위치 | 검증 위치 |
|---|---|---|
| R1/AC-공통 | `roles/receipt/` | 양방향 sha256 3역할 0 불일치(작성자·오케스트레이터·QA 3중) |
| R2 | `docker/entrypoint.sh` | 3역할 기동 + 오타 역할 exit 64 |
| R3/AC1 | `requirements.txt` | 이미지 `pip show` 1.55.0/2.12.0 · chromium 140 실구동 72/72 |
| R4 | (미발동) | 1.55 import 감사 회귀 0건 |
| R5/AC5 | compose 2종 | 카프카 왕복 정상·위반 양방향 |
| R6/AC6 | 별도 compose 프로젝트 | 공용망 28컨테이너 전부 running(교체 전후) |
| R7/AC7 | `00-baseline.md`(168줄) | 구 컨테이너·이미지 ID 3/3 일치, mounts 0, 롤백 왕복 1회 성공 |
| AC2 | allowlist COPY · cwd=역할디렉토리 | 통합 전 원본 이미지와 A/B: thread 14=14 · delegate 21=21 · receipt 0 |
| AC3 | — | delegate 38=38 · receipt 4=4 |
| AC4 | `docker-compose.cutover.yml` | 3 healthy / 3 exited · 별칭 3종이 각각 신규 IP 하나만 · env 차집합 0 |

## 남은 리스크 / 판단 필요

1. **[판단 필요] 영수증 크롤러의 브라우저 UA 가 바뀐다.** 세 역할 중 **receipt 만** UA 를 실제 브라우저에 물린다
   (`crawlers/crawling_utils.py:26` → `browser.new_context(user_agent=...)`). thread/delegate 는 HTTP 헤더용이라 무관.
   - 운영 현재(fake-useragent 2.2.0): `Chrome/135 **Mobile** Safari` (안드로이드)
   - 통합본(1.4.0, union 이 thread/delegate 핀 유지): `Chrome/117 Safari` (데스크톱 X11)
   - **국세청·KSTA 가 보는 값이 달라진다.** 접속 금지 제약상 어느 쪽을 받아주는지 검증 불가.
   - 2.2.0 으로 올리면 운영 동작이 보존되지만 thread/delegate 의 `ua.random` 풀도 같이 바뀐다.
2. **[인수인계] `/mnt/ex_disk1/renew-replay/docker-compose.python.yml` 이 구 서비스 3개를 아직 정의한다.**
   구 컨테이너가 `exited`+`unless-stopped` 로 남아 있어 **`docker start` 하나로도** 별칭 이중등록 + 컨슈머 그룹 동시참여가 된다.
   저장소 밖 파일이라 이번 변경으로는 못 막는다. 롤백이 그 컨테이너 생존에 의존하므로 지금 지울 수도 없다.
3. **미검증 구간이 한 곳에 몰려 있다.** `ENABLE_EXTERNAL_ACTIONS=true` 를 켜는 순간 처음 도는 코드가
   실사이트 셀렉터 · boto3 무결성헤더(1.40→1.34) · certifi 루트 CA(-7) · UA 핑거프린트인데,
   **네 가지가 전부 receipt 크롤러 + S3 업로드 한 지점에 몰려 있다.**
4. **운영 프로모션은 이미지만으로 안 된다.** 셋의 운영 배포 경로가 다르다 — thread=EC2 `docker run`(GitHub Actions ssh),
   delegate=ECS Fargate(task def + service), receipt=**배포 워크플로 자체가 없음**. 별도 작업 필요.
5. 이미지 3.14GB(+630MB). playwright 1.55 가 chromium 과 headless-shell 을 둘 다 깔아서다.
   실측상 **실제 도는 건 headless-shell 뿐**이라 `--only-shell` 로 594MB 감축 가능 — 재검증 필요해 후속으로 남김.
6. 검증 전용 토픽 4종(`legalcare.test.3role.receipt.*`)을 개발계 브로커에 남겨 뒀다(`docker-compose.dev.yml` 이 참조).

상세: `00-log.md` · `01-requirements.md` · `04-changes.md` · `00-baseline.md` · `round-1-review.md` · `round-1-response.md` · `round-2-review.md` · `07-qa-report.md`
