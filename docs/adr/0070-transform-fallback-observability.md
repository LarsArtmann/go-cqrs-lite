# ADR-0070: Transform fallback observability via slog.Default

|             |                                                                                               |
| ----------- | --------------------------------------------------------------------------------------------- |
| **Status**  | Accepted                                                                                      |
| **Date**    | 2026-07-27                                                                                    |
| **Context** | transport/http (SSE CBOR→JSON delivery)                                                       |

## Context

`transport/http.CBORToJSONTransform` (ADR-0052) wraps `codec.TranscodeToJSON`
as a ready-made `WithPayloadTransform` adapter. Its core contract is **graceful
degradation**: when the CBOR payload cannot be decoded (corruption, truncated
bytes, a future encoding the generic decoder does not understand), the transform
returns the **raw payload unchanged** so SSE clients always receive data rather
than a gap.

The open question is **how the failure is observed** by operators. Three
options were considered:

1. **`slog.Default()` Warn** — structured log at Warn level with `event_id`,
   `encoding`, and the wrapped error. Zero configuration; uses the stdlib
   handler the process already configures (or the default text handler).
2. **OTel counter** — `otel.Int64Counter("cqrs.sse.transcode.fallback")`
   incremented on each failure. `transport/http` already imports `otel/`, so no
   new dependency. Metricable and alertable, but invisible without a metrics
   backend, and a counter alone loses the per-event context (which event, which
   encoding, which error).
3. **Callback hook** (`WithTransformFallbackHook(func(evt, err) error)`) —
   maximum flexibility, lets the consumer decide log/metric/drop. Adds API
   surface and configuration for a path that should be exceptional, not routine.

## Decision

Keep **option 1: `slog.Default()` Warn**. It is already implemented in
`transport/http/transform.go`.

```go
out, err := codec.TranscodeToJSON(raw, evt.Encoding())
if err != nil {
    slog.Warn("CBORToJSONTransform: transcode failed, sending raw payload",
        "event_id", evt.ID(), "encoding", evt.Encoding(), "error", err)
    return raw
}
```

### Rationale

- **The fallback is an exceptional path, not a metric to optimize.** A correctly
  encoded CBOR stream never hits it; when it fires, the cause is almost always a
  concrete bug (truncated event, codec mismatch) that a human must investigate.
  A log line with the event ID and error is the right artifact for that
  investigation; a bare counter increment is not.
- **Zero configuration wins for a library.** `slog.Default()` works in every
  process — with or without OTel, with or without a metrics backend. An OTel
  counter is silently swallowed when no meter provider is configured (the
  `otel/` helpers are no-op without `Setup()`), so the failure would become
  invisible precisely in lightweight/test deployments.
- **No new API surface.** Consumers needing custom handling already have an
  escape hatch: call `codec.TranscodeToJSON` directly inside their own
  `func(event.Event) []byte` transform (documented in `codec/doc.go` and
  `transport/http/transform.go`). A dedicated hook option would be
  over-engineering for a rare path.

## Consequences

- Operators see transcode failures as structured Warn logs, not as metrics.
- Alerting on fallback **rate** requires log-based alerting (e.g. grep the
  message) rather than a Prometheus query — acceptable because the rate should
  be ~0 in healthy deployments.

### Reconsider if

- Fallback becomes a hot path in real deployments (it should not — it indicates
  corrupted events). At that point, add an OTel counter via the already-imported
  `otel/` module (`meter.Int64Counter("cqrs.sse.transcode.fallback")`) alongside
  the log, not as a replacement. The log retains the per-event context; the
  counter enables rate alerting.
- A consumer needs to **fail hard** on transcode error rather than degrade. That
  is a different transform (`PayloadTransformE` returning `error`) and would be a
  separate, opt-in API — see the "future-proofing" note in the UP1 follow-up
  backlog. Not warranted today: no consumer has requested it, and the SSE
  contract prioritizes delivery over strictness.
