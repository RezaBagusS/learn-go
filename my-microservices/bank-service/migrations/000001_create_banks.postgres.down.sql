DROP TRIGGER IF EXISTS update_banks_updated_at ON banks;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_banks_bank_code;

DROP TABLE IF EXISTS banks;