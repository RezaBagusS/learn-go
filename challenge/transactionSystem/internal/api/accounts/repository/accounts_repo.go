package repository

import (
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	// "strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type AccountRepository interface {
	GetAllAccounts() ([]models.Account, error)
	GetAccountById(id string) (*models.Account, error)
	GetTransactionsByAccountId(id string, trxType string) ([]models.Transaction, error)
	CreateAccount(account models.Account) (string, error)
	UpdateAccount(account models.Account) (string, error)
	DeleteAccount(id string) error
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
	query := `SELECT id, bank_code, account_number, account_holder, balance, created_at, updated_at 
	FROM accounts ORDER BY updated_at desc`

	err := r.db.Select(&accounts, query)
	if err != nil {
		return nil, models.ErrDatabaseIssue // Error Wrapping
	}

	if accounts == nil {
		accounts = []models.Account{}
	}

	return accounts, nil
}

// Get Account By ID
func (r *accountRepository) GetAccountById(id string) (*models.Account, error) {
	var account models.Account

	helper.PrintLog("account", helper.LogPositionRepo, fmt.Sprintf("Mengambil data account by id = %s", id))
	// Catatan: Gunakan $1 jika memakai PostgreSQL, atau ? jika memakai MySQL/SQLite
	query := "SELECT id, bank_code, account_number, account_holder, balance, created_at, updated_at FROM accounts WHERE id = $1"

	err := r.db.Get(&account, query, id)
	if err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			helper.PrintLog("account", helper.LogPositionRepo, "ID Account tidak ditemukan")
			return nil, models.ErrIdNotFound
		}

		helper.PrintLog("account", helper.LogPositionRepo, err.Error())
		return nil, models.ErrDatabaseIssue
	}

	helper.PrintLog("account", helper.LogPositionRepo, fmt.Sprintf("Berhasil mendapatkan akun dengan id = %s -> %+v", id, account))

	return &account, nil
}

// Get Transaction by Account Id
func (r *accountRepository) GetTransactionsByAccountId(id string, trxType string) ([]models.Transaction, error) {
	var transactions []models.Transaction

	helper.PrintLog("account", helper.LogPositionRepo, fmt.Sprintf("Mengambil data transaksi untuk akun dengan id = %s", id))

	// Catatan: Gunakan $1 jika memakai PostgreSQL, atau ? jika memakai MySQL/SQLite
	baseQuery := `SELECT id, from_account_id, from_bank_code, to_account_id, to_bank_code, amount, note, created_at FROM transactions`

	var whereQuery string
	var orderByQuery string = "ORDER BY created_at desc"
	switch {
	case trxType == "all":
		whereQuery = "WHERE from_account_id = $1 OR to_account_id = $2"
	case trxType == "in":
		whereQuery = "WHERE to_account_id = $1"
	case trxType == "out":
		whereQuery = "WHERE from_account_id = $1"
	}

	query := baseQuery + " " + whereQuery + " " + orderByQuery

	var err error
	if trxType == "all" {
		err = r.db.Select(&transactions, query, id, id)
	} else {
		err = r.db.Select(&transactions, query, id)
	}

	if transactions == nil {
		transactions = []models.Transaction{}
	}

	if err != nil {
		helper.PrintLog("account", helper.LogPositionRepo, err.Error())
		return nil, models.ErrDatabaseIssue
	}

	helper.PrintLog("account", helper.LogPositionRepo, fmt.Sprintf("Berhasil mendapatkan seluruh transaksi terkait akun dengan id = %s -> %+v", id, transactions))

	return transactions, nil
}

// Post Create New Account
func (r *accountRepository) CreateAccount(account models.Account) (string, error) {
	var newAccount string
	query := `INSERT INTO accounts (bank_code, account_number, account_holder, balance) VALUES ($1, $2, $3, $4) RETURNING id`

	helper.PrintLog("account", helper.LogPositionRepo, fmt.Sprintf("Menambahkan data akun = %+v", account))

	// Gunakan QueryRowx untuk mengeksekusi insert dan menangkap RETURNING id
	err := r.db.QueryRowx(query, account.BankCode, account.AccountNumber, account.AccountHolder, account.Balance).Scan(&newAccount)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			// [23505] Unique Violation
			if pqErr.Code == "23505" {
				helper.PrintLog("account", helper.LogPositionRepo, models.ErrDuplicateAccount.Error())
				return "", models.ErrDuplicateAccount
			}
		}

		helper.PrintLog("account", helper.LogPositionRepo, models.ErrDatabaseFailed.Error())
		return "", models.ErrDatabaseFailed
	}

	return newAccount, nil
}

// Method Update
func (r *accountRepository) UpdateAccount(account models.Account) (string, error) {
	fields := []string{}
	args := []any{}
	idx := 1

	// Cek AccountNumber
	if account.AccountNumber != "" {
		fields = append(fields, fmt.Sprintf("account_number = $%d", idx))
		args = append(args, account.AccountNumber)
		idx++
	}

	// Cek AccountHolder
	if account.AccountHolder != "" {
		fields = append(fields, fmt.Sprintf("account_holder = $%d", idx))
		args = append(args, account.AccountHolder)
		idx++
	}

	// Perbarui UpdatedAt
	fields = append(fields, fmt.Sprintf("updated_at = $%d", idx))
	args = append(args, time.Now())
	idx++

	// Tambahkan ID sebagai kondisi WHERE
	args = append(args, account.ID)
	query := fmt.Sprintf(
		"UPDATE accounts SET %s WHERE id = $%d",
		strings.Join(fields, ", "),
		idx,
	)

	// Query Execution
	fmt.Printf("Query [account][repo]: %v \n", query)
	fmt.Printf("Args [account][repo]: %v \n", args)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			// [23505] Unique Violation
			if pqErr.Code == "23505" {
				helper.PrintLog("account", helper.LogPositionRepo, models.ErrDuplicateAccount.Error())
				return "", models.ErrDuplicateAccount
			}
		}

		helper.PrintLog("account", helper.LogPositionRepo, err.Error())
		return "", models.ErrDatabaseFailed
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		helper.PrintLog("account", helper.LogPositionRepo, err.Error())
		return "", models.ErrDatabaseIssue
	}

	if rowsAffected == 0 {
		helper.PrintLog("account", helper.LogPositionRepo, models.ErrIdNotFound.Error())
		return "", models.ErrIdNotFound
	}

	return account.ID.String(), nil
}

// Method Delete
func (r *accountRepository) DeleteAccount(id string) error {
	query := `DELETE FROM accounts WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		helper.PrintLog("account", helper.LogPositionRepo, models.ErrDeleteFailed.Error())
		return models.ErrDeleteFailed
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		helper.PrintLog("account", helper.LogPositionRepo, err.Error())
		return models.ErrDatabaseIssue
	}

	if rowsAffected == 0 {
		helper.PrintLog("account", helper.LogPositionRepo, models.ErrIdNotFound.Error())
		return models.ErrIdNotFound
	}

	return nil
}
