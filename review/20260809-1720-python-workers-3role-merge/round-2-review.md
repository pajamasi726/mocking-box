# round-2 델타 교차 리뷰 — 크롤러 워커 3역할 통합 (reviewer: claude, 권한 review)

1라운드 지적 11건이 실제로 닫혔는지, 수정 패스가 새 결함을 만들지 않았는지만 봤다. 결론부터:
**11건 다 닫혔고, 반박 2건은 둘 다 근거가 실재한다. 새 결함은 없다.** 남은 건 문서 두 줄짜리 MINOR 2개다.

델타를 git 으로 직접 못 뽑는다는 제약이 있었다(round-1·2 모두 미커밋이라 `git diff HEAD` 가 두 라운드의
합이다). 그래서 **재빌드하지 않은 배포 이미지를 기준선 삼아** 라운드-2가 건드린 범위를 역산했다 —
`docker history` 의 apt 명령, 이미지 안 `entrypoint.sh` sha256, `pip list` 핀이 전부 현재 소스와 일치하면
그 파일들은 라운드-2에서 안 움직인 것이다. 아래 5번에 결과를 적었다.

## 1. healthcheck 새 구현 — 판정 논리에 구멍 없다

`docker-compose.dev.yml:126-159`. 네 가지를 따로 따졌다.

**자기 자신을 못 센다 — 구조적으로.** 배포본 컨테이너에서 직접 확인했다. tini 가 `ppid=0`, 워커
`PID 7 python worker.py` 가 `ppid=1`, 내가 `docker exec` 로 넣은 `sh -c` 가 `ppid=0`, 그 자식 python 이
`ppid=579`. `:139,143` 의 `ppid==1` 필터에 검사 프로세스도 그 자식도 안 걸린다. 1라운드 #1 의 실명 원인은
제거됐다. (`docker/entrypoint.sh:44` 의 `exec python worker.py` + `Dockerfile:145` tini ENTRYPOINT 가 같이
성립해야 하는 구조인데, 둘 다 이 저장소 안에 있고 어긋나면 fail-closed 라 위험 방향은 아니다.)

**거짓 실패 — 기동/리밸런싱/재연결.** 리밸런싱은 JoinGroup/SyncGroup 이 같은 TCP 세션 위에서 오가므로
ESTABLISHED 가 끊기지 않는다. 기동은 `start_period 30s`(원본 10s 에서 올림)가 흡수하고, 지금 워커는
브로커로 **ESTABLISHED 3본**을 쥐고 있어(`/proc/net/tcp` 실측: `03001CAC:2384`=172.28.0.3:9092, st=01 3행)
연결 하나가 끊겨도 판정이 안 흔들린다. 남는 건 브로커 전체가 45초(`interval 15s × retries 3`) 넘게
사라지는 경우뿐 → **F1**.

**env 가 비거나 형식이 다를 때.** `:129-134` 가 읽는 `KAFKA_SERVER_1/2` 는 **앱이 쓰는 바로 그 변수**다
(`roles/receipt/config.py:55-56` → `worker.py:19`). 검사와 앱이 같은 소스를 보므로 어긋날 여지가 없다.
포트 없는 값·비어 있는 값은 `ValueError`/`or {9092}` 로 9092 로 떨어지는데, 그래도 **9092 ESTABLISHED 를
요구하므로 조용히 통과하지 않는다 — fail-closed 다.** 브로커가 딴 포트면 오히려 항상 실패한다.

**unhealthy + `restart: unless-stopped` 가 겹치면?** — **아무 일도 안 난다.** 도커는 헬스 상태로 재시작을
걸지 않는다(restart 정책은 컨테이너 *종료*에만 반응한다). 공용망에 autoheal/health-watcher 류 컨테이너도
0건이고(전수 grep), 세 서비스 어디에도 `depends_on: service_healthy` 가 없다. 즉 거짓 unhealthy 의 대가는
**상태 플래그 하나**이고 브로커가 돌아오면 15초 안에 스스로 healthy 로 복귀한다.

### F1 (MINOR) 브로커가 죽으면 워커가 멀쩡해도 receipt 가 unhealthy 로 보인다 — 문서에 없다

`docker-compose.dev.yml:158` 은 브로커 ESTABLISHED 가 하나도 없으면 exit 1 이고, `:160,163` 이 45초 뒤
unhealthy 로 넘긴다. 개발계 카프카 재시작은 45초를 쉽게 넘는다. 위에 적었듯 파괴적 결과는 없지만,
README "receipt healthcheck" 절이 **못 보는 것(그룹 참여)** 만 적고 **거짓으로 보는 것(브로커 장애)** 은
안 적어서, 다음 사람이 receipt 를 먼저 의심하게 된다. 한 줄이면 닫힌다 —
"브로커가 내려가도 unhealthy 로 보인다. 워커 고장이 아니고, 도커가 이걸로 재시작을 걸지도 않는다."

## 2. 다이제스트 고정 — 진짜 그 이미지고, 대가도 적혀 있다

`Dockerfile:13`. DELL 실측으로 셋 다 맞다. (a) `sha256:229a2c5b…` 는 실재하는 OCI **image index**
다이제스트다(`docker manifest inspect` → amd64 매니페스트 포함 인덱스). config digest 를 잘못 박은 게
아니다 — Docker 29 + containerd 스냅샷터라 `.Id` 가 인덱스 다이제스트로 보일 뿐, RepoDigest 도 같은 값이다.
(b) DELL 의 `python:3.12-slim` RepoDigest 가 정확히 이 값이고, `crawler-worker:dev-3role` 의 하위 4개
rootfs 레이어가 그 이미지의 4개와 **한 자도 같다**. 검증한 그 베이스가 맞다. (c) 보안 패치가 끊기는
대가와 올리는 절차(ARG 한 줄 + AC1~AC3 재검증)는 `Dockerfile:20-21` 과 README "베이스는 다이제스트로
고정했다" 절에 적혀 있다. 닫힘.

## 3. 1라운드 11건 — 10건 완전 종결, 1건만 흔적 남음

#1·#3 은 위 1·2 에서 확인. #2 는 오히려 초과 이행이다(안 돌려 봤다고 적는 대신 실제로 돌려서 3건이
다르다는 걸 찾아내고 `04-changes.md:44`·README "낮아진 12건" 에 "실 엔드포인트 검증 0건"까지 명시).
#5·#6·#7앞·#8·#10·#11 은 해당 문서 절이 실제로 들어가 있는 것을 확인했다(README "리플레이 함정",
"네트워크 별칭", `docker-compose.dev.yml:77-83` earliest 주석, `04-changes.md:49,50`).

**반박 2건은 둘 다 타당하다.** #7 은 다른 클러스터가 실재한다 — `kafka1`/`kafka2`(망 `kafka_default`)에서
`medilawyer.crawling.review-boost.receipt.crawling:0:24` 를 직접 찍었다. 00-log 정정도 아니라 한정어를
붙인 게 맞고, 그 한정어는 `00-log.md:22-30` 에 들어가 있다. #9 도 사실이다 —
`roles/receipt/requirements.txt:3` 에 `boto3-stubs==1.40.51` 이 원래 있었으니 union 규칙 위반이 아니다.
스텁/런타임 어긋남을 인정하고 고칠 방향까지 적었으니 처리로 충분하다.

### F2 (MINOR) #4 반영이 한 군데 안 됐다 — `04-changes.md:8`

변경 파일 표에 아직 이렇게 적혀 있다: "`Dockerfile` | receipt allowlist COPY … · **1.55 new-headless 용
런타임 라이브러리 보강** · screenshots 디렉토리". 1라운드 #4 가 반증한 바로 그 근거다. `Dockerfile:33-46`
과 `04-changes.md:48` 은 "union 이라서 넓다"로 제대로 뒤집어 놨는데 이 한 줄만 남았다. 표를 먼저 읽는
사람에게는 정정 전 설명이 그대로 보인다. "원본 3종 apt 합집합" 으로 바꾸면 끝난다.

## 4. renew-replay compose 이중 등록 — 진짜지만 이 변경의 결함은 아니다

작성자 주장을 확인했다. `/mnt/ex_disk1/renew-replay/docker-compose.python.yml:22,68,181` 이
`crawler-web`(`legalcare/crawler:dev-20260806-safe`) · `ai-delegate-web` · `pg-receipt-worker` 를 아직
그대로 정의하고 있다. 한 가지 더 날카롭게 적어 두면 좋겠다: **구 3컨테이너는 지운 게 아니라 `exited`
상태로 남아 있고 정책이 `unless-stopped` 다**(실측). 그러니 `up` 뿐 아니라 그 프로젝트에서 `start` 를
누르거나 손으로 `docker start` 만 해도 별칭 3개가 이중 등록되고, receipt 는 컨슈머 그룹
`legalcare-local-pg-receipt-fixture-v1` 에 동시 참여해 파티션을 나눠 갖는다.

다만 그 파일은 이 저장소 밖이고, 롤백 절차 자체가 그 컨테이너들이 살아 있는 것에 의존한다(지우면 롤백이
깨진다). **이 변경으로 막을 수 있는 게 아니다 — 인수인계 사항이 맞다.** README "네트워크 별칭" 절에 이미
적혀 있으므로 처리로 충분하다고 본다.

## 5. 범위 이탈 — 0건

라운드-2가 손댔다고 한 것은 `docker-compose.dev.yml`(healthcheck + 주석)·`Dockerfile`(ARG + 주석)과
문서 3개다. 이미지를 재빌드하지 않았다는 점을 이용해 역으로 검증했다.

- `docker history crawler-worker:dev-3role` 의 apt 명령이 현재 `Dockerfile:61-79` 목록과 **글자 단위로 동일**
  → apt 블록은 라운드-2에서 안 움직였고, Dockerfile 은 여전히 도는 이미지를 재현한다.
- 이미지 안 `/app/docker/entrypoint.sh` sha256 `7a8414d2…` = 소스 트리 파일과 동일 → entrypoint 무변경.
- 이미지 `pip list` 의 boto3 1.34.34 · boto3-stubs 1.40.51 · certifi 2024.8.30 · fake-useragent 1.4.0 ·
  playwright 1.55.0 · pydantic 2.12.0 · hvac 2.3.0 · kafka-python 2.2.15 = `requirements.txt` 그대로
  → requirements 무변경(작성자가 "재빌드 유발 안 하려고 안 건드렸다"고 한 것과 일치).
- thread/delegate healthcheck(`docker-compose.dev.yml:47-53,70-75`)는 교체 대상
  `legalcare-local-crawler-web-1`/`-ai-delegate-web-1` 의 `Config.Healthcheck` 와 test·interval 10s·
  timeout 3s·start_period 30s·retries 12 까지 **완전 동일** → 주석의 "그대로 옮긴 것이다"가 사실이고
  라운드-2에서 손대지 않았다. receipt 만 원본 `roles/receipt/Dockerfile:85` 의 start_period 10s→30s.
- `roles/*` 17파일 무변경(untracked 그대로). 실험 잔여물도 0 —
  `hcdemo`/`hcfail-*`/`depverify-*`/`factcheck-*` 컨테이너·이미지 전부 없고, 컨슈머 그룹도
  `crawler-worker-hcdemo-v1` 은 소멸, 남은 건 문서화된 `crawler-worker-3role-verify-v1` 하나뿐이다.
  (`bw-test-1` 이 `Created` 로 있는데 2026-08-07 생성·`legalcare/booster` 이미지라 이 작업과 무관하다.)

## 확인 못 한 것

- 라운드-2의 의존성 A/B(스텁 S3·스텁 Vault)와 pydantic 모델 A/B 는 **내가 재실행하지 않았다.** 결과는
  작성자 기록으로만 받았다. 다만 안전에 걸리는 부분(실 엔드포인트 검증 0건)이 문서에 남았다는 것은 확인했다.
- F1 은 브로커를 실제로 내려서 관측한 게 아니라 검사 코드(`:158`)와 45초 창에서 도출한 것이다.
  컨테이너 정지 금지 제약 때문에 시뮬레이션하지 않았다.
- 라운드-1 시점 파일 상태를 git 으로 직접 비교하지 못했다. 위 5번의 역산으로 갈음했다.

## 정리

가짜 healthy 는 실제로 죽었다. 대체 구현은 이름 매칭을 버리고 구조(ppid)와 관측 신호(소켓)로 갔고,
자기 자신을 셀 수 없다는 것을 배포본에서 직접 확인했다. 못 보는 구간(그룹 이탈 후 ~10분)을 숨기지 않고
브로커 조회 절차로 보완한 것도 맞는 처리다. 다이제스트는 진짜 그 이미지고 고정의 대가도 적혀 있다.
반박 2건은 내가 따로 찍어 봐도 사실이었다. F1·F2 는 둘 다 문서 한 줄짜리라 다음 커밋에 얹으면 되고,
이것 때문에 라운드를 한 번 더 돌 이유는 없다.

VERDICT: APPROVE
