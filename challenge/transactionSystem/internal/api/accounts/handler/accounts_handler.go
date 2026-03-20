package handler

import (
	"belajar-go/challenge/transactionSystem/dto"
	"belajar-go/challenge/transactionSystem/internal/api/accounts/repository"
	"belajar-go/challenge/transactionSystem/internal/api/accounts/service"
	"belajar-go/challenge/transactionSystem/internal/helper"
	"net/http"

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
