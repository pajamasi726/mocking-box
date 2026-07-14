# mocking-box

**Universal differential-testing box for backend rewrites.**
Replays captured HTTP traffic against an *old* and a *new* implementation, then diffs
**responses** AND **per-request DB write-sets** — so it keeps working even when the
framework/ORM changed completely (jOOQ → MyBatis, monolith consolidation, language port, …).

Single Go binary. Zero code changes in the systems under test. Any language, any framework.

> **Why not Keploy?** Keploy records DB *wire-protocol* traffic as mocks, so replay breaks
> the moment your new ORM emits different SQL. mocking-box never looks at SQL text: it
> compares the **row changes** each request actually committed, read from the MySQL ROW
> binlog — the same mechanism Debezium uses. Different SQL, same effect ⇒ `MATCH`.
> See [research/01](research/01-differential-testing-tools.md) (tool landscape) and
> [research/02](research/02-architecture.md) (architecture).

## Two verification modes

**Record & Verify** (Keploy-style, recommended) — capture once, verify forever.
The golden stores requests + recorded responses + per-request write-sets, so
verification needs **only the new stack running**:

```
[capture, once]  traffic ─▶ recording input ─▶ old stack ─▶ old DB ─ binlog ┐
                              └── golden = requests + responses + write-sets ◀┘
[verify, ∞]      golden ─▶ Replayer ─▶ new stack ─▶ new DB (seeded to capture-time state)
                       compare: recorded expectations vs live behavior
```

**Live parallel** (Diffy-style) — both stacks running, same request sent to each,
responses + write-sets diffed in real time.

Verdicts: `MATCH | RESPONSE_DIFF | WRITESET_DIFF | BOTH_DIFF | ERROR`.
Write-sets are read from the MySQL ROW binlog via an *attribution window*
(quiesce detection), canonicalized (changed columns only, noise columns dropped,
order-insensitive), and compared. See `research/02` and `research/03`.

## Capture inputs (box philosophy — never required in the request path)

| input | intrusion | command |
|---|---|---|
| recording proxy | in-path (dev/staging) | UI capture tab, golden checkbox |
| live NIC sniffing | side-car, `CAP_NET_RAW` | `mockingbox sniff --iface eth0 --port 8080 --out x.golden.jsonl` |
| tcpdump .pcap conversion | zero (offline) | `mockingbox convert --pcap dump.pcap --port 8080 --out x.golden.jsonl` |
| AWS VPC Traffic Mirroring | zero-touch on hosts | `mockingbox mirror --listen :4789 --port 8080 --out x.golden.jsonl` |

`--out *.golden.jsonl` records responses (Record & Verify); plain `.jsonl`
records requests only (live-parallel corpus). Passive inputs can't attribute
write-sets per request (concurrent traffic) — those goldens verify responses;
per-request write-set goldens come from the serialized recording proxy.
VPC Traffic Mirroring setup (console + Terraform): `research/03`.

## Quickstart (self-contained demo)

The demo spins up two MySQL instances and two deliberately different wallet apps:
the "old" one uses relative `UPDATE balance = balance + ?`, the "new" one uses
`SELECT ... FOR UPDATE` + absolute update — plus one **planted bug** (withdraw
forgets the history INSERT while returning a perfectly identical HTTP response).

```bash
go build -o bin/mockingbox ./cmd/mockingbox

cd demo
docker compose up -d --build
../bin/mockingbox run -c config.yaml --corpus corpus.jsonl   # live-parallel, CI mode
../bin/mockingbox ui  -c config.yaml                         # → http://localhost:8642

# Record & Verify: capture a golden through the UI (or API), then the old
# stack can be OFF — verification hits the new stack only:
docker stop demo-app-old-1 demo-mysql-old-1
../bin/mockingbox verify -c config.yaml --golden corpus/golden-demo.golden.jsonl
```

Expected: 7 × `MATCH` (different SQL, same behavior) and 1 × `WRITESET_DIFF`
pinpointing the missing `wallet_history` INSERT. Non-zero exit on any divergence,
so it slots straight into CI.

Reset demo state between runs (identical snapshots matter):

```bash
docker compose down -v && docker compose up -d
```

## Web console (`mockingbox ui`)

A single embedded web console (no external assets), English/한국어:

- **Dashboard** — run history, verdict tiles, per-request results, side-by-side
  write-set / response diffs
- **Capture** — start/stop a recording proxy from the browser: choose the listen
  port, the upstream server to forward to (old/new presets), and a corpus name;
  every request flowing through is appended to the corpus
- **Replay** — pick a corpus, run it against both stacks with a live progress bar
- **Settings** — edit the config (stack URLs, MySQL, noise rules, array-sort rules)
  in forms; saving rewrites the config file

## JSON comparison semantics

- Object **key order never matters** (mapper-dependent field order is ignored).
- **Array order** is significant by default; when a mapper/ORM legitimately
  reorders lists, add a sort rule so the array is canonically sorted before diffing:

```yaml
compare:
  sort_arrays:
    - { path: "data.items", by: "id" }         # sort by an element key
    - { path: "**",         by: "$canonical" } # or by whole-element canonical JSON
```

- Numbers compare scale-insensitively (`50000` == `50000.0`), UTF-8 (Korean etc.)
  compares byte-exact.

## Requirements

- MySQL 5.7/8.x with ROW binlog. Recommended flags (see `demo/mysql/my.cnf`):
  ```
  binlog_format=ROW  binlog_row_image=FULL  binlog_row_metadata=FULL
  binlog_rows_query_log_events=ON   # optional: attaches SQL text per change
  ```
- A MySQL user with `REPLICATION SLAVE, REPLICATION CLIENT` (+ `PROCESS` for quiesce checks).
- Go 1.23+ to build (release binaries later).

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
- **HAR**: any browser/proxy capture
- Converters for Keploy YAML / GoReplay `.gor`: planned (v0.2)

## Current limitations (v0.1)

- **Attribution is Level 0** (sequential + quiesce window): fire-and-forget async writes
  that commit *after* the window may be attributed to the next request (a warning is
  logged). Level 1 — trace-id in SQL comments via OTel/SQLCommenter, recovered from
  `Rows_query` binlog events — is wired in the capture layer but not exposed yet.
- Generated-ID drift: once a write bug makes auto-increment counters diverge, subsequent
  inserts report pk mismatches (cascade). Fix the first divergence and rerun.
- MySQL/MariaDB only for write-sets (Postgres via logical decoding: roadmap).
- External side effects (mail, third-party HTTP) are out of scope — pair with an egress
  recorder if needed.

## Roadmap

| version | scope |
|---|---|
| v0.3 | noise auto-learning, Keploy/GoReplay corpus converters, end-of-run full-state hash safety net, corpus editing in UI |
| v0.4 | Level-1 attribution (rid/SQLCommenter), generated-ID mapping, `{{last_created_id}}` scenario templating |
| v0.5 | Postgres (logical decoding), parallel scenario lanes, GoReleaser distribution (brew/docker) |

## Development

```bash
go test ./...        # diff-engine unit tests, no DB needed
go vet ./...
```

`poc/python/` contains the original Python proof-of-concept that validated the
architecture end-to-end; the Go implementation is the product line.

## License

Apache-2.0 (to be added before first public release).
