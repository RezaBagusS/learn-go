package middleware

import (
	"my-microservices/account-service/helper"
	"my-microservices/account-service/internal/domain"
	"my-microservices/account-service/internal/dto"
	"my-microservices/account-service/observability/metrics"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

func ValidateSNAPToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		var secretKey = []byte(os.Getenv("JWT_SECRET_KEY"))
		ctx := r.Context()
		span, tracer := AllCtx(ctx)

		// --- Validasi header ---
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			helper.Log.Error(domain.ErrUnauthorized.Error(),
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
				zap.String("remote_addr", r.RemoteAddr),
			)
			span.SetStatus(codes.Error, domain.ErrUnauthorized.Error())
			span.SetAttributes(attribute.String("auth.error", "bearer_tidak_ada"))
			metrics.CacheRequestsTotal.WithLabelValues("token", "unauthorized").Inc()
			dto.WriteError(
				w,
				domain.StatusCodeHandler(domain.ErrUnauthorized),
				strconv.Itoa(http.StatusUnauthorized),
				domain.ErrUnauthorized.Error(),
			)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		helper.Log.Info("Memeriksa access token",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
			zap.String("remote_addr", r.RemoteAddr),
		)

		// --- Trace & metrics: validasi token ---
		tokenCtx, tokenSpan := tracer.Start(ctx, "Validasi-Access-Token")
		tokenStart := time.Now()

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("metode signing tidak dikenali: %v", t.Header["alg"])
			}
			return secretKey, nil
		})

		metrics.CacheDuration.WithLabelValues("validasi", "token").
			Observe(time.Since(tokenStart).Seconds())

		// --- Token tidak valid ---
		if err != nil || !token.Valid {
			tokenSpan.RecordError(err)
			tokenSpan.SetStatus(codes.Error, domain.ErrUnauthorizedToken.Error())
			tokenSpan.SetAttributes(attribute.String("auth.error", "token_tidak_valid"))
			tokenSpan.End()

			span.RecordError(err)
			span.SetStatus(codes.Error, "validasi token gagal")

			helper.Log.Error(domain.ErrUnauthorizedToken.Error(),
				zap.Error(err),
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
				zap.String("remote_addr", r.RemoteAddr),
			)
			metrics.CacheRequestsTotal.WithLabelValues("token", "tidak_valid").Inc()
			dto.WriteError(
				w,
				domain.SnapInvalidToken.HttpCode,
				strconv.Itoa(domain.SnapInvalidToken.HttpCode),
				domain.SnapInvalidToken.ResponseMessage,
			)
			return
		}

		// --- Token valid ---
		claims := token.Claims.(jwt.MapClaims)

		tokenSpan.SetStatus(codes.Ok, "token valid")
		tokenSpan.SetAttributes(
			attribute.String("auth.sub", fmt.Sprintf("%v", claims["sub"])),
			attribute.String("auth.iss", fmt.Sprintf("%v", claims["iss"])),
			attribute.String("auth.jti", fmt.Sprintf("%v", claims["jti"])),
		)
		tokenSpan.End()

		span.SetAttributes(
			attribute.String("auth.sub", fmt.Sprintf("%v", claims["sub"])),
			attribute.String("auth.status", "valid"),
		)
		metrics.CacheRequestsTotal.WithLabelValues("token", "valid").Inc()

		helper.Log.Info("Token berhasil divalidasi",
			zap.String("sub", fmt.Sprintf("%v", claims["sub"])),
			zap.String("iss", fmt.Sprintf("%v", claims["iss"])),
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
			zap.String("remote_addr", r.RemoteAddr),
		)

		// Inject claims ke context
		newCtx := context.WithValue(tokenCtx, "claims", claims)
		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}
