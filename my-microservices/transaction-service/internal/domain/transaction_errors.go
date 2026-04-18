package domain

import "errors"

var (
	ErrDatabaseIssue  = errors.New("Gagal mengambil data dari database")
	ErrDatabaseFailed = errors.New("Gagal menambahkan/mengedit data")
	ErrDeleteFailed   = errors.New("Gagal menghapus data")
	ErrDatabaseTrx    = errors.New("Gagal memulai transaksi database")

	ErrInvalidUuid          = errors.New("Format tidak valid")
	ErrInvalidTranserAmount = errors.New("Nominal transfer harus lebih dari 0")
	ErrInvalidJsonFormat    = errors.New("Format JSON tidak valid")
	ErrInvalidDate          = errors.New("Format date tidak sesuai YYYY-MM-DD")
	ErrInvalidFutureDate    = errors.New("Tidak dapat mencari riwayat di masa depan")
	ErrInvalidMaximumNote   = errors.New("Catatan transfer maksimal 255 karakter")
	ErrInvalidTrxAccount    = errors.New("Rekening pengirim/penerima tidak valid")
	ErrInvalidField         = errors.New("Terdapat field yang kosong")

	ErrIdNotFound = errors.New("data tidak ditemukan")

	ErrLogicSelfTranser = errors.New("Tidak dapat melakukan transfer ke rekening sendiri")
	ErrLogicBalanceTrx  = errors.New("Saldo rekening tidak mencukupi")
	ErrLogicMutationTrx = errors.New("Gagal melakukan mutasi transaksi")
	ErrLogicCommitTrx   = errors.New("Gagal melakukan commit transaksi")

	ErrRedisInvalidate = errors.New("Gagal menghapus cache redis")

	ErrUnauthorized      = errors.New("Header otorisasi tidak ada atau tidak valid")
	ErrUnauthorizedToken = errors.New("Token tidak valid atau sudah kadaluarsa")
)
