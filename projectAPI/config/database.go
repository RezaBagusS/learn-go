package config

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	// Driver postgreSQL wajib di-import di file tempat sqlx.Connect dipanggil
	_ "github.com/lib/pq"
)

func ConnectDB() (*sqlx.DB, error) {
	// DSN (Data Source Name)
	dsn := "host=localhost port=5432 user=postgres password=Persebaya27. dbname=dbIntegration sslmode=disable"

	// Buka koneksi ke database
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		// Gunakan Error Wrapping untuk memberi konteks
		return nil, fmt.Errorf("gagal terhubung ke database: %w", err)
	}

	fmt.Println("Database PostgreSQL berhasil terkoneksi dari package config!")
	return db, nil
}
