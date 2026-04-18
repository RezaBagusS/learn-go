package repository

import (
	"context"
	"database/sql"
	"errors"
	"my-microservices/transaction-service/helper"
	"my-microservices/transaction-service/internal/domain"
	"my-microservices/transaction-service/internal/middleware"
	"my-microservices/transaction-service/observability/metrics"
	"time"

	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type TransactionRepository interface {
	GetAllTransactions(ctx context.Context) ([]domain.Transaction, *domain.SnapDetail)
	GetTransactionById(ctx context.Context, id string) (*domain.Transaction, *domain.SnapDetail)
	GetSummary(ctx context.Context, date time.Time) ([]domain.Transaction, *domain.SnapDetail)
	GetTransactionsByAccountNo(ctx context.Context, accountNo string) ([]domain.Transaction, *domain.SnapDetail)
	TransferIntraBank(ctx context.Context, trx domain.Transaction) (string, *domain.SnapDetail)
}

type transactionRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewTransactionRepository(db *sqlx.DB) TransactionRepository {
	return &transactionRepository{
		db:     db,
		logger: helper.Log,
	}
}

const repoTransaction = "transaction"

// Get All
func (r *transactionRepository) GetAllTransactions(ctx context.Context) ([]domain.Transaction, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.GetAll")
	defer span.End()
	operation := "select"

	query := `
		SELECT
			id, from_account_no, to_account_no, amount, currency,
			reference_no, partner_reference_no, external_id,
			status, note, additional_info, created_at
		FROM transactions
		ORDER BY created_at DESC`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "transactions"),
	)

	var transactions []domain.Transaction

	dbStart := time.Now()
	err := r.db.SelectContext(ctx, &transactions, query)
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("query failed", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return nil, &domain.SnapInternalError
	}

	if transactions == nil {
		transactions = []domain.Transaction{}
	}

	span.SetAttributes(attribute.Int("db.result.count", len(transactions)))
	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return transactions, nil
}

// Get Summary by Date
func (r *transactionRepository) GetSummary(ctx context.Context, date time.Time) ([]domain.Transaction, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.GetSummary")
	defer span.End()
	operation := "select"

	query := `
		SELECT
			id, from_account_no, to_account_no, amount, currency,
			reference_no, partner_reference_no, external_id,
			status, note, additional_info, created_at
		FROM transactions
		WHERE created_at >= $1::date
		  AND created_at < ($1::date + INTERVAL '1 day')
		ORDER BY created_at DESC`

	span.SetAttributes(
		attribute.String("db.query.date", date.Format("2006-01-02")),
	)

	transactions := make([]domain.Transaction, 0)

	dbStart := time.Now()
	err := r.db.SelectContext(ctx, &transactions, query, date.Format("2006-01-02"))
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("query failed", zap.Error(err), zap.String("date", date.Format("2006-01-02")))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return nil, &domain.SnapInternalError
	}

	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return transactions, nil
}

// Get Transaction By ID
func (r *transactionRepository) GetTransactionById(ctx context.Context, id string) (*domain.Transaction, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.GetById")
	defer span.End()
	operation := "select"

	query := `
		SELECT
			id, from_account_no, to_account_no, amount, currency,
			reference_no, partner_reference_no, external_id,
			status, note, additional_info, created_at
		FROM transactions
		WHERE id = $1`

	var transaction domain.Transaction

	dbStart := time.Now()
	err := r.db.GetContext(ctx, &transaction, query, id)
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).Observe(time.Since(dbStart).Seconds())

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			span.RecordError(domain.ErrIdNotFound)
			span.SetStatus(codes.Error, domain.ErrIdNotFound.Error())
			r.logger.Error("transaction not found", zap.String("id", id))
			metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "not_found").Inc()
			return nil, &domain.SnapTrxNotFound
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("query failed", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return nil, &domain.SnapInternalError
	}

	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return &transaction, nil
}

// Get Transactions By Account Number
func (r *transactionRepository) GetTransactionsByAccountNo(ctx context.Context, accountNo string) ([]domain.Transaction, *domain.SnapDetail) {
	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.GetByAccountNo")
	defer span.End()
	operation := "select"

	query := `
		SELECT
			id, from_account_no, to_account_no, amount, currency,
			reference_no, partner_reference_no, external_id,
			status, note, additional_info, created_at
		FROM transactions
		WHERE from_account_no = $1 OR to_account_no = $1
		ORDER BY created_at DESC`

	var transactions []domain.Transaction

	dbStart := time.Now()
	err := r.db.SelectContext(ctx, &transactions, query, accountNo)
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		r.logger.Error("query failed", zap.Error(err), zap.String("accountNo", accountNo))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return nil, &domain.SnapInternalError
	}

	if transactions == nil {
		transactions = []domain.Transaction{}
	}

	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()
	return transactions, nil
}

// TransferIntraBank — DB Transaction
func (r *transactionRepository) TransferIntraBank(ctx context.Context, trx domain.Transaction) (string, *domain.SnapDetail) {
	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "TransactionRepo.TransferIntraBank")
	defer span.End()
	operation := "insert_snap"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("snap.partner_ref", trx.PartnerRefNo),
	)

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error(domain.ErrDatabaseTrx.Error(), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", &domain.SnapInternalError
	}
	defer tx.Rollback()

	// Insert transaction history
	var referenceNo string
	queryInsert := `
		INSERT INTO transactions (
			reference_no, from_account_no, to_account_no, amount, currency,
			partner_reference_no, external_id,
			status, note, additional_info
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING reference_no`

	dbStart := time.Now()
	err = tx.QueryRowContext(ctx, queryInsert,
		trx.ReferenceNo, trx.FromAccountNo, trx.ToAccountNo, trx.Amount, trx.Currency,
		trx.PartnerRefNo, trx.ExternalID,
		trx.Status, trx.Note, trx.AdditionalInfo,
	).Scan(&referenceNo)
	metrics.DBQueryDuration.WithLabelValues(repoTransaction, operation).Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		r.logger.Error("failed to insert transaction history", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "error").Inc()
		return "", &domain.SnapInternalError
	}

	if err = tx.Commit(); err != nil {
		r.logger.Error("failed to commit", zap.Error(err))
		return "", &domain.SnapInternalError
	}

	span.SetStatus(codes.Ok, "transfer success")
	span.SetAttributes(attribute.String("db.result.reference_no", referenceNo))
	metrics.DBQueryTotal.WithLabelValues(repoTransaction, operation, "success").Inc()

	return referenceNo, nil
}
