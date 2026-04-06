package service

import (
	"belajar-go/challenge/transactionSystem/internal/api/banks/repository"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"belajar-go/challenge/transactionSystem/observability/metrics"
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type BankService interface {
	FetchAllBanks(ctx context.Context) ([]models.Bank, error)
	FetchBankById(ctx context.Context, id string) (*models.Bank, error)
	CreateNewBank(ctx context.Context, bank models.Bank) (*models.Bank, error)
	PatchBank(ctx context.Context, bank models.Bank) (string, error)
	DeleteBank(ctx context.Context, bankCode string) error
}

type bankService struct {
	repo repository.BankRepository // Depend pada Interface, bukan struct DB langsung
}

func NewBanksService(repo repository.BankRepository) BankService {
	return &bankService{repo: repo}
}

const service = "bank"

// Fetch All Data
func (s *bankService) FetchAllBanks(ctx context.Context) ([]models.Bank, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.GetAll")
	defer span.End()
	operation := "fetch_all"

	logger.Info("fetching banks from repository")

	svcStart := time.Now()
	banks, err := s.repo.GetAllBanks(ctx)
	metrics.ServiceDuration.WithLabelValues(service, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error("failed fetching banks", zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(service, operation, "error").Inc()
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("service.result.count", len(banks)),
	)

	logger.Info("success fetching banks",
		zap.Int("count", len(banks)),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(service, operation, "success").Inc()

	return banks, nil
}

// Fetch Bank by code
func (s *bankService) FetchBankById(ctx context.Context, id string) (*models.Bank, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.GetById")
	defer span.End()
	operation := "fetch_by_id"

	logger.Info("fetching bank from repository")

	svcStart := time.Now()
	bank, err := s.repo.GetBankById(ctx, id)
	metrics.CacheDuration.WithLabelValues(service, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		metrics.CacheRequestsTotal.WithLabelValues(service, operation, "error").Inc()
		return nil, err
	}

	span.SetAttributes(
		attribute.String("service.result.id", bank.ID.String()),
	)

	logger.Info("success fetching bank",
		zap.String("service.result.id", bank.ID.String()),
	)

	metrics.CacheRequestsTotal.WithLabelValues(service, operation, "success").Inc()

	return bank, nil
}

// Create new bank
func (s *bankService) CreateNewBank(ctx context.Context, bank models.Bank) (*models.Bank, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.Create")
	defer span.End()
	operation := "create"

	logger.Info("checking payload")

	// Logika Bisnis: Validasi input tidak boleh kosong
	if bank.BankCode == "" || bank.BankName == "" {
		span.RecordError(models.ErrInvalidField)
		logger.Error(models.ErrInvalidField.Error(), zap.Error(models.ErrInvalidField))
		metrics.BusinessValidationErrors.WithLabelValues(service, operation).Inc()
		metrics.CacheRequestsTotal.WithLabelValues(service, operation, "error").Inc()
		return nil, models.ErrInvalidField
	}

	logger.Info("creating new bank")

	svcStart := time.Now()
	newId, err := s.repo.CreateBank(ctx, bank)
	metrics.ServiceDuration.WithLabelValues(service, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		metrics.CacheRequestsTotal.WithLabelValues(service, operation, "error").Inc()
		return nil, err
	}

	bank.ID = uuid.MustParse(newId)

	span.SetAttributes(
		attribute.String("service.result.id", bank.ID.String()),
	)

	logger.Info("success creating new bank",
		zap.String("service.result.id", bank.ID.String()),
	)

	metrics.CacheRequestsTotal.WithLabelValues(service, operation, "success").Inc()

	return &bank, nil
}

// Update task
func (s *bankService) PatchBank(ctx context.Context, bank models.Bank) (string, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.Update")
	defer span.End()
	operation := "update"

	logger.Info("checking payload")

	// Logika Bisnis: Validasi input tidak boleh kosong
	if bank.ID == uuid.Nil || bank.BankCode == "" || bank.BankName == "" {
		span.RecordError(models.ErrInvalidField)
		logger.Error(models.ErrInvalidField.Error(), zap.Error(models.ErrInvalidField))
		metrics.BusinessValidationErrors.WithLabelValues(service, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(service, operation, "error").Inc()
		return "", models.ErrInvalidField
	}

	logger.Info("updating bank data")

	svcStart := time.Now()
	bankCode, err := s.repo.UpdateBank(ctx, bank)
	metrics.CacheDuration.WithLabelValues(service, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(service, operation, "error").Inc()
		return "", err
	}

	span.SetAttributes(
		attribute.String("service.result.bankCode", bankCode),
	)

	logger.Info("success updating bank data",
		zap.String("service.result.bankCode", bankCode),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(service, operation, "success").Inc()

	return bankCode, nil
}

// Delete bank
func (s *bankService) DeleteBank(ctx context.Context, bankId string) error {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.Delete")
	defer span.End()
	operation := "delete"

	logger.Info("deleting bank data",
		zap.String("service.delete.id", bankId),
	)

	svcStart := time.Now()
	err := s.repo.DeleteBank(ctx, bankId)
	metrics.CacheDuration.WithLabelValues(service, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		metrics.CacheRequestsTotal.WithLabelValues(service, operation, "error").Inc()
		return err
	}

	span.SetAttributes(
		attribute.String("service.delete.id", bankId),
	)

	logger.Info("success deleting bank data",
		zap.String("service.delete.id", bankId),
	)

	metrics.CacheRequestsTotal.WithLabelValues(service, operation, "success").Inc()

	return nil
}
