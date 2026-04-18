package middleware

import (
	"my-microservices/transaction-service/config"
	"my-microservices/transaction-service/helper"
	"my-microservices/transaction-service/internal/domain"
	"my-microservices/transaction-service/internal/dto"
	"my-microservices/transaction-service/observability/metrics"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type IdempotencyMiddleware struct {
	rdb        *redis.Client
	keyManager *helper.RedisKeyManager
}

func NewIdempotencyMiddleware(rdb *redis.Client, keyManager *helper.RedisKeyManager) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{rdb: rdb, keyManager: keyManager}
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
					helper.PrintLog("idempotency", helper.LogPositionHandler, "Request duplikat terdeteksi via Idempotency Key")
					dto.WriteError(w, http.StatusConflict, strconv.Itoa(http.StatusConflict), "Request sedang diproses atau sudah terkirim")
					return
				}
				helper.PrintLog("redis", helper.LogPositionHandler, "Peringatan: Redis Down, melewati pengecekan idempotensi: "+err.Error())
			}
		}

		next(w, r)
	}
}

func ValidateSNAPToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		var secretKey = []byte(os.Getenv("JWT_SECRET_KEY"))
		ctx := r.Context()
		span, tracer := AllCtx(ctx)

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			helper.Log.Error(domain.ErrUnauthorized.Error())
			span.SetStatus(codes.Error, domain.ErrUnauthorized.Error())
			metrics.CacheRequestsTotal.WithLabelValues("token", "unauthorized").Inc()
			dto.WriteError(w, http.StatusUnauthorized, strconv.Itoa(http.StatusUnauthorized), domain.ErrUnauthorized.Error())
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		tokenCtx, tokenSpan := tracer.Start(ctx, "Validasi-Access-Token")
		tokenStart := time.Now()

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("metode signing tidak dikenali: %v", t.Header["alg"])
			}
			return secretKey, nil
		})

		metrics.CacheDuration.WithLabelValues("validasi", "token").Observe(time.Since(tokenStart).Seconds())

		if err != nil || !token.Valid {
			tokenSpan.RecordError(err)
			tokenSpan.SetStatus(codes.Error, domain.ErrUnauthorizedToken.Error())
			tokenSpan.End()
			span.RecordError(err)

			helper.Log.Error(domain.ErrUnauthorizedToken.Error(), zap.Error(err))
			metrics.CacheRequestsTotal.WithLabelValues("token", "tidak_valid").Inc()
			dto.WriteError(w, domain.SnapInvalidToken.HttpCode, strconv.Itoa(domain.SnapInvalidToken.HttpCode), domain.SnapInvalidToken.ResponseMessage)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		tokenSpan.SetStatus(codes.Ok, "token valid")
		tokenSpan.SetAttributes(attribute.String("auth.sub", fmt.Sprintf("%v", claims["sub"])))
		tokenSpan.End()

		metrics.CacheRequestsTotal.WithLabelValues("token", "valid").Inc()

		newCtx := context.WithValue(tokenCtx, "claims", claims)
		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}
