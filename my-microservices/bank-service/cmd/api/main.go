package main // Wajib package main

import (
	"context"
	"log"
	"my-microservices/bank-service/config"
	"my-microservices/bank-service/helper"
	"my-microservices/bank-service/observability/metrics"
	"my-microservices/bank-service/server"
	"time"

	"github.com/joho/godotenv"
)

func main() {

	// Init logger
	helper.InitLogger()
	defer helper.Log.Sync()

	// Inisialisasi Telemetry
	tp, _ := config.InitBankTracer()
	defer tp.Shutdown(context.Background())

	// Init Metrics
	metrics.Init()

	// Load Env
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Info: File .env tidak ditemukan, Bank service menggunakan environment OS/Docker")
	}

	log.Println("⏳ Bank service Mencoba menghubungi database PostgreSQL...")

	// Init & Connect DB
	db, err := config.ConnectDB()
	if err != nil {
		// Jika gagal, aplikasi akan mati dan mencetak errornya
		log.Fatalf("❌ TEST GAGAL [Bank service]: %v", err)
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

	// Kafka Producer
	kafkaProducer := config.ConnectKafka(helper.Log)
	defer kafkaProducer.Close()

	// Jika sampai di baris ini, berarti 100% aman!
	log.Println("✅ TEST BERHASIL [Bank service]: Koneksi Database & Redis siap digunakan!")

	svr := server.NewServer(db, rdb, kafkaProducer)
	svr.Run()
}
