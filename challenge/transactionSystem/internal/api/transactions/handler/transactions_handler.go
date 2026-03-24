package handler

import (
	"belajar-go/challenge/transactionSystem/dto"
	"belajar-go/challenge/transactionSystem/internal/api/transactions/repository"
	"belajar-go/challenge/transactionSystem/internal/api/transactions/service"
	"belajar-go/challenge/transactionSystem/internal/helper"
	"belajar-go/challenge/transactionSystem/internal/models"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"
)

type TransactionsHandler struct {
	mux *http.ServeMux
	svc service.TransactionService
}

func NewTransactionsHandler(mux *http.ServeMux, db *sqlx.DB) *TransactionsHandler {
	trxRepo := repository.NewtransactionRepository(db)
	TrxSvc := service.NewTransactionsService(trxRepo)

	return &TransactionsHandler{
		mux: mux,
		svc: TrxSvc,
	}
}

func (a *TransactionsHandler) MapRoutes() {
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/transactions"),
		a.GetAll(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/transaction/{id}"),
		a.GetById(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, "/transaction"),
		a.Create(),
	)
}

// GET /transactions
func (h *TransactionsHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		helper.PrintLog("transaction", helper.LogPositionHandler, "Mengambil seluruh data transaksi ...")

		transactions, err := h.svc.FetchAllTransactions()
		if err != nil {
			helper.PrintLog("transaction", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}

		helper.PrintLog("transaction", helper.LogPositionHandler, "Berhasil mengambil list data transaksi")
		dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil list data transaksi", map[string]any{
			"transactions": transactions,
		})
	}
}

// GET /transaction/{id}
func (h *TransactionsHandler) GetById() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")
		helper.PrintLog("transaction", helper.LogPositionHandler, fmt.Sprintf("Mendapatkan id transaction = %s", idStr))

		transaction, err := h.svc.FetchTransactionById(idStr)
		if err != nil {
			helper.PrintLog("transaction", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}

		helper.PrintLog("transaction", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil data transaksi dengan id = %s", idStr))
		dto.WriteResponse(w, http.StatusOK, fmt.Sprintf("Berhasil mengambil data transaksi dengan id = %s", idStr), map[string]any{
			"transaction": transaction,
		})
	}
}

// POST /transaction
func (h *TransactionsHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var payload models.Transaction
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			dto.WriteError(w, http.StatusBadRequest, "Format JSON tidak valid!")
			return
		}

		helper.PrintLog("transaction", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))

		transactionID, err := h.svc.CreateTrx(payload)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusCreated, "Transfer berhasil dilakukan", map[string]any{
			"id":     transactionID,
			"amount": payload.Amount,
			"note":   payload.Note,
		})
	}
}
