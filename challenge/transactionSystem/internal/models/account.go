package models

import (
	"encoding/json"
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

type AccountCreateRequest struct {
	PartnerReferenceNo string         `json:"partner_reference_no"`
	BankCode           string         `json:"bank_code"`
	CountryCode        string         `json:"country_code"`
	CustomerID         string         `json:"customer_id"`
	DeviceInfo         DeviceInfo     `json:"device_info"`
	Email              string         `json:"email"`
	Lang               string         `json:"lang"`
	Locale             string         `json:"locale"`
	Name               string         `json:"name"`
	OnboardingPartner  string         `json:"onboarding_partner"`
	PhoneNo            string         `json:"phone_no"`
	RedirectURL        string         `json:"redirect_url"`
	Scopes             string         `json:"scopes"`
	SeamlessData       string         `json:"seamless_data"`
	SeamlessSign       string         `json:"seamless_sign"`
	State              string         `json:"state"`
	MerchantID         string         `json:"merchant_id"`
	SubMerchantID      string         `json:"sub_merchant_id"`
	TerminalType       string         `json:"terminal_type"`
	AdditionalInfo     AdditionalInfo `json:"additional_info"`
}

type AccountCreateResponse struct {
	ReferenceNo        string         `json:"reference_no"`
	PartnerReferenceNo string         `json:"partner_reference_no"`
	AuthCode           string         `json:"auth_code"`
	APIKey             string         `json:"api_key"`
	AccountID          string         `json:"account_id"`
	State              string         `json:"state"`
	AdditionalInfo     AdditionalInfo `json:"additional_info"`
}

type Account struct {
	ID            uuid.UUID `db:"id"             json:"id"`
	BankCode      string    `db:"bank_code"       json:"bank_code"`
	AccountNumber string    `db:"account_number"  json:"account_number"`
	AccountHolder string    `db:"account_holder"  json:"account_holder"`

	// Referensi & Tracking
	ReferenceNo        string `db:"reference_no"        json:"reference_no"`
	PartnerReferenceNo string `db:"partner_reference_no" json:"partner_reference_no"`

	// Balance
	Balance  int64  `db:"balance"  json:"balance"`
	Currency string `db:"currency" json:"currency"`

	// Customer Info
	Email       string `db:"email"        json:"email"`
	PhoneNo     string `db:"phone_no"     json:"phone_no"`
	CountryCode string `db:"country_code" json:"country_code"`
	Lang        string `db:"lang"         json:"lang"`
	Locale      string `db:"locale"       json:"locale"`

	// Merchant Info
	MerchantID        string `db:"merchant_id"        json:"merchant_id"`
	SubMerchantID     string `db:"sub_merchant_id"    json:"sub_merchant_id"`
	OnboardingPartner string `db:"onboarding_partner" json:"onboarding_partner"`
	TerminalType      string `db:"terminal_type"      json:"terminal_type"`

	// Auth & Security
	Scopes      string `db:"scopes"       json:"scopes"`
	RedirectURL string `db:"redirect_url" json:"redirect_url"`

	// Additional Info
	AdditionalInfo json.RawMessage `db:"additional_info" json:"additional_info"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
