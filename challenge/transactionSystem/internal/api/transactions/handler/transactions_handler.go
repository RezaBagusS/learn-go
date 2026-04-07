package handler

import (
	"belajar-go/challenge/transactionSystem/config"
	"belajar-go/challenge/transactionSystem/dto"
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/transactions/repository"
	"belajar-go/challenge/transactionSystem/internal/api/transactions/service"
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

type TransactionsHandler struct {
	mux         *http.ServeMux
	svc         service.TransactionService
	rdb         *redis.Client
	keyManager  *helper.RedisKeyManager
	idempotency *middleware.IdempotencyMiddleware
}

func NewTransactionsHandler(mux *http.ServeMux, db *sqlx.DB, rdb *redis.Client) *TransactionsHandler {
	trxRepo := repository.NewtransactionRepository(db)
	TrxSvc := service.NewTransactionsService(trxRepo)
	keyManager := helper.NewRedisKeyManager("transaction_system", config.DOMAIN_TRANSACTION)
	idempotency := middleware.NewIdempotencyMiddleware(rdb, keyManager)

	return &TransactionsHandler{
		mux:         mux,
		svc:         TrxSvc,
		rdb:         rdb,
		keyManager:  keyManager,
		idempotency: idempotency,
	}
}

func (a *TransactionsHandler) MapRoutes(obs *middleware.ObservabilityMiddleware) {
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/transactions"),
		obs.Wrap("TransactionHandler.GetAll", config.DOMAIN_TRANSACTION, a.GetAll()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/transactions/summary"),
		obs.Wrap("TransactionHandler.GetSummary", config.DOMAIN_TRANSACTION, a.GetSummary()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/transaction/{id}"),
		obs.Wrap("TransactionHandler.GetById", config.DOMAIN_TRANSACTION, a.GetById()).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, "/transaction"),
		obs.Wrap("TransactionHandler.Create", config.DOMAIN_TRANSACTION, a.idempotency.Check(a.Create())).ServeHTTP,
	)
}

// GET /transactions
func (h *TransactionsHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)
		key := "transaction_list"

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_LIST)
		logger.Info("Checking cache", zap.String("key", cacheKey))

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()

		metrics.CacheDuration.WithLabelValues(
			"get",
			key,
		).Observe(time.Since(cacheStart).Seconds())

		cacheSpan.End()

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
					logger.Info("Cache Hit - Berhasil mengambil list data transaksi",
						zap.String("source", "redis"),
						zap.Int("count", len(transactions)),
					)
					dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil list data transaksi", map[string]any{"transactions": transactions})
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
		transactions, err := h.svc.FetchAllTransactions(dbCtx)
		dbSpan.End()

		if err != nil {
			logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		span.SetAttributes(attribute.Int("result.count", len(transactions)))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transactions); err != nil {
			logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		logger.Info("Berhasil mengambil list data transaksi",
			zap.String("source", "database"),
			zap.Int("count", len(transactions)),
		)

		dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil list data transaksi", map[string]any{
			"transactions": transactions,
		})
	}
}

// GET /transactions/summary
func (h *TransactionsHandler) GetSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)

		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}

		// YYYY-MM-DD
		timeParse, errDate := time.Parse("2006-01-02", dateStr)
		if errDate != nil {
			logger.Error("Invalid date format", zap.String("date", dateStr), zap.Error(errDate))
			span.RecordError(errDate)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidDate), models.ErrInvalidDate.Error())
			return
		}

		key := "transaction_summary"
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_SUMMARY, dateStr)
		logger.Info("Checking cache", zap.String("key", cacheKey), zap.String("date", dateStr))

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()

		metrics.CacheDuration.WithLabelValues(
			"get",
			key,
		).Observe(time.Since(cacheStart).Seconds())

		cacheSpan.End()

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
					logger.Info("Cache Hit - Berhasil mengambil data summary transaksi",
						zap.String("source", "redis"),
						zap.String("date", dateStr),
						zap.Int("count", len(transactions)),
					)
					dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil data summary transaksi", map[string]any{"transactions": transactions})
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
		logger.Info("Cache miss", zap.String("key", cacheKey), zap.String("date", dateStr))

		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-Database")
		transactions, err := h.svc.FetchSummaryToday(dbCtx, timeParse)
		dbSpan.End()

		if err != nil {
			logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		span.SetAttributes(
			attribute.Int("result.count", len(transactions)),
			attribute.String("query.date", dateStr),
		)

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transactions); err != nil {
			logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		logger.Info("Berhasil mengambil data summary transaksi",
			zap.String("source", "database"),
			zap.String("date", dateStr),
			zap.Int("count", len(transactions)),
		)

		dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil data summary transaksi", map[string]any{
			"transactions": transactions,
		})
	}
}

// GET /transaction/{id}
func (h *TransactionsHandler) GetById() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)

		idStr := r.PathValue("id")
		logger.Info("Mendapatkan id transaction", zap.String("id", idStr))

		_, err := uuid.Parse(idStr)
		if err != nil {
			logger.Error("Invalid UUID format", zap.String("id", idStr), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		key := "transaction_detail"
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_ID, idStr)
		logger.Info("Checking cache", zap.String("key", cacheKey), zap.String("id", idStr))

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()

		metrics.CacheDuration.WithLabelValues(
			"get",
			key,
		).Observe(time.Since(cacheStart).Seconds())

		cacheSpan.End()

		if errRedis == nil {

			metrics.CacheRequestsTotal.WithLabelValues(
				key,
				"hit",
			).Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var transaction models.Transaction
				if err := json.Unmarshal(decompressed, &transaction); err == nil {
					span.AddEvent("Cache hit occurred")
					logger.Info("Cache Hit - Berhasil mengambil data transaksi",
						zap.String("source", "redis"),
						zap.String("id", idStr),
					)
					dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data transaksi dengan id = %s", idStr), map[string]any{"transaction": transaction})
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
		logger.Info("Cache miss", zap.String("key", cacheKey), zap.String("id", idStr))

		dbCtx, dbSpan := tracer.Start(ctx, "Fetch-from-Database")
		transaction, err := h.svc.FetchTransactionById(dbCtx, idStr)
		dbSpan.End()

		if err != nil {
			logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		span.SetAttributes(
			attribute.String("query.id", idStr),
		)

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transaction); err != nil {
			logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		logger.Info("Berhasil mengambil data transaksi",
			zap.String("source", "database"),
			zap.String("id", idStr),
		)

		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data transaksi dengan id = %s", idStr), map[string]any{
			"transaction": transaction,
		})
	}
}

// POST /transaction
func (h *TransactionsHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)
		key := "transaction_list"

		var payload models.Transaction
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			logger.Error(models.ErrInvalidJsonFormat.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		logger.Info("Payload received", zap.Any("payload", payload))

		dbCtx, dbSpan := tracer.Start(ctx, "Create-Transaction")
		transactionID, err := h.svc.CreateTrx(dbCtx, payload)
		dbSpan.End()

		if err != nil {
			logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		// Invalidate transaction_list cache
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_LIST)
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

		// Invalidate transaction_summary cache (semua tanggal)
		summaryPattern := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_SUMMARY, "*")
		summaryKeys, errScan := h.rdb.Keys(ctx, summaryPattern).Result()
		if errScan == nil && len(summaryKeys) > 0 {
			if err := h.rdb.Del(ctx, summaryKeys...).Err(); err != nil {
				span.RecordError(err)
				logger.Warn("Failed to invalidate summary cache", zap.Error(err))
			} else {
				span.AddEvent("Summary Cache Invalidated")
				logger.Info("Summary cache invalidated", zap.Strings("keys", summaryKeys))
			}
		}

		span.SetAttributes(attribute.String("handler.result.id", transactionID))

		logger.Info("Transfer berhasil dilakukan",
			zap.String("source", "database"),
			zap.String("handler.result.id", transactionID),
		)

		dto.WriteResponse(w, http.StatusCreated, "Transfer berhasil dilakukan", map[string]any{
			"id":     transactionID,
			"amount": payload.Amount,
			"note":   payload.Note,
		})
	}
}
