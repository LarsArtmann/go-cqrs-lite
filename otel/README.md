# otel — OpenTelemetry Helpers

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/otel/v3.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/otel/v3)

Shared OTel instrumentation utilities. All instrumentation is opt-in — no-op when no provider is configured.

```bash
go get github.com/larsartmann/go-cqrs-lite/otel/v3
```

## Quick Start (5 Lines)

```go
// 1. Set up providers (stdout exporter by default; swap for OTLP in production)
provider, _ := cqrsotel.Setup(cqrsotel.WithService("orders", "1.0.0", "i-1"))
defer provider.Shutdown(ctx)

// 2. Create the middleware bundle (tracing + metrics for all message kinds)
bundle, _ := middleware.NewOTelBundle(cqrsotel.NewTracer("orders"), cqrsotel.NewMeter("orders"))

// 3. Wire it into your dispatchers and bus
cmdDisp.Use(bundle.Command()...)
bus.Use(bundle.Event()...)
bus.UsePublish(bundle.Publish()...)
qryDisp.Use(bundle.Query()...)
```

That's it. Every command, event, and query now carries distributed trace spans and operation metrics.

## What You Get

| Span Name                               | Kind     | Attributes                                                                              |
| --------------------------------------- | -------- | --------------------------------------------------------------------------------------- |
| `command.handle`                        | Server   | `cqrs.command.type`, `cqrs.aggregate.id`                                                |
| `event.handle`                          | Consumer | `cqrs.event.type`, `cqrs.aggregate.id`, `cqrs.aggregate.type`, `cqrs.aggregate.version` |
| `event.publish`                         | Producer | `cqrs.event.count`, `cqrs.event.type`, `cqrs.aggregate.id`                              |
| `query.handle`                          | Server   | `cqrs.query.type`                                                                       |
| `grpc.command.dispatch`                 | Server   | `cqrs.command.type`, `cqrs.aggregate.id`                                                |
| `grpc.query.ask`                        | Server   | `cqrs.query.type`                                                                       |
| `watermill.event.publish`               | Producer | `cqrs.event.count`, `cqrs.event.type`, `cqrs.aggregate.id`                              |
| `watermill.command.publish`             | Producer | `cqrs.command.count`, `cqrs.command.type`, `cqrs.aggregate.id`                          |
| `sse.fanout`                            | Consumer | `cqrs.event.type`, `cqrs.aggregate.id`, `cqrs.sse.client_count`                         |
| `event.store.load` / `event.store.save` | Client   | `cqrs.aggregate.type`, `cqrs.aggregate.id`, `cqrs.aggregate.version`                    |
| `decider.load` / `decider.execute`      | Internal | `cqrs.aggregate.type`, `cqrs.aggregate.id`                                              |

### Metrics

| Instrument                | Type      | Attributes                                                                                               |
| ------------------------- | --------- | -------------------------------------------------------------------------------------------------------- |
| `cqrs.operation.duration` | Histogram | `operation`, `cqrs.message.kind`, `cqrs.command.type`/`cqrs.event.type`/`cqrs.query.type`, `cqrs.status` |
| `cqrs.operation.count`    | Counter   | Same as above                                                                                            |

## Provider Setup

`otel.Setup()` creates and registers both providers in one call with functional options:

```go
// Minimal: no-op exporter, W3C propagator, CQRS histogram views
provider, _ := cqrsotel.Setup(cqrsotel.WithService("svc", "1.0", ""))

// Production: OTLP exporter
provider, _ := cqrsotel.Setup(
    cqrsotel.WithService("svc", "1.0", "i-1"),
    cqrsotel.WithSpanExporter(otlpExporter),
    cqrsotel.WithMetricReader(otlpReader),
)

// Prometheus: use the prometheus module instead
provider, _ := prometheus.Setup()
otel.SetMeterProvider(provider.AsMeterProvider())
```

## Distributed Correlation

CQRS has two complementary correlation mechanisms — use both:

| Mechanism                                   | Type                           | Purpose                               |
| ------------------------------------------- | ------------------------------ | ------------------------------------- |
| `event.WithCorrelationID(id.CorrelationID)` | Branded ULID in event metadata | In-service command→event traceability |
| `cqrsotel.WithCorrelationID(ctx, string)`   | OTel baggage                   | Cross-service distributed traces      |

Bridge them with `middleware.OTelCorrelationEnricher`:

```go
ctx = cqrsotel.WithCorrelationID(ctx, traceID.String())
// OTelCorrelationEnricher stamps the baggage correlation ID into event metadata
```

## Related Modules

- [**middleware/v3**](../middleware/README.md) — `OTelBundle` and tracing/metrics middleware
- [**storage/v2**](../storage/README.md) — SQL stores record spans via `otel/` re-exports
- [**prometheus/v3**](../prometheus/README.md) — OTel→Prometheus metrics bridge
- [**transport/http**](../transport/http/) — SSE event delivery with consumer spans
- [**transport/grpc**](../transport/grpc/) — Remote command/query dispatch with server spans
- [**watermill**](../watermill/) — Broker bridges with producer spans

> **Rule:** Import OTel via `otel/v3`, NOT `go.opentelemetry.io` directly. This keeps the SDK indirect in go.mod files.
