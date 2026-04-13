package kafka

import "time"

// TransactionCreatedEvent
type TransactionCreatedEvent struct {
	TransactionID   string    `json:"transaction_id"`
	SenderAccount   string    `json:"sender_id"`
	ReceiverAccount string    `json:"receiver_id"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	CreatedAt       time.Time `json:"created_at"`
}

// TransactionFailedEvent
type TransactionFailedEvent struct {
	TransactionID   string    `json:"transaction_id"`
	SenderAccount   string    `json:"sender_id"`
	ReceiverAccount string    `json:"receiver_id"`
	Amount          float64   `json:"amount"`
	Reason          string    `json:"reason"`
	FailedAt        time.Time `json:"failed_at"`
}

// AccountBalanceUpdatedEvent
type AccountBalanceUpdatedEvent struct {
	AccountNo string    `json:"account_no"`
	Amount    float64   `json:"amount"`
	Type      string    `json:"type"` // "in" = masuk, "out" = keluar
	UpdatedAt time.Time `json:"updated_at"`
}

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
	BankCode           string    `json:"bank_code"`
	AuthCode           string    `json:"auth_code"`
	State              string    `json:"state"`
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
	Timestamp          time.Time `json:"timestamp"`
}
