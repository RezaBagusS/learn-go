package service

import (
	"belajar-go/challenge/transactionSystem/internal/api/banks/repository"
	"belajar-go/challenge/transactionSystem/internal/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"fmt"

	"github.com/google/uuid"
	// "errors"
)

type BankService interface {
	FetchAllBanks() ([]models.Bank, error)
	FetchBankByCode(bankCode string) (*models.Bank, error)
	CreateNewBank(bank models.Bank) (models.Bank, error)
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
func (s *bankService) FetchBankByCode(bankCode string) (*models.Bank, error) {
	return s.repo.GetBankByCode(bankCode)
}

// Create new bank
func (s *bankService) CreateNewBank(bank models.Bank) (models.Bank, error) {
	// Logika Bisnis: Validasi input tidak boleh kosong
	if bank.BankCode == "" || bank.BankName == "" {
		return models.Bank{}, fmt.Errorf("Field tidak boleh kosong!")
	}

	// Simpan ke repository
	newBank, err := s.repo.CreateBank(bank)
	if err != nil {
		return models.Bank{}, err
	}

	helper.PrintLog("bank", helper.LogPositionService, fmt.Sprintf("Berhasil menambahkan data bank : %s", newBank))

	return bank, nil
}

// Update task
func (s *bankService) PatchBank(bank models.Bank) (string, error) {
	bankId, err := s.repo.UpdateBank(bank)
	if err != nil {
		return "", err
	}

	return bankId, nil
}

// Delete bank
func (s *bankService) DeleteBank(bankId string) error {

	_, err := uuid.Parse(bankId)
	if err != nil {
		// Jika gagal di-parse, kembalikan error validasi
		return fmt.Errorf("format ID tidak valid: harus berupa UUID")
	}

	return s.repo.DeleteBank(bankId)
}
