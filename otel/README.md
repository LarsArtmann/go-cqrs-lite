# otel — OpenTelemetry Helpers

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/otel/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/otel/v3)

Shared OTel instrumentation utilities. All instrumentation is opt-in — no-op when no provider is configured.

```bash
go get github.com/larsartmann/go-cqrs-lite/otel/v3
```

## Related Modules

- [**middleware/v2**](../middleware/README.md) — Tracing and metrics middleware import OTel helpers from here
- [**storage/v2**](../storage/README.md) — SQL stores record spans via `otel/` re-exports
- [**turso/v2**](../turso/README.md) — Index analysis emits spans via `otel/` re-exports

> **Rule:** Import OTel via `otel/v2`, NOT `go.opentelemetry.io` directly. This keeps the SDK indirect in go.mod files.
