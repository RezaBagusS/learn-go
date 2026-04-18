package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"my-microservices/account-service/config"
	"my-microservices/account-service/helper"
	"my-microservices/account-service/internal/domain"
	"my-microservices/account-service/internal/dto"
	"my-microservices/account-service/internal/kafka"
	"my-microservices/account-service/internal/middleware"
	"my-microservices/account-service/internal/repository"
	"my-microservices/account-service/internal/service"
	"my-microservices/account-service/observability/metrics"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type AccountsHandler struct {
	mux         *http.ServeMux
	svc         service.AccountsService
	rdb         *redis.Client
	keyManager  *helper.RedisKeyManager
	idempotency *middleware.IdempotencyMiddleware
	logger      *zap.Logger
	producer    *kafka.Producer
}

func NewAccountsHandler(mux *http.ServeMux, db *sqlx.DB, rdb *redis.Client, producer *kafka.Producer) *AccountsHandler {

	keyManager := helper.NewRedisKeyManager("account_service", config.DOMAIN_ACCOUNT)
	idempotency := middleware.NewIdempotencyMiddleware(rdb, keyManager)
	logger := helper.Log
	accountRepo := repository.NewAccountRepository(db)
	accountSvc := service.NewAccountsService(accountRepo)

	return &AccountsHandler{
		mux:         mux,
		svc:         accountSvc,
		rdb:         rdb,
		keyManager:  keyManager,
		idempotency: idempotency,
		logger:      logger,
		producer:    producer,
	}
}

func (a *AccountsHandler) MapRoutes(obs *middleware.ObservabilityMiddleware) {

	version := "v1.0"

	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, version, "/accounts"),
		obs.Wrap("AccountHandler.GetAll", config.DOMAIN_ACCOUNT, a.GetAll()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, version, "/account/{id}"),
		obs.Wrap("AccountHandler.GetById", config.DOMAIN_ACCOUNT, a.GetById()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, version, "/registration-account-creation"),
		middleware.ValidateSNAPToken(
			obs.Wrap("AccountHandler.Create", config.DOMAIN_ACCOUNT, a.idempotency.Check(a.Create())),
		).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPatch, version, "/account/{id}"),
		obs.Wrap("AccountHandler.Update", config.DOMAIN_ACCOUNT, a.idempotency.Check(a.Update())).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodDelete, version, "/account/{id}"),
		obs.Wrap("AccountHandler.Delete", config.DOMAIN_ACCOUNT, a.Delete()).ServeHTTP,
	)
}

// GET /accounts
func (h *AccountsHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		key := "account_list"
		svcCode := config.SVC_CODE_ACCOUNT_INQUIRY

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_LIST)
		h.logger.Info("Checking cache", zap.String("key", cacheKey))

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")

		cacheStart := time.Now()
		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()
		cacheDuration := time.Since(cacheStart).Seconds()

		metrics.CacheDuration.WithLabelValues("get", key).Observe(cacheDuration)

		cacheSpan.End()

		if errRedis == nil {

			metrics.CacheRequestsTotal.WithLabelValues(key, "hit").Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var accounts []domain.Account
				if err := json.Unmarshal(decompressed, &accounts); err == nil {
					span.AddEvent("Cache hit occured")
					h.logger.Info("Cache Hit - Berhasil mengambil list data account",
						zap.String("source", "Redis"),
						zap.Int("count", len(accounts)),
					)
					dto.WriteResponse(
						w,
						domain.SnapSuccess.HttpCode,
						domain.SnapSuccess.GetResponseCode(svcCode),
						domain.SnapSuccess.ResponseMessage,
						map[string]any{"accounts": accounts},
					)
					return
				}
			}
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(key, "miss").Inc()
		}

		span.AddEvent("Cache miss")
		h.logger.Info("Cache miss", zap.String("key", cacheKey))

		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-database")
		accounts, snapErr := h.svc.FetchAllAccounts(dbCtx)
		dbSpan.End()

		if snapErr != nil {
			prefixError := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixError.Error(), zap.Error(prefixError))
			span.RecordError(prefixError)
			dto.WriteError(w, snapErr.HttpCode, snapErr.GetResponseCode(svcCode), snapErr.ResponseMessage)
			return
		}

		span.SetAttributes(attribute.Int("result.count", len(accounts)))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, accounts); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).Observe(time.Since(cacheSetStart).Seconds())

		h.logger.Info("Berhasil mengambil list data akun",
			zap.String("source", "database"),
			zap.Int("count", len(accounts)),
		)

		dto.WriteResponse(
			w,
			domain.SnapSuccess.HttpCode,
			domain.SnapSuccess.GetResponseCode(svcCode),
			domain.SnapSuccess.ResponseMessage,
			map[string]any{"accounts": accounts},
		)
	}
}

// GET /account/{id}
func (h *AccountsHandler) GetById() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		key := "account_id"
		svcCode := config.SVC_CODE_ACCOUNT_INQUIRY_INTERNAL

		idStr := r.PathValue("id")
		h.logger.Info("Path received", zap.String("handler.query", idStr))

		idParse, err := uuid.Parse(idStr)
		if err != nil {
			h.logger.Error(domain.ErrInvalidUuid.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, domain.SnapInvalidFormat.HttpCode, domain.SnapInvalidFormat.GetResponseCode(svcCode), domain.SnapInvalidFormat.ResponseMessage)
			return
		}

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_ID, idParse.String())
		h.logger.Info("Checking cache", zap.String("key", cacheKey))

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()
		cacheDuration := time.Since(cacheStart).Seconds()

		metrics.CacheDuration.WithLabelValues("get", key).Observe(cacheDuration)
		cacheSpan.End()

		if errRedis == nil {
			metrics.CacheRequestsTotal.WithLabelValues(key, "hit").Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var account domain.Account
				if err := json.Unmarshal(decompressed, &account); err == nil {
					span.AddEvent("Cache hit occurred")
					h.logger.Info("Cache Hit - Berhasil mengambil data akun",
						zap.String("source", "redis"),
						zap.String("handler.result.id", account.ID.String()),
					)
					dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode), domain.SnapSuccess.ResponseMessage, map[string]any{"account": account})
					return
				}
			}
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(key, "miss").Inc()
		}

		span.AddEvent("Cache miss")
		h.logger.Info("Cache miss", zap.String("key", cacheKey))

		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-Database")
		account, snapErr := h.svc.FetchAccountById(dbCtx, idParse.String())
		dbSpan.End()

		if snapErr != nil {
			prefixError := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixError.Error(), zap.Error(prefixError))
			span.RecordError(prefixError)
			dto.WriteError(w, snapErr.HttpCode, snapErr.GetResponseCode(svcCode), snapErr.ResponseMessage)
			return
		}

		span.SetAttributes(attribute.String("handler.result.id", account.ID.String()))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, account); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).Observe(time.Since(cacheSetStart).Seconds())

		h.logger.Info("Berhasil mengambil data akun",
			zap.String("source", "database"),
			zap.String("handler.result.id", account.ID.String()),
		)

		dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode), domain.SnapSuccess.ResponseMessage, map[string]any{"account": account})
	}
}

// POST /v1.0/registration-account-creation
func (h *AccountsHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		key := "account_list"
		svcCode := config.SVC_CODE_ACCOUNT_CREATION

		snapHeader := domain.ExtractSNAPHeader(r)
		if snapHeader.Timestamp == "" || snapHeader.PartnerID == "" ||
			snapHeader.ExternalID == "" || snapHeader.ChannelID == "" {
			h.logger.Error(domain.SnapUnauthorized.ResponseMessage)
			span.SetStatus(codes.Error, domain.SnapUnauthorized.ResponseMessage)
			dto.WriteError(w, domain.SnapUnauthorized.HttpCode, domain.SnapUnauthorized.GetResponseCode(svcCode), "Header tidak lengkap")
			return
		}

		span.SetAttributes(
			attribute.String("snap.partner_id", snapHeader.PartnerID),
			attribute.String("snap.external_id", snapHeader.ExternalID),
		)

		var payload domain.AccountCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.logger.Error(domain.ErrInvalidJsonFormat.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, domain.SnapInvalidFormat.HttpCode, domain.SnapInvalidFormat.GetResponseCode(svcCode), "Format payload tidak valid")
			return
		}

		if payload.PartnerReferenceNo == "" || payload.CustomerID == "" || payload.Name == "" ||
			payload.PhoneNo == "" || payload.Email == "" || payload.OnboardingPartner == "" ||
			payload.MerchantID == "" || payload.RedirectURL == "" || payload.State == "" {

			h.logger.Error("Field wajib tidak lengkap")
			span.SetStatus(codes.Error, "mandatory field is missing")
			dto.WriteError(w, domain.SnapMandatoryField.HttpCode, domain.SnapMandatoryField.GetResponseCode(svcCode), "Payload tidak lengkap")
			return
		}

		payload.ExternalID = snapHeader.ExternalID
		payload.PartnerID = snapHeader.PartnerID

		svcCtx, svcSpan := tracer.Start(ctx, "Create-Account")
		newAccount, snapErr := h.svc.CreateNewAccount(svcCtx, payload, h.producer, svcCode)
		svcSpan.End()

		if snapErr != nil {
			prefixError := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixError.Error(), zap.Error(prefixError))
			span.RecordError(prefixError)
			span.SetStatus(codes.Error, snapErr.ResponseMessage)
			dto.WriteError(w, snapErr.HttpCode, snapErr.GetResponseCode(svcCode), snapErr.ResponseMessage)
			return
		}

		// Invalidate Cache
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_LIST)
		cacheStart := time.Now()
		if err := h.rdb.Del(ctx, cacheKey).Err(); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues(key, "error").Inc()
			span.RecordError(err)
			h.logger.Error(domain.ErrRedisInvalidate.Error(), zap.Error(err))
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(key, "invalidate").Inc()
			span.AddEvent("Cache Invalidated")
		}
		metrics.CacheDuration.WithLabelValues("invalidate", key).Observe(time.Since(cacheStart).Seconds())

		span.SetAttributes(attribute.String("handler.result.id", newAccount.AccountID))

		responseBody := domain.AccountCreateResponse{
			ReferenceNo:        newAccount.ReferenceNo,
			PartnerReferenceNo: newAccount.PartnerReferenceNo,
			AuthCode:           newAccount.AuthCode,
			APIKey:             newAccount.APIKey,
			AccountID:          newAccount.AccountID,
			State:              newAccount.State,
			AdditionalInfo:     newAccount.AdditionalInfo,
		}

		dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode), "Berhasil membuat data akun", responseBody)
	}
}

// PATCH /account/{id}
func (h *AccountsHandler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		keyList := "account_list"
		keyId := "account_id"
		svcCode := config.SVC_CODE_ACCOUNT_BIND

		idStr := r.PathValue("id")
		h.logger.Info("Path received", zap.String("handler.query", idStr))

		idParse, err := uuid.Parse(idStr)
		if err != nil {
			h.logger.Error(domain.ErrInvalidUuid.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, domain.SnapInvalidFormat.HttpCode, domain.SnapInvalidFormat.GetResponseCode(svcCode), domain.SnapInvalidFormat.ResponseMessage)
			return
		}

		var payload domain.Account
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.logger.Error(domain.ErrInvalidJsonFormat.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, domain.SnapInvalidFormat.HttpCode, domain.SnapInvalidFormat.GetResponseCode(svcCode), domain.SnapInvalidFormat.ResponseMessage)
			return
		}

		payload.ID = idParse

		dbCtx, dbSpan := tracer.Start(ctx, "Update-Account")
		updatedId, snapErr := h.svc.PatchAccountById(dbCtx, payload)
		dbSpan.End()

		if snapErr != nil {
			prefixError := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixError.Error(), zap.Error(prefixError))
			span.RecordError(prefixError)
			dto.WriteError(w, snapErr.HttpCode, snapErr.GetResponseCode(svcCode), snapErr.ResponseMessage)
			return
		}

		// Invalidate Cache
		cacheKeyList := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_LIST)
		cacheKeyId := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_ID, updatedId)

		cacheStart := time.Now()
		if err := h.rdb.Del(ctx, cacheKeyList, cacheKeyId).Err(); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues(keyList, "error").Inc()
			metrics.CacheRequestsTotal.WithLabelValues(keyId, "error").Inc()
			span.RecordError(err)
			h.logger.Error(domain.ErrRedisInvalidate.Error(), zap.Error(err))
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(keyList, "invalidate").Inc()
			metrics.CacheRequestsTotal.WithLabelValues(keyId, "invalidate").Inc()
			span.AddEvent("Cache Invalidated")
		}
		metrics.CacheDuration.WithLabelValues("invalidate", keyList).Observe(time.Since(cacheStart).Seconds())
		metrics.CacheDuration.WithLabelValues("invalidate", keyId).Observe(time.Since(cacheStart).Seconds())

		span.SetAttributes(attribute.String("handler.result.id", updatedId))
		h.logger.Info("Berhasil memperbarui data akun", zap.String("handler.result.id", updatedId))

		dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
			fmt.Sprintf("Berhasil mengupdate akun dengan id : %s", idStr),
			map[string]any{"id": updatedId},
		)
	}
}

// DELETE /account/{id}
func (h *AccountsHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		keyList := "account_list"
		keyId := "account_id"
		svcCode := config.SVC_CODE_ACCOUNT_UNBIND

		idStr := r.PathValue("id")
		h.logger.Info("Path received", zap.String("handler.query", idStr))

		idParse, errId := uuid.Parse(idStr)
		if errId != nil {
			h.logger.Error(domain.ErrInvalidUuid.Error(), zap.Error(errId))
			span.RecordError(errId)
			dto.WriteError(w, domain.SnapInvalidFormat.HttpCode, domain.SnapInvalidFormat.GetResponseCode(svcCode), domain.ErrInvalidUuid.Error())
			return
		}

		dbCtx, dbSpan := tracer.Start(ctx, "Delete-Account")
		snapErr := h.svc.DeleteAccountById(dbCtx, idParse.String())
		dbSpan.End()

		if snapErr != nil {
			prefixErr := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixErr.Error(), zap.Error(prefixErr))
			span.RecordError(prefixErr)
			dto.WriteError(w, snapErr.HttpCode, snapErr.GetResponseCode(svcCode), snapErr.ResponseMessage)
			return
		}

		// Invalidate Cache
		cacheKeyList := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_LIST)
		cacheKeyId := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_ID, idParse.String())

		cacheStart := time.Now()
		if err := h.rdb.Del(ctx, cacheKeyList, cacheKeyId).Err(); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues(keyList, "error").Inc()
			metrics.CacheRequestsTotal.WithLabelValues(keyId, "error").Inc()
			span.RecordError(err)
			h.logger.Error(domain.ErrRedisInvalidate.Error(), zap.Error(err))
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(keyList, "invalidate").Inc()
			metrics.CacheRequestsTotal.WithLabelValues(keyId, "invalidate").Inc()
			span.AddEvent("Cache Invalidated")
		}
		metrics.CacheDuration.WithLabelValues("invalidate", keyList).Observe(time.Since(cacheStart).Seconds())
		metrics.CacheDuration.WithLabelValues("invalidate", keyId).Observe(time.Since(cacheStart).Seconds())

		h.logger.Info("Berhasil menghapus data akun", zap.String("handler.delete.id", idParse.String()))

		dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
			fmt.Sprintf("Berhasil menghapus akun dengan id : %s", idStr),
			map[string]any{},
		)
	}
}
