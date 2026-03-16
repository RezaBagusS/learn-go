package handler

import (
	"belajar-go/baseCodeAPI/dto"
	"encoding/json"
	"net/http"
)

type AboutHandler struct {
	mux *http.ServeMux
}

func NewAboutHandler(mux *http.ServeMux) *AboutHandler {
	return &AboutHandler{
		mux: mux,
	}
}

func (a *AboutHandler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// layer service
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dto.BaseResponse{
			ResponseCode: "200",
			ResponseDesc: "Success",
		})

		// layer repostory

	}
}
