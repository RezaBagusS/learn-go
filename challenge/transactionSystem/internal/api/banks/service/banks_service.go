package service

import (
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/banks/repository"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	// "errors"
)

type BankService interface {
	FetchAllBanks(ctx context.Context) ([]models.Bank, error)
	FetchBankById(ctx context.Context, id string) (*models.Bank, error)
	CreateNewBank(ctx context.Context, bank models.Bank) (*models.Bank, error)
	PatchBank(bank models.Bank) (string, error)
	DeleteBank(bankCode string) error
}

type bankService struct {
	repo repository.BankRepository // Depend pada Interface, bukan struct DB langsung
}

func NewBanksService(repo repository.BankRepository) BankService {
	return &bankService{repo: repo}
}

// Fetch All Data
func (s *bankService) FetchAllBanks(ctx context.Context) ([]models.Bank, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.GetAll")
	defer span.End()

	logger.Info("fetching banks from repository")

	banks, err := s.repo.GetAllBanks(ctx)
	if err != nil {
		span.RecordError(err)
		logger.Error("failed fetching banks", zap.Error(err))
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("service.result.count", len(banks)),
	)

	logger.Info("success fetching banks",
		zap.Int("count", len(banks)),
	)

	return banks, nil
}

// Fetch Bank by code
func (s *bankService) FetchBankById(ctx context.Context, id string) (*models.Bank, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.GetById")
	defer span.End()

	logger.Info("fetching banks from repository")

	bank, err := s.repo.GetBankById(ctx, id)
	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		return nil, err
	}

	span.SetAttributes(
		attribute.String("service.result.id", bank.ID.String()),
	)

	logger.Info("success fetching banks",
		zap.String("service.result.id", bank.ID.String()),
	)

	return bank, nil
}

// Create new bank
func (s *bankService) CreateNewBank(ctx context.Context, bank models.Bank) (*models.Bank, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.Create")
	defer span.End()

	logger.Info("checking payload")

	// Logika Bisnis: Validasi input tidak boleh kosong
	if bank.BankCode == "" || bank.BankName == "" {
		span.RecordError(models.ErrInvalidField)
		logger.Error(models.ErrInvalidField.Error(), zap.Error(models.ErrInvalidField))
		return nil, models.ErrInvalidField
	}

	logger.Info("creating new bank")

	// Simpan ke repository
	newId, err := s.repo.CreateBank(ctx, bank)
	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		return nil, err
	}

	bank.ID = uuid.MustParse(newId)

	span.SetAttributes(
		attribute.String("service.result.id", bank.ID.String()),
	)

	logger.Info("success creating new bank",
		zap.String("service.result.id", bank.ID.String()),
	)

	return &bank, nil
}

// Update task
func (s *bankService) PatchBank(bank models.Bank) (string, error) {

	// Logika Bisnis: Validasi input tidak boleh kosong
	if bank.BankCode == "" && bank.BankName == "" {
		helper.PrintLog("bank", helper.LogPositionHandler, models.ErrInvalidField.Error())
		return "", models.ErrInvalidField
	}

	bankCode, err := s.repo.UpdateBank(bank)
	if err != nil {
		return "", err
	}

	return bankCode, nil
}

// Delete bank
func (s *bankService) DeleteBank(bankId string) error {
	return s.repo.DeleteBank(bankId)
}
