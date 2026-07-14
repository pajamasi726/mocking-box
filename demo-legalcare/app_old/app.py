"""OLD stack — lawkit-contract 흉내 (jOOQ 스타일: 상대 UPDATE, ORDER BY id ASC)."""

import os

import pymysql
from flask import Flask, jsonify, request

app = Flask(__name__)


def db():
    return pymysql.connect(
        host=os.environ.get("DB_HOST", "127.0.0.1"),
        port=int(os.environ.get("DB_PORT", 3306)),
        user="root", password="root", database="lawkit", autocommit=False,
        charset="utf8mb4",
    )


def contract_json(row):
    return {
        "contract_id": row[0], "user_id": row[1], "title": row[2],
        "status": row[3], "amount": int(row[4]), "updated_at": row[5].isoformat(),
    }


COLS = "id, user_id, title, status, amount, updated_at"


@app.get("/contract/<int:cid>")
def get_contract(cid):
    conn = db()
    try:
        with conn.cursor() as cur:
            cur.execute(f"SELECT {COLS} FROM contract WHERE id = %s", (cid,))
            row = cur.fetchone()
        if row is None:
            return jsonify({"error_code": "C001", "message": "contract not found"}), 404
        return jsonify(contract_json(row))
    finally:
        conn.close()


@app.get("/contracts")
def list_contracts():
    user_id = int(request.args.get("user_id", 0))
    conn = db()
    try:
        with conn.cursor() as cur:
            cur.execute(
                f"SELECT {COLS} FROM contract WHERE user_id = %s ORDER BY id ASC",
                (user_id,),
            )
            rows = cur.fetchall()
        return jsonify({"contracts": [contract_json(r) for r in rows], "total": len(rows)})
    finally:
        conn.close()


@app.post("/contract")
def create_contract():
    body = request.get_json(force=True)
    conn = db()
    try:
        with conn.cursor() as cur:
            cur.execute(
                "INSERT INTO contract (user_id, title, amount) VALUES (%s, %s, %s)",
                (int(body["user_id"]), body["title"], int(body.get("amount", 0))),
            )
            cid = cur.lastrowid
            cur.execute("INSERT INTO contract_history (contract_id, action) VALUES (%s, 'CREATE')", (cid,))
            cur.execute(f"SELECT {COLS} FROM contract WHERE id = %s", (cid,))
            row = cur.fetchone()
        conn.commit()
        return jsonify(contract_json(row)), 201
    finally:
        conn.close()


def transition(cid, from_status, to_status, action):
    conn = db()
    try:
        with conn.cursor() as cur:
            # jOOQ 스타일: 조건부 상대 UPDATE 한 방
            cur.execute(
                "UPDATE contract SET status = %s WHERE id = %s AND status = %s",
                (to_status, cid, from_status),
            )
            if cur.rowcount == 0:
                conn.rollback()
                cur2 = conn.cursor()
                cur2.execute("SELECT COUNT(*) FROM contract WHERE id = %s", (cid,))
                exists = cur2.fetchone()[0]
                if not exists:
                    return jsonify({"error_code": "C001", "message": "contract not found"}), 404
                return jsonify({"error_code": "C002", "message": "invalid status transition"}), 409
            cur.execute(
                "INSERT INTO contract_history (contract_id, action) VALUES (%s, %s)",
                (cid, action),
            )
        conn.commit()
        return jsonify({"contract_id": cid, "status": to_status})
    finally:
        conn.close()


@app.post("/contract/<int:cid>/sign")
def sign(cid):
    return transition(cid, "DRAFT", "SIGNED", "SIGN")


@app.post("/contract/<int:cid>/cancel")
def cancel(cid):
    return transition(cid, "DRAFT", "CANCELLED", "CANCEL")


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080, threaded=True)
