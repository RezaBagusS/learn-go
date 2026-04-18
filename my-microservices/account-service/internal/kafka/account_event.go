package kafka

import "time"

// AccountCreatedEvent
type AccountCreatedEvent struct {
	AccountID          string    `json:"account_id"`
	ReferenceNo        string    `json:"reference_no"`
	PartnerReferenceNo string    `json:"partner_reference_no"`
	CustomerID         string    `json:"customer_id"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	PartnerID          string    `json:"partner_id"`
	ExternalID         string    `json:"external_id"`
	BankID             string    `json:"bank_id"`
	AuthCode           string    `json:"auth_code"`
	State              string    `json:"state"`
	CallbackURL        string    `json:"callback_url"`
	CreatedAt          time.Time `json:"created_at"`
}

// AccountCreationFailedEvent
type AccountFailedEvent struct {
	PartnerReferenceNo string    `json:"partner_reference_no"`
	CustomerID         string    `json:"customer_id"`
	MerchantID         string    `json:"merchant_id"`
	PartnerID          string    `json:"partner_id"`
	ExternalID         string    `json:"external_id"`
	ErrorCode          string    `json:"error_code"`
	ErrorMessage       string    `json:"error_message"`
	HttpCode           int       `json:"http_code"`
	CallbackURL        string    `json:"callback_url"`
	FailedAt           time.Time `json:"failed_at"`
}
