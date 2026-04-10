package service

import (
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/transactions/repository"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"belajar-go/challenge/transactionSystem/observability/metrics"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	// "errors"
)

type TransactionService interface {
	FetchAllTransactions(ctx context.Context) ([]models.Transaction, *models.SnapDetail)
	FetchTransactionById(ctx context.Context, id string) (*models.Transaction, *models.SnapDetail)
	// CreateTrx(ctx context.Context, trx models.Transaction) (string, error)
	FetchSummaryToday(ctx context.Context, date time.Time) ([]models.Transaction, *models.SnapDetail)
	TransferIntrabank(ctx context.Context, accountID string, header models.SNAPHeader, payload models.TransferIntrabankRequest) (string, *models.SnapDetail)
}

type transactionService struct {
	repo   repository.TransactionRepository // Depend pada Interface, bukan struct DB langsung
	logger *zap.Logger
}

func NewTransactionsService(repo repository.TransactionRepository) TransactionService {
	logger := helper.Log

	return &transactionService{
		repo:   repo,
		logger: logger,
	}
}

const svcTransaction = "transaction"

// Fetch All Data
func (s *transactionService) FetchAllTransactions(ctx context.Context) ([]models.Transaction, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.GetAll")
	defer span.End()
	operation := "fetch_all"

	s.logger.Info("fetching transactions from repository")

	svcStart := time.Now()
	transactions, snapErr := s.repo.GetAllTransactions(ctx)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).
		Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		errPrefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(errPrefix)
		s.logger.Error("failed fetching transactions", zap.Error(errPrefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, snapErr
	}

	span.SetAttributes(
		attribute.Int("service.result.count", len(transactions)),
	)

	s.logger.Info("success fetching transactions",
		zap.Int("count", len(transactions)),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "success").Inc()

	return transactions, nil
}

// Fetch Transaction by date only
func (s *transactionService) FetchSummaryToday(ctx context.Context, date time.Time) ([]models.Transaction, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.GetSummary")
	defer span.End()
	operation := "fetch_summary"

	s.logger.Info("fetching transaction summary from repository", zap.String("date", date.Format("2006-01-02")))

	if date.After(time.Now()) {
		err := models.ErrInvalidFutureDate
		span.RecordError(err)
		s.logger.Error("invalid future date", zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, &models.SnapInvalidFormat
	}

	svcStart := time.Now()
	transactions, snapErr := s.repo.GetSummary(ctx, date)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).
		Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		errPrefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(errPrefix)
		s.logger.Error("failed fetching transaction summary", zap.Error(errPrefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, snapErr
	}

	span.SetAttributes(
		attribute.Int("service.result.count", len(transactions)),
		attribute.String("service.query.date", date.Format("2006-01-02")),
	)

	s.logger.Info("success fetching transaction summary",
		zap.Int("count", len(transactions)),
		zap.String("date", date.Format("2006-01-02")),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "success").Inc()

	return transactions, nil
}

// Fetch Transaction by Id
func (s *transactionService) FetchTransactionById(ctx context.Context, id string) (*models.Transaction, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.GetById")
	defer span.End()
	operation := "fetch_by_id"

	s.logger.Info("fetching transaction from repository", zap.String("id", id))

	svcStart := time.Now()
	transaction, snapErr := s.repo.GetTransactionById(ctx, id)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).
		Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		prefixError := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefixError)
		s.logger.Error("failed fetching transaction", zap.Error(prefixError))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, snapErr
	}

	span.SetAttributes(
		attribute.String("service.query.id", id),
	)

	s.logger.Info("success fetching transaction",
		zap.String("id", id),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "success").Inc()

	return transaction, nil
}

// Create new transaction
// func (s *transactionService) CreateTrx(ctx context.Context, trx models.Transaction) (string, error) {

// 	tracer := middleware.TracerFromCtx(ctx)
// 	ctx, span := tracer.Start(ctx, "TransactionService.Create")
// 	defer span.End()
// 	operation := "create"

// 	s.logger.Info("checking payload")

// 	// Logika Bisnis: Validasi input tidak boleh kosong
// 	if trx.FromAccountID == "" || trx.ToAccountID == "" || trx.Amount == 0 {
// 		span.RecordError(models.ErrInvalidField)
// 		s.logger.Error(models.ErrInvalidField.Error(), zap.Error(models.ErrInvalidField))
// 		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
// 		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
// 		return "", models.ErrInvalidField
// 	}

// 	// Logika Bisnis: Validasi amount harus positif
// 	if trx.Amount <= 0 {
// 		span.RecordError(models.ErrInvalidTranserAmount)
// 		s.logger.Error(models.ErrInvalidTranserAmount.Error(), zap.Error(models.ErrInvalidTranserAmount))
// 		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
// 		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
// 		return "", models.ErrInvalidTranserAmount
// 	}

// 	// Logika Bisnis: Validasi tidak boleh transfer ke diri sendiri
// 	if trx.FromAccountID == trx.ToAccountID {
// 		span.RecordError(models.ErrLogicSelfTranser)
// 		s.logger.Error(models.ErrLogicSelfTranser.Error(), zap.Error(models.ErrLogicSelfTranser))
// 		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
// 		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
// 		return "", models.ErrLogicSelfTranser
// 	}

// 	// Logika Bisnis: Validasi maksimal karakter Note
// 	if len(trx.Note) > 255 {
// 		span.RecordError(models.ErrInvalidMaximumNote)
// 		s.logger.Error(models.ErrInvalidMaximumNote.Error(), zap.Error(models.ErrInvalidMaximumNote))
// 		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
// 		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
// 		return "", models.ErrInvalidMaximumNote
// 	}

// 	s.logger.Info("creating new transaction")

// 	svcStart := time.Now()
// 	transactionID, err := s.repo.CreateTransaction(ctx, trx)
// 	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).
// 		Observe(time.Since(svcStart).Seconds())

// 	if err != nil {
// 		span.RecordError(err)
// 		s.logger.Error(err.Error(), zap.Error(err))
// 		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
// 		return "", err
// 	}

// 	span.SetAttributes(
// 		attribute.String("service.result.id", transactionID),
// 	)

// 	s.logger.Info("success creating new transaction",
// 		zap.String("service.result.id", transactionID),
// 	)

// 	metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "success").Inc()

// 	return transactionID, nil
// }

func (s *transactionService) TransferIntrabank(ctx context.Context, accountID string, header models.SNAPHeader, payload models.TransferIntrabankRequest) (string, *models.SnapDetail) {
	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.TransferIntrabank")
	defer span.End()

	operation := "transfer_intrabank"
	s.logger.Info("processing transfer intrabank",
		zap.String("partner_ref", payload.PartnerReferenceNo),
		zap.String("account_id", accountID),
	)

	// Validasi & konversi amount
	amountValue, err := strconv.ParseFloat(payload.Amount.Value, 64)
	if err != nil {
		span.RecordError(err)
		s.logger.Error("invalid amount format", zap.Error(err))
		return "", &models.SnapInvalidFormat
	}

	if amountValue <= 0 {
		span.RecordError(models.ErrInvalidTranserAmount)
		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
		return "", &models.SnapInsufficient
	}

	// Validasi self-transfer
	if payload.SourceAccountNo == payload.BeneficiaryAccountNo {
		span.RecordError(models.ErrLogicSelfTranser)
		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
		return "", &models.SnapBadRequest
	}

	// Serialisasi AdditionalInfo LENGKAP — sertakan semua field SNAP
	additionalInfoBytes, err := json.Marshal(models.TransactionAdditionalInfo{
		DeviceID:         payload.AdditionalInfo.DeviceId,
		Channel:          payload.AdditionalInfo.Channel,
		BeneficiaryEmail: payload.BeneficiaryEmail,
		FeeType:          payload.FeeType,
		OriginatorInfos:  payload.OriginatorInfos,
	})
	if err != nil {
		s.logger.Warn("failed to marshal additional info, using empty object", zap.Error(err))
		additionalInfoBytes = []byte("{}")
	}

	trx := models.Transaction{
		FromAccountID:   payload.SourceAccountNo,
		ToAccountID:     payload.BeneficiaryAccountNo,
		Amount:          amountValue,
		Currency:        payload.Amount.Currency,
		PartnerRefNo:    payload.PartnerReferenceNo,
		ExternalID:      header.ExternalID,
		ResponseCode:    "2001700",
		ResponseMessage: "Request has been processed successfully",
		Status:          "SUCCESS",
		Note:            payload.Remark, // fix: Remark bukan CustomerReference
		AdditionalInfo:  additionalInfoBytes,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	s.logger.Info("persisting transaction to database")

	svcStart := time.Now()
	referenceNo, snapErr := s.repo.TransferIntraBank(ctx, trx)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).
		Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		errPrefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(errPrefix)
		s.logger.Error("database transaction failed", zap.Error(errPrefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return "", snapErr
	}

	span.SetStatus(codes.Ok, "transfer berhasil")
	span.SetAttributes(
		attribute.String("service.result.referenceNo", referenceNo),
		attribute.String("service.result.partnerRefNo", payload.PartnerReferenceNo),
	)

	s.logger.Info("transfer intrabank processed successfully",
		zap.String("reference_no", referenceNo),
		zap.String("partner_ref_no", payload.PartnerReferenceNo),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "success").Inc()

	return referenceNo, nil
}
