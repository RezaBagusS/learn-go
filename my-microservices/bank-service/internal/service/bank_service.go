package service

import (
	"context"
	"errors"
	"my-microservices/bank-service/helper"
	"my-microservices/bank-service/internal/domain"
	"my-microservices/bank-service/internal/middleware"
	"my-microservices/bank-service/internal/repository"
	"my-microservices/bank-service/observability/metrics"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type BankService interface {
	FetchAllBanks(ctx context.Context) ([]domain.Bank, error)
	FetchBankById(ctx context.Context, id string) (*domain.Bank, *domain.SnapDetail)
	CreateNewBank(ctx context.Context, bank domain.Bank) (*domain.Bank, error)
	PatchBank(ctx context.Context, bank domain.Bank) (string, error)
	DeleteBank(ctx context.Context, bankCode string) error
}

type bankService struct {
	repo   repository.BankRepository // Depend pada Interface, bukan struct DB langsung
	logger *zap.Logger
}

func NewBanksService(repo repository.BankRepository) BankService {
	logger := helper.Log

	return &bankService{
		repo:   repo,
		logger: logger,
	}
}

const svcBank = "bank"

// Fetch All Data
func (s *bankService) FetchAllBanks(ctx context.Context) ([]domain.Bank, error) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.GetAll")
	defer span.End()
	operation := "fetch_all"

	s.logger.Info("fetching banks from repository")

	svcStart := time.Now()
	banks, err := s.repo.GetAllBanks(ctx)
	metrics.ServiceDuration.WithLabelValues(svcBank, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		s.logger.Error("failed fetching banks", zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "error").Inc()
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("service.result.count", len(banks)),
	)

	s.logger.Info("success fetching banks",
		zap.Int("count", len(banks)),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "success").Inc()

	return banks, nil
}

// Fetch Bank by code
func (s *bankService) FetchBankById(ctx context.Context, id string) (*domain.Bank, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.GetById")
	defer span.End()
	operation := "fetch_by_id"

	s.logger.Info("fetching bank from repository")

	svcStart := time.Now()
	bank, snapErr := s.repo.GetBankById(ctx, id)
	metrics.CacheDuration.WithLabelValues(svcBank, operation).
		Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		prefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefix)
		s.logger.Error(prefix.Error(), zap.Error(prefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "error").Inc()
		return nil, snapErr
	}

	span.SetAttributes(
		attribute.String("service.result.id", bank.ID.String()),
	)

	s.logger.Info("success fetching bank",
		zap.String("service.result.id", bank.ID.String()),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "success").Inc()

	return bank, nil
}

// Create new bank
func (s *bankService) CreateNewBank(ctx context.Context, bank domain.Bank) (*domain.Bank, error) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.Create")
	defer span.End()
	operation := "create"

	s.logger.Info("checking payload")

	// Logika Bisnis: Validasi input tidak boleh kosong
	if bank.BankCode == "" || bank.BankName == "" {
		span.RecordError(domain.ErrInvalidField)
		s.logger.Error(domain.ErrInvalidField.Error(), zap.Error(domain.ErrInvalidField))
		metrics.BusinessValidationErrors.WithLabelValues(svcBank, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "error").Inc()
		return nil, domain.ErrInvalidField
	}

	s.logger.Info("creating new bank")

	svcStart := time.Now()
	newId, err := s.repo.CreateBank(ctx, bank)
	metrics.ServiceDuration.WithLabelValues(svcBank, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		s.logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "error").Inc()
		return nil, err
	}

	bank.ID = uuid.MustParse(newId)

	span.SetAttributes(
		attribute.String("service.result.id", bank.ID.String()),
	)

	s.logger.Info("success creating new bank",
		zap.String("service.result.id", bank.ID.String()),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "success").Inc()

	return &bank, nil
}

// Update task
func (s *bankService) PatchBank(ctx context.Context, bank domain.Bank) (string, error) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.Update")
	defer span.End()
	operation := "update"

	s.logger.Info("checking payload")

	// Logika Bisnis: Validasi input tidak boleh kosong
	if bank.ID == uuid.Nil || bank.BankCode == "" && bank.BankName == "" {
		span.RecordError(domain.ErrInvalidField)
		s.logger.Error(domain.ErrInvalidField.Error(), zap.Error(domain.ErrInvalidField))
		metrics.BusinessValidationErrors.WithLabelValues(svcBank, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "error").Inc()
		return "", domain.ErrInvalidField
	}

	s.logger.Info("updating bank data")

	svcStart := time.Now()
	bankCode, err := s.repo.UpdateBank(ctx, bank)
	metrics.CacheDuration.WithLabelValues(svcBank, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		s.logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "error").Inc()
		return "", err
	}

	span.SetAttributes(
		attribute.String("service.result.bankCode", bankCode),
	)

	s.logger.Info("success updating bank data",
		zap.String("service.result.bankCode", bankCode),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "success").Inc()

	return bankCode, nil
}

// Delete bank
func (s *bankService) DeleteBank(ctx context.Context, bankId string) error {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.Delete")
	defer span.End()
	operation := "delete"

	s.logger.Info("deleting bank data",
		zap.String("service.delete.id", bankId),
	)

	svcStart := time.Now()
	err := s.repo.DeleteBank(ctx, bankId)
	metrics.CacheDuration.WithLabelValues(svcBank, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		s.logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "error").Inc()
		return err
	}

	span.SetAttributes(
		attribute.String("service.delete.id", bankId),
	)

	s.logger.Info("success deleting bank data",
		zap.String("service.delete.id", bankId),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcBank, operation, "success").Inc()

	return nil
}
