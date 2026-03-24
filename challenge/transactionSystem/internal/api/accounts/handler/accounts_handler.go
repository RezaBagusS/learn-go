package handler

import (
	"belajar-go/challenge/transactionSystem/dto"
	"belajar-go/challenge/transactionSystem/internal/api/accounts/repository"
	"belajar-go/challenge/transactionSystem/internal/api/accounts/service"
	"belajar-go/challenge/transactionSystem/internal/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type AccountsHandler struct {
	mux *http.ServeMux
	svc service.AccountsService
}

func NewAccountsHandler(mux *http.ServeMux, db *sqlx.DB) *AccountsHandler {
	accountRepo := repository.NewAccountRepository(db)
	accountSvc := service.NewAccountsService(accountRepo)

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
			dto.WriteError(w, http.StatusInternalServerError, err.Error())
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

		account, err := h.svc.FetchAccountById(idStr)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}

		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil data akun dengan id = %s", idStr))
		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data akun dengan id = %s", idStr), map[string]any{
			"account": account,
		})
	}
}

// POST /account
func (h *AccountsHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var payload models.Account
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			dto.WriteError(w, http.StatusBadRequest, "Format JSON tidak valid!")
			return
		}

		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))

		newBank, err := h.svc.CreateNewAccount(payload)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusCreated, "Berhasil membuat data account baru", map[string]any{
			"account": newBank,
		})
	}
}

// PATCH /account/{id}
func (h *AccountsHandler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")
		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Mendapatkan id account = %s", idStr))

		var payload models.Account
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			dto.WriteError(w, http.StatusBadRequest, "Format JSON tidak valid!")
			return
		}

		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))
		payload.ID = uuid.MustParse(idStr)

		updatedId, err := h.svc.PatchAccountById(payload)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

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

		err := h.svc.DeleteAccountById(idStr)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil menghapus akun dengan id : %s", idStr), map[string]any{})
	}
}
