package server

import (
	accountHandler "belajar-go/challenge/transactionSystem/internal/api/accounts/handler"
	bankHandler "belajar-go/challenge/transactionSystem/internal/api/banks/handler"
	transactionHandler "belajar-go/challenge/transactionSystem/internal/api/transactions/handler"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	mux *http.ServeMux
	db  *sqlx.DB
	rdb *redis.Client
}

func NewServer(db *sqlx.DB, rdb *redis.Client) *Server {
	s := &Server{
		mux: http.NewServeMux(),
		db:  db,
		rdb: rdb,
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {

	// Account Domain =====
	accHandler := accountHandler.NewAccountsHandler(s.mux, s.db, s.rdb)
	accHandler.MapRoutes()

	// Bank Domain =====
	bnkHandler := bankHandler.NewBanksHandler(s.mux, s.db, s.rdb)
	bnkHandler.MapRoutes()

	// Transaction Domain =====
	trxHandler := transactionHandler.NewTransactionsHandler(s.mux, s.db, s.rdb)
	trxHandler.MapRoutes()
}

func (s *Server) Run() {
	addr := os.Getenv("APP_ADDR")
	port := os.Getenv("APP_PORT")
	listen := fmt.Sprintf(":%s", port)

	fmt.Printf("Server berjalan di %s:%s\n", addr, port)
	log.Fatal(http.ListenAndServe(listen, middleware.ErrorHandling(s.mux)))
}
