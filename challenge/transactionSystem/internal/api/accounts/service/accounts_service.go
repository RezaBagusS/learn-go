package service

import (
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/accounts/repository"
	"belajar-go/challenge/transactionSystem/internal/api/banks/service"
	"belajar-go/challenge/transactionSystem/internal/models"
	"log"

	"github.com/google/uuid"
)

type AccountsService interface {
	FetchAllAccounts() ([]models.Account, error)
	FetchAccountById(id string) (*models.Account, error)
	FetchTransactionsByAccountId(id string, trxType string) ([]models.Transaction, error)
	CreateNewAccount(account models.Account) (*models.Account, error)
	PatchAccountById(account models.Account) (string, error)
	DeleteAccountById(id string) error
}

type accountsService struct {
	repo    repository.AccountRepository // Depend pada Interface, bukan struct DB langsung
	bankSvc service.BankService
}

func NewAccountsService(repo repository.AccountRepository, bankSvc service.BankService) AccountsService {
	return &accountsService{
		repo:    repo,
		bankSvc: bankSvc,
	}
}

// Fetch All Data
func (s *accountsService) FetchAllAccounts() ([]models.Account, error) {
	return s.repo.GetAllAccounts()
}

// Fetch Account by Id
func (s *accountsService) FetchAccountById(id string) (*models.Account, error) {
	return s.repo.GetAccountById(id)
}

// Fetch Transaction by Account Id
func (s *accountsService) FetchTransactionsByAccountId(id string, trxType string) ([]models.Transaction, error) {

	// Check account exist
	_, err := s.repo.GetAccountById(id)
	if err != nil {
		return nil, err
	}

	return s.repo.GetTransactionsByAccountId(id, trxType)
}

// Create new account
func (s *accountsService) CreateNewAccount(account models.Account) (*models.Account, error) {

	// Balance checking ...
	if account.Balance < 0 {
		helper.PrintLog("account", helper.LogPositionService, models.ErrInvalidInitBalance.Error())
		return nil, models.ErrInvalidInitBalance
	}

	// Bank Checking ...
	data, err := s.bankSvc.FetchBankById(account.BankCode)

	log.Println(data)

	if err != nil {
		return nil, models.ErrInvalidBankCode
	}

	// Simpan ke repository
	newAccount, err := s.repo.CreateAccount(account)
	if err != nil {
		return nil, err
	}

	account.ID = uuid.MustParse(newAccount)
	return &account, nil
}

// Update account
func (s *accountsService) PatchAccountById(account models.Account) (string, error) {
	getId, err := s.repo.UpdateAccount(account)
	if err != nil {
		return "", err
	}

	return getId, nil
}

// Delete account
func (s *accountsService) DeleteAccountById(id string) error {
	return s.repo.DeleteAccount(id)
}
