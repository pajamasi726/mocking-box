# 00-baseline — 교체 대상 개발계 3컨테이너 실측 덤프 (되돌리기 근거)

수집: 2026-08-09 · DELL(`ssh -p 50022 legalcare@114.203.1.178`) · `docker inspect` 원본은
`/tmp/baseline-3containers.json`(947줄) 에 그대로 남겨 뒀다. 아래는 되돌릴 때 필요한 값만 추린 것이다.

**중요** — 이 3컨테이너는 `legalcare-local` compose 프로젝트 소속이다(`/mnt/ex_disk1/renew-replay` 의
compose 5개 파일 묶음). 그 묶음을 다시 부르지 않고 `docker stop/start` 로만 다룬다. 컨테이너를 지우지
않으므로 롤백은 `docker start` 한 번이면 끝난다 — 2026-08-07 처럼 스택 전체를 재생성할 이유가 없다.

## 요약

| 컨테이너 | 이미지 | 이미지 ID | 상태 | 별칭 | IP |
|---|---|---|---|---|---|
| `legalcare-local-crawler-web-1` | `legalcare/crawler:dev-20260806-safe` | `433a8a07a602` | running/healthy | `crawler-web` | 172.28.0.21 |
| `legalcare-local-ai-delegate-web-1` | `legalcare/ai-delegate-crawler:dev-20260806-reliable5` | `42b79a73a960` | running/healthy | `ai-delegate-web` | 172.28.0.22 |
| `legalcare-local-pg-receipt-worker-1` | `legalcare/pg-receipt-crawler:dev-20260806-safe` | `f8bf1c4519e8` | running/healthy | `pg-receipt-worker` | 172.28.0.23 |

망: `legalcare-local_legalcare` 하나뿐. 볼륨 마운트 0건(3개 모두 `Mounts: []`) — 상태가 컨테이너 밖에 없다.
호스트 포트 바인딩도 0건(`PortBindings: {}`) — 망 내부에서 별칭으로만 불린다.

## `legalcare-local-crawler-web-1`

```
image        : legalcare/crawler:dev-20260806-safe  (sha256:433a8a07a602cce9290815ba8bdf3f280129069dfb84a7cb54b859d36c6224f2)
restart      : unless-stopped
entrypoint   : ['/usr/bin/tini', '--']
cmd          : /bin/sh -c
               if [ "$APP_ENV" = "prod" ]; then gunicorn main:app -k uvicorn.workers.UvicornWorker --workers=4 --bind=0.0.0.0:42030 --access-logfile=- --error-logfile=- --log-level=info --max-requests=10000 --max-requests-jitter=1000 --timeout=300 --graceful-timeout=300 --keep-alive=65; else uvicorn main:app --host=0.0.0.0 --port=42030 --no-access-log; fi
healthcheck  : ['CMD', 'python', '-c', "import urllib.request; urllib.request.urlopen('http://127.0.0.1:42030/health', timeout=2).read()"]
               interval=10s timeout=3s start_period=30s retries=12
mounts       : []
ports        : {}
network      : legalcare-local_legalcare
  aliases    : ['crawler-web', 'legalcare-local-crawler-web-1']
  ip         : 172.28.0.21
compose proj : legalcare-local / service=crawler-web
compose files: /mnt/ex_disk1/renew-replay/docker-compose.yml,/mnt/ex_disk1/renew-replay/docker-compose.dell.yml,/mnt/ex_disk1/renew-replay/docker-compose.replay.yml,/mnt/ex_disk1/renew-replay/docker-compose.roles.yml,/mnt/ex_disk1/renew-replay/docker-compose.python.yml
env (이미지 기본값 제외 · 실제 주입값):
  APP_ENV=dev
  ASYNCIO_SEMPAPHORE=1
  ELASTIC_APM_ENABLED=false
  ENABLE_EXTERNAL_ACTIONS=false
  ENABLE_KAFKA_CONSUMER=false
  ENABLE_LAMBDA_PROXY=false
  ENVIRONMENT=dev
  KAFKA_HEADER=dev-fixture
  KAFKA_SERVER_1=kafka:9092
  KAFKA_SERVER_2=kafka:9092
```

## `legalcare-local-ai-delegate-web-1`

```
image        : legalcare/ai-delegate-crawler:dev-20260806-reliable5  (sha256:42b79a73a960d32abc4d5f41be1066e6af5e403e0f46f2c4719635203ca3da27)
restart      : unless-stopped
entrypoint   : ['/usr/bin/tini', '--']
cmd          : /bin/sh -c
               if [ "$APP_ENV" = "prod" ]; then gunicorn main:app -k uvicorn.workers.UvicornWorker --workers=4 --bind=0.0.0.0:42010 --error-logfile=- --log-level=info --max-requests=10000 --max-requests-jitter=1000 --timeout=300 --graceful-timeout=300 --keep-alive=65; else uvicorn main:app --host=0.0.0.0 --port=42010 --no-access-log; fi
healthcheck  : ['CMD', 'python', '-c', "import urllib.request; urllib.request.urlopen('http://127.0.0.1:42010/health', timeout=2).read()"]
               interval=10s timeout=3s start_period=30s retries=12
mounts       : []
ports        : {}
network      : legalcare-local_legalcare
  aliases    : ['ai-delegate-web', 'legalcare-local-ai-delegate-web-1']
  ip         : 172.28.0.22
compose proj : legalcare-local / service=ai-delegate-web
compose files: /mnt/ex_disk1/renew-replay/docker-compose.yml,/mnt/ex_disk1/renew-replay/docker-compose.dell.yml,/mnt/ex_disk1/renew-replay/docker-compose.replay.yml,/mnt/ex_disk1/renew-replay/docker-compose.roles.yml,/mnt/ex_disk1/renew-replay/docker-compose.python.yml
env (이미지 기본값 제외 · 실제 주입값):
  APP_ENV=dev
  ELASTIC_APM_ENABLED=false
  ENABLED_CRAWLING_CHANNELS=NAVER_MAP,KAKAO_MAP,GOOGLE_MAP
  ENABLE_EUREKA=false
  ENABLE_EXTERNAL_ACTIONS=false
  ENABLE_KAFKA_CONSUMER=false
  ENVIRONMENT=dev
  USER_SERVICE_BASE_URL=http://booster-web:18080/user
  USER_SERVICE_TIMEOUT_SECONDS=3
```

## `legalcare-local-pg-receipt-worker-1`

```
image        : legalcare/pg-receipt-crawler:dev-20260806-safe  (sha256:f8bf1c4519e829fb33252c4ce199451e01ccb77929c5924683d95ecc3a55597c)
restart      : unless-stopped
entrypoint   : None
cmd          : ['python', 'worker.py']
healthcheck  : ['CMD', 'python', '-c', 'import os; os.kill(1, 0)']
               interval=15s timeout=3s start_period=10s retries=3
mounts       : []
ports        : {}
network      : legalcare-local_legalcare
  aliases    : ['legalcare-local-pg-receipt-worker-1', 'pg-receipt-worker']
  ip         : 172.28.0.23
compose proj : legalcare-local / service=pg-receipt-worker
compose files: /mnt/ex_disk1/renew-replay/docker-compose.yml,/mnt/ex_disk1/renew-replay/docker-compose.dell.yml,/mnt/ex_disk1/renew-replay/docker-compose.replay.yml,/mnt/ex_disk1/renew-replay/docker-compose.roles.yml,/mnt/ex_disk1/renew-replay/docker-compose.python.yml
env (이미지 기본값 제외 · 실제 주입값):
  ALLOW_DOTENV=false
  ENABLE_EXTERNAL_ACTIONS=false
  ENVIRONMENT=dev
  FAILURE_POLICY=dlq
  FIXTURE_IMAGE_URL=https://fixture.invalid/legalcare/receipt.png
  KAFKA_AUTO_OFFSET_RESET=earliest
  KAFKA_CONSUMER_GROUP_ID=legalcare-local-pg-receipt-fixture-v1
  KAFKA_SERVER_1=kafka:9092
  KAFKA_SERVER_2=kafka:9092
  KAFKA_TOPIC_RECEIPT_COMPLETED=legalcare.dev.fixture.receipt.completed
  KAFKA_TOPIC_RECEIPT_DLQ=legalcare.dev.fixture.receipt.dlq
  KAFKA_TOPIC_RECEIPT_KCS=legalcare.dev.fixture.receipt.kcs
  KAFKA_TOPIC_RECEIPT_KSTA=legalcare.dev.fixture.receipt.ksta
```

## 교체 전 `legalcare-local_legalcare` 망 전체 상태 (AC6 기준선)

컨테이너 28개, 전부 running. 교체 후 이 목록과 대조한다.

```
deploy-postgres-new-1                      running healthy
embedding-natural-dev                      running 
legalcare-local-ai-delegate-fixture-1      running healthy
legalcare-local-ai-delegate-web-1          running healthy
legalcare-local-ai-model-fixture-1         running healthy
legalcare-local-booster-web-5              running 
legalcare-local-booster-worker-1           running healthy
legalcare-local-crawler-web-1              running healthy
legalcare-local-dev-edge-1                 running healthy
legalcare-local-embedding-bridge-1         running healthy
legalcare-local-embedding-fixture-1        running healthy
legalcare-local-es-1                       running healthy
legalcare-local-gateway-1                  running healthy
legalcare-local-kafka-1                    running healthy
legalcare-local-lawkit-admin-1             running healthy
legalcare-local-lawkit-web-1               running healthy
legalcare-local-lawkit-worker-1            running healthy
legalcare-local-legacy-bridge-1            running healthy
legalcare-local-mongo-1                    running healthy
legalcare-local-mysql-1                    running healthy
legalcare-local-pg-1                       running healthy
legalcare-local-pg-receipt-worker-1        running healthy
legalcare-local-rabbit-1                   running healthy
legalcare-local-redis-1                    running healthy
legalcare-local-reviewed-web-2             running 
legalcare-local-reviewed-worker-1          running healthy
legalcare-local-stub-1                     running 
ticker-dev                                 running
```

## 되돌리기 (검증된 절차 — 실제로 왕복해 봤다. 04-changes.md AC7 참조)

교체는 **정지만** 한다(`docker stop`). 지우지 않으므로 컨테이너 정의·env·별칭이 도커에 그대로 남는다.

```bash
# --- 신규로 교체 ---
docker stop legalcare-local-crawler-web-1 legalcare-local-ai-delegate-web-1 legalcare-local-pg-receipt-worker-1
cd ~/crawler-worker-build
docker compose -p crawler-worker -f docker-compose.dev.yml -f docker-compose.cutover.yml up -d

# --- 되돌리기 ---
docker compose -p crawler-worker -f docker-compose.dev.yml -f docker-compose.cutover.yml down
docker start legalcare-local-crawler-web-1 legalcare-local-ai-delegate-web-1 legalcare-local-pg-receipt-worker-1
```

순서가 중요하다. 신규를 먼저 내려야 별칭(`crawler-web`/`ai-delegate-web`/`pg-receipt-worker`)과
컨슈머 그룹(`legalcare-local-pg-receipt-fixture-v1`)이 비고, 그 다음에 구 컨테이너를 올려야
도커 DNS 가 한 곳만 가리킨다. 반대로 하면 요청 절반이 엉뚱한 컨테이너로 샌다.

`legalcare-local` compose 묶음(`/mnt/ex_disk1/renew-replay/docker-compose*.yml` 5개)은 **어느 단계에서도
호출하지 않는다.** 파일 조합이 달라진 채로 `up` 하면 compose 가 공용 망을 재생성하려 들어 스택 전체가
끊긴다(2026-08-07 실사고). 이 절차는 그 묶음을 건드리지 않는다.
