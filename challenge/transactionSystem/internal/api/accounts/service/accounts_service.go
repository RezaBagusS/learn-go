package service

import (
	"belajar-go/challenge/transactionSystem/internal/api/accounts/repository"
	"belajar-go/challenge/transactionSystem/internal/api/banks/service"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"belajar-go/challenge/transactionSystem/observability/metrics"
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type AccountsService interface {
	FetchAllAccounts(ctx context.Context) ([]models.Account, error)
	FetchAccountById(ctx context.Context, id string) (*models.Account, error)
	FetchTransactionsByAccountId(ctx context.Context, id string, trxType string) ([]models.Transaction, error)
	CreateNewAccount(ctx context.Context, account models.Account) (*models.Account, error)
	PatchAccountById(ctx context.Context, account models.Account) (string, error)
	DeleteAccountById(ctx context.Context, id string) error
}

type accountsService struct {
	repo    repository.AccountRepository // Depend pada Interface, bukan struct DB langsung
	bankSvc service.BankService
}

func NewAccountsService(repo repository.AccountRepository, bankSvc service.BankService) AccountsService {
	return &accountsService{
		repo:    repo,
		bankSvc: bankSvc,
	}
}

const svcAccount = "account"

// Fetch All Data
func (s *accountsService) FetchAllAccounts(ctx context.Context) ([]models.Account, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.GetAll")
	defer span.End()
	operation := "fetch_all"

	logger.Info("fetching accounts from repository")

	svcStart := time.Now()
	accounts, err := s.repo.GetAllAccounts(ctx)
	metrics.ServiceDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error("failed fetching accounts", zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("service.result.count", len(accounts)),
	)

	logger.Info("success fetching accounts",
		zap.Int("count", len(accounts)),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return accounts, nil
}

// Fetch Account by Id
func (s *accountsService) FetchAccountById(ctx context.Context, id string) (*models.Account, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.GetById")
	defer span.End()
	operation := "fetch_by_id"

	logger.Info("fetching account from repository")

	svcStart := time.Now()
	account, err := s.repo.GetAccountById(ctx, id)
	metrics.CacheDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return nil, err
	}

	span.SetAttributes(
		attribute.String("service.result.id", account.ID.String()),
	)

	logger.Info("success fetching account",
		zap.String("service.result.id", account.ID.String()),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return account, nil
}

// Fetch Transaction by Account Id
func (s *accountsService) FetchTransactionsByAccountId(ctx context.Context, id string, trxType string) ([]models.Transaction, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.GetTrxById")
	defer span.End()
	operation := "fetch_by_id"
	operationTrx := "fetch_trx_by_id"

	logger.Info("checking valid accound id")

	svcStart := time.Now()
	_, err := s.repo.GetAccountById(ctx, id)
	metrics.CacheDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return nil, err
	}

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	logger.Info("fetching trx account from repository")

	svcStartTrx := time.Now()
	transactions, err := s.repo.GetTransactionsByAccountId(ctx, id, trxType)
	metrics.CacheDuration.WithLabelValues(svcAccount, operationTrx).
		Observe(time.Since(svcStartTrx).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operationTrx, "error").Inc()
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("service.result.count", len(transactions)),
	)

	logger.Info("success fetching account",
		zap.Int("service.result.count", len(transactions)),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operationTrx, "success").Inc()

	return transactions, nil
}

// Create new account
func (s *accountsService) CreateNewAccount(ctx context.Context, account models.Account) (*models.Account, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.Create")
	defer span.End()
	operation := "account_exist"
	operationCreate := "create"

	logger.Info("checking payload")

	if account.BankCode == "" || account.AccountNumber == "" || account.AccountHolder == "" {
		span.RecordError(models.ErrInvalidField)
		logger.Error(models.ErrInvalidField.Error(), zap.Error(models.ErrInvalidField))
		metrics.BusinessValidationErrors.WithLabelValues(svcAccount, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return nil, models.ErrInvalidField
	}

	// Balance checking ...
	if account.Balance < 0 {
		span.RecordError(models.ErrInvalidInitBalance)
		logger.Error(models.ErrInvalidInitBalance.Error(), zap.Error(models.ErrInvalidInitBalance))
		metrics.BusinessValidationErrors.WithLabelValues(svcAccount, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return nil, models.ErrInvalidInitBalance
	}

	logger.Info("checking bank code")

	svcStartCheckBank := time.Now()
	_, err := s.bankSvc.FetchBankById(ctx, account.BankCode)
	metrics.ServiceDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStartCheckBank).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return nil, err
	}

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	logger.Info("creating new account")

	// Simpan ke repository
	svcStart := time.Now()
	returnedId, err := s.repo.CreateAccount(ctx, account)
	metrics.ServiceDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operationCreate, "error").Inc()
		return nil, err
	}

	account.ID = uuid.MustParse(returnedId)

	span.SetAttributes(
		attribute.String("service.result.id", returnedId),
	)

	logger.Info("success creating new account",
		zap.String("service.result.id", returnedId),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operationCreate, "success").Inc()

	return &account, nil
}

// Update account
func (s *accountsService) PatchAccountById(ctx context.Context, account models.Account) (string, error) {

	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.Update")
	defer span.End()
	operation := "update"

	logger.Info("checking payload")

	// Jika tidak ada field yang diupdate
	if account.AccountHolder == "" && account.AccountNumber == "" {
		span.RecordError(models.ErrInvalidField)
		logger.Error(models.ErrInvalidField.Error(), zap.Error(models.ErrInvalidField))
		metrics.BusinessValidationErrors.WithLabelValues(svcAccount, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return "", models.ErrInvalidField
	}

	logger.Info("updating account data")

	svcStart := time.Now()
	getId, err := s.repo.UpdateAccount(ctx, account)
	metrics.CacheDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return "", err
	}

	span.SetAttributes(
		attribute.String("service.result.id", getId),
	)

	logger.Info("success updating account data",
		zap.String("service.result.id", getId),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return getId, nil
}

// Delete account
func (s *accountsService) DeleteAccountById(ctx context.Context, id string) error {
	_, logger, tracer := middleware.AllCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.Delete")
	defer span.End()
	operation := "delete"

	logger.Info("deleting account data",
		zap.String("service.delete.id", id),
	)

	svcStart := time.Now()
	err := s.repo.DeleteAccount(ctx, id)
	metrics.CacheDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStart).Seconds())

	if err != nil {
		span.RecordError(err)
		logger.Error(err.Error(), zap.Error(err))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return err
	}

	span.SetAttributes(
		attribute.String("service.delete.id", id),
	)

	logger.Info("success deleting account data",
		zap.String("service.delete.id", id),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return nil
}
