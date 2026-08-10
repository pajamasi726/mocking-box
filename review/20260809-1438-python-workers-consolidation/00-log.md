# 팀 작업 로그 — 파이썬 워커 5개 → 3개 통합

**사용자 원문 요청**
> 3개로 줄일수있구나 그러면 셀프 리뷰모드로 분리해서 합쳐봐

**구성**: writer=claude · reviewer=claude(셀프리뷰 단독) · mode=review · rounds=2 · Codex 미사용 · 개발계 대상

## 14:55 오케스트레이터 → 의존성 선실측 (합치기 가능성 판정)
| 서비스 | python | fastapi | playwright | torch | pydantic | deps |
|---|---|---|---|---|---|---|
| crawler | 3.12-slim | 0.110.0 | 1.42.0 | - | 2.9.2 | 84 |
| ai-delegate-crawler | 3.12-slim | 0.110.0 | 1.42.0 | - | 2.9.2 | 117 |
| pg-receipt-crawler | 3.12.10-slim | - | **1.55.0** | - | **2.12.0** | 25 |
| ai-model-serve | 3.12-slim | 0.115.5 | - | **2.5.1** | 2.10.0 | 140 |
| embedding-service | **3.11.15-slim** | >=0.110 | - | **2.6.0** | >=2.6 | 30 |

**판정**
- **묶음A (crawler + ai-delegate-crawler)**: python·fastapi·uvicorn·aio-pika·playwright·numpy·pydantic **전부 동일 버전** → 합치기 쉬움. 최우선.
- **묶음B (pg-receipt-crawler)**: playwright 1.55 vs 1.42, pydantic 2.12 vs 2.9 로 A와 충돌. **단독 유지가 맞음**(사용자도 "브라우저 무거움" 인지).
- **묶음C (ai-model-serve + embedding-service)**: **python 3.12 vs 3.11**, **torch 2.5.1 vs 2.6.0**, transformers 4.46 vs 4.51 → 버전 통일 필요. 모델 동작 변화 위험 있어 A보다 위험. 신중히.
→ 3개 목표는 타당하나 난이도가 A ≪ C. A 먼저, C는 버전 통일 검증 후.

## 15:05 team-pm → 오케스트레이터
- R1~R9/AC1~AC7. RISK MEDIUM. OPEN QUESTIONS 없음.
- 통합방식: **단일 이미지 + 역할별 프로세스**(CRAWLER_ROLE=thread|delegate). 한 프로세스 병합 불가 — 동명 최상위 패키지(config.py/kafka/routes/agents)가 내용 상이, 경로 3개 정면충돌, Kafka 구독 토픽 상이.
- 신규 위치 `/Users/steve/steve/legal-care/crawler-worker/` (roles/thread, roles/delegate). 원본 저장소 무변경.
- 묶음C(AI 2종)는 계획·검증만 — embedding 은 기동 시 학습 실행 + Qwen2.5-3B LoRA 로드, HP 운영 GPU 워커라 torch/python 통일 전 출력 동등성 실측 필요.
- 못박음: group.id 양쪽 동일 기본값이라 현행 유지, 플래그 기본값 상이(crawler=true/ai-delegate=false) 보존, fixture/safe 이미지 별도 빌드 유지.
- 오케스트레이터 자가점검: 사용자 의도("합쳐도 되는 것끼리")와 일치. A만 실통합·C는 계획이 타당(위험 비대칭).

## 15:45 team-dev(IMPLEMENT) → 묶음A 통합 완료
- `/Users/steve/steve/legal-care/crawler-worker/` 생성: 단일 Dockerfile + entrypoint(CRAWLER_ROLE 분기) + requirements union(128) + roles/thread(72파일)·roles/delegate(108파일) + README + tools.
- **실검증(DELL)**: 이미지 2.51GB 빌드 성공, 두 역할 동시기동 /health 200, **라우트 런타임 덤프가 통합 전과 문자열 동일(차집합 0)**, 외부 라우트 5건 503, established TCP 0건, 브로커에 consumer group 미생성, delegate 테스트 38 passed, fixture 이미지 test stage 5 passed.
- union 동일패키지 상이버전 0건. 원본 5개 저장소 수정 0건, 기존 dev 컨테이너 무접촉.
- 자체 발견·수정: rsync `--exclude 'aws/'` 가 `agents/aws/` 까지 삭제 → thread ModuleNotFoundError. 앵커드 패턴 재복사 + **양방향 파일집합 대조 + import 정적해석** 검사 추가(첫 검증이 해시만 봐서 "빠진 파일"을 못 잡았음).
- 리뷰 요청 2건: ①psutil 7.2.2→6.1.1(thread 미고정이었고 union 에서 delegate 핀 적용) ②thread 의 ELASTIC_APM_ENABLED 기본 true + APM 주소 하드코딩 — 검증 1회차에 실제로 그 주소로 연결됨(env 로 false 주고 재검증, 기본값은 원본 보존)
- STATUS: DONE

## 15:53 team-dev(RESPOND) → 오케스트레이터
- 라운드1 지적 3건(MAJOR) 전부 반영. 상세 `round-1-claude.md`. STATUS: REVISED
- 핵심: ①Kafka 검사가 브로커·그룹명이 둘 다 틀려 탐지력 0이었음을 인정하고 재검사 ②AC3 를
  오프라인 임포트 덤프 + 통합 전 실제 이미지 A/B 로 교체 ③Dockerfile allowlist 복귀
- 반박 2건: fixture 컨텍스트 .dockerignore(바이트 동일 불변식 우선), requirements 이중선언(원문 보존)

## 16:20 오케스트레이터 → 독립 재검증 (저자 보고 교차확인, DELL 실측)
사무실 VPN 경로 변경으로 192.168.1.55 직접 접근 불가 → 공인IP 포트포워드(`ssh -p 50022 legalcare@114.203.1.178`)로 수행.

| 항목 | 저자 주장 | 내 재실측 | 판정 |
|---|---|---|---|
| Kafka 미참여 | 그룹 2개 조회, 통합컨테이너 없음 | **두 클러스터 전체 그룹 전수 스캔**: kafka1 30그룹/멤버 36건, 격리스택 1그룹/0건 → 192.168.80.2/.3 매치 **0건** | 확인(더 강한 검사로) |
| 라우트 동등 | 양쪽 diff 0 | 내 스크립트로 4조합 재덤프: thread 14=14, delegate 21=21 **집합 완전동일** | 확인 |
| /health 동등 | 키·값 동일 | 통합 전 컨테이너 내부 호출과 **바이트 동일** | 확인 |
| 이미지 allowlist | thread 68 / delegate 87 | 동일. fixture·tests·문서·lint 7종 전부 부재 | 확인 |
| 원본 무수정 | 세션 이후 수정 0건 | 변경파일 mtime 전부 08-06~08-08 (세션시작 08-09 14:38 이전) | 확인 |
| roles 바이트 동일 | sha256 동일 | 내용상이 0건 · 복사본 전용 파일 0건 · 누락은 로그/스크린샷/.dockerignore 뿐 | 확인 |
| 외부차단·역할가드 | 503 / exit 64 | 4개 라우트 503, exit=64(미설정·오타 모두) | 확인 |
| delegate 테스트 38 | 38 passed | **`pytest tests` 그대로는 수집 실패** (`fixture_runtime` 없음). 다만 **통합 전 원본 이미지도 동일하게 실패**하고, 제외 시 양쪽 다 38 passed → 회귀 아님 | 확인, 단 보고 누락 |

**저자 보고에서 잡아낸 것 1건(문서 부정확)**: "통합 이미지에서 38 passed" 는 `--ignore=tests/test_fixture_runtime.py`
가 필요하다는 조건이 빠져 있었다. 다음 사람이 그대로 돌리면 수집 에러를 보고 통합을 의심하게 된다.
→ README 에 "테스트 돌리는 법" 절 신설, 통합 전에도 같았다는 A/B 근거까지 명시.

**추가로 내가 발견한 것 1건(리스크, 코드 결함 아님)**: `roles/*` 는 원본의 **작업 디렉토리** 복사본이지
git 최신 커밋이 아니다(crawler 11건·ai-delegate-crawler 34건 미커밋). 개발계 실행 이미지도 그 상태라
현재는 정합하지만, 나중에 git 에서 다시 뽑으면 다른 코드가 나온다. → README 에 경고 명시.

## 16:35 오케스트레이터 → 커밋
- `legal-care/crawler-worker` git init + 최초 커밋 `d6aee86` (190파일). 원격 없음·푸시 안 함.
- 자격정보 혼입 점검 통과(.env 계열 0건), `.gitignore` 신규 작성.
