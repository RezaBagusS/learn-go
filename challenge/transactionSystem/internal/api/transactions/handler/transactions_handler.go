package handler

import (
	"belajar-go/challenge/transactionSystem/dto"
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/internal/api/transactions/repository"
	"belajar-go/challenge/transactionSystem/internal/api/transactions/service"
	"belajar-go/challenge/transactionSystem/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type TransactionsHandler struct {
	mux *http.ServeMux
	svc service.TransactionService
	rdb *redis.Client
}

func NewTransactionsHandler(mux *http.ServeMux, db *sqlx.DB, rdb *redis.Client) *TransactionsHandler {
	trxRepo := repository.NewtransactionRepository(db)
	TrxSvc := service.NewTransactionsService(trxRepo)

	return &TransactionsHandler{
		mux: mux,
		svc: TrxSvc,
		rdb: rdb,
	}
}

func (a *TransactionsHandler) MapRoutes() {
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/transactions"),
		a.GetAll(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/transactions/summary"),
		a.GetSummary(),
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
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		helper.PrintLog("transaction", helper.LogPositionHandler, "Berhasil mengambil list data transaksi")
		dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil list data transaksi", map[string]any{
			"transactions": transactions,
		})
	}
}

// GET /transactions/summary
func (h *TransactionsHandler) GetSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		dateStr := r.URL.Query().Get("date")

		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}

		// YYYY-MM-DD
		timeParse, errDate := time.Parse("2006-01-02", dateStr)

		if errDate != nil {
			helper.PrintLog("transaction", helper.LogPositionHandler, models.ErrInvalidDate.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidDate), models.ErrInvalidDate.Error())
			return
		}

		transactions, err := h.svc.FetchSummaryToday(timeParse)
		if err != nil {
			helper.PrintLog("transaction", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
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

		_, err := uuid.Parse(idStr)
		if err != nil {
			helper.PrintLog("account", helper.LogPositionHandler, models.ErrInvalidUuid.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidUuid), models.ErrInvalidUuid.Error())
			return
		}

		transaction, err := h.svc.FetchTransactionById(idStr)
		if err != nil {
			helper.PrintLog("transaction", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
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
			helper.PrintLog("transaction", helper.LogPositionHandler, models.ErrInvalidJsonFormat.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidJsonFormat), models.ErrInvalidJsonFormat.Error())
			return
		}

		if payload.FromAccountID == "" || payload.ToAccountID == "" || payload.Amount == 0 {
			helper.PrintLog("transaction", helper.LogPositionHandler, models.ErrInvalidField.Error())
			dto.WriteError(w, models.StatusCodeHandler(models.ErrInvalidField), models.ErrInvalidField.Error())
			return
		}

		helper.PrintLog("transaction", helper.LogPositionHandler, fmt.Sprintf("Berhasil mengambil payload : %+v", payload))

		transactionID, err := h.svc.CreateTrx(payload)
		if err != nil {
			helper.PrintLog("transaction", helper.LogPositionHandler, err.Error())
			dto.WriteError(w, models.StatusCodeHandler(err), err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusCreated, "Transfer berhasil dilakukan", map[string]any{
			"id":     transactionID,
			"amount": payload.Amount,
			"note":   payload.Note,
		})
	}
}
