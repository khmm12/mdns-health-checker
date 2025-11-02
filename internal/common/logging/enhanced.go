package logging

import (
	"context"
	"log/slog"

	"github.com/khmm12/mdns-health-checker/internal/common/tracing"
)

var _ slog.Handler = (*EnhancedHandler)(nil)

type EnhancedHandler struct {
	w slog.Handler
}

func NewEnhancedHandler(handler slog.Handler) *EnhancedHandler {
	return &EnhancedHandler{w: handler}
}

func (h *EnhancedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.w.Enabled(ctx, level)
}

func (h *EnhancedHandler) Handle(ctx context.Context, r slog.Record) error {
	if traceID := tracing.GetTraceID(ctx); traceID != "" {
		r.Add(slog.String("trace_id", traceID))
	}

	return h.w.Handle(ctx, r)
}

func (h *EnhancedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.clone(h.w.WithAttrs(attrs))
}

func (h *EnhancedHandler) WithGroup(name string) slog.Handler {
	return h.clone(h.w.WithGroup(name))
}

func (h *EnhancedHandler) clone(handler slog.Handler) *EnhancedHandler {
	return &EnhancedHandler{w: handler}
}
