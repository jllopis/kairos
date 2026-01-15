# ADR 0005: Production-Grade Error Handling Strategy

**Status:** Proposed  
**Date:** 2026-01-15  
**Decision Makers:** Kairos Architecture Team

---

## Problem Statement

Kairos currently has **limited production-level error handling**:

1. **No centralized error strategy**: Error types and handling vary across packages
2. **OTEL available but underutilized**: Traces exist but errors aren't systematically recorded with rich context
3. **No error recovery patterns**: Critical failures may cause cascading issues
4. **Insufficient error context**: Stack traces are minimal; diagnosis requires tracing through code
5. **No circuit breaker/retry patterns**: Tool invocations fail hard without graceful degradation
6. **Limited observability**: Errors aren't classified or aggregated for monitoring

## Context

Kairos has excellent telemetry foundations:
- OTEL SDK fully integrated (`pkg/telemetry/`)
- Structured logging with `log/slog`
- Context propagation for traces and spans
- Event emitter infrastructure (`pkg/core/event.go`)

However, error handling doesn't leverage these capabilities systematically.

## Decision

Implement a **production-grade error handling framework** with these components:

### 1. Typed Error Hierarchy

Create domain-specific error types with rich context:

```go
// pkg/errors/errors.go
type ErrorCode string

const (
    CodeInternal      ErrorCode = "INTERNAL_ERROR"
    CodeInvalidInput  ErrorCode = "INVALID_INPUT"
    CodeToolFailure   ErrorCode = "TOOL_FAILURE"
    CodeContextLost   ErrorCode = "CONTEXT_LOST"
    CodeTimeout       ErrorCode = "TIMEOUT"
    CodeRateLimit     ErrorCode = "RATE_LIMITED"
    CodeNotFound      ErrorCode = "NOT_FOUND"
    CodeUnauthorized  ErrorCode = "UNAUTHORIZED"
)

type KairosError struct {
    Code       ErrorCode
    Message    string
    Err        error              // Original error
    Context    map[string]any     // Rich context
    Attributes map[string]string  // OTEL attributes
    Recoverable bool              // Can be retried?
    StatusCode int                // HTTP/gRPC status
}

func (e *KairosError) Error() string {
    return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
}

func (e *KairosError) Unwrap() error {
    return e.Err
}
```

### 2. Error Wrapping with Context

Extend errors with structured context at each layer:

```go
// Helper functions
func New(code ErrorCode, msg string, cause error) *KairosError {
    return &KairosError{
        Code:    code,
        Message: msg,
        Err:     cause,
        Context: make(map[string]any),
    }
}

func WithContext(err error, key string, value any) error {
    if ke, ok := err.(*KairosError); ok {
        ke.Context[key] = value
        return ke
    }
    return err
}

func WithAttributes(err error, attrs map[string]string) error {
    if ke, ok := err.(*KairosError); ok {
        ke.Attributes = attrs
        return ke
    }
    return err
}
```

### 3. Automatic OTEL Integration

Record errors in traces with full context:

```go
// pkg/telemetry/errors.go
func RecordError(span trace.Span, err error) {
    if err == nil {
        return
    }
    
    span.RecordError(err)
    
    // Extract KairosError context
    if ke, ok := err.(*errors.KairosError); ok {
        span.SetAttributes(
            attribute.String("error.code", string(ke.Code)),
            attribute.Bool("error.recoverable", ke.Recoverable),
        )
        
        // Add custom attributes
        for k, v := range ke.Attributes {
            span.SetAttributes(attribute.String("error."+k, v))
        }
        
        // Log rich context
        slog.Error("KairosError recorded",
            "code", ke.Code,
            "message", ke.Message,
            "recoverable", ke.Recoverable,
            "context", ke.Context,
        )
    }
}
```

### 4. Retry and Circuit Breaker Patterns

Implement resilience:

```go
// pkg/resilience/retry.go
type RetryConfig struct {
    MaxAttempts  int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
    IsRecoverable func(error) bool
}

func WithRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
    var lastErr error
    for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
        if attempt > 0 {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
            }
        }
        
        lastErr = fn()
        if lastErr == nil {
            return nil
        }
        
        if !cfg.IsRecoverable(lastErr) {
            return lastErr
        }
    }
    return lastErr
}

// pkg/resilience/circuit_breaker.go
type CircuitBreakerConfig struct {
    FailureThreshold int
    ResetTimeout    time.Duration
    HalfOpenMaxAttempts int
}

type CircuitBreaker struct {
    state    string // "closed", "open", "half-open"
    failures int
    lastFail time.Time
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    switch cb.state {
    case "open":
        if time.Since(cb.lastFail) > cb.ResetTimeout {
            cb.state = "half-open"
            cb.failures = 0
        } else {
            return errors.New(errors.CodeInternal, "circuit breaker open", nil)
        }
    }
    
    err := fn()
    if err != nil {
        cb.failures++
        cb.lastFail = time.Now()
        if cb.failures >= cfg.FailureThreshold {
            cb.state = "open"
        }
    } else {
        cb.state = "closed"
        cb.failures = 0
    }
    return err
}
```

### 5. Tool Execution Error Handling

Enhance tool invocation resilience:

```go
// pkg/mcp/tool_adapter.go - Enhanced
func (ta *ToolAdapter) ExecuteWithResilience(ctx context.Context, name string, args map[string]any) (any, error) {
    ctx, span := ta.tracer.Start(ctx, "ExecuteTool", 
        trace.WithAttributes(attribute.String("tool.name", name)))
    defer span.End()
    
    // Retry config for tool execution
    retryConfig := ta.retryConfigFor(name)
    
    var result any
    err := resilience.WithRetry(ctx, retryConfig, func() error {
        var execErr error
        result, execErr = ta.Execute(ctx, name, args)
        return execErr
    })
    
    if err != nil {
        kairoErr := errors.New(
            errors.CodeToolFailure,
            fmt.Sprintf("tool %s failed", name),
            err,
        ).WithContext("tool", name).WithContext("args_count", len(args))
        
        telemetry.RecordError(span, kairoErr)
        return nil, kairoErr
    }
    
    return result, nil
}
```

### 6. Agent Loop Error Boundaries

Add structured error recovery in agent execution:

```go
// pkg/agent/agent.go - Enhanced Run method
func (a *Agent) Run(ctx context.Context, input any) (any, error) {
    ctx, span := a.tracer.Start(ctx, "Agent.Run")
    defer span.End()
    
    // ... setup code ...
    
    for iteration := 0; iteration < a.maxIterations; iteration++ {
        // Tool invocation with error boundary
        toolResult, toolErr := a.executeToolWithBoundary(ctx, toolCall)
        
        if toolErr != nil {
            if ke, ok := toolErr.(*errors.KairosError); ok && ke.Recoverable {
                // Log but continue
                slog.Warn("Tool failed but recoverable", "tool", toolCall.Name, "error", ke)
                span.AddEvent("tool_failure_recovered",
                    trace.WithAttributes(attribute.String("tool", toolCall.Name)))
                toolResult = fmt.Sprintf("Tool %s unavailable, continuing with cached data", toolCall.Name)
            } else {
                // Hard failure - propagate
                telemetry.RecordError(span, toolErr)
                return nil, toolErr
            }
        }
        
        // ... continue loop ...
    }
}

func (a *Agent) executeToolWithBoundary(ctx context.Context, toolCall llm.ToolCall) (any, error) {
    // Timeout boundary per tool
    timeout := 30 * time.Second
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    resultChan := make(chan any)
    errChan := make(chan error, 1)
    
    go func() {
        result, err := a.executeTool(ctx, toolCall)
        if err != nil {
            errChan <- err
            return
        }
        resultChan <- result
    }()
    
    select {
    case <-ctx.Done():
        return nil, errors.New(
            errors.CodeTimeout,
            fmt.Sprintf("tool %s timeout", toolCall.Name),
            ctx.Err(),
        ).WithContext("timeout", timeout.String())
    case result := <-resultChan:
        return result, nil
    case err := <-errChan:
        return nil, err
    }
}
```

### 7. Health Check and Degraded State

Expose system health through errors:

```go
// pkg/core/health.go
type HealthStatus string

const (
    Healthy    HealthStatus = "HEALTHY"
    Degraded   HealthStatus = "DEGRADED"
    Unavailable HealthStatus = "UNAVAILABLE"
)

type HealthCheck interface {
    Check(ctx context.Context) HealthStatus
}

// In Agent.Run:
if status := a.llm.Check(ctx); status != Healthy {
    return nil, errors.New(
        errors.CodeInternal,
        "LLM provider unhealthy",
        nil,
    ).WithContext("status", string(status)).WithRecoverable(status == Degraded)
}
```

## Consequences

### Positive
- ✅ Rich error context for debugging
- ✅ Automatic OTEL integration (errors visible in traces)
- ✅ Retry/circuit breaker patterns reduce cascading failures
- ✅ Clear error codes enable monitoring and alerting
- ✅ Recoverable vs non-recoverable errors enable intelligent handling
- ✅ Production-ready observability

### Negative
- ❌ Requires migration of existing error handling
- ❌ Adds type safety overhead (compared to simple `error`)
- ❌ New developers need to learn error patterns

## Implementation Phases

### Phase 1: Foundation (Current)
- [ ] Create `pkg/errors/` package
- [ ] Implement `KairosError` type
- [ ] Integrate with OTEL in `pkg/telemetry/`
- [ ] Update agent loop with error boundaries

### Phase 2: Resilience
- [ ] Implement `pkg/resilience/` with retry and circuit breaker
- [ ] Add retry config for tool execution
- [ ] Add timeout boundaries

### Phase 3: Observability
- [ ] Health check interfaces
- [ ] Error classification for monitoring
- [ ] Dashboards for error rates by type

### Phase 4: Documentation
- [ ] Error handling guide for developers
- [ ] Migration guide for existing code
- [ ] Troubleshooting guide

## Open Questions

1. Should we use custom error types or `errors.As()` matching?
2. How to handle cross-service errors (A2A protocol)?
3. Circuit breaker per-tool or global?
4. What constitutes a "recoverable" error?

## References

- [Go Error Handling Best Practices](https://go.dev/blog/error-handling-and-go)
- [OpenTelemetry Error Semantics](https://opentelemetry.io/docs/specs/otel/trace/semantic_conventions/exceptions/)
- [Release It! Design and Deploy Production-Ready Software](https://pragprog.com/titles/mnee2/release-it-second-edition/)

---

**Next Steps**: Implement Phase 1 foundation, starting with `pkg/errors/` package.
