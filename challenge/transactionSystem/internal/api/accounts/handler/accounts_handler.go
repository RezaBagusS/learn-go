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
}

func NewAccountsHandler(mux *http.ServeMux, db *sqlx.DB, rdb *redis.Client) *AccountsHandler {

	keyManager := helper.NewRedisKeyManager("transaction_system", config.DOMAIN_ACCOUNT)
	idempotency := middleware.NewIdempotencyMiddleware(rdb, keyManager)
	bankRepo := bankRepository.NewBankRepository(db)
	bankSvc := bankService.NewBanksService(bankRepo)

	accountRepo := repository.NewAccountRepository(db)
	accountSvc := service.NewAccountsService(accountRepo, bankSvc)

	return &AccountsHandler{
		mux:         mux,
		svc:         accountSvc,
		rdb:         rdb,
		keyManager:  keyManager,
		idempotency: idempotency,
	}
}

func (a *AccountsHandler) MapRoutes(obs *middleware.ObservabilityMiddleware) {
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/accounts"),
		obs.Wrap("AccountHandler.GetAll", config.DOMAIN_ACCOUNT, a.GetAll()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/account/{id}"),
		obs.Wrap("AccountHandler.GetById", config.DOMAIN_ACCOUNT, a.GetById()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/account/{id}/transactions"),
		obs.Wrap("AccountHandler.GetTrx", config.DOMAIN_ACCOUNT, a.GetTransactions()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, "/account"),
		obs.Wrap("AccountHandler.Create", config.DOMAIN_ACCOUNT, a.idempotency.Check(a.Create())).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPatch, "/account/{id}"),
		obs.Wrap("AccountHandler.Update", config.DOMAIN_ACCOUNT, a.idempotency.Check(a.Update())).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodDelete, "/account/{id}"),
		obs.Wrap("AccountHandler.Delete", config.DOMAIN_ACCOUNT, a.Delete()).ServeHTTP,
	)
}

// GET /accounts
func (h *AccountsHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)
		key := "account_list"

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_LIST)
		logger.Info("Checking cache", zap.String("key", cacheKey))

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
					logger.Info("Cache Hit - Berhasil mengambil list data account",
						zap.String("source", "Redis"),
						zap.Int("count", len(accounts)),
					)
					dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil list data account", map[string]any{
						"accounts": accounts,
					})
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
		logger.Info("Cache miss", zap.String("key", cacheKey))

		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-database")
		accounts, err := h.svc.FetchAllAccounts(dbCtx)
		dbSpan.End()

		if err != nil {
			logger.Error("Database fetch failed", zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		span.SetAttributes(attribute.Int("result.count", len(accounts)))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, accounts); err != nil {
			logger.Warn("Failed to save to cache", zap.Error(err))
		}

		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		logger.Info("Berhasil mengambil list data akun",
			zap.String("source", "database"),
			zap.Int("count", len(accounts)),
		)

		dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil list data akun", map[string]any{
			"accounts": accounts,
		})
	}
}

// GET /account/{id}
func (h *AccountsHandler) GetById() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)
		key := "account_id"

		idStr := r.PathValue("id")
		logger.Info("Path received", zap.String("handler.query", idStr))

		idParse, err := uuid.Parse(idStr)
		if err != nil {
			logger.Error(models.ErrInvalidUuid.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_ID, idParse.String())
		logger.Info("Checking cache",
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

		logger.Info("Mencari akun", zap.String("handler.account.id", idParse.String()))

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
					logger.Info("Cache Hit - Berhasil mengambil data akun",
						zap.String("source", "redis"),
						zap.String("handler.result.id", account.ID.String()),
					)
					dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data akun dengan id = %s", idParse), map[string]any{
						"account": account,
					})
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
		logger.Info("Cache miss", zap.String("key", cacheKey))

		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-Database")
		account, err := h.svc.FetchAccountById(dbCtx, idParse.String())
		dbSpan.End()

		if err != nil {
			logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		span.SetAttributes(attribute.String("handler.result.id", account.ID.String()))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, account); err != nil {
			logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		logger.Info("Berhasil mengambil data bank",
			zap.String("source", "database"),
			zap.String("handler.result.id", account.ID.String()),
		)

		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data akun dengan id = %s", account.ID), map[string]any{
			"account": account,
		})
	}
}

// GET /account/{id}/transactions?type=all/in/out
func (h *AccountsHandler) GetTransactions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)
		trxTypeEnum := []string{"all", "in", "out"}
		key := "account_trx"

		idStr := r.PathValue("id")
		trxType := r.URL.Query().Get("type")

		if trxType == "" {
			trxType = "all"
		}

		logger.Info("Path & trx type received",
			zap.String("handler.query", idStr),
			zap.String("handler.trxType", trxType),
		)

		// Valid Uuid
		idParse, err := uuid.Parse(idStr)
		if err != nil {
			logger.Error(models.ErrInvalidUuid.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		isValidType := slices.Contains(trxTypeEnum, trxType)

		// Valid Trx Type
		if !isValidType {
			logger.Error(models.ErrInvalidTrxType.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidTrxType), models.ErrInvalidTrxType.Error())
			return
		}

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_TRANSACTION + ":" + idParse.String() + ":" + trxType)
		logger.Info("Checking cache",
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

		logger.Info("Mencari transaksi akun",
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
					logger.Info("Cache Hit - Berhasil mengambil data transaksi akun",
						zap.String("source", "redis"),
						zap.Int("handler.result.count", len(transactions)),
					)
					dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data transaksi dengan id akun = %s & tipe transaksi = %s", idParse, trxType), map[string]any{
						"transactions": transactions,
					})
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
		logger.Info("Cache miss", zap.String("key", cacheKey))

		// Exec
		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-Database")
		transactions, err := h.svc.FetchTransactionsByAccountId(dbCtx, idStr, trxType)
		dbSpan.End()

		if err != nil {
			logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		span.SetAttributes(attribute.Int("handler.result.count", len(transactions)))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transactions); err != nil {
			logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		logger.Info("Berhasil mengambil data transaksi akun",
			zap.String("source", "database"),
			zap.Int("handler.result.count", len(transactions)),
		)

		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data transaksi dengan id akun = %s & tipe transaksi = %s", idParse, trxType), map[string]any{
			"transactions": transactions,
		})
	}
}

// POST /account
func (h *AccountsHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)
		key := "account_list"

		var payload models.Account
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			logger.Error(models.ErrInvalidJsonFormat.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		logger.Info("Payload received", zap.Any("payload", payload))

		dbCtx, dbSpan := tracer.Start(ctx, "Create-Account")
		newAccount, err := h.svc.CreateNewAccount(dbCtx, payload)
		dbSpan.End()

		if err != nil {
			logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		// Invalidate Existing Cache
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_LIST)
		cacheStart := time.Now()
		if err := h.rdb.Del(ctx, cacheKey).Err(); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues(key, "error").Inc()
			span.RecordError(err)
			span.SetStatus(codes.Error, models.ErrRedisInvalidate.Error())
			logger.Error(models.ErrRedisInvalidate.Error(), zap.Error(err))
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(key, "invalidate").Inc()
			span.AddEvent("Cache Invalidated")
		}

		metrics.CacheDuration.WithLabelValues("invalidate", key).
			Observe(time.Since(cacheStart).Seconds())

		span.SetAttributes(attribute.String("handler.result.id", newAccount.ID.String()))

		logger.Info("Berhasil membuat data akun baru",
			zap.String("source", "database"),
			zap.String("handler.result.id", newAccount.ID.String()),
		)

		dto.WriteResponse(w, http.StatusCreated, "Berhasil membuat data akun baru", map[string]any{
			"account": newAccount,
		})
	}
}

// PATCH /account/{id}
func (h *AccountsHandler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)
		keyList := "account_list"
		keyId := "account_id"

		idStr := r.PathValue("id")
		logger.Info("Path received", zap.String("handler.query", idStr))

		// Valid Uuid
		idParse, err := uuid.Parse(idStr)
		if err != nil {
			logger.Error(models.ErrInvalidUuid.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		var payload models.Account
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			logger.Error(models.ErrInvalidJsonFormat.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		logger.Info("Payload received", zap.Any("payload", payload))

		payload.ID = idParse

		dbCtx, dbSpan := tracer.Start(ctx, "Update-Account")
		updatedId, err := h.svc.PatchAccountById(dbCtx, payload)
		dbSpan.End()

		if err != nil {
			logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
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
			logger.Error(models.ErrRedisInvalidate.Error(), zap.Error(err))
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

		logger.Info("Berhasil memperbarui data akun",
			zap.String("source", "database"),
			zap.String("handler.result.id", updatedId),
		)

		dto.WriteResponse(w, http.StatusOK, "Berhasil mengupdate data account", map[string]any{
			"id": updatedId,
		})
	}
}

// DELETE /account/{id}
func (h *AccountsHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)
		keyList := "account_list"
		keyId := "account_id"

		idStr := r.PathValue("id")
		logger.Info("Path received", zap.String("handler.query", idStr))

		// Valid Uuid
		idParse, errId := uuid.Parse(idStr)
		if errId != nil {
			logger.Error(models.ErrInvalidUuid.Error(), zap.Error(errId))
			span.RecordError(errId)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		dbCtx, dbSpan := tracer.Start(ctx, "Delete-Account")
		err := h.svc.DeleteAccountById(dbCtx, idParse.String())
		dbSpan.End()

		if err != nil {
			logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
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
			logger.Error(models.ErrRedisInvalidate.Error(), zap.Error(err))
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
		logger.Info("Berhasil menghapus data akun",
			zap.String("source", "database"),
			zap.String("handler.delete.id", idParse.String()),
		)

		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil menghapus akun dengan id : %s", idStr), map[string]any{})
	}
}
