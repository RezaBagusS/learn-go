package handler

import (
	"belajar-go/challenge/transactionSystem/config"
	"belajar-go/challenge/transactionSystem/dto"
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/banks/repository"
	"belajar-go/challenge/transactionSystem/internal/api/banks/service"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"belajar-go/challenge/transactionSystem/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type BanksHandler struct {
	mux         *http.ServeMux
	svc         service.BankService
	rdb         *redis.Client
	keyManager  *helper.RedisKeyManager
	idempotency *middleware.IdempotencyMiddleware
}

const (
	REDIS_KEY_BANK_LIST = "bank_list"
	REDIS_KEY_BANK_ID   = "bank_id"
)

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

func (a *BanksHandler) MapRoutes() {
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/banks"),
		a.GetAll(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/bank/{identifier}"),
		a.GetById(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, "/bank"),
		a.idempotency.Check(a.Create()),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPatch, "/bank/{id}"),
		a.idempotency.Check(a.Update()),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodDelete, "/bank/{id}"),
		a.Delete(),
	)
}

func (h *BanksHandler) saveToCacheCompressed(ctx context.Context, key string, data any) {
	jsonData, _ := json.Marshal(data)

	compressed, err := helper.CompressData(jsonData)
	if err != nil {
		helper.PrintLog("redis", helper.LogPositionHandler, "Gagal kompresi: "+err.Error())
		return
	}

	err = h.rdb.Set(ctx, key, compressed, config.TimeCache).Err()
	if err != nil {
		helper.PrintLog("redis", helper.LogPositionHandler, "Peringatan: Gagal menyimpan cache ke Redis: "+err.Error())
	}
}

// GET /banks
func (h *BanksHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		cacheKey := h.keyManager.Generate(REDIS_KEY_BANK_LIST)
		val, errRedis := h.rdb.Get(ctx, cacheKey).Bytes()

		helper.Log.Info("Mencari data bank",
			zap.String("module", "bank"),
			zap.String("position", string(helper.LogPositionHandler)),
			// zap.String("identifier", idStr),
		)

		if errRedis == nil {
			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var banks []models.Bank
				if err := json.Unmarshal(decompressed, &banks); err == nil {
					helper.Log.Info("Cache Hit - Berhasil mengambil list data bank",
						zap.String("module", "bank"),
						zap.String("bank_name", string(helper.LogPositionHandler)),
					)
					dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil list data bank", map[string]any{"banks": banks})
					return
				}
			}
		}

		banks, err := h.svc.FetchAllBanks()
		if err != nil {
			helper.Log.Error(err.Error(),
				zap.String("module", "bank"),
				zap.Error(err),
			)
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		h.saveToCacheCompressed(ctx, cacheKey, banks)

		helper.Log.Info("Berhasil mengambil list data bank",
			zap.String("module", "bank"),
			zap.Int("count", len(banks)),
			zap.Any("data", banks),
		)
		helper.PrintLog("bank", helper.LogPositionHandler, "Berhasil mengambil list data bank")
		dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil list data bank", map[string]any{
			"banks": banks,
		})
	}
}

// GET /bank/{identifier}
func (h *BanksHandler) GetById() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("identifier")
		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Mencari bank dengan keyword: %s", idStr))

		ctx := r.Context()
		cacheKey := h.keyManager.Generate(REDIS_KEY_BANK_ID + ":" + idStr)
		val, errRedis := h.rdb.Get(ctx, cacheKey).Bytes()

		if errRedis == nil {
			decompressed, err := helper.DecompressData(val)
			if err == nil {
				var banks models.Bank
				if err := json.Unmarshal(decompressed, &banks); err == nil {
					helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Cache Hit - Berhasil mengambil data bank dengan code = %s", idStr))
					dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data bank dengan code = %s", idStr), map[string]any{
						"bank": banks,
					})
					return
				}
			}
		}

		bank, err := h.svc.FetchBankById(idStr)
		if err != nil {
			helper.PrintLog("bank", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		h.saveToCacheCompressed(ctx, cacheKey, bank)

		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil data bank dengan code = %s", idStr))
		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data bank dengan code = %s", idStr), map[string]any{
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
		cacheBankList := h.keyManager.Generate(REDIS_KEY_BANK_LIST)
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
		cacheBankList := h.keyManager.Generate(REDIS_KEY_BANK_LIST)
		cacheBankId := h.keyManager.Generate(REDIS_KEY_BANK_ID + ":" + returnedId)
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
		cacheBankList := h.keyManager.Generate(REDIS_KEY_BANK_LIST)
		cacheBankId := h.keyManager.Generate(REDIS_KEY_BANK_ID)
		errDel := h.rdb.Del(ctx, cacheBankList, cacheBankId).Err()
		if errDel != nil {
			helper.PrintLog("redis", helper.LogPositionHandler, "Gagal menghapus cache: "+errDel.Error())
		}

		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Berhasil menghapus bank : %s", bankId))
		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil menghapus bank : %s", bankId), map[string]any{})
	}
}
