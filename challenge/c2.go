package challenge

import (
	"context"
	"fmt"
	"log"
	"time"

	"belajar-go/challenge/transactionSystem/config"
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/observability/metrics"
	"belajar-go/challenge/transactionSystem/server"

	"github.com/joho/godotenv"
)

func Challenge2() {

	// Init logger
	helper.InitLogger()
	defer helper.Log.Sync()

	// Inisialisasi Telemetry
	tp, _ := config.InitTracer()
	defer tp.Shutdown(context.Background())

	// Init Metrics
	metrics.Init()

	// Load Env
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("App Error loading .env file: %v", err)
	}

	// DB Connect
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalln("Error saat inisialisasi aplikasi:", err)
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

	fmt.Println("Koneksi Database dan Redis berhasil!")

	svr := server.NewServer(db, rdb)
	svr.Run()
}
