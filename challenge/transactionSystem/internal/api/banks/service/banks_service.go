package service

import (
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/banks/repository"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	// "errors"
)

type BankService interface {
	FetchAllBanks(ctx context.Context) ([]models.Bank, error)
	FetchBankById(id string) (*models.Bank, error)
	CreateNewBank(bank models.Bank) (*models.Bank, error)
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
	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankService.GetAll")
	defer span.End()

	logger := middleware.LoggerFromCtx(ctx)

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
func (s *bankService) FetchBankById(id string) (*models.Bank, error) {
	return s.repo.GetBankById(id)
}

// Create new bank
func (s *bankService) CreateNewBank(bank models.Bank) (*models.Bank, error) {

	// Logika Bisnis: Validasi input tidak boleh kosong
	if bank.BankCode == "" || bank.BankName == "" {
		helper.PrintLog("bank", helper.LogPositionHandler, models.ErrInvalidField.Error())
		return nil, models.ErrInvalidField
	}

	// Simpan ke repository
	newId, err := s.repo.CreateBank(bank)
	if err != nil {
		return nil, err
	}

	helper.PrintLog("bank", helper.LogPositionService, fmt.Sprintf("Berhasil menambahkan data bank : %s", newId))

	bank.ID = uuid.MustParse(newId)

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
