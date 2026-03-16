package repository

import (
	"belajar-go/projectAPI/models"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// 1. Buat Kontrak (Interface)
type TaskRepository interface {
	GetAllTasks() ([]models.Task, error)
	CreateTask(task models.Task) (int, error)
}

// 2. Struct implementasi
type taskRepository struct {
	db *sqlx.DB
}

// 3. Constructor (Dependency Injection)
func NewTaskRepository(db *sqlx.DB) TaskRepository {
	return &taskRepository{db: db}
}

// 4. Method Get All
func (r *taskRepository) GetAllTasks() ([]models.Task, error) {
	var tasks []models.Task
	query := "SELECT id, title, description, is_completed FROM tasks"

	// sqlx.Select otomatis memetakan hasil query banyak baris ke dalam slice struct
	err := r.db.Select(&tasks, query)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data dari db: %w", err) // Error Wrapping
	}

	// Jika database kosong, return slice kosong agar API mengembalikan [] bukan null
	if tasks == nil {
		tasks = []models.Task{}
	}

	return tasks, nil
}

// 5. Method Create
func (r *taskRepository) CreateTask(task models.Task) (int, error) {
	var newID int
	query := `INSERT INTO tasks (title, description) VALUES ($1, $2) RETURNING id`

	// Gunakan QueryRowx untuk mengeksekusi insert dan menangkap RETURNING id
	err := r.db.QueryRowx(query, task.Title, task.Description).Scan(&newID)
	if err != nil {
		return 0, fmt.Errorf("gagal insert task ke db: %w", err)
	}

	return newID, nil
}
