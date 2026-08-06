# prometheus — OTel-to-Prometheus Metrics Bridge

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/prometheus/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/prometheus/v4)

OTel→Prometheus bridge for go-cqrs-lite metrics.

```bash
go get github.com/larsartmann/go-cqrs-lite/prometheus/v4
```

## Overview

This module wraps the OpenTelemetry Prometheus exporter into a convenient API.
It creates a `MeterProvider` backed by a Prometheus registry, so all OTel
instruments (including `middleware.CommandOTelMetricsWithCounter` and
`middleware.NewOTelMetricsRecorder`) are automatically exposed as Prometheus
metrics via the standard `/metrics` endpoint.

## Quick Start

```go
import (
    "log"
    "net/http"

    "go.opentelemetry.io/otel"
    "github.com/larsartmann/go-cqrs-lite/prometheus/v4"
)

func main() {
    provider, err := prometheus.Setup()
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Shutdown(context.Background())

    otel.SetMeterProvider(provider.AsMeterProvider())

    mux := http.NewServeMux()
    mux.Handle("/metrics", provider.Handler())
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## With Custom Registry

```go
reg := prometheus.NewRegistry()
provider, err := prometheus.Setup(prometheus.WithRegistry(reg))
```

## How It Works

1. Creates an OTel Prometheus exporter registered with a Prometheus registry
2. Creates an OTel `MeterProvider` backed by the exporter
3. Returns a `Provider` that wraps both the meter provider and an HTTP handler

When Prometheus scrapes `/metrics`, the handler calls `Gather()` on the
registry, which triggers collection from the OTel SDK — all instruments
registered on the meter provider are collected and exposed.

## API

| Symbol                         | Description                                                        |
| ------------------------------ | ------------------------------------------------------------------ |
| `Setup(opts...)`               | Creates a `Provider` with a Prometheus-backed `MeterProvider`.     |
| `WithRegistry(r)`             | Use a custom Prometheus registry (default: global default).        |
| `WithViews(views...)`          | Apply OTel views (e.g., `cqrsotel.NewCQRSViews()`).                |
| `Provider.Handler()`           | Returns the `http.Handler` for the `/metrics` endpoint.            |
| `Provider.AsMeterProvider()`   | Returns the underlying OTel `MeterProvider` for `otel.Set*`.       |
| `Provider.Shutdown(ctx)`       | Flushes and shuts down the provider.                               |

## Related Modules

- [**otel**](../otel/README.md) — `Setup()` for tracing; `NewCQRSViews()` for histogram boundaries
- [**middleware**](../middleware/README.md) — `CommandOTelMetricsWithCounter`, `NewOTelMetricsRecorder` feed this bridge
- [**decider**](../decider/README.md) — Repository metrics feed this bridge via OTel
