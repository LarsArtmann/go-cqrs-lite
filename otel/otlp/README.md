# otel/otlp

One-call OTLP/HTTP export for go-cqrs-lite telemetry: builds the standard
OTLP trace and metric exporters and hands them to
[`otel.Setup`](../../README.md) — CQRS histogram views, propagation, resource
attributes, and flush-on-shutdown all included.

```go
import cqrsotlp "github.com/larsartmann/go-cqrs-lite/otel/otlp/v4"

provider, err := cqrsotlp.SetupOTLP(ctx, cqrsotlp.OTLPConfig{
    Endpoint:    "localhost:4318", // host:port, no scheme (collector default)
    Insecure:    true,             // plain HTTP — local or sidecar collectors
    Headers:     map[string]string{"x-scope-token": "..."}, // collector auth
    ServiceName: "my-service",
})
defer func() { _ = provider.Shutdown(ctx) }()
```

Point it at any OTLP/HTTP receiver: an OpenTelemetry Collector, SigNoz,
Jaeger, Grafana Alloy.

## Design notes

- **HTTP transport only** — the module adds no gRPC dependency to your
  graph. gRPC users inject their own exporter via
  `cqrsotel.WithSpanExporter` into `cqrsotel.Setup` directly.
- **Trailing options win** — every `cqrsotel.SetupOption` passed after the
  config overrides the OTLP wiring (`WithSpanProcessor`,
  `WithoutGlobalRegistration`, a custom metric reader, ...).
- **Exemplars are on by default** — the SDK's trace-based exemplar filter
  attaches trace/span IDs to histogram observations made under sampled
  spans; control it with the standard `OTEL_METRICS_EXEMPLAR_FILTER` env var.
