CREATE TABLE IF NOT EXISTS transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_account_id VARCHAR(50),
    from_bank_code  VARCHAR(50),
    to_account_id   VARCHAR(50),
    to_bank_code    VARCHAR(50),
    amount          BIGINT NOT NULL,
    note            TEXT,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transactions_from_account ON transactions(from_account_id);
CREATE INDEX idx_transactions_to_account ON transactions(to_account_id);