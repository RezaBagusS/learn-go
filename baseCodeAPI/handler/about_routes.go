package handler

import (
	"belajar-go/baseCodeAPI/server"
	"net/http"
)

func (a *AboutHandler) MapRoutes() {
	a.mux.HandleFunc(
		server.NewAPIPath(http.MethodGet, "/about"),
		a.Get(),
	)
}
