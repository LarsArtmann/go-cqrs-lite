# Migration Guide: Hand-Wired → Stack Presets

> **Stop wiring infrastructure by hand.** The `stack/` layer eliminates the boilerplate every consumer currently reimplements: schema migration, dialect mapping, bus wiring, projection replay, and deduplication. This guide shows you how to replace 200–400 lines of custom infrastructure code with 5–10 lines of stack preset.

---

## Why Migrate?

| Concern              | Hand-wired                                          | Stack preset                              |
| -------------------- | --------------------------------------------------- | ----------------------------------------- |
| Schema migration     | Manual `CREATE TABLE` + migration discovery         | Automatic (auto-migrate on first run)     |
| Bus wiring           | Manual event.Bus + subscriber registration          | One line: `sqlite.New("app.db")`          |
| Projection replay    | Custom journal reader + dedup logic (150–260 lines) | `bundle.CatchUpSubscriber()` (0 lines)    |
| View updates         | Custom handler with tombstone checks                | `stack.Materialize` struct with callbacks |
| Dialect mapping      | `if sqlite ... if postgres ...` branches            | Preset swap — zero conditionals           |
| Storage switching    | Build tags (`-tags turso`) — silent prod bugs       | Runtime choice — one function call        |
| Connection lifecycle | Manual `defer db.Close()` everywhere                | `bundle.Close()` handles everything       |

---

## Step 1: Replace Your Event Store

### Before (hand-wired)

```go
// 80+ lines: open DB, create schema, create store, wire bus
db, _ := sql.Open("sqlite", "app.db")
storage.SQLiteInitSchema(ctx, db)
backend, _ := storage.NewSQLiteBackend(db)
eventStore := backend.EventStore()
snapStore, _ := backend.SnapshotStore()
cpStore, _ := backend.CheckpointStore()
bus := cqrswatermill.NewEventBus()
// ... 20 more lines of wiring
defer db.Close()
```

### After (stack preset)

```go
// 1 line: everything wired, schema migrated, lifecycle owned
bundle, _ := sqlite.New("app.db")
defer bundle.Close()

// Access the same interfaces you used before:
var sink event.EventSink = bundle.EventSink
var source event.EventSource = bundle.EventSource
```

**Lines removed: ~80.** The preset handles schema migration, WAL enablement, busy_timeout, backend creation, store extraction, and lifecycle registration.

---

## Step 2: Replace Your Projection Runner

### Before (hand-wired, ~150–260 lines)

```go
// Replay historical events from journal
events := journal.ReadAll(ctx)
for _, evt := range events {
    if isDuplicate(evt.ID, checkpoint) { continue }
    applyToView(evt)
    saveCheckpoint(evt.ID)
}
// Subscribe to live events
bus.Subscribe(ctx, topic, func(evt event.Event) {
    if isDuplicate(evt.ID, checkpoint) { continue }
    applyToView(evt)
    saveCheckpoint(evt.ID)
})
// ... dedup logic, ordering logic, error handling, tombstone checks
```

### After (CatchUpSubscriber + Materialize)

```go
// Define how events map to views
mat := stack.Materialize[UserView, UserID]{
    Store:        bundle.ReadModels,
    KeyFromEvent: func(evt event.Event) (UserID, error) { /* ... */ },
    OnCreate:     func(ctx, evt) (*UserView, error) { /* ... */ },
    OnUpdate:     func(ctx, evt, existing) (*UserView, error) { /* ... */ },
    OnTombstone:  func(ctx, evt, existing) (*UserView, error) { /* ... */ },
}

// CatchUpSubscriber handles replay + live + dedup automatically
catchUp, _ := bundle.CatchUpSubscriber()
defer catchUp.Close()

msgs, _ := catchUp.Subscribe(ctx, "user.*")
// Consume from a SINGLE goroutine (FIFO ordering guaranteed)
for msg := range msgs {
    _ = mat.HandleMessage(msg)
    msg.Ack()
}
```

**Lines removed: ~200.** CatchUpSubscriber handles:

- Phase 1: journal replay with `ProcessingMode=ModeReplay`
- Phase 2: live handoff with EventID-based deduplication
- Checkpoint persistence after every forwarded event
- Tombstone-aware view updates via Materialize callbacks

---

## Step 3: Replace Build-Tag Storage Switching

### Before (dangerous — SEC's production bug)

```go
//go:build !turso
func newStore() event.Store { return memory.NewMemoryStore() }
//go:build turso
func newStore() event.Store { return tursoStore }
```

If the Dockerfile omits `-tags turso`, production silently runs in-memory.
**All data is lost on restart.**

### After (runtime preset choice)

```go
func newBundle(env string) (*stack.Bundle, error) {
    switch env {
    case "production":
        return postgres.New(os.Getenv("DATABASE_URL"))
    case "staging":
        return sqlite.New("/data/app.db")
    default:
        return memory.New()
    }
}
```

No build tags. No silent in-memory production. The deployer chooses at runtime.

---

## Step 4: Use the Multi-DB Split

### The problem: everything in one database

Write-heavy event streams compete with read-model scans for I/O. Under load, event appends block behind long view queries.

### The solution: multi-DB split

```go
bundle, _ := sqlite.New("/data/primary.db",
    sqlite.WithEventDB("/data/events.db"),    // events + snapshots + checkpoints
    sqlite.WithQueryDB("/data/queries.db"),   // commands + queries (audit)
    sqlite.WithViewDB("/data/views.db"),      // read models (cqrs_kv)
)
```

Same API for Turso and Postgres:

```go
// Turso
bundle, _ := turso.New("/data/app.libsql",
    turso.WithEventDB("/data/events.libsql"),
    turso.WithQueryDB("/data/queries.libsql"),
    turso.WithViewDB("/data/views.libsql"),
)

// Postgres
bundle, _ := postgres.New(primaryDSN,
    postgres.WithEventDB("postgres://host/events_db"),
    postgres.WithQueryDB("postgres://host/queries_db"),
    postgres.WithViewDB("postgres://host/views_db"),
)
```

The consumer code doesn't change — the Bundle fields point to the right stores.

---

## Decision Checklist

| Question                    | If yes →                                                   |
| --------------------------- | ---------------------------------------------------------- |
| Single process, dev/test?   | `memory.New()`                                             |
| Single process, persistent? | `sqlite.New("app.db")`                                     |
| High-throughput embedded?   | `pebble.New("data/")`                                      |
| Multi-process, shared DB?   | `postgres.New(dsn, postgres.WithDistributedBus(listener))` |
| Edge/remote sync?           | `turso.NewSync(ctx, path, remoteURL, token)`               |
| Need I/O isolation?         | Add `WithEventDB` + `WithQueryDB` + `WithViewDB`           |

---

## What NOT to Migrate

- **Session management** — Sessions are an application-layer concern, not CQRS infrastructure. Keep your session store as-is.
- **Custom SQL queries for dashboards** — Use `bundle.ReadModels` for materialized views, but ad-hoc analytics queries belong in your application code.
- **External service integrations** — The Bundle handles CQRS data stores only. External APIs, email, webhooks stay in your application layer.

See [ADR-0034](adr/0034-session-store-boundary.md) for the session store boundary decision.

---

## Related Documents

- [PRESETS.md](./PRESETS.md) — Available presets and their options
- [INFRASTRUCTURE_RECOMMENDATIONS.md](./INFRASTRUCTURE_RECOMMENDATIONS.md) — Which DB fits which concern
- [example/deployer-first](../example/deployer-first/) — Working example of the recommended pattern
- [stack/contracttest](../stack/contracttest/) — Contract test suite for preset verification
