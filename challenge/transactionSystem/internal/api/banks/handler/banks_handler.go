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
	"go.uber.org/zap"
)

type BanksHandler struct {
	mux         *http.ServeMux
	svc         service.BankService
	rdb         *redis.Client
	keyManager  *helper.RedisKeyManager
	idempotency *middleware.IdempotencyMiddleware
}

// Bank handler init
func NewBanksHandler(mux *http.ServeMux, db *sqlx.DB, rdb *redis.Client) *BanksHandler {

	keyManager := helper.NewRedisKeyManager("transaction_system", "bank")
	idempotency := middleware.NewIdempotencyMiddleware(rdb, keyManager)
	bankRepo := repository.NewBankRepository(db)
	bankSvc := service.NewBanksService(bankRepo)

	return &BanksHandler{
		mux:         mux,
		svc:         bankSvc,
		rdb:         rdb,
		keyManager:  keyManager,
		idempotency: idempotency,
	}
}

// Map route
func (a *BanksHandler) MapRoutes(obs *middleware.ObservabilityMiddleware) {

	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/banks"),
		obs.Wrap("BankHandler.GetAll", config.DOMAIN_BANK, a.GetAll()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/bank/{identifier}"),
		obs.Wrap("BankHandler.GetById", config.DOMAIN_BANK, a.GetById()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, "/bank"),
		obs.Wrap("BankHandler.Create", config.DOMAIN_BANK, a.idempotency.Check(a.Create())).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPatch, "/bank/{id}"),
		obs.Wrap("BankHandler.Patch", config.DOMAIN_BANK, a.idempotency.Check(a.Update())).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodDelete, "/bank/{id}"),
		obs.Wrap("BankHandler.Delete", config.DOMAIN_BANK, a.Delete()).ServeHTTP,
	)
}

// GET /banks
func (h *BanksHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cacheStart := time.Now()
		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_BANK_LIST)
		logger.Info("Checking cache", zap.String("key", cacheKey))

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()
		cacheDuration := time.Since(cacheStart).Seconds()

		metrics.CacheDuration.WithLabelValues(
			"get",
			"bank_list",
		).Observe(cacheDuration)

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
					logger.Info("Cache Hit - Berhasil mengambil list data bank",
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
		logger.Info("Cache miss", zap.String("key", cacheKey))

		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-Database")
		banks, err := h.svc.FetchAllBanks(dbCtx)
		dbSpan.End()

		if err != nil {
			logger.Error("Database fetch failed", zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		span.SetAttributes(attribute.Int("result.count", len(banks)))
		helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, banks)

		logger.Info("Berhasil mengambil list data bank",
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

		idStr := r.PathValue("identifier")
		cacheStart := time.Now()
		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_BANK_ID + ":" + idStr)
		logger.Info("Checking cache",
			zap.String("key", cacheKey),
			zap.String("handler.query", idStr),
		)

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()
		cacheDuration := time.Since(cacheStart).Seconds()

		metrics.CacheDuration.WithLabelValues(
			"get",
			"bank_id",
		).Observe(cacheDuration)

		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Mencari bank dengan keyword: %s", idStr))

		cacheSpan.End()

		if errRedis == nil {

			metrics.CacheRequestsTotal.WithLabelValues(
				"account_id",
				"hit",
			).Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var bank models.Bank
				if err := json.Unmarshal(decompressed, &bank); err == nil {
					span.AddEvent("Cache hit occurred")
					logger.Info("Cache Hit - Berhasil mengambil data bank",
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
				"account_id",
				"miss",
			).Inc()
		}

		span.AddEvent("Cache miss")
		logger.Info("Cache miss", zap.String("key", cacheKey))

		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-Database")
		bank, err := h.svc.FetchBankById(dbCtx, idStr)
		dbSpan.End()

		if err != nil {
			logger.Error("Database fetch failed", zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		span.SetAttributes(attribute.String("handler.result.id", bank.ID.String()))
		helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, bank)

		logger.Info("Berhasil mengambil data bank",
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

		var payload models.Bank
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			helper.PrintLog("bank", helper.LogPositionHandler, models.ErrInvalidJsonFormat.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))

		newBank, err := h.svc.CreateNewBank(payload)
		if err != nil {
			helper.PrintLog("bank", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		// Invalidate Existing Cache
		cacheBankList := h.keyManager.Generate(config.REDIS_KEY_BANK_LIST)
		errDel := h.rdb.Del(ctx, cacheBankList).Err()
		if errDel != nil {
			helper.PrintLog("redis", helper.LogPositionHandler, "Gagal menghapus cache: "+errDel.Error())
		}

		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Berhasil membuat data bank baru : %+v", newBank))
		dto.WriteResponse(w, http.StatusCreated, "Berhasil membuat data bank baru", map[string]any{
			"bank": newBank,
		})
	}
}

// PATCH /bank/{id}
func (h *BanksHandler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		bankId := r.PathValue("id")
		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Mendapatkan kode bank = %s", bankId))

		var payload models.Bank
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			helper.PrintLog("bank", helper.LogPositionHandler, models.ErrInvalidJsonFormat.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))

		bankIdParse, err := uuid.Parse(bankId)
		if err != nil {
			// Jika gagal di-parse, kembalikan error validasi
			helper.PrintLog("bank", helper.LogPositionHandler, models.ErrInvalidUuid.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		payload.ID = bankIdParse

		returnedId, err := h.svc.PatchBank(payload)
		if err != nil {
			helper.PrintLog("bank", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		// Invalidate Existing Cache
		cacheBankList := h.keyManager.Generate(config.REDIS_KEY_BANK_LIST)
		cacheBankId := h.keyManager.Generate(config.REDIS_KEY_BANK_ID + ":" + returnedId)
		errDel := h.rdb.Del(ctx, cacheBankList, cacheBankId).Err()
		if errDel != nil {
			helper.PrintLog("redis", helper.LogPositionHandler, "Gagal menghapus cache: "+errDel.Error())
		}

		helper.PrintLog("bank", helper.LogPositionHandler, "Berhasil mengupdate data bank")
		dto.WriteResponse(w, http.StatusOK, "Berhasil mengupdate data bank", map[string]any{
			"id": returnedId,
		})
	}
}

// DELETE /bank/{id}
func (h *BanksHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		bankId := r.PathValue("id")
		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Mendapatkan id bank = %s", bankId))

		bankIdParse, errId := uuid.Parse(bankId)
		if errId != nil {
			// Jika gagal di-parse, kembalikan error validasi
			helper.PrintLog("bank", helper.LogPositionHandler, models.ErrInvalidUuid.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		err := h.svc.DeleteBank(bankIdParse.String())
		if err != nil {
			helper.PrintLog("bank", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		// Invalidate Existing Cache
		cacheBankList := h.keyManager.Generate(config.REDIS_KEY_BANK_LIST)
		cacheBankId := h.keyManager.Generate(config.REDIS_KEY_BANK_ID)
		errDel := h.rdb.Del(ctx, cacheBankList, cacheBankId).Err()
		if errDel != nil {
			helper.PrintLog("redis", helper.LogPositionHandler, "Gagal menghapus cache: "+errDel.Error())
		}

		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Berhasil menghapus bank : %s", bankId))
		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil menghapus bank : %s", bankId), map[string]any{})
	}
}
