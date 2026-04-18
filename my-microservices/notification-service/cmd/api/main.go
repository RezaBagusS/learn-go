package main // Wajib package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"my-microservices/notification-service/config"
	"my-microservices/notification-service/helper"
	"my-microservices/notification-service/observability/metrics"
	"my-microservices/notification-service/server"
)

func main() {
	// Init logger
	helper.InitLogger()
	defer helper.Log.Sync()

	// Inisialisasi Telemetry (Sesuaikan nama fungsinya dengan yang ada di config)
	tp, _ := config.InitNotificationTracer()
	defer tp.Shutdown(context.Background())

	// Init Metrics (Prometheus/Grafana)
	metrics.Init()

	// Load Env
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Info: File .env tidak ditemukan, Notification service menggunakan environment OS/Docker")
	}

	// Load Config Notification (Karena service ini membutuhkan cfg.Callback dsb)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ GAGAL [Notification service]: Gagal memuat config %v", err)
	}

	log.Println("⏳ Notification service Mencoba menghubungi database PostgreSQL...")

	// Init & Connect DB
	db, err := config.ConnectDB() // Atau tetap gunakan config.NewPostgresConnection(cfg.Database) jika belum diubah
	if err != nil {
		log.Fatalf("❌ GAGAL [Notification service]: %v", err)
	}
	defer db.Close()

	// Run Migrations
	config.RunMigrations(db.DB)

	// Init Context Time Out (Digunakan jika ada ping check saat startup)
	_, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Kafka Consumers Setup
	kafkaReaders := config.ConnectKafkaReaders(cfg.Kafka, helper.Log)

	log.Println("✅ BERHASIL [Notification service]: Koneksi Database & Kafka siap digunakan!")

	// Init App Server / Handler runner
	svr := server.NewServer(db.DB, kafkaReaders, cfg)
	svr.Run()
}
