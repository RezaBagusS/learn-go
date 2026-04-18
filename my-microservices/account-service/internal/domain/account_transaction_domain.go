package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type TransferIntrabankRequest struct {
	PartnerReferenceNo   string           `json:"partner_reference_no"`
	Amount               Amount           `json:"amount"`
	BeneficiaryAccountNo string           `json:"beneficiary_account_no"`
	BeneficiaryEmail     string           `json:"beneficiary_email"`
	Currency             string           `json:"currency"`
	CustomerReference    string           `json:"customer_reference"`
	FeeType              string           `json:"fee_type"`
	Remark               string           `json:"remark"`
	ExternalID           string           `json:"external_id"`
	SourceAccountNo      string           `json:"source_account_no"`
	TransactionDate      string           `json:"transaction_date"`
	OriginatorInfos      []OriginatorInfo `json:"originator_infos"`
	AdditionalInfo       AdditionalInfo   `json:"additional_info"`
}

type TransferIntrabankResponse struct {
	ReferenceNo          string           `json:"reference_no"`
	PartnerReferenceNo   string           `json:"partner_reference_no"`
	Amount               Amount           `json:"amount"`
	BeneficiaryAccountNo string           `json:"beneficiary_account_no"`
	Currency             string           `json:"currency"`
	CustomerReference    string           `json:"customer_reference"`
	SourceAccount        string           `json:"source_account"`
	TransactionDate      string           `json:"transaction_date"`
	OriginatorInfos      []OriginatorInfo `json:"originator_infos"`
	AdditionalInfo       AdditionalInfo   `json:"additional_info"`
}

type Transaction struct {
	ID            uuid.UUID `db:"id" json:"id"`
	FromAccountNo string    `db:"from_account_no" json:"from_account_no"`
	ToAccountNo   string    `db:"to_account_no"   json:"to_account_no"`

	Amount   float64 `db:"amount" json:"amount"`
	Currency string  `db:"currency" json:"currency"`

	ReferenceNo  string `db:"reference_no" json:"reference_no"`
	PartnerRefNo string `db:"partner_reference_no" json:"partner_reference_no"`
	ExternalID   string `db:"external_id" json:"external_id"`

	// Status & Log
	Status string `db:"status" json:"status"` // SUCCESS/FAILED/PENDING

	// Informasi Tambahan
	Note           string          `db:"note" json:"note"`
	AdditionalInfo json.RawMessage `db:"additional_info" json:"additional_info"`

	// Waktu
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
