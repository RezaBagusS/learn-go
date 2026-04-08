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
	driver, _ := postgres.WithInstance(db, &postgres.Config{})

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "file://migrations" // relatif ke working directory
	}

	// "file:///migrations" adalah path ke folder migrations Anda
	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres", driver)

	if err != nil {
		log.Fatal("Gagal inisialisasi migrasi:", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("Gagal menjalankan migrasi:", err)
	}

	fmt.Println("Migrasi berhasil dijalankan!")
}
