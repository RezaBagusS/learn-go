package middleware

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	contextKeySpan   contextKey = "span"
	contextKeyTracer contextKey = "tracer"
)

func WithSpan(ctx context.Context, span trace.Span) context.Context {
	return context.WithValue(ctx, contextKeySpan, span)
}

func WithTracer(ctx context.Context, tracer trace.Tracer) context.Context {
	return context.WithValue(ctx, contextKeyTracer, tracer)
}

func SpanFromCtx(ctx context.Context) trace.Span {
	span, ok := ctx.Value(contextKeySpan).(trace.Span)
	if !ok || span == nil {
		return trace.SpanFromContext(ctx)
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
	return SpanFromCtx(ctx), TracerFromCtx(ctx)
}
