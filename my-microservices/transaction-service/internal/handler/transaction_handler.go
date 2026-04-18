package handler

import (
	"encoding/json"
	"errors"
	"my-microservices/transaction-service/config"
	"my-microservices/transaction-service/helper"
	"my-microservices/transaction-service/internal/domain"
	"my-microservices/transaction-service/internal/dto"
	"my-microservices/transaction-service/internal/kafka"
	"my-microservices/transaction-service/internal/middleware"
	"my-microservices/transaction-service/internal/repository"
	"my-microservices/transaction-service/internal/service"
	"my-microservices/transaction-service/observability/metrics"
	"net/http"
	"time"

	pbAccount "my-microservices/shared/pb/account"
	pbFraud "my-microservices/shared/pb/fraud"

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
	producer    *kafka.Producer
}

func NewTransactionsHandler(mux *http.ServeMux, db *sqlx.DB, rdb *redis.Client, producer *kafka.Producer, accountCli pbAccount.AccountGRPCServiceClient, fraudCli pbFraud.FraudServiceClient) *TransactionsHandler {
	trxRepo := repository.NewTransactionRepository(db)
	keyManager := helper.NewRedisKeyManager("transaction_service", config.DOMAIN_TRANSACTION)
	TrxSvc := service.NewTransactionsService(trxRepo, accountCli, fraudCli, rdb, keyManager)
	idempotency := middleware.NewIdempotencyMiddleware(rdb, keyManager)
	logger := helper.Log

	return &TransactionsHandler{
		mux:         mux,
		svc:         TrxSvc,
		rdb:         rdb,
		keyManager:  keyManager,
		idempotency: idempotency,
		logger:      logger,
		producer:    producer,
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
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, version, "/transactions/account/{accountNo}"),
		middleware.ValidateSNAPToken(
			obs.Wrap("TransactionHandler.GetByAccountNo", config.DOMAIN_TRANSACTION, a.GetByAccountNo()),
		).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, version, "/transfer-intrabank"),
		middleware.ValidateSNAPToken(
			obs.Wrap("TransactionHandler.TransferIntraBank", config.DOMAIN_TRANSACTION, a.idempotency.Check(a.TransferIntraBank())),
		).ServeHTTP,
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, version, "/topup"),
		middleware.ValidateSNAPToken(
			obs.Wrap("TransactionHandler.Topup", config.DOMAIN_TRANSACTION, a.idempotency.Check(a.Topup())),
		).ServeHTTP,
	)
}

// GET /v1.0/transactions
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
		metrics.CacheDuration.WithLabelValues("get", key).Observe(time.Since(cacheStart).Seconds())
		cacheSpan.End()

		if errRedis == nil {
			metrics.CacheRequestsTotal.WithLabelValues(key, "hit").Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var transactions []domain.Transaction
				if err := json.Unmarshal(decompressed, &transactions); err == nil {
					span.AddEvent("Cache hit occurred")
					dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
						"Berhasil mendapatkan list data transaksi",
						map[string]any{"transactions": transactions})
					return
				}
			}
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(key, "miss").Inc()
		}

		svcCtx, svcSpan := tracer.Start(ctx, "Fetch-from-Database")
		transactions, snapErr := h.svc.FetchAllTransactions(svcCtx)
		svcSpan.End()

		if snapErr != nil {
			prefixErr := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixErr.Error(), zap.Error(prefixErr))
			span.RecordError(prefixErr)
			dto.WriteError(w, snapErr.HttpCode, snapErr.GetResponseCode(svcCode), snapErr.ResponseMessage)
			return
		}

		span.SetAttributes(attribute.Int("result.count", len(transactions)))

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transactions); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).Observe(time.Since(cacheSetStart).Seconds())

		dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
			"Berhasil mendapatkan list data transaksi",
			map[string]any{"transactions": transactions})
	}
}

// GET /v1.0/transactions/summary?date=YYYY-MM-DD
func (h *TransactionsHandler) GetSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		svcCode := config.SVC_CODE_TRX_HISTORY_LIST

		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}

		timeParse, errDate := time.Parse("2006-01-02", dateStr)
		if errDate != nil {
			span.RecordError(errDate)
			dto.WriteError(w, domain.SnapInvalidFormat.HttpCode, domain.SnapInvalidFormat.GetResponseCode(svcCode), "Data tanggal tidak valid")
			return
		}

		key := "transaction_summary"
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_SUMMARY, dateStr)

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()
		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()
		metrics.CacheDuration.WithLabelValues("get", key).Observe(time.Since(cacheStart).Seconds())
		cacheSpan.End()

		if errRedis == nil {
			metrics.CacheRequestsTotal.WithLabelValues(key, "hit").Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var transactions []domain.Transaction
				if err := json.Unmarshal(decompressed, &transactions); err == nil {
					dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
						"Berhasil mendapatkan list data transaksi berdasarkan tanggal",
						map[string]any{"transactions": transactions})
					return
				}
			}
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(key, "miss").Inc()
		}

		svcCtx, svcSpan := tracer.Start(ctx, "Fetch-from-Database")
		transactions, snapErr := h.svc.FetchSummaryToday(svcCtx, timeParse)
		svcSpan.End()

		if snapErr != nil {
			prefixErr := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixErr.Error(), zap.Error(prefixErr))
			span.RecordError(prefixErr)
			dto.WriteError(w, snapErr.HttpCode, snapErr.GetResponseCode(svcCode), snapErr.ResponseMessage)
			return
		}

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transactions); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).Observe(time.Since(cacheSetStart).Seconds())

		dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
			"Berhasil mendapatkan list data transaksi berdasarkan tanggal",
			map[string]any{"transactions": transactions})
	}
}

// GET /v1.0/transaction/{id}
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
			dto.WriteError(w, domain.SnapInvalidFormat.HttpCode, domain.SnapInvalidFormat.GetResponseCode(svcCode), "UUID tidak valid")
			return
		}

		key := "transaction_detail"
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_ID, idStr)

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()
		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()
		metrics.CacheDuration.WithLabelValues("get", key).Observe(time.Since(cacheStart).Seconds())
		cacheSpan.End()

		if errRedis == nil {
			metrics.CacheRequestsTotal.WithLabelValues(key, "hit").Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var transaction domain.Transaction
				if err := json.Unmarshal(decompressed, &transaction); err == nil {
					dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
						"Berhasil mendapatkan data transaksi",
						map[string]any{"transaction": transaction})
					return
				}
			}
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(key, "miss").Inc()
		}

		svcCtx, svcSpan := tracer.Start(ctx, "Fetch-from-Database")
		transaction, snapErr := h.svc.FetchTransactionById(svcCtx, idStr)
		svcSpan.End()

		if snapErr != nil {
			prefixErr := errors.New(snapErr.ResponseMessage)
			h.logger.Error(prefixErr.Error(), zap.Error(prefixErr))
			span.RecordError(prefixErr)
			dto.WriteError(w, snapErr.HttpCode, snapErr.GetResponseCode(svcCode), snapErr.ResponseMessage)
			return
		}

		cacheSetStart := time.Now()
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transaction); err != nil {
			h.logger.Warn("Failed to save to cache", zap.Error(err))
		}
		metrics.CacheDuration.WithLabelValues("set", key).Observe(time.Since(cacheSetStart).Seconds())

		dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
			"Berhasil mendapatkan data transaksi",
			map[string]any{"transaction": transaction})
	}
}

// GET /v1.0/transactions/account/{accountNo}
func (h *TransactionsHandler) GetByAccountNo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		svcCode := config.SVC_CODE_TRX_HISTORY_LIST

		accountNo := r.PathValue("accountNo")
		h.logger.Info("Mendapatkan history transaksi by account_no", zap.String("accountNo", accountNo))

		if accountNo == "" {
			prefixErr := errors.New("Nomor rekening tidak boleh kosong")
			h.logger.Error(prefixErr.Error(), zap.Error(prefixErr))
			span.RecordError(prefixErr)
			dto.WriteError(w, domain.SnapInvalidFormat.HttpCode, domain.SnapInvalidFormat.GetResponseCode(svcCode), "Nomor rekening tidak boleh kosong")
			return
		}

		key := "transaction_account"
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_LIST, accountNo)

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")
		cacheStart := time.Now()
		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()
		metrics.CacheDuration.WithLabelValues("get", key).Observe(time.Since(cacheStart).Seconds())
		cacheSpan.End()

		if errRedis == nil {
			metrics.CacheRequestsTotal.WithLabelValues(key, "hit").Inc()

			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var transactions []domain.Transaction
				if err := json.Unmarshal(decompressed, &transactions); err == nil {
					dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
						"Berhasil mendapatkan data transaksi accout",
						map[string]any{"transactions": transactions})
					return
				}
			}
		}

		svcCtx, svcSpan := tracer.Start(ctx, "Fetch-from-Database")
		transactions, snapErr := h.svc.FetchTransactionsByAccountNo(svcCtx, accountNo)
		svcSpan.End()

		if snapErr != nil {
			dto.WriteError(w, snapErr.HttpCode, snapErr.GetResponseCode(svcCode), snapErr.ResponseMessage)
			return
		}

		// Save to cache
		if err := helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, transactions); err != nil {
			h.logger.Warn("Failed to save account history to cache", zap.Error(err))
		}

		dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
			"Berhasil mendapatkan data transaksi account",
			map[string]any{"transactions": transactions})
	}
}

// POST /v1.0/transfer-intrabank
func (h *TransactionsHandler) TransferIntraBank() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		key := "transfer_intra_bank"
		accountId := r.Header.Get("X-ACCOUNT-ID")
		svcCode := config.SVC_CODE_TRANSFER_INTRABANK

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
			attribute.String("auth.sub", accountId),
		)

		var payload domain.TransferIntrabankRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.logger.Error(domain.ErrInvalidJsonFormat.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, domain.SnapInvalidFormat.HttpCode, domain.SnapInvalidFormat.GetResponseCode(svcCode), err.Error())
			return
		}

		if payload.PartnerReferenceNo == "" || payload.Amount.Value == "" ||
			payload.BeneficiaryAccountNo == "" || payload.SourceAccountNo == "" {
			h.logger.Error("Field wajib tidak lengkap")
			span.SetStatus(codes.Error, "field wajib tidak lengkap")
			dto.WriteError(w, domain.SnapMandatoryField.HttpCode, domain.SnapMandatoryField.GetResponseCode(svcCode), "Payload tidak lengkap")
			return
		}

		payload.ExternalID = snapHeader.ExternalID

		svcCtx, svcSpan := tracer.Start(ctx, "Create-TransferIntrabank")
		svcSpan.SetAttributes(
			attribute.String("db.partner_reference_no", payload.PartnerReferenceNo),
			attribute.String("db.source_account_no", payload.SourceAccountNo),
			attribute.String("db.beneficiary_account_no", payload.BeneficiaryAccountNo),
		)

		referenceNo, snapErr := h.svc.TransferIntrabank(svcCtx, accountId, h.producer, payload, svcCode)
		svcSpan.End()

		if snapErr != nil {
			errWrapper := errors.New(snapErr.ResponseMessage)
			h.logger.Error(snapErr.ResponseMessage, zap.Error(errWrapper))
			span.RecordError(errWrapper)
			span.SetStatus(codes.Error, snapErr.ResponseMessage)
			dto.WriteError(w, snapErr.HttpCode, snapErr.GetResponseCode(svcCode), snapErr.ResponseMessage)
			return
		}

		// Invalidate cache
		cacheKey := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_LIST)
		cacheStart := time.Now()
		if err := h.rdb.Del(ctx, cacheKey).Err(); err != nil {
			metrics.CacheRequestsTotal.WithLabelValues(key, "error").Inc()
			span.RecordError(err)
			h.logger.Error(domain.ErrRedisInvalidate.Error(), zap.Error(err))
		} else {
			metrics.CacheRequestsTotal.WithLabelValues(key, "invalidate").Inc()
		}
		metrics.CacheDuration.WithLabelValues("invalidate", key).Observe(time.Since(cacheStart).Seconds())

		// Invalidate summary
		summaryPattern := h.keyManager.Generate(config.REDIS_KEY_TRANSACTION_SUMMARY, "*")
		summaryKeys, errScan := h.rdb.Keys(ctx, summaryPattern).Result()
		if errScan == nil && len(summaryKeys) > 0 {
			if err := h.rdb.Del(ctx, summaryKeys...).Err(); err != nil {
				h.logger.Warn("Failed to invalidate summary cache", zap.Error(err))
			}
		}

		span.SetStatus(codes.Ok, "transfer intrabank berhasil")
		span.SetAttributes(attribute.String("snap.reference_no", referenceNo))
		h.logger.Info("Transfer intrabank berhasil", zap.String("reference_no", referenceNo))

		responseBody := domain.TransferIntrabankResponse{
			ReferenceNo:          referenceNo,
			PartnerReferenceNo:   payload.PartnerReferenceNo,
			Amount:               domain.Amount{Value: payload.Amount.Value, Currency: payload.Amount.Currency},
			BeneficiaryAccountNo: payload.BeneficiaryAccountNo,
			Currency:             payload.Currency,
			CustomerReference:    payload.CustomerReference,
			SourceAccount:        payload.SourceAccountNo,
			TransactionDate:      payload.TransactionDate,
			OriginatorInfos:      payload.OriginatorInfos,
			AdditionalInfo:       payload.AdditionalInfo,
		}

		dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
			"Transfer berhasil dilakukan",
			responseBody)
	}
}

// POST /v1.0/topup
func (h *TransactionsHandler) Topup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		span, tracer := middleware.AllCtx(ctx)
		accountId := r.Header.Get("X-ACCOUNT-ID")
		svcCode := config.SVC_CODE_TRANSFER_INTRABANK // Reuse or add new

		snapHeader := domain.ExtractSNAPHeader(r)
		span.SetAttributes(
			attribute.String("snap.partner_id", snapHeader.PartnerID),
			attribute.String("snap.external_id", snapHeader.ExternalID),
			attribute.String("auth.sub", accountId),
		)

		var payload domain.TopupRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.logger.Error(domain.ErrInvalidJsonFormat.Error(), zap.Error(err))
			span.RecordError(err)
			dto.WriteError(w, domain.SnapInvalidFormat.HttpCode, domain.SnapInvalidFormat.GetResponseCode(svcCode), err.Error())
			return
		}

		// Validation
		if payload.PartnerReferenceNo == "" || payload.Amount.Value == "" || payload.SourceAccountNo == "" {
			h.logger.Error("Field wajib tidak lengkap")
			span.SetStatus(codes.Error, "field wajib tidak lengkap")
			dto.WriteError(w, domain.SnapMandatoryField.HttpCode, domain.SnapMandatoryField.GetResponseCode(svcCode), "Field tidak lengkap")
			return
		}

		payload.ExternalID = snapHeader.ExternalID

		svcCtx, svcSpan := tracer.Start(ctx, "Service-Topup")
		svcSpan.SetAttributes(
			attribute.String("db.partner_reference_no", payload.PartnerReferenceNo),
			attribute.String("db.source_account_no", payload.SourceAccountNo),
		)

		referenceNo, snapErr := h.svc.Topup(svcCtx, accountId, h.producer, payload, svcCode)
		svcSpan.End()

		if snapErr != nil {
			errWrapper := errors.New(snapErr.ResponseMessage)
			h.logger.Error(snapErr.ResponseMessage, zap.Error(errWrapper))
			span.RecordError(errWrapper)
			span.SetStatus(codes.Error, snapErr.ResponseMessage)
			dto.WriteError(w, snapErr.HttpCode, snapErr.GetResponseCode(svcCode), snapErr.ResponseMessage)
			return
		}

		span.SetStatus(codes.Ok, "topup berhasil")
		span.SetAttributes(attribute.String("snap.reference_no", referenceNo))
		h.logger.Info("Topup berhasil dilakukan", zap.String("reference_no", referenceNo))

		dto.WriteResponse(w, domain.SnapSuccess.HttpCode, domain.SnapSuccess.GetResponseCode(svcCode),
			"Topup berhasil dilakukan",
			domain.TopupResponse{
				ReferenceNo:        referenceNo,
				PartnerReferenceNo: payload.PartnerReferenceNo,
				Amount:             payload.Amount,
				SourceAccountNo:    payload.SourceAccountNo,
			})
	}
}
