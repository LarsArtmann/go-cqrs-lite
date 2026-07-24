## Common Pitfalls (FAQ)

> The mistakes AI assistants and new users make most often. Check here when something won't compile, won't decode, or returns an unexpected error.

### "My event payload won't decode"

**Cause:** `event.NewEvent` takes `[]byte`, not a struct. You must encode the payload before passing it.

```go
// Wrong — won't compile (struct where []byte expected)
evt, _ := event.NewEvent("user.created", aggID, "User", 1, UserCreated{Name: "Alice"})

// Correct — encode first
payload, _ := codec.JSONCodec{}.Encode(UserCreated{Name: "Alice"})
evt, _ := event.NewEvent("user.created", aggID, "User", 1, payload)

// Or use NewEvents (accepts []any, encodes internally)
events, _ := event.NewEvents(aggID, "User", 0,
    []event.Type{"user.created"}, []any{UserCreated{Name: "Alice"}})
```

### "My decider Repository won't load — type parameter error"

**Cause:** Go infers the type parameter from the `Decider[State]` argument, so you rarely need to specify it explicitly.

```go
// Both work — the second is more idiomatic
repo, _ := decider.NewRepository[UserState](store, bus, d)
repo, _ := decider.NewRepository(store, bus, d) // type inferred from d (Decider[UserState])
```

### "snapshot.EveryNEvents returns an error"

**Cause:** It returns `(SnapshotStrategy, error)`, not just the strategy. Handle the error:

```go
strategy, _ := snapshot.EveryNEvents(100) // ← returns two values
repo, _ := decider.NewRepository(store, bus, d, decider.WithSnapshotStrategy(strategy))
```

### "Projection Builder — `.On()` doesn't exist as a method"

**Cause:** `On` is a **free function** with a type parameter, not a method on `*Builder`:

```go
// Wrong
b.On("user.created", handler)

// Correct — free function with type parameter
projection.On[UserCreated](b, "user.created", codec.CBORCodec{}, handler)
```

### "Pebble KV — `NewKVAdapter` not found"

**Cause:** The constructor is `NewKVStore`, not `NewKVAdapter`. The option is `WithSyncWrites()`, not `WithKVSyncWrites()`:

```go
// Wrong
kvStore := pebble.NewKVAdapter(db, pebble.WithKVSyncWrites())

// Correct
kvStore, _ := cqrspebble.NewKVStore(db, cqrspebble.WithSyncWrites())
```

### "My SQL backend needs a dialect parameter"

**Cause:** `NewSQLBackend` infers the dialect from the `*sql.DB` driver name — no explicit dialect needed:

```go
// Wrong
backend, _ := storage.NewSQLBackend(db, sql.PostgresDialect{})

// Correct — dialect auto-detected
backend, _ := storage.NewSQLBackend(db)
```

### "catalog.NewRegistry needs arguments"

**Cause:** It requires a title and version:

```go
// Wrong
reg := catalog.NewRegistry()

// Correct
reg := catalog.NewRegistry("My API", "1.0.0")
```

### "Local `go mod tidy` fails with eventtest pseudo-version errors"

**Cause:** `event/v4/eventtest` is a standalone Go module at `event/v4/eventtest/go.mod` with no published tag. It resolves via `replace` directives in `go.work`, but `go mod tidy` in a **consumer** workspace can't inherit those replaces.

**Fix (consumer side):** Add a `replace` directive in your `go.work`:

```go
replace github.com/larsartmann/go-cqrs-lite/event/v4/eventtest => ../go-cqrs-lite/event/v4/eventtest
```

Inside the go-cqrs-lite repo, run `go mod tidy -e` (the warnings are cosmetic — the build works via `go.work`).

### "event.New() rejects nil payload but event.NewEvent() accepts []byte{}"

**Cause:** `event.New()` validates the payload (rejects nil), while `event.NewEvent()` accepts raw `[]byte` (including empty). This is intentional — `New()` is the typed-payload constructor, `NewEvent()` is the low-level constructor for stores and tests.

```go
// event.New() — typed, validates payload
evt, err := event.New("user.created", aggID, "User", 1, UserCreated{Name: "Alice"})
// err is non-nil if payload is nil

// event.NewEvent() — raw bytes, no validation
evt, err := event.NewEvent("user.created", aggID, "User", 1, []byte{})
// works fine — for test helpers and store internals
```

### "WithEnricher can't infer the type parameter"

**Cause:** Go generics can't infer `State` from the enricher function type alone. Provide it explicitly:

```go
// Wrong — compiler error: cannot infer State
repo := decider.WithEnricher(event.CommandCausalityEnricher)

// Correct — explicit type parameter
repo := decider.WithEnricher[UserState](event.CommandCausalityEnricher)
```

### "go-error-family vs event/v4 error constructors — which should I use?"

**Relationship:** `go-error-family` is the **standalone** extraction of the five-family error taxonomy (Rejection, Conflict, Transient, Infrastructure, Corruption). `event/v4` wraps the **same** families with event-store context (event payloads, codec integration, metadata).

- **CQRS apps:** use `errorfamily.NewRejection(...)`, `errorfamily.WrapTransient(err, ...)` directly from [go-error-family](https://github.com/larsartmann/go-error-family).
- **Non-CQRS apps** (middleware-only consumers, HTTP services): use `go-error-family` directly. It's the same classification without event coupling.
- The event package retains type aliases (`event.Family`, `event.Error`) for backward compatibility, but construction/classification functions were removed. Always import `go-error-family` directly.

### "Is eventtest.FakeBus production-safe?"

**Yes** — `FakeBus` is a synchronous in-memory event bus suitable for **single-process production apps**. The name is historical (it started as a test double). For multi-process or distributed setups, use `watermill.EventBus` with a real message broker.

### "Can I share one \*sql.DB for events AND read models?"

**Yes.** The `stack/*` presets create separate `*sql.DB` connections, but you can wire a shared database manually:

```go
// Shared *sql.DB for events + projections + read models
db, _ := sql.Open("sqlite3", "app.db")
eventStore, _ := storage.NewSQLiteEventStore(db)
viewStore, _ := storage.NewSQLiteViewStore[TodoView, TodoID](db, mapper)
// Pass the SAME db to both — transactions span both if needed.
```

Do NOT use `stack/sqlite.New()` for this case — it creates separate connections. See `references/recipes.md` §2.3 for the shared-database recipe.

### "When should I use snapshots?"

**Rule of thumb:** use snapshots when your largest stream exceeds **~100 events**. Below that, full replay is faster than snapshot load + remaining replay.

```go
// Snapshot every 50 events — good for streams that grow large
strategy, _ := snapshot.EveryNEvents(50)
repo, _ := decider.NewRepository(store, bus, d,
    decider.WithSnapshotStore(snapStore),
    decider.WithSnapshotStrategy(strategy),
)
// Load() transparently uses snapshots when available.
```

For small streams (5-20 events), skip snapshots — the overhead exceeds the savings.

### "Storage package restructure — where did types move?"

v3.5.0 reorganized `storage/` into sub-packages. All old import paths still work via
backward-compatible aliases, but the canonical paths are:

| Old path (still works)         | Canonical path (v3.5+)                                  | Types                   |
| ------------------------------ | ------------------------------------------------------- | ----------------------- |
| `storage.SQLViewStore`         | `storage.NewSQLiteViewStore` / `storage.NewPGViewStore` | View store constructors |
| `storage.ViewMapper`           | `storage.ViewMapper` (unchanged)                        | View column mapping     |
| `storage.RelationalProjection` | `storage.NewRelationalProjection` (unchanged)           | Multi-table projections |
| `storage/sql.Dialect`          | `storage/sql.Dialect` (unchanged)                       | SQL dialect types       |
| `storage/view.*`               | `storage/view.*` (unchanged)                            | View internals          |

**If your old code compiles, it still works.** The aliases are permanent.
