package helper

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

const (
	referenceNoLength = 22     // panjang standar SNAP BI
	bankCodePadded    = "7777" // 4 digit kode bank/merchant, sesuaikan dengan kode bank kamu
)

// Format: YYYYMMDD + BankCode(4) + Random(10)
func GenerateReferenceNo() string {
	now := time.Now()
	date := now.Format("20060102")

	randomPart := generateRandomNumeric(10) // 10 digit random

	return fmt.Sprintf("%s%s%s", date, bankCodePadded, randomPart)
}

func generateRandomNumeric(n int) string {
	result := make([]byte, n)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(10)) // 0-9
		if err != nil {
			// fallback ke timestamp nano jika crypto/rand gagal
			result[i] = byte('0' + (time.Now().UnixNano()%10+int64(i))%10)
			continue
		}
		result[i] = byte('0' + num.Int64())
	}
	return string(result)
}

// GenerateAuthCode menghasilkan kode otentikasi sementara
// Format: timestamp + random hex, panjang 32 karakter
func GenerateAuthCode() string {
	b := make([]byte, 12)
	rand.Read(b)
	return fmt.Sprintf("%d%s", time.Now().Unix(), hex.EncodeToString(b))[:32]
}

// GenerateAPIKey menghasilkan API key permanen untuk account
// Format: hex 32 byte = 64 karakter
func GenerateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
