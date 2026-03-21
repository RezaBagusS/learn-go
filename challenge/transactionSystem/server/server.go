package server

import (
	accountHandler "belajar-go/challenge/transactionSystem/internal/api/accounts/handler"
	bankHandler "belajar-go/challenge/transactionSystem/internal/api/banks/handler"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
)

type Server struct {
	mux *http.ServeMux
	db  *sqlx.DB
}

func NewServer(db *sqlx.DB) *Server {
	s := &Server{
		mux: http.NewServeMux(),
		db:  db,
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {

	// Account Domain =====
	accHandler := accountHandler.NewAccountsHandler(s.mux, s.db)
	accHandler.MapRoutes()

	// Bank Domain =====
	bnkHandler := bankHandler.NewBanksHandler(s.mux, s.db)
	bnkHandler.MapRoutes()
}

func (s *Server) Run() {
	addr := os.Getenv("APP_ADDR")
	port := os.Getenv("APP_PORT")
	listen := fmt.Sprintf(":%s", port)

	fmt.Printf("Server berjalan di %s:%s\n", addr, port)
	log.Fatal(http.ListenAndServe(listen, middleware.ErrorHandling(s.mux)))
}
