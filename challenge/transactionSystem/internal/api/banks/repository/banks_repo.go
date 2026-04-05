package repository

import (
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	// "strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type BankRepository interface {
	GetAllBanks(ctx context.Context) ([]models.Bank, error)
	GetBankById(ctx context.Context, id string) (*models.Bank, error)
	CreateBank(ctx context.Context, bank models.Bank) (string, error)
	UpdateBank(bank models.Bank) (string, error)
	DeleteBank(bankCode string) error
}

type bankRepository struct {
	db *sqlx.DB
}

func NewBankRepository(db *sqlx.DB) BankRepository {
	return &bankRepository{db: db}
}

// Get All
func (r *bankRepository) GetAllBanks(ctx context.Context) ([]models.Bank, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankRepo.GetAll")
	defer span.End()

	query := "SELECT id, bank_code, bank_name, created_at FROM banks ORDER BY created_at desc"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "banks"),
	)

	logger.Info("executing query",
		zap.String("query", "SELECT banks"),
	)

	var banks []models.Bank

	err := r.db.SelectContext(ctx, &banks, query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		logger.Error("query failed", zap.Error(err))

		return nil, models.ErrDatabaseIssue
	}

	span.SetAttributes(attribute.Int("db.result.count", len(banks)))

	logger.Info("query success",
		zap.Int("rows", len(banks)),
	)

	return banks, nil
}

// Get Bank by Bank Id
func (r *bankRepository) GetBankById(ctx context.Context, id string) (*models.Bank, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankRepo.GetById")
	defer span.End()

	query := "SELECT id, bank_code, bank_name, created_at FROM banks WHERE id::text = $1 or bank_code =$1"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "banks"),
	)

	logger.Info("executing query",
		zap.String("query", "SELECT banks"),
	)

	var bank models.Bank

	err := r.db.GetContext(ctx, &bank, query, id)
	if err != nil {

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(models.ErrIdNotFound.Error(), zap.Error(err))
			return nil, models.ErrIdNotFound
		}

		logger.Error("query failed", zap.Error(err))

		return nil, models.ErrDatabaseIssue
	}

	span.SetAttributes(
		attribute.String("db.result.id", bank.ID.String()),
	)

	logger.Info("query success",
		zap.String("db.result.id", bank.ID.String()),
	)

	return &bank, nil
}

// Post Create New Bank
func (r *bankRepository) CreateBank(ctx context.Context, bank models.Bank) (string, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "BankRepo.Create")
	defer span.End()

	query := `INSERT INTO banks (bank_code, bank_name) VALUES ($1, $2) RETURNING id`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "banks"),
	)

	logger.Info("executing query",
		zap.String("query", "INSERT banks"),
	)

	var newId string
	err := r.db.QueryRowxContext(ctx, query, bank.BankCode, bank.BankName).Scan(&newId)
	if err != nil {

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		if pqErr, ok := err.(*pq.Error); ok {
			// [23505] Unique Violation
			if pqErr.Code == "23505" {
				logger.Error(models.ErrDuplicateBank.Error(), zap.Error(err))
				return "", models.ErrDuplicateBank
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

	return newId, nil
}

// Method Update
func (r *bankRepository) UpdateBank(bank models.Bank) (string, error) {
	fields := []string{}
	args := []any{}
	idx := 1

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
	fmt.Printf("Query [bank][repo]: %v \n", query)
	fmt.Printf("Args [bank][repo]: %v \n", args)

	result, err := r.db.Exec(query, args...)
	if err != nil {

		if pqErr, ok := err.(*pq.Error); ok {
			// [23505] Unique Violation
			if pqErr.Code == "23505" {
				helper.PrintLog("account", helper.LogPositionRepo, models.ErrDuplicateBank.Error())
				return "", models.ErrDuplicateBank
			}
		}

		helper.PrintLog("bank", helper.LogPositionRepo, err.Error())
		return "", models.ErrDatabaseFailed
	}

	// Cek apakah data dengan ID tersebut ditemukan
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		helper.PrintLog("bank", helper.LogPositionRepo, err.Error())
		return "", models.ErrDatabaseIssue
	}

	if rowsAffected == 0 {
		helper.PrintLog("bank", helper.LogPositionRepo, models.ErrIdNotFound.Error())
		return "", models.ErrIdNotFound
	}

	return bank.BankCode, nil
}

// Method Delete
func (r *bankRepository) DeleteBank(bankId string) error {
	query := `DELETE FROM banks WHERE id = $1`

	result, err := r.db.Exec(query, bankId)
	if err != nil {
		helper.PrintLog("bank", helper.LogPositionRepo, err.Error())
		return models.ErrDeleteFailed
	}

	// Cek apakah data dengan ID tersebut ditemukan
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		helper.PrintLog("bank", helper.LogPositionRepo, err.Error())
		return models.ErrDatabaseIssue
	}

	if rowsAffected == 0 {
		helper.PrintLog("bank", helper.LogPositionRepo, models.ErrIdNotFound.Error())
		return models.ErrIdNotFound
	}

	return nil
}
