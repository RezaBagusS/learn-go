package repository

import (
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"belajar-go/challenge/transactionSystem/observability/metrics"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type TransactionRepository interface {
	GetAllTransactions(ctx context.Context) ([]models.Transaction, error)
	GetTransactionById(ctx context.Context, id string) (*models.Transaction, error)
	CreateTransaction(ctx context.Context, trx models.Transaction) (string, error)
	GetSummary(ctx context.Context, date time.Time) ([]models.Transaction, error)
	// UpdateBank(bank models.Bank) (string, error)
	// DeleteBank(bankCode string) error
}

type transactionRepository struct {
	db *sqlx.DB
}

func NewtransactionRepository(db *sqlx.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

const repoTransaction = "transaction"

// Get All
func (r *transactionRepository) GetAllTransactions(ctx context.Context) ([]models.Transaction, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.GetAll")
	defer span.End()
	operation := "select"

	query := `SELECT id, from_account_id, from_bank_code, to_account_id, to_bank_code, amount, note, created_at 
	FROM transactions
	ORDER BY created_at desc`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "transactions"),
	)

	logger.Info("executing query",
		zap.String("query", "SELECT transactions"),
	)

	var transactions []models.Transaction

	dbStart := time.Now()
	err := r.db.SelectContext(ctx, &transactions, query)
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		logger.Error("query failed", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()

		return nil, models.ErrDatabaseIssue
	}

	if transactions == nil {
		transactions = []models.Transaction{}
	}

	span.SetAttributes(attribute.Int("db.result.count", len(transactions)))

	logger.Info("query success",
		zap.Int("rows", len(transactions)),
	)

	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return transactions, nil
}

// Get Transaction on today
func (r *transactionRepository) GetSummary(ctx context.Context, date time.Time) ([]models.Transaction, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.GetSummary")
	defer span.End()
	operation := "select"

	query := `SELECT id, from_account_id, from_bank_code, to_account_id, to_bank_code, amount, note, created_at 
	FROM transactions
	WHERE DATE(created_at) = Date($1)
	ORDER BY created_at desc`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "transactions"),
		attribute.String("db.query.date", date.Format("2006-01-02")),
	)

	logger.Info("executing query",
		zap.String("query", "SELECT transactions by date"),
		zap.String("date", date.Format("2006-01-02")),
	)

	transactions := make([]models.Transaction, 0)

	dbStart := time.Now()
	err := r.db.SelectContext(ctx, &transactions, query, date)
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		logger.Error("query failed", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()

		return nil, models.ErrDatabaseFailed
	}

	if transactions == nil {
		transactions = []models.Transaction{}
	}

	span.SetAttributes(attribute.Int("db.result.count", len(transactions)))

	logger.Info("query success",
		zap.String("date", date.Format("2006-01-02")),
		zap.Int("rows", len(transactions)),
	)

	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return transactions, nil
}

// Get Transaction By ID
func (r *transactionRepository) GetTransactionById(ctx context.Context, id string) (*models.Transaction, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.GetById")
	defer span.End()
	operation := "select"

	query := `SELECT id, from_account_id, from_bank_code, to_account_id, to_bank_code, amount, note, created_at FROM transactions WHERE id = $1`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "transactions"),
		attribute.String("db.query.id", id),
	)

	logger.Info("executing query",
		zap.String("query", "SELECT transaction by id"),
		zap.String("id", id),
	)

	var transaction models.Transaction

	dbStart := time.Now()
	err := r.db.GetContext(ctx, &transaction, query, id)
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			span.RecordError(models.ErrIdNotFound)
			span.SetStatus(codes.Error, models.ErrIdNotFound.Error())

			logger.Error("transaction not found", zap.String("id", id))
			metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()

			return nil, models.ErrIdNotFound
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		logger.Error("query failed", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()

		return nil, models.ErrDatabaseIssue
	}

	span.SetAttributes(attribute.String("db.result.id", id))

	logger.Info("query success",
		zap.String("id", id),
	)

	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return &transaction, nil
}

// Post Create transaction
func (r *transactionRepository) CreateTransaction(ctx context.Context, trx models.Transaction) (string, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.Create")
	defer span.End()
	operation := "insert"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "transactions"),
	)

	// START TRX
	logger.Info("starting database transaction")
	tx, err := r.db.Beginx()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error(models.ErrDatabaseTrx.Error(), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", models.ErrDatabaseTrx
	}

	// ROLLBACK
	defer tx.Rollback()

	// Check Sender
	logger.Info("checking sender account", zap.String("from_account_id", trx.FromAccountID))
	var senderBalance int64
	var fromBankCode string

	err = tx.QueryRow("SELECT balance, bank_code FROM accounts WHERE id = $1 FOR UPDATE", trx.FromAccountID).
		Scan(&senderBalance, &fromBankCode)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error(models.ErrInvalidTrxAccount.Error(), zap.String("from_account_id", trx.FromAccountID), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", models.ErrInvalidTrxAccount
	}

	if senderBalance < trx.Amount {
		span.RecordError(models.ErrLogicBalanceTrx)
		span.SetStatus(codes.Error, models.ErrLogicBalanceTrx.Error())
		logger.Error(models.ErrLogicBalanceTrx.Error(),
			zap.Int64("balance", senderBalance),
			zap.Int64("amount", trx.Amount),
		)
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", models.ErrLogicBalanceTrx
	}

	// Check Receiver
	logger.Info("checking receiver account", zap.String("to_account_id", trx.ToAccountID))
	var receiverID string
	var toBankCode string

	err = tx.QueryRow("SELECT id, bank_code FROM accounts WHERE id = $1 FOR UPDATE", trx.ToAccountID).
		Scan(&receiverID, &toBankCode)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error(models.ErrInvalidTrxAccount.Error(), zap.String("to_account_id", trx.ToAccountID), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", models.ErrInvalidTrxAccount
	}

	// Set Bank Code
	trx.FromBankCode = fromBankCode
	trx.ToBankCode = toBankCode

	// Sender Mutation
	logger.Info("mutating sender balance", zap.String("from_account_id", trx.FromAccountID))
	_, err = tx.Exec("UPDATE accounts SET balance = balance - $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", trx.Amount, trx.FromAccountID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error(models.ErrLogicMutationTrx.Error(), zap.String("from_account_id", trx.FromAccountID), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", models.ErrLogicMutationTrx
	}

	// Receiver Mutation
	logger.Info("mutating receiver balance", zap.String("to_account_id", trx.ToAccountID))
	_, err = tx.Exec("UPDATE accounts SET balance = balance + $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", trx.Amount, trx.ToAccountID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error(models.ErrLogicMutationTrx.Error(), zap.String("to_account_id", trx.ToAccountID), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", models.ErrLogicMutationTrx
	}

	// Push History Trx
	logger.Info("inserting transaction history")
	var insertedID string
	queryInsert := `
		INSERT INTO transactions (from_account_id, from_bank_code, to_account_id, to_bank_code, amount, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	dbStart := time.Now()
	err = tx.QueryRowx(queryInsert,
		trx.FromAccountID, trx.FromBankCode,
		trx.ToAccountID, trx.ToBankCode,
		trx.Amount, trx.Note,
	).Scan(&insertedID)
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error(models.ErrDatabaseFailed.Error(), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", models.ErrDatabaseFailed
	}

	// Commit
	logger.Info("committing database transaction")
	if err = tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error(models.ErrLogicCommitTrx.Error(), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", models.ErrLogicCommitTrx
	}

	span.SetAttributes(
		attribute.String("db.result.id", insertedID),
	)

	logger.Info("query success",
		zap.String("db.result.id", insertedID),
	)

	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return insertedID, nil
}
