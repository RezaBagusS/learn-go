package service

import (
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/banks/repository"
	"belajar-go/challenge/transactionSystem/internal/models"
	"fmt"

	"github.com/google/uuid"
	// "errors"
)

type BankService interface {
	FetchAllBanks() ([]models.Bank, error)
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
func (s *bankService) FetchAllBanks() ([]models.Bank, error) {
	return s.repo.GetAllBanks()
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
