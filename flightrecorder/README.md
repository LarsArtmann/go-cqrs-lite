# flightrecorder

> Wraps Go 1.25's `runtime/trace.FlightRecorder` with composable triggers for CQRS/ES systems.

A flight recorder buffers the last few seconds of execution trace in memory. When a problem is detected (slow operation, error, panic), the program snapshots exactly the problematic window for offline analysis with `go tool trace`.

## Quick Start

```go
import (
    "context"
    "time"

    flightrecorder "github.com/larsartmann/go-cqrs-lite/flightrecorder/v4"
)

recorder, _ := flightrecorder.New(
    flightrecorder.WithMinAge(10*time.Second),
    flightrecorder.WithMaxBytes(1<<20), // 1 MiB
    flightrecorder.WithFile("snapshot.trace"),
)
if err := recorder.Start(); err != nil {
    if errors.Is(err, flightrecorder.ErrAlreadyEnabled) {
        // Another recorder is already active in this process.
        log.Fatal(err)
    }
    log.Fatal(err)
}
defer recorder.Close() // stops recording + closes the file

// Later, when something goes wrong:
recorder.Snapshot(context.Background())
```

## Lifecycle

| Method | Purpose |
|--------|---------|
| `Start()` | Begins buffering. Returns `ErrAlreadyEnabled` if another recorder is active. |
| `Close()` | Stops recording + closes file writer. Implements `io.Closer`. Safe to call multiple times. |
| `Stop()` | Stops recording only (does not close file). Use `Close()` for normal shutdown. |

> **Process-global constraint:** Go's `runtime/trace` allows only ONE active flight recorder per process. If two recorders call `Start()`, the second gets `ErrAlreadyEnabled`.

## CQRS Middleware Integration

```go
import (
    "github.com/larsartmann/go-cqrs-lite/middleware/v4"
    flightrecorder "github.com/larsartmann/go-cqrs-lite/flightrecorder/v4"
)

recorder, _ := flightrecorder.New(
    flightrecorder.WithMinAge(10*time.Second),
    flightrecorder.WithFile("slow.trace"),
)
recorder.Start()
defer recorder.Close()

// Capture when any command exceeds 100ms OR errors:
cmdDisp.Use(middleware.CommandFlightRecorder(recorder,
    flightrecorder.OnErrorOrLatency(100*time.Millisecond)))

// Same for events and queries:
bus.Use(middleware.EventFlightRecorder(recorder,
    flightrecorder.OnError()))
qryDisp.Use(middleware.QueryFlightRecorder(recorder,
    flightrecorder.OnLatency(500*time.Millisecond)))
```

## Snapshot Methods

| Method | Description |
|--------|-------------|
| `Snapshot(ctx)` | Writes trace to the configured writer (or file). Checks `ctx.Done()` before writing. |
| `SnapshotToFile(ctx, path)` | Writes trace to a specific file path. |
| `SnapshotIf(ctx, tc, trigger)` | Evaluates trigger; captures if it returns true. |
| `Reset()` | Re-arms the once-latch for another capture. |

Once-semantics: by default only the **first** successful `Snapshot`/`SnapshotToFile` wins. Call `Reset()` to allow subsequent captures.

## Triggers

| Trigger | Fires when |
|---------|-----------|
| `OnLatency(threshold)` | Duration > threshold |
| `OnError()` | Non-nil error |
| `OnErrorOrLatency(threshold)` | Error OR slow |
| `OnAlways()` | Every operation (first only, due to once-semantics) |
| `OnAny(triggers...)` | Any trigger fires |
| `OnAll(triggers...)` | All triggers fire |

## Analyzing

```bash
go tool trace snapshot.trace
```

## Design

- **Zero dependencies** — stdlib only (`runtime/trace`, `sync`, `time`, `context`, `io`, `os`)
- **Once-semantics** — by default, only the first snapshot succeeds (prevents races when multiple goroutines detect a problem simultaneously). Call `Reset()` for multiple captures.
- **Async capture** — the middleware snapshots in a goroutine to avoid blocking the request path
- **Process-global** — Go's `runtime/trace` allows only ONE active flight recorder per process. `Start()` returns `ErrAlreadyEnabled` if another is already running.
