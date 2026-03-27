package handler

import (
	"belajar-go/challenge/transactionSystem/dto"
	"belajar-go/challenge/transactionSystem/internal/api/banks/repository"
	"belajar-go/challenge/transactionSystem/internal/api/banks/service"
	"belajar-go/challenge/transactionSystem/internal/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BanksHandler struct {
	mux *http.ServeMux
	svc service.BankService
}

func NewBanksHandler(mux *http.ServeMux, db *sqlx.DB) *BanksHandler {
	bankRepo := repository.NewBankRepository(db)
	bankSvc := service.NewBanksService(bankRepo)

	return &BanksHandler{
		mux: mux,
		svc: bankSvc,
	}
}

func (a *BanksHandler) MapRoutes() {
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/banks"),
		a.GetAll(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, "/bank"),
		a.Create(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPatch, "/bank/{id}"),
		a.Update(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodDelete, "/bank/{id}"),
		a.Delete(),
	)
}

// GET /banks
func (h *BanksHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		helper.PrintLog("bank", helper.LogPositionHandler, "Mengambil seluruh data bank ...")

		banks, err := h.svc.FetchAllBanks()
		if err != nil {
			helper.PrintLog("bank", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}

		helper.PrintLog("bank", helper.LogPositionHandler, "Berhasil mengambil list data bank")
		dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil list data bank", map[string]any{
			"banks": banks,
		})
	}
}

// GET /account/{id}
// func (h *AccountsHandler) GetById() http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {

// 		idStr := r.PathValue("id")
// 		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Mendapatkan id account = %s", idStr))

// 		account, err := h.svc.FetchAccountById(idStr)
// 		if err != nil {
// 			helper.PrintLog("account", helper.LogPositionHandler, err.Error())
// 			dto.WriteError(w, http.StatusInternalServerError, err.Error())
// 			return
// 		}

// 		helper.PrintLog("account", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil data akun dengan id = %s", idStr))
// 		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data akun dengan id = %s", idStr), map[string]any{
// 			"account": account,
// 		})
// 	}
// }

// POST /bank
func (h *BanksHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var payload models.Bank
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			dto.WriteError(w, http.StatusBadRequest, "Format JSON tidak valid!")
			return
		}

		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))

		newBank, err := h.svc.CreateNewBank(payload)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusCreated, "Berhasil membuat data bank baru", map[string]any{
			"bank": newBank,
		})
	}
}

// PATCH /bank/{id}
func (h *BanksHandler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		bankId := r.PathValue("id")
		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Mendapatkan kode bank = %s", bankId))

		var payload models.Bank
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			dto.WriteError(w, http.StatusBadRequest, "Format JSON tidak valid!")
			return
		}

		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))

		bankIdParse, err := uuid.Parse(bankId)
		if err != nil {
			// Jika gagal di-parse, kembalikan error validasi
			dto.WriteError(w, http.StatusBadRequest, "format ID tidak valid atau Data tidak ditemukan")
			return
		}

		payload.ID = bankIdParse

		returnId, err := h.svc.PatchBank(payload)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusPartialContent, "Berhasil mengupdate data bank", map[string]any{
			"id": returnId,
		})
	}
}

// DELETE /bank/{id}
func (h *BanksHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		bankId := r.PathValue("id")
		helper.PrintLog("bank", helper.LogPositionHandler, fmt.Sprintf("Mendapatkan id bank = %s", bankId))

		err := h.svc.DeleteBank(bankId)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil menghapus bank : %s", bankId), map[string]any{})
	}
}
