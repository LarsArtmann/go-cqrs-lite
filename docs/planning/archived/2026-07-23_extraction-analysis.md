# Extraction Analysis: go-cqrs-lite → Standable Repos

> **Date:** 2026-07-23
> **Question:** "This project got REALLY big, what would be something we could extract into a standalone stable repo?"

## Evaluation Criteria

A module is a good extraction candidate when it is:

1. **Domain-agnostic** — useful outside CQRS/Event-Sourcing
2. **Loosely coupled** — few shallow importers in this repo
3. **Self-contained** — zero or one local dependency
4. **Substantial** — enough API surface to justify its own repo
5. **Stable** — well-tested, clear contract

## Dependency Landscape (by coupling depth)

| Module           | Local Deps | Production Importers        | Coupling      | Domain-Specific?                  |
| ---------------- | ---------- | --------------------------- | ------------- | --------------------------------- |
| **retry/**       | 0          | 1 (middleware)              | **Shallow**   | No — pure utility                 |
| **idempotency/** | 0          | 2 (middleware, kvstore)     | **Shallow**   | No — pure utility                 |
| dispatcher/      | 0          | 3 (command, query, storage) | **Deep**      | Mild (generic types)              |
| kv/ (core)       | 1 (codec)  | ~15                         | **Very deep** | Core is clean; view layer is CQRS |
| codec/           | 0          | ~25                         | **Deepest**   | Serialization foundation          |

## Module Size Reference

| Module            | Src Files | Test Files | Src LOC |
| ----------------- | --------- | ---------- | ------- |
| `projection/`     | 1         | 1          | 57      |
| `dedup/`          | 1         | 2          | 94      |
| `metadata/`       | 2         | 1          | 140     |
| `prometheus/`     | 2         | 1          | 168     |
| `deriver/`        | 2         | 1          | 177     |
| `scenario/`       | 2         | 1          | 258     |
| `retry/`          | 3         | 1          | 217     |
| `scheduling/`     | 3         | 1          | 307     |
| `idempotency/`    | 3         | 2          | 355     |
| `dispatcher/`     | 4         | 3          | 303     |
| `testutil/`       | 5         | 1          | 246     |
| `listing/`        | 6         | 9          | 546     |
| `snapshot/`       | 7         | 8          | 523     |
| `otel/`           | 11        | 8          | 766     |
| `query/`          | 10        | 15         | 801     |
| `kv/`             | 10        | 6          | 1,448   |
| `graph/`          | 10        | 5          | 1,940   |
| `projectionhost/` | 10        | 8          | 2,044   |
| `command/`        | 12        | 14         | 946     |
| `codec/`          | 13        | 12         | 1,085   |
| `id/`             | 15        | 13         | 619     |
| `encryption/`     | 18        | 23         | 1,777   |
| `signing/`        | 19        | 20         | 1,875   |
| `event/`          | 38        | 39         | 3,850   |

## Ranked Recommendations

### #1: Extract `retry/` → `go-retry` — Best Candidate

| Attribute            | Assessment                                                         |
| -------------------- | ------------------------------------------------------------------ |
| **LOC**              | 217 source + tests                                                 |
| **Dependencies**     | `go-error-family` only (1 dep)                                     |
| **CQRS coupling**    | **Zero** — comments mention CQRS only as "what it doesn't receive" |
| **Importers**        | 1 (middleware wraps it as a convenience)                           |
| **API**              | `Do()`, `Config`, `Backoff()`, `DefaultConfig()` — textbook Go API |
| **Standalone value** | Every Go service needs exponential backoff + jitter                |

**Why it wins:** Universal need, zero domain leakage, cleanest API, trivially extractable. The `go-error-family` dependency is the only coupling point, and it's used purely for error classification (`IsRetryable` default + sentinel errors). Either keep it as a dep or swap to `errors.New`.

**Exported API surface:**

```go
// Errors
var ErrExhausted = errorfamily.NewInfrastructure("retry.exhausted", ...)
var ErrCanceled  = errorfamily.NewInfrastructure("retry.canceled", ...)

// Types
type AttemptFunc func(ctx context.Context, attempt int) error

type Config struct {
    MaxAttempts  int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
    IsRetryable  func(error) bool
    OnRetry      func(attempt int, delay time.Duration, err error)
    OnExhausted  func(attempts int, err error)
}

// Functions
func DefaultConfig() Config
func (c Config) Validate() error
func Do(ctx context.Context, config Config, fn AttemptFunc) error
func Backoff(config Config, attempt int) time.Duration
func ComputeDelay(initial, max time.Duration, multiplier float64, attempt int) time.Duration
```

---

### #2: Extract `idempotency/` → `go-idempotency` — Strong Candidate

| Attribute            | Assessment                                                                     |
| -------------------- | ------------------------------------------------------------------------------ |
| **LOC**              | 355 source + tests                                                             |
| **Dependencies**     | `go-error-family` only (1 dep)                                                 |
| **CQRS coupling**    | **Zero** — keys are opaque `string`, TTL is `time.Duration`                    |
| **Importers**        | 2 (middleware adapter, kvstore subpackage)                                     |
| **API**              | 3-method `Store` interface: `Seen`, `Record`, `CheckAndRecord` + `MemoryStore` |
| **Standalone value** | Any at-least-once delivery system needs dedup                                  |

**Why it wins:** The cleanest interface in the repo (3 methods), functional in-memory implementation out of the box, zero domain types. The `kvstore` subpackage would stay in go-cqrs-lite as an adapter.

**Exported API surface:**

```go
// Errors
var ErrDuplicate = errorfamily.NewConflict("idempotency.duplicate", ...)

// Interface
type Store interface {
    Seen(ctx context.Context, key string) (bool, error)
    Record(ctx context.Context, key string, ttl time.Duration) error
    CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error
}

// In-memory implementation
type MemoryStore struct { ... }
func NewMemoryStore(sweepInterval time.Duration) *MemoryStore
func (s *MemoryStore) Seen(ctx context.Context, key string) (bool, error)
func (s *MemoryStore) Record(ctx context.Context, key string, ttl time.Duration) error
func (s *MemoryStore) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error
func (s *MemoryStore) Close()
```

---

### #3: Split `kv/` → Extract core as `go-kv` (conditional)

The **core interfaces** (`Store`, `Reader`, `Writer`, `Iterator`, `Batch`, `ConditionalWriter`, `MemStore`) are excellent — zero CQRS types, matches the Pebble/BadgerDB/bbolt common denominator. But `view_store.go` (SQL query DSL, tombstone filtering, projection reset) is deeply CQRS-coupled and imported by ~15 modules.

**Recommendation:** Don't extract yet. The view-model layer would need to be split out first (into a separate `viewstore/` module within this repo), and ~15 modules would need their import paths updated. High effort, moderate payoff.

**Core KV API (clean, extractable):**

```go
type Store interface { Reader; Writer; io.Closer }
type Reader interface {
    Get(ctx context.Context, key []byte) ([]byte, error)
    Has(ctx context.Context, key []byte) (bool, error)
    NewIterator(ctx context.Context, prefix []byte) (Iterator, error)
}
type Writer interface {
    Set(ctx context.Context, key, value []byte) error
    Delete(ctx context.Context, key []byte) error
    Batch(ctx context.Context) (Batch, error)
}
type Iterator interface { Next() bool; Key() []byte; Value() []byte; Error() error; Close() error }
type Batch interface { Set(...); Delete(...); Commit(ctx) error; Close() error }
type ConditionalWriter interface {
    SetIfAbsent(ctx context.Context, key, value []byte) (bool, error)
}

// In-memory implementation
type MemStore struct { ... }
func NewMemStore() *MemStore
```

**View-model layer (stays in go-cqrs-lite):**

```go
// These are CQRS-specific and cannot be extracted:
type ViewStore[V any, K fmt.Stringer] interface { Get; Set; Delete; Scan }
type ViewQuery struct { Conditions; OrderBy; Desc; Order; Limit; Offset; Keyset; RawWhere; RawArgs }
type TombstoneQuerier[V any] interface { QueryByTombstone(...) }
type ViewResetter[V any] interface { DeleteAll(ctx) error }
// ... 7 optional interfaces, SQL query DSL
```

---

## What NOT to Extract

| Module        | Why it stays                                                                              |
| ------------- | ----------------------------------------------------------------------------------------- |
| `codec/`      | Fundamental serialization layer, 25+ importers, self-describing envelope is CQRS-specific |
| `dispatcher/` | Architectural foundation — command/query dispatch are built directly on it                |
| `kv/` (full)  | View-model layer is deeply CQRS-coupled, 15 importers                                     |
| `signing/`    | Tightly integrated with event types and codec                                             |
| `encryption/` | Tightly integrated with event types and codec                                             |
| `otel/`       | CQRS-opinionated histogram views baked into setup                                         |
| `dedup/`      | Too small for a standalone repo (94 LOC)                                                  |
| `id/`         | CQRS marker types (AggregateID, EventID, CommandID) are the entire value                  |

## Dependency Layer Map

```
Layer 0 (leaf):    retry  dedup  codec  id  otel  dispatcher  idempotency
Layer 1:           kv(→codec)  metadata(→id)  prometheus(→otel)
Layer 2:           event(→codec, id, metadata, schema, snapshot, storage/memory, eventtest)
                   command(→codec, dispatcher, id, metadata, storage/memory)
                   query  (→codec, dispatcher, id, metadata, storage/memory)
                   snapshot(→codec, event, id, schema)
Layer 3:           projection(→event, id)  scheduling(→testutil)
Layer 4:           scenario(→event, id, projection)  graph(→event, id, projection)
                   listing(→event, id, storage/memory)
```

## Recommended Action Plan

### Phase 1 (immediate, low-risk)

Extract `retry/` and `idempotency/` as standalone repos. Both have zero CQRS coupling, single dependency on `go-error-family`, and only shallow importers. This reduces the repo from 52 → 50 modules and gives the ecosystem two genuinely useful standalone libraries.

### Phase 2 (optional, medium effort)

If the kv/ core abstraction has independent value, split `view_store.go` into a `viewstore/` module within this repo first, then extract the clean `kv/` core.

## The Shared Dependency: `go-error-family`

The common dependency across all extraction candidates is `github.com/larsartmann/go-error-family` — used for error classification (`NewRejection`, `NewConflict`, `NewInfrastructure`, `Wrap`, `IsRetryable`). This is already a standalone repo and serves as the shared foundation. If extracting `retry/` or `idempotency/`, this dependency travels with them cleanly.
