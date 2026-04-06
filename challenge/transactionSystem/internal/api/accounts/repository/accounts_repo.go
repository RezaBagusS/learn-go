package repository

import (
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"belajar-go/challenge/transactionSystem/observability/metrics"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	// "strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type AccountRepository interface {
	GetAllAccounts(ctx context.Context) ([]models.Account, error)
	GetAccountById(ctx context.Context, id string) (*models.Account, error)
	GetTransactionsByAccountId(ctx context.Context, id string, trxType string) ([]models.Transaction, error)
	CreateAccount(ctx context.Context, account models.Account) (string, error)
	UpdateAccount(ctx context.Context, account models.Account) (string, error)
	DeleteAccount(ctx context.Context, id string) error
}

type accountRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) AccountRepository {
	return &accountRepository{db: db}
}

const repoAccount = "account"

// Get All
func (r *accountRepository) GetAllAccounts(ctx context.Context) ([]models.Account, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankRepo.GetAll")
	defer span.End()
	operation := "select"

	query := `SELECT id, bank_code, account_number, account_holder, balance, created_at, updated_at 
	FROM accounts ORDER BY updated_at desc`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "accounts"),
	)

	logger.Info("executing query",
		zap.String("query", "SELECT accounts"),
	)

	var accounts []models.Account

	dbStart := time.Now()
	err := r.db.SelectContext(ctx, &accounts, query)
	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		logger.Error("query failed", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()

		return nil, models.ErrDatabaseIssue
	}

	span.SetAttributes(attribute.Int("db.result.count", len(accounts)))

	logger.Info("query success",
		zap.Int("rows", len(accounts)),
	)

	return accounts, nil
}

// Get Account By ID
func (r *accountRepository) GetAccountById(ctx context.Context, id string) (*models.Account, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.GetById")
	defer span.End()
	operation := "select_by_id"

	query := "SELECT id, bank_code, account_number, account_holder, balance, created_at, updated_at FROM accounts WHERE id = $1"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "accounts"),
	)

	logger.Info("executing query",
		zap.String("query", "SELECT accounts"),
		zap.String("account.id", id),
	)

	var account models.Account

	dbStart := time.Now()
	err := r.db.GetContext(ctx, &account, query, id)
	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()

		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(models.ErrIdNotFound.Error(), zap.Error(err))
			return nil, models.ErrIdNotFound
		}

		logger.Error("query failed", zap.Error(err))
		return nil, models.ErrDatabaseIssue
	}

	span.SetAttributes(
		attribute.String("db.result.id", account.ID.String()),
	)

	logger.Info("query success",
		zap.String("db.result.id", account.ID.String()),
	)

	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return &account, nil
}

// Get Transaction by Account Id
func (r *accountRepository) GetTransactionsByAccountId(ctx context.Context, id string, trxType string) ([]models.Transaction, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.GetTransactionsByAccountId")
	defer span.End()
	operation := "select_transactions_by_account"

	baseQuery := `SELECT id, from_account_id, from_bank_code, to_account_id, to_bank_code, amount, note, created_at FROM transactions`

	var whereQuery string
	orderByQuery := "ORDER BY created_at desc"
	switch trxType {
	case "all":
		whereQuery = "WHERE from_account_id = $1 OR to_account_id = $2"
	case "in":
		whereQuery = "WHERE to_account_id = $1"
	case "out":
		whereQuery = "WHERE from_account_id = $1"
	}

	query := baseQuery + " " + whereQuery + " " + orderByQuery

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "transactions"),
		attribute.String("db.filter.account_id", id),
		attribute.String("db.filter.trx_type", trxType),
	)

	logTrxType := trxType
	if logTrxType == "" {
		logTrxType = "all"
	}

	logger.Info("executing query",
		zap.String("query", "SELECT transactions"),
		zap.String("account.id", id),
		zap.String("trx.type", logTrxType),
	)

	var transactions []models.Transaction

	var err error
	dbStart := time.Now()
	if trxType == "all" {
		err = r.db.SelectContext(ctx, &transactions, query, id, id)
	} else {
		err = r.db.SelectContext(ctx, &transactions, query, id)
	}
	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()

		logger.Error("query failed", zap.Error(err))
		return nil, models.ErrDatabaseIssue
	}

	if transactions == nil {
		transactions = []models.Transaction{}
	}

	span.SetAttributes(
		attribute.Int("db.result.count", len(transactions)),
	)

	logger.Info("query success",
		zap.String("account.id", id),
		zap.String("trx.type", logTrxType),
		zap.Int("count", len(transactions)),
	)

	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return transactions, nil
}

// Post Create New Account
func (r *accountRepository) CreateAccount(ctx context.Context, account models.Account) (string, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.Create")
	defer span.End()
	operation := "insert"

	query := `INSERT INTO accounts (bank_code, account_number, account_holder, balance) VALUES ($1, $2, $3, $4) RETURNING id`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "accounts"),
	)

	logger.Info("executing query",
		zap.String("query", "INSERT accounts"),
	)

	var newId string
	dbStart := time.Now()
	err := r.db.QueryRowxContext(ctx, query, account.BankCode, account.AccountNumber, account.AccountHolder, account.Balance).Scan(&newId)
	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()

		if pqErr, ok := err.(*pq.Error); ok {
			// [23505] Unique Violation
			if pqErr.Code == "23505" {
				logger.Error(models.ErrDuplicateAccount.Error(), zap.Error(err))
				return "", models.ErrDuplicateAccount
			}
		}

		logger.Error(models.ErrDatabaseFailed.Error(), zap.Error(err))
		return "", models.ErrDatabaseFailed
	}

	span.SetAttributes(
		attribute.String("db.result.id", newId),
	)

	logger.Info("query success",
		zap.String("db.result.id", newId),
	)

	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return newId, nil
}

// Method Update
func (r *accountRepository) UpdateAccount(ctx context.Context, account models.Account) (string, error) {
	fields := []string{}
	args := []any{}
	idx := 1
	operation := "update"

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.Update")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.table", "accounts"),
	)

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

	logger.Info("executing query",
		zap.String("query", "UPDATE accounts"),
	)

	dbStart := time.Now()
	result, err := r.db.ExecContext(ctx, query, args...) // ✅ ExecContext, bukan Exec
	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()

		if pqErr, ok := err.(*pq.Error); ok {
			// [23505] Unique Violation
			if pqErr.Code == "23505" {
				logger.Error(models.ErrDuplicateAccount.Error(), zap.Error(err))
				return "", models.ErrDuplicateAccount
			}
		}

		logger.Error(models.ErrDatabaseFailed.Error(), zap.Error(err))
		return "", models.ErrDatabaseFailed
	}

	// Cek apakah data dengan ID tersebut ditemukan
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		logger.Error(models.ErrDatabaseIssue.Error(), zap.Error(err))
		return "", models.ErrDatabaseIssue
	}

	if rowsAffected == 0 {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		logger.Error(models.ErrIdNotFound.Error())
		return "", models.ErrIdNotFound
	}

	span.SetAttributes(
		attribute.String("db.result.id", account.ID.String()),
	)

	logger.Info("query success",
		zap.String("db.result.id", account.ID.String()),
	)

	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return account.ID.String(), nil
}

// Method Delete
func (r *accountRepository) DeleteAccount(ctx context.Context, id string) error {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.Delete")
	defer span.End()
	operation := "delete"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.table", "accounts"),
	)

	query := `DELETE FROM accounts WHERE id = $1`

	logger.Info("executing query",
		zap.String("query", "DELETE accounts"),
	)

	dbStart := time.Now()
	result, err := r.db.ExecContext(ctx, query, id) // ✅ ExecContext, bukan Exec
	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()

		logger.Error(models.ErrDatabaseFailed.Error(), zap.Error(err))
		return models.ErrDeleteFailed
	}

	// Cek apakah data dengan ID tersebut ditemukan
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		logger.Error(models.ErrDatabaseIssue.Error(), zap.Error(err))
		return models.ErrDatabaseIssue
	}

	if rowsAffected == 0 {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		logger.Error(models.ErrIdNotFound.Error())
		return models.ErrIdNotFound
	}

	span.SetAttributes(
		attribute.String("db.delete.id", id),
	)

	logger.Info("query success",
		zap.String("db.delete.id", id),
	)

	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return nil
}
