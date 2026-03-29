package models

import (
	"errors"
	"net/http"
)

var (
	ErrDatabaseIssue  = errors.New("gagal mengambil data dari db")
	ErrDatabaseFailed = errors.New("gagal menambahkan/mengedit data")
	ErrDeleteFailed   = errors.New("gagal menghapus data")

	ErrInvalidUuid        = errors.New("Format tidak valid")
	ErrInvalidInitBalance = errors.New("Balance tidak boleh minus")
	ErrInvalidJsonFormat  = errors.New("Format JSON tidak valid")
	ErrInvalidTrxType     = errors.New("Tipe transaksi tidak sesuai (all/in/out)")
	ErrInvalidBankCode    = errors.New("Kode bank tidak terdaftar pada sistem")
	ErrInvalidField       = errors.New("Terdapat field yang kosong")

	ErrIdNotFound = errors.New("data tidak ditemukan")

	ErrDuplicateAccount = errors.New("Nomor rekening sudah terdaftar")
	ErrDuplicateBank    = errors.New("Kode bank sudah terdaftar")
)

func StatusCodeHandler(err error) int {
	var statusCode int
	switch {
	case errors.Is(err, ErrIdNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, ErrInvalidUuid), errors.Is(err, ErrInvalidInitBalance), errors.Is(err, ErrInvalidJsonFormat), errors.Is(err, ErrInvalidTrxType), errors.Is(err, ErrInvalidBankCode), errors.Is(err, ErrInvalidField):
		statusCode = http.StatusBadRequest
	case errors.Is(err, ErrDuplicateAccount), errors.Is(err, ErrDuplicateBank):
		statusCode = http.StatusConflict
	default:
		statusCode = http.StatusInternalServerError
	}
	return statusCode
}
