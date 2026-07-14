# AI로 조종하기 — MCP 서버

mocking-box를 **대화로** 조종하세요. AI 어시스턴트(Claude Desktop, Cursor, Claude Code 등)에
mocking-box를 도구로 등록하면, "우리 운영 DB에 붙여서 검증해봐" 같은 말로
연결 설정 → 탐색 → 복사 → 검증을 AI가 대신 수행합니다.

우리 팀은 코드·자격증명을 알아서 직접 하지만, **다른 사람은 자기 AI한테 시키면** 되도록
만든 인터페이스입니다.

## 구조

```
AI 어시스턴트  ──(MCP stdio)──▶  mockingbox mcp  ──(HTTP)──▶  대시보드  ──▶  DB·앱
```

MCP 서버는 실행 중인 대시보드의 API를 감쌉니다. 대시보드를 먼저 띄워두세요
(`mockingbox dashboard -c config.yaml`).

## 등록 (Claude Desktop 예)

`claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mocking-box": {
      "command": "mockingbox",
      "args": ["mcp", "--dashboard", "http://192.168.1.55:8643"]
    }
  }
}
```

토큰을 쓰면 `"--token", "<secret>"` 추가.

## AI가 쓸 수 있는 도구

| 도구 | 하는 일 | 안전성 |
|---|---|---|
| `get_config` | 현재 설정 조회 | 읽기 |
| `test_connection` | DB 접속 확인(호스트·계정 검증) | **읽기전용** — 운영 안전 |
| `set_config` | 검증 설정 채우기 (연결정보 등) | DB 무접촉 |
| `health` | 설정된 API·DB 연결 상태 확인 | 읽기 |
| `discover_db` | 스키마·테이블·행수·크기 나열 | **읽기전용** — 운영 안전 |
| `copy_db` | 운영 → 검증 DB 복사 | **쓰기 · `confirm:true` 필수** |
| `list_recordings` | 녹화본 목록 | 읽기 |
| `verify` | 녹화본 재생·검증 | **쓰기 · `confirm:true` 필수** |
| `list_runs` / `get_results` | 검증 결과 조회 | 읽기 |

## 안전장치

- **운영 DB는 읽기(SELECT)만**: `test_connection`·`discover_db`는 접속·조회만.
  운영에 write가 갈 일이 없습니다.
- **쓰기 작업은 사람 확인 필수**: `copy_db`·`verify`는 `confirm:true` 인자가 없으면
  거부됩니다 — AI가 실수로 데이터를 옮기거나 검증을 돌리지 못하게, 대화에서 사람이
  먼저 승인하도록.

## 대화 예시

```
사용자: legalcare-renew를 운영 medilawyer 데이터로 검증하고 싶어. 접속은 …
AI:    (test_connection) 운영 medilawyer 연결 확인 — wal_level=logical OK.
       (discover_db) 93개 테이블, boost_dental_image가 975GB네요. 이건 제외할까요?
사용자: 응 그거 빼고 복사해
AI:    (set_config로 exclude 설정) 준비됐습니다. 복사를 실행할까요? (~16GB)
사용자: 해
AI:    (copy_db confirm:true) 복사 완료 — 92 테이블. 이제 녹화본으로 검증할까요?
```
