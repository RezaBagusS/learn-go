package repository

import (
	"belajar-go/challenge/transactionSystem/helper"
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
	GetAllTransactions(ctx context.Context) ([]models.Transaction, *models.SnapDetail)
	GetTransactionById(ctx context.Context, id string) (*models.Transaction, *models.SnapDetail)
	// CreateTransaction(ctx context.Context, trx models.Transaction) (string, error)
	GetSummary(ctx context.Context, date time.Time) ([]models.Transaction, *models.SnapDetail)
	TransferIntraBank(ctx context.Context, trx models.Transaction) (string, *models.SnapDetail)
}

type transactionRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewtransactionRepository(db *sqlx.DB) TransactionRepository {
	logger := helper.Log

	return &transactionRepository{
		db:     db,
		logger: logger,
	}
}

const repoTransaction = "transaction"

// Get All
func (r *transactionRepository) GetAllTransactions(ctx context.Context) ([]models.Transaction, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.GetAll")
	defer span.End()
	operation := "select"

	// Sesuaikan kolom dengan schema — hapus from_bank_code & to_bank_code
	query := `
		SELECT 
			id,
			from_account_id,
			to_account_id,
			amount,
			currency,
			reference_no,
			partner_reference_no,
			external_id,
			status,
			note,
			additional_info,
			created_at,
			updated_at
		FROM transactions
		ORDER BY created_at DESC`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "transactions"),
		attribute.String("db.statement", query),
	)

	r.logger.Info("executing query",
		zap.String("operation", "SELECT"),
		zap.String("table", "transactions"),
	)

	var transactions []models.Transaction

	dbStart := time.Now()
	err := r.db.SelectContext(ctx, &transactions, query)
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("query failed",
			zap.Error(err),
			zap.String("operation", operation),
		)
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return nil, &models.SnapInternalError
	}

	// Hindari nil slice — kembalikan slice kosong jika tidak ada data
	if transactions == nil {
		transactions = []models.Transaction{}
	}

	span.SetStatus(codes.Ok, "query success")
	span.SetAttributes(attribute.Int("db.result.count", len(transactions)))

	r.logger.Info("query success",
		zap.Int("rows", len(transactions)),
		zap.String("operation", operation),
	)

	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return transactions, nil
}

// Get Transaction on today
func (r *transactionRepository) GetSummary(ctx context.Context, date time.Time) ([]models.Transaction, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.GetSummary")
	defer span.End()
	operation := "select"

	// Gunakan timezone-aware comparison untuk TIMESTAMPTZ
	query := `
		SELECT 
			id,
			from_account_id,
			to_account_id,
			amount,
			currency,
			reference_no,
			partner_reference_no,
			external_id,
			status,
			note,
			additional_info,
			created_at,
			updated_at
		FROM transactions
		WHERE created_at >= $1::date 
		  AND created_at < ($1::date + INTERVAL '1 day')
		ORDER BY created_at DESC`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "transactions"),
		attribute.String("db.query.date", date.Format("2006-01-02")),
	)

	r.logger.Info("executing query",
		zap.String("operation", "SELECT"),
		zap.String("table", "transactions"),
		zap.String("date", date.Format("2006-01-02")),
	)

	transactions := make([]models.Transaction, 0)

	dbStart := time.Now()
	err := r.db.SelectContext(ctx, &transactions, query, date.Format("2006-01-02"))
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("query failed",
			zap.Error(err),
			zap.String("date", date.Format("2006-01-02")),
		)
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return nil, &models.SnapInternalError
	}

	span.SetStatus(codes.Ok, "query success")
	span.SetAttributes(attribute.Int("db.result.count", len(transactions)))

	r.logger.Info("query success",
		zap.String("date", date.Format("2006-01-02")),
		zap.Int("rows", len(transactions)),
	)

	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return transactions, nil
}

// Get Transaction By ID
func (r *transactionRepository) GetTransactionById(ctx context.Context, id string) (*models.Transaction, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.GetById")
	defer span.End()
	operation := "select"

	query := `
		SELECT 
			id,
			from_account_id,
			to_account_id,
			amount,
			currency,
			reference_no,
			partner_reference_no,
			external_id,
			status,
			note,
			additional_info,
			created_at,
			updated_at
		FROM transactions 
		WHERE id = $1`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "transactions"),
		attribute.String("db.query.id", id),
	)

	r.logger.Info("executing query",
		zap.String("operation", "SELECT"),
		zap.String("table", "transactions"),
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
			r.logger.Error("transaction not found", zap.String("id", id))
			metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "not_found").Inc()
			return nil, &models.SnapTrxNotFound
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("query failed",
			zap.Error(err),
			zap.String("id", id),
		)
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return nil, &models.SnapInternalError
	}

	span.SetStatus(codes.Ok, "query success")
	span.SetAttributes(attribute.String("db.result.id", id))

	r.logger.Info("query success", zap.String("id", id))
	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return &transaction, nil
}

func (r *transactionRepository) TransferIntraBank(ctx context.Context, trx models.Transaction) (string, *models.SnapDetail) {
	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.TransferIntraBank")
	defer span.End()
	operation := "insert_snap"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "transactions"),
		attribute.String("snap.partner_ref", trx.PartnerRefNo),
	)

	r.logger.Info("starting database transaction for SNAP transfer")
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error(models.ErrDatabaseTrx.Error(), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", &models.SnapInternalError
	}
	defer tx.Rollback()

	// LOCK DENGAN URUTAN KONSISTEN — cegah deadlock
	// Selalu lock account_id yang lebih kecil (secara string) duluan
	firstLock, secondLock := trx.FromAccountID, trx.ToAccountID
	if firstLock > secondLock {
		firstLock, secondLock = secondLock, firstLock
	}

	r.logger.Info("locking accounts in consistent order",
		zap.String("first_lock", firstLock),
		zap.String("second_lock", secondLock),
	)

	// Lock account pertama
	var dummy string
	err = tx.QueryRowContext(ctx,
		"SELECT id FROM accounts WHERE id = $1 FOR UPDATE", firstLock,
	).Scan(&dummy)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "lock first account failed")
		r.logger.Error("failed to lock first account", zap.String("account_id", firstLock), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", &models.SnapInvalidAccount
	}

	// Lock account kedua
	err = tx.QueryRowContext(ctx,
		"SELECT id FROM accounts WHERE id = $1 FOR UPDATE", secondLock,
	).Scan(&dummy)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "lock second account failed")
		r.logger.Error("failed to lock second account", zap.String("account_id", secondLock), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", &models.SnapInvalidAccount
	}

	// BACA SALDO SENDER (row sudah terkunci)
	r.logger.Info("reading sender balance", zap.String("from_account", trx.FromAccountID))
	var senderBalance float64
	err = tx.QueryRowContext(ctx,
		"SELECT balance FROM accounts WHERE id = $1", trx.FromAccountID,
	).Scan(&senderBalance)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "read sender balance failed")
		r.logger.Error("failed to read sender balance", zap.String("from_account", trx.FromAccountID), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", &models.SnapInternalError
	}

	// VALIDASI SALDO
	if senderBalance < trx.Amount {
		span.SetStatus(codes.Error, "insufficient balance")
		r.logger.Warn("insufficient balance",
			zap.Float64("balance", senderBalance),
			zap.Float64("amount", trx.Amount),
		)
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", &models.SnapInsufficient
	}

	// MUTASI SALDO (SENDER)
	r.logger.Info("deducting sender balance", zap.String("from_account", trx.FromAccountID))
	_, err = tx.ExecContext(ctx,
		"UPDATE accounts SET balance = balance - $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		trx.Amount, trx.FromAccountID,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "deduct sender failed")
		r.logger.Error("failed to mutate sender", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", &models.SnapInternalError
	}

	// MUTASI SALDO (RECEIVER)
	r.logger.Info("adding receiver balance", zap.String("to_account", trx.ToAccountID))
	_, err = tx.ExecContext(ctx,
		"UPDATE accounts SET balance = balance + $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		trx.Amount, trx.ToAccountID,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "credit receiver failed")
		r.logger.Error("failed to mutate receiver", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", &models.SnapInternalError
	}

	// INSERT TRANSACTION HISTORY
	r.logger.Info("inserting transaction history with SNAP fields")
	var referenceNo string
	queryInsert := `
		INSERT INTO transactions (
			from_account_id, to_account_id, amount, currency,
			partner_reference_no, external_id, 
			status, note, additional_info
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING reference_no`

	dbStart := time.Now()
	err = tx.QueryRowContext(ctx, queryInsert,
		trx.FromAccountID, trx.ToAccountID, trx.Amount, trx.Currency,
		trx.PartnerRefNo, trx.ExternalID,
		trx.Status, trx.Note, trx.AdditionalInfo,
	).Scan(&referenceNo)
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "insert transaction history failed")
		r.logger.Error("failed to insert history", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", &models.SnapInternalError
	}

	// COMMIT
	r.logger.Info("committing transaction", zap.String("reference_no", referenceNo))
	if err = tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "commit failed")
		r.logger.Error("failed to commit", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", &models.SnapInternalError
	}

	span.SetStatus(codes.Ok, "transfer success")
	span.SetAttributes(attribute.String("db.result.reference_no", referenceNo))
	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return referenceNo, nil
}
