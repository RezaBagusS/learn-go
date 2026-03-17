package repository

import (
	"belajar-go/projectAPI/internal/models"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// 1. Buat Kontrak (Interface)
type TaskRepository interface {
	GetAllTasks() ([]models.Task, error)
	CreateTask(task models.Task) (int, error)
	UpdateTask(task models.Task) (int, error)
	DeleteTask(id int) error
}

// 2. Struct implementasi
type taskRepository struct {
	db *sqlx.DB
}

// 3. Constructor (Dependency Injection)
func NewTaskRepository(db *sqlx.DB) TaskRepository {
	return &taskRepository{db: db}
}

// 4. Get All
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

// Method Create
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

// Method Update
func (r *taskRepository) UpdateTask(task models.Task) (int, error) {
	fields := []string{}
	args := []any{}
	idx := 1

	// Cek title
	if task.Title != "" {
		fields = append(fields, fmt.Sprintf("title = $%d", idx))
		args = append(args, task.Title)
		idx++
	}

	// Cek Desc
	if task.Description != "" {
		fields = append(fields, fmt.Sprintf("description = $%d", idx))
		args = append(args, task.Description)
		idx++
	}

	// selalu diupdate jika ada di payload
	fields = append(fields, fmt.Sprintf("is_completed = $%d", idx))
	args = append(args, task.IsCompleted)
	idx++

	// Jika tidak ada field yang diupdate
	if len(fields) == 0 {
		return 0, fmt.Errorf("tidak ada field yang diupdate")
	}

	// Tambahkan ID sebagai kondisi WHERE
	args = append(args, task.ID)
	query := fmt.Sprintf(
		"UPDATE tasks SET %s WHERE id = $%d",
		strings.Join(fields, ", "),
		idx,
	)

	// Query Execution
	fmt.Printf("Query [task][repo]: %v \n", query)
	fmt.Printf("Args [task][repo]: %v \n", args)
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("gagal update task: %w", err)
	}

	// Cek apakah data dengan ID tersebut ditemukan
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("gagal membaca rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return 0, fmt.Errorf("task dengan id %d tidak ditemukan", task.ID)
	}

	return task.ID, nil
}

// Method Delete
func (r *taskRepository) DeleteTask(id int) error {
	query := `DELETE FROM tasks WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("gagal menghapus task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal membaca rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task dengan id %d tidak ditemukan", id)
	}

	return nil
}
