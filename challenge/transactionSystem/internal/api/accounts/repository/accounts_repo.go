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

	// "strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type AccountRepository interface {
	GetAllAccounts(ctx context.Context) ([]models.Account, *models.SnapDetail)
	GetAccountById(ctx context.Context, id string) (*models.Account, *models.SnapDetail)
	GetTransactionsByAccountId(ctx context.Context, id string, trxType string) ([]models.Transaction, *models.SnapDetail)
	CreateAccount(ctx context.Context, account models.Account) (string, *models.SnapDetail)
	UpdateAccount(ctx context.Context, account models.Account) (string, *models.SnapDetail)
	DeleteAccount(ctx context.Context, id string) *models.SnapDetail
}

type accountRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewAccountRepository(db *sqlx.DB) AccountRepository {
	logger := helper.Log

	return &accountRepository{
		db:     db,
		logger: logger,
	}
}

const repoAccount = "account"

// Get All
func (r *accountRepository) GetAllAccounts(ctx context.Context) ([]models.Account, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.GetAll")
	defer span.End()
	operation := "select"

	query := `
		SELECT
			id,
			bank_code,
			account_number,
			account_holder,
			reference_no,
			partner_reference_no,
			balance,
			currency,
			email,
			phone_no,
			country_code,
			lang,
			locale,
			merchant_id,
			sub_merchant_id,
			onboarding_partner,
			terminal_type,
			scopes,
			redirect_url,
			additional_info,
			created_at,
			updated_at
		FROM accounts
		ORDER BY updated_at DESC`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "accounts"),
	)

	r.logger.Info("executing query", zap.String("query", "SELECT accounts"))

	var accounts []models.Account

	dbStart := time.Now()
	err := r.db.SelectContext(ctx, &accounts, query)
	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("query failed", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		return nil, &models.SnapInternalError
	}

	span.SetAttributes(attribute.Int("db.result.count", len(accounts)))
	r.logger.Info("query success", zap.Int("rows", len(accounts)))
	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return accounts, nil
}

// Get Account By ID
func (r *accountRepository) GetAccountById(ctx context.Context, id string) (*models.Account, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.GetById")
	defer span.End()
	operation := "select_by_id"

	query := `
		SELECT
			id,
			bank_code,
			account_number,
			account_holder,
			reference_no,
			partner_reference_no,
			balance,
			currency,
			email,
			phone_no,
			country_code,
			lang,
			locale,
			merchant_id,
			sub_merchant_id,
			onboarding_partner,
			terminal_type,
			scopes,
			redirect_url,
			additional_info,
			created_at,
			updated_at
		FROM accounts
		WHERE id = $1`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "accounts"),
		attribute.String("db.account.id", id),
	)

	r.logger.Info("executing query",
		zap.String("query", "SELECT accounts by id"),
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
			r.logger.Error(models.ErrIdNotFound.Error(), zap.Error(err))
			return nil, &models.SnapInvalidAccount
		}

		r.logger.Error("query failed", zap.Error(err))
		return nil, &models.SnapInternalError
	}

	span.SetAttributes(attribute.String("db.result.id", account.ID.String()))
	r.logger.Info("query success", zap.String("db.result.id", account.ID.String()))
	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return &account, nil
}

// GET transaction by id
func (r *accountRepository) GetTransactionsByAccountId(ctx context.Context, id string, trxType string) ([]models.Transaction, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.GetTransactionsByAccountId")
	defer span.End()
	operation := "select_transactions_by_account"

	baseQuery := `
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
		FROM transactions`

	var whereQuery string
	orderByQuery := "ORDER BY created_at DESC"

	switch trxType {
	case "all":
		whereQuery = "WHERE from_account_id = $1 OR to_account_id = $2"
	case "in":
		whereQuery = "WHERE to_account_id = $1"
	case "out":
		whereQuery = "WHERE from_account_id = $1"
	default:
		whereQuery = "WHERE from_account_id = $1 OR to_account_id = $2"
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

	r.logger.Info("executing query",
		zap.String("query", "SELECT transactions by account"),
		zap.String("account.id", id),
		zap.String("trx.type", logTrxType),
	)

	var transactions []models.Transaction

	var err error
	dbStart := time.Now()
	if trxType == "all" || trxType == "" {
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
		r.logger.Error("query failed", zap.Error(err))
		return nil, &models.SnapInternalError
	}

	if transactions == nil {
		transactions = []models.Transaction{}
	}

	span.SetAttributes(attribute.Int("db.result.count", len(transactions)))
	r.logger.Info("query success",
		zap.String("account.id", id),
		zap.String("trx.type", logTrxType),
		zap.Int("count", len(transactions)),
	)
	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return transactions, nil
}

// Post Create New Account
func (r *accountRepository) CreateAccount(ctx context.Context, account models.Account) (string, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.Create")
	defer span.End()
	operation := "insert"

	query := `
		INSERT INTO accounts (
			bank_code,
			account_number,
			account_holder,
			reference_no,
			partner_reference_no,
			balance,
			currency,
			email,
			phone_no,
			country_code,
			lang,
			locale,
			merchant_id,
			sub_merchant_id,
			onboarding_partner,
			terminal_type,
			scopes,
			redirect_url,
			additional_info
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19 
		) RETURNING id`

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "accounts"),
		attribute.String("snap.partner_ref", account.PartnerReferenceNo),
	)

	r.logger.Info("executing query",
		zap.String("query", "INSERT accounts"),
		zap.String("reference_no", account.ReferenceNo),
		zap.String("partner_reference_no", account.PartnerReferenceNo),
	)

	var newId string
	dbStart := time.Now()
	err := r.db.QueryRowxContext(ctx, query,
		account.BankCode,           // $1
		account.AccountNumber,      // $2
		account.AccountHolder,      // $3
		account.ReferenceNo,        // $4
		account.PartnerReferenceNo, // $5
		account.Balance,            // $6
		account.Currency,           // $7
		account.Email,              // $8
		account.PhoneNo,            // $9
		account.CountryCode,        // $10
		account.Lang,               // $11
		account.Locale,             // $12
		account.MerchantID,         // $13
		account.SubMerchantID,      // $14
		account.OnboardingPartner,  // $15
		account.TerminalType,       // $16
		account.Scopes,             // $17
		account.RedirectURL,        // $18
		account.AdditionalInfo,     // $19
	).Scan(&newId)

	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()

		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				r.logger.Error(models.ErrDuplicateAccount.Error(), zap.Error(err))
				return "", &models.SnapDuplicateExtID
			}
		}

		r.logger.Error(models.ErrDatabaseFailed.Error(), zap.Error(err))
		return "", &models.SnapInternalError
	}

	span.SetAttributes(attribute.String("db.result.id", newId))
	r.logger.Info("query success", zap.String("db.result.id", newId))
	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return newId, nil
}

// Method Update
func (r *accountRepository) UpdateAccount(ctx context.Context, account models.Account) (string, *models.SnapDetail) {
	fields := []string{}
	args := []any{}
	idx := 1
	operation := "update"

	tracer := middleware.TracerFromCtx(ctx)
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

	r.logger.Info("executing query",
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
				r.logger.Error(models.ErrDuplicateAccount.Error(), zap.Error(err))
				return "", &models.SnapDuplicateRefNo
			}
		}

		r.logger.Error(models.ErrDatabaseFailed.Error(), zap.Error(err))
		return "", &models.SnapInternalError
	}

	// Cek apakah data dengan ID tersebut ditemukan
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		r.logger.Error(models.ErrDatabaseIssue.Error(), zap.Error(err))
		return "", &models.SnapInternalError
	}

	if rowsAffected == 0 {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		r.logger.Error(models.ErrIdNotFound.Error())
		return "", &models.SnapInvalidAccount
	}

	span.SetAttributes(
		attribute.String("db.result.id", account.ID.String()),
	)

	r.logger.Info("query success",
		zap.String("db.result.id", account.ID.String()),
	)

	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return account.ID.String(), nil
}

// Method Delete
func (r *accountRepository) DeleteAccount(ctx context.Context, id string) *models.SnapDetail {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.Delete")
	defer span.End()
	operation := "delete"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.table", "accounts"),
	)

	query := `DELETE FROM accounts WHERE id = $1`

	r.logger.Info("executing query",
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

		r.logger.Error(models.ErrDatabaseFailed.Error(), zap.Error(err))
		return &models.SnapInternalError
	}

	// Cek apakah data dengan ID tersebut ditemukan
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		r.logger.Error(models.ErrDatabaseIssue.Error(), zap.Error(err))
		return &models.SnapInternalError
	}

	if rowsAffected == 0 {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		r.logger.Error(models.ErrIdNotFound.Error())
		return &models.SnapInvalidAccount
	}

	span.SetAttributes(
		attribute.String("db.delete.id", id),
	)

	r.logger.Info("query success",
		zap.String("db.delete.id", id),
	)

	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return nil
}
