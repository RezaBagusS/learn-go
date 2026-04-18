package service

import (
	"context"
	"encoding/json"
	"errors"
	"my-microservices/account-service/helper"
	"my-microservices/account-service/internal/domain"
	"my-microservices/account-service/internal/kafka"
	"my-microservices/account-service/internal/middleware"
	"my-microservices/account-service/internal/repository"
	"my-microservices/account-service/observability/metrics"
	"os"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type AccountsService interface {
	FetchAllAccounts(ctx context.Context) ([]domain.Account, *domain.SnapDetail)
	FetchAccountById(ctx context.Context, id string) (*domain.Account, *domain.SnapDetail)
	CreateNewAccount(ctx context.Context, account domain.AccountCreateRequest, producer *kafka.Producer, svcCode string) (*domain.AccountCreateResponse, *domain.SnapDetail)
	PatchAccountById(ctx context.Context, account domain.Account) (string, *domain.SnapDetail)
	DeleteAccountById(ctx context.Context, id string) *domain.SnapDetail
}

type accountsService struct {
	repo   repository.AccountRepository
	logger *zap.Logger
}

func NewAccountsService(repo repository.AccountRepository) AccountsService {
	logger := helper.Log

	return &accountsService{
		repo:   repo,
		logger: logger,
	}
}

const svcAccount = "account"
const DEFAULT_CURRENCY = "IDR"

// Fetch All Data
func (s *accountsService) FetchAllAccounts(ctx context.Context) ([]domain.Account, *domain.SnapDetail) {

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

	span.SetAttributes(attribute.Int("service.result.count", len(accounts)))

	s.logger.Info("success fetching accounts", zap.Int("count", len(accounts)))

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return accounts, nil
}

// Fetch Account by Id
func (s *accountsService) FetchAccountById(ctx context.Context, id string) (*domain.Account, *domain.SnapDetail) {

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

	span.SetAttributes(attribute.String("service.result.id", account.ID.String()))

	s.logger.Info("success fetching account", zap.String("service.result.id", account.ID.String()))

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return account, nil
}

// Create new account
func (s *accountsService) CreateNewAccount(ctx context.Context, account domain.AccountCreateRequest, producer *kafka.Producer, svcCode string) (*domain.AccountCreateResponse, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.Create")
	defer span.End()
	operationCreate := "create"

	s.logger.Info("creating new account")

	// Serialisasi AdditionalInfo
	additionalInfoBytes, err := json.Marshal(domain.TransactionAdditionalInfo{
		DeviceID: account.AdditionalInfo.DeviceId,
		Channel:  account.AdditionalInfo.Channel,
	})
	if err != nil {
		s.logger.Warn("failed to marshal additional info, using empty object", zap.Error(err))
		additionalInfoBytes = []byte("{}")
	}

	referenceNo := helper.GenerateReferenceNo()

	NewAccount := domain.Account{
		BankID:             account.BankID,
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

		// publish AccountFailedEvent
		failedEvent := kafka.AccountFailedEvent{
			PartnerReferenceNo: account.PartnerReferenceNo,
			CustomerID:         account.CustomerID,
			MerchantID:         account.MerchantID,
			PartnerID:          account.PartnerID,
			ExternalID:         account.ExternalID,
			ErrorCode:          snapErr.GetResponseCode(svcCode),
			HttpCode:           snapErr.HttpCode,
			ErrorMessage:       snapErr.ResponseMessage,
			CallbackURL:        os.Getenv("CALLBACK_URL"),
			FailedAt:           time.Now(),
		}

		if pubErr := producer.Publish(ctx, kafka.TopicAccountFailed, account.PartnerReferenceNo, failedEvent); pubErr != nil {
			s.logger.Error("failed to publish account.failed event", zap.Error(pubErr))
		}

		return nil, snapErr
	}

	NewAccount.ID = uuid.MustParse(returnedId)

	span.SetAttributes(attribute.String("service.result.id", returnedId))

	s.logger.Info("success creating new account", zap.String("service.result.id", returnedId))

	// publish AccountCreatedEvent
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
		BankID:             account.BankID,
		AuthCode:           authCode,
		CallbackURL:        os.Getenv("CALLBACK_URL"),
		State:              account.State,
		CreatedAt:          time.Now(),
	}
	if pubErr := producer.Publish(ctx, kafka.TopicAccountCreated, NewAccount.ReferenceNo, createdEvent); pubErr != nil {
		s.logger.Error("failed to publish account.created event", zap.Error(pubErr))
	}

	accountResponse := domain.AccountCreateResponse{
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
func (s *accountsService) PatchAccountById(ctx context.Context, account domain.Account) (string, *domain.SnapDetail) {

	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.Update")
	defer span.End()
	operation := "update"

	s.logger.Info("checking payload")

	if account.AccountHolder == "" && account.AccountNumber == "" {
		span.RecordError(domain.ErrInvalidField)
		s.logger.Error(domain.ErrInvalidField.Error(), zap.Error(domain.ErrInvalidField))
		metrics.BusinessValidationErrors.WithLabelValues(svcAccount, operation).Inc()
		metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "error").Inc()
		return "", &domain.SnapMandatoryField
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

	span.SetAttributes(attribute.String("service.result.id", getId))

	s.logger.Info("success updating account data", zap.String("service.result.id", getId))

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return getId, nil
}

// Delete account
func (s *accountsService) DeleteAccountById(ctx context.Context, id string) *domain.SnapDetail {
	tracer := middleware.TracerFromCtx(ctx)
	ctx, span := tracer.Start(ctx, "AccountService.Delete")
	defer span.End()
	operation := "delete"

	s.logger.Info("deleting account data", zap.String("service.delete.id", id))

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

	span.SetAttributes(attribute.String("service.delete.id", id))

	s.logger.Info("success deleting account data", zap.String("service.delete.id", id))

	metrics.ServiceRequestsTotal.WithLabelValues(svcAccount, operation, "success").Inc()

	return nil
}
