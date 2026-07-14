"""OLD stack — jOOQ 스타일 (상대 UPDATE, ORDER BY id ASC)."""
import os
import psycopg
from flask import Flask, jsonify, request

app = Flask(__name__)


def db():
    return psycopg.connect(
        host=os.environ.get("DB_HOST", "127.0.0.1"),
        port=int(os.environ.get("DB_PORT", 5432)),
        user="postgres", password="postgres", dbname="postgres", autocommit=False,
    )


def review_json(r):
    return {"review_id": r[0], "hospital_id": r[1], "channel": r[2],
            "status": r[3], "score": float(r[4]), "updated_at": r[5].isoformat()}


COLS = "id, hospital_id, channel, status, score, updated_at"


@app.get("/review/<int:rid>")
def get_review(rid):
    with db() as c, c.cursor() as cur:
        cur.execute(f"SELECT {COLS} FROM booster.review WHERE id = %s", (rid,))
        row = cur.fetchone()
    if not row:
        return jsonify({"error_code": "R001", "message": "not found"}), 404
    return jsonify(review_json(row))


@app.get("/reviews")
def list_reviews():
    hid = int(request.args.get("hospital_id", 0))
    with db() as c, c.cursor() as cur:
        cur.execute(f"SELECT {COLS} FROM booster.review WHERE hospital_id = %s ORDER BY id ASC", (hid,))
        rows = cur.fetchall()
    return jsonify({"reviews": [review_json(r) for r in rows], "total": len(rows)})


@app.post("/review")
def create_review():
    b = request.get_json(force=True)
    with db() as c, c.cursor() as cur:
        cur.execute("INSERT INTO booster.review (hospital_id, channel, score) VALUES (%s,%s,%s) RETURNING " + COLS,
                    (int(b["hospital_id"]), b["channel"], float(b.get("score", 0))))
        row = cur.fetchone()
        cur.execute("INSERT INTO booster.review_history (review_id, action) VALUES (%s, 'CREATE')", (row[0],))
        c.commit()
    return jsonify(review_json(row)), 201


@app.post("/review/<int:rid>/publish")
def publish(rid):
    with db() as c, c.cursor() as cur:
        cur.execute("UPDATE booster.review SET status='PUBLISHED' WHERE id=%s AND status='DRAFT'", (rid,))
        if cur.rowcount == 0:
            c.rollback()
            cur.execute("SELECT 1 FROM booster.review WHERE id=%s", (rid,))
            return (jsonify({"error_code": "R002", "message": "invalid transition"}), 409) if cur.fetchone() \
                else (jsonify({"error_code": "R001", "message": "not found"}), 404)
        cur.execute("INSERT INTO booster.review_history (review_id, action) VALUES (%s, 'PUBLISH')", (rid,))
        c.commit()
    return jsonify({"review_id": rid, "status": "PUBLISHED"})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080, threaded=True)
