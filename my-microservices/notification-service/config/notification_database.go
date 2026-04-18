package config

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"

	// Driver postgreSQL wajib di-import di file tempat sqlx.Connect dipanggil
	_ "github.com/lib/pq"
)

func ConnectDB() (*sqlx.DB, error) {
	// DSN (Data Source Name)
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	ssl := os.Getenv("DB_SSLMODE")
	schema := os.Getenv("DB_SCHEMA")

	if schema == "" {
		schema = "public"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=%s",
		user, pass, host, port, name, ssl, schema)

	// Buka koneksi ke database
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		// Gunakan Error Wrapping untuk memberi konteks
		return nil, fmt.Errorf("gagal terhubung ke database: %w", err)
	}

	fmt.Printf("Database PostgreSQL berhasil terkoneksi ke schema '%s' dari package config!\n", schema)
	return db, nil
}
