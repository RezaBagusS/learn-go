package kafka

import "time"

// TransactionCreatedEvent
type TransactionCreatedEvent struct {
	TransactionID   string    `json:"transaction_id"`
	SenderAccount   string    `json:"sender_id"`
	ReceiverAccount string    `json:"receiver_id"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	PartnerID       string    `json:"partner_id"`
	CallbackURL     string    `json:"callback_url"`
	CreatedAt       time.Time `json:"created_at"`
}

// TransactionFailedEvent
type TransactionFailedEvent struct {
	TransactionID   string    `json:"transaction_id"`
	SenderAccount   string    `json:"sender_id"`
	ReceiverAccount string    `json:"receiver_id"`
	Amount          float64   `json:"amount"`
	Reason          string    `json:"reason"`
	ErrorCode       string    `json:"error_code"`
	ErrorMessage    string    `json:"error_message"`
	HttpCode        int       `json:"http_code"`
	PartnerID       string    `json:"partner_id"`
	CallbackURL     string    `json:"callback_url"`
	FailedAt        time.Time `json:"failed_at"`
}

// AccountBalanceUpdatedEvent
type AccountBalanceUpdatedEvent struct {
	AccountNo   string    `json:"account_no"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"` // "in" = masuk, "out" = keluar
	PartnerID   string    `json:"partner_id"`
	CallbackURL string    `json:"callback_url"`
	UpdatedAt   time.Time `json:"updated_at"`
}
