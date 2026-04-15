package middleware

import (
	"my-microservices/bank-service/config"
	"my-microservices/bank-service/helper"
	"my-microservices/bank-service/internal/dto"
	"net/http"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

type IdempotencyMiddleware struct {
	rdb        *redis.Client
	keyManager *helper.RedisKeyManager
}

func NewIdempotencyMiddleware(rdb *redis.Client, keyManager *helper.RedisKeyManager) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{
		rdb:        rdb,
		keyManager: keyManager,
	}
}

func (m *IdempotencyMiddleware) Check(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idempotencyKey := r.Header.Get("X-EXTERNAL-ID")

		if idempotencyKey != "" {
			ctx := r.Context()
			lockKey := m.keyManager.Generate("idempotency:", strings.ToLower(idempotencyKey))

			_, err := m.rdb.SetArgs(ctx, lockKey, "processing", redis.SetArgs{
				Mode: "NX",
				TTL:  config.TimeLock,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					helper.PrintLog("idempotency", "Middleware", "Request duplikat terdeteksi via Idempotency Key")
					dto.WriteError(
						w,
						http.StatusConflict,
						strconv.Itoa(http.StatusConflict),
						"Request sedang diproses atau sudah terkirim",
					)
					return
				}

				helper.PrintLog("redis", "Middleware", "Peringatan: Redis Down, melewati pengecekan idempotensi: "+err.Error())
			}
		}

		next(w, r)
	}
}
