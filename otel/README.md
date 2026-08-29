# otel — OpenTelemetry Helpers

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/otel/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/otel/v4)

Shared OTel instrumentation utilities. All instrumentation is opt-in — no-op when no provider is configured.

```bash
go get github.com/larsartmann/go-cqrs-lite/otel/v4
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
| `sse.replay`                            | Internal | `cqrs.sse.last_event_id`, `cqrs.event.count`                                            |
| `watermill.replay.from_journal`         | Internal | `cqrs.projection.name`, `cqrs.event.count`                                              |
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
// Development: stdout exporter so you see traces in your terminal
provider, _ := cqrsotel.Setup(
    cqrsotel.WithService("svc", "1.0", ""),
    cqrsotel.WithStdoutExporter(os.Stdout),
)

// Production: OTLP exporter
provider, _ := cqrsotel.Setup(
    cqrsotel.WithService("svc", "1.0", "i-1"),
    cqrsotel.WithSpanExporter(otlpExporter),
    cqrsotel.WithMetricReader(otlpReader),
)

// Tracing-only (no metrics): pass nil meter with WithMetricsDisabled
bundle, _ := middleware.NewOTelBundle(
    cqrsotel.NewTracer("svc"), nil,
    middleware.WithMetricsDisabled(),
)
```

### Combined: OTel Tracing + Prometheus Metrics

Use `otel.Setup()` for tracing and `prometheus.Setup()` for the `/metrics` endpoint:

```go
// 1. OTel tracing (spans via OTLP or stdout)
otelProvider, _ := cqrsotel.Setup(
    cqrsotel.WithService("orders", "1.0.0", "i-1"),
    cqrsotel.WithSpanExporter(otlpExporter),
)
defer otelProvider.Shutdown(ctx)

// 2. Prometheus metrics bridge (serves /metrics)
promProvider, _ := prometheus.Setup(prometheus.WithViews(cqrsotel.NewCQRSViews()...))
defer promProvider.Shutdown(ctx)

// 3. Bundle uses the Prometheus-backed meter for CQRS metrics
bundle, _ := middleware.NewOTelBundle(
    cqrsotel.NewTracer("orders"),
    promProvider.AsMeterProvider().Meter("orders"),
)
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

## Exporter Lifecycle: Shutdown Flushes Your Telemetry

Batch span processors export asynchronously — **telemetry produced without a
`Shutdown` is silently dropped on process exit**. Batched spans sit in memory
until the batch interval elapses; a server that finishes a request and exits
before the next tick loses exactly the spans you were debugging. The
`Provider` returned by `Setup` owns both providers, and its `Shutdown` flushes
pending spans and metrics before releasing resources. Wire it into your
server's shutdown sequence — not a bare `defer` after `os.Exit`, which never
runs defers at all:

```go
provider, err := cqrsotel.Setup(
    cqrsotel.WithStdoutExporter(os.Stdout),
)
if err != nil {
    log.Fatal(err)
}

srv := &http.Server{Addr: ":8080"}
go func() {
    if serveErr := srv.ListenAndServe(); serveErr != http.ErrServerClosed {
        log.Fatal(serveErr)
    }
}()

// On SIGINT/SIGTERM: stop accepting first (new spans stop),
// THEN flush what already exists.
stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
<-stop

shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

_ = srv.Shutdown(shutdownCtx)          // 1. stop accepting work
if err := provider.Shutdown(shutdownCtx); err != nil { // 2. flush + release
    log.Printf("otel shutdown incomplete: %v", err)
}
```

Order matters: `srv.Shutdown` before `provider.Shutdown` — in-flight requests
still produce spans, and shutting the exporter down first drops them. Give the
flush its own bounded context (5s is a sane ceiling); a shutdown deadline
shared with a hung server would eat the flush window. For tests, prefer
`cqrsotel.WithStdoutExporter(buf)` against a `bytes.Buffer` and assert on the
buffer — no provider-shutdown timing involved.

## Related Modules

- [**middleware**](../middleware/README.md) — `OTelBundle` and tracing/metrics middleware
- [**storage**](../storage/README.md) — SQL stores record spans via `otel/` re-exports
- [**prometheus**](../prometheus/README.md) — OTel→Prometheus metrics bridge
- [**transport/http**](../transport/http/) — SSE event delivery with consumer spans
- [**transport/grpc**](../transport/grpc/) — Remote command/query dispatch with server spans
- [**watermill**](../watermill/) — Broker bridges with producer spans

> **Rule:** Import OTel via `otel/v4`, NOT `go.opentelemetry.io` directly. This keeps the SDK indirect in go.mod files.
