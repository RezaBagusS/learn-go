package kafkahandler

import (
	"context"

	"belajar-go/challenge/transactionSystem/internal/kafka"

	"go.uber.org/zap"
)

type TransactionKafkaHandler struct {
	logger *zap.Logger
}

func NewTransactionKafkaHandler(logger *zap.Logger) *TransactionKafkaHandler {
	return &TransactionKafkaHandler{logger: logger}
}

// HandleCreated — logic setelah transaction.created diterima
// Cache invalidation sudah dilakukan di HTTP handler,
// jadi di sini hanya untuk keperluan audit/monitoring
func (h *TransactionKafkaHandler) HandleCreated(ctx context.Context, event kafka.TransactionCreatedEvent) error {
	h.logger.Info("transaction.created event consumed",
		zap.String("transaction_id", event.TransactionID),
		zap.String("sender", event.SenderAccount),
		zap.String("receiver", event.ReceiverAccount),
		zap.Float64("amount", event.Amount),
		zap.String("currency", event.Currency),
		zap.Time("created_at", event.CreatedAt),
	)

	// Contoh yang bisa ditambahkan ke depannya:
	// - Kirim email/push notification ke sender & receiver
	// - Catat ke audit log service terpisah
	// - Update reporting/dashboard service

	return nil
}

// HandleFailed — logic setelah transaction.failed diterima
func (h *TransactionKafkaHandler) HandleFailed(ctx context.Context, event kafka.TransactionFailedEvent) error {
	h.logger.Warn("transaction.failed event consumed",
		zap.String("transaction_id", event.TransactionID),
		zap.String("sender", event.SenderAccount),
		zap.String("receiver", event.ReceiverAccount),
		zap.Float64("amount", event.Amount),
		zap.String("reason", event.Reason),
		zap.Time("failed_at", event.FailedAt),
	)

	// Contoh yang bisa ditambahkan ke depannya:
	// - Kirim notifikasi gagal ke sender
	// - Trigger alert ke tim ops jika failure rate tinggi
	// - Catat ke dead letter queue untuk investigasi

	return nil
}

// HandleBalanceUpdated — logic setelah account.balance.updated diterima
func (h *TransactionKafkaHandler) HandleBalanceUpdated(ctx context.Context, event kafka.AccountBalanceUpdatedEvent) error {
	h.logger.Info("account.balance.updated event consumed",
		zap.String("account_no", event.AccountNo),
		zap.Float64("amount", event.Amount),
		zap.String("type", event.Type),
		zap.Time("updated_at", event.UpdatedAt),
	)

	// Contoh yang bisa ditambahkan ke depannya:
	// - Kirim notifikasi perubahan saldo
	// - Sync ke read model / materialized view
	// - Trigger alert jika saldo di bawah threshold

	return nil
}
