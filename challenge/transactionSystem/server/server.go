package server

import (
	"belajar-go/challenge/transactionSystem/helper"
	accountHandler "belajar-go/challenge/transactionSystem/internal/api/accounts/handler"
	bankHandler "belajar-go/challenge/transactionSystem/internal/api/banks/handler"
	oauthHandler "belajar-go/challenge/transactionSystem/internal/api/oauth/handler"
	transactionHandler "belajar-go/challenge/transactionSystem/internal/api/transactions/handler"
	"belajar-go/challenge/transactionSystem/internal/kafka"
	kafkahandler "belajar-go/challenge/transactionSystem/internal/kafka/handler"
	"belajar-go/challenge/transactionSystem/internal/middleware"
	"context"
	"fmt"
	"log"
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

func NewServer(db *sqlx.DB, rdb *redis.Client, producer *kafka.Producer, brokers []string) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		db:       db,
		rdb:      rdb,
		obs:      middleware.NewObservabilityMiddleware(),
		producer: producer,
	}

	s.registerRoutes()
	s.startConsumers(brokers)
	return s
}

func (s *Server) registerRoutes() {

	// Oauth
	oauthHandler := oauthHandler.NewTokenHandler(s.mux, s.rdb)
	oauthHandler.MapRoutes(s.obs)

	// Account Domain =====
	accHandler := accountHandler.NewAccountsHandler(s.mux, s.db, s.rdb, s.producer)
	accHandler.MapRoutes(s.obs)

	// Bank Domain =====
	bnkHandler := bankHandler.NewBanksHandler(s.mux, s.db, s.rdb)
	bnkHandler.MapRoutes(s.obs)

	// Transaction Domain =====
	trxHandler := transactionHandler.NewTransactionsHandler(s.mux, s.db, s.rdb, s.producer)
	trxHandler.MapRoutes(s.obs)

	s.mux.Handle("/metrics", promhttp.Handler())
}

func (s *Server) startConsumers(brokers []string) {
	ctx, cancel := context.WithCancel(context.Background())
	s.stopFn = cancel // simpan cancel untuk dipanggil saat shutdown

	logger := helper.Log
	trxKafkaHandler := kafkahandler.NewTransactionKafkaHandler(logger)
	accKafkaHandler := kafkahandler.NewAccountKafkaHandler(logger)

	// Consumer: transaction.created
	TrxCreatedConsumer := kafka.NewConsumer[kafka.TransactionCreatedEvent](
		brokers, kafka.TopicTransactionCreated, "transaction-service", logger,
	)
	TrxCreatedConsumer.Start(ctx, trxKafkaHandler.HandleTransferCreated)

	// Consumer: transaction.failed
	TrxFailedConsumer := kafka.NewConsumer[kafka.TransactionFailedEvent](
		brokers, kafka.TopicTransactionFailed, "transaction-service", logger,
	)
	TrxFailedConsumer.Start(ctx, trxKafkaHandler.HandleTransferFailed)

	// Consumer: account.balance.updated
	balanceConsumer := kafka.NewConsumer[kafka.AccountBalanceUpdatedEvent](
		brokers, kafka.TopicAccountBalanceUpdated, "transaction-service", logger,
	)
	balanceConsumer.Start(ctx, trxKafkaHandler.HandleTransferBalanceUpdated)

	// Consumer: account.created
	AccCreatedConsumer := kafka.NewConsumer[kafka.AccountCreatedEvent](
		brokers, kafka.TopicAccountCreated, "account-service", logger,
	)
	AccCreatedConsumer.Start(ctx, accKafkaHandler.HandleAccountCreated)

	// Consumer: account.failed
	AccFailedConsumer := kafka.NewConsumer[kafka.AccountFailedEvent](
		brokers, kafka.TopicAccountFailed, "account-service", logger,
	)
	AccFailedConsumer.Start(ctx, accKafkaHandler.HandleAccountFailed)

	logger.Info("kafka consumers started")
}

func (s *Server) Run() {
	addr := os.Getenv("APP_ADDR")
	port := os.Getenv("APP_PORT")
	listen := fmt.Sprintf(":%s", port)

	fmt.Printf("Server berjalan di %s:%s\n", addr, port)
	log.Fatal(http.ListenAndServe(listen, middleware.ErrorHandling(s.mux)))
}

func (s *Server) Shutdown() {
	if s.stopFn != nil {
		s.stopFn()
	}
	s.producer.Close()
}
