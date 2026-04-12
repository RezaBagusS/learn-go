package handler

import (
	"belajar-go/challenge/transactionSystem/config"
	"belajar-go/challenge/transactionSystem/dto"
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/accounts/repository"
	"belajar-go/challenge/transactionSystem/internal/api/accounts/service"
	bankRepository "belajar-go/challenge/transactionSystem/internal/api/banks/repository"
	bankService "belajar-go/challenge/transactionSystem/internal/api/banks/service"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"belajar-go/challenge/transactionSystem/observability/metrics"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
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
}

func NewAccountsHandler(mux *http.ServeMux, db *sqlx.DB, rdb *redis.Client) *AccountsHandler {

	keyManager := helper.NewRedisKeyManager("transaction_system", config.DOMAIN_ACCOUNT)
	idempotency := middleware.NewIdempotencyMiddleware(rdb, keyManager)
	bankRepo := bankRepository.NewBankRepository(db)
	bankSvc := bankService.NewBanksService(bankRepo)
	logger := helper.Log
	accountRepo := repository.NewAccountRepository(db)
	accountSvc := service.NewAccountsService(accountRepo, bankSvc)

	return &AccountsHandler{
		mux:         mux,
		svc:         accountSvc,
		rdb:         rdb,
		keyManager:  keyManager,
		idempotency: idempotency,
		logger:      logger,
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
		helper.NewAPIPath(http.MethodGet, version, "/account/{id}/transactions"),
		obs.Wrap("AccountHandler.GetTrx", config.DOMAIN_ACCOUNT, a.GetTransactions()).ServeHTTP,
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

		metrics.CacheDuration.WithLabelValues(
			"get",
			key,
		).Observe(cacheDuration)

		cacheSpan.End()

		if errRedis == nil {

			metrics.CacheRequestsTotal.WithLabelValues(
				key,
				"hit",
			).Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var accounts []models.Account
				if err := json.Unmarshal(decompressed, &accounts); err == nil {
					span.AddEvent("Cache hit occured")
					h.logger.Info("Cache Hit - Berhasil mengambil list data account",
						zap.String("source", "Redis"),
						zap.Int("count", len(accounts)),
					)
					dto.WriteResponse(
						w,
						models.SnapSuccess.HttpCode,
						models.SnapSuccess.GetResponseCode(svcCode),
						models.SnapSuccess.ResponseMessage,
						map[string]any{
							"accounts": accounts,
						},
					)
					return
				}
			}
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(
				key,
				"miss",
			).Inc()
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
			dto.WriteError(
				w,
				snapErr.HttpCode,
				snapErr.GetResponseCode(svcCode),
				snapErr.ResponseMessage,
			)
			return
		}

		span.SetAttributes(attribute.Int("result.count", len(accounts)))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, accounts); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}

		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		h.logger.Info("Berhasil mengambil list data akun",
			zap.String("source", "database"),
			zap.Int("count", len(accounts)),
		)

		dto.WriteResponse(
			w,
			models.SnapSuccess.HttpCode,
			models.SnapSuccess.GetResponseCode(svcCode),
			models.SnapSuccess.ResponseMessage,
			map[string]any{
				"accounts": accounts,
			},
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
			h.logger.Error(models.ErrInvalidUuid.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(
				w,
				models.SnapInvalidFormat.HttpCode,
				models.SnapInvalidFormat.GetResponseCode(svcCode),
				models.SnapInvalidFormat.ResponseMessage,
			)
			return
		}

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_ID, idParse.String())
		h.logger.Info("Checking cache",
			zap.String("key", cacheKey),
			zap.String("handler.query", idParse.String()),
		)

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()
		cacheDuration := time.Since(cacheStart).Seconds()

		metrics.CacheDuration.WithLabelValues(
			"get",
			key,
		).Observe(cacheDuration)

		cacheSpan.End()

		h.logger.Info("Mencari akun", zap.String("handler.account.id", idParse.String()))

		if errRedis == nil {

			metrics.CacheRequestsTotal.WithLabelValues(
				key,
				"hit",
			).Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var account models.Account
				if err := json.Unmarshal(decompressed, &account); err == nil {
					span.AddEvent("Cache hit occurred")
					h.logger.Info("Cache Hit - Berhasil mengambil data akun",
						zap.String("source", "redis"),
						zap.String("handler.result.id", account.ID.String()),
					)
					dto.WriteResponse(
						w,
						models.SnapSuccess.HttpCode,
						models.SnapSuccess.GetResponseCode(svcCode),
						models.SnapSuccess.ResponseMessage,
						map[string]any{
							"account": account,
						},
					)
					return
				}
			}
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(
				key,
				"miss",
			).Inc()
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
			dto.WriteError(
				w,
				snapErr.HttpCode,
				snapErr.GetResponseCode(svcCode),
				snapErr.ResponseMessage,
			)
			return
		}

		span.SetAttributes(attribute.String("handler.result.id", account.ID.String()))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, account); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		h.logger.Info("Berhasil mengambil data bank",
			zap.String("source", "database"),
			zap.String("handler.result.id", account.ID.String()),
		)

		dto.WriteResponse(
			w,
			models.SnapSuccess.HttpCode,
			models.SnapSuccess.GetResponseCode(svcCode),
			models.SnapSuccess.ResponseMessage,
			map[string]any{
				"account": account,
			},
		)
	}
}

// GET /account/{id}/transactions?type=all/in/out
func (h *AccountsHandler) GetTransactions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		trxTypeEnum := []string{"all", "in", "out"}
		key := "account_trx"
		svcCode := config.SVC_CODE_TRX_HISTORY_LIST

		idStr := r.PathValue("id")
		trxType := r.URL.Query().Get("type")

		if trxType == "" {
			trxType = "all"
		}

		h.logger.Info("Path & trx type received",
			zap.String("handler.query", idStr),
			zap.String("handler.trxType", trxType),
		)

		// Valid Uuid
		idParse, err := uuid.Parse(idStr)
		if err != nil {
			h.logger.Error(models.ErrInvalidUuid.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(
				w,
				models.SnapInvalidFormat.HttpCode,
				models.SnapInvalidFormat.GetResponseCode(svcCode),
				models.SnapInvalidFormat.ResponseMessage,
			)
			return
		}

		isValidType := slices.Contains(trxTypeEnum, trxType)

		// Valid Trx Type
		if !isValidType {
			h.logger.Error(models.ErrInvalidTrxType.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(
				w,
				models.SnapInvalidFormat.HttpCode,
				models.SnapInvalidFormat.GetResponseCode(svcCode),
				models.SnapInvalidFormat.ResponseMessage,
			)
			return
		}

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_TRANSACTION + ":" + idParse.String() + ":" + trxType)
		h.logger.Info("Checking cache",
			zap.String("key", cacheKey),
			zap.String("handler.query", idParse.String()),
			zap.String("handler.trxType", trxType),
		)

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()
		cacheDuration := time.Since(cacheStart).Seconds()

		metrics.CacheDuration.WithLabelValues(
			"get",
			key,
		).Observe(cacheDuration)

		cacheSpan.End()

		h.logger.Info("Mencari transaksi akun",
			zap.String("handler.account.id", idParse.String()),
			zap.String("handler.trxType", trxType),
		)

		if errRedis == nil {

			metrics.CacheRequestsTotal.WithLabelValues(
				key,
				"hit",
			).Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var transactions []models.Transaction
				if err := json.Unmarshal(decompressed, &transactions); err == nil {
					span.AddEvent("Cache hit occurred")
					h.logger.Info("Cache Hit - Berhasil mengambil data transaksi akun",
						zap.String("source", "redis"),
						zap.Int("handler.result.count", len(transactions)),
					)
					dto.WriteResponse(
						w,
						models.SnapSuccess.HttpCode,
						models.SnapSuccess.GetResponseCode(svcCode),
						models.SnapSuccess.ResponseMessage,
						map[string]any{
							"transactions": transactions,
						},
					)
					return
				}
			}
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(
				key,
				"miss",
			).Inc()
		}

		span.AddEvent("Cache miss")
		h.logger.Info("Cache miss", zap.String("key", cacheKey))

		// Exec
		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-Database")
		transactions, snapErr := h.svc.FetchTransactionsByAccountId(dbCtx, idStr, trxType)
		dbSpan.End()

		if snapErr != nil {
			prefixError := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixError.Error(), zap.Error(prefixError))
			span.RecordError(prefixError)
			dto.WriteError(
				w,
				snapErr.HttpCode,
				snapErr.GetResponseCode(svcCode),
				snapErr.ResponseMessage,
			)
			return
		}

		span.SetAttributes(attribute.Int("handler.result.count", len(transactions)))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transactions); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		h.logger.Info("Berhasil mengambil data transaksi akun",
			zap.String("source", "database"),
			zap.Int("handler.result.count", len(transactions)),
		)

		dto.WriteResponse(
			w,
			models.SnapSuccess.HttpCode,
			models.SnapSuccess.GetResponseCode(svcCode),
			models.SnapSuccess.ResponseMessage,
			map[string]any{
				"transactions": transactions,
			},
		)
	}
}

// POST /v1.0/registration-account-creation
func (h *AccountsHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		key := "account_list"
		svcCode := config.SVC_CODE_ACCOUNT_CREATION

		snapHeader := models.ExtractSNAPHeader(r)
		if snapHeader.Timestamp == "" || snapHeader.PartnerID == "" ||
			snapHeader.ExternalID == "" || snapHeader.ChannelID == "" {
			h.logger.Error(models.SnapUnauthorized.ResponseMessage,
				zap.String("x_timestamp", snapHeader.Timestamp),
				zap.String("x_partner_id", snapHeader.PartnerID),
				zap.String("x_external_id", snapHeader.ExternalID),
				zap.String("channel_id", snapHeader.ChannelID),
			)
			span.SetStatus(codes.Error, models.SnapUnauthorized.ResponseMessage)
			span.SetAttributes(attribute.String("snap.error", "header_tidak_lengkap"))
			dto.WriteError(
				w,
				models.SnapUnauthorized.HttpCode,
				models.SnapUnauthorized.GetResponseCode(svcCode),
				models.SnapUnauthorized.ResponseMessage,
			)
			return
		}

		span.SetAttributes(
			attribute.String("snap.partner_id", snapHeader.PartnerID),
			attribute.String("snap.external_id", snapHeader.ExternalID),
			attribute.String("snap.channel_id", snapHeader.ChannelID),
			attribute.String("snap.timestamp", snapHeader.Timestamp),
		)

		// --- Decode request body ---
		var payload models.AccountCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.logger.Error(models.ErrInvalidJsonFormat.Error(),
				zap.Error(err),
				zap.String("x_partner_id", snapHeader.PartnerID),
				zap.String("x_external_id", snapHeader.ExternalID),
			)
			span.RecordError(err)
			span.SetStatus(codes.Error, models.ErrInvalidJsonFormat.Error())
			dto.WriteError(
				w,
				models.SnapInvalidFormat.HttpCode,
				models.SnapInvalidFormat.GetResponseCode(svcCode),
				models.SnapInvalidFormat.ResponseMessage,
			)
			return
		}

		// --- Validasi field wajib (SNAP Mandatory Fields) ---
		if payload.PartnerReferenceNo == "" ||
			payload.CustomerID == "" ||
			payload.Name == "" ||
			payload.PhoneNo == "" ||
			payload.Email == "" ||
			payload.OnboardingPartner == "" ||
			payload.MerchantID == "" ||
			payload.RedirectURL == "" ||
			payload.State == "" {

			h.logger.Error("Field wajib tidak lengkap",
				zap.String("partner_ref_no", payload.PartnerReferenceNo),
				zap.String("customer_id", payload.CustomerID),
				zap.String("merchant_id", payload.MerchantID),
			)

			span.SetStatus(codes.Error, "mandatory field is missing")

			dto.WriteError(
				w,
				models.SnapMandatoryField.HttpCode,
				models.SnapMandatoryField.GetResponseCode(svcCode),
				models.SnapMandatoryField.ResponseMessage,
			)
			return
		}

		h.logger.Info("Payload registration account creation diterima",
			zap.String("partner_reference_no", payload.PartnerReferenceNo),
			zap.String("customer_id", payload.CustomerID),
			zap.String("customer_name", payload.Name),
			zap.String("phone_no", payload.PhoneNo),
			zap.String("email", payload.Email),
			zap.String("onboarding_partner", payload.OnboardingPartner),
			zap.String("merchant_id", payload.MerchantID),
			zap.String("redirect_url", payload.RedirectURL),
		)

		svcCtx, svcSpan := tracer.Start(ctx, "Create-Account")
		svcSpan.SetAttributes(
			attribute.String("db.partner_reference_no", payload.PartnerReferenceNo),
			attribute.String("db.bank_code", payload.BankCode),
			attribute.String("db.account_holder", payload.Name),
			attribute.String("db.account_number", payload.CustomerID),
			attribute.String("db.email", payload.Email),
			attribute.String("db.phone_no", payload.PhoneNo),
			attribute.String("db.merchant_id", payload.MerchantID),
		)
		newAccount, snapErr := h.svc.CreateNewAccount(svcCtx, payload)
		svcSpan.End()

		if snapErr != nil {
			prefixError := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixError.Error(), zap.Error(prefixError))
			span.RecordError(prefixError)
			dto.WriteError(
				w,
				snapErr.HttpCode,
				snapErr.GetResponseCode(svcCode),
				snapErr.ResponseMessage,
			)
			return
		}

		// Invalidate Existing Cache
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_LIST)
		cacheStart := time.Now()
		if err := h.rdb.Del(ctx, cacheKey).Err(); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues(key, "error").Inc()
			span.RecordError(err)
			span.SetStatus(codes.Error, models.ErrRedisInvalidate.Error())
			h.logger.Error(models.ErrRedisInvalidate.Error(), zap.Error(err))
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(key, "invalidate").Inc()
			span.AddEvent("Cache Invalidated")
		}

		metrics.CacheDuration.WithLabelValues("invalidate", key).
			Observe(time.Since(cacheStart).Seconds())

		span.SetAttributes(attribute.String("handler.result.id", newAccount.AccountID))

		h.logger.Info("Berhasil membuat data akun baru",
			zap.String("source", "database"),
			zap.String("handler.result.id", newAccount.AccountID),
		)

		responseBody := models.AccountCreateResponse{
			ReferenceNo:        newAccount.ReferenceNo,
			PartnerReferenceNo: newAccount.PartnerReferenceNo,
			AuthCode:           newAccount.AuthCode,
			APIKey:             newAccount.APIKey,
			AccountID:          newAccount.AccountID,
			State:              newAccount.State,
			AdditionalInfo:     newAccount.AdditionalInfo,
		}

		dto.WriteResponse(
			w,
			models.SnapSuccess.HttpCode,
			models.SnapSuccess.GetResponseCode(svcCode),
			models.SnapSuccess.ResponseMessage, responseBody,
		)
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

		// Valid Uuid
		idParse, err := uuid.Parse(idStr)
		if err != nil {
			h.logger.Error(models.ErrInvalidUuid.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(
				w,
				models.SnapInvalidFormat.HttpCode,
				models.SnapInvalidFormat.GetResponseCode(svcCode),
				models.SnapInvalidFormat.ResponseMessage,
			)
			return
		}

		var payload models.Account
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.logger.Error(models.ErrInvalidJsonFormat.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(
				w,
				models.SnapInvalidFormat.HttpCode,
				models.SnapInvalidFormat.GetResponseCode(svcCode),
				models.SnapInvalidFormat.ResponseMessage,
			)
			return
		}

		h.logger.Info("Payload received", zap.Any("payload", payload))

		payload.ID = idParse

		dbCtx, dbSpan := tracer.Start(ctx, "Update-Account")
		updatedId, snapErr := h.svc.PatchAccountById(dbCtx, payload)
		dbSpan.End()

		if snapErr != nil {
			prefixError := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixError.Error(), zap.Error(prefixError))
			span.RecordError(prefixError)
			dto.WriteError(
				w,
				snapErr.HttpCode,
				snapErr.GetResponseCode(svcCode),
				snapErr.ResponseMessage,
			)
			return
		}

		// Invalidate Existing Cache
		cacheKeyList := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_LIST)
		cacheKeyId := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_ID, updatedId)

		cacheStart := time.Now()
		if err := h.rdb.Del(ctx, cacheKeyList, cacheKeyId).Err(); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues(keyList, "error").Inc()
			metrics.CacheRequestsTotal.WithLabelValues(keyId, "error").Inc()
			span.RecordError(err)
			span.SetStatus(codes.Error, models.ErrRedisInvalidate.Error())
			h.logger.Error(models.ErrRedisInvalidate.Error(), zap.Error(err))
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(keyList, "invalidate").Inc()
			metrics.CacheRequestsTotal.WithLabelValues(keyId, "invalidate").Inc()
			span.AddEvent("Cache Invalidated")
		}
		metrics.CacheDuration.WithLabelValues("invalidate", keyList).
			Observe(time.Since(cacheStart).Seconds())
		metrics.CacheDuration.WithLabelValues("invalidate", keyId).
			Observe(time.Since(cacheStart).Seconds())

		span.SetAttributes(attribute.String("handler.result.id", updatedId))

		h.logger.Info("Berhasil memperbarui data akun",
			zap.String("source", "database"),
			zap.String("handler.result.id", updatedId),
		)

		dto.WriteResponse(
			w,
			models.SnapSuccess.HttpCode,
			models.SnapSuccess.GetResponseCode(svcCode),
			fmt.Sprintf("Berhasil mengupdate akun dengan id : %s", idStr),
			map[string]any{
				"id": updatedId,
			},
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

		// Valid Uuid
		idParse, errId := uuid.Parse(idStr)
		if errId != nil {
			h.logger.Error(models.ErrInvalidUuid.Error(), zap.Error(errId))
			span.RecordError(errId)
			dto.WriteError(
				w,
				models.SnapInvalidFormat.HttpCode,
				models.SnapInvalidFormat.GetResponseCode(svcCode),
				models.ErrInvalidUuid.Error(),
			)
			return
		}

		dbCtx, dbSpan := tracer.Start(ctx, "Delete-Account")
		snapErr := h.svc.DeleteAccountById(dbCtx, idParse.String())
		dbSpan.End()

		if snapErr != nil {
			prefixErr := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixErr.Error(), zap.Error(prefixErr))
			span.RecordError(prefixErr)
			dto.WriteError(
				w,
				snapErr.HttpCode,
				snapErr.GetResponseCode(svcCode),
				snapErr.ResponseMessage,
			)
			return
		}

		// Invalidate Existing Cache
		cacheKeyList := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_LIST)
		cacheKeyId := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_ID, idParse.String())

		cacheStart := time.Now()
		if err := h.rdb.Del(ctx, cacheKeyList, cacheKeyId).Err(); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues(keyList, "error").Inc()
			metrics.CacheRequestsTotal.WithLabelValues(keyId, "error").Inc()
			span.RecordError(err)
			span.SetStatus(codes.Error, models.ErrRedisInvalidate.Error())
			h.logger.Error(models.ErrRedisInvalidate.Error(), zap.Error(err))
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(keyList, "invalidate").Inc()
			metrics.CacheRequestsTotal.WithLabelValues(keyId, "invalidate").Inc()
			span.AddEvent("Cache Invalidated")
		}
		metrics.CacheDuration.WithLabelValues("invalidate", keyList).
			Observe(time.Since(cacheStart).Seconds())
		metrics.CacheDuration.WithLabelValues("invalidate", keyId).
			Observe(time.Since(cacheStart).Seconds())

		span.SetAttributes(attribute.String("handler.delete.id", idParse.String()))
		h.logger.Info("Berhasil menghapus data akun",
			zap.String("source", "database"),
			zap.String("handler.delete.id", idParse.String()),
		)

		dto.WriteResponse(
			w,
			models.SnapSuccess.HttpCode,
			models.SnapSuccess.GetResponseCode(svcCode),
			fmt.Sprintf("Berhasil menghapus akun dengan id : %s", idStr),
			map[string]any{},
		)
	}
}
