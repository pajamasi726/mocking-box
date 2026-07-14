"""Demo NEW stack — emulates the rewritten service (MyBatis-style SQL).

Deliberately different SQL shapes from app_old for the SAME business effect:
  - charge: SELECT ... FOR UPDATE, compute in app, absolute UPDATE with
    explicit updated_at, INSERT with a different column order.
  - PLANTED BUG: withdraw forgets to insert the wallet_history row.
    mocking-box must flag it as WRITESET_DIFF even though the HTTP
    response is identical to the old stack's.
"""

import os

import pymysql
from flask import Flask, jsonify, request

app = Flask(__name__)


def db():
    return pymysql.connect(
        host=os.environ.get("DB_HOST", "127.0.0.1"),
        port=int(os.environ.get("DB_PORT", 3306)),
        user="root",
        password="root",
        database="demo",
        autocommit=False,
    )


def wallet_json(wallet_id, user_id, balance, updated_at):
    return {
        "wallet_id": wallet_id,
        "user_id": user_id,
        "balance": int(balance),
        "updated_at": updated_at.isoformat(),
    }


@app.get("/wallet/<int:wallet_id>")
def get_wallet(wallet_id):
    conn = db()
    try:
        with conn.cursor() as cur:
            # different projection order than the old stack
            cur.execute(
                "SELECT user_id, balance, updated_at, id FROM wallet WHERE id = %s",
                (wallet_id,),
            )
            row = cur.fetchone()
        if row is None:
            return jsonify({"error_code": "W001", "message": "wallet not found"}), 404
        return jsonify(wallet_json(row[3], row[0], row[1], row[2]))
    finally:
        conn.close()


@app.post("/wallet/<int:wallet_id>/charge")
def charge(wallet_id):
    amount = int(request.get_json(force=True)["amount"])
    if amount <= 0:
        return jsonify({"error_code": "T001", "message": "invalid amount"}), 400
    conn = db()
    try:
        with conn.cursor() as cur:
            # rewrite style: pessimistic lock, compute in app, absolute UPDATE
            cur.execute(
                "SELECT balance, user_id FROM wallet WHERE id = %s FOR UPDATE",
                (wallet_id,),
            )
            row = cur.fetchone()
            if row is None:
                conn.rollback()
                return jsonify({"error_code": "W001", "message": "wallet not found"}), 404
            new_balance = int(row[0]) + amount
            cur.execute(
                "UPDATE wallet SET updated_at = NOW(), balance = %s WHERE id = %s",
                (new_balance, wallet_id),
            )
            # different column order than the old stack's INSERT
            cur.execute(
                "INSERT INTO wallet_history (type, balance_after, amount, wallet_id)"
                " VALUES ('CHARGE', %s, %s, %s)",
                (new_balance, amount, wallet_id),
            )
            cur.execute("SELECT updated_at FROM wallet WHERE id = %s", (wallet_id,))
            updated_at = cur.fetchone()[0]
        conn.commit()
        return jsonify(wallet_json(wallet_id, row[1], new_balance, updated_at))
    finally:
        conn.close()


@app.post("/wallet/<int:wallet_id>/withdraw")
def withdraw(wallet_id):
    amount = int(request.get_json(force=True)["amount"])
    if amount <= 0:
        return jsonify({"error_code": "T001", "message": "invalid amount"}), 400
    conn = db()
    try:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT balance, user_id FROM wallet WHERE id = %s FOR UPDATE",
                (wallet_id,),
            )
            row = cur.fetchone()
            if row is None:
                conn.rollback()
                return jsonify({"error_code": "W001", "message": "wallet not found"}), 404
            if int(row[0]) < amount:
                conn.rollback()
                return jsonify({"error_code": "W002", "message": "insufficient balance"}), 400
            new_balance = int(row[0]) - amount
            cur.execute(
                "UPDATE wallet SET updated_at = NOW(), balance = %s WHERE id = %s",
                (new_balance, wallet_id),
            )
            # PLANTED BUG: wallet_history INSERT was lost in the rewrite.
            cur.execute("SELECT updated_at FROM wallet WHERE id = %s", (wallet_id,))
            updated_at = cur.fetchone()[0]
        conn.commit()
        return jsonify(wallet_json(wallet_id, row[1], new_balance, updated_at))
    finally:
        conn.close()


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
