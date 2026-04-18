package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"my-microservices/notification-service/internal/client"
	"my-microservices/notification-service/internal/domain"
	"my-microservices/notification-service/internal/repository"
)

type NotificationService interface {
	ProcessEvent(ctx context.Context, topic string, payload []byte) error
}

type notificationService struct {
	repo       repository.NotificationRepository
	partnerCli client.PartnerClient
	log        *zap.Logger
}

func NewNotificationService(repo repository.NotificationRepository, partnerCli client.PartnerClient, log *zap.Logger) NotificationService {
	return &notificationService{
		repo:       repo,
		partnerCli: partnerCli,
		log:        log,
	}
}

func (s *notificationService) ProcessEvent(ctx context.Context, topic string, payload []byte) error {
	var cbPayload domain.CallbackPayload
	var callbackURL string

	// Mapping berdasarkan Topic
	switch topic {
	case "account.created":
		var evt domain.AccountCreatedEvent
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("failed unmarshal AccountCreatedEvent: %w", err)
		}
		callbackURL = evt.CallbackURL
		cbPayload = domain.CallbackPayload{
			EventType:   "AccountCreated",
			ReferenceID: evt.AccountID,
			PartnerID:   evt.PartnerID,
			Status:      "success",
			Data:        evt,
			Timestamp:   time.Now(),
		}

	case "account.failed":
		var evt domain.AccountFailedEvent
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("failed unmarshal AccountFailedEvent: %w", err)
		}
		callbackURL = evt.CallbackURL
		cbPayload = domain.CallbackPayload{
			EventType:   "AccountFailed",
			ReferenceID: evt.PartnerReferenceNo,
			PartnerID:   evt.PartnerID,
			Status:      "failed",
			Data:        evt,
			Timestamp:   time.Now(),
		}

	case "transaction.created":
		var evt domain.TransactionCreatedEvent
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("failed unmarshal TransactionCreatedEvent: %w", err)
		}
		callbackURL = evt.CallbackURL
		cbPayload = domain.CallbackPayload{
			EventType:   "TransactionCreated",
			ReferenceID: evt.TransactionID,
			PartnerID:   evt.PartnerID,
			Status:      "success",
			Data:        evt,
			Timestamp:   time.Now(),
		}

	case "transaction.failed":
		var evt domain.TransactionFailedEvent
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("failed unmarshal TransactionFailedEvent: %w", err)
		}
		callbackURL = evt.CallbackURL
		cbPayload = domain.CallbackPayload{
			EventType:   "TransactionFailed",
			ReferenceID: evt.TransactionID,
			PartnerID:   evt.PartnerID,
			Status:      "failed",
			Data:        evt,
			Timestamp:   time.Now(),
		}

	case "account.balance_updated":
		var evt domain.AccountBalanceUpdatedEvent
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("failed unmarshal AccountBalanceUpdatedEvent: %w", err)
		}
		callbackURL = evt.CallbackURL
		cbPayload = domain.CallbackPayload{
			EventType:   "AccountBalanceUpdated",
			ReferenceID: evt.AccountNo,
			PartnerID:   evt.PartnerID,
			Status:      "success",
			Data:        evt,
			Timestamp:   time.Now(),
		}

	default:
		s.log.Warn("Ignored unknown topic", zap.String("topic", topic))
		return nil
	}

	// Validasi Callback URL
	if callbackURL == "" {
		s.log.Warn("Event ignored due to empty CallbackURL", zap.String("event_type", cbPayload.EventType))
		return nil
	}

	// Persiapkan payload JSON untuk DB dan Webhook
	payloadBytes, err := json.Marshal(cbPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal callback payload: %w", err)
	}

	logID := uuid.New().String()
	now := time.Now()

	// Simpan Log (Pending) ke Database
	notifLog := domain.NotificationLog{
		ID:             logID,
		EventType:      cbPayload.EventType,
		ReferenceID:    cbPayload.ReferenceID,
		PartnerID:      cbPayload.PartnerID,
		CallbackURL:    callbackURL,
		Payload:        string(payloadBytes),
		Status:         domain.StatusPending,
		RetryCount:     0,
		LastError:      "",
		HttpStatusCode: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
		SentAt:         nil,
	}

	if err := s.repo.CreateLog(ctx, notifLog); err != nil {
		s.log.Error("Failed to save pending log", zap.Error(err))
		return err // Return error agar offset Kafka tidak di-commit
	}

	// Eksekusi HTTP Client ke Partner
	s.log.Info("Sending webhook", zap.String("url", callbackURL), zap.String("event", cbPayload.EventType))
	httpCode, partnerErr := s.partnerCli.SendWebhook(ctx, callbackURL, payloadBytes)

	// Analisis Hasil & Update Database
	finalStatus := domain.StatusSuccess
	lastErrorMsg := ""
	var sentAt *time.Time

	if partnerErr != nil {
		finalStatus = domain.StatusFailed
		lastErrorMsg = partnerErr.Error()
	} else {
		successTime := time.Now()
		sentAt = &successTime
	}

	errUpdate := s.repo.UpdateLogStatus(
		ctx,
		logID,
		finalStatus,
		0, // Retry awal 0
		lastErrorMsg,
		httpCode,
		sentAt,
	)

	if errUpdate != nil {
		s.log.Error("Failed to update log status after sending webhook", zap.Error(errUpdate))
		return nil
	}

	return nil
}
