
DROP INDEX IF EXISTS idx_accounts_email;
DROP INDEX IF EXISTS idx_accounts_partner_ref;
DROP INDEX IF EXISTS idx_accounts_ref;
DROP INDEX IF EXISTS idx_accounts_id;
DROP INDEX IF EXISTS idx_accounts_customer_id;

ALTER TABLE IF EXISTS accounts DROP CONSTRAINT IF EXISTS unique_bank_account;

DROP TABLE IF EXISTS accounts;