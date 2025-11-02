package tracing

import (
	"context"

	"github.com/google/uuid"
)

func WithTraceID(ctx context.Context) context.Context {
	if _, ok := ctx.Value(traceIDCtxKey).(string); ok {
		return ctx
	}

	traceID := generateTraceID()

	return context.WithValue(ctx, traceIDCtxKey, traceID)
}

func GetTraceID(ctx context.Context) string {
	traceID, ok := ctx.Value(traceIDCtxKey).(string)
	if !ok {
		return ""
	}

	return traceID
}

func generateTraceID() string {
	v, _ := uuid.NewV7()
	return v.String()
}
