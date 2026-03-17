package service

import (
	"belajar-go/projectAPI/internal/api/task/repository"
	"belajar-go/projectAPI/internal/models"
	"errors"
)

type TaskService interface {
	FetchAllTasks() ([]models.Task, error)
	CreateNewTask(task models.Task) (models.Task, error)
	PatchTaskById(task models.Task) (int, error)
	DeleteTaskById(id int) error
}

type taskService struct {
	repo repository.TaskRepository // Depend pada Interface, bukan struct DB langsung
}

func NewTaskService(repo repository.TaskRepository) TaskService {
	return &taskService{repo: repo}
}

// Fetch All Data
func (s *taskService) FetchAllTasks() ([]models.Task, error) {
	return s.repo.GetAllTasks()
}

// Create new task
func (s *taskService) CreateNewTask(task models.Task) (models.Task, error) {
	// Logika Bisnis: Validasi input tidak boleh kosong
	if task.Title == "" {
		return models.Task{}, errors.New("title task tidak boleh kosong")
	}

	// Simpan ke repository
	insertedID, err := s.repo.CreateTask(task)
	if err != nil {
		return models.Task{}, err
	}

	// Lengkapi data task dengan ID yang baru digenerate oleh database
	task.ID = insertedID
	task.IsCompleted = false // Default
	return task, nil
}

// Update task
func (s *taskService) PatchTaskById(task models.Task) (int, error) {
	getId, err := s.repo.UpdateTask(task)
	if err != nil {
		return 0, err
	}

	return getId, nil
}

// Delete task
func (s *taskService) DeleteTaskById(id int) error {
	err := s.repo.DeleteTask(id)
	if err != nil {
		return err
	}

	return nil
}
