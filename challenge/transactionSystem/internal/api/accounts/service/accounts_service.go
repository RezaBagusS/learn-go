package service

import (
	"belajar-go/challenge/transactionSystem/internal/api/accounts/repository"
	"belajar-go/challenge/transactionSystem/internal/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"errors"
	"fmt"

	"github.com/google/uuid"
	// "errors"
)

type AccountsService interface {
	FetchAllAccounts() ([]models.Account, error)
	FetchAccountById(id string) (models.Account, error)
	CreateNewAccount(task models.Account) (models.Account, error)
	// PatchTaskById(task models.Task) (int, error)
	// DeleteTaskById(id int) error
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

// Create new task
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

// // Update task
// func (s *taskService) PatchTaskById(task models.Task) (int, error) {
// 	getId, err := s.repo.UpdateTask(task)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return getId, nil
// }

// // Delete task
// func (s *taskService) DeleteTaskById(id int) error {
// 	err := s.repo.DeleteTask(id)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
