package repository

import (
	"belajar-go/challenge/transactionSystem/internal/models"
	"fmt"

	// "strings"

	"github.com/jmoiron/sqlx"
)

type AccountRepository interface {
	GetAllAccounts() ([]models.Account, error)
	// GetAccountById() ([]models.Account, error)
	// CreateAccount(account models.Account) (int, error)
	// UpdateAccount(account models.Account) (int, error)
	// DeleteAccount(id int) error
}

type accountRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) AccountRepository {
	return &accountRepository{db: db}
}

// 4. Get All
func (r *accountRepository) GetAllAccounts() ([]models.Account, error) {
	var accounts []models.Account
	query := "SELECT id, bank_code, account_number, account_holder, balance, created_at, updated_at FROM accounts"

	err := r.db.Select(&accounts, query)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data dari db: %w", err) // Error Wrapping
	}

	if accounts == nil {
		accounts = []models.Account{}
	}

	return accounts, nil
}

// // Method Create
// func (r *accountRepository) CreateTask(task models.Account) (int, error) {
// 	var newID int
// 	query := `INSERT INTO tasks (title, description) VALUES ($1, $2) RETURNING id`

// 	// Gunakan QueryRowx untuk mengeksekusi insert dan menangkap RETURNING id
// 	err := r.db.QueryRowx(query, task.Title, task.Description).Scan(&newID)
// 	if err != nil {
// 		return 0, fmt.Errorf("gagal insert task ke db: %w", err)
// 	}

// 	return newID, nil
// }

// // Method Update
// func (r *accountRepository) UpdateTask(task models.Account) (int, error) {
// 	fields := []string{}
// 	args := []any{}
// 	idx := 1

// 	// Cek title
// 	if task.Title != "" {
// 		fields = append(fields, fmt.Sprintf("title = $%d", idx))
// 		args = append(args, task.Title)
// 		idx++
// 	}

// 	// Cek Desc
// 	if task.Description != "" {
// 		fields = append(fields, fmt.Sprintf("description = $%d", idx))
// 		args = append(args, task.Description)
// 		idx++
// 	}

// 	// selalu diupdate jika ada di payload
// 	fields = append(fields, fmt.Sprintf("is_completed = $%d", idx))
// 	args = append(args, task.IsCompleted)
// 	idx++

// 	// Jika tidak ada field yang diupdate
// 	if len(fields) == 0 {
// 		return 0, fmt.Errorf("tidak ada field yang diupdate")
// 	}

// 	// Tambahkan ID sebagai kondisi WHERE
// 	args = append(args, task.ID)
// 	query := fmt.Sprintf(
// 		"UPDATE tasks SET %s WHERE id = $%d",
// 		strings.Join(fields, ", "),
// 		idx,
// 	)

// 	// Query Execution
// 	fmt.Printf("Query [task][repo]: %v \n", query)
// 	fmt.Printf("Args [task][repo]: %v \n", args)
// 	result, err := r.db.Exec(query, args...)
// 	if err != nil {
// 		return 0, fmt.Errorf("gagal update task: %w", err)
// 	}

// 	// Cek apakah data dengan ID tersebut ditemukan
// 	rowsAffected, err := result.RowsAffected()
// 	if err != nil {
// 		return 0, fmt.Errorf("gagal membaca rows affected: %w", err)
// 	}

// 	if rowsAffected == 0 {
// 		return 0, fmt.Errorf("task dengan id %d tidak ditemukan", task.ID)
// 	}

// 	return task.ID, nil
// }

// // Method Delete
// func (r *accountRepository) DeleteTask(id int) error {
// 	query := `DELETE FROM tasks WHERE id = $1`

// 	result, err := r.db.Exec(query, id)
// 	if err != nil {
// 		return fmt.Errorf("gagal menghapus task: %w", err)
// 	}

// 	rowsAffected, err := result.RowsAffected()
// 	if err != nil {
// 		return fmt.Errorf("gagal membaca rows affected: %w", err)
// 	}

// 	if rowsAffected == 0 {
// 		return fmt.Errorf("task dengan id %d tidak ditemukan", id)
// 	}

// 	return nil
// }
