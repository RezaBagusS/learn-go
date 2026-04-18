package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"my-microservices/account-service/helper"
	"my-microservices/account-service/internal/domain"
	"my-microservices/account-service/internal/middleware"
	"my-microservices/account-service/observability/metrics"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type AccountRepository interface {
	GetAllAccounts(ctx context.Context) ([]domain.Account, *domain.SnapDetail)
	GetAccountById(ctx context.Context, id string) (*domain.Account, *domain.SnapDetail)
	CreateAccount(ctx context.Context, account domain.Account) (string, *domain.SnapDetail)
	UpdateAccount(ctx context.Context, account domain.Account) (string, *domain.SnapDetail)
	DeleteAccount(ctx context.Context, id string) *domain.SnapDetail
	ProcessTransferMutation(ctx context.Context, sourceAcc, beneficiaryAcc string, amount int64) error
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
func (r *accountRepository) GetAllAccounts(ctx context.Context) ([]domain.Account, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.GetAll")
	defer span.End()
	operation := "select"

	query := `
		SELECT
			id,
			bank_id,
			account_number,
			account_holder,
			customer_id,
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

	var accounts []domain.Account

	dbStart := time.Now()
	err := r.db.SelectContext(ctx, &accounts, query)
	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error("query failed", zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		return nil, &domain.SnapInternalError
	}

	span.SetAttributes(attribute.Int("db.result.count", len(accounts)))
	r.logger.Info("query success", zap.Int("rows", len(accounts)))
	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return accounts, nil
}

// Get Account By ID
func (r *accountRepository) GetAccountById(ctx context.Context, id string) (*domain.Account, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.GetById")
	defer span.End()
	operation := "select_by_id"

	query := `
		SELECT
			id,
			bank_id,
			account_number,
			account_holder,
			customer_id,
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

	var account domain.Account

	dbStart := time.Now()
	err := r.db.GetContext(ctx, &account, query, id)
	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()

		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(domain.ErrIdNotFound.Error(), zap.Error(err))
			return nil, &domain.SnapInvalidAccount
		}

		r.logger.Error("query failed", zap.Error(err))
		return nil, &domain.SnapInternalError
	}

	span.SetAttributes(attribute.String("db.result.id", account.ID.String()))
	r.logger.Info("query success", zap.String("db.result.id", account.ID.String()))
	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return &account, nil
}

// Post Create New Account
func (r *accountRepository) CreateAccount(ctx context.Context, account domain.Account) (string, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.Create")
	defer span.End()
	operation := "insert"

	query := `
		INSERT INTO accounts (
			bank_id,
			account_number,
			account_holder,
			customer_id,
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
			$16, $17, $18, $19, $20
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
		account.BankID,             // $1
		account.AccountNumber,      // $2a
		account.AccountHolder,      // $3
		account.CustomerID,         // $4
		account.ReferenceNo,        // $5
		account.PartnerReferenceNo, // $6
		100000,                     // $7 initial balance
		account.Currency,           // $8
		account.Email,              // $9
		account.PhoneNo,            // $10
		account.CountryCode,        // $11
		account.Lang,               // $12
		account.Locale,             // $13
		account.MerchantID,         // $14
		account.SubMerchantID,      // $15
		account.OnboardingPartner,  // $16
		account.TerminalType,       // $17
		account.Scopes,             // $18
		account.RedirectURL,        // $19
		account.AdditionalInfo,     // $20
	).Scan(&newId)

	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()

		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				r.logger.Error(domain.ErrDuplicateAccount.Error(), zap.Error(err))
				return "", &domain.SnapDuplicateExtID
			}
		}

		r.logger.Error(domain.ErrDatabaseFailed.Error(), zap.Error(err))
		return "", &domain.SnapInternalError
	}

	span.SetAttributes(attribute.String("db.result.id", newId))
	r.logger.Info("query success", zap.String("db.result.id", newId))
	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return newId, nil
}

// Method Update
func (r *accountRepository) UpdateAccount(ctx context.Context, account domain.Account) (string, *domain.SnapDetail) {
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

	if account.AccountNumber != "" {
		fields = append(fields, fmt.Sprintf("account_number = $%d", idx))
		args = append(args, account.AccountNumber)
		idx++
	}

	if account.AccountHolder != "" {
		fields = append(fields, fmt.Sprintf("account_holder = $%d", idx))
		args = append(args, account.AccountHolder)
		idx++
	}

	fields = append(fields, fmt.Sprintf("updated_at = $%d", idx))
	args = append(args, time.Now())
	idx++

	args = append(args, account.ID)
	query := fmt.Sprintf(
		"UPDATE accounts SET %s WHERE id = $%d",
		strings.Join(fields, ", "),
		idx,
	)

	r.logger.Info("executing query", zap.String("query", "UPDATE accounts"))

	dbStart := time.Now()
	result, err := r.db.ExecContext(ctx, query, args...)
	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()

		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				r.logger.Error(domain.ErrDuplicateAccount.Error(), zap.Error(err))
				return "", &domain.SnapDuplicateRefNo
			}
		}

		r.logger.Error(domain.ErrDatabaseFailed.Error(), zap.Error(err))
		return "", &domain.SnapInternalError
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		r.logger.Error(domain.ErrDatabaseIssue.Error(), zap.Error(err))
		return "", &domain.SnapInternalError
	}

	if rowsAffected == 0 {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		r.logger.Error(domain.ErrIdNotFound.Error())
		return "", &domain.SnapInvalidAccount
	}

	span.SetAttributes(attribute.String("db.result.id", account.ID.String()))
	r.logger.Info("query success", zap.String("db.result.id", account.ID.String()))
	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return account.ID.String(), nil
}

// Method Delete
func (r *accountRepository) DeleteAccount(ctx context.Context, id string) *domain.SnapDetail {

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

	r.logger.Info("executing query", zap.String("query", "DELETE accounts"))

	dbStart := time.Now()
	result, err := r.db.ExecContext(ctx, query, id)
	metrics.DBQueryDuration.WithLabelValues(repoAccount, operation).
		Observe(time.Since(dbStart).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		r.logger.Error(domain.ErrDatabaseFailed.Error(), zap.Error(err))
		return &domain.SnapInternalError
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		r.logger.Error(domain.ErrDatabaseIssue.Error(), zap.Error(err))
		return &domain.SnapInternalError
	}

	if rowsAffected == 0 {
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		r.logger.Error(domain.ErrIdNotFound.Error())
		return &domain.SnapInvalidAccount
	}

	span.SetAttributes(attribute.String("db.delete.id", id))
	r.logger.Info("query success", zap.String("db.delete.id", id))
	metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "success").Inc()

	return nil
}

// ProcessTransferMutation
func (r *accountRepository) ProcessTransferMutation(ctx context.Context, sourceAcc, beneficiaryAcc string, amount int64) error {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountRepo.TransferMutation")
	defer span.End()
	operation := "insert_snap"

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("source.account", sourceAcc),
		attribute.String("beneficiary.account", beneficiaryAcc),
	)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		r.logger.Error(domain.ErrDatabaseTrx.Error(), zap.Error(err))
		metrics.DBQueryTotal.WithLabelValues(repoAccount, operation, "error").Inc()
		return errors.New(domain.SnapInternalError.ResponseMessage)
	}

	defer tx.Rollback()

	// Lock order yang konsisten — hindari deadlock
	firstLock, secondLock := sourceAcc, beneficiaryAcc
	if firstLock > secondLock {
		firstLock, secondLock = secondLock, firstLock
	}

	// Account validation
	var dummy string
	err = tx.QueryRowContext(ctx, "SELECT id FROM accounts WHERE account_number = $1 FOR UPDATE", firstLock).Scan(&dummy)
	if err != nil {
		r.logger.Error("Account pengirim tidak ditemukan", zap.Error(err))
		return errors.New(domain.SnapInvalidAccount.ResponseMessage)
	}

	err = tx.QueryRowContext(ctx, "SELECT id FROM accounts WHERE account_number = $1 FOR UPDATE", secondLock).Scan(&dummy)
	if err != nil {
		r.logger.Error("Account penerima tidak ditemukan", zap.Error(err))
		return errors.New(domain.SnapInvalidAccount.ResponseMessage)
	}

	// Balance validation
	var senderBalance int64
	err = tx.QueryRowContext(ctx, "SELECT balance::BIGINT FROM accounts WHERE account_number = $1", sourceAcc).Scan(&senderBalance)
	if err != nil {
		r.logger.Error("Gagal mengambil data saldo pengirim", zap.Error(err))
		return domain.ErrDatabaseFailed
	}

	if senderBalance < amount {
		r.logger.Error("Saldo pengirim tidak cukup", zap.Error(err))
		return domain.ErrLogicBalanceTrx
	}

	// Mutasi Sender
	_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance - $1, updated_at = NOW() WHERE account_number = $2", amount, sourceAcc)
	if err != nil {
		r.logger.Error("Gagal mutasi saldo pengirim", zap.Error(err))
		return fmt.Errorf("failed to deduct sender balance: %w", err)
	}

	// Mutasi Receiver
	_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance + $1, updated_at = NOW() WHERE account_number = $2", amount, beneficiaryAcc)
	if err != nil {
		r.logger.Error("Gagal mutasi saldo penerima", zap.Error(err))
		return fmt.Errorf("failed to add beneficiary balance: %w", err)
	}

	// Commit
	if err := tx.Commit(); err != nil {
		r.logger.Error("Commit proses transfer mutasi akun", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
