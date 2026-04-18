package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func RunMigrations(db *sql.DB) {
	// 1. Ambil nama schema dari .env
	schema := os.Getenv("DB_SCHEMA")
	if schema == "" {
		schema = "public"
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{
		SchemaName: schema,
	})
	if err != nil {
		log.Fatalf("Gagal membuat driver migrasi postgres: %v", err)
	}

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "file://migrations"
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres", driver)
	if err != nil {
		log.Fatalf("Gagal inisialisasi migrasi: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Gagal menjalankan migrasi: %v", err)
	}

	fmt.Printf("✅ Migrasi untuk schema '%s' berhasil dijalankan!\n", schema)
}
