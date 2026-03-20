package challenge

import (
	"log"

	"belajar-go/challenge/transactionSystem/config"
	"belajar-go/challenge/transactionSystem/server"

	"github.com/joho/godotenv"
)

func Challenge2() {

	// Muat access environment
	if err := godotenv.Load("challenge/transactionSystem/.env"); err != nil {
		log.Fatalf("App Error loading .env file: %v", err)
	}

	// Inisialisasi Koneksi Database
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalln("Error saat inisialisasi aplikasi:", err)
	}

	// Ini memastikan koneksi database baru akan ditutup HANYA KETIKA aplikasi server mati.
	defer db.Close()

	// 3. Jalankan server
	svr := server.NewServer(db)
	svr.Run()
}
