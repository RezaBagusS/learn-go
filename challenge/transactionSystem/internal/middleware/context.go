package middleware

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
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

func WithTracer(ctx context.Context, tracer trace.Tracer) context.Context {
	return context.WithValue(ctx, contextKeyTracer, tracer)
}

// Getter — dipanggil di handler
func SpanFromCtx(ctx context.Context) trace.Span {
	span, ok := ctx.Value(contextKeySpan).(trace.Span)
	if !ok || span == nil {
		return trace.SpanFromContext(ctx) // returns no-op span if none exists
	}
	return span
}

func TracerFromCtx(ctx context.Context) trace.Tracer {
	tracer, ok := ctx.Value(contextKeyTracer).(trace.Tracer)
	if !ok {
		return otel.Tracer("fallback")
	}
	return tracer
}

func AllCtx(ctx context.Context) (trace.Span, trace.Tracer) {
	span := SpanFromCtx(ctx)
	tracer := TracerFromCtx(ctx)

	return span, tracer
}
