package service

import (
	"belajar-go/challenge/transactionSystem/internal/api/accounts/repository"
	"belajar-go/challenge/transactionSystem/internal/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	// "errors"
)

type AccountsService interface {
	FetchAllAccounts() ([]models.Account, error)
	FetchAccountById(id string) (models.Account, error)
	FetchTransactionsByAccountId(id string, trxType string) ([]models.Transaction, error)
	CreateNewAccount(account models.Account) (models.Account, error)
	PatchAccountById(account models.Account) (string, error)
	DeleteAccountById(id string) error
}

type accountsService struct {
	repo repository.AccountRepository // Depend pada Interface, bukan struct DB langsung
}

func NewAccountsService(repo repository.AccountRepository) AccountsService {
	return &accountsService{repo: repo}
}

// Fetch All Data
func (s *accountsService) FetchAllAccounts() ([]models.Account, error) {
	return s.repo.GetAllAccounts()
}

// Fetch Account by Id
func (s *accountsService) FetchAccountById(id string) (models.Account, error) {

	_, err := uuid.Parse(id)
	if err != nil {
		// Jika gagal di-parse, kembalikan error validasi
		return models.Account{}, fmt.Errorf("format ID tidak valid atau Data tidak ditemukan")
	}

	return s.repo.GetAccountById(id)
}

// Fetch Transaction by Account Id
func (s *accountsService) FetchTransactionsByAccountId(id string, trxType string) ([]models.Transaction, error) {

	trxTypeEnum := []string{"all", "in", "out"}

	_, err := uuid.Parse(id)
	if err != nil {
		// Jika gagal di-parse, kembalikan error validasi
		return nil, fmt.Errorf("format ID tidak valid atau Data tidak ditemukan")
	}

	isValidType := slices.Contains(trxTypeEnum, trxType)

	if !isValidType {
		return nil, fmt.Errorf("Tipe transaksi tidak ditemukan (all/in/out)!")
	}

	return s.repo.GetTransactionsByAccountId(id, trxType)
}

// Create new account
func (s *accountsService) CreateNewAccount(account models.Account) (models.Account, error) {

	if account.BankCode == "" || account.AccountNumber == "" || account.AccountHolder == "" {
		return models.Account{}, errors.New("Terdapat field yang kosong!")
	}

	if account.Balance < 0 {
		return models.Account{}, errors.New("Balance tidak boleh minus!")
	}

	// Simpan ke repository
	newAccount, err := s.repo.CreateAccount(account)
	if err != nil {
		return models.Account{}, err
	}

	helper.PrintLog("account", helper.LogPositionService, fmt.Sprintf("Berhasil menambahkan data account : %s", newAccount))

	account.ID = uuid.MustParse(newAccount)
	return account, nil
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

	_, err := uuid.Parse(id)
	if err != nil {
		// Jika gagal di-parse, kembalikan error validasi
		return fmt.Errorf("format ID tidak valid atau Data tidak ditemukan")
	}

	return s.repo.DeleteAccount(id)
}
