package main

import (
	"context"
	"log"
	"my-microservices/transaction-service/config"
	"my-microservices/transaction-service/helper"
	"my-microservices/transaction-service/observability/metrics"
	"my-microservices/transaction-service/server"
	"my-microservices/transaction-service/internal/kafka"
	"os"
	"strings"
	"time"

	pbAccount "my-microservices/shared/pb/account"
	pbFraud "my-microservices/shared/pb/fraud"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	// Init logger
	helper.InitLogger()
	defer helper.Log.Sync()

	// Inisialisasi Telemetry
	tp, _ := config.InitTransactionTracer()
	defer tp.Shutdown(context.Background())

	// Init Metrics
	metrics.Init()

	// Load Env
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Info: File .env tidak ditemukan, Transaction service menggunakan environment OS/Docker")
	}

	log.Println("⏳ Transaction service mencoba menghubungi database PostgreSQL...")

	// Init & Connect DB
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("❌ GAGAL [Transaction service]: %v", err)
	}
	defer db.Close()

	// Run Migrations
	config.RunMigrations(db.DB)

	// Init Context Time Out
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Redis Connect
	rdb := config.ConnectRedis()
	defer rdb.Close()

	// Redis Ping
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Redis tidak merespon (transaction-service)")
	}

	// Kafka Topics Setup
	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	kafka.EnsureTopics(brokers)

	// Kafka Producer
	kafkaProducer := config.ConnectKafka(helper.Log)
	defer kafkaProducer.Close()

	log.Println("✅ BERHASIL [Transaction service]: Koneksi Database & Redis siap digunakan!")

	log.Println("⏳ Transaction service mencoba menghubungi gRPC Account Service...")

	grpcTarget := os.Getenv("ACCOUNT_GRPC_ADDR")
	if grpcTarget == "" {
		// Fallback jika dijalankan di laptop tanpa docker
		grpcTarget = "localhost:50051"
	}

	accountGrpcConn, err := grpc.NewClient(grpcTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("❌ GAGAL [Transaction service]: Tidak dapat terhubung ke gRPC Account: %v", err)
	}
	defer accountGrpcConn.Close()

	accountCli := pbAccount.NewAccountGRPCServiceClient(accountGrpcConn)
	log.Println("✅ BERHASIL [Transaction service]: gRPC Account Client siap digunakan!")

	log.Println("⏳ Transaction service mencoba menghubungi gRPC Fraud Service...")
	fraudTarget := os.Getenv("FRAUD_GRPC_ADDR")
	if fraudTarget == "" {
		fraudTarget = "localhost:50051"
	}

	fraudGrpcConn, err := grpc.NewClient(fraudTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("❌ GAGAL [Transaction service]: Tidak dapat terhubung ke gRPC Fraud: %v", err)
	}
	defer fraudGrpcConn.Close()

	fraudCli := pbFraud.NewFraudServiceClient(fraudGrpcConn)
	log.Println("✅ BERHASIL [Transaction service]: gRPC Fraud Client siap digunakan!")

	svr := server.NewServer(db, rdb, kafkaProducer, accountCli, fraudCli)
	svr.Run()
}
