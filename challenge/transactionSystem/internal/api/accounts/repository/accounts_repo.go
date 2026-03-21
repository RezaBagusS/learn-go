package repository

import (
	"belajar-go/challenge/transactionSystem/internal/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"fmt"

	// "strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type AccountRepository interface {
	GetAllAccounts() ([]models.Account, error)
	GetAccountById(id string) (models.Account, error)
	CreateAccount(account models.Account) (string, error)
	// UpdateAccount(account models.Account) (int, error)
	// DeleteAccount(id int) error
}

type accountRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) AccountRepository {
	return &accountRepository{db: db}
}

// Get All
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

// Get Account By ID
func (r *accountRepository) GetAccountById(id string) (models.Account, error) {
	var accounts []models.Account

	helper.PrintLog("account", helper.LogPositionRepo, fmt.Sprintf("Mengambil data account by id = %s", id))
	// Catatan: Gunakan $1 jika memakai PostgreSQL, atau ? jika memakai MySQL/SQLite
	query := "SELECT id, bank_code, account_number, account_holder, balance, created_at, updated_at FROM accounts WHERE id = $1"

	err := r.db.Select(&accounts, query, id)
	if err != nil {
		helper.PrintLog("account", helper.LogPositionRepo, "gagal mengambil data dari db")
		return models.Account{}, fmt.Errorf("gagal mengambil data dari db: %w", err)
	}

	if len(accounts) == 0 {
		helper.PrintLog("account", helper.LogPositionRepo, "ID Account tidak ditemukan")
		return models.Account{}, fmt.Errorf("akun dengan ID %s tidak ditemukan", id)
	}

	if len(accounts) > 1 {
		helper.PrintLog("account", helper.LogPositionRepo, "Terdapat lebih dari 1 akun")
		return models.Account{}, fmt.Errorf("terdapat lebih dari 1 akun dengan ID %s", id)
	}

	account := accounts[0]
	helper.PrintLog("account", helper.LogPositionRepo, fmt.Sprintf("Berhasil mendapatkan akun dengan id = %s -> %+v", id, account))

	return account, nil
}

// Post Create New Account
func (r *accountRepository) CreateAccount(account models.Account) (string, error) {
	var newAccount string
	query := `INSERT INTO accounts (bank_code, account_number, account_holder, balance) VALUES ($1, $2, $3, $4) RETURNING id`

	// Gunakan QueryRowx untuk mengeksekusi insert dan menangkap RETURNING id
	err := r.db.QueryRowx(query, account.BankCode, account.AccountNumber, account.AccountHolder, account.Balance).Scan(&newAccount)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			// [23505] Unique Violation
			if pqErr.Code == "23505" {
				return "", fmt.Errorf("nomor rekening %s sudah terdaftar untuk bank %s", account.AccountNumber, account.BankCode)
			}
		}

		return "", fmt.Errorf("gagal insert account ke db: %w", err)
	}

	return newAccount, nil
}

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
