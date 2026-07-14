# 프레임워크 전환(jOOQ→MyBatis) 후 요청/응답 기반 차등 테스트 — 툴 조사 및 build-vs-adopt 보고서

> 조사일: 2026-07-14 · 대상: legal-care(~70개 Spring MSA) → legalcare-renew(단일 서비스) 통폐합 검증
> 출처는 각 절 하단에 표기. 검증 단계 일부가 세션 한도로 미완료되어, 핵심 주장은 1차 소스(공식 README/문서) 인용 기반이며 교차검증은 부분적임.

---

## TL;DR — 결론 먼저

1. **읽기(GET) 경로**: 기성 툴로 충분. Diffy 계열 프록시 또는 단순 리플레이+diff 러너로 해결 가능.
2. **쓰기(CUD) 경로**: **요청/응답 + DB 상태 diff를 함께 해주는 기성 툴은 존재하지 않음.** Diffy, GoReplay, Scientist, Speedscale, Signadot 전부 이 문제를 "사용자 몫"으로 남겨두거나(문서에 전략 부재), 읽기 전용으로 명시적으로 제한함(GitHub Scientist).
3. **권고: Adopt(부품) + Build(얇은 하네스)** — GoReplay/Keploy 캡처본을 요청 코퍼스로 재활용하고, docker-compose 기반의 커스텀 "mocking-box" 하네스(구스택+신스택+동일 스냅샷 MySQL 2대, 순차 리플레이, 응답 diff + 테이블 상태 diff)를 직접 만드는 것이 현실적. 시니어 1인 기준 MVP 1~2주 수준으로 추정.

---

## 1. Keploy가 왜 안 됐는가 (실패 원인 확정)

Keploy는 eBPF로 네트워크 계층에서 동작한다:
- **인그레스**: 들어오는 HTTP 콜을 YAML 테스트케이스로 저장.
- **이그레스**: 앱이 내보내는 TCP 연결(= MySQL 와이어 프로토콜)을 프록시로 가로채 **바이너리 스트림을 YAML mock으로 변환**해 저장.
- **테스트 모드**: 기록된 HTTP 요청을 재전송하면서, 앱의 아웃고잉 콜(SQL)을 기록된 mock과 **요청 매칭**으로 응답. 실제 DB를 치는 HTTP-only 리플레이 모드는 문서상 존재하지 않음.

즉 리플레이 성공 여부가 "**앱이 내보내는 SQL이 녹화 당시와 같은가**"에 묶여 있다. jOOQ→MyBatis 전환으로 생성 SQL이 구조적으로 달라지면 mock 매칭이 깨진다. 미지 프로토콜용 fuzzy matching이 있으나 구조적으로 다른 SQL은 감당 못 함. 문서상 해법은 "재녹화"뿐 — 그러면 구버전과의 비교 기준 자체가 사라지므로 이 용도로는 원천적으로 부적합.

**단, 이미 만든 Keploy 녹화본의 인그레스 HTTP 테스트케이스(YAML)는 요청 코퍼스로 재활용 가치가 있음.** mock 부분만 버리면 됨. Keploy의 noise detection(타임스탬프/랜덤값 자동 식별)도 설계 참고 가치 있음.

출처: Keploy Wiki "Know more about Keploy" (2024-07)

---

## 2. 후보 툴별 적합성 평가

### 2.1 Diffy / opendiffy (HTTP 차등 테스트의 원조)

- **동작**: HTTP 프록시로서 각 요청을 3개 인스턴스에 멀티캐스트 — candidate(신코드), primary(구코드), secondary(구코드 복제). 응답만 비교하며 내부 구현(SQL)은 전혀 안 봄 → **ORM 전환이 보이지 않는다는 점에서 요구사항 (1) 정확히 충족.**
- **노이즈 필터링(핵심 아이디어)**: primary vs secondary(둘 다 구코드)가 서로 불일치하는 빈도를 측정해, 그 정도의 불일치는 노이즈(타임스탬프, 생성 ID 등)로 자동 분류. candidate와의 불일치가 그보다 유의하게 크면 회귀로 판정.
- **형태**: Docker 이미지(diffy/diffy)로 배포 — "box" 요구사항 충족.
- **치명적 한계 ①**: 쓰기 요청도 3개 인스턴스에 그대로 멀티캐스트 → **쓰기가 3중 실행됨.** DB 스냅샷/복원, 상태 diff, 쓰기 격리 전략이 전무. README에 관련 메커니즘 없음.
- **치명적 한계 ②(라이선스)**: opendiffy 포크는 **CC BY-NC-ND 4.0(비상업·변경금지)** — 상용 환경 도입/포크에 법적 제약. 원본 twitter/diffy는 Apache-2.0이지만 아카이브됨(Scala, 최종 릴리스 2023-09 이전).
- **평가**: 아이디어(3-way 노이즈 캔슬링)는 훔치고, 코드는 안 쓰는 게 맞음.

출처: github.com/opendiffy/diffy README, Twitter Engineering "Diffy: Testing services without writing tests" (2015)

### 2.2 GoReplay

- **동작**: 프록시가 아니라 **libpcap 방식의 패시브 네트워크 리스너** — 운영 인프라 무변경으로 트래픽 캡처, 실시간 또는 파일/Kafka 경유로 테스트 환경에 리플레이. "패킷 캡처 후 시뮬레이션" 요구사항에 정확히 부합.
- **미들웨어**: 리플레이 요청/응답 양쪽에 접근 가능한 확장 훅 → 토큰 재발급, 민감정보 마스킹, 커스텀 diff 삽입 지점.
- **한계**: OSS README에는 응답 diff 기능·쓰기(POST/PUT/DELETE) 안전장치가 없음(공식 사이트의 diff 기능은 PRO/상용 쪽 마케팅). **OSS 최종 릴리스 v1.3.3이 2021-10** — 유지보수 정체가 2026년 도입 리스크.
- **평가**: 트래픽 수집/리플레이 **부품**으로는 최상급. end-to-end 솔루션은 아님. GoReplay로 캡처 → Diffy로 흘리는 조합 사례도 존재(선행 사례 있음).

출처: github.com/buger/goreplay README, goreplay.org/shadow-testing, Medium 실습기(2019)

### 2.3 Speedscale / proxymock (2024–2026 상용 신흥주자)

- proxymock: 무료 로컬 CLI(brew 설치, 127.0.0.1:4140 프록시). 인바운드+아웃바운드 녹화, `proxymock replay --test-against http://localhost:8080`으로 신규 빌드에 리플레이 — 리라이트 검증을 명시적으로 마케팅.
- **주의**: DB 의존성도 "프로토콜/쿼리 레벨 mock"으로 처리 → **DB mock을 쓰는 순간 Keploy와 동일한 실패 모드.** 단, DB mock을 쓰지 않고 실제 DB를 물린 채 인바운드 리플레이만 쓰는 구성은 가능해 보임.
- 자동 노이즈 처리(만료 토큰/타임스탬프/correlation ID 재작성)는 이 분야에서 가장 성숙. 팀 워크플로/CI 리플레이는 유료.
- 쓰기 상태 처리: 문서상 "테스트 데이터 시딩" 수준. 스냅샷/상태 diff 없음.

출처: speedscale.com/proxymock, "Definitive Guide to Traffic Replay" (2026-03)

### 2.4 Signadot SmartTests (2025, K8s 섀도 테스팅)

- 신버전을 격리 샌드박스에서 구버전과 나란히 실행, 같은 트래픽을 보내 응답 diff. **CDC(Debezium)로 운영 DB를 복제한 격리 데이터스토어** 개념까지 제품화 — 질문의 "shadow traffic + 복제 격리 DB" 전략의 상용 구현체.
- 한계: Kubernetes 종속(온프렘 docker-compose 환경엔 무거움), 상용, 쓰기는 결국 "공유 baseline"에 닿는 모델 — 요청 단위 스냅샷 복원은 아님.

출처: signadot.com 블로그 (2025-04)

### 2.5 Scientist 패턴 / scientist4j (인프로세스 병렬 실행)

- GitHub이 권한 시스템 재작성 등에 실제 사용한 패턴: control(구코드)과 candidate(신코드)를 같은 프로세스에서 둘 다 실행, 결과 비교·기록, 반환은 항상 control.
- **GitHub 공식 입장: 부수효과 있는 코드에 절대 사용 금지 — 읽기 전용에만 사용.** 같은 DB에 쓰는 candidate는 "dangerous and incorrect".
- scientist4j(Java 포트): Experiment<T>, 순서 랜덤화, Dropwizard/Micrometer 메트릭. **README에 "no longer actively maintained" 명시** — 패턴만 참고하고 직접 구현할 것(코어는 수십 줄).
- Flexport 사례: control/candidate를 **순서를 바꿔 두 번 실행**해 mismatch율 차이로 숨은 상태 간섭을 탐지하는 휴리스틱 — 쓰기 간섭 "탐지"는 되지만 "격리"는 아님.
- 이번 케이스 부적합 사유: 구·신 코드가 **다른 프로세스/다른 저장 계층**이므로 인프로세스 패턴 자체가 안 맞음. 다만 "노이즈 측정→비교" 사고방식은 동일하게 유효.

출처: github.blog "Scientist: Measure Twice, Cut Once", github.com/rawls238/Scientist4J, Flexport Engineering (2023-04)

### 2.6 기타

- **mitmproxy**: HTTPS 캡처 → flow 파일 → client-replay. 스크립터블한 부품. diff/상태 처리 없음.
- **Hoverfly/VCR류**: 아웃바운드 HTTP mock 용도라 이 문제(DB가 의존성)엔 비껴감.
- **tcpreplay**: L2~L4 패킷 재생이라 TCP 세션 상태 때문에 HTTP 서버 테스트엔 부적합.
- **Pact(계약 테스트)**: 스키마/계약 수준 검증이라 "동일 입력→동일 동작" 검증엔 해상도 부족. 통폐합 후 소비자가 있는 API 계약 고정용으로는 보조 가치 있음.

---

## 3. 실제 회사들은 어떻게 했나 (쓰기 처리 중심)

| 회사 | 방식 | 쓰기(CUD) 처리 |
|---|---|---|
| **Zalando** (모놀리스→MSA, Returns 서비스) | 앱 코드에서 트래픽 복제: 모놀리스가 응답 후 비동기로 신서비스 `/consistency-checks`에 사본 POST(202 즉시 응답). Prometheus Matched/Unmatched/Failed 카운터 + Grafana. 엔드포인트별 임계치 도달 시 Skipper 프록시로 점진 컷오버 | **멱등 POST에만 병렬 실행 허용**, 멱등 보장 안 되는 쓰기는 패턴 적용 제외 명시. 스냅샷/상태 diff 없음 |
| **GitHub** (권한 시스템·코드서치 재작성) | Scientist로 운영 트래픽에서 구·신 병렬 실행, mismatch 리포팅, % 램프업 | **읽기 전용으로 명시적 제한** |
| **Flexport** | Scientist 확장(FlexportExperiment): 실험별 대시보드(정합율+성능), 순서 교차 2회 실행으로 상태 간섭 탐지 | 간섭 "탐지"만, 격리 안 함. Java/Kotlin 포트는 미투자 상태라고 자인(2023) |
| **Microsoft 플레이북** | 섀도 테스팅 = V-Current/V-Next 응답 수집·비교(응답은 V-Next에서 절대 서빙 안 함). 도구로 Diffy, Envoy, McRouter, Scientist, Keploy 열거 | 쓰기/부수효과 처리 지침 **부재** — 업계 표준 문서조차 이 갭을 인정하는 셈 |

**업계 공통 결론: 쓰기 경로는 (a) 멱등 엔드포인트로 한정하거나 (b) 읽기만 비교하거나 (c) 격리 DB를 별도로 마련하는 3가지뿐이었고, "요청 단위 동일 스냅샷 + 상태 diff"를 제품화한 사례는 발견되지 않음.** → 이게 바로 만들 가치가 있는 빈칸.

---

## 4. DB 상태 비교·스냅샷 부품 조사

### 스냅샷/리셋 속도 (요청·시나리오 단위 리셋 가능 여부 결정)

| 방식 | 속도 | 비고 |
|---|---|---|
| ZFS/Btrfs 스냅샷 | 사실상 즉시 | DBLab(Postgres 전용이지만 패턴 이식 가능)이 ZFS CoW로 1TiB를 ~10초에 thin clone. MySQL datadir에도 동일 기법 적용 가능 |
| LVM 스냅샷 | 빠름 | CoW. 일반 CI 컨테이너 안에서는 사용 곤란 |
| 데이터 미리 구운 Docker 이미지 재기동 | 수 초~수십 초 | **가장 실용적.** 시딩 완료된 datadir을 이미지에 베이크 → 컨테이너 재생성으로 리셋. Docker에 네이티브 볼륨 스냅샷 기능은 없음 |
| mysqldump 복원 | 느림(풀카피) | `--single-transaction`으로 무잠금 캡처는 가능 |
| 트랜잭션 롤백 | 즉시 | 앱 커밋을 막을 수 없는 블랙박스 리플레이엔 부적합(앱이 자체 커밋) |

- Testcontainers의 체크포인트/복원(CRIU)은 **2015년 제안 후 끝내 미구현**(issue #29, 2019 클로즈) — 여기 기대면 안 됨.
- 스냅샷은 쓰기 인플라이트 중 캡처 시 손상 위험 → 리셋 시점엔 DB 정지(quiesce) 필요.

### 상태 diff

- **pt-table-checksum: 부적합.** 복제 토폴로지 전용(statement-based replication 필수, 소스에서 체크섬 쿼리를 흘려 레플리카에서 재실행하는 구조) — 독립된 구스택 DB vs 신스택 DB 비교에는 못 씀.
- **mysqldbcompare**(MySQL Utilities): 서로 다른 호스트의 두 DB를 스키마+데이터 diff 가능하나 **유틸리티 자체가 EOL/레거시**, 체크섬 판정에 노이즈 사례 보고 → 의존하지 말고 패턴만 참고.
- **현실적 방법(직접 구현이 오히려 쉬움)**: 테이블별 `SELECT ... ORDER BY pk`를 정규화(제외 컬럼: created_at/updated_at/auto-increment 계열) 후 해시 비교, 불일치 테이블만 행 단위 diff. mysqldump `--skip-dump-date` + 정렬 옵션 + `diff`로도 초기 버전 구성 가능.

출처: Percona pt-table-checksum docs, postgres-ai/database-lab-engine, testcontainers-java#29, OneUptime MySQL/Docker 스냅샷 가이드(2026), mysqldbcompare 실습기

---

## 5. 커스텀 "mocking-box" 설계안 (Build 파트)

### 아키텍처

```
                ┌─────────────────────────────────────────────┐
                │                mocking-box                   │
 캡처 소스 ──▶  │  ┌──────────┐   ┌──────────────────────────┐ │
 (GoReplay .gor │  │ Ingestor │─▶│  Replayer (순차 실행)      │ │
  Keploy YAML,  │  │ 정규화    │   │  요청 n ─▶ 구스택         │ │
  HAR, access   │  └──────────┘   │  요청 n ─▶ 신스택         │ │
  log)          │                 └──────┬───────────┬───────┘ │
                │                        ▼           ▼         │
                │        ┌─────────────────┐ ┌───────────────┐ │
                │        │ legal-care (구) │ │ renew (신)     │ │
                │        │ MySQL-A ◀ 스냅샷│ │ MySQL-B ◀ 스냅샷│ │
                │        └─────────────────┘ └───────────────┘ │
                │   ┌────────────────┐  ┌────────────────────┐ │
                │   │ Response Differ│  │ State Differ        │ │
                │   │ (노이즈 규칙)   │  │ (테이블 해시/행 diff)│ │
                │   └───────┬────────┘  └─────────┬──────────┘ │
                │           └────▶ Divergence Report ◀┘        │
                └─────────────────────────────────────────────┘
```

### 핵심 설계 결정

1. **동일 시드 스냅샷 2부**: MySQL-A/B를 같은 베이크드 이미지에서 기동 → auto_increment 카운터까지 동일 → 생성 ID가 양쪽에서 같게 나와 후속 요청 divergence 최소화.
2. **시나리오 단위 순차 리플레이**: 쓰기 순서 결정성을 위해 시나리오(연관 요청 묶음) 내부는 단일 스레드. 리셋은 요청 단위가 아니라 **시나리오 단위**(컨테이너 재생성 수 초)로 시작 → 필요 시 ZFS로 고도화.
3. **응답 diff + 노이즈 학습**: Diffy의 3-way 아이디어를 축약 — 구스택에 같은 시나리오를 2회 돌려 자기 불일치 필드(타임스탬프, UUID, 세션토큰)를 자동 수집해 노이즈 규칙(JSONPath 제외 목록) 생성.
4. **상태 diff = 진짜 판정 기준(쓰기 경로)**: 응답이 같아도 DB가 다르면 실패, 응답이 달라도(예: 메시지 문구) DB가 같으면 경고 강등. 시나리오 종료 시 테이블별 정규화 해시 비교 → 불일치 테이블만 행 diff 출력.
5. **ID 치환 세션**: 응답에서 생성 ID를 추출해 후속 요청의 경로/바디에 스택별로 치환(`{{last_created_id}}`)하는 얇은 템플릿 기능 — 캡처 트래픽을 시나리오로 승격시키는 핵심.

### 재사용 부품

- 트래픽 수집: **GoReplay**(캡처·파일 포맷) 또는 이미 있는 **Keploy 녹화 YAML의 인그레스 파트 변환**
- 요청 코퍼스 보강: 운영 access log → 요청 생성(읽기 경로 대량 커버리지)
- 리플레이·diff 러너: Kotlin + OkHttp + jackson (JSON 정규화 diff) — 팀 스택 그대로
- DB 리셋: 시딩 베이크드 MySQL Docker 이미지 + docker compose recreate
- 리포트: 엔드포인트별 Matched/Unmatched/Failed 카운터(Zalando 방식) + HTML diff 리포트

### 공수 추정 (시니어 1인)

| 단계 | 내용 | 기간 |
|---|---|---|
| MVP | 읽기 경로: 캡처 변환 + 순차 리플레이 + JSON diff + 노이즈 제외 목록 | 2~4일 |
| v0.2 | 쓰기 경로: 스냅샷 리셋 + 테이블 해시 상태 diff + 시나리오 격리 | 3~5일 |
| v0.3 | ID 치환 세션, 노이즈 자동 학습(2회 실행), HTML 리포트 | 3~5일 |
| 합계 | | **약 1.5~2주** |

리스크: 시나리오 경계 자동 추출(캡처 트래픽을 유저 세션 단위로 묶기)이 가장 손이 감 — 초기엔 x-user-id/토큰 기준 그룹핑으로 단순화 권장.

---

## 6. 최종 권고

- **Adopt만으로는 불가**: 쓰기 상태 검증을 해주는 기성품이 없음(업계 문서·제품 전수 조사 결과 공백 확인). Diffy 포크는 라이선스(CC BY-NC-ND)로도 탈락.
- **순수 Build도 비추**: 트래픽 캡처·노이즈 필터링은 이미 검증된 부품/아이디어가 있으므로 재발명 금지.
- **권고 = 하이브리드**:
  1. 기존 Keploy 녹화본에서 인그레스 HTTP 케이스를 추출해 첫 코퍼스로 재활용 (매몰비용 회수)
  2. 읽기 경로부터 MVP 리플레이+diff로 즉시 가동 (2~4일 내 가치 실현)
  3. 쓰기 경로는 "동일 베이크드 스냅샷 × 2 + 시나리오 순차 리플레이 + 상태 diff" 하네스로 확장
  4. 컷오버 판단은 Zalando식 엔드포인트별 정합율 카운터로 정량화

DELL 워크스테이션(RTX 6000 Ada, docker 62컨테이너 여유)에 구·신 스택 + MySQL 2대 + 하네스를 docker-compose로 올리는 구성이 자연스러움 (ex_disk1 710G 여유면 스냅샷 이미지 다수 보관 가능).

---

## 부록: 주요 출처

- Keploy Wiki — architecture (eBPF, 프로토콜 레벨 mock): github.com/keploy/keploy/wiki
- opendiffy/diffy README + Twitter Engineering Diffy 발표문(2015)
- buger/goreplay README, goreplay.org/shadow-testing
- Speedscale proxymock 제품 페이지, Traffic Replay Definitive Guide (2026-03)
- Signadot "Shadow Testing" (2025-04)
- GitHub Blog "Scientist: Measure Twice, Cut Once" / rawls238/Scientist4J / Flexport Engineering (2023-04)
- Zalando Engineering "Parallel Run Pattern" (2021-11)
- Microsoft Code-with-Engineering Playbook — Shadow Testing
- Percona pt-table-checksum docs / postgres-ai/database-lab-engine / testcontainers-java#29
