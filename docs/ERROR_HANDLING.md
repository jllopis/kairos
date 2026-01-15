# Error Handling in Kairos

**Status**: 🟡 **In Progress** - Foundation phase  
**Last Updated**: 2026-01-15

---

## Overview

This document explains Kairos's approach to error handling and observability through OpenTelemetry integration.

## Current State Analysis

### ✅ What's Good

1. **OTEL Foundation Solid**
   - `pkg/telemetry/` provides trace and metric exporters
   - Support for stdout, OTLP gRPC, and no-op exporters
   - Proper resource attributes and propagation

2. **Tracing in Critical Paths**
   - Agent loop uses `tracer.Start()` and `span.End()`
   - Tool execution is traced
   - Run IDs ensure request correlation

3. **Structured Logging**
   - Using Go's standard `log/slog` package
   - Potential for JSON output to logs
   - Easy integration with log aggregation

4. **Context Propagation**
   - `context.Context` threaded through all async operations
   - `core.RunID` context value for correlation
   - Task context attached at runtime

### 🟡 What Needs Improvement

1. **No Typed Error Hierarchy**
   ```go
   // Current: Generic errors
   return nil, fmt.Errorf("tool %s failed: %w", name, err)
   
   // Needed: Structured error types
   return nil, KairosError{
       Code: CodeToolFailure,
       Tool: name,
       Err:  err,
   }
   ```
   - No way to distinguish between error types programmatically
   - Monitoring/alerting can't classify errors
   - Recovery decisions must parse error messages

2. **Limited Error Context in Traces**
   ```go
   // Current: Only message
   span.RecordError(err)
   
   // Needed: Rich attributes
   span.RecordError(err)
   span.SetAttributes(
       attribute.String("error.code", "TOOL_FAILURE"),
       attribute.String("tool.name", toolName),
       attribute.Bool("error.recoverable", true),
   )
   ```
   - Errors recorded but not searchable
   - No metadata for dashboards
   - Correlation with metrics is hard

3. **No Retry or Circuit Breaker Patterns**
   - Tool failures cause immediate agent failure
   - No graceful degradation for transient errors
   - No protection against cascading failures

4. **Missing Error Recovery**
   - Tool timeouts not explicitly handled
   - Network errors cause hard failures
   - No fallback mechanisms

5. **No Health Checks**
   - Can't determine if LLM provider is healthy before use
   - Memory system failures cause hard failures
   - Tool availability unknown until invocation

## Impact on Production Readiness

| Issue | Impact | Severity |
|-------|--------|----------|
| No error classification | Can't monitor/alert on error types | 🔴 High |
| Limited trace attributes | Debugging is manual and slow | 🔴 High |
| No retries | Transient failures cause cascades | 🔴 High |
| No timeouts | Hanging tools can freeze agent | 🟡 Medium |
| No health checks | Failures are only discovered at runtime | 🟡 Medium |
| No circuit breakers | Failing tools can degrade performance | 🟡 Medium |

## Proposed Solution

See [ADR 0005: Production-Grade Error Handling Strategy](internal/adr/0005-error-handling-strategy.md) for detailed design.

### Quick Summary

1. **Typed Errors with Rich Context**
   ```go
   type KairosError struct {
       Code       ErrorCode           // e.g., CodeToolFailure
       Message    string              // Human-readable
       Err        error               // Original error
       Context    map[string]any      // Domain context
       Attributes map[string]string   // OTEL attributes
       Recoverable bool               // Can retry?
   }
   ```

2. **Automatic OTEL Integration**
   - Errors automatically recorded to traces
   - Context and attributes attached
   - Searchable and monitorable

3. **Resilience Patterns**
   - Exponential backoff retries
   - Circuit breakers for failing components
   - Timeout boundaries per tool

4. **Health Checks**
   - Component health before use
   - Graceful degradation
   - Observable via metrics

## Implementation Roadmap

### Phase 1: Foundation (Next Sprint)
- [ ] Implement `pkg/errors/` package
- [ ] Create `KairosError` type
- [ ] Add `telemetry.RecordError()` helper
- [ ] Update agent loop with error boundaries
- **Impact**: Errors become searchable in traces

### Phase 2: Resilience (Following Sprint)
- [ ] Implement `pkg/resilience/` package
- [ ] Add retry mechanism
- [ ] Add circuit breaker
- [ ] Configure tool execution resilience
- **Impact**: Transient failures no longer cause cascades

### Phase 3: Observability (2-3 Sprints)
- [ ] Health check interfaces
- [ ] Error rate dashboards
- [ ] Alert rules by error type
- [ ] Documentation for operators
- **Impact**: Operations team can monitor and debug

### Phase 4: Migration (Ongoing)
- [ ] Update all error handling in packages
- [ ] Update A2A server error responses
- [ ] Update CLI error messages
- [ ] Complete by v0.3.0 release

## Example: Before & After

### Tool Execution (Before)

```go
// pkg/mcp/tool_adapter.go
func (ta *ToolAdapter) Execute(ctx context.Context, name string, args map[string]any) (any, error) {
    tool := ta.findTool(name)
    if tool == nil {
        return nil, fmt.Errorf("tool not found: %s", name)
    }
    
    result, err := tool.Call(args)
    if err != nil {
        return nil, fmt.Errorf("tool execution failed: %w", err)
    }
    return result, nil
}

// pkg/agent/agent.go - Usage
result, err := a.mcpClient.Execute(ctx, toolCall.Name, toolCall.Args)
if err != nil {
    slog.Error("tool failed", "tool", toolCall.Name, "error", err)
    return nil, err  // Hard failure
}
```

**Problems**:
- No distinction between "tool not found" and "network timeout"
- No metadata for monitoring
- No recovery attempt
- Error lost in trace

### Tool Execution (After)

```go
// pkg/errors/errors.go
type KairosError struct {
    Code       ErrorCode
    Message    string
    Err        error
    Context    map[string]any
    Recoverable bool
}

// pkg/mcp/tool_adapter.go
func (ta *ToolAdapter) Execute(ctx context.Context, name string, args map[string]any) (any, error) {
    tool := ta.findTool(name)
    if tool == nil {
        return nil, &KairosError{
            Code:        CodeNotFound,
            Message:     "tool not found",
            Context:     map[string]any{"tool": name},
            Recoverable: false, // Not recoverable
        }
    }
    
    result, err := tool.Call(args)
    if err != nil {
        return nil, &KairosError{
            Code:        CodeToolFailure,
            Message:     "tool execution failed",
            Err:         err,
            Context:     map[string]any{"tool": name, "args": len(args)},
            Recoverable: isTransient(err), // Retry on network errors
        }
    }
    return result, nil
}

// pkg/agent/agent.go - Usage with resilience
func (a *Agent) executeToolWithRetry(ctx context.Context, toolCall llm.ToolCall) (any, error) {
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        if attempt > 0 {
            delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(delay):
            }
        }
        
        result, err := a.executeToolWithTimeout(ctx, toolCall)
        if err == nil {
            return result, nil
        }
        
        if ke, ok := err.(*KairosError); ok && !ke.Recoverable {
            telemetry.RecordError(a.tracer.CurrentSpan(ctx), err)
            return nil, err // Hard failure
        }
        
        lastErr = err
    }
    
    telemetry.RecordError(a.tracer.CurrentSpan(ctx), lastErr)
    return nil, lastErr // All retries exhausted
}

// Automatic trace enrichment
func (t *telemetry) RecordError(span trace.Span, err error) {
    if ke, ok := err.(*KairosError); ok {
        span.RecordError(ke.Err)
        span.SetAttributes(
            attribute.String("error.code", string(ke.Code)),
            attribute.Bool("error.recoverable", ke.Recoverable),
        )
        for k, v := range ke.Context {
            span.SetAttributes(attribute.String("error."+k, fmt.Sprintf("%v", v)))
        }
        slog.Error("KairosError",
            "code", ke.Code,
            "message", ke.Message,
            "recoverable", ke.Recoverable,
            "context", ke.Context,
        )
    }
}
```

**Benefits**:
- ✅ Distinguishable error types for monitoring
- ✅ Context and attributes in traces
- ✅ Automatic retry for transient errors
- ✅ Observable: errors searchable in OTEL UI
- ✅ Metrics: error_code + tool_name breakdown
- ✅ Recovery: agent continues despite transient failures

## Monitoring & Observability

### OTEL Queries

With this approach, operations teams can:

```
# Find all tool failures
attributes["error.code"] = "TOOL_FAILURE"

# Find recoverable errors
attributes["error.recoverable"] = true AND duration > 5s

# Find errors by tool
attributes["error.tool"] = "get_weather" AND status = ERROR

# Tool resilience: retries per tool
SELECT tool, COUNT(*) as attempts, COUNT(DISTINCT span_id) as unique_calls
WHERE attributes["error.code"] = "TOOL_FAILURE" AND attributes["error.recoverable"] = true
```

### Metrics

```go
// Counter: errors by code
otel.Meter("kairos").NewInt64Counter("error.count",
    metric.WithUnit("1"),
    metric.WithDescription("Kairos errors by code"),
).Add(ctx, 1, metric.WithAttributes(
    attribute.String("error.code", string(ke.Code)),
))

// Histogram: tool execution time including retries
otel.Meter("kairos").NewInt64Histogram("tool.execution.ms",
    metric.WithUnit("ms"),
    metric.WithDescription("Tool execution time with retries"),
).Record(ctx, durationMs, metric.WithAttributes(
    attribute.String("tool", toolName),
    attribute.Int("attempts", retryCount),
))
```

## FAQ

### Q: Doesn't OTEL already capture errors?

A: Yes, but without classification. The improvements add:
- Error codes for categorization
- Rich context for debugging
- Recoverable flag for automation
- Automatic metrics aggregation

### Q: Why not use Go's errors.Is/As directly?

A: We're not excluding that! `KairosError` implements `Unwrap()` for full `errors.Is/As` compatibility. We're just adding structure on top.

### Q: What about backward compatibility?

A: Phase 4 migration is careful:
1. Keep generic `error` returns in public APIs
2. Gradually add structured errors
3. Document migration path
4. Deprecate unstructured approaches

### Q: How does this affect A2A protocol?

A: A2A server maps `KairosError` to gRPC status codes:
```go
CodeNotFound       → NOT_FOUND
CodeInvalidInput   → INVALID_ARGUMENT
CodeToolFailure    → UNAVAILABLE (if recoverable) else INTERNAL
CodeTimeout        → DEADLINE_EXCEEDED
CodeUnauthorized   → PERMISSION_DENIED
```

## Related

- [ADR 0005: Production-Grade Error Handling](internal/adr/0005-error-handling-strategy.md)
- [Observability Guide: Dashboards & Monitoring](../OBSERVABILITY.md) - Complete guide to dashboards, alerts, and SLOs
- [OTEL Semantic Conventions](https://opentelemetry.io/docs/specs/otel/trace/semantic_conventions/)
- [Go Error Handling](https://go.dev/blog/error-handling-and-go)

---

**Next**: 
1. Review ADR 0005 for detailed technical design
2. See [Observability Guide](../OBSERVABILITY.md) for setting up dashboards and alerts
