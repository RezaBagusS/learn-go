package service

import (
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/transactions/repository"
	"belajar-go/challenge/transactionSystem/internal/models"
	"fmt"
	"time"
	// "errors"
)

type TransactionService interface {
	FetchAllTransactions() ([]models.Transaction, error)
	FetchTransactionById(id string) (*models.Transaction, error)
	CreateTrx(trx models.Transaction) (string, error)
	FetchSummaryToday(date time.Time) ([]models.Transaction, error)
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

// Fetch Transaction by date only
func (s *transactionService) FetchSummaryToday(date time.Time) ([]models.Transaction, error) {

	if date.After(time.Now()) {
		helper.PrintLog("account", helper.LogPositionService, models.ErrInvalidFutureDate.Error())
		return nil, models.ErrInvalidFutureDate
	}

	return s.repo.GetSummary(date)
}

// Fetch Transaction by Id
func (s *transactionService) FetchTransactionById(id string) (*models.Transaction, error) {
	return s.repo.GetTransactionById(id)
}

// Create new transaction
func (s *transactionService) CreateTrx(trx models.Transaction) (string, error) {

	if trx.Amount <= 0 {
		helper.PrintLog("account", helper.LogPositionService, models.ErrInvalidTranserAmount.Error())
		return "", models.ErrInvalidTranserAmount
	}

	if trx.FromAccountID == trx.ToAccountID {
		helper.PrintLog("account", helper.LogPositionService, models.ErrLogicSelfTranser.Error())
		return "", models.ErrLogicSelfTranser
	}

	// (Opsional) Validasi tambahan, misalnya maksimal karakter 'Note'
	if len(trx.Note) > 255 {
		helper.PrintLog("account", helper.LogPositionService, models.ErrInvalidMaximumNote.Error())
		return "", models.ErrInvalidMaximumNote
	}

	transactionID, err := s.repo.CreateTransaction(trx)
	if err != nil {
		helper.PrintLog("transaction", helper.LogPositionService, err.Error())
		return "", err
	}

	helper.PrintLog("transaction", helper.LogPositionService, fmt.Sprintf("Transaksi berhasil dicatat dengan ID: %s", transactionID))

	return transactionID, nil
}
