"""NEW stack — legalcare-renew 흉내 (MyBatis 스타일: FOR UPDATE + 절대 UPDATE).

의도적으로 심어둔 것:
  ①(버그) sign 시 contract_history INSERT 누락 — 응답은 구와 동일
     → 패시브 골든(응답만)으로는 안 잡히고, 직렬화 프록시 골든(write-set)이 잡음
  ②(버그) cancel 응답의 status 철자가 "CANCELED" — DB에는 정상 기록
     → 응답 diff가 잡음
  ③(매퍼 차이) 목록이 ORDER BY id DESC — sort_arrays 규칙으로 흡수해야 함
"""

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


def contract_json(cid, user_id, title, status, amount, updated_at):
    return {
        "contract_id": cid, "user_id": user_id, "title": title,
        "status": status, "amount": int(amount), "updated_at": updated_at.isoformat(),
    }


@app.get("/contract/<int:cid>")
def get_contract(cid):
    conn = db()
    try:
        with conn.cursor() as cur:
            # 다른 프로젝션 순서 (매퍼 차이)
            cur.execute(
                "SELECT user_id, title, status, amount, updated_at, id"
                " FROM contract WHERE id = %s", (cid,),
            )
            row = cur.fetchone()
        if row is None:
            return jsonify({"error_code": "C001", "message": "contract not found"}), 404
        return jsonify(contract_json(row[5], row[0], row[1], row[2], row[3], row[4]))
    finally:
        conn.close()


@app.get("/contracts")
def list_contracts():
    user_id = int(request.args.get("user_id", 0))
    conn = db()
    try:
        with conn.cursor() as cur:
            # ③ 매퍼 차이: 정렬 방향이 다름
            cur.execute(
                "SELECT user_id, title, status, amount, updated_at, id"
                " FROM contract WHERE user_id = %s ORDER BY id DESC", (user_id,),
            )
            rows = cur.fetchall()
        contracts = [contract_json(r[5], r[0], r[1], r[2], r[3], r[4]) for r in rows]
        return jsonify({"contracts": contracts, "total": len(contracts)})
    finally:
        conn.close()


@app.post("/contract")
def create_contract():
    body = request.get_json(force=True)
    conn = db()
    try:
        with conn.cursor() as cur:
            # 다른 컬럼 순서의 INSERT (매퍼 차이, 효과 동일)
            cur.execute(
                "INSERT INTO contract (amount, title, user_id) VALUES (%s, %s, %s)",
                (int(body.get("amount", 0)), body["title"], int(body["user_id"])),
            )
            cid = cur.lastrowid
            cur.execute("INSERT INTO contract_history (action, contract_id) VALUES ('CREATE', %s)", (cid,))
            cur.execute(
                "SELECT user_id, title, status, amount, updated_at FROM contract WHERE id = %s",
                (cid,),
            )
            row = cur.fetchone()
        conn.commit()
        return jsonify(contract_json(cid, row[0], row[1], row[2], row[3], row[4])), 201
    finally:
        conn.close()


def load_for_update(cur, cid):
    cur.execute("SELECT status FROM contract WHERE id = %s FOR UPDATE", (cid,))
    return cur.fetchone()


@app.post("/contract/<int:cid>/sign")
def sign(cid):
    conn = db()
    try:
        with conn.cursor() as cur:
            row = load_for_update(cur, cid)
            if row is None:
                conn.rollback()
                return jsonify({"error_code": "C001", "message": "contract not found"}), 404
            if row[0] != "DRAFT":
                conn.rollback()
                return jsonify({"error_code": "C002", "message": "invalid status transition"}), 409
            cur.execute("UPDATE contract SET status = 'SIGNED' WHERE id = %s", (cid,))
            # ① 심어둔 버그: contract_history INSERT가 리라이트에서 누락됨
        conn.commit()
        return jsonify({"contract_id": cid, "status": "SIGNED"})
    finally:
        conn.close()


@app.post("/contract/<int:cid>/cancel")
def cancel(cid):
    conn = db()
    try:
        with conn.cursor() as cur:
            row = load_for_update(cur, cid)
            if row is None:
                conn.rollback()
                return jsonify({"error_code": "C001", "message": "contract not found"}), 404
            if row[0] != "DRAFT":
                conn.rollback()
                return jsonify({"error_code": "C002", "message": "invalid status transition"}), 409
            cur.execute("UPDATE contract SET status = 'CANCELLED' WHERE id = %s", (cid,))
            cur.execute("INSERT INTO contract_history (action, contract_id) VALUES ('CANCEL', %s)", (cid,))
        conn.commit()
        # ② 심어둔 버그: 응답 철자가 다름 (DB에는 CANCELLED로 정상 기록)
        return jsonify({"contract_id": cid, "status": "CANCELED"})
    finally:
        conn.close()


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080, threaded=True)
