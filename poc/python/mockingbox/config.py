"""Configuration loading for mocking-box."""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path

import yaml


@dataclass
class MysqlConfig:
    host: str
    port: int = 3306
    user: str = "root"
    password: str = ""

    def connection_settings(self) -> dict:
        return {
            "host": self.host,
            "port": self.port,
            "user": self.user,
            "passwd": self.password,
        }


@dataclass
class StackConfig:
    """One side of the comparison: a running app + the MySQL it writes to."""

    name: str
    base_url: str
    mysql: MysqlConfig | None = None  # None = response-diff only (no write-set)


@dataclass
class AttributionConfig:
    # Level 0: sequential replay + quiesce window (see research/02-architecture.md §4)
    quiet_ms: int = 300
    timeout_ms: int = 5000
    check_innodb_trx: bool = True


@dataclass
class NoiseConfig:
    # dotted-path patterns pruned from JSON bodies before comparison
    # segments: literal | * (one segment) | ** (any depth)
    response_paths: list[str] = field(default_factory=list)
    # "table.column" patterns pruned from write-sets ("*" wildcard allowed per part)
    columns: list[str] = field(default_factory=list)
    # tables whose changes are ignored entirely
    tables_ignore: list[str] = field(default_factory=list)


@dataclass
class Config:
    old: StackConfig
    new: StackConfig
    attribution: AttributionConfig
    noise: NoiseConfig
    report_dir: Path
    http_timeout_s: float = 10.0
    compare_headers: list[str] = field(default_factory=list)


def _stack(name: str, raw: dict) -> StackConfig:
    mysql = None
    if raw.get("mysql"):
        m = raw["mysql"]
        mysql = MysqlConfig(
            host=m["host"],
            port=int(m.get("port", 3306)),
            user=m.get("user", "root"),
            password=str(m.get("password", "")),
        )
    return StackConfig(name=name, base_url=raw["base_url"].rstrip("/"), mysql=mysql)


def load_config(path: str | Path) -> Config:
    raw = yaml.safe_load(Path(path).read_text())

    attribution = AttributionConfig(**(raw.get("attribution") or {}))
    noise_raw = raw.get("noise") or {}
    noise = NoiseConfig(
        response_paths=list(noise_raw.get("response_paths") or []),
        columns=list(noise_raw.get("columns") or []),
        tables_ignore=list(noise_raw.get("tables_ignore") or []),
    )
    report = raw.get("report") or {}

    return Config(
        old=_stack("old", raw["old"]),
        new=_stack("new", raw["new"]),
        attribution=attribution,
        noise=noise,
        report_dir=Path(report.get("dir", "./report")),
        http_timeout_s=float(raw.get("http_timeout_s", 10.0)),
        compare_headers=[h.lower() for h in (raw.get("compare_headers") or [])],
    )
