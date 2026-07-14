"""mocking-box CLI.

Usage:
    mockingbox run -c config.yaml --corpus corpus.jsonl
"""

from __future__ import annotations

import argparse
import logging
import sys

from .config import load_config
from .corpus import load_corpus
from .diff import MATCH
from .replay import Replayer
from .report import print_console, write_json


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        prog="mockingbox",
        description=(
            "Replay captured HTTP traffic against an old and a new backend, "
            "then diff responses AND per-request DB write-sets."
        ),
    )
    sub = parser.add_subparsers(dest="command", required=True)

    run = sub.add_parser("run", help="replay a corpus and report divergences")
    run.add_argument("-c", "--config", required=True, help="config YAML path")
    run.add_argument("--corpus", required=True, help="corpus file (.jsonl or .har)")
    run.add_argument("-v", "--verbose", action="store_true")

    args = parser.parse_args(argv)
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(asctime)s %(levelname)-7s %(message)s",
        datefmt="%H:%M:%S",
    )
    # binlog stream internals are chatty at INFO
    logging.getLogger("pymysqlreplication").setLevel(logging.WARNING)

    config = load_config(args.config)
    corpus = load_corpus(args.corpus)
    logging.info(
        "corpus: %d request(s) | old=%s new=%s",
        len(corpus),
        config.old.base_url,
        config.new.base_url,
    )

    replayer = Replayer(config)
    replayer.start()
    try:
        results = replayer.run(corpus)
    finally:
        replayer.stop()

    print_console(results)
    report_path = write_json(results, config.report_dir)
    logging.info("JSON report: %s", report_path)

    return 0 if all(r.verdict == MATCH for r in results) else 1


if __name__ == "__main__":
    sys.exit(main())
