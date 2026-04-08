package models

import (
	"errors"
	"net/http"
)

var (
	ErrDatabaseIssue  = errors.New("Gagal mengambil data dari database")
	ErrDatabaseFailed = errors.New("Gagal menambahkan/mengedit data")
	ErrDeleteFailed   = errors.New("Gagal menghapus data")
	ErrDatabaseTrx    = errors.New("Gagal memulai transaksi database")

	ErrInvalidUuid          = errors.New("Format tidak valid")
	ErrInvalidInitBalance   = errors.New("Balance tidak boleh minus")
	ErrInvalidTranserAmount = errors.New("Nominal transfer harus lebih dari 0")
	ErrInvalidJsonFormat    = errors.New("Format JSON tidak valid")
	ErrInvalidTrxType       = errors.New("Tipe transaksi tidak sesuai (all/in/out)")
	ErrInvalidBankCode      = errors.New("Kode bank tidak terdaftar pada sistem")
	ErrInvalidField         = errors.New("Terdapat field yang kosong")
	ErrInvalidDate          = errors.New("Format date tidak sesuai YYYY-MM-DD")
	ErrInvalidFutureDate    = errors.New("Tidak dapat mencari riwayat di masa depan")
	ErrInvalidMaximumNote   = errors.New("Catatan transfer maksimal 255 karakter")
	ErrInvalidTrxAccount    = errors.New("Rekening pengirim/penerima tidak valid")

	ErrIdNotFound = errors.New("data tidak ditemukan")

	ErrDuplicateAccount = errors.New("Nomor rekening sudah terdaftar")
	ErrDuplicateBank    = errors.New("Kode bank sudah terdaftar")

	ErrLogicSelfTranser = errors.New("Tidak dapat melakukan transfer ke rekening sendiri")
	ErrLogicBalanceTrx  = errors.New("Saldo rekening tidak mencukupi")
	ErrLogicMutationTrx = errors.New("Gagal melakukan mutasi transaksi")
	ErrLogicCommitTrx   = errors.New("Gagal melakukan commit transaksi")

	ErrRedisInvalidate = errors.New("Gagal menghapus cache redis")

	ErrUnauthorized      = errors.New("Header otorisasi tidak ada atau tidak valid")
	ErrUnauthorizedToken = errors.New("Token tidak valid atau sudah kadaluarsa")
)

func StatusCodeHandler(err error) int {
	var statusCode int
	switch {
	case errors.Is(err, ErrIdNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, ErrInvalidUuid), errors.Is(err, ErrInvalidInitBalance),
		errors.Is(err, ErrInvalidJsonFormat), errors.Is(err, ErrInvalidTrxType),
		errors.Is(err, ErrInvalidBankCode), errors.Is(err, ErrInvalidField),
		errors.Is(err, ErrInvalidDate), errors.Is(err, ErrInvalidFutureDate),
		errors.Is(err, ErrInvalidMaximumNote), errors.Is(err, ErrInvalidTrxAccount):
		statusCode = http.StatusBadRequest
	case errors.Is(err, ErrDuplicateAccount), errors.Is(err, ErrDuplicateBank):
		statusCode = http.StatusConflict
	case errors.Is(err, ErrLogicSelfTranser), errors.Is(err, ErrLogicBalanceTrx),
		errors.Is(err, ErrLogicMutationTrx), errors.Is(err, ErrLogicCommitTrx):
		statusCode = http.StatusUnprocessableEntity
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrUnauthorizedToken):
		statusCode = http.StatusUnauthorized
	default:
		statusCode = http.StatusInternalServerError
	}
	return statusCode
}
