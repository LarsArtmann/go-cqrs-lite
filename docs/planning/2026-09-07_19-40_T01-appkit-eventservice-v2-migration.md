# T01 — go-appkit EventService v2 migration decision note

**Date:** 2026-09-07 · **Status:** DECIDED (executed as go-appkit/cqrs v0.5.0) · **Input:** SUPERB plan T01

Verified 2026-09-07 at source level: `Bundle()`/`SQLitePath` usage is fully contained in
`go-appkit/cqrs` (production file + its tests). No other appkit module (core, realtime, otel,
health, docs-mod, integration, example) calls either. The breaking change is mechanical.

## Field/method mapping (v1 → v2)

| v1 (`EventConfig` v0.4.0) | v2 decision | Rationale |
| --- | --- | --- |
| `SQLitePath string` (required) | `DSN string` + `Driver string` (default `sqlite`). `SQLitePath` kept as deprecated alias. Empty DSN = in-memory (`memory` driver). | Driver names are metaengine's; operator can swap engines without code change. |
| `StackOptions []sqlite.Option` | **DROPPED** → `Pragmas []string`. | `stack/sqlite` preset is removed at v5 (ADR-0123); wrapping it is the v5 cliff. Pragmas are the EngineConfig-shaped equivalent. |
| `Logger *slog.Logger` | kept → `projectionhost.WithLogger` | unchanged semantics |
| `DLQ *DLQConfig` | kept. Default store: aux `*sql.DB` on the same SQLite DSN (one handle shared with the checkpoint store, closed via `System.RegisterCloser`). Non-sqlite drivers must supply `DLQConfig.Store`. | parity with v1's in-bundle store |
| `FlightRecorder` / `FlightRecorderTrigger` | kept → `projectionhost.WithFlightRecorder` | unchanged |
| `Metrics projectionhost.MetricsRecorder` | kept → `projectionhost.WithMetrics` | unchanged |
| `HostOptions []projectionhost.HostOption` | kept → `DomainConfig.ProjectionHostOptions` (consumer options first, derived wiring appended after — wins conflicts) | identical ordering contract to v1 |
| — | **NEW** `CheckpointStore event.CheckpointStore` override; default = persistent `eventstore.NewSQLiteCheckpointStore` on the aux handle (sqlite+DSN), in-memory otherwise | v1 bundle checkpoints were persistent; system defaults to in-memory — parity restores durability |
| — | **NEW** `ConfigPath string` (koanf YAML via `system.LoadConfig`) and `Deployment *system.DeploymentConfig` (pre-loaded config) | G3: the operator declares engines at deployment time |
| — | **NEW** `CommandMiddleware` / `QueryMiddleware` passthroughs (T29 C/Q facade) | M29.2 override hook |

## Service surface mapping

| v1 method | v2 decision |
| --- | --- |
| `Bundle() *stack.Bundle` | **DEPRECATED → removed at next minor; replaced by `System() *system.System`.** Accessors: `EventSink()`/`EventSource()`/`Publisher()` preserved as direct methods so simple consumers don't touch System. |
| `Host() *projectionhost.Host` | kept (delegates to `System().ProjectionHost()`) |
| `DeadLetterStore`, `ReplayDeadLetters`, `ResetProjection` | kept, delegate to host |
| `DB() (*sql.DB, error)` | kept — returns the aux handle (sqlite); Rejection for non-SQL drivers |
| `ReadyCheck`, `LagPerProjection`, `CheckStaleness`, `CheckProjectionStaleness` | kept, delegate to host |
| `StartProjections(ctx)` | kept → `System().Start(ctx)` |
| `Shutdown(ctx)` | kept — appkit idempotency guard + in-flight command drain, then `System().GracefulClose(ctx)` |
| — | **NEW (T29)** `RegisterDecider`, `RegisterCommand`, `RegisterQuery`, `Execute`, `DispatchQuery`, `DispatchQueryChecked`, `CommandDispatcher()`, `DefaultCommandMiddleware(logger)` |

## Default deployment (engine room)

```go
system.DeploymentConfig{
    Engines: map[string]system.EngineConfig{
        "primary": {Driver: driver, DSN: dsn, Pragmas: pragmas},
    },
    Instances: []system.InstanceConfig{
        {Role: system.RoleSourceOfTruth, Engine: "primary"},
        {Role: system.RoleProjections, Engine: "primary"},
    },
}
```

Mirrors FIR (`file-and-image-renamer/pkg/cqrs/system.go`), the one production consumer.
Blank-import contract (go-cqrs-lite gotcha #19): the sqlite driver self-registers via
`_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"` in appkit's go.mod
graph; `memory` is registered by metaengine core. A missing blank import fails at
`system.New` with "unknown driver" — consumers adding engines must blank-import them.

## Lifecycle semantics (v1 parity)

- Construction failure → close already-opened resources (`closeOnConstructionFailure` closes
  the aux handle; `system.New` closes its own engines on failure).
- `Shutdown` idempotent under mutex; second call is a no-op nil.
- Drain ordering: in-flight commands (tracked by an always-installed WaitGroup middleware)
  → `System.GracefulClose` (drainers → projection host stop → engines in dependency order).

## Rejected alternatives

- **Keeping a deprecated `Bundle()`** backed by a hand-built stack shim: violates go-cqrs-lite
  ADR-0126 (no hand-written Store wrappers) and guardrail 9 (never build on Bundle sinks).
- **Default middleware chain installed silently:** appkit YAGNI (guardrail 1) — middleware is
  opt-in via `EventConfig.CommandMiddleware` + `DefaultCommandMiddleware(logger)` builder.
- **DLQ in a separate `<db>.dlq` file:** changes v1 semantics (same-database table) for no gain.
