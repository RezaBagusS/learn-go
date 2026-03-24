package service

import (
	"belajar-go/challenge/transactionSystem/internal/api/transactions/repository"
	"belajar-go/challenge/transactionSystem/internal/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"errors"
	"fmt"

	"github.com/google/uuid"
	// "errors"
)

type TransactionService interface {
	FetchAllTransactions() ([]models.Transaction, error)
	FetchTransactionById(id string) (models.Transaction, error)
	CreateTrx(trx models.Transaction) (string, error)
	// PatchBank(bank models.Bank) (string, error)
	// DeleteBank(bankCode string) error
}

type transactionService struct {
	repo repository.TransactionRepository // Depend pada Interface, bukan struct DB langsung
}

func NewTransactionsService(repo repository.TransactionRepository) TransactionService {
	return &transactionService{repo: repo}
}

// Fetch All Data
func (s *transactionService) FetchAllTransactions() ([]models.Transaction, error) {
	return s.repo.GetAllTransactions()
}

// Fetch Transaction by Id
func (s *transactionService) FetchTransactionById(id string) (models.Transaction, error) {

	_, err := uuid.Parse(id)
	if err != nil {
		// Jika gagal di-parse, kembalikan error validasi
		return models.Transaction{}, fmt.Errorf("format ID tidak valid: harus berupa UUID")
	}

	return s.repo.GetTransactionById(id)
}

// Create new transaction
func (s *transactionService) CreateTrx(trx models.Transaction) (string, error) {
	if trx.FromAccountID == "" || trx.ToAccountID == "" {
		return "", errors.New("rekening pengirim dan penerima tidak boleh kosong")
	}

	if trx.Amount <= 0 {
		return "", errors.New("nominal transfer harus lebih besar dari 0")
	}

	if trx.FromAccountID == trx.ToAccountID {
		return "", errors.New("tidak dapat melakukan transfer ke rekening sendiri")
	}

	// (Opsional) Validasi tambahan, misalnya maksimal karakter 'Note'
	if len(trx.Note) > 255 {
		return "", errors.New("catatan transfer maksimal 255 karakter")
	}

	transactionID, err := s.repo.CreateTransaction(trx)
	if err != nil {
		helper.PrintLog("transaction", helper.LogPositionService, fmt.Sprintf("Gagal memproses transaksi: %v", err))
		return "", err
	}

	helper.PrintLog("transaction", helper.LogPositionService, fmt.Sprintf("Transaksi berhasil dicatat dengan ID: %s", transactionID))

	return transactionID, nil
}
