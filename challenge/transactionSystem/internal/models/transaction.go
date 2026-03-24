package models

import (
	"time"

	"github.com/google/uuid"
)

// SQL
/*
-- 3. Table transactions
CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    from_account_id VARCHAR(50),
    from_bank_code VARCHAR(50),
    to_account_id VARCHAR(50),
    to_bank_code VARCHAR(50),
    amount BIGINT NOT NULL,
    note BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index
CREATE INDEX idx_transactions_from_account ON transactions(from_account_id);
CREATE INDEX idx_transactions_to_account ON transactions(to_account_id);
*/

type Transaction struct {
	ID            uuid.UUID `db:"id" json:"id"` // Primary Key
	FromAccountID string    `db:"from_account_id" json:"from_account_id"`
	FromBankCode  string    `db:"from_bank_code" json:"from_bank_code"`
	ToAccountID   string    `db:"to_account_id" json:"to_account_id"`
	ToBankCode    string    `db:"to_bank_code" json:"to_bank_code"`
	Amount        int64     `db:"amount" json:"amount"`
	Note          string    `db:"note" json:"note"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}
