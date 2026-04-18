package server

import (
	"fmt"
	"log"
	"my-microservices/account-service/helper"
	"my-microservices/account-service/internal/handler"
	"my-microservices/account-service/internal/kafka"
	"my-microservices/account-service/internal/middleware"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"net"

	"my-microservices/account-service/internal/grpcserver"
	"my-microservices/account-service/internal/repository"
	pb "my-microservices/shared/pb/account"
)

type Server struct {
	mux      *http.ServeMux
	db       *sqlx.DB
	rdb      *redis.Client
	obs      *middleware.ObservabilityMiddleware
	producer *kafka.Producer
	stopFn   func()
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

	// Oauth
	oauthHandler := handler.NewTokenHandler(s.mux, s.rdb)
	oauthHandler.MapRoutes(s.obs)

	// Account Domain =====
	accHandler := handler.NewAccountsHandler(s.mux, s.db, s.rdb, s.producer)
	accHandler.MapRoutes(s.obs)

	s.mux.Handle("/metrics", promhttp.Handler())
}

func (s *Server) Run() {
	addr := os.Getenv("APP_ADDR")
	port := os.Getenv("APP_PORT")
	listen := fmt.Sprintf(":%s", port)

	// --- Jalankan gRPC Server ---
	go func() {
		grpcPort := "50051"
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("failed to listen for gRPC: %v", err)
		}

		grpcSvr := grpc.NewServer()
		repo := repository.NewAccountRepository(s.db)
		pb.RegisterAccountGRPCServiceServer(grpcSvr, grpcserver.NewAccountGRPCServer(repo, helper.Log))

		fmt.Printf("gRPC Account Service berjalan di port %s\n", grpcPort)
		if err := grpcSvr.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	fmt.Printf("Account Service berjalan di %s:%s\n", addr, port)
	log.Fatal(http.ListenAndServe(listen, middleware.ErrorHandling(s.mux)))
}
