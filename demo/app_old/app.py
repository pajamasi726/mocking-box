"""Demo OLD stack — emulates the legacy service (jOOQ-style SQL).

Charge uses a single relative UPDATE (`balance = balance + %s`), the way a
query-builder ORM tends to. Response/write behavior is the reference.
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


def wallet_json(row):
    return {
        "wallet_id": row[0],
        "user_id": row[1],
        "balance": int(row[2]),
        "updated_at": row[3].isoformat(),
    }


@app.get("/wallet/<int:wallet_id>")
def get_wallet(wallet_id):
    conn = db()
    try:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT id, user_id, balance, updated_at FROM wallet WHERE id = %s",
                (wallet_id,),
            )
            row = cur.fetchone()
        if row is None:
            return jsonify({"error_code": "W001", "message": "wallet not found"}), 404
        return jsonify(wallet_json(row))
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
            # legacy style: relative UPDATE in one statement
            cur.execute(
                "UPDATE wallet SET balance = balance + %s WHERE id = %s",
                (amount, wallet_id),
            )
            if cur.rowcount == 0:
                conn.rollback()
                return jsonify({"error_code": "W001", "message": "wallet not found"}), 404
            cur.execute(
                "SELECT id, user_id, balance, updated_at FROM wallet WHERE id = %s",
                (wallet_id,),
            )
            row = cur.fetchone()
            cur.execute(
                "INSERT INTO wallet_history (wallet_id, type, amount, balance_after)"
                " VALUES (%s, 'CHARGE', %s, %s)",
                (wallet_id, amount, int(row[2])),
            )
        conn.commit()
        return jsonify(wallet_json(row))
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
                "SELECT balance FROM wallet WHERE id = %s FOR UPDATE", (wallet_id,)
            )
            row = cur.fetchone()
            if row is None:
                conn.rollback()
                return jsonify({"error_code": "W001", "message": "wallet not found"}), 404
            if int(row[0]) < amount:
                conn.rollback()
                return jsonify({"error_code": "W002", "message": "insufficient balance"}), 400
            cur.execute(
                "UPDATE wallet SET balance = balance - %s WHERE id = %s",
                (amount, wallet_id),
            )
            cur.execute(
                "SELECT id, user_id, balance, updated_at FROM wallet WHERE id = %s",
                (wallet_id,),
            )
            row = cur.fetchone()
            cur.execute(
                "INSERT INTO wallet_history (wallet_id, type, amount, balance_after)"
                " VALUES (%s, 'WITHDRAW', %s, %s)",
                (wallet_id, amount, int(row[2])),
            )
        conn.commit()
        return jsonify(wallet_json(row))
    finally:
        conn.close()


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
