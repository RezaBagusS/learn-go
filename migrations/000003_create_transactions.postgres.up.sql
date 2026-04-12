-- Sequence untuk Reference Number
CREATE SEQUENCE IF NOT EXISTS transaction_ref_seq 
START WITH 1000000001 
INCREMENT BY 1 
NO CYCLE;

-- Tabel Transactions
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_account_id VARCHAR(50) NOT NULL,
    to_account_id VARCHAR(50) NOT NULL,
    amount DECIMAL(16, 2) NOT NULL,
    currency VARCHAR(5) DEFAULT 'IDR',
    -- Perbaikan: Memastikan pemanggilan fungsi dalam default tertata dengan benar
    reference_no VARCHAR(64) UNIQUE DEFAULT (
        'TRX' || to_char(CURRENT_DATE, 'YYYYMMDD') || lpad(nextval('transaction_ref_seq')::text, 12, '0')
    ),
    partner_reference_no VARCHAR(64) NOT NULL,
    external_id VARCHAR(64),
    status VARCHAR(20) DEFAULT 'PENDING',
    note TEXT,
    additional_info JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX idx_transactions_from_account ON transactions(from_account_id);
CREATE INDEX idx_transactions_to_account ON transactions(to_account_id);
CREATE INDEX idx_transactions_partner_ref ON transactions(partner_reference_no);
CREATE INDEX idx_transactions_ref_no ON transactions(reference_no);
CREATE INDEX idx_transactions_status_date ON transactions(status, created_at);

-- Trigger Function
-- PERBAIKAN: Menghapus spasi pada $$
CREATE OR REPLACE FUNCTION update_updated_at_column() 
RETURNS TRIGGER AS $$ 
BEGIN 
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

CREATE TRIGGER update_transactions_updated_at 
BEFORE UPDATE ON transactions 
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();