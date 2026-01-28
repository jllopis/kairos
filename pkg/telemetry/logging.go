// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// ConfigureSlog sets the global slog logger with trace-aware attributes.
func ConfigureSlog(output io.Writer, level, format string) *slog.Logger {
	handler := newSlogHandler(output, level, format)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func newSlogHandler(output io.Writer, level, format string) slog.Handler {
	opts := &slog.HandlerOptions{
		Level: parseLogLevel(level),
	}
	var base slog.Handler
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		base = slog.NewJSONHandler(output, opts)
	default:
		base = slog.NewTextHandler(output, opts)
	}
	return &traceHandler{next: base}
}

type traceHandler struct {
	next slog.Handler
}

func (h *traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *traceHandler) Handle(ctx context.Context, record slog.Record) error {
	traceID, spanID := spanIDsFromContext(ctx)
	if traceID != "" && !recordHasAttr(record, "trace_id") {
		record.AddAttrs(slog.String("trace_id", traceID))
	}
	if spanID != "" && !recordHasAttr(record, "span_id") {
		record.AddAttrs(slog.String("span_id", spanID))
	}
	return h.next.Handle(ctx, record)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{next: h.next.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{next: h.next.WithGroup(name)}
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func spanIDsFromContext(ctx context.Context) (string, string) {
	if ctx == nil {
		return "", ""
	}
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return "", ""
	}
	sc := span.SpanContext()
	if !sc.IsValid() {
		return "", ""
	}
	return sc.TraceID().String(), sc.SpanID().String()
}

func recordHasAttr(record slog.Record, key string) bool {
	found := false
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == key {
			found = true
			return false
		}
		return true
	})
	return found
}
