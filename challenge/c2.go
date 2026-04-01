package challenge

import (
	"context"
	"fmt"
	"log"
	"time"

	"belajar-go/challenge/transactionSystem/config"
	"belajar-go/challenge/transactionSystem/helper"
	"belajar-go/challenge/transactionSystem/server"

	"github.com/joho/godotenv"
)

func Challenge2() {

	helper.InitLogger()
	defer helper.Log.Sync()

	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("App Error loading .env file: %v", err)
	}

	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalln("Error saat inisialisasi aplikasi:", err)
	}

	defer db.Close()

	rdb := config.ConnectRedis()
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Redis tidak merespon")
	}

	fmt.Println("Koneksi Database dan Redis berhasil!")

	svr := server.NewServer(db, rdb)
	svr.Run()
}
