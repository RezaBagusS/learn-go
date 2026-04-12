CREATE TABLE IF NOT EXISTS accounts (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_code      VARCHAR(50) NOT NULL REFERENCES banks(bank_code),
    account_number VARCHAR(50) NOT NULL, -- CustomerID
    account_holder VARCHAR(150) NOT NULL, -- Name

    -- Partner & Reference
    reference_no        VARCHAR(100) UNIQUE NOT NULL,       
    partner_reference_no VARCHAR(100) UNIQUE NOT NULL,      

    -- Balance (kebutuhan transfer intrabank)
    balance             NUMERIC(19, 4) NOT NULL DEFAULT 0,
    currency            VARCHAR(10) NOT NULL DEFAULT 'IDR',

    -- Customer Info
    email               VARCHAR(255) NOT NULL,
    phone_no            VARCHAR(20) NOT NULL,
    country_code        VARCHAR(10) NOT NULL,
    lang                VARCHAR(10),
    locale              VARCHAR(20),

    -- Merchant Info
    merchant_id         VARCHAR(100),
    sub_merchant_id     VARCHAR(100),
    onboarding_partner  VARCHAR(100),
    terminal_type       VARCHAR(50),

    -- Auth & Security                     
    scopes              TEXT,
    redirect_url        TEXT,

    -- Additional Info (flatten atau simpan sebagai JSONB)
    additional_info     JSONB,

    created_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE accounts
ADD CONSTRAINT unique_bank_account UNIQUE (bank_code, account_number);

-- Index untuk query yang sering dipakai
CREATE INDEX idx_accounts_account_number     ON accounts(account_number);
CREATE INDEX idx_accounts_id                  ON accounts(id);
CREATE INDEX idx_accounts_ref                 ON accounts(reference_no);
CREATE INDEX idx_accounts_partner_ref         ON accounts(partner_reference_no);
CREATE INDEX idx_accounts_email               ON accounts(email);