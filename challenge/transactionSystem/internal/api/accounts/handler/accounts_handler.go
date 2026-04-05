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

	keyManager := helper.NewRedisKeyManager("transaction_system", "bank")
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

		cacheStart := time.Now()
		ctx := r.Context()
		span, logger, tracer := middleware.AllCtx(ctx)

		cacheKey := h.keyManager.Generate(config.REDIS_KEY_ACCOUNT_LIST)
		logger.Info("Checking cache", zap.String("key", cacheKey))

		cacheCtx, cacheSpan := tracer.Start(ctx, "Cache-Lookup")

		val, errRedis := h.rdb.Get(cacheCtx, cacheKey).Bytes()
		cacheDuration := time.Since(cacheStart).Seconds()

		metrics.CacheDuration.WithLabelValues(
			"get",
			"account_list",
		).Observe(cacheDuration)

		cacheSpan.End()

		if errRedis == nil {

			metrics.CacheRequestsTotal.WithLabelValues(
				"account_list",
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
				"account_list",
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
		helper.SaveToCacheCompressed(ctx, h.rdb, cacheKey, accounts)

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

		idStr := r.PathValue("id")
		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Mendapatkan id account = %s", idStr))

		_, err := uuid.Parse(idStr)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, models.ErrInvalidUuid.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		account, err := h.svc.FetchAccountById(idStr)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil data akun dengan id = %s", idStr))
		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data akun dengan id = %s", idStr), map[string]any{
			"account": account,
		})
	}
}

// GET /account/{id}/transactions?type=all/in/out
func (h *AccountsHandler) GetTransactions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		trxTypeEnum := []string{"all", "in", "out"}

		idStr := r.PathValue("id")
		trxType := r.URL.Query().Get("type")
		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Mendapatkan id account = %s", idStr))

		// Valid Uuid
		_, err := uuid.Parse(idStr)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, models.ErrInvalidUuid.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		if trxType == "" {
			trxType = "all"
		}

		isValidType := slices.Contains(trxTypeEnum, trxType)

		// Valid Trx Type
		if !isValidType {
			helper.PrintLog("account", helper.LogPositionHandler, models.ErrInvalidTrxType.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidTrxType), models.ErrInvalidTrxType.Error())
			return
		}

		// Exec
		transactions, err := h.svc.FetchTransactionsByAccountId(idStr, trxType)
		if err != nil {
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		// Success
		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil data transaksi terkait akun dengan id = %s", idStr))
		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data transaksi terkait akun dengan id = %s", idStr), map[string]any{
			"transactions": transactions,
		})
	}
}

// POST /account
func (h *AccountsHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var payload models.Account
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, models.ErrInvalidJsonFormat.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		if payload.BankCode == "" || payload.AccountNumber == "" || payload.AccountHolder == "" {
			helper.PrintLog("account", helper.LogPositionHandler, models.ErrInvalidField.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidField), models.ErrInvalidField.Error())
			return
		}

		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))

		newAccount, err := h.svc.CreateNewAccount(payload)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Berhasil membuat akun baru : %+v", newAccount))
		dto.WriteResponse(w, http.StatusCreated, "Berhasil membuat data account baru", map[string]any{
			"account": newAccount,
		})
	}
}

// PATCH /account/{id}
func (h *AccountsHandler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")
		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Mendapatkan id account = %s", idStr))

		// Valid Uuid
		_, err := uuid.Parse(idStr)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, models.ErrInvalidUuid.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		var payload models.Account
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			helper.PrintLog("account", helper.LogPositionRepo, models.ErrInvalidJsonFormat.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))

		// Jika tidak ada field yang diupdate
		if payload.AccountHolder == "" && payload.AccountNumber == "" {
			helper.PrintLog("account", helper.LogPositionRepo, models.ErrInvalidField.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidField), models.ErrInvalidField.Error())
			return
		}

		accountIdParse, err := uuid.Parse(idStr)
		if err != nil {
			// Jika gagal di-parse, kembalikan error validasi
			helper.PrintLog("account", helper.LogPositionRepo, models.ErrInvalidUuid.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		payload.ID = accountIdParse

		updatedId, err := h.svc.PatchAccountById(payload)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionRepo, err.Error())
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		helper.PrintLog("account", helper.LogPositionRepo, "Berhasil mengupdate data akun")
		dto.WriteResponse(w, http.StatusOK, "Berhasil mengupdate data account", map[string]any{
			"id": updatedId,
		})
	}
}

// DELETE /account/{id}
func (h *AccountsHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")
		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Mendapatkan id account = %s", idStr))

		// Valid Uuid
		_, errId := uuid.Parse(idStr)
		if errId != nil {
			helper.PrintLog("account", helper.LogPositionHandler, models.ErrInvalidUuid.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		err := h.svc.DeleteAccountById(idStr)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil menghapus akun dengan id : %s", idStr), map[string]any{})
	}
}
