CREATE TABLE IF NOT EXISTS banks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_code  VARCHAR(50) NOT NULL UNIQUE,
    bank_name  VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_banks_bank_code ON banks(bank_code);

-- Trigger Function
CREATE OR REPLACE FUNCTION update_updated_at_column() 
RETURNS TRIGGER AS $$ 
BEGIN 
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

CREATE TRIGGER update_banks_updated_at 
BEFORE UPDATE ON banks 
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();