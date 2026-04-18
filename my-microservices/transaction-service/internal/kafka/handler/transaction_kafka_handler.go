package kafkahandler

import (
	"context"
	"my-microservices/transaction-service/internal/kafka"

	"go.uber.org/zap"
)

type TransactionKafkaHandler struct {
	logger *zap.Logger
}

func NewTransactionKafkaHandler(logger *zap.Logger) *TransactionKafkaHandler {
	return &TransactionKafkaHandler{logger: logger}
}

func (h *TransactionKafkaHandler) HandleTransactionCreated(ctx context.Context, event kafka.TransactionCreatedEvent) error {

	h.logger.Info("transaction.created event consumed",
		zap.String("transaction_id", event.TransactionID),
		zap.String("sender_account", event.SenderAccount),
		zap.String("receiver_account", event.ReceiverAccount),
		zap.Float64("amount", event.Amount),
		zap.String("currency", event.Currency),
		zap.Time("created_at", event.CreatedAt),
	)

	return nil
}

func (h *TransactionKafkaHandler) HandleTransactionFailed(ctx context.Context, event kafka.TransactionFailedEvent) error {

	h.logger.Warn("transaction.failed event consumed",
		zap.String("transaction_id", event.TransactionID),
		zap.String("sender_account", event.SenderAccount),
		zap.String("receiver_account", event.ReceiverAccount),
		zap.Float64("amount", event.Amount),
		zap.String("error_code", event.ErrorCode),
		zap.String("error_message", event.ErrorMessage),
		zap.Int("http_code", event.HttpCode),
		zap.Time("failed_at", event.FailedAt),
	)

	return nil
}

func (h *TransactionKafkaHandler) HandleAccountBalanceUpdated(ctx context.Context, event kafka.AccountBalanceUpdatedEvent) error {

	h.logger.Info("account.balance.updated event consumed",
		zap.String("account_no", event.AccountNo),
		zap.Float64("amount", event.Amount),
		zap.String("type", event.Type),
		zap.Time("updated_at", event.UpdatedAt),
	)

	return nil
}
