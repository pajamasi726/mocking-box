-- lawkit-contract 도메인 축소판 (양쪽 스택 동일 시드)
SET NAMES utf8mb4;
CREATE DATABASE IF NOT EXISTS lawkit CHARACTER SET utf8mb4;
USE lawkit;

CREATE TABLE contract (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id    BIGINT         NOT NULL,
    title      VARCHAR(200)   NOT NULL,
    status     VARCHAR(20)    NOT NULL DEFAULT 'DRAFT', -- DRAFT/SIGNED/CANCELLED
    amount     DECIMAL(15, 0) NOT NULL DEFAULT 0,
    updated_at DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE contract_history (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    contract_id BIGINT      NOT NULL,
    action      VARCHAR(20) NOT NULL,
    created_at  DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO contract (user_id, title, status, amount) VALUES
    (1, '표준 근로계약서 검토', 'DRAFT', 300000),
    (1, '용역계약서 자문',      'DRAFT', 500000),
    (2, 'NDA 검토',            'DRAFT', 200000),
    (2, '임대차계약 자문',      'DRAFT', 400000),
    (3, '주주간계약 검토',      'DRAFT', 900000),
    (3, 'M&A 실사 자문',       'DRAFT', 1500000),
    (4, '개인정보처리방침 검토', 'DRAFT', 250000),
    (4, '이용약관 개정 자문',    'DRAFT', 350000);
