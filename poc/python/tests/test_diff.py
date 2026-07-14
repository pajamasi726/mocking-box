"""Unit tests for the diff engine (no DB required)."""

import datetime
import decimal

from mockingbox.binlog import RowChange
from mockingbox.diff import (
    MATCH,
    RESPONSE_DIFF,
    WRITESET_DIFF,
    diff_responses,
    diff_writesets,
    normalize_writeset,
    path_matches,
    strip_noise,
    verdict_of,
)

NOISE_COLS = ["*.created_at", "*.updated_at"]


def test_path_matches():
    assert path_matches("data.updated_at", ["data", "updated_at"])
    assert path_matches("**.updated_at", ["a", "b", "updated_at"])
    assert path_matches("**.updated_at", ["updated_at"])
    assert path_matches("data.*.id", ["data", "0", "id"])
    assert not path_matches("data.updated_at", ["data", "balance"])
    assert not path_matches("*.id", ["a", "b", "id"])


def test_strip_noise_nested():
    body = {
        "wallet_id": 1,
        "updated_at": "2026-07-14T10:00:00",
        "items": [{"id": 1, "created_at": "x"}, {"id": 2, "created_at": "y"}],
    }
    cleaned = strip_noise(body, ["**.updated_at", "**.created_at"])
    assert cleaned == {"wallet_id": 1, "items": [{"id": 1}, {"id": 2}]}


def test_response_diff_ignores_noise_and_key_order():
    old = '{"wallet_id": 1, "balance": 55000, "updated_at": "2026-07-14T10:00:00"}'
    new = '{"updated_at": "2026-07-14T10:00:03", "balance": 55000, "wallet_id": 1}'
    diffs = diff_responses(200, old, 200, new, ["**.updated_at"])
    assert diffs == []


def test_response_diff_catches_real_change():
    old = '{"balance": 55000}'
    new = '{"balance": 56000}'
    diffs = diff_responses(200, old, 200, new, [])
    assert len(diffs) == 1
    assert diffs[0].path == "balance"


def test_response_diff_status():
    diffs = diff_responses(200, "{}", 404, "{}", [])
    assert any(d.path == "status" for d in diffs)


def _charge_changes_old():
    """Old stack: relative UPDATE + INSERT (jOOQ-ish)."""
    return [
        RowChange(
            "demo.wallet",
            "UPDATE",
            {"id": 1, "user_id": 101, "balance": decimal.Decimal("50000"),
             "updated_at": datetime.datetime(2026, 7, 14, 10, 0, 0),
             "created_at": datetime.datetime(2026, 1, 1)},
            {"id": 1, "user_id": 101, "balance": decimal.Decimal("55000"),
             "updated_at": datetime.datetime(2026, 7, 14, 10, 0, 1),
             "created_at": datetime.datetime(2026, 1, 1)},
            query="UPDATE wallet SET balance = balance + 5000 WHERE id = 1",
        ),
        RowChange(
            "demo.wallet_history",
            "INSERT",
            None,
            {"id": 1, "wallet_id": 1, "type": "CHARGE",
             "amount": decimal.Decimal("5000"),
             "balance_after": decimal.Decimal("55000"),
             "created_at": datetime.datetime(2026, 7, 14, 10, 0, 1)},
            query="INSERT INTO wallet_history (wallet_id, type, amount, balance_after) VALUES ...",
        ),
    ]


def _charge_changes_new():
    """New stack: same effect, different SQL, different execution order,
    different timestamps."""
    return [
        RowChange(  # history INSERT first (different ordering)
            "demo.wallet_history",
            "INSERT",
            None,
            {"id": 1, "wallet_id": 1, "type": "CHARGE",
             "amount": decimal.Decimal("5000"),
             "balance_after": decimal.Decimal("55000"),
             "created_at": datetime.datetime(2026, 7, 14, 10, 0, 7)},
            query="INSERT INTO wallet_history (type, balance_after, amount, wallet_id) VALUES ...",
        ),
        RowChange(
            "demo.wallet",
            "UPDATE",
            {"id": 1, "user_id": 101, "balance": decimal.Decimal("50000"),
             "updated_at": datetime.datetime(2026, 7, 14, 10, 0, 0),
             "created_at": datetime.datetime(2026, 1, 1)},
            {"id": 1, "user_id": 101, "balance": decimal.Decimal("55000"),
             "updated_at": datetime.datetime(2026, 7, 14, 10, 0, 7),
             "created_at": datetime.datetime(2026, 1, 1)},
            query="UPDATE wallet SET updated_at = NOW(), balance = 55000 WHERE id = 1",
        ),
    ]


def test_writeset_match_despite_different_sql_and_order():
    old_ws = normalize_writeset(_charge_changes_old(), NOISE_COLS, [])
    new_ws = normalize_writeset(_charge_changes_new(), NOISE_COLS, [])
    assert diff_writesets(old_ws, new_ws) == []
    assert verdict_of([], diff_writesets(old_ws, new_ws)) == MATCH


def test_writeset_detects_missing_insert():
    old_ws = normalize_writeset(_charge_changes_old(), NOISE_COLS, [])
    new_changes = [_charge_changes_new()[1]]  # history INSERT lost
    new_ws = normalize_writeset(new_changes, NOISE_COLS, [])
    diffs = diff_writesets(old_ws, new_ws)
    assert len(diffs) == 1
    assert diffs[0].new == "<absent>"
    assert "wallet_history" in diffs[0].path
    assert verdict_of([], diffs) == WRITESET_DIFF


def test_writeset_detects_wrong_value():
    old_ws = normalize_writeset(_charge_changes_old(), NOISE_COLS, [])
    bad = _charge_changes_new()
    bad[1].after["balance"] = decimal.Decimal("56000")  # wrong computation
    new_ws = normalize_writeset(bad, NOISE_COLS, [])
    diffs = diff_writesets(old_ws, new_ws)
    assert any("balance" in d.path for d in diffs)


def test_noise_only_update_is_dropped():
    touch = [
        RowChange(
            "demo.wallet",
            "UPDATE",
            {"id": 1, "balance": decimal.Decimal("50000"),
             "updated_at": datetime.datetime(2026, 7, 14, 10, 0, 0)},
            {"id": 1, "balance": decimal.Decimal("50000"),
             "updated_at": datetime.datetime(2026, 7, 14, 10, 0, 5)},
        )
    ]
    assert normalize_writeset(touch, NOISE_COLS, []) == []


def test_decimal_scale_insensitive():
    a = [RowChange("demo.wallet", "UPDATE",
                   {"id": 1, "balance": decimal.Decimal("50000")},
                   {"id": 1, "balance": decimal.Decimal("55000")})]
    b = [RowChange("demo.wallet", "UPDATE",
                   {"id": 1, "balance": decimal.Decimal("50000.0")},
                   {"id": 1, "balance": decimal.Decimal("55000.0")})]
    assert diff_writesets(
        normalize_writeset(a, [], []), normalize_writeset(b, [], [])
    ) == []


def test_ignored_tables():
    changes = [
        RowChange("demo._replay_marker", "INSERT", None, {"id": 9, "rid": 1}),
    ]
    assert normalize_writeset(changes, [], ["_replay_marker"]) == []


def test_verdicts():
    resp = diff_responses(200, '{"a":1}', 200, '{"a":2}', [])
    assert verdict_of(resp, []) == RESPONSE_DIFF
    assert verdict_of([], []) == MATCH
