package models

import (
	"time"
)

// SQL
/*
-- 1. Table banks
CREATE TABLE banks (
    bank_code VARCHAR(50) PRIMARY KEY,
    bank_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
*/

type Bank struct {
	BankCode  string    `db:"bank_code" json:"bank_code"` // Primary Key
	BankName  string    `db:"bank_name" json:"bank_name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
