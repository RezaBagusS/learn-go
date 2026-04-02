package middleware

import (
	"belajar-go/challenge/transactionSystem/helper"
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Pakai unexported type — hindari collision dengan package lain
type contextKey string

const (
	contextKeySpan   contextKey = "span"
	contextKeyLogger contextKey = "logger"
	contextKeyTracer contextKey = "tracer"
)

// Setter — dipanggil di middleware
func WithSpan(ctx context.Context, span trace.Span) context.Context {
	return context.WithValue(ctx, contextKeySpan, span)
}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, contextKeyLogger, logger)
}

func WithTracer(ctx context.Context, tracer trace.Tracer) context.Context {
	return context.WithValue(ctx, contextKeyTracer, tracer)
}

// Getter — dipanggil di handler
func SpanFromCtx(ctx context.Context) trace.Span {
	span, _ := ctx.Value(contextKeySpan).(trace.Span)
	return span
}

func LoggerFromCtx(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(contextKeyLogger).(*zap.Logger)
	if !ok || logger == nil {
		return helper.Log // fallback ke global logger
	}
	return logger
}

func TracerFromCtx(ctx context.Context) trace.Tracer {
	tracer, ok := ctx.Value(contextKeyTracer).(trace.Tracer)
	if !ok {
		return otel.Tracer("fallback")
	}
	return tracer
}

func AllCtx(ctx context.Context) (trace.Span, *zap.Logger, trace.Tracer) {
	span := SpanFromCtx(ctx)
	logger := LoggerFromCtx(ctx)
	tracer := TracerFromCtx(ctx)

	return span, logger, tracer
}
