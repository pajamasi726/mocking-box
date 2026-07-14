# mocking-box

**Universal differential-testing box for backend rewrites.**
Replays captured HTTP traffic against an *old* and a *new* implementation, then diffs
**responses** AND **per-request DB write-sets** — so it keeps working even when the
framework/ORM changed completely (jOOQ → MyBatis, monolith consolidation, language port, …).

> Why not Keploy? Keploy records DB *wire-protocol* traffic as mocks, so replay breaks the
> moment your new ORM emits different SQL. mocking-box never looks at SQL text: it compares
> the **row changes** each request actually committed, read from the MySQL ROW binlog —
> the same mechanism Debezium uses. Different SQL, same effect ⇒ MATCH.
> See [research/01](research/01-differential-testing-tools.md) (tool landscape) and
> [research/02](research/02-architecture.md) (architecture) for the full story.

## How it works

```
corpus (JSONL/HAR) ──▶ Replayer ──▶ old stack ──▶ MySQL-old ─┐ binlog
   (sequential)          │  └─────▶ new stack ──▶ MySQL-new ─┤ (ROW)
                         ▼                                   ▼
                   response diff  ◀──────────────  write-set capture
                         └──────────▶ verdict per request ◀──┘
        MATCH | RESPONSE_DIFF | WRITESET_DIFF | BOTH_DIFF | ERROR
```

1. Each request is sent to the old stack, then the new stack (sequential, deterministic).
2. A binlog *attribution window* collects every row change committed until the DB
   quiesces (no binlog events for `quiet_ms` + no active InnoDB transactions).
3. Responses are deep-compared as JSON with noise rules (`**.updated_at`, …).
4. Write-sets are canonicalized (changed columns only, noise columns dropped,
   order-insensitive) and compared.

The target apps need **zero code changes**, any language/framework. Both MySQL instances
should start from the **same seed/snapshot** so auto-increment counters line up.

## Requirements

- Python 3.11+
- MySQL 5.7/8.x with ROW binlog. Recommended flags (see `demo/mysql/my.cnf`):
  ```
  binlog_format=ROW  binlog_row_image=FULL  binlog_row_metadata=FULL
  binlog_rows_query_log_events=ON   # optional: attaches SQL text for Level-1 attribution
  ```
- A MySQL user with `REPLICATION SLAVE, REPLICATION CLIENT` (+ `PROCESS` for quiesce checks).

## Quickstart (self-contained demo)

The demo spins up two MySQL instances and two deliberately different wallet apps:
the "old" one uses relative `UPDATE balance = balance + ?`, the "new" one uses
`SELECT ... FOR UPDATE` + absolute update — plus one **planted bug** (withdraw
forgets the history INSERT, while returning a perfectly identical HTTP response).

```bash
python3 -m venv .venv && .venv/bin/pip install -e .
cd demo
docker compose up -d --build
../.venv/bin/mockingbox run -c config.yaml --corpus corpus.jsonl
```

Expected output: 7 × `MATCH` (different SQL, same behavior) and 1 × `WRITESET_DIFF`
pinpointing the missing `wallet_history` INSERT. Exit code is non-zero on any
divergence, so it slots straight into CI.

Reset the demo state between runs (identical snapshots matter):

```bash
docker compose down -v && docker compose up -d
```

## Configuration

```yaml
old:
  base_url: "http://localhost:8081"
  mysql: { host: 127.0.0.1, port: 3307, user: root, password: root }
new:
  base_url: "http://localhost:8082"
  mysql: { host: 127.0.0.1, port: 3308, user: root, password: root }

attribution:
  quiet_ms: 300      # binlog silence required to close a request window
  timeout_ms: 5000   # hard cap per window

noise:
  response_paths: ["**.updated_at", "**.created_at", "**.trace_id"]
  columns: ["*.updated_at", "*.created_at"]   # table.column, * wildcards
  tables_ignore: []

report: { dir: "./report" }
```

Omit a stack's `mysql:` block to run response-diff only (no write-set capture).

## Corpus formats

- **JSONL** (native): one request per line —
  `{"name": "charge", "method": "POST", "path": "/wallet/3/charge", "body": {"amount": 5000}}`
- **HAR**: any browser/proxy capture (`mockingbox run --corpus traffic.har ...`)
- Converters for Keploy YAML / GoReplay `.gor` files: planned (v0.2).

## Current limitations (v0.1)

- **Attribution is Level 0** (sequential + quiesce window): fire-and-forget async writes
  that commit *after* the window may be attributed to the next request. A warning is
  logged when changes arrive between windows. Level 1 (trace-id in SQL comments via
  OTel/SQLCommenter, recovered from `Rows_query` binlog events) is wired but not exposed yet.
- Generated-ID drift: if a write bug makes auto-increment counters diverge, subsequent
  inserts report pk mismatches (cascade). Fix the first divergence and rerun; pk-mapping
  normalization is planned (v0.3).
- MySQL/MariaDB only for write-sets (Postgres via logical decoding / Debezium: v0.4).
- External side effects (mail, HTTP calls to third parties) are out of scope — pair with
  an egress mock/recorder if needed.

## Roadmap

| version | scope |
|---|---|
| v0.2 | noise auto-learning (replay old twice, collect self-mismatches), end-of-scenario full-state hash safety net, Keploy/GoReplay corpus converters, HTML report |
| v0.3 | Level-1 attribution (rid/SQLCommenter), generated-ID mapping, `{{last_created_id}}` templating for scenario chaining |
| v0.4 | Debezium-based capture (Postgres, etc.), parallel scenario lanes |

## Running the box itself in Docker

```bash
docker build -t mocking-box .
docker run --rm --network host \
  -v $PWD/demo/config.yaml:/config.yaml -v $PWD/demo/corpus.jsonl:/corpus.jsonl \
  -v $PWD/report:/report \
  mocking-box run -c /config.yaml --corpus /corpus.jsonl
```

## Development

```bash
.venv/bin/python -m pytest tests/   # diff-engine unit tests, no DB needed
```
