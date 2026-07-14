-- medilawyer 축소판 (부스터 리뷰 도메인) — 양쪽 스택 동일 시드
CREATE SCHEMA IF NOT EXISTS booster;

CREATE TABLE booster.review (
    id          BIGSERIAL PRIMARY KEY,
    hospital_id BIGINT      NOT NULL,
    channel     VARCHAR(30) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    score       NUMERIC(3,1) NOT NULL DEFAULT 0,
    updated_at  TIMESTAMP   NOT NULL DEFAULT now(),
    created_at  TIMESTAMP   NOT NULL DEFAULT now()
);

CREATE TABLE booster.review_history (
    id        BIGSERIAL PRIMARY KEY,
    review_id BIGINT      NOT NULL,
    action    VARCHAR(20) NOT NULL,
    created_at TIMESTAMP  NOT NULL DEFAULT now()
);

INSERT INTO booster.review (hospital_id, channel, score) VALUES
    (1, 'naver',   4.5),
    (1, 'kakao',   4.0),
    (2, 'naver',   3.5),
    (2, 'babitalk',4.8),
    (3, 'gangnam', 5.0),
    (3, 'naver',   4.2);
