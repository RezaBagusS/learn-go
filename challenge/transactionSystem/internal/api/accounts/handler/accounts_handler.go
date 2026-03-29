package handler

import (
	"belajar-go/challenge/transactionSystem/dto"
	"belajar-go/challenge/transactionSystem/internal/api/accounts/repository"
	"belajar-go/challenge/transactionSystem/internal/api/accounts/service"
	bankRepository "belajar-go/challenge/transactionSystem/internal/api/banks/repository"
	bankService "belajar-go/challenge/transactionSystem/internal/api/banks/service"
	"belajar-go/challenge/transactionSystem/internal/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type AccountsHandler struct {
	mux *http.ServeMux
	svc service.AccountsService
}

func StatusCodeHandler(err error) int {
	var statusCode int
	switch {
	case errors.Is(err, models.ErrIdNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, models.ErrInvalidUuid), errors.Is(err, models.ErrInvalidInitBalance), errors.Is(err, models.ErrInvalidJsonFormat), errors.Is(err, models.ErrInvalidTrxType), errors.Is(err, models.ErrInvalidBankCode), errors.Is(err, models.ErrInvalidField):
		statusCode = http.StatusBadRequest
	case errors.Is(err, models.ErrDuplicateAccount):
		statusCode = http.StatusConflict
	default:
		statusCode = http.StatusInternalServerError
	}
	return statusCode
}

func NewAccountsHandler(mux *http.ServeMux, db *sqlx.DB) *AccountsHandler {

	bankRepo := bankRepository.NewBankRepository(db)
	bankSvc := bankService.NewBanksService(bankRepo)

	accountRepo := repository.NewAccountRepository(db)
	accountSvc := service.NewAccountsService(accountRepo, bankSvc)

	return &AccountsHandler{
		mux: mux,
		svc: accountSvc,
	}
}

func (a *AccountsHandler) MapRoutes() {
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/accounts"),
		a.GetAll(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/account/{id}"),
		a.GetById(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/account/{id}/transactions"),
		a.GetTransactions(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, "/account"),
		a.Create(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPatch, "/account/{id}"),
		a.Update(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodDelete, "/account/{id}"),
		a.Delete(),
	)
}

// GET /accounts
func (h *AccountsHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		helper.PrintLog("account", helper.LogPositionHandler, "Mengambil seluruh data account ...")

		accounts, err := h.svc.FetchAllAccounts()
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, StatusCodeHandler(err), err.Error())
			return
		}

		helper.PrintLog("account", helper.LogPositionHandler, "Berhasil mengambil list data akun")
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
			dto.WriteError(w, StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		account, err := h.svc.FetchAccountById(idStr)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, StatusCodeHandler(err), err.Error())
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
			dto.WriteError(w, StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		if trxType == "" {
			trxType = "all"
		}

		isValidType := slices.Contains(trxTypeEnum, trxType)

		// Valid Trx Type
		if !isValidType {
			helper.PrintLog("account", helper.LogPositionHandler, models.ErrInvalidTrxType.Error())
			dto.WriteError(w, StatusCodeHandler(models.ErrInvalidTrxType), models.ErrInvalidTrxType.Error())
			return
		}

		// Exec
		transactions, err := h.svc.FetchTransactionsByAccountId(idStr, trxType)
		if err != nil {
			dto.WriteError(w, StatusCodeHandler(err), err.Error())
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
			dto.WriteError(w, StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		if payload.BankCode == "" || payload.AccountNumber == "" || payload.AccountHolder == "" {
			helper.PrintLog("account", helper.LogPositionHandler, models.ErrInvalidField.Error())
			dto.WriteError(w, StatusCodeHandler(models.ErrInvalidField), models.ErrInvalidField.Error())
			return
		}

		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))

		newAccount, err := h.svc.CreateNewAccount(payload)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, StatusCodeHandler(err), err.Error())
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
			dto.WriteError(w, StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		var payload models.Account
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			helper.PrintLog("account", helper.LogPositionRepo, models.ErrInvalidJsonFormat.Error())
			dto.WriteError(w, StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))

		// Jika tidak ada field yang diupdate
		if payload.AccountHolder == "" && payload.AccountNumber == "" {
			helper.PrintLog("account", helper.LogPositionRepo, models.ErrInvalidField.Error())
			dto.WriteError(w, StatusCodeHandler(models.ErrInvalidField), models.ErrInvalidField.Error())
			return
		}

		accountIdParse, err := uuid.Parse(idStr)
		if err != nil {
			// Jika gagal di-parse, kembalikan error validasi
			helper.PrintLog("account", helper.LogPositionRepo, models.ErrInvalidUuid.Error())
			dto.WriteError(w, StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		payload.ID = accountIdParse

		updatedId, err := h.svc.PatchAccountById(payload)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionRepo, err.Error())
			dto.WriteError(w, StatusCodeHandler(err), err.Error())
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
			dto.WriteError(w, StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		err := h.svc.DeleteAccountById(idStr)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, StatusCodeHandler(err), err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil menghapus akun dengan id : %s", idStr), map[string]any{})
	}
}
