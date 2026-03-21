package service

import (
	"belajar-go/challenge/transactionSystem/internal/api/banks/repository"
	"belajar-go/challenge/transactionSystem/internal/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"fmt"
	// "errors"
)

type BankService interface {
	FetchAllBanks() ([]models.Bank, error)
	// FetchAccountById(id string) (models.Account, error)
	CreateNewBank(bank models.Bank) (models.Bank, error)
	// PatchTaskById(task models.Task) (int, error)
	// DeleteTaskById(id int) error
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

// Fetch Account by Id
// func (s *bankService) FetchAccountById(id string) (models.Bank, error) {

// 	_, err := uuid.Parse(id)
// 	if err != nil {
// 		// Jika gagal di-parse, kembalikan error validasi
// 		return models.Bank{}, fmt.Errorf("format ID tidak valid: harus berupa UUID")
// 	}

// 	return s.repo.GetAccountById(id)
// }

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
