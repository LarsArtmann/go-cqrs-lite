# ADR-0089: Flight Recorder Integration

**Date:** 2026-08-01
**Status:** Accepted
**Related:** [Go 1.25 Flight Recorder blog post](https://go.dev/blog/flight-recorder)

## Context

Go 1.25 introduced `runtime/trace.FlightRecorder`, which buffers the last few
seconds of execution trace in memory. When a problem is detected (slow
operation, error, crash), the program can snapshot exactly the problematic
window for offline analysis with `go tool trace`.

This is a natural fit for CQRS/ES systems: every command dispatch, event
handler, and query execution has a clear duration and error signal. The flight
recorder can capture a trace snapshot when an operation crosses a latency
threshold or returns an error — the exact scenarios where root-cause analysis
is most valuable.

The challenge: how to integrate this capability into go-cqrs-lite's
multi-module architecture without adding dependencies to leaf modules, while
providing composable trigger conditions and lifecycle management.

## Decision

### 1. Two-layer design (zero-dep core + CQRS middleware)

Mirror the `retry/` pattern: a zero-dependency `flightrecorder/` module
(stdlib only) wraps `runtime/trace.FlightRecorder`, and the `middleware/`
module adds CQRS-aware triggers.

- Consumers who only need flight recording (CLI tools, batch processors) import
  `flightrecorder/` without pulling CQRS types.
- CQRS consumers use `middleware.CommandFlightRecorder` etc. for dispatch-level
  trigger wiring.

### 2. Once-semantics by default

The first `Snapshot()` call wins; subsequent calls are no-ops until `Reset()`
re-arms. This prevents snapshot races when multiple goroutines detect a problem
simultaneously (e.g., a slow command triggering both the command middleware and
the projection host at the same time).

### 3. Trigger composition via TriggerFunc

```go
type TriggerFunc func(TriggerContext) bool
```

Built-in triggers: `OnLatency(threshold)`, `OnError()`, `OnErrorOrLatency(threshold)`,
`OnAlways()`. Composition via `OnAny(triggers...)` and `OnAll(triggers...)`.

This maps naturally to CQRS middleware: every dispatch has a duration and error
result, which the middleware wraps into a `TriggerContext`.

### 4. Context pre-check (not full cancellation)

`runtime/trace.FlightRecorder.WriteTo()` does not accept `context.Context`.
Rather than removing the parameter (dishonest if we add it back later) or
leaving it unused (lying API), `Snapshot(ctx)` checks `ctx.Done()` before
starting the write. If the context is already cancelled, the snapshot is
skipped. Cancellation during `WriteTo` is not possible.

### 5. Process-global constraint

Go's `runtime/trace` allows only ONE active `FlightRecorder` per process.
`Start()` returns `ErrAlreadyEnabled` if another recorder is already running.
This is documented prominently in `doc.go` and in the error message.

Tests are serialized via `sync.Mutex` because of this constraint.

### 6. io.Closer implementation

`Recorder` implements `io.Closer` (`Close()` stops recording and closes any
file writer). This allows the recorder to participate in shutdown ordering
alongside other closable resources (`stack.Bundle.Close()`, `MultiCloser`).

### 7. Integration points

| Module          | Option                          | Trigger fires on                     |
| --------------- | ------------------------------- | ------------------------------------ |
| `middleware`    | `CommandFlightRecorder`         | Per-dispatch latency/error           |
| `middleware`    | `EventFlightRecorder`           | Per-handler latency/error            |
| `middleware`    | `QueryFlightRecorder`           | Per-query latency/error              |
| `projectionhost`| `WithFlightRecorder`            | Terminal worker failure (WorkerFailed) |
| `decider`       | `WithFlightRecorder[State]`     | Execute latency/error                |
| `stack`         | `WithFlightRecorder`            | Lifecycle management + discovery     |

The `stack.Bundle` integration registers the recorder for `Close()` lifecycle
management and exposes it via `FlightRecorder()` accessor. The consumer calls
`Start()` separately (it can fail), then wires triggers via the accessor.

## Consequences

### Positive

- Zero-dependency core: leaf modules (`decider`, `projectionhost`) add only a
  stdlib-only dependency, staying within their dependency budgets
- Composable triggers: consumers can combine conditions (e.g., "slow AND error")
  without extending the API
- Once-semantics: prevents snapshot races without requiring caller coordination
- Lifecycle integration: `stack.Bundle.Close()` stops the recorder automatically
- CQRS-native triggers: middleware constructs `TriggerContext` from dispatch
  metadata, so consumers write one trigger function and reuse it across
  command/event/query/projection/decider

### Negative

- Process-global constraint: only one recorder per process (Go limitation, not
  ours). Consumers must design around a single recorder instance.
- No mid-WriteTo cancellation: the context pre-check is best-effort. If
  `WriteTo` takes a long time (large trace buffer), cancellation cannot abort it.
- Once-semantics requires `Reset()` for periodic captures. Consumers who want
  multiple snapshots must call `Reset()` between captures.
