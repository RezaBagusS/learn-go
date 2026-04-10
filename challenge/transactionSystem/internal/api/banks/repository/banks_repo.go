package repository

import (
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"belajar-go/challenge/transactionSystem/observability/metrics"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type BankRepository interface {
	GetAllBanks(ctx context.Context) ([]models.Bank, error)
	GetBankById(ctx context.Context, id string) (*models.Bank, *models.SnapDetail)
	CreateBank(ctx context.Context, bank models.Bank) (string, error)
	UpdateBank(ctx context.Context, bank models.Bank) (string, error)
	DeleteBank(ctx context.Context, bankCode string) error
}

type bankRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewBankRepository(db *sqlx.DB) BankRepository {
	logger := helper.Log

	return &bankRepository{
		db:     db,
		logger: logger,
	}
}

const repoBank = "bank"

// Get All
func (r *bankRepository) GetAllBanks(ctx context.Context) ([]models.Bank, error) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankRepo.GetAll")
	defer span.End()
	operation := "select"

	query := "SELECT id, bank_code, bank_name, created_at FROM banks ORDER BY created_at desc"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "banks"),
	)

	r.logger.Info("executing query",
		zap.String("query", "SELECT banks"),
	)

	var banks []models.Bank

	dbStart := time.Now()
	err := r.db.SelectContext(ctx, &banks, query)
	metrics.DBQueryDuration.WithLabelValues(repoBank, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		r.logger.Error("query failed", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "error").Inc()

		return nil, models.ErrDatabaseIssue
	}

	span.SetAttributes(attribute.Int("db.result.count", len(banks)))

	r.logger.Info("query success",
		zap.Int("rows", len(banks)),
	)

	metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "success").Inc()

	return banks, nil
}

// Get Bank by Bank Id
func (r *bankRepository) GetBankById(ctx context.Context, id string) (*models.Bank, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankRepo.GetById")
	defer span.End()
	operation := "select_by_id"

	query := "SELECT id, bank_code, bank_name, created_at FROM banks WHERE id::text = $1 or bank_code =$1"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "banks"),
	)

	r.logger.Info("executing query",
		zap.String("query", "SELECT banks"),
	)

	var bank models.Bank

	dbStart := time.Now()
	err := r.db.GetContext(ctx, &bank, query, id)
	metrics.DBQueryDuration.WithLabelValues(repoBank, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "error").Inc()

		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(models.ErrIdNotFound.Error(), zap.Error(err))
			return nil, &models.SnapInvalidAccount
		}

		r.logger.Error("query failed", zap.Error(err))

		return nil, &models.SnapInternalError
	}

	span.SetAttributes(
		attribute.String("db.result.id", bank.ID.String()),
	)

	r.logger.Info("query success",
		zap.String("db.result.id", bank.ID.String()),
	)

	metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "success").Inc()

	return &bank, nil
}

// Post Create New Bank
func (r *bankRepository) CreateBank(ctx context.Context, bank models.Bank) (string, error) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankRepo.Create")
	defer span.End()
	operation := "insert"

	query := `INSERT INTO banks (bank_code, bank_name) VALUES ($1, $2) RETURNING id`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "banks"),
	)

	r.logger.Info("executing query",
		zap.String("query", "INSERT banks"),
	)

	var newId string
	dbStart := time.Now()
	err := r.db.QueryRowxContext(ctx, query, bank.BankCode, bank.BankName).Scan(&newId)
	metrics.DBQueryDuration.WithLabelValues(repoBank, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "error").Inc()

		if pqErr, ok := err.(*pq.Error); ok {
			// [23505] Unique Violation
			if pqErr.Code == "23505" {
				r.logger.Error(models.ErrDuplicateBank.Error(), zap.Error(err))
				return "", models.ErrDuplicateBank
			}
		}

		r.logger.Error(models.ErrDatabaseFailed.Error(), zap.Error(err))

		return "", models.ErrDatabaseFailed
	}

	span.SetAttributes(
		attribute.String("db.result.id", newId),
	)

	r.logger.Info("query success",
		zap.String("db.result.id", newId),
	)

	metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "success").Inc()

	return newId, nil
}

// Method Update
func (r *bankRepository) UpdateBank(ctx context.Context, bank models.Bank) (string, error) {
	fields := []string{}
	args := []any{}
	idx := 1
	operation := "update"

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankRepo.Update")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.table", "banks"),
	)

	// Cek BankCode
	if bank.BankCode != "" {
		fields = append(fields, fmt.Sprintf("bank_code = $%d", idx))
		args = append(args, bank.BankCode)
		idx++
	}

	// Cek Desc
	if bank.BankName != "" {
		fields = append(fields, fmt.Sprintf("bank_name = $%d", idx))
		args = append(args, bank.BankName)
		idx++
	}

	// Tambahkan id sebagai kondisi WHERE
	args = append(args, bank.ID)
	query := fmt.Sprintf(
		"UPDATE banks SET %s WHERE id = $%d",
		strings.Join(fields, ", "),
		idx,
	)

	// Query Execution
	r.logger.Info("executing query",
		zap.String("query", "UPDATE banks"),
	)

	dbStart := time.Now()
	result, err := r.db.ExecContext(ctx, query, args...)
	metrics.DBQueryDuration.WithLabelValues(repoBank, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "error").Inc()

		if pqErr, ok := err.(*pq.Error); ok {
			// [23505] Unique Violation
			if pqErr.Code == "23505" {
				r.logger.Error(models.ErrDuplicateBank.Error(), zap.Error(err))
				return "", models.ErrDuplicateBank
			}
		}

		r.logger.Error(models.ErrDatabaseFailed.Error(), zap.Error(err))
		return "", models.ErrDatabaseFailed
	}

	// Cek apakah data dengan ID tersebut ditemukan
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "error").Inc()
		r.logger.Error(models.ErrDatabaseIssue.Error(), zap.Error(err))
		return "", models.ErrDatabaseIssue
	}

	if rowsAffected == 0 {
		metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "error").Inc()
		r.logger.Error(models.ErrIdNotFound.Error())
		return "", models.ErrIdNotFound
	}

	span.SetAttributes(
		attribute.String("db.result.bankCode", bank.BankCode),
	)

	r.logger.Info("query success",
		zap.String("db.result.bankCode", bank.BankCode),
	)

	metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "success").Inc()

	return bank.BankCode, nil
}

// Method Delete
func (r *bankRepository) DeleteBank(ctx context.Context, bankId string) error {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankRepo.Delete")
	defer span.End()
	operation := "delete"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.table", "banks"),
	)

	query := `DELETE FROM banks WHERE id = $1`

	// Query Execution
	r.logger.Info("executing query",
		zap.String("query", "DELETE banks"),
	)

	dbStart := time.Now()
	result, err := r.db.ExecContext(ctx, query, bankId)
	metrics.DBQueryDuration.WithLabelValues(repoBank, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "error").Inc()

		r.logger.Error(models.ErrDatabaseFailed.Error(), zap.Error(err))
		return models.ErrDeleteFailed
	}

	// Cek apakah data dengan ID tersebut ditemukan
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "error").Inc()
		r.logger.Error(models.ErrDatabaseIssue.Error(), zap.Error(err))
		return models.ErrDatabaseIssue
	}

	if rowsAffected == 0 {
		metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "error").Inc()
		r.logger.Error(models.ErrIdNotFound.Error())
		return models.ErrIdNotFound
	}

	span.SetAttributes(
		attribute.String("db.delete.id", bankId),
	)

	r.logger.Info("query success",
		zap.String("db.delete.id", bankId),
	)

	metrics.DBQueryTotal.WithLabelValues(repoBank, operation, "success").Inc()

	return nil
}
