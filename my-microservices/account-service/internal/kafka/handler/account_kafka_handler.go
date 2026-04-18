package kafkahandler

import (
	"context"

	"my-microservices/account-service/internal/kafka"

	"go.uber.org/zap"
)

type AccountKafkaHandler struct {
	logger *zap.Logger
}

func NewAccountKafkaHandler(logger *zap.Logger) *AccountKafkaHandler {
	return &AccountKafkaHandler{logger: logger}
}

func (h *AccountKafkaHandler) HandleAccountCreated(ctx context.Context, event kafka.AccountCreatedEvent) error {

	h.logger.Info("account.created event consumed",
		zap.String("account_id", event.AccountID),
		zap.String("reference_no", event.ReferenceNo),
		zap.String("partner_reference_no", event.PartnerReferenceNo),
		zap.String("customer_id", event.CustomerID),
		zap.String("name", event.Name),
		zap.String("email", event.Email),
		zap.String("partner_id", event.PartnerID),
		zap.String("external_id", event.ExternalID),
		zap.String("bank_id", event.BankID),
		zap.String("auth_code", event.AuthCode),
		zap.String("state", event.State),
		zap.Time("created_at", event.CreatedAt),
	)

	return nil
}

func (h *AccountKafkaHandler) HandleAccountFailed(ctx context.Context, event kafka.AccountFailedEvent) error {

	h.logger.Info("account.failed event consumed",
		zap.String("partner_reference_no", event.PartnerReferenceNo),
		zap.String("customer_id", event.CustomerID),
		zap.String("merchant_id", event.MerchantID),
		zap.String("partner_id", event.PartnerID),
		zap.String("external_id", event.ExternalID),
		zap.String("error_code", event.ErrorCode),
		zap.String("error_message", event.ErrorMessage),
		zap.Int("http_code", event.HttpCode),
		zap.Time("failedAt", event.FailedAt),
	)

	return nil
}
