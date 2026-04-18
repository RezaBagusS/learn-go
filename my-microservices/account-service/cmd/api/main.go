package main // Wajib package main

import (
	"context"
	"log"
	"my-microservices/account-service/config"
	"my-microservices/account-service/helper"
	"my-microservices/account-service/internal/kafka"
	"my-microservices/account-service/observability/metrics"
	"my-microservices/account-service/server"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func main() {

	// Init logger
	helper.InitLogger()
	defer helper.Log.Sync()

	// Inisialisasi Telemetry
	tp, _ := config.InitAccountTracer()
	defer tp.Shutdown(context.Background())

	// Init Metrics
	metrics.Init()

	// Load Env
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Info: File .env tidak ditemukan, Account service menggunakan environment OS/Docker")
	}

	log.Println("⏳ Account service Mencoba menghubungi database PostgreSQL...")

	// Init & Connect DB
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("❌ GAGAL [Account service]: %v", err)
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
		log.Fatal("Redis tidak merespon")
	}

	// Kafka Topics Setup
	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	kafka.EnsureTopics(brokers)

	// Kafka Producer
	kafkaProducer := config.ConnectKafka(helper.Log)
	defer kafkaProducer.Close()

	log.Println("✅ BERHASIL [Account service]: Koneksi Database & Redis siap digunakan!")

	svr := server.NewServer(db, rdb, kafkaProducer)
	svr.Run()
}
