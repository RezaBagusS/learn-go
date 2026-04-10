DROP TRIGGER IF EXISTS update_transactions_updated_at ON transactions;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_transactions_status_date;
DROP INDEX IF EXISTS idx_transactions_ref_no;
DROP INDEX IF EXISTS idx_transactions_partner_ref;
DROP INDEX IF EXISTS idx_transactions_to_account;
DROP INDEX IF EXISTS idx_transactions_from_account;

DROP TABLE IF EXISTS transactions;