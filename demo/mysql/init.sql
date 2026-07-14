-- Identical seed for BOTH stacks (same auto_increment counters => comparable ids)
CREATE DATABASE IF NOT EXISTS demo;
USE demo;

CREATE TABLE wallet (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id    BIGINT         NOT NULL,
    balance    DECIMAL(15, 0) NOT NULL DEFAULT 0,
    updated_at DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE wallet_history (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    wallet_id     BIGINT         NOT NULL,
    type          VARCHAR(20)    NOT NULL,
    amount        DECIMAL(15, 0) NOT NULL,
    balance_after DECIMAL(15, 0) NOT NULL,
    created_at    DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO wallet (user_id, balance) VALUES (101, 50000), (102, 30000);
