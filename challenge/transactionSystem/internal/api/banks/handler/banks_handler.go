package handler

import (
	"belajar-go/challenge/transactionSystem/config"
	"belajar-go/challenge/transactionSystem/dto"
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/banks/repository"
	"belajar-go/challenge/transactionSystem/internal/api/banks/service"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"belajar-go/challenge/transactionSystem/observability/metrics"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type BanksHandler struct {
	mux         *http.ServeMux
	svc         service.BankService
	rdb         *redis.Client
	keyManager  *helper.RedisKeyManager
	idempotency *middleware.IdempotencyMiddleware
	logger      *zap.Logger
}

// Bank handler init
func NewBanksHandler(mux *http.ServeMux, db *sqlx.DB, rdb *redis.Client) *BanksHandler {

	keyManager := helper.NewRedisKeyManager("transaction_system", config.DOMAIN_BANK)
	idempotency := middleware.NewIdempotencyMiddleware(rdb, keyManager)
	bankRepo := repository.NewBankRepository(db)
	bankSvc := service.NewBanksService(bankRepo)
	logger := helper.Log

	return &BanksHandler{
		mux:         mux,
		svc:         bankSvc,
		rdb:         rdb,
		keyManager:  keyManager,
		idempotency: idempotency,
		logger:      logger,
	}
}

// Map route
func (a *BanksHandler) MapRoutes(obs *middleware.ObservabilityMiddleware) {

	version := "v1.0"

	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, version, "/banks"),
		obs.Wrap("BankHandler.GetAll", config.DOMAIN_BANK, a.GetAll()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, version, "/bank/{identifier}"),
		obs.Wrap("BankHandler.GetById", config.DOMAIN_BANK, a.GetById()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, version, "/bank"),
		obs.Wrap("BankHandler.Create", config.DOMAIN_BANK, a.idempotency.Check(a.Create())).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPatch, version, "/bank/{id}"),
		obs.Wrap("BankHandler.Patch", config.DOMAIN_BANK, a.idempotency.Check(a.Update())).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodDelete, version, "/bank/{id}"),
		obs.Wrap("BankHandler.Delete", config.DOMAIN_BANK, a.Delete()).ServeHTTP,
	)
}

// GET /banks
func (h *BanksHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_BANK_LIST)
		h.logger.Info("Checking cache", zap.String("key", cacheKey))

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()

		metrics.CacheDuration.WithLabelValues(
			"get",
			"bank_list",
		).Observe(time.Since(cacheStart).Seconds())

		cacheSpan.End()

		if errRedis == nil {

			metrics.CacheRequestsTotal.WithLabelValues(
				"bank_list",
				"hit",
			).Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var banks []models.Bank
				if err := json.Unmarshal(decompressed, &banks); err == nil {
					span.AddEvent("Cache hit occurred")
					h.logger.Info("Cache Hit - Berhasil mengambil list data bank",
						zap.String("source", "redis"),
						zap.Int("count", len(banks)),
					)
					dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil list data bank", map[string]any{"banks": banks})
					return
				}
			}
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(
				"bank_list",
				"miss",
			).Inc()
		}

		span.AddEvent("Cache miss")
		h.logger.Info("Cache miss", zap.String("key", cacheKey))

		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-Database")
		banks, err := h.svc.FetchAllBanks(dbCtx)
		dbSpan.End()

		if err != nil {
			h.logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		span.SetAttributes(attribute.Int("result.count", len(banks)))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, banks); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", "bank_list").
			Observe(time.Since(cacheSetStart).Seconds())

		h.logger.Info("Berhasil mengambil list data bank",
			zap.String("source", "database"),
			zap.Int("count", len(banks)),
		)

		dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil list data bank", map[string]any{
			"banks": banks,
		})
	}
}

// GET /bank/{identifier}
func (h *BanksHandler) GetById() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)

		idStr := r.PathValue("identifier")
		h.logger.Info("Path received", zap.String("handler.query", idStr))

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_BANK_ID, idStr)
		h.logger.Info("Checking cache",
			zap.String("key", cacheKey),
			zap.String("handler.query", idStr),
		)

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()
		cacheDuration := time.Since(cacheStart).Seconds()

		metrics.CacheDuration.WithLabelValues(
			"get",
			"bank_id",
		).Observe(cacheDuration)

		cacheSpan.End()

		h.logger.Info("Mencari bank", zap.String("identifier", idStr))

		if errRedis == nil {

			metrics.CacheRequestsTotal.WithLabelValues(
				"bank_id",
				"hit",
			).Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var bank models.Bank
				if err := json.Unmarshal(decompressed, &bank); err == nil {
					span.AddEvent("Cache hit occurred")
					h.logger.Info("Cache Hit - Berhasil mengambil data bank",
						zap.String("source", "redis"),
						zap.String("handler.result.id", bank.ID.String()),
					)
					dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data bank dengan identifier = %s", idStr), map[string]any{
						"bank": bank,
					})
					return
				}
			}
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(
				"bank_id",
				"miss",
			).Inc()
		}

		span.AddEvent("Cache miss")
		h.logger.Info("Cache miss", zap.String("key", cacheKey))

		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-Database")
		bank, err := h.svc.FetchBankById(dbCtx, idStr)
		dbSpan.End()

		if err != nil {
			h.logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		span.SetAttributes(attribute.String("handler.result.id", bank.ID.String()))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, bank); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", "bank_id").
			Observe(time.Since(cacheSetStart).Seconds())

		h.logger.Info("Berhasil mengambil data bank",
			zap.String("source", "database"),
			zap.String("handler.result.id", bank.ID.String()),
		)

		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data bank dengan identifier = %s", idStr), map[string]any{
			"bank": bank,
		})
	}
}

// POST /bank
func (h *BanksHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		// idempotencyKey := r.Header.Get("X-Idempotency-Key")

		var payload models.Bank
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.logger.Error(models.ErrInvalidJsonFormat.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		h.logger.Info("Payload received", zap.Any("payload", payload))

		dbCtx, dbSpan := tracer.Start(ctx, "Create-Bank")
		newBank, err := h.svc.CreateNewBank(dbCtx, payload)
		dbSpan.End()

		if err != nil {
			h.logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		// Invalidate Existing Cache
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_BANK_LIST)
		cacheStart := time.Now()
		if err := h.rdb.Del(ctx, cacheKey).Err(); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues("bank_list", "error").Inc()
			span.RecordError(err)
			span.SetStatus(codes.Error, models.ErrRedisInvalidate.Error())
			h.logger.Error(models.ErrRedisInvalidate.Error(), zap.Error(err))
		} else {
			metrics.CacheRequestsTotal.WithLabelValues("bank_list", "invalidate").Inc()
			span.AddEvent("Cache Invalidated")
		}
		metrics.CacheDuration.WithLabelValues("invalidate", "bank_list").
			Observe(time.Since(cacheStart).Seconds())

		span.SetAttributes(attribute.String("handler.result.id", newBank.ID.String()))

		h.logger.Info("Berhasil membuat data bank baru",
			zap.String("source", "database"),
			zap.String("handler.result.id", newBank.ID.String()),
		)

		dto.WriteResponse(w, http.StatusCreated, "Berhasil membuat data bank baru", map[string]any{
			"bank": newBank,
		})
	}
}

// PATCH /bank/{id}
func (h *BanksHandler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)

		bankId := r.PathValue("id")
		h.logger.Info("Path received", zap.String("handler.query", bankId))

		var payload models.Bank
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.logger.Error(models.ErrInvalidJsonFormat.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		h.logger.Info("Payload received", zap.Any("payload", payload))

		bankIdParse, err := uuid.Parse(bankId)
		if err != nil {
			// Jika gagal di-parse, kembalikan error validasi
			h.logger.Error(models.ErrInvalidUuid.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		payload.ID = bankIdParse

		dbCtx, dbSpan := tracer.Start(ctx, "Update-Bank")
		bankCode, err := h.svc.PatchBank(dbCtx, payload)
		dbSpan.End()

		if err != nil {
			h.logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		// Invalidate Existing Cache
		cacheKeyList := h.keyManager.Generate(config.REDIS_KEY_BANK_LIST)
		cacheKeyId := h.keyManager.Generate(config.REDIS_KEY_BANK_ID, bankCode)

		cacheStart := time.Now()
		if err := h.rdb.Del(ctx, cacheKeyList, cacheKeyId).Err(); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues("bank_list", "error").Inc()
			metrics.CacheRequestsTotal.WithLabelValues("bank_id", "error").Inc()
			span.RecordError(err)
			span.SetStatus(codes.Error, models.ErrRedisInvalidate.Error())
			h.logger.Error(models.ErrRedisInvalidate.Error(), zap.Error(err))
		} else {
			metrics.CacheRequestsTotal.WithLabelValues("bank_list", "invalidate").Inc()
			metrics.CacheRequestsTotal.WithLabelValues("bank_id", "invalidate").Inc()
			span.AddEvent("Cache Invalidated")
		}
		metrics.CacheDuration.WithLabelValues("invalidate", "bank_list").
			Observe(time.Since(cacheStart).Seconds())
		metrics.CacheDuration.WithLabelValues("invalidate", "bank_id").
			Observe(time.Since(cacheStart).Seconds())

		span.SetAttributes(attribute.String("handler.result.bankCode", bankCode))

		h.logger.Info("Berhasil memperbarui data bank",
			zap.String("source", "database"),
			zap.String("handler.result.bankCode", bankCode),
		)

		dto.WriteResponse(w, http.StatusOK, "Berhasil mengupdate data bank", map[string]any{
			"bank_code": bankCode,
		})
	}
}

// DELETE /bank/{id}
func (h *BanksHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)

		bankId := r.PathValue("id")
		h.logger.Info("Path received", zap.String("handler.query", bankId))

		bankIdParse, errId := uuid.Parse(bankId)
		if errId != nil {
			h.logger.Error(models.ErrInvalidUuid.Error(), zap.Error(errId))
			span.RecordError(errId)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		dbCtx, dbSpan := tracer.Start(ctx, "Delete-Bank")
		err := h.svc.DeleteBank(dbCtx, bankIdParse.String())
		dbSpan.End()

		if err != nil {
			h.logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		// Invalidate Existing Cache
		cacheKeyList := h.keyManager.Generate(config.REDIS_KEY_BANK_LIST)
		cacheKeyId := h.keyManager.Generate(config.REDIS_KEY_BANK_ID, bankIdParse.String())

		cacheStart := time.Now()
		if err := h.rdb.Del(ctx, cacheKeyList, cacheKeyId).Err(); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues("bank_list", "error").Inc()
			metrics.CacheRequestsTotal.WithLabelValues("bank_id", "error").Inc()
			span.RecordError(err)
			span.SetStatus(codes.Error, models.ErrRedisInvalidate.Error())
			h.logger.Error(models.ErrRedisInvalidate.Error(), zap.Error(err))
		} else {
			metrics.CacheRequestsTotal.WithLabelValues("bank_list", "invalidate").Inc()
			metrics.CacheRequestsTotal.WithLabelValues("bank_id", "invalidate").Inc()
			span.AddEvent("Cache Invalidated")
		}
		metrics.CacheDuration.WithLabelValues("invalidate", "bank_list").
			Observe(time.Since(cacheStart).Seconds())
		metrics.CacheDuration.WithLabelValues("invalidate", "bank_id").
			Observe(time.Since(cacheStart).Seconds())

		span.SetAttributes(attribute.String("handler.delete.id", bankIdParse.String()))
		h.logger.Info("Berhasil menghapus data bank",
			zap.String("source", "database"),
			zap.String("handler.delete.id", bankIdParse.String()),
		)

		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil menghapus bank : %s", bankId), map[string]any{
			"id": bankIdParse.String(),
		})
	}
}
