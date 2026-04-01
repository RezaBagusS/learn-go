CREATE TABLE IF NOT EXISTS accounts (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_code      VARCHAR(50) NOT NULL REFERENCES banks(bank_code),
    account_number VARCHAR(50) NOT NULL,
    account_holder VARCHAR(150) NOT NULL,
    balance        BIGINT NOT NULL DEFAULT 0,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE accounts
ADD CONSTRAINT unique_bank_account UNIQUE (bank_code, account_number);