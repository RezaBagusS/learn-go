package service

import (
	"belajar-go/challenge/transactionSystem/internal/api/accounts/repository"
	"belajar-go/challenge/transactionSystem/internal/models"
	"fmt"

	"github.com/google/uuid"
	// "errors"
)

type AccountsService interface {
	FetchAllAccounts() ([]models.Account, error)
	FetchAccountById(id string) (models.Account, error)
	// CreateNewTask(task models.Task) (models.Task, error)
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
		return models.Account{}, fmt.Errorf("format ID tidak valid: harus berupa UUID")
	}

	return s.repo.GetAccountById(id)
}

// // Create new task
// func (s *taskService) CreateNewTask(task models.Task) (models.Task, error) {
// 	// Logika Bisnis: Validasi input tidak boleh kosong
// 	if task.Title == "" {
// 		return models.Task{}, errors.New("title task tidak boleh kosong")
// 	}

// 	// Simpan ke repository
// 	insertedID, err := s.repo.CreateTask(task)
// 	if err != nil {
// 		return models.Task{}, err
// 	}

// 	// Lengkapi data task dengan ID yang baru digenerate oleh database
// 	task.ID = insertedID
// 	task.IsCompleted = false // Default
// 	return task, nil
// }

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
