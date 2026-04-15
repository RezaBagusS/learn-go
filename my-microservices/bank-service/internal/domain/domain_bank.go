package domain

import (
	"time"

	"github.com/google/uuid"
)

type Bank struct {
	ID        uuid.UUID `db:"id" json:"id"` // Primary Key
	BankCode  string    `db:"bank_code" json:"bank_code"`
	BankName  string    `db:"bank_name" json:"bank_name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
