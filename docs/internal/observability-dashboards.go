// SPDX-License-Identifier: Apache-2.0
// Kairos Error Handling & Observability Dashboards
// This file documents dashboard templates for OpenTelemetry OTEL UI or Grafana.
//
// DASHBOARD: Error Rate & Recovery
//   Shows error trends over time with breakdown by error code and component.
//
//   Queries:
//   - kairos.errors.total{error.code} (rate 5m)
//     Metric: Error rate by error code
//     Display: Line chart with legend (TOOL_FAILURE, TIMEOUT, RATE_LIMITED, LLM_ERROR, etc)
//     Alert Threshold: > 10 errors/min for CodeInternal or CodeMemoryError
//
//   - kairos.errors.recovered{error.code} (rate 5m)
//     Metric: Recovery rate by error code
//     Display: Stacked area chart
//     Goal: errors.recovered / errors.total > 80% (recovery rate)
//
//   - kairos.errors.rate{component}
//     Metric: Error rate per component (errors/min)
//     Display: Single stat with gauge
//     Threshold: Warning > 5/min, Critical > 20/min
//
// DASHBOARD: Component Health
//   Shows the health status of Kairos services and dependencies.
//
//   Queries:
//   - kairos.health.status{component}
//     Metric: Current health (0=unhealthy, 1=degraded, 2=healthy)
//     Display: Status grid with color coding
//     Color Map: Red (0), Yellow (1), Green (2)
//
//   - kairos.circuitbreaker.state{component}
//     Metric: Circuit breaker state (0=open, 1=half-open, 2=closed)
//     Display: Status panels per component
//     Meaning:
//       OPEN (0): Circuit is broken, fallback active, requests rejected
//       HALF_OPEN (1): Testing recovery, allowing limited requests
//       CLOSED (2): Normal operation, requests flowing
//
// DASHBOARD: Error Details
//   Deep dive into specific error patterns and recovery strategies.
//
//   Queries:
//   - kairos.errors.total by (error.code, component, recoverable)
//     Breakdown: Error code × component × recoverability
//     Display: Heatmap or table
//     Insight: Which components have non-recoverable errors?
//
//   - kairos.errors.total{error.code="TIMEOUT"} vs kairos.circuitbreaker.state
//     Correlation: Timeouts vs circuit breaker trips
//     Display: Dual axis line chart
//     Insight: Do timeouts trigger circuit breaker opens?
//
// ALERT RULES (Prometheus/AlertManager format):
//
// Alert 1: High Error Rate
//   Name: KairosHighErrorRate
//   Condition: rate(kairos.errors.total[5m]) > 10
//   Duration: 2m
//   Severity: critical
//   Message: "Kairos error rate {{ $value }} errors/sec, threshold 10"
//   Action: Page on-call engineer, check service logs
//
// Alert 2: Low Recovery Rate
//   Name: KairosLowRecoveryRate
//   Condition: rate(kairos.errors.recovered[5m]) / rate(kairos.errors.total[5m]) < 0.8
//   Duration: 5m
//   Severity: warning
//   Message: "Error recovery rate {{ $value }}%, goal 80%"
//   Action: Review retry/fallback configurations
//
// Alert 3: Circuit Breaker Open
//   Name: KairosCircuitBreakerOpen
//   Condition: kairos.circuitbreaker.state{component=~".*"} == 0
//   Duration: 1m
//   Severity: critical
//   Message: "Circuit breaker OPEN on {{ $labels.component }}, using fallback"
//   Action: Investigate component health, check dependencies
//
// Alert 4: Component Degraded
//   Name: KairosComponentDegraded
//   Condition: kairos.health.status{component=~".*"} == 1
//   Duration: 3m
//   Severity: warning
//   Message: "Component {{ $labels.component }} DEGRADED"
//   Action: Monitor for further degradation or recovery
//
// Alert 5: Component Unhealthy
//   Name: KairosComponentUnhealthy
//   Condition: kairos.health.status{component=~".*"} == 0
//   Duration: 1m
//   Severity: critical
//   Message: "Component {{ $labels.component }} UNHEALTHY"
//   Action: Immediate investigation, possible failover needed
//
// Alert 6: Non-Recoverable Errors
//   Name: KairosNonRecoverableErrors
//   Condition: rate(kairos.errors.total{recoverable="false"}[5m]) > 1
//   Duration: 2m
//   Severity: critical
//   Message: "{{ $value }} non-recoverable errors/sec"
//   Action: Check for bugs or configuration issues
//
// OTEL QUERY EXAMPLES for OTEL UI or Grafana:
//
// 1. Error Rate by Code (5-minute)
//    Metric QL: rate(kairos_errors_total[5m]) by (error_code)
//    PromQL: rate(kairos.errors.total{error.code=~".+"}[5m]) group by (error.code)
//
// 2. Recovery Success Percentage
//    PromQL: (rate(kairos.errors.recovered[5m]) / rate(kairos.errors.total[5m])) * 100
//    Display: Single stat, goal >= 80%
//
// 3. Top Components by Error Count
//    PromQL: topk(5, sum(rate(kairos.errors.total[5m])) by (component))
//    Display: Bar chart
//
// 4. Error Rate Trend (24h)
//    PromQL: rate(kairos.errors.total[5m])
//    Range: 24h
//    Display: Area chart
//
// 5. Circuit Breaker State Changes
//    PromQL: rate(changes(kairos.circuitbreaker.state[5m])[1h:5m]) by (component)
//    Display: Line chart, shows how often circuit breakers flip
//
// INTEGRATION PATTERNS:
//
// 1. Auto-Recovery Tracking:
//    - Start span before operation
//    - On failure: RecordErrorMetric(ctx, err, component)
//    - On retry success: RecordRecovery(ctx, errorCode)
//    - Dashboard shows: errors vs recoveries ratio
//
// 2. Health-Based Routing:
//    - Query kairos.health.status{component} for current state
//    - Route traffic away from UNHEALTHY (0) or DEGRADED (1) components
//    - Use fallback strategies when status != HEALTHY (2)
//
// 3. SLO Tracking:
//    - Error rate SLO: errors/min < 5 (99.9% availability at 1M req/min)
//    - Recovery rate SLO: recovered/errors > 80% (resilience goal)
//    - Component health SLO: all components HEALTHY >= 95% of time
//
// 4. Cost Optimization:
//    - Monitor CodeRateLimit errors to adjust capacity
//    - Monitor CodeToolFailure to identify expensive/unreliable tools
//    - Use error metrics to right-size resource allocation
//
package main

// This file is documentation only and is not compiled.
// See docs/ERROR_HANDLING.md and metrics.go for implementation.
