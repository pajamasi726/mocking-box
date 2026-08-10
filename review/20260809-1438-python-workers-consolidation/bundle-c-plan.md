# 묶음C 통합 계획: ai-model-serve + embedding-service

> 이번 라운드 산출물은 **계획서뿐이다**. 두 저장소 소스는 한 줄도 고치지 않았다(R9).
> 묶음A와 달리 "복사해서 합치면 끝"이 아니다. 두 서비스는 파이썬·torch·transformers 메이저 라인이
> 서로 다르고, 그 차이가 **모델 출력**에 닿아 있다. 그래서 통합 자체보다 **출력이 안 변했다는 증명**이
> 먼저다. 아래는 그 증명을 어떻게 할지에 대한 계획이다.

## 1. 현황 실측

| 항목 | ai-model-serve | embedding-service | 비고 |
|---|---|---|---|
| python | 3.12-slim | **3.11.15-slim** | `Dockerfile:2` / `Dockerfile:1` |
| torch | **2.5.1** (PyPI, CPU) | **2.6.0+cu124** (pytorch index) | `requirements.txt:126` / `Dockerfile:19`+`requirements.txt:17` |
| transformers | **4.46.2** | **4.51.1** | `requirements.txt:128` / `requirements.txt:18` |
| sentence-transformers | 3.3.1 | 3.0.1 | 역방향 차이 (`:114` / `:27`) |
| scikit-learn | 1.5.2 | 1.5.1 | (`:112` / `:8`) |
| numpy | 1.26.4 | 1.26.4 | 동일 |
| pydantic / fastapi | 2.10.0 / 0.115.5 | >=2.6 / >=0.110 | embedding 은 미고정 |
| 추론 엔진 | transformers 직접 | **vllm 0.8.5.post1** | `requirements.txt:24` |
| 포트 | 42020 | 48008 | |
| 기동 동작 | uvicorn/gunicorn 만 | **CMD 에서 학습 2건 실행 후 uvicorn** | `Dockerfile:31-33` |
| 배포 위치 | dev(DELL)/AWS, CPU | **HP(prod) 상주 GPU 워커(A6000×2)** | |
| deps 수 | 140 | 30(+torch 3종 별도 설치) | |

## 2. 통일 목표 버전과 그 근거

방향은 하나뿐이다. **ai-model-serve 가 embedding-service 쪽으로 올라간다.**

| 대상 | 통일 버전 | 왜 이 방향인가 |
|---|---|---|
| python | **3.12** | embedding 만 3.11. vllm 0.8.5·torch 2.6·transformers 4.51 모두 3.12 지원 → 올리는 쪽이 제약이 적다 |
| torch | **2.6.0 (cu124)** | vllm 0.8.5.post1 이 `torch==2.6.0` 을 **하드 핀**한다. 2.5.1 로 내리려면 vllm 을 버려야 하므로 하향 불가 |
| transformers | **4.51.1** | vllm 0.8.5 가 요구하는 하한. ai-model-serve 의 4.46.2 는 상향 대상 |
| sentence-transformers | **3.3.1** | 양쪽 상위값. embedding 의 3.0.1 → 3.3.1 상향 |
| scikit-learn | **1.5.2** | 양쪽 상위값, patch 차이 |
| numpy | 1.26.4 | 이미 동일 — 변경 없음 |

즉 "합치기 위해 버전을 맞춘다"가 아니라 **ai-model-serve 를 torch 2.6/transformers 4.51 로 올릴 수 있느냐**가
묶음C의 전부다. 이게 통과하면 통합은 묶음A와 같은 방식(단일 이미지 + `MODEL_ROLE=serve|embedding` 분기)으로
기계적으로 끝난다.

## 3. 먼저 깨질 것으로 예상되는 지점 (블로커)

| # | 블로커 | 근거 | 성격 |
|---|---|---|---|
| B1 | `torch.load()` 를 `weights_only` 없이 호출 | `ai-model-serve/dl/model_utils.py:50`, `:72` | **torch 2.6 에서 기본값이 `True` 로 바뀌어 전체 pickle 체크포인트 로드가 실패한다.** S3의 `p_n_token_classification_v2.pt`·`best_regression_model.pt` 를 그대로 못 읽는다 |
| B2 | koelectra 토크나이저/모델 동작 | `dl/model_utils.py:21` (`monologg/koelectra-base-v3-discriminator`) | transformers 4.46→4.51 사이 tokenizer 기본 동작·`AutoModelFor*` 헤드 초기화 변화 가능 → 출력 드리프트 |
| B3 | 파이썬 3.11→3.12 휠 | embedding `requirements.txt` (kiwipiepy 0.18.0, konlpy, trl 0.9.6, peft 0.11.1, datasets 2.20.0) | 소스 빌드로 떨어지면 빌드 시간·재현성 저하. konlpy 는 JVM 의존 |
| B4 | 기동 시 학습 실행 | `embedding-service/Dockerfile:31-33` | 통합 이미지의 entrypoint 가 역할별로 갈라져야 하고, 재기동 = 학습 재실행. **학습 산출물이 바뀌면 출력 동등성 검증 자체가 무의미해진다** → 검증 중에는 학습을 끄고(`SKIP_STARTUP_TRAINING` 류 플래그 도입 필요) 기존 산출물을 고정해서 비교해야 한다 |
| B5 | GPU 상주 프로세스 | HP(prod) A6000×2, vllm `LLM(...)` (`natural_query/vllm_engine.py:65`) | 검증을 위해 HP 를 내렸다 올리는 것 자체가 운영 중단. 검증은 **DELL(RTX 6000 Ada)에서 별도 포트로** 수행하고 HP 는 최종 전환 때만 건드린다 |
| B6 | 이미지 비대화 | torch cu124 + vllm ≈ 8~10GB | CPU 추론만 하는 ai-model-serve 가 CUDA 런타임을 지고 다닌다. **통합의 대가**로 명시하고, 받아들일 수 없으면 "저장소만 통합, 이미지는 2개(role 별 stage 분리)"로 후퇴하는 선택지를 남긴다 |

B1 은 계획 단계에서 이미 확정된 실패다. `torch.load(..., weights_only=False)` 명시가 **필수 선행 수정**이며,
이건 "버전만 올리는 무수정 통합"이 불가능하다는 뜻이다.

## 4. 출력 동등성 검증 절차

원칙: **버전을 바꾼 쪽(ai-model-serve)과 안 바꾼 쪽(embedding-service) 모두** 기존 이미지(baseline)와
신규 이미지(candidate)를 **동시에 띄워** 같은 입력을 넣고 결과를 비교한다. 모델 가중치·S3 체크포인트·
LoRA 어댑터는 동일 리비전으로 고정한다(해시 기록).

### 4.1 골든셋

| 대상 | 엔드포인트 | 입력 건수 | 구성 |
|---|---|---|---|
| ai-model-serve | `POST /analyze_content` (`main.py:196`) | **200건** (최소 20건, 실제 200건 권장) | 운영 로그에서 추출한 실제 리뷰 문장. 긍/부정/중립 각 60건 + 이모지·자모·URL 포함 엣지 20건 |
| embedding-service | 임베딩(`vector_search/index`), 유사도, 번역, 감성, 카테고리/시술 분류 | 라우터당 **20건 이상**, 합계 150건 이상 | 라우터별 대표 입력 + 길이 극단(1토큰/최대길이) 포함 |

골든셋은 `model-worker/goldenset/bundle-c/*.jsonl` 로 커밋하고, **개인정보는 마스킹**한 뒤 저장한다.

### 4.2 통과 임계값 (숫자 명시)

| 출력 종류 | 지표 | 통과 기준 |
|---|---|---|
| 토큰 분류 라벨열 (`pos_neg_result`) | 라벨 시퀀스 완전일치율 | **100% (200/200)**. 1건이라도 다르면 FAIL |
| 회귀 점수 (`score_result`) | 절대오차 max | **≤ 1e-4**, 평균 절대오차 ≤ 1e-5 |
| 임베딩 벡터 | 코사인 유사도 min | **≥ 0.9999** (전 건), L2 상대오차 max ≤ 1e-3 |
| 임베딩 기반 top-k 검색 | k=10 순위 완전일치율 | **≥ 99%**, 불일치 시 상위 3개는 100% 일치 |
| 분류기 (category/treatment) | argmax 일치율 / softmax 절대오차 | **100% / ≤ 1e-3** |
| 생성계 (번역·요약 LoRA) | greedy(`do_sample=False`, seed 고정) 문자열 완전일치율 | **≥ 95%**, 불일치 건은 chrF **≥ 0.98** 이어야 통과 |
| 학습 스크립트 (`vector_search/sentiment/train*.py`) | 학습 후 eval accuracy/F1 차이 | **≤ 0.5%p** (동일 seed·동일 데이터 스냅샷) |
| 응답 스키마 | JSON 키 집합 diff | **차집합 0** |

부동소수 임계값(1e-4/1e-3)은 CPU↔CPU 비교 기준이다. **GPU 에서 비교할 때는 같은 GPU·같은 dtype**
(fp16 여부 고정)으로만 비교한다. 서로 다른 디바이스 간 비교는 애초에 판정 근거로 쓰지 않는다.

### 4.3 비교 하네스

- **위치: `/Users/steve/steve/legal-care/model-worker/tools/bundle-c-diff.py`** (신규).
  골든셋도 같은 저장소의 `model-worker/goldenset/bundle-c/*.jsonl` 에 둔다.
  통합 저장소 `model-worker/` 는 **1단계에서 하네스와 골든셋만 담은 껍데기로 먼저 만들고**, 5단계에서 그 안에
  `roles/{serve,embedding}` 과 Dockerfile 을 채운다. 이렇게 해야 2~4단계(통합 전 단독 상향)에서도 같은 경로의
  같은 하네스로 비교할 수 있다. 묶음A 의 `crawler-worker/tools/` 와 같은 배치다.
  동작: 골든셋 JSONL 을 baseline/candidate 두 URL 에 동시 POST → 위 지표 계산 → `PASS/FAIL` 과 위반 건
  리스트 출력. 실패 건은 입력·양쪽 출력을 그대로 덤프한다.
- 이미 저장소에 있는 `Dockerfile.fixture`/`fixture_app.py`(양쪽 모두 보유)를 그대로 써서 **외부 의존 없이**
  하네스 자체를 먼저 검증한다.
- 실행 위치는 **DELL**. HP(prod) 는 최종 전환 전까지 접촉 금지.

## 5. 실행 순서

1. **골든셋 확보** — 운영 로그에서 입력 수집·마스킹, baseline(현행 이미지 2개)으로 기대출력 스냅샷 생성.
   가중치/체크포인트/LoRA 해시를 함께 기록한다. *게이트: 스냅샷 재현 2회 연속 동일*
2. **B1 선수정** — `dl/model_utils.py` 의 `torch.load` 2곳에 `weights_only=False` 명시.
   **torch 2.5.1 환경에서 먼저** 적용하고 골든셋 100% 동일함을 확인한다(버전 변경과 코드 변경을 분리).
3. **ai-model-serve 단독 상향** — python 3.12 유지, torch 2.5.1→2.6.0, transformers 4.46.2→4.51.1.
   이미지만 바꾸고 코드는 2단계 결과 그대로. *게이트: 4.2 표 전 항목 PASS*
4. **embedding-service 단독 상향** — python 3.11→3.12, sentence-transformers 3.0.1→3.3.1.
   torch/transformers 는 이미 목표 버전이라 변경 없음. 기동 학습은 끄고 기존 산출물 고정.
   *게이트: 4.2 표 전 항목 PASS + 학습 스크립트 eval 지표 ≤0.5%p*
5. **저장소·이미지 통합** — 3·4 가 모두 통과한 뒤에야 착수. 묶음A와 동일한 형태
   (`model-worker/roles/{serve,embedding}` + 단일 Dockerfile + `MODEL_ROLE` entrypoint 분기).
   entrypoint 는 embedding 역할에서만 기동 학습을 실행하고, 플래그로 끌 수 있어야 한다.
   *게이트: 두 역할 병렬 기동 + 라우트 집합 차집합 0 + 4.2 재실행 PASS*
6. **HP 전환** — 이미지 태그 교체 → 헬스체크 → 골든셋 스모크(각 라우터 5건) → 이상 시 이전 태그로 롤백.
   GPU 상주 특성상 무중단이 아니므로 **트래픽 적은 시간대 + 롤백 태그 사전 pull** 을 전제로 한다.

각 단계는 **앞 단계 게이트를 통과하지 못하면 다음으로 넘어가지 않는다.** 3단계에서 FAIL 이 나면
묶음C 통합은 "지금은 하지 않는다"로 끝내는 것이 정답이다. 배포단위 1개 줄이자고 모델 출력을 흔들 이유는 없다.

## 6. 롤백

1~4단계는 코드/이미지 태그 되돌리기로 끝난다(원본 저장소 유지). 5단계 산출물은 신규 디렉토리라 삭제가 롤백이다.
6단계만 운영 영향이 있으며, 이전 이미지 태그를 HP 로컬에 남겨두는 것으로 대비한다.

## 7. 이번 라운드 미변경 확인

`ai-model-serve/`, `embedding-service/` 소스 변경 0건. 이 문서 외 산출물 없음.
