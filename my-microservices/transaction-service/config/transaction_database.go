package config

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func ConnectDB() (*sqlx.DB, error) {
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

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("gagal terhubung ke database transaction-service: %w", err)
	}

	fmt.Printf("Database PostgreSQL berhasil terkoneksi ke schema '%s' (transaction-service)\n", schema)
	return db, nil
}
