package day

// Notes: 'go mod tidy' digunakan utuk menscan seluruh code dan library apa saja yang digunakan
// tapi belum ada pada go.mod (berlaku sebaliknya)

// Terdapat 3 Layer (Handler/Controller, Service, Repository)
// 1. Handler -> Operasi pada sistem (Validasi, Request, Dst)
// 2. Service -> Bisnis Logic
// 3. Repo -> Interaksi dengan db

// Error Wrapping memungkinkan kita menambahkan konteks (keterangan tambahan)
// pada sebuah error tanpa menghilangkan jejak error aslinya. Kita menggunakan fmt.Errorf
// dengan lambang verb %w (wrap).

// import (
//     "github.com/jmoiron/sqlx"
//     _ "github.com/lib/pq" // Driver PostgreSQL
//     "fmt"
// )

import (
	"log"

	"belajar-go/projectAPI/config" // Import package config yang baru kita buat
	"belajar-go/projectAPI/server"

	"github.com/joho/godotenv"
)

func Day3() {

	// Muat access environment
	if err := godotenv.Load(".env"); err != nil {
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
