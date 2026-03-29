package repository

import (
	"belajar-go/challenge/transactionSystem/internal/models"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	// "strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type BankRepository interface {
	GetAllBanks() ([]models.Bank, error)
	GetBankByCode(bankCode string) (*models.Bank, error)
	CreateBank(bank models.Bank) (string, error)
	UpdateBank(bank models.Bank) (string, error)
	DeleteBank(bankCode string) error
}

type bankRepository struct {
	db *sqlx.DB
}

func NewBankRepository(db *sqlx.DB) BankRepository {
	return &bankRepository{db: db}
}

// Get All
func (r *bankRepository) GetAllBanks() ([]models.Bank, error) {
	var banks []models.Bank
	query := "SELECT id, bank_code, bank_name, created_at FROM banks ORDER BY created_at desc"

	err := r.db.Select(&banks, query)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data dari db: %w", err) // Error Wrapping
	}

	if banks == nil {
		banks = []models.Bank{}
	}

	return banks, nil
}

// Get Bank by Bank Code
func (r *bankRepository) GetBankByCode(bankCode string) (*models.Bank, error) {
	var bank models.Bank
	query := "SELECT id, bank_code, bank_name, created_at FROM banks WHERE bank_code = $1"
	log.Printf("Kode bank : %s", bankCode)

	err := r.db.Get(&bank, query, bankCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("Data bank tidak ditemukan")
			return nil, fmt.Errorf("Data bank tidak ditemukan")
		}

		log.Printf("Gagal mengambil data dari db : %s", err)
		return nil, fmt.Errorf("gagal mengambil data dari db: %w", err) // Error Wrapping
	}

	return &bank, nil
}

// Post Create New Bank
func (r *bankRepository) CreateBank(bank models.Bank) (string, error) {
	var newBank string
	query := `INSERT INTO banks (bank_code, bank_name) VALUES ($1, $2) RETURNING bank_code`

	err := r.db.QueryRowx(query, bank.BankCode, bank.BankName).Scan(&newBank)
	if err != nil {

		if pqErr, ok := err.(*pq.Error); ok {
			// [23505] Unique Violation
			if pqErr.Code == "23505" {
				return "", fmt.Errorf("Kode Bank %s sudah terdaftar pada sistem", bank.BankCode)
			}
		}

		return "", fmt.Errorf("gagal insert data bank ke db: %w", err)
	}

	return newBank, nil
}

// Method Update
func (r *bankRepository) UpdateBank(bank models.Bank) (string, error) {
	fields := []string{}
	args := []any{}
	idx := 1

	// Cek BankCode
	if bank.BankCode != "" {
		fields = append(fields, fmt.Sprintf("bank_code = $%d", idx))
		args = append(args, bank.BankCode)
		idx++
	}

	// Cek Desc
	if bank.BankName != "" {
		fields = append(fields, fmt.Sprintf("bank_name = $%d", idx))
		args = append(args, bank.BankName)
		idx++
	}

	// Jika tidak ada field yang diupdate
	if len(fields) == 0 {
		return "", fmt.Errorf("tidak ada field yang diupdate")
	}

	// Tambahkan id sebagai kondisi WHERE
	args = append(args, bank.ID)
	query := fmt.Sprintf(
		"UPDATE banks SET %s WHERE id = $%d",
		strings.Join(fields, ", "),
		idx,
	)

	// Query Execution
	fmt.Printf("Query [bank][repo]: %v \n", query)
	fmt.Printf("Args [bank][repo]: %v \n", args)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return "", fmt.Errorf("gagal update bank: %w", err)
	}

	// Cek apakah data dengan ID tersebut ditemukan
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("gagal membaca rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return "", fmt.Errorf("bank dengan id : %s tidak ditemukan", bank.ID)
	}

	return bank.ID.String(), nil
}

// Method Delete
func (r *bankRepository) DeleteBank(bankId string) error {
	query := `DELETE FROM banks WHERE id = $1`

	result, err := r.db.Exec(query, bankId)
	if err != nil {
		return fmt.Errorf("gagal menghapus bank: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal membaca rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("bank dengan id : %s tidak ditemukan", bankId)
	}

	return nil
}
