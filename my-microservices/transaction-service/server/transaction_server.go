package server

import (
	"fmt"
	"log"
	"my-microservices/transaction-service/internal/handler"
	"my-microservices/transaction-service/internal/kafka"
	"my-microservices/transaction-service/internal/middleware"
	"net/http"
	"os"

	pbAccount "my-microservices/shared/pb/account"
	pbFraud "my-microservices/shared/pb/fraud"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	mux        *http.ServeMux
	db         *sqlx.DB
	rdb        *redis.Client
	obs        *middleware.ObservabilityMiddleware
	producer   *kafka.Producer
	accountCli pbAccount.AccountGRPCServiceClient
	fraudCli   pbFraud.FraudServiceClient
	stopFn     func()
}

func NewServer(db *sqlx.DB, rdb *redis.Client, producer *kafka.Producer, accountCli pbAccount.AccountGRPCServiceClient, fraudCli pbFraud.FraudServiceClient) *Server {
	s := &Server{
		mux:        http.NewServeMux(),
		db:         db,
		rdb:        rdb,
		obs:        middleware.NewObservabilityMiddleware(),
		producer:   producer,
		accountCli: accountCli,
		fraudCli:   fraudCli,
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {

	// Transaction Domain =====
	trxHandler := handler.NewTransactionsHandler(s.mux, s.db, s.rdb, s.producer, s.accountCli, s.fraudCli)
	trxHandler.MapRoutes(s.obs)

	s.mux.Handle("/metrics", promhttp.Handler())
}

func (s *Server) Run() {
	addr := os.Getenv("APP_ADDR")
	port := os.Getenv("APP_PORT")
	listen := fmt.Sprintf(":%s", port)

	fmt.Printf("Transaction Service berjalan di %s:%s\n", addr, port)
	log.Fatal(http.ListenAndServe(listen, middleware.ErrorHandling(s.mux)))
}
