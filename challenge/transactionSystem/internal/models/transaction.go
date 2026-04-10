package models

import (
	"encoding/json"
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

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type OriginatorInfo struct {
	OriginatorCustomerNo   string `json:"originatorCustomerNo"`
	OriginatorCustomerName string `json:"originatorCustomerName"`
	OriginatorBankCode     string `json:"originatorBankCode"`
}

type AdditionalInfo struct {
	DeviceId string `json:"deviceId"`
	Channel  string `json:"channel"`
}

type TransactionAdditionalInfo struct {
	DeviceID         string           `json:"deviceId,omitempty"`
	Channel          string           `json:"channel,omitempty"`
	BeneficiaryEmail string           `json:"beneficiaryEmail,omitempty"`
	FeeType          string           `json:"feeType,omitempty"`
	OriginatorInfos  []OriginatorInfo `json:"originatorInfos,omitempty"`
}

type TransferIntrabankRequest struct {
	PartnerReferenceNo   string           `json:"partnerReferenceNo"`
	Amount               Amount           `json:"amount"`
	BeneficiaryAccountNo string           `json:"beneficiaryAccountNo"`
	BeneficiaryEmail     string           `json:"beneficiaryEmail"`
	Currency             string           `json:"currency"`
	CustomerReference    string           `json:"customerReference"`
	FeeType              string           `json:"feeType"`
	Remark               string           `json:"remark"`
	SourceAccountNo      string           `json:"sourceAccountNo"`
	TransactionDate      string           `json:"transactionDate"`
	OriginatorInfos      []OriginatorInfo `json:"originatorInfos"`
	AdditionalInfo       AdditionalInfo   `json:"additionalInfo"`
}

type TransferIntrabankResponse struct {
	ReferenceNo          string           `json:"referenceNo"`
	PartnerReferenceNo   string           `json:"partnerReferenceNo"`
	Amount               Amount           `json:"amount"`
	BeneficiaryAccountNo string           `json:"beneficiaryAccountNo"`
	Currency             string           `json:"currency"`
	CustomerReference    string           `json:"customerReference"`
	SourceAccount        string           `json:"sourceAccount"`
	TransactionDate      string           `json:"transactionDate"`
	OriginatorInfos      []OriginatorInfo `json:"originatorInfos"`
	AdditionalInfo       AdditionalInfo   `json:"additionalInfo"`
}

type Transaction struct {
	ID uuid.UUID `db:"id" json:"id"`
	// Identitas Akun
	FromAccountID string `db:"from_account_id" json:"from_account_id"`
	ToAccountID   string `db:"to_account_id" json:"to_account_id"`

	// Data Finansial
	Amount   float64 `db:"amount" json:"amount"`     // Diubah ke float64
	Currency string  `db:"currency" json:"currency"` // Tambahkan mata uang (IDR)

	// Referensi & Tracking (Crucial for SNAP)
	ReferenceNo  string `db:"reference_no" json:"reference_no"`                 // Dari Bank/Provider
	PartnerRefNo string `db:"partner_reference_no" json:"partner_reference_no"` // Dari Klien
	ExternalID   string `db:"external_id" json:"external_id"`                   // Dari Header SNAP

	// Status & Log
	ResponseCode    string `db:"response_code" json:"response_code"`       // Contoh: 2001700
	ResponseMessage string `db:"response_message" json:"response_message"` // Contoh: Success
	Status          string `db:"status" json:"status"`                     // SUCCESS/FAILED/PENDING

	// Informasi Tambahan
	Note           string          `db:"note" json:"note"`
	AdditionalInfo json.RawMessage `db:"additional_info" json:"additional_info"`

	// Waktu
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
