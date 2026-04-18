package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"my-microservices/transaction-service/config"
	"my-microservices/transaction-service/helper"
	"my-microservices/transaction-service/internal/domain"
	"my-microservices/transaction-service/internal/kafka"
	"my-microservices/transaction-service/internal/middleware"
	"my-microservices/transaction-service/internal/repository"
	"my-microservices/transaction-service/observability/metrics"
	"os"
	"strconv"
	"time"

	pbAccount "my-microservices/shared/pb/account"
	pbFraud "my-microservices/shared/pb/fraud"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type TransactionService interface {
	FetchAllTransactions(ctx context.Context) ([]domain.Transaction, *domain.SnapDetail)
	FetchTransactionById(ctx context.Context, id string) (*domain.Transaction, *domain.SnapDetail)
	FetchSummaryToday(ctx context.Context, date time.Time) ([]domain.Transaction, *domain.SnapDetail)
	FetchTransactionsByAccountNo(ctx context.Context, accountNo string) ([]domain.Transaction, *domain.SnapDetail)
	TransferIntrabank(ctx context.Context, accountID string, producer *kafka.Producer, payload domain.TransferIntrabankRequest, svcCode string) (string, *domain.SnapDetail)
	Topup(ctx context.Context, accountID string, producer *kafka.Producer, payload domain.TopupRequest, svcCode string) (string, *domain.SnapDetail)
}

type transactionService struct {
	repo       repository.TransactionRepository
	logger     *zap.Logger
	accountCli pbAccount.AccountGRPCServiceClient
	fraudCli   pbFraud.FraudServiceClient
	rdb        *redis.Client
	keyManager *helper.RedisKeyManager
}

func NewTransactionsService(repo repository.TransactionRepository, accountCli pbAccount.AccountGRPCServiceClient, fraudCli pbFraud.FraudServiceClient, rdb *redis.Client, keyManager *helper.RedisKeyManager) TransactionService {
	return &transactionService{
		repo:       repo,
		logger:     helper.Log,
		accountCli: accountCli,
		fraudCli:   fraudCli,
		rdb:        rdb,
		keyManager: keyManager,
	}
}

const svcTransaction = "transaction"

func (s *transactionService) FetchAllTransactions(ctx context.Context) ([]domain.Transaction, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.GetAll")
	defer span.End()
	operation := "fetch_all"

	svcStart := time.Now()
	transactions, snapErr := s.repo.GetAllTransactions(ctx)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		errPrefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(errPrefix)
		s.logger.Error("failed fetching transactions", zap.Error(errPrefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, snapErr
	}

	span.SetAttributes(attribute.Int("service.result.count", len(transactions)))
	metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "success").Inc()

	return transactions, nil
}

func (s *transactionService) FetchSummaryToday(ctx context.Context, date time.Time) ([]domain.Transaction, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.GetSummary")
	defer span.End()
	operation := "fetch_summary"

	if date.After(time.Now()) {
		err := domain.ErrInvalidFutureDate
		span.RecordError(err)
		s.logger.Error("invalid future date", zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, &domain.SnapInvalidFormat
	}

	svcStart := time.Now()
	transactions, snapErr := s.repo.GetSummary(ctx, date)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		errPrefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(errPrefix)
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, snapErr
	}

	metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "success").Inc()

	return transactions, nil
}

func (s *transactionService) FetchTransactionById(ctx context.Context, id string) (*domain.Transaction, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.GetById")
	defer span.End()
	operation := "fetch_by_id"

	svcStart := time.Now()
	transaction, snapErr := s.repo.GetTransactionById(ctx, id)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		prefixError := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefixError)
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, snapErr
	}

	span.SetAttributes(attribute.String("service.query.id", id))
	metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "success").Inc()

	return transaction, nil
}

func (s *transactionService) FetchTransactionsByAccountNo(ctx context.Context, accountNo string) ([]domain.Transaction, *domain.SnapDetail) {
	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.GetByAccountNo")
	defer span.End()
	operation := "fetch_by_account_no"

	svcStart := time.Now()
	transactions, snapErr := s.repo.GetTransactionsByAccountNo(ctx, accountNo)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		prefixError := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefixError)
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, snapErr
	}

	span.SetAttributes(attribute.String("service.query.account_no", accountNo))
	metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "success").Inc()

	return transactions, nil
}

func (s *transactionService) TransferIntrabank(ctx context.Context, accountID string, producer *kafka.Producer, payload domain.TransferIntrabankRequest, svcCode string) (string, *domain.SnapDetail) {
	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.TransferIntrabank")
	defer span.End()
	operation := "transfer_intrabank"

	// Validasi & konversi amount
	amountValue, err := strconv.ParseFloat(payload.Amount.Value, 64)
	if err != nil {
		span.RecordError(err)
		return "", &domain.SnapInvalidFormat
	}

	if amountValue <= 0 {
		span.RecordError(domain.ErrInvalidTranserAmount)
		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
		return "", &domain.SnapInsufficient
	}

	if payload.SourceAccountNo == payload.BeneficiaryAccountNo {
		span.RecordError(domain.ErrLogicSelfTranser)
		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
		return "", &domain.SnapBadRequest
	}

	// ─── PANGGIL GRPC FRAUD SERVICE UNTUK VALIDASI TRX ─────────────────────
	fraudStart := time.Now()
	fraudResp, err := s.fraudCli.ValidateTransaction(ctx, &pbFraud.FraudValidationRequest{
		TransactionId: payload.PartnerReferenceNo,
		SenderId:      payload.SourceAccountNo,
		ReceiverId:    payload.BeneficiaryAccountNo,
		Amount:        amountValue,
		Currency:      payload.Amount.Currency,
		DeviceId:      payload.AdditionalInfo.DeviceId,
		Channel:       payload.AdditionalInfo.Channel,
	})
	metrics.ServiceDuration.WithLabelValues("grpc_fraud", "validate_transaction").Observe(time.Since(fraudStart).Seconds())

	if err != nil {
		s.logger.Warn("fraud-service is down, entering FAIL-OPEN mode with safety limits", zap.Error(err))

		// ─── Limit 5jt/hari ───
		const maxFallbackDaily = 5000000.0
		fallbackKey := s.keyManager.Generate(config.REDIS_KEY_FRAUD_FALLBACK, payload.SourceAccountNo, time.Now().Format("20060102"))

		// total trx/day di redis
		currentTotalStr, _ := s.rdb.Get(ctx, fallbackKey).Result()
		currentTotal, _ := strconv.ParseFloat(currentTotalStr, 64)

		// Validasi total trx/day
		if currentTotal+amountValue > maxFallbackDaily {
			s.logger.Warn("fallback limit exceeded",
				zap.String("account", payload.SourceAccountNo),
				zap.Float64("total_today", currentTotal),
				zap.Float64("current_request", amountValue))
			return "", &domain.SnapExceedLimit
		}

		// Update Redis (exp 24 jam)
		newTotal := currentTotal + amountValue
		s.rdb.Set(ctx, fallbackKey, fmt.Sprintf("%.2f", newTotal), 24*time.Hour)

		s.logger.Info("transaction allowed by FAIL-OPEN policy",
			zap.String("account", payload.SourceAccountNo),
			zap.Float64("amount", amountValue))
	} else {
		// Log hasil validasi FDS (Berhasil terkoneksi)
		s.logger.Info("FDS validation response received",
			zap.String("trx_id", payload.PartnerReferenceNo),
			zap.String("action", fraudResp.Action),
			zap.String("reason", fraudResp.Reason),
			zap.Bool("is_fraud", fraudResp.IsFraud))

		if fraudResp.IsFraud {
			s.logger.Warn("FRAUD DETECTED!",
				zap.String("trx_id", payload.PartnerReferenceNo),
				zap.String("reason", fraudResp.Reason),
				zap.String("action", fraudResp.Action))

			if fraudResp.Action == "BLOCK" {
				return "", &domain.SnapSuspiciousTransaction
			}
		}
	}

	// ─── PANGGIL GRPC UNTUK VALIDASI & MUTASI SALDO ─────────────────────
	mutationStart := time.Now()
	mutationResp, grpcErr := s.accountCli.ExecuteTransferMutation(ctx, &pbAccount.TransferMutationRequest{
		SourceAccount:      payload.SourceAccountNo,
		BeneficiaryAccount: payload.BeneficiaryAccountNo,
		Amount:             int64(amountValue),
	})

	metrics.ServiceDuration.WithLabelValues("grpc_account", "execute_mutation").Observe(time.Since(mutationStart).Seconds())

	if grpcErr != nil {
		span.RecordError(grpcErr)
		s.logger.Error("gRPC call to account-service failed", zap.Error(grpcErr))

		// publish TransactionFailedEvent
		failedEvent := kafka.TransactionFailedEvent{
			TransactionID:   payload.PartnerReferenceNo,
			SenderAccount:   payload.SourceAccountNo,
			ReceiverAccount: payload.BeneficiaryAccountNo,
			Amount:          amountValue,
			Reason:          grpcErr.Error(),
			ErrorCode:       domain.SnapInternalError.GetResponseCode(svcCode),
			HttpCode:        domain.SnapInternalError.HttpCode,
			ErrorMessage:    domain.SnapInternalError.ResponseMessage,
			PartnerID:       "TRX-XXXXXX-XXXX",
			CallbackURL:     os.Getenv("CALLBACK_URL"),
			FailedAt:        time.Now(),
		}

		if pubErr := producer.Publish(ctx, kafka.TopicTransactionFailed, payload.PartnerReferenceNo, failedEvent); pubErr != nil {
			s.logger.Error("failed to publish transaction.failed", zap.Error(pubErr))
		}

		return "", &domain.SnapInternalError
	}

	if !mutationResp.Success {
		var snapErr *domain.SnapDetail
		switch mutationResp.ErrorCode {
		case "ERR_INSUFFICIENT_FUNDS":
			snapErr = &domain.SnapInsufficient
		case "ERR_INVALID_ACCOUNT":
			snapErr = &domain.SnapInvalidAccount
		default:
			snapErr = &domain.SnapInternalError
		}

		errBiz := errors.New(mutationResp.ErrorMessage)
		span.RecordError(errBiz)
		s.logger.Warn("transfer rejected by account-service", zap.String("reason", mutationResp.ErrorMessage))

		// publish TransactionFailedEvent
		failedEvent := kafka.TransactionFailedEvent{
			TransactionID:   payload.PartnerReferenceNo,
			SenderAccount:   payload.SourceAccountNo,
			ReceiverAccount: payload.BeneficiaryAccountNo,
			Amount:          amountValue,
			Reason:          mutationResp.ErrorMessage, // grpcErr == nil di sini, gunakan pesan dari response
			ErrorCode:       snapErr.GetResponseCode(svcCode),
			HttpCode:        snapErr.HttpCode,
			ErrorMessage:    snapErr.ResponseMessage,
			PartnerID:       "TRX-XXXXXX-XXXX",
			CallbackURL:     os.Getenv("CALLBACK_URL"),
			FailedAt:        time.Now(),
		}

		if pubErr := producer.Publish(ctx, kafka.TopicTransactionFailed, payload.PartnerReferenceNo, failedEvent); pubErr != nil {
			s.logger.Error("failed to publish transaction.failed", zap.Error(pubErr))
		}

		return "", snapErr
	}

	// Serialisasi AdditionalInfo
	additionalInfoBytes, err := json.Marshal(domain.TransactionAdditionalInfo{
		DeviceID:         payload.AdditionalInfo.DeviceId,
		Channel:          payload.AdditionalInfo.Channel,
		BeneficiaryEmail: payload.BeneficiaryEmail,
		FeeType:          payload.FeeType,
		OriginatorInfos:  payload.OriginatorInfos,
	})
	if err != nil {
		s.logger.Warn("failed to marshal additional info", zap.Error(err))
		additionalInfoBytes = []byte("{}")
	}

	// Generate RefNo
	referenceNoGenerated := fmt.Sprintf("%d%04d", time.Now().Unix(), time.Now().Nanosecond()%10000)

	trx := domain.Transaction{
		ReferenceNo:    referenceNoGenerated,
		FromAccountNo:  payload.SourceAccountNo,
		ToAccountNo:    payload.BeneficiaryAccountNo,
		Amount:         amountValue,
		Currency:       payload.Amount.Currency,
		PartnerRefNo:   payload.PartnerReferenceNo,
		ExternalID:     payload.ExternalID,
		Status:         "SUCCESS",
		Note:           payload.Remark,
		AdditionalInfo: additionalInfoBytes,
		CreatedAt:      time.Now(),
	}

	svcStart := time.Now()
	referenceNo, snapErr := s.repo.TransferIntraBank(ctx, trx)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		errPrefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(errPrefix)
		s.logger.Error("database transaction failed", zap.Error(errPrefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()

		// publish TransactionFailedEvent
		failedEvent := kafka.TransactionFailedEvent{
			TransactionID:   payload.PartnerReferenceNo,
			SenderAccount:   payload.SourceAccountNo,
			ReceiverAccount: payload.BeneficiaryAccountNo,
			Amount:          amountValue,
			Reason:          snapErr.ResponseMessage,
			ErrorCode:       snapErr.GetResponseCode(svcCode),
			HttpCode:        snapErr.HttpCode,
			ErrorMessage:    snapErr.ResponseMessage,
			PartnerID:       "TRX-XXXXXX-XXXX",
			CallbackURL:     os.Getenv("CALLBACK_URL"),
			FailedAt:        time.Now(),
		}

		if pubErr := producer.Publish(ctx, kafka.TopicTransactionFailed, payload.PartnerReferenceNo, failedEvent); pubErr != nil {
			s.logger.Error("failed to publish transaction.failed", zap.Error(pubErr))
		}

		return "", snapErr
	}

	// publish TransactionCreatedEvent
	createdEvent := kafka.TransactionCreatedEvent{
		TransactionID:   referenceNo,
		SenderAccount:   payload.SourceAccountNo,
		ReceiverAccount: payload.BeneficiaryAccountNo,
		Amount:          amountValue,
		Currency:        payload.Amount.Currency,
		CallbackURL:     os.Getenv("CALLBACK_URL"),
		PartnerID:       "TRX-XXXXXX-XXXX",
		CreatedAt:       time.Now(),
	}
	if pubErr := producer.Publish(ctx, kafka.TopicTransactionCreated, referenceNo, createdEvent); pubErr != nil {
		s.logger.Error("failed to publish transaction.created", zap.Error(pubErr))
	}

	// publish AccountBalanceUpdatedEvent (sender)
	senderEvent := kafka.AccountBalanceUpdatedEvent{
		AccountNo:   payload.SourceAccountNo,
		Amount:      amountValue,
		Type:        "out",
		PartnerID:   "TRX-XXXXXX-XXXX",
		CallbackURL: os.Getenv("CALLBACK_URL"),
		UpdatedAt:   time.Now(),
	}
	if pubErr := producer.Publish(ctx, kafka.TopicAccountBalanceUpdated, payload.SourceAccountNo, senderEvent); pubErr != nil {
		s.logger.Error("failed to publish account.balance.updated (sender)", zap.Error(pubErr))
	}

	// publish AccountBalanceUpdatedEvent (receiver)
	receiverEvent := kafka.AccountBalanceUpdatedEvent{
		AccountNo:   payload.BeneficiaryAccountNo,
		Amount:      amountValue,
		Type:        "in",
		PartnerID:   "TRX-XXXXXX-XXXX",
		CallbackURL: os.Getenv("CALLBACK_URL"),
		UpdatedAt:   time.Now(),
	}
	if pubErr := producer.Publish(ctx, kafka.TopicAccountBalanceUpdated, payload.BeneficiaryAccountNo, receiverEvent); pubErr != nil {
		s.logger.Error("failed to publish account.balance.updated (receiver)", zap.Error(pubErr))
	}

	span.SetStatus(codes.Ok, "transfer berhasil")
	span.SetAttributes(attribute.String("service.result.referenceNo", referenceNo))
	s.logger.Info("transfer intrabank success", zap.String("reference_no", referenceNo))
	metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "success").Inc()

	return referenceNo, nil
}

func (s *transactionService) Topup(ctx context.Context, accountID string, producer *kafka.Producer, payload domain.TopupRequest, svcCode string) (string, *domain.SnapDetail) {
	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.Topup")
	defer span.End()
	// operation := "topup"

	amountValue, err := strconv.ParseFloat(payload.Amount.Value, 64)
	if err != nil {
		span.RecordError(err)
		return "", &domain.SnapInvalidFormat
	}

	if amountValue <= 0 {
		return "", &domain.SnapInvalidAmount
	}

	// ─── PANGGIL GRPC ACCOUNT SERVICE UNTUK TOPUP ──────────────────────
	mutationStart := time.Now()
	mutationResp, grpcErr := s.accountCli.ExecuteTopupMutation(ctx, &pbAccount.TopupMutationRequest{
		AccountNo: payload.SourceAccountNo,
		Amount:    int64(amountValue),
	})
	metrics.ServiceDuration.WithLabelValues("grpc_account", "execute_topup").Observe(time.Since(mutationStart).Seconds())

	if grpcErr != nil {
		s.logTopupFailed(ctx, payload, producer, grpcErr.Error(), svcCode)
		return "", &domain.SnapInternalError
	}

	if !mutationResp.Success {
		s.logTopupFailed(ctx, payload, producer, mutationResp.ErrorMessage, svcCode)
		return "", &domain.SnapInternalError
	}

	// ─── SIMPAN TRANSAKSI KE DB ────────────────────────────────────────
	referenceNoGenerated := fmt.Sprintf("TOP%d", time.Now().Unix())
	trx := domain.Transaction{
		ReferenceNo:    referenceNoGenerated,
		FromAccountNo:  "SYSTEM_TOPUP", // Virtual sender
		ToAccountNo:    payload.SourceAccountNo,
		Amount:         amountValue,
		Currency:       payload.Amount.Currency,
		PartnerRefNo:   payload.PartnerReferenceNo,
		ExternalID:     payload.ExternalID,
		Status:         "SUCCESS",
		Note:           "Topup Balance",
		AdditionalInfo: []byte("{}"),
		CreatedAt:      time.Now(),
	}

	referenceNo, snapErr := s.repo.TransferIntraBank(ctx, trx) // Reuse existing repo method
	if snapErr != nil {
		return "", snapErr
	}

	// publish AccountBalanceUpdatedEvent
	producer.Publish(ctx, kafka.TopicAccountBalanceUpdated, payload.SourceAccountNo, kafka.AccountBalanceUpdatedEvent{
		AccountNo: payload.SourceAccountNo,
		Amount:    amountValue,
		Type:      "in",
		UpdatedAt: time.Now(),
	})

	return referenceNo, nil
}

func (s *transactionService) logTopupFailed(ctx context.Context, payload domain.TopupRequest, producer *kafka.Producer, reason string, svcCode string) {
	failedEvent := kafka.TransactionFailedEvent{
		TransactionID:   payload.PartnerReferenceNo,
		SenderAccount:   "SYSTEM_TOPUP",
		ReceiverAccount: payload.SourceAccountNo,
		Amount:          0, // or actual amount
		Reason:          reason,
		FailedAt:        time.Now(),
	}
	producer.Publish(ctx, kafka.TopicTransactionFailed, payload.PartnerReferenceNo, failedEvent)
}
