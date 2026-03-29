package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// SQL
/*
-- 1. Table Accounts
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    bank_code VARCHAR(50) NOT NULL REFERENCES banks(bank_code),
    account_number VARCHAR(50) NOT NULL,
    account_holder VARCHAR(150) NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE accounts
ADD CONSTRAINT unique_bank_account UNIQUE (bank_code, account_number);

ALTER TABLE accounts ALTER COLUMN id SET DEFAULT gen_random_uuid();
*/

var (
	ErrDatabaseIssue  = errors.New("gagal mengambil data dari db untuk account")
	ErrDatabaseFailed = errors.New("gagal melakukan aksi pada database")
	ErrDeleteFailed   = errors.New("gagal menghapus data")

	ErrInvalidUuid        = errors.New("Format tidak valid")
	ErrInvalidInitBalance = errors.New("Balance tidak boleh minus")
	ErrInvalidJsonFormat  = errors.New("Format JSON tidak valid")
	ErrInvalidTrxType     = errors.New("Tipe transaksi tidak sesuai (all/in/out)")
	ErrInvalidBankCode    = errors.New("Kode bank tidak terdaftar pada sistem")
	ErrInvalidField       = errors.New("Terdapat field yang kosong")

	ErrIdNotFound = errors.New("data tidak ditemukan")

	ErrDuplicateAccount = errors.New("Nomor rekening sudah terdaftar")
)

type Account struct {
	ID            uuid.UUID `db:"id" json:"id"`               // Primary Key
	BankCode      string    `db:"bank_code" json:"bank_code"` // Foreign Key from bankTable
	AccountNumber string    `db:"account_number" json:"account_number"`
	AccountHolder string    `db:"account_holder" json:"account_holder"`
	Balance       int64     `db:"balance" json:"balance"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}
