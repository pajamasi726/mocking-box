"""Write-set capture from MySQL ROW binlog.

A BinlogCapture connects to one stack's MySQL as a replication client
(the same mechanism Debezium uses) and streams row events on a background
thread. The replayer opens an attribution *window* per request: every row
change committed between window start and quiesce belongs to that request
(Level 0 attribution — see research/02-architecture.md §4).

If binlog_rows_query_log_events=ON, the originating SQL text (including any
/* rid=... */ comment injected by SQLCommenter/datasource-proxy) is attached
to each change, so Level 1 attribution can be layered on later.
"""

from __future__ import annotations

import logging
import threading
import time
from dataclasses import dataclass, field

import pymysql
from pymysqlreplication import BinLogStreamReader
from pymysqlreplication.event import RowsQueryLogEvent
from pymysqlreplication.row_event import (
    DeleteRowsEvent,
    UpdateRowsEvent,
    WriteRowsEvent,
)

from .config import AttributionConfig, MysqlConfig

log = logging.getLogger(__name__)


@dataclass
class RowChange:
    """One committed row change, normalized from a binlog row event."""

    table: str
    op: str  # INSERT | UPDATE | DELETE
    before: dict | None
    after: dict | None
    query: str | None = None  # Rows_query text when available (Level 1 hook)


@dataclass
class _Buffer:
    changes: list[RowChange] = field(default_factory=list)
    last_event_monotonic: float = 0.0


class BinlogCapture:
    """Streams row events from one MySQL server into a window buffer."""

    def __init__(self, name: str, mysql: MysqlConfig, server_id: int):
        self.name = name
        self.mysql = mysql
        self.server_id = server_id
        self._lock = threading.Lock()
        self._buf = _Buffer()
        self._stream: BinLogStreamReader | None = None
        self._thread: threading.Thread | None = None
        self._stop = threading.Event()
        self._error: Exception | None = None

    # -- lifecycle -----------------------------------------------------------

    def start(self) -> None:
        log_file, log_pos = self._current_position()
        self._stream = BinLogStreamReader(
            connection_settings=self.mysql.connection_settings(),
            server_id=self.server_id,
            blocking=True,
            resume_stream=True,
            log_file=log_file,
            log_pos=log_pos,
            only_events=[WriteRowsEvent, UpdateRowsEvent, DeleteRowsEvent, RowsQueryLogEvent],
        )
        self._thread = threading.Thread(
            target=self._pump, name=f"binlog-{self.name}", daemon=True
        )
        self._thread.start()
        log.info("[%s] binlog capture started at %s:%s", self.name, log_file, log_pos)

    def stop(self) -> None:
        self._stop.set()
        # NOTE: BinLogStreamReader.close() deadlocks if called while the pump
        # thread is blocked reading the socket. Close the raw socket instead
        # to unblock the reader; the daemon thread then exits on its own.
        stream = self._stream
        if stream is None:
            return
        try:
            conn = getattr(stream, "_stream_connection", None)
            sock = getattr(conn, "_sock", None) if conn is not None else None
            if sock is not None:
                sock.close()
        except Exception:  # noqa: BLE001 - best-effort teardown
            pass

    def _current_position(self) -> tuple[str, int]:
        conn = pymysql.connect(
            host=self.mysql.host,
            port=self.mysql.port,
            user=self.mysql.user,
            password=self.mysql.password,
        )
        try:
            with conn.cursor() as cur:
                try:
                    cur.execute("SHOW MASTER STATUS")  # MySQL <= 8.0
                except pymysql.err.ProgrammingError:
                    cur.execute("SHOW BINARY LOG STATUS")  # MySQL >= 8.4
                row = cur.fetchone()
                if not row:
                    raise RuntimeError(
                        f"[{self.name}] binary logging appears disabled on "
                        f"{self.mysql.host}:{self.mysql.port}"
                    )
                return str(row[0]), int(row[1])
        finally:
            conn.close()

    # -- pump thread ---------------------------------------------------------

    def _pump(self) -> None:
        last_query: str | None = None
        try:
            for event in self._stream:  # blocks; daemon thread dies with process
                if self._stop.is_set():
                    break
                if isinstance(event, RowsQueryLogEvent):
                    last_query = getattr(event, "query", None)
                    continue
                changes = self._normalize(event, last_query)
                with self._lock:
                    self._buf.changes.extend(changes)
                    self._buf.last_event_monotonic = time.monotonic()
        except Exception as exc:  # noqa: BLE001 - surfaced to the replayer via take_window
            if not self._stop.is_set():
                self._error = exc
                log.error("[%s] binlog stream died: %s", self.name, exc)

    @staticmethod
    def _normalize(event, last_query: str | None) -> list[RowChange]:
        table = f"{event.schema}.{event.table}" if event.schema else event.table
        out: list[RowChange] = []
        if isinstance(event, WriteRowsEvent):
            for row in event.rows:
                out.append(RowChange(table, "INSERT", None, dict(row["values"]), last_query))
        elif isinstance(event, DeleteRowsEvent):
            for row in event.rows:
                out.append(RowChange(table, "DELETE", dict(row["values"]), None, last_query))
        elif isinstance(event, UpdateRowsEvent):
            for row in event.rows:
                out.append(
                    RowChange(
                        table,
                        "UPDATE",
                        dict(row["before_values"]),
                        dict(row["after_values"]),
                        last_query,
                    )
                )
        return out

    # -- attribution window (Level 0) ----------------------------------------

    def begin_window(self) -> None:
        with self._lock:
            leftovers = len(self._buf.changes)
            if leftovers:
                log.warning(
                    "[%s] %d row change(s) arrived between windows (async leak?) — "
                    "they will be attributed to the next request",
                    self.name,
                    leftovers,
                )
            self._buf.last_event_monotonic = 0.0

    def take_window(self, attribution: AttributionConfig) -> list[RowChange]:
        """Wait for quiesce, then return (and clear) changes seen in this window."""
        if self._error is not None:
            raise RuntimeError(f"[{self.name}] binlog capture failed") from self._error

        quiet_s = attribution.quiet_ms / 1000.0
        deadline = time.monotonic() + attribution.timeout_ms / 1000.0
        window_opened = time.monotonic()

        while True:
            now = time.monotonic()
            with self._lock:
                last = self._buf.last_event_monotonic or window_opened
            quiet = (now - last) >= quiet_s
            if quiet and (
                not attribution.check_innodb_trx or not self._has_active_trx()
            ):
                break
            if now >= deadline:
                log.warning(
                    "[%s] quiesce timeout after %dms — window may be truncated",
                    self.name,
                    attribution.timeout_ms,
                )
                break
            time.sleep(0.05)

        with self._lock:
            changes = self._buf.changes
            self._buf.changes = []
            return changes

    def _has_active_trx(self) -> bool:
        try:
            conn = pymysql.connect(
                host=self.mysql.host,
                port=self.mysql.port,
                user=self.mysql.user,
                password=self.mysql.password,
            )
            try:
                with conn.cursor() as cur:
                    cur.execute("SELECT COUNT(*) FROM information_schema.innodb_trx")
                    (count,) = cur.fetchone()
                    return int(count) > 0
            finally:
                conn.close()
        except Exception as exc:  # noqa: BLE001 - quiesce check is best-effort
            log.debug("[%s] innodb_trx check failed: %s", self.name, exc)
            return False
