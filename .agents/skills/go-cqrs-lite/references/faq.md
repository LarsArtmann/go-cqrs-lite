## Common Pitfalls (FAQ)

> The mistakes AI assistants and new users make most often. Check here when something won't compile, won't decode, or returns an unexpected error.

> **Contents** — jump to your symptom:
>
> - [Payload won't decode](#my-event-payload-wont-decode)
> - [Decider Repository type-parameter error](#my-decider-repository-wont-load--type-parameter-error)
> - [EveryNEvents returns an error](#snapshoteverynevents-returns-an-error)
> - [Projection Builder `.On()` doesn't exist](#projection-builder--on-doesnt-exist-as-a-method)
> - [Pebble `NewKVAdapter` not found](#pebble-kv--newkvadapter-not-found)
> - [SQL backend dialect parameter](#my-sql-backend-needs-a-dialect-parameter)
> - [MySQL `parseTime=true` required](#mysql-parsetimetrue-is-required)
> - [catalog.NewRegistry needs arguments](#catalognewregistry-needs-arguments)
> - [eventtest pseudo-version tidy errors](#local-go-mod-tidy-fails-with-eventtest-pseudo-version-errors)
> - [event.New vs event.NewEvent payloads](#eventnew-rejects-nil-payload-but-eventnewevent-accepts-byte)
> - [WithEnricher type inference](#withenricher-cant-infer-the-type-parameter)
> - [go-error-family vs event constructors](#go-error-family-vs-eventv4-error-constructors--which-should-i-use)
> - [eventtest.FakeBus in production?](#iseventtestfakebus-production-safe)
> - [Sharing one \*sql.DB](#can-i-share-one-sqldb-for-events-and-read-models)
> - [When to use snapshots](#when-should-i-use-snapshots)
> - [Storage package restructure](#storage-package-restructure--where-did-types-move)
> - [Integrating metaengine](#how-do-i-integrate-metaengine-with-my-stack)
> - [Circuit breaker: middleware vs failsafe-go](#should-i-use-the-circuit-breaker-from-middleware-or-failsafe-go-directly)
> - [ProjectionSink.Increment went negative](#projectionsinkincrement-went-negative--shouldnt-it-clamp-to-zero)
> - [KeysetPositionQuery empty string](#why-does-storagesqlkeysetpositionquery-return-an-empty-string-for-a-bad-table-name)
> - [Planned-collection scan misses meta_map rows](#my-planned-collection-scan-doesnt-see-rows-that-exist-in-meta_map)
> - [Will the v5 cut break my imports?](#will-the-v5-cut-break-my-imports-what-is-going-away)

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

### "MySQL: `parseTime=true` is required"

**Cause:** Without `parseTime=true` in the DSN, `DATETIME` columns are returned as
`[]byte` instead of `time.Time`, breaking event timestamps:

```go
// Wrong — timestamps will be []byte
b, _ := mysql.New("root:pass@tcp(localhost:3306)/myapp")

// Correct — add parseTime=true
b, _ := mysql.New("root:pass@tcp(localhost:3306)/myapp?parseTime=true")
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

**Relationship:** `go-error-family` is the **standalone** extraction of the six-family error taxonomy (Rejection, Conflict, Transient, Infrastructure, Corruption, Orchestration). `event/v4` wraps the **same** families with event-store context (event payloads, codec integration, metadata).

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

### "How do I integrate metaengine with my stack?"

Use `stack.WithMetaEngine(store)` to register a metaengine Store with the Bundle.
The Bundle manages its lifecycle (Close), and benchkit auto-discovers it.

```go
store, _ := metaengine.Plan(engines, queries...)
bundle, _ := sqlite.New(dsn, sqlite.WithStack(stack.WithMetaEngine(store)))
// bundle.MetaEngine() returns the store for runtime queries
// bundle.Close() closes the store automatically
```

For projection lifecycle (checkpoint, retry, DLQ), wrap the store in
`projectionadapter.New(name, store, decoder)` and register with the projection host.
The consumer calls `metaengine.Plan()` themselves because typed generics
can't flow through `any` constraints.

### "Should I use the circuit breaker from middleware/ or failsafe-go directly?"

For CQRS integration (command/event/query dispatchers), use
`middleware/circuit_breaker.go` which wraps `failsafe-go/circuitbreaker` and
provides `CircuitBreakerConfig` with the CQRS-specific API surface.

For standalone circuit breaking outside the CQRS pipeline (e.g., protecting
an HTTP client, a database connection pool, or an external API call), import
`failsafe-go/circuitbreaker` directly. The middleware package is designed
specifically for the command/event/query dispatcher chain and adds no value
when you're not dispatching through CQRS types.

See `middleware/circuit_breaker.go` for the integration pattern if you need
to wire circuit breaking into a custom execution path.

### "ProjectionSink.Increment went negative — shouldn't it clamp to zero?"

No — this is deliberate. `Increment` emits
`counter = COALESCE(counter, 0) + delta` and does NOT clamp on negative
deltas. A negative rollup counter is a loud signal that your projection saw
a `DELETE` (or decrement) for a row that never had the matching `CREATE`
(increment) — usually a replay bug, a missed event, or a handler ordering
problem. Silently clamping to zero would hide that data-loss bug behind a
plausible-looking number.

What to do instead: fix the fold so increments and decrements pair up
(replay the projection from zero if the rollups are already wrong), and
monitor for `counter < 0` as a projection-health alert. See
`storage/relational/sink.go` (`Increment` doc comment) for the rationale.

### "Why does `storage/sql.KeysetPositionQuery` return an empty string for a bad table name?"

It was a deprecated footgun: the pre-validation keyset builder silently
yielded `""` when the table or timestamp column failed identifier
validation, and the empty query surfaced downstream as a baffling SQL
syntax error instead of a classified rejection. Use
`storage/sql.KeysetPositionQueryChecked` — same result shape, but invalid
identifiers return a fail-fast Infrastructure error. `KeysetPositionQuery`
stays as a Deprecated wrapper until v5; every in-repo journal path
(`ReadStreamFrom`, `JournalReader.ReadFrom`) already uses the checked form.
The same `storage/sql.ValidateJournalIdentifiers` guard protects the
`JournalReader` and cursor-timestamp interpolation paths, backed by
adversarial injection tests and a persisted fuzz corpus.

## Will the v5 cut break my imports? What is going away?

Everything scheduled for deletion at v5 (ADR-0123/0126/0127) already carries a
`Deprecated:` doc marker in the code — `go build` succeeds, but linters that
check deprecations (staticcheck SA1019, gopls) will flag uses. The big
buckets, all deleted at the v5.0.0 cut:

- **Tombstone metadata APIs** (`event.DetectTombstone`, `MarkTombstone`,
  `MarkRebirth`, `TombstoneStatus`, `Metadata.Tombstone`): deletion is
  expressed as a domain event type (`user.deleted`), read via
  `listing.StatusMiddleware` / the classifier. See ADR-0114.
- **`stack/` presets and `Materialize`/`RunProjections`/`GraphProjection`**:
  one composition root (`system.New`) and one projection runner
  (`projectionhost`) replace them. See ADR-0123.
- **`storage/view` + `storage/relational`**: the metaengine auto-projection
  and `storage/relational`'s sink replacement path cover the same ground.
- **`transport/http` (SSE) + `transport/grpc`**: use `watermill/` brokers (or
  go-sse directly). See ADR-0127.
- **ADR-0126 compatibility shells** (`schema.VersionedStore`,
  `schema.VersionedSeekableJournal`, `metadata.CustomData`, and friends):
  compose `event.DecorateStore` / `event.DecorateJournal` with
  `SinkTransform`/`SourceTransform` instead.

Nothing in the tier-0/1 core (`id`, `record`, `event`, `command`, `query`,
`decider`, `metaengine`) is removed at v5 beyond the tombstone metadata
surface above; v5 renames (`StreamRef` → `StreamKey`, stricter constructors)
are migration-guide items, not deletions.

### "My planned-collection scan doesn't see rows that exist in meta_map"

That is the no-backfill contract working as designed: after
`ApplyLayoutPlan`, `MapScan`/`PushdownMapScan`/`MapUpdate` read ONLY the
planned table (visibility split closed on pg, mysql, sqlite, and duckdb).
Rows written before registration stay in `meta_map`. Remedies, in order of
preference: (1) register planned tables at deployment time before data
flows; (2) run the opt-in
`metaengine.BackfillPlannedCollection(ctx, eng, collection, batchSize)` —
idempotent, requires the `KeyScanBackend` capability (pgengine,
mysqlengine); (3) for acceleration of EXISTING data without migration, use
`ApplyLayout` generated columns instead. If a planned table is empty when it
should have data, check `Doctor`'s `--- Planned tables ---` section: it
lists every registered planned table with a live row count, so you can see
at a glance whether the collection you queried is registered — and on which
engine.

**Automated detection:** `cqrs-lint` catches this drift for you — rule
V007 (`v5-removed-api-usage`) flags every wholly-removed module (stack
presets, `storage/relational`, `storage/view`) and each deprecated symbol
above at its use site, with the ADR reference and the sanctioned replacement
in the suggestion. F030 covers the `transport/*` module imports. Run
`cqrs-lint .` in CI and the v5 cut becomes a non-event.
