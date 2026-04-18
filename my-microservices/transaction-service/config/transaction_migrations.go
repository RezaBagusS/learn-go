package config

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		fmt.Println("⚠️ Gagal membuat driver migrasi (transaction-service):", err)
		return
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		fmt.Println("⚠️ Gagal membuat instance migrasi (transaction-service):", err)
		return
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		fmt.Println("⚠️ Migrasi gagal (transaction-service):", err)
		return
	}

	fmt.Println("✅ Migrasi berhasil (transaction-service)")
}
