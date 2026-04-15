package service

import (
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/accounts/repository"
	"belajar-go/challenge/transactionSystem/internal/api/banks/service"
	"belajar-go/challenge/transactionSystem/internal/kafka"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"belajar-go/challenge/transactionSystem/observability/metrics"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type AccountsService interface {
	FetchAllAccounts(ctx context.Context) ([]models.Account, *models.SnapDetail)
	FetchAccountById(ctx context.Context, id string) (*models.Account, *models.SnapDetail)
	FetchTransactionsByAccountId(ctx context.Context, id string, trxType string) ([]models.Transaction, *models.SnapDetail)
	CreateNewAccount(ctx context.Context, account models.AccountCreateRequest, producer *kafka.Producer, svcCode string) (*models.AccountCreateResponse, *models.SnapDetail)
	PatchAccountById(ctx context.Context, account models.Account) (string, *models.SnapDetail)
	DeleteAccountById(ctx context.Context, id string) *models.SnapDetail
}

type accountsService struct {
	repo    repository.AccountRepository // Depend pada Interface, bukan struct DB langsung
	bankSvc service.BankService
	logger  *zap.Logger
}

func NewAccountsService(repo repository.AccountRepository, bankSvc service.BankService) AccountsService {
	logger := helper.Log

	return &accountsService{
		repo:    repo,
		bankSvc: bankSvc,
		logger:  logger,
	}
}

const svcAccount = "account"
const DEFAULT_CURRENCY = "IDR"

// Fetch All Data
func (s *accountsService) FetchAllAccounts(ctx context.Context) ([]models.Account, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.GetAll")
	defer span.End()
	operation := "fetch_all"

	s.logger.Info("fetching accounts from repository")

	svcStart := time.Now()
	accounts, snapErr := s.repo.GetAllAccounts(ctx)
	metrics.ServiceDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		prefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefix)
		s.logger.Error("failed fetching accounts", zap.Error(prefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return nil, snapErr
	}

	span.SetAttributes(
		attribute.Int("service.result.count", len(accounts)),
	)

	s.logger.Info("success fetching accounts",
		zap.Int("count", len(accounts)),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return accounts, nil
}

// Fetch Account by Id
func (s *accountsService) FetchAccountById(ctx context.Context, id string) (*models.Account, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.GetById")
	defer span.End()
	operation := "fetch_by_id"

	s.logger.Info("fetching account from repository")

	svcStart := time.Now()
	account, snapErr := s.repo.GetAccountById(ctx, id)
	metrics.CacheDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		prefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefix)
		s.logger.Error(snapErr.ResponseMessage, zap.Error(prefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return nil, snapErr
	}

	span.SetAttributes(
		attribute.String("service.result.id", account.ID.String()),
	)

	s.logger.Info("success fetching account",
		zap.String("service.result.id", account.ID.String()),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return account, nil
}

// Fetch Transaction by Account Id
func (s *accountsService) FetchTransactionsByAccountId(ctx context.Context, id string, trxType string) ([]models.Transaction, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.GetTrxById")
	defer span.End()
	operation := "fetch_by_id"
	operationTrx := "fetch_trx_by_id"

	s.logger.Info("checking valid accound id")

	svcStart := time.Now()
	_, snapErr := s.repo.GetAccountById(ctx, id)
	metrics.CacheDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		prefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefix)
		s.logger.Error(snapErr.ResponseMessage, zap.Error(prefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return nil, snapErr
	}

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	s.logger.Info("fetching trx account from repository")

	svcStartTrx := time.Now()
	transactions, snapErr := s.repo.GetTransactionsByAccountId(ctx, id, trxType)
	metrics.CacheDuration.WithLabelValues(svcAccount, operationTrx).
		Observe(time.Since(svcStartTrx).Seconds())

	if snapErr != nil {
		prefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefix)
		s.logger.Error(snapErr.ResponseMessage, zap.Error(prefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return nil, snapErr
	}

	span.SetAttributes(
		attribute.Int("service.result.count", len(transactions)),
	)

	s.logger.Info("success fetching account",
		zap.Int("service.result.count", len(transactions)),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operationTrx, "success").Inc()

	return transactions, nil
}

// Create new account
func (s *accountsService) CreateNewAccount(ctx context.Context, account models.AccountCreateRequest, producer *kafka.Producer, svcCode string) (*models.AccountCreateResponse, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.Create")
	defer span.End()
	operation := "account_exist"
	operationCreate := "create"

	s.logger.Info("checking bank code")

	svcStartCheckBank := time.Now()
	_, snapErr := s.bankSvc.FetchBankById(ctx, account.BankCode)
	metrics.ServiceDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStartCheckBank).Seconds())

	if snapErr != nil {
		prefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefix)
		s.logger.Error(snapErr.ResponseMessage, zap.Error(prefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()

		// --- publish AccountFailedEvent ---
		failedEvent := kafka.AccountFailedEvent{
			PartnerReferenceNo: account.PartnerReferenceNo,
			CustomerID:         account.CustomerID,
			MerchantID:         account.MerchantID,
			PartnerID:          account.PartnerID,
			ExternalID:         account.ExternalID,
			ErrorCode:          snapErr.GetResponseCode(svcCode),
			HttpCode:           snapErr.HttpCode,
			ErrorMessage:       snapErr.ResponseMessage,
			FailedAt:           time.Now(),
		}

		if pubErr := producer.Publish(ctx, kafka.TopicAccountFailed, account.PartnerReferenceNo, failedEvent); pubErr != nil {
			s.logger.Error("failed to publish account.failed event", zap.Error(pubErr))
		}

		return nil, snapErr
	}
	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	// Serialisasi AdditionalInfo LENGKAP — sertakan semua field SNAP
	additionalInfoBytes, err := json.Marshal(models.TransactionAdditionalInfo{
		DeviceID: account.AdditionalInfo.DeviceId,
		Channel:  account.AdditionalInfo.Channel,
	})
	if err != nil {
		s.logger.Warn("failed to marshal additional info, using empty object", zap.Error(err))
		additionalInfoBytes = []byte("{}")
	}

	referenceNo := helper.GenerateReferenceNo()

	NewAccount := models.Account{
		BankCode:           account.BankCode,
		AccountNumber:      helper.GenerateAccountNumber(),
		CustomerID:         account.CustomerID,
		AccountHolder:      account.Name,
		Currency:           DEFAULT_CURRENCY,
		ReferenceNo:        referenceNo,
		PartnerReferenceNo: account.PartnerReferenceNo,
		Email:              account.Email,
		PhoneNo:            account.PhoneNo,
		CountryCode:        account.CountryCode,
		Lang:               account.Lang,
		Locale:             account.Locale,
		MerchantID:         account.MerchantID,
		SubMerchantID:      account.SubMerchantID,
		OnboardingPartner:  account.OnboardingPartner,
		TerminalType:       account.TerminalType,
		Scopes:             account.Scopes,
		RedirectURL:        account.RedirectURL,
		AdditionalInfo:     additionalInfoBytes,
	}

	s.logger.Info("creating new account")

	// Simpan ke repository
	svcStart := time.Now()
	returnedId, snapErr := s.repo.CreateAccount(ctx, NewAccount)
	metrics.ServiceDuration.WithLabelValues(svcAccount, operationCreate).
		Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		prefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefix)
		s.logger.Error(snapErr.ResponseMessage, zap.Error(prefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operationCreate, "error").Inc()

		// --- publish AccountFailedEvent ---
		failedEvent := kafka.AccountFailedEvent{
			PartnerReferenceNo: account.PartnerReferenceNo,
			CustomerID:         account.CustomerID,
			MerchantID:         account.MerchantID,
			PartnerID:          account.PartnerID,
			ExternalID:         account.ExternalID,
			ErrorCode:          snapErr.GetResponseCode(svcCode),
			HttpCode:           snapErr.HttpCode,
			ErrorMessage:       snapErr.ResponseMessage,
			FailedAt:           time.Now(),
		}

		if pubErr := producer.Publish(ctx, kafka.TopicAccountFailed, account.PartnerReferenceNo, failedEvent); pubErr != nil {
			s.logger.Error("failed to publish account.failed event", zap.Error(pubErr))
		}

		return nil, snapErr
	}

	NewAccount.ID = uuid.MustParse(returnedId)

	span.SetAttributes(
		attribute.String("service.result.id", returnedId),
	)

	s.logger.Info("success creating new account",
		zap.String("service.result.id", returnedId),
	)

	// --- publish AccountSucceedEvent ---

	authCode := helper.GenerateAuthCode()
	apiKey := helper.GenerateAPIKey()
	accountId := NewAccount.ID.String()

	createdEvent := kafka.AccountCreatedEvent{
		AccountID:          accountId,
		ReferenceNo:        NewAccount.ReferenceNo,
		PartnerReferenceNo: NewAccount.PartnerReferenceNo,
		Name:               account.Name,
		Email:              account.Email,
		CustomerID:         account.CustomerID,
		PartnerID:          account.PartnerID,
		ExternalID:         account.ExternalID,
		BankCode:           account.BankCode,
		AuthCode:           authCode,
		State:              account.State,
		CreatedAt:          time.Now(),
	}
	if pubErr := producer.Publish(ctx, kafka.TopicAccountCreated, NewAccount.ReferenceNo, createdEvent); pubErr != nil {
		s.logger.Error("failed to publish account.created event", zap.Error(pubErr))
	}

	accountResponse := models.AccountCreateResponse{
		ReferenceNo:        NewAccount.ReferenceNo,
		PartnerReferenceNo: NewAccount.PartnerReferenceNo,
		AuthCode:           authCode,
		APIKey:             apiKey,
		AccountID:          accountId,
		State:              account.State,
		AdditionalInfo:     account.AdditionalInfo,
	}

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operationCreate, "success").Inc()

	return &accountResponse, nil
}

// Update account
func (s *accountsService) PatchAccountById(ctx context.Context, account models.Account) (string, *models.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.Update")
	defer span.End()
	operation := "update"

	s.logger.Info("checking payload")

	// Jika tidak ada field yang diupdate
	if account.AccountHolder == "" && account.AccountNumber == "" {
		span.RecordError(models.ErrInvalidField)
		s.logger.Error(models.ErrInvalidField.Error(), zap.Error(models.ErrInvalidField))
		metrics.BusinessValidationErrors.WithLabelValues(svcAccount, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return "", &models.SnapMandatoryField
	}

	s.logger.Info("updating account data")

	svcStart := time.Now()
	getId, snapErr := s.repo.UpdateAccount(ctx, account)
	metrics.CacheDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		prefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefix)
		s.logger.Error(snapErr.ResponseMessage, zap.Error(prefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return "", snapErr
	}

	span.SetAttributes(
		attribute.String("service.result.id", getId),
	)

	s.logger.Info("success updating account data",
		zap.String("service.result.id", getId),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return getId, nil
}

// Delete account
func (s *accountsService) DeleteAccountById(ctx context.Context, id string) *models.SnapDetail {
	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.Delete")
	defer span.End()
	operation := "delete"

	s.logger.Info("deleting account data",
		zap.String("service.delete.id", id),
	)

	svcStart := time.Now()
	snapErr := s.repo.DeleteAccount(ctx, id)
	metrics.CacheDuration.WithLabelValues(svcAccount, operation).
		Observe(time.Since(svcStart).Seconds())

	if snapErr != nil {
		prefix := errors.New(snapErr.ResponseMessage)
		span.RecordError(prefix)
		s.logger.Error(snapErr.ResponseMessage, zap.Error(prefix))
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return snapErr
	}

	span.SetAttributes(
		attribute.String("service.delete.id", id),
	)

	s.logger.Info("success deleting account data",
		zap.String("service.delete.id", id),
	)

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return nil
}
