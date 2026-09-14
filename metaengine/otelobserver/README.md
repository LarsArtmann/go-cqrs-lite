# metaengine/otelobserver

OpenTelemetry counters for [metaengine](../) health transitions (ADR-0137):
engine quarantine, reactivation, health probes, and catch-up rebuilds become
dashboard metrics instead of log lines you grep after the incident.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/metaengine/otelobserver/v4"
    cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

store, _ := metaengine.Plan(engines, queries...)
obs, err := otelobserver.Attach(store, cqrsotel.NewMeter("cqrs"))
```

## Instruments

| Instrument                         | Attributes          | Meaning                                                    |
| ---------------------------------- | ------------------- | ---------------------------------------------------------- |
| `cqrs.metaengine.quarantine.total` | `engine`            | Engine quarantined after N classified failures             |
| `cqrs.metaengine.reactivate.total` | `engine`, `reason`  | Quarantine lifted: `manual` / `catchup` / `probe-fallback` |
| `cqrs.metaengine.probe.total`      | `engine`, `outcome` | Health probe result: `ok` / `fail`                         |
| `cqrs.metaengine.catchup.total`    | `engine`, `outcome` | Catch-up rebuild attempt: `ok` / `fail`                    |
| `cqrs.metaengine.catchup.replayed` | `engine`            | Events replayed into a rebuilding engine                   |

All instruments are prefixed `cqrs.` so `cqrsotel.NewCQRSViews()` applies
where relevant.

## Wiring

- `Attach(store, meter)` — one call; merges with any hooks the store already
  has (fold/execute metrics keep working) via `metaengine.Hooks.Merge`.
- `New(meter)` + `metaengine.WithHooks(store, obs.Hooks())` — manual wiring
  when you compose hook sets yourself.

Hooks fire after the health mutex is released; observers must not call back
into the Store (meters and loggers are always safe).
