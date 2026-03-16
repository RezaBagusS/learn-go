package service

import (
	"belajar-go/projectAPI/models"
	"belajar-go/projectAPI/repository"
	"errors"
)

type TaskService interface {
	FetchAllTasks() ([]models.Task, error)
	CreateNewTask(task models.Task) (models.Task, error)
}

type taskService struct {
	repo repository.TaskRepository // Depend pada Interface, bukan struct DB langsung
}

func NewTaskService(repo repository.TaskRepository) TaskService {
	return &taskService{repo: repo}
}

func (s *taskService) FetchAllTasks() ([]models.Task, error) {
	return s.repo.GetAllTasks()
}

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
