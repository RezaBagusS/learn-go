package repository

import (
	"belajar-go/challenge/transactionSystem/internal/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type TransactionRepository interface {
	GetAllTransactions() ([]models.Transaction, error)
	GetTransactionById(id string) (*models.Transaction, error)
	CreateTransaction(trx models.Transaction) (string, error)
	GetSummary(date time.Time) ([]models.Transaction, error)
	// UpdateBank(bank models.Bank) (string, error)
	// DeleteBank(bankCode string) error
}

type transactionRepository struct {
	db *sqlx.DB
}

func NewtransactionRepository(db *sqlx.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

// Get All
func (r *transactionRepository) GetAllTransactions() ([]models.Transaction, error) {
	var transactions []models.Transaction
	query := `SELECT id, from_account_id, from_bank_code, to_account_id, to_bank_code, amount, note, created_at 
	FROM transactions
	ORDER BY created_at desc
	`

	err := r.db.Select(&transactions, query)
	if err != nil {
		helper.PrintLog("transaction", helper.LogPositionRepo, err.Error())
		return nil, models.ErrDatabaseIssue
	}

	if transactions == nil {
		transactions = []models.Transaction{}
	}

	return transactions, nil
}

// Get Transaction on today
func (r *transactionRepository) GetSummary(date time.Time) ([]models.Transaction, error) {
	transactions := make([]models.Transaction, 0)

	query := `SELECT id, from_account_id, from_bank_code, to_account_id, to_bank_code, amount, note, created_at 
	FROM transactions
	WHERE DATE(created_at) = Date($1)
	ORDER BY created_at desc
	`
	err := r.db.Select(&transactions, query, date)
	if err != nil {
		helper.PrintLog("transaction", helper.LogPositionRepo, err.Error())
		return nil, models.ErrDatabaseFailed
	}

	if transactions == nil {
		transactions = []models.Transaction{}
	}

	return transactions, nil
}

// Get Transaction By ID
func (r *transactionRepository) GetTransactionById(id string) (*models.Transaction, error) {
	var transaction models.Transaction

	helper.PrintLog("transaction", helper.LogPositionRepo, fmt.Sprintf("Mengambil data transaction by id = %s", id))
	// Catatan: Gunakan $1 jika memakai PostgreSQL, atau ? jika memakai MySQL/SQLite
	query := `SELECT id, from_account_id, from_bank_code, to_account_id, to_bank_code, amount, note, created_at FROM transactions WHERE id = $1`

	err := r.db.Get(&transaction, query, id)
	if err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			helper.PrintLog("transaction", helper.LogPositionRepo, models.ErrIdNotFound.Error())
			return nil, models.ErrIdNotFound
		}

		helper.PrintLog("transaction", helper.LogPositionRepo, err.Error())
		return nil, models.ErrDatabaseIssue
	}

	helper.PrintLog("transaction", helper.LogPositionRepo, fmt.Sprintf("Berhasil mendapatkan transaksi dengan id = %s -> %+v", id, transaction))

	return &transaction, nil
}

// Post Create transaction
func (r *transactionRepository) CreateTransaction(trx models.Transaction) (string, error) {

	// START TRX
	tx, err := r.db.Beginx()
	if err != nil {
		helper.PrintLog("transaction", helper.LogPositionRepo, models.ErrDatabaseTrx.Error())
		return "", models.ErrDatabaseTrx
	}

	// ROLLBACK
	defer tx.Rollback()

	// Check Sender
	var senderBalance int64
	var fromBankCode string

	err = tx.QueryRow("SELECT balance, bank_code FROM accounts WHERE id = $1 FOR UPDATE", trx.FromAccountID).
		Scan(&senderBalance, &fromBankCode)
	if err != nil {
		helper.PrintLog("transaction", helper.LogPositionRepo, models.ErrInvalidTrxAccount.Error())
		return "", models.ErrInvalidTrxAccount
	}

	if senderBalance < trx.Amount {
		return "", models.ErrLogicBalanceTrx
	}

	// Check Receiver
	var receiverID string
	var toBankCode string

	err = tx.QueryRow("SELECT id, bank_code FROM accounts WHERE id = $1 FOR UPDATE", trx.ToAccountID).
		Scan(&receiverID, &toBankCode)
	if err != nil {
		helper.PrintLog("transaction", helper.LogPositionRepo, models.ErrInvalidTrxAccount.Error())
		return "", models.ErrInvalidTrxAccount
	}

	// Set Bank Code
	trx.FromBankCode = fromBankCode
	trx.ToBankCode = toBankCode

	// Sender Mutation
	_, err = tx.Exec("UPDATE accounts SET balance = balance - $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", trx.Amount, trx.FromAccountID)
	if err != nil {
		helper.PrintLog("transaction", helper.LogPositionRepo, models.ErrLogicMutationTrx.Error())
		return "", models.ErrLogicMutationTrx
	}

	// Receiver Mutation
	_, err = tx.Exec("UPDATE accounts SET balance = balance + $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", trx.Amount, trx.ToAccountID)
	if err != nil {
		helper.PrintLog("transaction", helper.LogPositionRepo, models.ErrLogicMutationTrx.Error())
		return "", models.ErrLogicMutationTrx
	}

	// Push History Trx
	var insertedID string
	queryInsert := `
		INSERT INTO transactions (from_account_id, from_bank_code, to_account_id, to_bank_code, amount, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err = tx.QueryRowx(queryInsert,
		trx.FromAccountID, trx.FromBankCode,
		trx.ToAccountID, trx.ToBankCode,
		trx.Amount, trx.Note,
	).Scan(&insertedID)

	if err != nil {
		helper.PrintLog("transaction", helper.LogPositionRepo, err.Error())
		return "", models.ErrDatabaseFailed
	}

	// Commit
	if err = tx.Commit(); err != nil {
		helper.PrintLog("transaction", helper.LogPositionRepo, models.ErrLogicCommitTrx.Error())
		return "", models.ErrLogicCommitTrx
	}

	return insertedID, nil
}
