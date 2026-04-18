package middleware

import (
	"my-microservices/account-service/helper"
	"my-microservices/account-service/observability/metrics"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ObservabilityMiddleware struct {
	tracer trace.Tracer
}

func NewObservabilityMiddleware() *ObservabilityMiddleware {
	return &ObservabilityMiddleware{
		tracer: otel.Tracer("http-middleware"),
	}
}

// Wrap langsung terima next http.Handler
func (m *ObservabilityMiddleware) Wrap(spanName, module string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, span := m.tracer.Start(r.Context(), spanName)
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", r.URL.Path),
			attribute.String("http.target", r.RequestURI),
			attribute.String("module", module),
		)

		traceID := span.SpanContext().TraceID().String()
		logger := helper.Log.With(
			zap.String("trace_id", traceID),
			zap.String("module", module),
		)

		ctx = WithSpan(ctx, span)
		ctx = WithTracer(ctx, m.tracer)

		rw := newResponseWriter(w)
		next.ServeHTTP(rw, r.WithContext(ctx))

		duration := time.Since(start)
		durationSec := duration.Seconds()

		span.SetAttributes(
			attribute.Int("http.status_code", rw.statusCode),
			attribute.Float64("duration_ms", float64(duration.Milliseconds())),
		)

		status := strconv.Itoa(rw.statusCode)

		metrics.HTTPDuration.WithLabelValues(
			spanName,
			r.Method,
			status,
		).Observe(durationSec)

		metrics.HTTPRequestTotal.WithLabelValues(
			spanName,
			r.Method,
			status,
		).Inc()

		if rw.statusCode >= 500 {
			span.SetStatus(codes.Error, http.StatusText(rw.statusCode))
			logger.Error("Request selesai dengan error",
				zap.Int("status_code", rw.statusCode),
				zap.Duration("duration", duration),
			)
		} else if rw.statusCode >= 400 {
			span.SetStatus(codes.Ok, "")
			logger.Warn("Request selesai dengan client error",
				zap.Int("status_code", rw.statusCode),
				zap.Duration("duration", duration),
			)
		} else {
			span.SetStatus(codes.Ok, "")
			logger.Info("Request selesai",
				zap.Int("status_code", rw.statusCode),
				zap.Duration("duration", duration),
			)
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
