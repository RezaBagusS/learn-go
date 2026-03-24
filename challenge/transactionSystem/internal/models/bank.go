package models

import (
	"time"

	"github.com/google/uuid"
)

// SQL
/*
-- 1. Table banks
CREATE TABLE banks (
	id UUID PRIMARY KEY,
    bank_code VARCHAR(50) NOT NULL,
    bank_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
*/

type Bank struct {
	ID        uuid.UUID `db:"id" json:"id"` // Primary Key
	BankCode  string    `db:"bank_code" json:"bank_code"`
	BankName  string    `db:"bank_name" json:"bank_name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
