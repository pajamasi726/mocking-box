"""NEW stack — JPA 스타일 (SELECT FOR UPDATE + 절대 UPDATE).

심어둔 것:
  ①(버그) publish 시 review_history INSERT 누락 — 응답 동일 → write-set만 검출
  ②(매퍼 차이) 목록 ORDER BY id DESC — sort_arrays로 흡수
"""
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


def review_json(rid, hid, ch, st, sc, ua):
    return {"review_id": rid, "hospital_id": hid, "channel": ch,
            "status": st, "score": float(sc), "updated_at": ua.isoformat()}


@app.get("/review/<int:rid>")
def get_review(rid):
    with db() as c, c.cursor() as cur:
        cur.execute("SELECT hospital_id, channel, status, score, updated_at, id FROM booster.review WHERE id=%s", (rid,))
        r = cur.fetchone()
    if not r:
        return jsonify({"error_code": "R001", "message": "not found"}), 404
    return jsonify(review_json(r[5], r[0], r[1], r[2], r[3], r[4]))


@app.get("/reviews")
def list_reviews():
    hid = int(request.args.get("hospital_id", 0))
    with db() as c, c.cursor() as cur:
        # ② 매퍼 차이: DESC
        cur.execute("SELECT hospital_id, channel, status, score, updated_at, id FROM booster.review WHERE hospital_id=%s ORDER BY id DESC", (hid,))
        rows = cur.fetchall()
    revs = [review_json(r[5], r[0], r[1], r[2], r[3], r[4]) for r in rows]
    return jsonify({"reviews": revs, "total": len(revs)})


@app.post("/review")
def create_review():
    b = request.get_json(force=True)
    with db() as c, c.cursor() as cur:
        cur.execute("INSERT INTO booster.review (score, channel, hospital_id) VALUES (%s,%s,%s) "
                    "RETURNING id, hospital_id, channel, status, score, updated_at",
                    (float(b.get("score", 0)), b["channel"], int(b["hospital_id"])))
        r = cur.fetchone()
        cur.execute("INSERT INTO booster.review_history (action, review_id) VALUES ('CREATE', %s)", (r[0],))
        c.commit()
    return jsonify(review_json(r[0], r[1], r[2], r[3], r[4], r[5])), 201


@app.post("/review/<int:rid>/publish")
def publish(rid):
    with db() as c, c.cursor() as cur:
        cur.execute("SELECT status FROM booster.review WHERE id=%s FOR UPDATE", (rid,))
        row = cur.fetchone()
        if not row:
            c.rollback()
            return jsonify({"error_code": "R001", "message": "not found"}), 404
        if row[0] != "DRAFT":
            c.rollback()
            return jsonify({"error_code": "R002", "message": "invalid transition"}), 409
        cur.execute("UPDATE booster.review SET status='PUBLISHED' WHERE id=%s", (rid,))
        # ① 심어둔 버그: review_history INSERT 누락
        c.commit()
    return jsonify({"review_id": rid, "status": "PUBLISHED"})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080, threaded=True)
