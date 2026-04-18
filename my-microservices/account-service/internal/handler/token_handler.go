package handler

import (
	"encoding/json"
	"my-microservices/account-service/config"
	"my-microservices/account-service/helper"
	"my-microservices/account-service/internal/domain"
	"my-microservices/account-service/internal/dto"
	"my-microservices/account-service/internal/middleware"
	"my-microservices/account-service/observability/metrics"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type TokenHandler struct {
	mux        *http.ServeMux
	rdb        *redis.Client
	keyManager *helper.RedisKeyManager
	logger     *zap.Logger
}

func NewTokenHandler(mux *http.ServeMux, rdb *redis.Client) *TokenHandler {
	keyManager := helper.NewRedisKeyManager("transaction_system", config.DOMAIN_OAUTH)
	logger := helper.Log

	return &TokenHandler{
		mux:        mux,
		rdb:        rdb,
		keyManager: keyManager,
		logger:     logger,
	}
}

func (a *TokenHandler) MapRoutes(obs *middleware.ObservabilityMiddleware) {
	version := "v1.0"

	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, version, "/access-token"),
		obs.Wrap("Oauth.Get", config.DOMAIN_OAUTH, a.Get()).ServeHTTP,
	)
}

// POST /v1.0/access-token
func (h *TokenHandler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		key := "token"

		accountId := r.Header.Get("X-ACCOUNT-ID")
		if accountId == "" {
			h.logger.Error("X-ACCOUNT-ID is required", zap.String("account.id", accountId))
			dto.WriteResponse(
				w,
				http.StatusBadRequest,
				strconv.Itoa(http.StatusBadRequest),
				"X-ACCOUNT-ID header is required",
				nil,
			)
			return
		}

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCESS_TOKEN, accountId)
		h.logger.Info("Checking cache", zap.String("key", cacheKey))

		// --- Cache Lookup ---
		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()

		metrics.CacheDuration.WithLabelValues("get", key).Observe(time.Since(cacheStart).Seconds())
		cacheSpan.End()

		// --- Cache Hit ---
		if errRedis == nil {
			metrics.CacheRequestsTotal.WithLabelValues(key, "hit").Inc()

			decompressed, err := helper.DecompressData(val)
			if err != nil {
				h.logger.Error("Failed to decompress cache data", zap.Error(err))
				dto.WriteResponse(
					w,
					http.StatusInternalServerError,
					strconv.Itoa(http.StatusInternalServerError),
					"Gagal memproses data cache",
					nil,
				)
				return
			}

			var token string
			if err := json.Unmarshal(decompressed, &token); err != nil {
				h.logger.Error("Failed to unmarshal token from cache", zap.Error(err))
				dto.WriteResponse(
					w,
					http.StatusInternalServerError,
					strconv.Itoa(http.StatusInternalServerError),
					"Gagal memproses token dari cache",
					nil,
				)
				return
			}

			span.AddEvent("Cache hit occurred")
			h.logger.Info("Cache Hit - Berhasil mengambil access token",
				zap.String("source", "redis"),
				zap.String("account_id", accountId),
			)
			dto.WriteResponse(
				w,
				http.StatusOK,
				strconv.Itoa(http.StatusOK),
				"Berhasil mengambil access token",
				map[string]any{"token": token},
			)
			return
		}

		// --- Cache Miss ---
		metrics.CacheRequestsTotal.WithLabelValues(key, "miss").Inc()
		h.logger.Info("Cache Miss - Token tidak ditemukan di cache", zap.String("account_id", accountId))

		token, err := helper.GenerateAccessToken(accountId)
		if err != nil {
			h.logger.Error(err.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(
				w,
				domain.StatusCodeHandler(err),
				strconv.Itoa(domain.StatusCodeHandler(err)),
				err.Error(),
			)
			return
		}

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, token); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", "bank_list").
			Observe(time.Since(cacheSetStart).Seconds())

		h.logger.Info("Berhasil mengambil list data bank",
			zap.String("source", "jwt"),
			zap.String("result.key", token),
		)

		dto.WriteResponse(
			w,
			http.StatusOK,
			strconv.Itoa(http.StatusOK),
			"Berhasil mengambil access token",
			map[string]any{"token": token},
		)
	}
}
