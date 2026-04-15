package server

import (
	"fmt"
	"log"
	"my-microservices/bank-service/internal/handler"
	"my-microservices/bank-service/internal/kafka"
	"my-microservices/bank-service/internal/middleware"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	mux      *http.ServeMux
	db       *sqlx.DB
	rdb      *redis.Client
	obs      *middleware.ObservabilityMiddleware
	producer *kafka.Producer // tambah
	stopFn   func()          // untuk stop semua consumer saat shutdown
}

func NewServer(db *sqlx.DB, rdb *redis.Client, producer *kafka.Producer) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		db:       db,
		rdb:      rdb,
		obs:      middleware.NewObservabilityMiddleware(),
		producer: producer,
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {

	// Bank Domain =====
	bnkHandler := handler.NewBanksHandler(s.mux, s.db, s.rdb)
	bnkHandler.MapRoutes(s.obs)

	s.mux.Handle("/metrics", promhttp.Handler())
}

func (s *Server) Run() {
	addr := os.Getenv("APP_ADDR")
	port := os.Getenv("APP_PORT")
	listen := fmt.Sprintf(":%s", port)

	fmt.Printf("Server berjalan di %s:%s\n", addr, port)
	log.Fatal(http.ListenAndServe(listen, middleware.ErrorHandling(s.mux)))
}
