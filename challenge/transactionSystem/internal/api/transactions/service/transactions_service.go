package service

import (
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/transactions/repository"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"belajar-go/challenge/transactionSystem/observability/metrics"
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	// "errors"
)

type TransactionService interface {
	FetchAllTransactions(ctx context.Context) ([]models.Transaction, error)
	FetchTransactionById(ctx context.Context, id string) (*models.Transaction, error)
	CreateTrx(ctx context.Context, trx models.Transaction) (string, error)
	FetchSummaryToday(ctx context.Context, date time.Time) ([]models.Transaction, error)
	// PatchBank(bank models.Bank) (string, error)
	// DeleteBank(bankCode string) error
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
func (s *transactionService) FetchAllTransactions(ctx context.Context) ([]models.Transaction, error) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.GetAll")
	defer span.End()
	operation := "fetch_all"

	s.logger.Info("fetching transactions from repository")

	svcStart := time.Now()
	transactions, err := s.repo.GetAllTransactions(ctx)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		s.logger.Error("failed fetching transactions", zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, err
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
func (s *transactionService) FetchSummaryToday(ctx context.Context, date time.Time) ([]models.Transaction, error) {

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
		return nil, err
	}

	svcStart := time.Now()
	transactions, err := s.repo.GetSummary(ctx, date)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		s.logger.Error("failed fetching transaction summary", zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, err
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
func (s *transactionService) FetchTransactionById(ctx context.Context, id string) (*models.Transaction, error) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.GetById")
	defer span.End()
	operation := "fetch_by_id"

	s.logger.Info("fetching transaction from repository", zap.String("id", id))

	svcStart := time.Now()
	transaction, err := s.repo.GetTransactionById(ctx, id)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		s.logger.Error("failed fetching transaction", zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return nil, err
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
func (s *transactionService) CreateTrx(ctx context.Context, trx models.Transaction) (string, error) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionService.Create")
	defer span.End()
	operation := "create"

	s.logger.Info("checking payload")

	// Logika Bisnis: Validasi input tidak boleh kosong
	if trx.FromAccountID == "" || trx.ToAccountID == "" || trx.Amount == 0 {
		span.RecordError(models.ErrInvalidField)
		s.logger.Error(models.ErrInvalidField.Error(), zap.Error(models.ErrInvalidField))
		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return "", models.ErrInvalidField
	}

	// Logika Bisnis: Validasi amount harus positif
	if trx.Amount <= 0 {
		span.RecordError(models.ErrInvalidTranserAmount)
		s.logger.Error(models.ErrInvalidTranserAmount.Error(), zap.Error(models.ErrInvalidTranserAmount))
		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return "", models.ErrInvalidTranserAmount
	}

	// Logika Bisnis: Validasi tidak boleh transfer ke diri sendiri
	if trx.FromAccountID == trx.ToAccountID {
		span.RecordError(models.ErrLogicSelfTranser)
		s.logger.Error(models.ErrLogicSelfTranser.Error(), zap.Error(models.ErrLogicSelfTranser))
		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return "", models.ErrLogicSelfTranser
	}

	// Logika Bisnis: Validasi maksimal karakter Note
	if len(trx.Note) > 255 {
		span.RecordError(models.ErrInvalidMaximumNote)
		s.logger.Error(models.ErrInvalidMaximumNote.Error(), zap.Error(models.ErrInvalidMaximumNote))
		metrics.BusinessValidationErrors.WithLabelValues(svcTransaction, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return "", models.ErrInvalidMaximumNote
	}

	s.logger.Info("creating new transaction")

	svcStart := time.Now()
	transactionID, err := s.repo.CreateTransaction(ctx, trx)
	metrics.ServiceDuration.WithLabelValues(svcTransaction, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		s.logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "error").Inc()
		return "", err
	}

	span.SetAttributes(
		attribute.String("service.result.id", transactionID),
	)

	s.logger.Info("success creating new transaction",
		zap.String("service.result.id", transactionID),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcTransaction, operation, "success").Inc()

	return transactionID, nil
}
