package domain

import "time"

// ─── Kafka Event Structs ───────────────────────────────────────────────────

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

type AccountBalanceUpdatedEvent struct {
	AccountNo   string    `json:"account_no"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"` // "in" = masuk, "out" = keluar
	PartnerID   string    `json:"partner_id"`
	CallbackURL string    `json:"callback_url"`
	UpdatedAt   time.Time `json:"updated_at"`
}

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

// ─── Callback Payload (dikirim ke partner) ────────────────────────────────

type CallbackPayload struct {
	EventType   string      `json:"event_type"`
	ReferenceID string      `json:"reference_id"` // transaction_id / account_id
	PartnerID   string      `json:"partner_id"`
	Status      string      `json:"status"` // "success" | "failed"
	Data        interface{} `json:"data"`
	Timestamp   time.Time   `json:"timestamp"`
}

// ─── Notification Log (DB model) ─────────────────────────────────────────

type NotificationStatus string

const (
	StatusPending NotificationStatus = "pending"
	StatusSuccess NotificationStatus = "success"
	StatusFailed  NotificationStatus = "failed"
)

type NotificationLog struct {
	ID             string             `db:"id"`
	EventType      string             `db:"event_type"`
	ReferenceID    string             `db:"reference_id"`
	PartnerID      string             `db:"partner_id"`
	CallbackURL    string             `db:"callback_url"`
	Payload        string             `db:"payload"` // JSON string
	Status         NotificationStatus `db:"status"`
	RetryCount     int                `db:"retry_count"`
	LastError      string             `db:"last_error"`
	HttpStatusCode int                `db:"http_status_code"`
	CreatedAt      time.Time          `db:"created_at"`
	UpdatedAt      time.Time          `db:"updated_at"`
	SentAt         *time.Time         `db:"sent_at"`
}
