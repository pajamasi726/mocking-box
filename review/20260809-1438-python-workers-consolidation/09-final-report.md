# 최종 보고: 파이썬 워커 5개 → 3개 통합 (묶음A 실통합 + 묶음C 계획)

**상태: MERGE_READY** (핑퐁 1라운드 · 셀프 크로스리뷰 REQUEST_CHANGES→REVISED · 오케스트레이터 독립 재검증 통과)
구성: writer=claude · reviewer=claude(단독/셀프) · rounds=3(1라운드에 합의) · RISK: MEDIUM
결과물: `legal-care/crawler-worker` (git init + 커밋 `d6aee86`, 190파일, **푸시 안 함**)

## 5개 → 3개, 무엇이 어떻게 되나

| 묶음 | 대상 | 이번 결과 |
|---|---|---|
| A | crawler + ai-delegate-crawler | **실통합 완료** → `crawler-worker` 이미지 1개, 역할 2개 |
| B | pg-receipt-crawler | **단독 유지** — playwright 1.55 vs 1.42, pydantic 2.12 vs 2.9 로 A와 충돌 |
| C | ai-model-serve + embedding-service | **계획서만** (`bundle-c-plan.md`) — python 3.12 vs 3.11, torch 2.5.1 vs 2.6.0 |

묶음A를 한 프로세스로 합치지 않고 **이미지 1개 + 역할 2개**로 간 이유: 두 저장소의 최상위 패키지
이름이 겹치는데(config.py·agents·kafka·routes) 내용이 다르고, 라우트 3개(`/check_id`,
`/kakao_login_2`, `/naver_login_captcha`)가 정면충돌한다. 소스를 한 줄도 고치지 않으려면 이 방식뿐이다.

## 변경 파일
| 파일 | 내용 | R# |
|---|---|---|
| `crawler-worker/Dockerfile` | 단일 이미지. 원본 두 Dockerfile 의 allowlist COPY 를 그대로 옮김 | R1,R4,R7 |
| `crawler-worker/docker/entrypoint.sh` | `CRAWLER_ROLE` 분기, 역할 디렉토리를 cwd 로. 오류 시 exit 64 | R1,R3,R4 |
| `crawler-worker/roles/thread`(72) `roles/delegate`(108) | 원본 무수정 복사(sha256 동일) | R2,R3,R7 |
| `crawler-worker/requirements.txt` | 두 역할 union 128줄, 동일 패키지 상이 버전 0건 | R5 |
| `crawler-worker/docker-compose.dev.yml`, `.env.example` | 개발계 안전 env 를 파일에 고정 | R6,R8 |
| `crawler-worker/README.md`, `.gitignore`, `.dockerignore`, `tools/` | 문서·제외규칙·검증도구 | R1,R6 |
| `{TASK_DIR}/bundle-c-plan.md` | 묶음C 계획(코드 변경 0) | R9 |

## 리뷰 지적 → 처리
| # | 지적 (심각도) | 처리 |
|---|---|---|
| 1 | Kafka 검사가 그룹명 오타 + 다른 브로커 조회로 탐지력 0 (MAJOR) | 반영. 실브로커·실그룹으로 재검사, 역할별 그룹이 서로 다르다는 사실도 이때 발견 |
| 2 | delegate 업무 라우트 11개는 검증된 적 없음 (MAJOR) | 반영. 오프라인 임포트 덤프 + 통합 전 실제 이미지 A/B 로 교체 |
| 3 | Dockerfile 이 allowlist→denylist 로 통제 반전 (MAJOR) | 반영. allowlist 복귀 |
| 4 | 배포 env 산출물 부재 | 반영. compose + .env.example 추가 |
| 5 | psutil 버전·APM 기본값 | 현행 유지(죽은 코드 확인 / R6 계약) |
| 6 | fixture 컨텍스트 .dockerignore, requirements 이중선언 | **반박** — 바이트 동일 불변식·원문 보존 우선. 사유 README 기록 |

## 검증 (전부 개발계 DELL 실측, 통합 전 이미지와 A/B)
| 확인한 것 | 결과 |
|---|---|
| 라우트 집합 | thread 14=14 · delegate 21=21, **완전 동일** (오프라인 덤프, `--network none`) |
| `/health` 응답 | 통합 전 컨테이너와 **바이트 동일** (양 역할) |
| Kafka 미참여 | 두 클러스터 **전체 그룹 전수 스캔**(30+1그룹) — 실멤버 36건은 검출되는데 통합 컨테이너는 0건 |
| 외부 호출 차단 | 업무 라우트 4건 503, established TCP 0건 |
| 이미지 내용물 | thread 68 / delegate 87 파일. fixture·tests·문서·lint 7종 부재 |
| 회귀 테스트 | delegate **38 passed** — 통합 전 원본 이미지와 같은 수 |
| 역할 가드 | `CRAWLER_ROLE` 미설정·오타 모두 exit 64 |
| 원본 저장소 | 5개 전부 무변경(수정 mtime 전부 세션 시작 전), 기존 dev 컨테이너 무접촉 |

## 이행 점검
| R#/AC# | 구현 위치 | 검증 위치 |
|---|---|---|
| R1/AC1 이미지 1개 | `Dockerfile`, `entrypoint.sh` | 빌드 성공 2.51GB, 역할 2개 기동 |
| R2/AC6 원본 무수정 | `roles/**` | sha256 대조 0건 차이, 원본 git status mtime |
| R3/AC2 동시 기동 | `entrypoint.sh`, `docker-compose.dev.yml` | `/health` 200 ×2, 응답 바이트 동일 |
| R4/AC3 라우트 동등 | allowlist COPY, cwd=역할디렉토리 | 오프라인 덤프 A/B diff 0 |
| R5/AC5 버전 | `requirements.txt` | 동일 패키지 상이 `==` 0건, 실측 버전 일치 |
| R6/AC4 외부 무접촉 | compose 안전 env | 503 ×4, TCP 0, Kafka 전수 스캔 0 |
| R7 이미지 위생 | allowlist COPY | fixture/tests/문서 부재 확인 |
| R8 개발계 구동 | `docker-compose.dev.yml` | 컨테이너 2개 1시간+ 정상 유지 |
| R9/AC7 묶음C 계획 | `bundle-c-plan.md` | 버전표·골든셋·6단계 수록, 소스 변경 0 |

## 남은 리스크 / 판단 필요
1. **복사 기준이 git 최신 커밋이 아니라 작업 디렉토리다.** 원본 두 저장소에 8/6 안전화 작업분이
   미커밋으로 남아 있고(11건·34건), 개발계 실행 이미지도 그 상태에서 빌드됐다. 지금은 정합하지만
   나중에 git 에서 다시 뽑으면 다른 코드가 나온다. **원본 저장소를 먼저 커밋해두는 게 안전하다.**
2. **CI/배포 파이프라인은 범위 밖.** 신규 저장소에 `.github/` 워크플로가 없다. 원격 저장소 생성과
   배포 워크플로 이관은 별도 작업.
3. **이미지 2.51GB.** 두 역할의 `agents/` 중복이 주원인. 중복 제거는 후속 과제(소스 수정이 필요해
   이번 "원본 무수정" 원칙과 충돌).
4. **검증용 컨테이너 2개**(42130/42110)가 DELL 에 떠 있다. `docker compose -f docker-compose.dev.yml down` 으로 정리.
5. **묶음C는 계획만.** 실행하려면 torch/python 버전 통일 후 모델 출력 동등성(골든셋 200/150건) 실측이 선행돼야 한다.

상세: `00-log.md` · `01-requirements.md` · `04-changes.md` · `round-1-review.md` · `round-1-claude.md` · `bundle-c-plan.md`
