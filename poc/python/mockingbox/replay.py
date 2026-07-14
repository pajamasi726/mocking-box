"""Sequential replay orchestrator.

For each request in the corpus:
    1. open attribution window on the old stack's binlog, send request, quiesce
    2. same against the new stack
    3. diff responses + write-sets -> verdict
"""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass

import httpx

from .binlog import BinlogCapture
from .config import Config, StackConfig
from .corpus import RequestSpec
from .diff import (
    ERROR,
    RequestResult,
    diff_responses,
    diff_writesets,
    normalize_writeset,
    verdict_of,
)

log = logging.getLogger(__name__)


@dataclass
class StackRuntime:
    config: StackConfig
    client: httpx.Client
    capture: BinlogCapture | None


class Replayer:
    def __init__(self, config: Config):
        self.config = config
        self.old = self._runtime(config.old, server_id=5501)
        self.new = self._runtime(config.new, server_id=5502)

    def _runtime(self, stack: StackConfig, server_id: int) -> StackRuntime:
        capture = None
        if stack.mysql is not None:
            capture = BinlogCapture(stack.name, stack.mysql, server_id=server_id)
        client = httpx.Client(
            base_url=stack.base_url, timeout=self.config.http_timeout_s
        )
        return StackRuntime(stack, client, capture)

    def start(self) -> None:
        for rt in (self.old, self.new):
            if rt.capture:
                rt.capture.start()

    def stop(self) -> None:
        for rt in (self.old, self.new):
            if rt.capture:
                rt.capture.stop()
            rt.client.close()

    # ------------------------------------------------------------------

    def run(self, corpus: list[RequestSpec]) -> list[RequestResult]:
        results = []
        total = len(corpus)
        for i, spec in enumerate(corpus, start=1):
            log.info("(%d/%d) %s  %s", i, total, spec.name, spec.describe())
            results.append(self._run_one(spec))
        return results

    def _run_one(self, spec: RequestSpec) -> RequestResult:
        result = RequestResult(name=spec.name, request=spec.describe(), verdict=ERROR)
        try:
            old_resp, old_changes = self._fire(self.old, spec)
            new_resp, new_changes = self._fire(self.new, spec)
        except Exception as exc:  # noqa: BLE001 - recorded per-request, replay continues
            log.error("  %s: %s", type(exc).__name__, exc)
            result.error = f"{type(exc).__name__}: {exc}"
            return result

        result.old_status, result.new_status = old_resp.status_code, new_resp.status_code
        response_diffs = diff_responses(
            old_resp.status_code,
            old_resp.text,
            new_resp.status_code,
            new_resp.text,
            self.config.noise.response_paths,
            dict(old_resp.headers),
            dict(new_resp.headers),
            self.config.compare_headers,
        )

        old_ws = normalize_writeset(
            old_changes, self.config.noise.columns, self.config.noise.tables_ignore
        )
        new_ws = normalize_writeset(
            new_changes, self.config.noise.columns, self.config.noise.tables_ignore
        )
        result.old_writes, result.new_writes = len(old_ws), len(new_ws)
        result.old_writeset, result.new_writeset = old_ws, new_ws
        writeset_diffs = diff_writesets(old_ws, new_ws)

        result.differences = response_diffs + writeset_diffs
        result.verdict = verdict_of(response_diffs, writeset_diffs)
        return result

    def _fire(self, rt: StackRuntime, spec: RequestSpec):
        if rt.capture:
            rt.capture.begin_window()

        kwargs: dict = {"headers": spec.headers}
        if isinstance(spec.body, (dict, list)):
            kwargs["content"] = json.dumps(spec.body)
            kwargs["headers"] = {"content-type": "application/json", **spec.headers}
        elif isinstance(spec.body, str):
            kwargs["content"] = spec.body

        response = rt.client.request(spec.method, spec.path, **kwargs)

        changes = (
            rt.capture.take_window(self.config.attribution) if rt.capture else []
        )
        return response, changes
