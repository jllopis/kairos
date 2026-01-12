package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ApprovalExpirer is implemented by services that can expire pending approvals.
type ApprovalExpirer interface {
	ExpireApprovals(ctx context.Context) (int, error)
}

// AddApprovalExpirer registers an expirer to be swept on the configured interval.
func (r *LocalRuntime) AddApprovalExpirer(expirer ApprovalExpirer) {
	if expirer == nil {
		return
	}
	r.approvalExpirers = append(r.approvalExpirers, expirer)
}

// SetApprovalSweepInterval defines how often to sweep for expired approvals.
// Set to 0 to disable.
func (r *LocalRuntime) SetApprovalSweepInterval(interval time.Duration) {
	r.approvalSweepInterval = interval
}

// SetApprovalSweepTimeout defines a per-sweep timeout.
func (r *LocalRuntime) SetApprovalSweepTimeout(timeout time.Duration) {
	r.approvalSweepTimeout = timeout
}

func (r *LocalRuntime) startApprovalSweeper() {
	if r.approvalSweepInterval <= 0 || len(r.approvalExpirers) == 0 {
		log := slog.Default()
		log.Info("runtime.approval.sweeper.disabled",
			slog.Duration("interval", r.approvalSweepInterval),
			slog.Int("expirers", len(r.approvalExpirers)),
		)
		return
	}
	if r.approvalSweepCancel != nil {
		r.stopApprovalSweeper()
	}
	initApprovalMetrics()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	r.approvalSweepCancel = cancel
	r.approvalSweepDone = done
	go func() {
		defer close(done)
		ticker := time.NewTicker(r.approvalSweepInterval)
		defer ticker.Stop()
		log := slog.Default()
		log.Info("runtime.approval.sweeper.start",
			slog.Duration("interval", r.approvalSweepInterval),
			slog.Int("expirers", len(r.approvalExpirers)),
		)
		for {
			select {
			case <-ctx.Done():
				log.Info("runtime.approval.sweeper.stop")
				return
			case <-ticker.C:
				sweepStart := time.Now()
				sweepCtx := ctx
				var cancel context.CancelFunc
				if r.approvalSweepTimeout > 0 {
					sweepCtx, cancel = context.WithTimeout(ctx, r.approvalSweepTimeout)
				}
				sweepCtx, sweepSpan := otel.Tracer("kairos/runtime").Start(sweepCtx, "runtime.approval.sweep",
					trace.WithAttributes(
						attribute.Int("expirers", len(r.approvalExpirers)),
						attribute.String("timeout", r.approvalSweepTimeout.String()),
					),
				)
				traceID, spanID := traceIDs(sweepSpan)
				for _, expirer := range r.approvalExpirers {
					expirerType := expirerName(expirer)
					expirerCtx, expirerSpan := otel.Tracer("kairos/runtime").Start(sweepCtx, "runtime.approval.expire",
						trace.WithAttributes(
							attribute.String("expirer", expirerType),
						),
					)
					expirerTraceID, expirerSpanID := traceIDs(expirerSpan)
					start := time.Now()
					expired, err := expirer.ExpireApprovals(expirerCtx)
					durationMs := float64(time.Since(start).Seconds() * 1000)
					sweepCounter.Add(ctx, 1, metric.WithAttributes(
						attribute.String("expirer", expirerType),
					))
					sweepLatencyMs.Record(ctx, durationMs, metric.WithAttributes(
						attribute.String("expirer", expirerType),
					))
					if err != nil {
						sweepErrorCounter.Add(ctx, 1, metric.WithAttributes(
							attribute.String("expirer", expirerType),
						))
						expirerSpan.RecordError(err)
						log.Warn("runtime.approval.expire.error",
							slog.String("expirer", expirerType),
							slog.Float64("duration_ms", durationMs),
							slog.String("trace_id", expirerTraceID),
							slog.String("span_id", expirerSpanID),
							slog.String("error", err.Error()),
						)
						expirerSpan.End()
						continue
					}
					if expired > 0 {
						expiredCounter.Add(ctx, int64(expired), metric.WithAttributes(
							attribute.String("expirer", expirerType),
						))
					}
					expirerSpan.SetAttributes(
						attribute.Int("expired", expired),
						attribute.Float64("duration_ms", durationMs),
					)
					log.Info("runtime.approval.expire",
						slog.String("expirer", expirerType),
						slog.Int("expired", expired),
						slog.Float64("duration_ms", durationMs),
						slog.String("trace_id", expirerTraceID),
						slog.String("span_id", expirerSpanID),
					)
					expirerSpan.End()
				}
				log.Info("runtime.approval.sweep.complete",
					slog.Int("expirers", len(r.approvalExpirers)),
					slog.Duration("timeout", r.approvalSweepTimeout),
					slog.String("trace_id", traceID),
					slog.String("span_id", spanID),
				)
				sweepTotalLatencyMs.Record(ctx, float64(time.Since(sweepStart).Seconds()*1000), metric.WithAttributes(
					attribute.Int("expirers", len(r.approvalExpirers)),
				))
				if cancel != nil {
					cancel()
				}
				sweepSpan.End()
			}
		}
	}()
}

func (r *LocalRuntime) stopApprovalSweeper() {
	if r.approvalSweepCancel == nil {
		return
	}
	r.approvalSweepCancel()
	if r.approvalSweepDone != nil {
		<-r.approvalSweepDone
	}
	r.approvalSweepCancel = nil
	r.approvalSweepDone = nil
}

var (
	approvalMetricsOnce sync.Once
	sweepCounter        metric.Int64Counter
	sweepErrorCounter   metric.Int64Counter
	expiredCounter      metric.Int64Counter
	sweepLatencyMs      metric.Float64Histogram
	sweepTotalLatencyMs metric.Float64Histogram
)

func initApprovalMetrics() {
	approvalMetricsOnce.Do(func() {
		meter := otel.Meter("kairos/runtime")
		sweepCounter, _ = meter.Int64Counter("kairos.runtime.approval.sweep.count")
		sweepErrorCounter, _ = meter.Int64Counter("kairos.runtime.approval.sweep.error.count")
		expiredCounter, _ = meter.Int64Counter("kairos.runtime.approval.expired.count")
		sweepLatencyMs, _ = meter.Float64Histogram("kairos.runtime.approval.sweep.latency_ms")
		sweepTotalLatencyMs, _ = meter.Float64Histogram("kairos.runtime.approval.sweep.total_latency_ms")
	})
}

func expirerName(expirer ApprovalExpirer) string {
	if expirer == nil {
		return "unknown"
	}
	return fmt.Sprintf("%T", expirer)
}
