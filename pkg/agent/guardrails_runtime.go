// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"log/slog"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/guardrails"
	"github.com/jllopis/kairos/pkg/telemetry"
	"go.opentelemetry.io/otel/trace"
)

func (a *Agent) checkGuardrailsInput(ctx context.Context, log *slog.Logger, runID, traceID, spanID, input string) error {
	if a.guardrails == nil {
		return nil
	}
	result := a.guardrails.CheckInput(ctx, input)
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(telemetry.GuardrailInputAttributes(result.Blocked, result.GuardrailID, result.Confidence)...)
	if !result.Blocked {
		return nil
	}
	if log != nil {
		log.Warn("agent.guardrails.input_blocked",
			slog.String("agent_id", a.id),
			slog.String("run_id", runID),
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
			slog.String("guardrail", result.GuardrailID),
			slog.String("reason", result.Reason),
		)
	}
	a.emitEvent(ctx, core.EventAgentError, map[string]any{
		"run_id":    runID,
		"stage":     "guardrails.input",
		"guardrail": result.GuardrailID,
		"reason":    result.Reason,
	})
	return WrapGuardrailError(result)
}

func (a *Agent) applyGuardrailsOutput(ctx context.Context, log *slog.Logger, runID, traceID, spanID, output string) string {
	if a.guardrails == nil {
		return output
	}
	result := a.guardrails.FilterOutput(ctx, output)
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(telemetry.GuardrailOutputAttributes(result.Modified, len(result.Redactions))...)
	if result.Modified && log != nil {
		log.Info("agent.guardrails.output_filtered",
			slog.String("agent_id", a.id),
			slog.String("run_id", runID),
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
			slog.Int("redactions", len(result.Redactions)),
		)
	}
	return result.Content
}

func (a *Agent) guardrailsStats() guardrails.Stats {
	if a.guardrails == nil {
		return guardrails.Stats{}
	}
	return a.guardrails.Stats()
}
