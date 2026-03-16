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
	"fmt"
	"log"
	"net/http"

	"belajar-go/projectAPI/config" // Import package config yang baru kita buat
	"belajar-go/projectAPI/handler"
	"belajar-go/projectAPI/repository"
	"belajar-go/projectAPI/service"
)

func Day3() {
	// 1. Inisialisasi Koneksi Database
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalln("Error saat inisialisasi aplikasi:", err)
	}

	// Ini memastikan koneksi database baru akan ditutup HANYA KETIKA aplikasi server mati.
	defer db.Close()

	// 2. DEPENDENCY INJECTION (Perakitan)
	taskRepo := repository.NewTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)
	taskHandler := handler.NewTaskHandler(taskService)

	// 3. Routing HTTP
	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			taskHandler.GetAll(w, r)
		case http.MethodPost:
			taskHandler.Create(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	// 4. Jalankan Server
	fmt.Println("Server berjalan di http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
