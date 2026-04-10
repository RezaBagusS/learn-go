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
	"errors"
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
	logger      *zap.Logger
}

func NewTransactionsHandler(mux *http.ServeMux, db *sqlx.DB, rdb *redis.Client) *TransactionsHandler {
	trxRepo := repository.NewtransactionRepository(db)
	TrxSvc := service.NewTransactionsService(trxRepo)
	keyManager := helper.NewRedisKeyManager("transaction_system", config.DOMAIN_TRANSACTION)
	idempotency := middleware.NewIdempotencyMiddleware(rdb, keyManager)
	logger := helper.Log

	return &TransactionsHandler{
		mux:         mux,
		svc:         TrxSvc,
		rdb:         rdb,
		keyManager:  keyManager,
		idempotency: idempotency,
		logger:      logger,
	}
}

func (a *TransactionsHandler) MapRoutes(obs *middleware.ObservabilityMiddleware) {

	version := "v1.0"

	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, version, "/transactions"),
		middleware.ValidateSNAPToken(
			obs.Wrap("TransactionHandler.GetAll", config.DOMAIN_TRANSACTION, a.GetAll()),
		).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, version, "/transactions/summary"),
		middleware.ValidateSNAPToken(
			obs.Wrap("TransactionHandler.GetSummary", config.DOMAIN_TRANSACTION, a.GetSummary()),
		).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, version, "/transaction/{id}"),
		middleware.ValidateSNAPToken(
			obs.Wrap("TransactionHandler.GetById", config.DOMAIN_TRANSACTION, a.GetById()),
		).ServeHTTP,
	)
	// a.mux.HandleFunc(
	// 	helper.NewAPIPath(http.MethodPost, version, "/transaction"),
	// 	middleware.ValidateSNAPToken(
	// 		obs.Wrap("TransactionHandler.Create", config.DOMAIN_TRANSACTION, a.idempotency.Check(a.Create())),
	// 	).ServeHTTP,
	// )
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, version, "/transfer-intrabank"),
		middleware.ValidateSNAPToken(
			obs.Wrap("TransactionHandler.TransferIntraBank", config.DOMAIN_TRANSACTION, a.idempotency.Check(a.TransferIntraBank())),
		).ServeHTTP,
	)
}

// GET /transactions
func (h *TransactionsHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		key := "transaction_list"
		svcCode := config.SVC_CODE_TRX_HISTORY_LIST

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_LIST)
		h.logger.Info("Checking cache", zap.String("key", cacheKey))

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
					h.logger.Info("Cache Hit - Berhasil mengambil list data transaksi",
						zap.String("source", "redis"),
						zap.Int("count", len(transactions)),
					)
					dto.WriteResponse(
						w,
						models.SnapSuccess.HttpCode,
						models.SnapSuccess.GetResponseCode(svcCode),
						models.SnapSuccess.ResponseMessage, map[string]any{
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
		h.logger.Info("Cache miss", zap.String("key", cacheKey))

		svcCtx, svcSpan := tracer.Start(ctx, "Fetch-from-Database")
		transactions, snapErr := h.svc.FetchAllTransactions(svcCtx)
		svcSpan.End()

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

		span.SetAttributes(attribute.Int("result.count", len(transactions)))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transactions); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		h.logger.Info("Berhasil mengambil list data transaksi",
			zap.String("source", "database"),
			zap.Int("count", len(transactions)),
		)

		dto.WriteResponse(
			w,
			models.SnapSuccess.HttpCode,
			models.SnapSuccess.GetResponseCode(svcCode),
			models.SnapSuccess.ResponseMessage,
			map[string]any{
				"transactions": transactions,
			})
	}
}

// GET /transactions/summary
func (h *TransactionsHandler) GetSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		svcCode := config.SVC_CODE_TRX_HISTORY_LIST

		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}

		// YYYY-MM-DD
		timeParse, errDate := time.Parse("2006-01-02", dateStr)
		if errDate != nil {
			h.logger.Error("Invalid date format", zap.String("date", dateStr), zap.Error(errDate))
			span.RecordError(errDate)
			dto.WriteError(
				w,
				models.SnapInvalidFormat.HttpCode,
				models.SnapInvalidFormat.GetResponseCode(svcCode),
				models.SnapInvalidFormat.ResponseMessage,
			)
			return
		}

		key := "transaction_summary"
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_SUMMARY, dateStr)
		h.logger.Info("Checking cache", zap.String("key", cacheKey), zap.String("date", dateStr))

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
					h.logger.Info("Cache Hit - Berhasil mengambil data summary transaksi",
						zap.String("source", "redis"),
						zap.String("date", dateStr),
						zap.Int("count", len(transactions)),
					)
					dto.WriteResponse(
						w,
						models.SnapSuccess.HttpCode,
						models.SnapSuccess.GetResponseCode(svcCode),
						models.SnapSuccess.ResponseMessage, map[string]any{
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
		h.logger.Info("Cache miss", zap.String("key", cacheKey), zap.String("date", dateStr))

		svcCtx, svcSpan := tracer.Start(ctx, "Fetch-from-Database")
		transactions, snapErr := h.svc.FetchSummaryToday(svcCtx, timeParse)
		svcSpan.End()

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

		span.SetAttributes(
			attribute.Int("result.count", len(transactions)),
			attribute.String("query.date", dateStr),
		)

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transactions); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		h.logger.Info("Berhasil mengambil data summary transaksi",
			zap.String("source", "database"),
			zap.String("date", dateStr),
			zap.Int("count", len(transactions)),
		)

		dto.WriteResponse(
			w,
			models.SnapSuccess.HttpCode,
			models.SnapSuccess.GetResponseCode(svcCode),
			models.SnapSuccess.ResponseMessage, map[string]any{
				"transactions": transactions,
			})
	}
}

// GET /transaction/{id}
func (h *TransactionsHandler) GetById() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		svcCode := config.SVC_CODE_TRX_HISTORY_DETAIL

		idStr := r.PathValue("id")
		h.logger.Info("Mendapatkan id transaction", zap.String("id", idStr))

		_, err := uuid.Parse(idStr)
		if err != nil {
			h.logger.Error("Invalid UUID format", zap.String("id", idStr), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(
				w,
				models.SnapInvalidFormat.HttpCode,
				models.SnapInvalidFormat.GetResponseCode(svcCode),
				models.SnapInvalidFormat.ResponseMessage,
			)
			return
		}

		key := "transaction_detail"
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_ID, idStr)
		h.logger.Info("Checking cache", zap.String("key", cacheKey), zap.String("id", idStr))

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
					h.logger.Info("Cache Hit - Berhasil mengambil data transaksi",
						zap.String("source", "redis"),
						zap.String("id", idStr),
					)
					dto.WriteResponse(
						w,
						models.SnapSuccess.HttpCode,
						models.SnapSuccess.GetResponseCode(svcCode),
						models.SnapSuccess.ResponseMessage, map[string]any{
							"transaction": transaction,
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
		h.logger.Info("Cache miss", zap.String("key", cacheKey), zap.String("id", idStr))

		svcCtx, svcSpan := tracer.Start(ctx, "Fetch-from-Database")
		transaction, snapErr := h.svc.FetchTransactionById(svcCtx, idStr)
		svcSpan.End()

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

		span.SetAttributes(
			attribute.String("query.id", idStr),
		)

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transaction); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).
			Observe(time.Since(cacheSetStart).Seconds())

		h.logger.Info("Berhasil mengambil data transaksi",
			zap.String("source", "database"),
			zap.String("id", idStr),
		)

		dto.WriteResponse(
			w,
			models.SnapSuccess.HttpCode,
			models.SnapSuccess.GetResponseCode(svcCode),
			models.SnapSuccess.ResponseMessage, map[string]any{
				"transaction": transaction,
			})
	}
}

// POST /transaction
// func (h *TransactionsHandler) Create() http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {

// 		ctx := r.Context()
// 		span, tracer := middleware.AllCtx(ctx)
// 		key := "transaction_list"

// 		var payload models.Transaction
// 		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
// 			h.logger.Error(models.ErrInvalidJsonFormat.Error(), zap.Error(err))
// 			span.RecordError(err)
// 			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
// 			return
// 		}

// 		h.logger.Info("Payload received", zap.Any("payload", payload))

// 		svcCtx, svcSpan := tracer.Start(ctx, "Create-Transaction")
// 		transactionID, err := h.svc.CreateTrx(svcCtx, payload)
// 		svcSpan.End()

// 		if err != nil {
// 			h.logger.Error(err.Error(), zap.Error(err))
// 			span.RecordError(err)
// 			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
// 			return
// 		}

// 		// Invalidate transaction_list cache
// 		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_LIST)
// 		cacheStart := time.Now()
// 		if err := h.rdb.Del(ctx, cacheKey).Err(); err != nil {
// 			metrics.CacheRequestsTotal.WithLabelValues(key, "error").Inc()
// 			span.RecordError(err)
// 			span.SetStatus(codes.Error, models.ErrRedisInvalidate.Error())
// 			h.logger.Error(models.ErrRedisInvalidate.Error(), zap.Error(err))
// 		} else {
// 			metrics.CacheRequestsTotal.WithLabelValues(key, "invalidate").Inc()
// 			span.AddEvent("Cache Invalidated")
// 		}

// 		metrics.CacheDuration.WithLabelValues("invalidate", key).
// 			Observe(time.Since(cacheStart).Seconds())

// 		// Invalidate transaction_summary cache (semua tanggal)
// 		summaryPattern := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_SUMMARY, "*")
// 		summaryKeys, errScan := h.rdb.Keys(ctx, summaryPattern).Result()
// 		if errScan == nil && len(summaryKeys) > 0 {
// 			if err := h.rdb.Del(ctx, summaryKeys...).Err(); err != nil {
// 				span.RecordError(err)
// 				h.logger.Warn("Failed to invalidate summary cache", zap.Error(err))
// 			} else {
// 				span.AddEvent("Summary Cache Invalidated")
// 				h.logger.Info("Summary cache invalidated", zap.Strings("keys", summaryKeys))
// 			}
// 		}

// 		span.SetAttributes(attribute.String("handler.result.id", transactionID))

// 		h.logger.Info("Transfer berhasil dilakukan",
// 			zap.String("source", "database"),
// 			zap.String("handler.result.id", transactionID),
// 		)

// 		dto.WriteResponse(w, http.StatusCreated, "Transfer berhasil dilakukan", map[string]any{
// 			"id":     transactionID,
// 			"amount": payload.Amount,
// 			"note":   payload.Note,
// 		})
// 	}
// }

// POST /v1.0/transfer-intrabank
func (h *TransactionsHandler) TransferIntraBank() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		key := "transfer_intra_bank"
		accountId := r.Header.Get("X-ACCOUNT-ID")
		svcCode := config.SVC_CODE_TRANSFER_INTRABANK

		snapHeader := models.ExtractSNAPHeader(r)
		if snapHeader.Timestamp == "" || snapHeader.PartnerID == "" ||
			snapHeader.ExternalID == "" || snapHeader.ChannelID == "" {
			h.logger.Error(models.SnapUnauthorized.ResponseMessage,
				zap.String("account_id", accountId),
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
			attribute.String("auth.sub", accountId),
		)

		// --- Decode request body ---
		var payload models.TransferIntrabankRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.logger.Error(models.ErrInvalidJsonFormat.Error(),
				zap.Error(err),
				zap.String("account_id", accountId),
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

		// --- Validasi field wajib ---
		if payload.PartnerReferenceNo == "" || payload.Amount.Value == "" ||
			payload.BeneficiaryAccountNo == "" || payload.SourceAccountNo == "" {
			h.logger.Error("Field wajib tidak lengkap",
				zap.String("account_id", accountId),
				zap.String("partner_reference_no", payload.PartnerReferenceNo),
				zap.String("source_account_no", payload.SourceAccountNo),
				zap.String("beneficiary_account_no", payload.BeneficiaryAccountNo),
			)
			span.SetStatus(codes.Error, "field wajib tidak lengkap")
			dto.WriteError(
				w,
				models.SnapMandatoryField.HttpCode,
				models.SnapMandatoryField.GetResponseCode(svcCode),
				models.SnapMandatoryField.ResponseMessage,
			)
			return
		}

		h.logger.Info("Payload transfer intrabank diterima",
			zap.String("account_id", accountId),
			zap.String("partner_reference_no", payload.PartnerReferenceNo),
			zap.String("source_account_no", payload.SourceAccountNo),
			zap.String("beneficiary_account_no", payload.BeneficiaryAccountNo),
			zap.String("amount", payload.Amount.Value),
			zap.String("currency", payload.Amount.Currency),
		)

		// --- Proses transfer via service ---
		svcCtx, svcSpan := tracer.Start(ctx, "Create-TransferIntrabank")
		svcSpan.SetAttributes(
			attribute.String("db.partner_reference_no", payload.PartnerReferenceNo),
			attribute.String("db.source_account_no", payload.SourceAccountNo),
			attribute.String("db.beneficiary_account_no", payload.BeneficiaryAccountNo),
			attribute.String("db.amount", payload.Amount.Value),
		)
		referenceNo, err := h.svc.TransferIntrabank(svcCtx, accountId, snapHeader, payload)
		svcSpan.End()

		if err != nil {

			er := errors.New(err.ResponseMessage)

			h.logger.Error(err.ResponseMessage, zap.Error(er))
			span.RecordError(er)
			span.SetStatus(codes.Error, err.ResponseMessage)
			dto.WriteError(
				w,
				err.HttpCode,
				err.GetResponseCode(svcCode),
				err.ResponseMessage,
			)
			return
		}

		// Invalidate transaction_list cache
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_LIST)
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

		// Invalidate transaction_summary cache (semua tanggal)
		summaryPattern := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_SUMMARY, "*")
		summaryKeys, errScan := h.rdb.Keys(ctx, summaryPattern).Result()
		if errScan == nil && len(summaryKeys) > 0 {
			if err := h.rdb.Del(ctx, summaryKeys...).Err(); err != nil {
				span.RecordError(err)
				h.logger.Warn("Failed to invalidate summary cache", zap.Error(err))
			} else {
				span.AddEvent("Summary Cache Invalidated")
				h.logger.Info("Summary cache invalidated", zap.Strings("keys", summaryKeys))
			}
		}

		span.SetStatus(codes.Ok, "transfer intrabank berhasil")
		span.SetAttributes(attribute.String("snap.reference_no", referenceNo))

		h.logger.Info("Transfer intrabank berhasil dilakukan",
			zap.String("account_id", accountId),
			zap.String("reference_no", referenceNo),
			zap.String("partner_reference_no", payload.PartnerReferenceNo),
			zap.String("amount", payload.Amount.Value),
		)

		responseBody := models.TransferIntrabankResponse{
			ReferenceNo:          referenceNo,
			PartnerReferenceNo:   payload.PartnerReferenceNo,
			Amount:               payload.Amount,
			BeneficiaryAccountNo: payload.BeneficiaryAccountNo,
			Currency:             payload.Currency,
			CustomerReference:    payload.CustomerReference,
			SourceAccount:        payload.SourceAccountNo,
			TransactionDate:      payload.TransactionDate,
			OriginatorInfos:      payload.OriginatorInfos,
			AdditionalInfo:       payload.AdditionalInfo, // ini masih oke karena response pakai struct
		}

		dto.WriteResponse(
			w,
			models.SnapSuccess.HttpCode,
			models.SnapSuccess.GetResponseCode(svcCode),
			models.SnapSuccess.ResponseMessage, responseBody)
	}
}
