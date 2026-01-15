# SPDX-License-Identifier: Apache-2.0

# OTEL Metrics Export Architecture

> How Kairos metrics are collected, exported, and made available to monitoring systems  
> Technical guide for operators and DevOps

---

## Quick Answer: Sí, Las Métricas Están en OTEL

**Las métricas se incluyen automáticamente en OpenTelemetry** y se exportan a través de:

1. **OTLP gRPC** (recomendado) → Datadog, New Relic, Prometheus, etc.
2. **stdout** (default para desarrollo) → Console output
3. **No HTTP/Prometheus endpoint nativo** → Todo va via OTLP

**No necesitas exponer un endpoint separado** - OTEL maneja todo automáticamente.

---

## Architecture: De Código a Dashboard

```
┌─────────────────────────────────────────────────────────────────┐
│ Kairos Agent Loop                                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Error occurs in agent code                                 │
│  2. telemetry.RecordErrorMetric(ctx, err, span)               │
│  3. Metric recorded to OTEL SDK                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────────┐
│ OTEL SDK (In-Process)                                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  - MeterProvider: Manages all metrics                          │
│  - Meter: Records metric values                                │
│  - Instrument: Counter, Gauge, Histogram                       │
│  - Reader: Collects metrics on schedule                        │
│                                                                 │
│  kairos.errors.total → Counter (increments)                   │
│  kairos.errors.recovered → Counter (increments)                │
│  kairos.health.status → Gauge (0, 1, or 2)                    │
│  kairos.circuitbreaker.state → Gauge (0, 1, or 2)             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────────┐
│ OTEL Exporters (Configured in telemetry.go)                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Option A: OTLP gRPC Export (Recommended)                      │
│  ├─ Endpoint: localhost:4317 (or configured)                  │
│  ├─ Protocol: gRPC                                             │
│  ├─ Batch size: Auto-batched                                  │
│  └─ Frequency: Every 60 seconds (default metric reader)       │
│                                                                 │
│  Option B: Stdout Export (Development)                         │
│  ├─ Output: Console                                            │
│  └─ Format: Pretty-printed JSON                               │
│                                                                 │
│  Option C: None (Disabled)                                     │
│  └─ No export, only in-memory                                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────────┐
│ Collector (OpenTelemetry Collector, Datadog Agent, etc.)       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Receives OTLP gRPC:                                           │
│  - Traces (spans with error attributes)                        │
│  - Metrics (counters, gauges)                                  │
│  - Logs (structured logs with trace context)                  │
│                                                                 │
│  Processes:                                                     │
│  - Filtering, aggregation, transformation                      │
│  - Sampling (keep 100% for errors, 10% for normal)            │
│  - Enrichment (add environment, region, etc.)                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────────┐
│ Backend Destinations (Your Choice)                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ✅ Datadog            (Native OTLP support)                   │
│  ✅ New Relic          (Native OTLP support)                   │
│  ✅ Prometheus         (Via OTLP exporter)                     │
│  ✅ Honeycomb          (Native OTLP support)                   │
│  ✅ Grafana Cloud      (Native OTLP support)                   │
│  ✅ Splunk             (Native OTLP support)                   │
│  ✅ AWS X-Ray          (Via OTLP exporter)                     │
│  ✅ Google Cloud Trace (Via OTLP exporter)                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────────┐
│ Dashboards, Alerts, Analysis                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Visualize:                                                     │
│  - Error rates (rate() in PromQL)                              │
│  - Recovery rates (sum(recovered) / sum(total))                │
│  - Health status (gauge)                                       │
│  - Circuit breaker state (transitions)                         │
│                                                                 │
│  Alert:                                                         │
│  - Error spike (>10/min)                                       │
│  - Low recovery (<50%)                                         │
│  - Service down (unhealthy >5min)                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Configuration: How to Export

### Option 1: OTLP gRPC (Recommended for Production)

**Code initialization:**
```go
import "github.com/jllopis/kairos/pkg/telemetry"

// Via config
cfg := telemetry.Config{
    Exporter: "otlp",
    OTLPEndpoint: "localhost:4317",
    OTLPInsecure: true,
    OTLPTimeoutSeconds: 10,
}

shutdown, err := telemetry.InitWithConfig("my-service", "v1.0.0", cfg)
defer shutdown(context.Background())
```

**Via environment variables:**
```bash
# Set these before running Kairos
export KAIROS_TELEMETRY_EXPORTER=otlp
export KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317
export KAIROS_TELEMETRY_OTLP_INSECURE=true
export KAIROS_TELEMETRY_OTLP_TIMEOUT_SECONDS=10

# Start Kairos
./kairos
```

**What happens:**
```
┌─ Kairos Agent ─────────────────┐
│ telemetry.RecordErrorMetric() │
└──────────────┬──────────────────┘
               │ Every 60 seconds
               ↓
┌──────────────────────────────────┐
│ OTLP Batch Exporter              │
│ - Collects all metrics           │
│ - Groups by time interval        │
│ - Sends via gRPC                 │
└──────────────┬───────────────────┘
               │ gRPC connection
               ↓
┌──────────────────────────────────┐
│ OTLP Receiver (Port 4317)        │
│ - OpenTelemetry Collector        │
│ - Datadog Agent                  │
│ - New Relic Agent                │
│ - etc.                           │
└──────────────────────────────────┘
```

---

### Option 2: Stdout (Development/Testing)

**Code initialization:**
```go
// Default: stdout exporter
shutdown, err := telemetry.Init("my-service", "v1.0.0")
defer shutdown(context.Background())
```

**Output example:**
```json
{
  "ResourceMetrics": [
    {
      "Resource": {
        "Attributes": {
          "service.name": "kairos",
          "service.version": "v0.0.1"
        }
      },
      "ScopeMetrics": [
        {
          "Scope": {
            "Name": "github.com/jllopis/kairos/pkg/telemetry",
            "Version": "1.0.0"
          },
          "Metrics": [
            {
              "Name": "kairos.errors.total",
              "Sum": {
                "DataPoints": [
                  {
                    "Attributes": {
                      "error.code": "llm_error",
                      "component": "llm"
                    },
                    "Value": 5
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ]
}
```

**Perfect for:**
- Local development
- Debugging what metrics are produced
- Testing without external dependencies

---

### Option 3: Disabled (Performance Testing)

```go
cfg := telemetry.Config{
    Exporter: "none",
}

shutdown, err := telemetry.InitWithConfig("my-service", "v1.0.0", cfg)
```

**Result:**
- No OTEL exporters initialized
- In-memory no-op providers
- Zero overhead
- Useful for performance benchmarks

---

## Metrics Details: What Gets Exported

### 1. kairos.errors.total (Counter)

**What it is:** Total error count with attributes

**Attributes:**
- `error.code` - The 9 error codes (llm_error, timeout, rate_limit, etc.)
- `component` - Where error occurred (llm, memory, tool, etc.)
- `recoverable` - "true" or "false"

**Example export (OTLP):**
```protobuf
Metric {
  name: "kairos.errors.total"
  type: COUNTER
  data_points: [
    {
      attributes: {
        "error.code": "llm_error",
        "component": "llm",
        "recoverable": "true"
      }
      value: 5,
      timestamp: 2026-01-15T17:50:00Z
    }
  ]
}
```

**Used for:**
- Alert: Error rate spike → `rate(kairos_errors_total[5m]) > 10`
- Dashboard: Show errors over time
- SLO: Track availability (errors < 5/min)

---

### 2. kairos.errors.recovered (Counter)

**What it is:** Successful recoveries (after retry)

**Attributes:** Same as errors.total

**Recovery Rate Calculation:**
```
recovery_rate = sum(rate(kairos_errors_recovered[5m])) 
              / sum(rate(kairos_errors_total[5m]))

Target: > 80% (80% of errors auto-resolved)
Alert:  < 50% (too many unrecoverable)
```

**Example Prometheus query:**
```promql
# Instant recovery rate
sum(rate(kairos_errors_recovered[5m])) / sum(rate(kairos_errors_total[5m]))

# Recovery rate per component
sum by (component) (rate(kairos_errors_recovered[5m])) 
/ 
sum by (component) (rate(kairos_errors_total[5m]))
```

---

### 3. kairos.errors.rate (Gauge)

**What it is:** Current error rate (errors per minute)

**Example export:**
```
Metric: kairos.errors.rate
Value: 2.5 (errors per minute)
Timestamp: 2026-01-15T17:50:00Z
```

**Used for:**
- Real-time dashboard gauge
- Quick "is system healthy?" check
- Alert threshold: > 10/min = CRITICAL

---

### 4. kairos.health.status (Gauge)

**What it is:** Health state of components

**Values:**
- `0` = UNHEALTHY (service down)
- `1` = DEGRADED (partial functionality)
- `2` = HEALTHY (all good)

**Attributes:**
- `component` - llm, memory, tools, etc.

**Example export:**
```
Metric: kairos.health.status
Attributes: component="llm"
Value: 2 (HEALTHY)

Metric: kairos.health.status
Attributes: component="memory"
Value: 1 (DEGRADED)
```

**Alert example (Prometheus):**
```promql
# Alert if LLM unhealthy for >5 minutes
kairos_health_status{component="llm"} == 0
```

---

### 5. kairos.circuitbreaker.state (Gauge)

**What it is:** Circuit breaker state per component

**Values:**
- `0` = OPEN (failing fast, not attempting)
- `1` = HALF_OPEN (testing if service recovered)
- `2` = CLOSED (normal operation)

**Attributes:**
- `component` - llm, memory, tools, etc.

**Example export:**
```
Metric: kairos.circuitbreaker.state
Attributes: component="llm"
Value: 1 (HALF_OPEN - testing recovery)
```

**Use for:**
- Dashboard: Show circuit breaker state
- Alert: Notify when breaker opens
- Incident correlation: "Breaker opened 5 min before error spike"

---

## Where Metrics Go: OTLP Receiver Endpoint

### Default Endpoint

**Port:** `4317` (standard OTLP gRPC port)  
**Protocol:** gRPC  
**Service:** `opentelemetry.proto.collector.v1.MetricsService`

### Who Listens on 4317?

Any OTLP receiver:

```
+──────────────────────────────────────+
│ OpenTelemetry Collector              │ (stands alone)
│ Datadog Agent                        │ (with OTLP enabled)
│ New Relic Agent                      │ (with OTLP enabled)
│ Splunk Forwarder                     │ (with OTLP enabled)
│ Custom service with OTLP listener    │ (your code)
+──────────────────────────────────────+
```

### Example: Setting Up Collector

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317  # Listen here

processors:
  batch:
    timeout: 10s
    send_batch_size: 1024

exporters:
  logging:
    loglevel: debug
  
  datadog:
    api:
      key: ${DD_API_KEY}
    host_metadata:
      enabled: true

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging, datadog]
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging, datadog]
```

**Start collector:**
```bash
docker run -p 4317:4317 \
  -v $(pwd)/otel-collector-config.yaml:/etc/otel-collector-config.yaml \
  otel/opentelemetry-collector:latest \
  --config=/etc/otel-collector-config.yaml
```

**Kairos exports to it:**
```bash
KAIROS_TELEMETRY_EXPORTER=otlp \
KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317 \
./kairos
```

---

## Integration with Monitoring Backends

### Datadog

**1. Enable OTLP in Datadog Agent:**
```yaml
# /etc/datadog-agent/datadog.yaml
otlp_config:
  receivers:
    grpc:
      enabled: true
      endpoint: 0.0.0.0:4317
```

**2. Point Kairos to agent:**
```bash
export KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317
```

**3. Datadog receives and indexes metrics:**
- Automatically creates dashboards
- Enables alerts
- Correlates with logs and traces

### New Relic

**1. Get ingest key from New Relic**

**2. Deploy collector:**
```docker
docker run -d \
  -e OTLP_ENABLED=true \
  -p 4317:4317 \
  -v $(pwd)/nr-config.yaml:/etc/otel-collector-config.yaml \
  otel/opentelemetry-collector:latest
```

**3. Configure nr-config.yaml:**
```yaml
exporters:
  otlphttp:
    endpoint: https://otlp.nr-data.net:443
    headers:
      api-key: YOUR_LICENSE_KEY
```

**4. Point Kairos:**
```bash
KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317 \
./kairos
```

### Prometheus

**Option A: Via OTLP Collector**
```yaml
exporters:
  prometheus:
    endpoint: 0.0.0.0:8888
```
Then scrape `http://localhost:8888/metrics`

**Option B: Direct (requires custom exporter)**
- OTEL doesn't have direct Prometheus support
- Use collector as proxy
- Or use Prometheus OTLP receiver (new feature)

---

## Batch Behavior: How OTLP Works

Kairos doesn't send metrics one-at-a-time. Instead:

```
Time     Event
────────────────────────────────────────
T+0s     Error #1 recorded → kairos.errors.total++
T+0.5s   Error #2 recorded → kairos.errors.total++
T+1s     Error #3 recorded → kairos.errors.total++
T+2s     Error #4 recorded → kairos.errors.recovered++
T+3s     Health check run → kairos.health.status=2
...
T+60s    ⏰ Metric reader triggers (default: 60s interval)
         All metrics batched together
         OTLP batch exported via gRPC
         Sent to receiver at localhost:4317
```

**Batch settings (configurable in metric reader):**
- **Interval:** 60 seconds (default)
- **Size:** Automatically determined
- **Timeout:** 30 seconds max

**Benefits:**
- ✅ Minimal CPU overhead
- ✅ Single network call per 60s
- ✅ Better compression
- ✅ Receiver efficiency

---

## Debugging: What's Being Exported?

### Method 1: Enable Stdout Exporter

Temporarily switch to stdout to see what's exported:

```go
cfg := telemetry.Config{
    Exporter: "stdout",  // Switch from "otlp"
}
shutdown, err := telemetry.InitWithConfig("kairos", "v1.0.0", cfg)
defer shutdown(context.Background())
```

**Run Kairos:**
```bash
./kairos 2>&1 | grep -A 20 "kairos.errors"
```

**Output shows:**
- All metric names
- All attribute combinations
- Exact values
- Timestamps

### Method 2: tcpdump (See Raw gRPC)

```bash
# Capture gRPC traffic to collector
tcpdump -i lo -A 'tcp port 4317' | grep -i metrics
```

**Limitation:** gRPC is binary, hard to read. Better to use Method 1.

### Method 3: Monitor Collector Logs

```bash
# If using otel-collector
docker logs otel-collector | grep metric
```

**Look for:**
```
Received metrics: 5 DataPoints
Processing batch: 1024 metrics
Exporting to Datadog: 1024 metrics
```

### Method 4: Check Datadog/New Relic

Once exported, instantly visible:

**Datadog:**
```
Metrics → kairos.errors.total
         kairos.errors.recovered
         kairos.health.status
         kairos.circuitbreaker.state
```

**New Relic:**
```
Metrics → kairos.errors.total
        → kairos.errors.recovered
        → kairos.health.status
        → kairos.circuitbreaker.state
```

---

## No Separate Prometheus Endpoint Needed

### Why not expose /metrics?

**Current approach (OTLP):**
- ✅ Works with any backend (Datadog, New Relic, Prometheus, etc.)
- ✅ One code path
- ✅ Single responsibility
- ✅ Push model (Kairos → Collector)

**Alternative (Prometheus HTTP /metrics):**
- ❌ Only works with Prometheus
- ❌ Pull model (Collector → Kairos)
- ❌ Requires extra HTTP server
- ❌ Different code path from traces

**Trade-off:**
- OTLP push: Better for most cases (containers, serverless, clouds)
- Prometheus pull: Better for bare-metal with Prometheus

**If you need /metrics anyway:**

```go
// Add Prometheus exporter alongside OTLP
import "go.opentelemetry.io/otel/exporters/prometheus"

// Configure both exporters
tp, mp := initProviders(resource, cfg)  // OTLP
prom := prometheus.New()                 // Add this
// Wrap mp to include both...
```

But Kairos doesn't require this out of the box.

---

## Summary: The Answer to Your Question

| Aspect | Answer |
|--------|--------|
| **Are metrics in OTEL?** | ✅ Yes, automatically |
| **Do I need to expose endpoint?** | ❌ No, OTEL exports automatically |
| **Where do metrics go?** | OTLP gRPC to configured receiver (4317) |
| **Receiver at 4317?** | Collector, Datadog Agent, New Relic, etc. |
| **How often?** | Every 60 seconds (batched) |
| **Separate /metrics endpoint?** | ❌ Not needed, use OTLP |
| **Can I see them locally?** | ✅ Set exporter to "stdout" |
| **Can I use Prometheus?** | ✅ Via collector proxy |

---

## Quick Start: Export Metrics Right Now

**1. Get OpenTelemetry Collector:**
```bash
docker run -p 4317:4317 \
  otel/opentelemetry-collector:latest
```

**2. Configure Kairos to export:**
```bash
export KAIROS_TELEMETRY_EXPORTER=otlp
export KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317
./kairos
```

**3. Check metrics in stdout mode (debugging):**
```bash
export KAIROS_TELEMETRY_EXPORTER=stdout
./kairos 2>&1 | grep kairos
```

**4. Send to Datadog (production):**
```bash
# Start Datadog Agent with OTLP enabled
# (or use otel-collector with Datadog exporter)

export KAIROS_TELEMETRY_OTLP_ENDPOINT=datadog-agent:4317
./kairos
```

---

## Resources

- **[OBSERVABILITY.md](OBSERVABILITY.md)** - Dashboard setup
- **[INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md)** - How to use in agents
- **[OTEL Docs](https://opentelemetry.io/docs/specs/otel/protocol/)** - OTLP specification
- **[Collector Config Guide](https://opentelemetry.io/docs/collector/configuration/)** - How to configure collector
- **[Datadog OTLP Integration](https://docs.datadoghq.com/opentelemetry/otlp_ingest_in_the_agent/)** - Datadog setup
- **[New Relic OTLP](https://docs.newrelic.com/docs/more-integrations/open-source-integrations/opentelemetry-integrations/opentelemetry-intro/)** - New Relic setup

---

**Version**: 1.0  
**Status**: Ready to use  
**Last Updated**: 2026-01-15

*Métricas en Kairos se exportan automáticamente vía OTEL sin necesidad de endpoints adicionales.*
