# WAL Unification: Commands + Events + PersistedQueries as One Log

> **Goal:** Eliminate the triplicated persistence layer for Commands, Events, and PersistedQueries. All three are append-only logs (WALs). `record.Record` already captures this — the storage layer needs to follow through.

## Problem

The codebase treats Commands, Events, and PersistedQueries as completely separate concepts at the persistence layer, despite all three being append-only logs with the same fundamental shape. This creates massive triplication:

| Layer | Lines | Triplication |
|-------|-------|-------------|
| Memory stores | 598 (126 + 309 + 163) | 3x complete implementations with identical lock/lifecycle/index patterns |
| SQL stores | ~900 | `JournalReader[T]` covers reads; insert/scan/save still duplicated 3x |
| System adapters | 593 (297 + 162 + 134) + 410 serial helpers | 3x adapter structs wrapping `metaengine.StreamLogBackend` |
| Store interfaces | ~150 | 3x parallel Sink/Source/Journal/Seekable hierarchies, 90% isomorphic |
| Metadata types | ~100 | `command.Metadata` and `query.Metadata` are **byte-for-byte identical** except type name |
| **Total** | **~2,750 lines** | **~1,800 lines are duplication** |

`record.Record` (ADR-0111) was extracted as "the shared structural base for Commands and Events" — but it's a one-way adapter (domain → Record) with no storage integration, no query support, and no way back.

## The Insight

```
Command    = append to log, read from log, dispatch to one handler
Event      = append to log, read from log, fan out to many handlers  
Query      = append to log, read from log, dispatch to one handler (return result)
```

The **persistence** shape is identical. The **dispatch** shape differs (one vs many vs return-value). The **domain semantics** differ (intent vs fact vs question). But the WAL is the same.

## What Should NOT Be Unified (Anti-Verschlimmbessern)

| Layer | Why it stays separate |
|-------|----------------------|
| **Domain types** (`Event`, `Command`, `Query`) | Different semantics: intent vs fact vs question. Different fields. Different lifecycles. |
| **Dispatch/handler** | Command = one handler, Event = fan-out pub/sub, Query = one handler + return value. Genuinely different. |
| **Bus interface** | `event.Bus` has pub/sub (`Publish` + `Subscribe` + `UsePublish`). Commands/queries don't. |
| **Event middleware** | `PublishMiddleware` vs `Middleware` split is justified (producer/consumer boundary). |
| **`event.Metadata`** | Has event-specific fields (`Source`, `IPAddress`, `UserAgent`, `Tombstone`, `Causation`) — genuinely different. |
| **Query `LoadQueries`** | Global-only (no per-stream scoping) — genuinely different from event/command stream reads. |

## What Should Be Unified

### 1. `command.Metadata` + `query.Metadata` → `metadata.Metadata[K]` (4% → 64%)

They are **literally identical**:
```go
// command/metadata.go AND query/query.go — same struct, different MetadataKey type
type Metadata struct {
    metadata.Tracing
    Custom map[MetadataKey]string `json:"custom,omitempty"`
}
```

**Fix:** Make both type aliases for a new `metadata.Metadata[K]`:
```go
// metadata/metadata.go
type Metadata[K ~string] struct {
    Tracing
    Custom map[K]string `json:"custom,omitempty"`
}
```

```go
// command/metadata.go
type Metadata = metadata.Metadata[MetadataKey]

// query/query.go  
type Metadata = metadata.Metadata[MetadataKey]
```

Existing `Clone()`/`Merge()`/`WithCustom()` methods move to `metadata.Metadata[K]` as generic methods. `event.Metadata` keeps its extra fields but embeds `metadata.Metadata[event.MetadataKey]` instead of re-declaring `Tracing` + `Custom`.

**Impact:** ~100 lines of duplicated methods eliminated. One canonical implementation. `event.Metadata` gets simpler.

**Risk:** Low. Type aliases are fully compatible. Existing code that constructs `command.Metadata{}` or `query.Metadata{}` compiles unchanged.

### 2. Generic `memory.LogStore[T, ID]` core (1% → 51%)

Three memory stores with identical patterns:
- `dispatcher.Lifecycle` embedding
- `sync.RWMutex` + `globalLog []T` + `streamIndex map[string][]int` + `idIndex map[ID]int`
- `withWriteLock` / `withReadLock[T]` helpers (duplicated 3x)
- `appendX` + `loadFiltered` methods
- Same `checkDuplicate` → `ErrDuplicateX` pattern

**Fix:** One generic core in `storage/memory/`:

```go
// storage/memory/log_store.go
type LogStore[T any, ID comparable] struct {
    dispatcher.Lifecycle
    mu         sync.RWMutex
    log        []T
    streamIdx  map[string][]int
    idIdx      map[ID]int
    getID      func(T) ID
    getRef     func(T) string
    dupErr     func(ID) error
}
```

Methods: `Append`, `AppendBatch`, `Load`, `ReadAll`, `ReadFrom`, `LoadBackwards` — all generic, all use the same lock/index pattern.

Existing `MemoryStore`, `MemoryCommandStore`, `MemoryQueryStore` become thin wrappers:
```go
type MemoryStore = logStoreWrapper[event.Event, id.EventID]  // or just use LogStore directly
```

**Impact:** ~598 lines → ~200 lines (generic core) + ~30 lines (3 thin wrappers) = ~230 lines. **-368 lines.**

**Risk:** Medium. Internal refactoring — public types stay the same (thin wrappers preserve API). Need to verify all interface assertions still hold.

### 3. `SinkTransform` + `SourceTransform` + `DecorateStore` (from previous plan)

Now applicable to **all three** store types via the generic core. Encryption, schema upcast, and any future transforms work on any `LogStore[T]`.

**Impact:** Eliminates `encryption/store.go` (241 lines) and `schema/versioned_source.go` (99 lines). **-340 lines.**

### 4. Add `query.AsRecord()` + `record.FromRecord` adapter (20% → 80%)

`record.Record` currently only has `event.AsRecord()` and `command.AsRecord()`. No query adapter. No way to convert back.

**Fix:**
```go
// query/asrecord.go
func AsRecord(q *PersistedQuery) record.Record

// record/record.go — add FromRecord
type Recordable interface {
    AsRecord() Record
}
```

This makes `record.Record` the true shared storage type. The metaengine already uses it — completing the adapter set makes the unification real.

**Impact:** Small code addition (~60 lines), but it completes the conceptual unification. The metaengine can now operate on all three entity types uniformly.

### 5. SQL store deduplication (remaining 20%)

`JournalReader[T]` already covers journal reads. Extend the pattern to cover inserts:

```go
// storage/sql/inserter.go
type Inserter[T any] struct {
    DB          *sql.DB
    Dialect     Dialect
    Table       string
    Columns     string
    Values      func(T) []any
    BatchSize   int
}
```

`Save` and `AppendBatch` become generic methods on `Inserter[T]`. Each SQL store shrinks to: constructor + `Scan` func + `Values` func.

**Impact:** ~300 lines of duplicated insert/scan code eliminated across 3 SQL stores.

### 6. System adapter deduplication (remaining 20%)

`EventAdapter`, `CommandAdapter`, `QueryAdapter` share the same `metaengine.StreamLogBackend` wrapping pattern. Extract a generic core:

```go
// system/adapter_core.go
type AdapterCore[T any] struct {
    backend    metaengine.StreamLogBackend
    collection string
    serialize  func(T) ([]byte, error)
    // ...
}
```

Each adapter embeds `AdapterCore[T]` and adds only its type-specific methods.

**Impact:** ~300 lines eliminated from system adapters.

## Pareto Breakdown

### 1% → 51%
Generic `memory.LogStore[T, ID]` core. Rewrite 3 memory stores as thin wrappers. **-368 lines.**

### 4% → 64%
Unify `command.Metadata` + `query.Metadata` → `metadata.Metadata[K]`. **-100 lines.** Removes a conceptual split.

### 20% → 80%
- `SinkTransform`/`SourceTransform` + `DecorateStore` (from previous plan). **-340 lines.**
- Add `query.AsRecord()` + complete `record.Record` adapter set. **+60 lines, completes unification.**

### Remaining 20% (to 100%)
- SQL store deduplication via `Inserter[T]`. **-300 lines.**
- System adapter deduplication via `AdapterCore[T]`. **-300 lines.**
- Tests, API stability, docs, full verify.

## Expected Outcome

| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| Memory stores | 598 lines | ~230 lines | -368 |
| Metadata types | ~100 lines | ~35 lines | -65 |
| Store wrappers (encryption + schema) | 340 lines | ~50 lines | -290 |
| SQL stores (insert/scan dup) | ~900 lines | ~600 lines | -300 |
| System adapters | 593 lines | ~300 lines | -293 |
| New shared code | 0 | ~300 lines | +300 |
| **Total** | **~2,750 lines** | **~1,485 lines** | **-1,265 lines** |

Net reduction: **~1,265 lines**. One generic `LogStore[T, ID]` core replaces 3 independent store hierarchies.

## Comprehensive Task List (sorted by impact/effort)

| # | Task | Impact | Effort | Phase |
|---|------|--------|--------|-------|
| 1 | Add `metadata.Metadata[K]` generic type | Critical | Low | Metadata |
| 2 | Make `command.Metadata` a type alias | High | Low | Metadata |
| 3 | Make `query.Metadata` a type alias | High | Low | Metadata |
| 4 | Update `event.Metadata` to embed `Metadata[K]` | Medium | Medium | Metadata |
| 5 | Run metadata-dependent module tests | High | Low | Metadata |
| 6 | Add `SinkTransform`/`SourceTransform` to `event/` | Critical | Low | Store MW |
| 7 | Implement `DecorateStore` (core + optional interfaces) | Critical | Medium | Store MW |
| 8 | Write `DecorateStore` tests | High | Medium | Store MW |
| 9 | Create `encryption.EncryptSinkTransform` + `DecryptSourceTransform` | Critical | Low | Store MW |
| 10 | Rewrite `NewEncryptedStore` to use `DecorateStore` | High | Low | Store MW |
| 11 | Delete old `encryptedStore` struct | High | Low | Store MW |
| 12 | Update encryption tests | High | Medium | Store MW |
| 13 | Create `schema.UpcastSourceTransform` | High | Low | Store MW |
| 14 | Rewrite `NewVersionedStore` to use `DecorateStore` | Medium | Low | Store MW |
| 15 | Update schema tests | Medium | Low | Store MW |
| 16 | Add `RejectingPublishMiddleware` + `RejectingMiddleware` to `event/` | Medium | Low | Nil-guard |
| 17 | Update `signing` + `encryption` to use shared nil-guards | Medium | Low | Nil-guard |
| 18 | Add `query.AsRecord()` adapter | Medium | Low | Record |
| 19 | Design generic `memory.LogStore[T, ID]` struct | Critical | Medium | Memory |
| 20 | Implement `LogStore` core methods (append, load, readall, readfrom) | Critical | High | Memory |
| 21 | Implement `LogStore` optional interfaces (seekable, backwards, closer) | High | Medium | Memory |
| 22 | Rewrite `MemoryStore` as thin wrapper over `LogStore` | High | Medium | Memory |
| 23 | Rewrite `MemoryCommandStore` as thin wrapper | High | Medium | Memory |
| 24 | Rewrite `MemoryQueryStore` as thin wrapper | High | Medium | Memory |
| 25 | Run memory store tests | High | Medium | Memory |
| 26 | Add `storage/sql.Inserter[T]` generic | High | Medium | SQL |
| 27 | Rewrite `SQLCommandStore` insert/scan using `Inserter[T]` | Medium | Medium | SQL |
| 28 | Rewrite `SQLQueryStore` insert/scan using `Inserter[T]` | Medium | Medium | SQL |
| 29 | Rewrite SQL event store insert/scan using `Inserter[T]` | Medium | Medium | SQL |
| 30 | Run SQL store tests | High | Low | SQL |
| 31 | Extract `system.AdapterCore[T]` generic | Medium | Medium | System |
| 32 | Rewrite `EventAdapter` to embed `AdapterCore[T]` | Low | Medium | System |
| 33 | Rewrite `CommandAdapter` to embed `AdapterCore[T]` | Low | Medium | System |
| 34 | Rewrite `QueryAdapter` to embed `AdapterCore[T]` | Low | Medium | System |
| 35 | Run system tests | High | Low | System |
| 36 | Regenerate API stability golden | High | Low | Verify |
| 37 | Run API stability meta-tests | High | Low | Verify |
| 38 | Update AGENTS.md | Medium | Low | Docs |
| 39 | Run doc-check | Medium | Low | Verify |
| 40 | Run `nix run .#verify` | Critical | Medium | Verify |
| 41 | Fix any issues from verify | Variable | Variable | Verify |

## Detailed Task Breakdown (each max 12 min)

| # | Task | Est | Depends on |
|---|------|-----|-----------|
| 1 | Add `Metadata[K ~string]` struct to `metadata/metadata.go` | 8 min | — |
| 2 | Add `Clone()`, `Merge()`, `WithCustom()` methods to `Metadata[K]` | 10 min | 1 |
| 3 | Make `command.Metadata` a type alias for `metadata.Metadata[command.MetadataKey]` | 5 min | 2 |
| 4 | Make `query.Metadata` a type alias for `metadata.Metadata[query.MetadataKey]` | 5 min | 2 |
| 5 | Remove old `command.Metadata` methods (now on `Metadata[K]`) | 5 min | 3 |
| 6 | Remove old `query.Metadata` methods (now on `Metadata[K]`) | 5 min | 4 |
| 7 | Update `event.Metadata` to embed `metadata.Metadata[event.MetadataKey]` | 10 min | 2 |
| 8 | Remove duplicated `Clone`/`Merge`/`WithCustom` from `event.Metadata` | 5 min | 7 |
| 9 | Run command tests: `cd command && GOWORK=off go test ./... -count=1` | 3 min | 5 |
| 10 | Run query tests: `cd query && GOWORK=off go test ./... -count=1` | 3 min | 6 |
| 11 | Run event tests: `cd event && GOWORK=off go test ./... -count=1` | 3 min | 8 |
| 12 | Run metadata tests: `cd metadata && GOWORK=off go test ./... -count=1` | 3 min | 2 |
| 13 | Add `SinkTransform`/`SourceTransform` type defs to `event/store_middleware.go` | 3 min | — |
| 14 | Add `DecorateStore` + `decoratedStore` struct to `event/store_middleware.go` | 5 min | 13 |
| 15 | Implement `decoratedStore.Save` + `AppendBatch` | 5 min | 14 |
| 16 | Implement `decoratedStore.Load` + `LoadFromVersion` + `LoadToVersion` + `LoadToTimestamp` | 8 min | 14 |
| 17 | Implement `decoratedStore.ReadAll` + `ReadFrom` + `LoadBackwards` + `Close` | 8 min | 14 |
| 18 | Handle nil transforms as pass-through | 5 min | 14 |
| 19 | Write `TestDecorateStore_PassThrough` | 8 min | 18 |
| 20 | Write `TestDecorateStore_SinkTransform` | 8 min | 18 |
| 21 | Write `TestDecorateStore_SourceTransform` | 8 min | 18 |
| 22 | Write `TestDecorateStore_OptionalInterfaces` | 10 min | 18 |
| 23 | Write `TestDecorateStore_NilInner` | 5 min | 14 |
| 24 | Run event store middleware tests | 3 min | 19-23 |
| 25 | Create `encryption.EncryptSinkTransform` | 8 min | 13 |
| 26 | Create `encryption.DecryptSourceTransform` | 8 min | 13 |
| 27 | Rewrite `NewEncryptedStore` to call `event.DecorateStore` | 5 min | 25, 26 |
| 28 | Delete old `encryptedStore` struct + methods | 5 min | 27 |
| 29 | Update encryption store tests | 10 min | 28 |
| 30 | Run encryption tests | 3 min | 29 |
| 31 | Create `schema.UpcastSourceTransform` | 8 min | 13 |
| 32 | Rewrite `NewVersionedStore` to call `event.DecorateStore` | 5 min | 31 |
| 33 | Delete old `VersionedStore` struct | 3 min | 32 |
| 34 | Update schema tests | 8 min | 33 |
| 35 | Run schema tests | 3 min | 34 |
| 36 | Add `RejectingPublishMiddleware` to `event/middleware.go` | 5 min | — |
| 37 | Add `RejectingMiddleware` to `event/middleware.go` | 5 min | — |
| 38 | Update `signing` to use `event` nil-guard helpers | 5 min | 36, 37 |
| 39 | Remove signing's `Rejecting*` helpers | 5 min | 38 |
| 40 | Update `encryption` to use `event` nil-guard helpers | 5 min | 36, 37 |
| 41 | Remove encryption's `rejecting*` helpers | 5 min | 40 |
| 42 | Run signing tests | 3 min | 39 |
| 43 | Run encryption tests again | 3 min | 41 |
| 44 | Add `query.AsRecord()` function | 8 min | — |
| 45 | Design `memory.LogStore[T, ID]` struct + constructor | 10 min | — |
| 46 | Implement `LogStore.Append` + `AppendBatch` | 10 min | 45 |
| 47 | Implement `LogStore.Load` + `LoadFromVersion` | 10 min | 45 |
| 48 | Implement `LogStore.ReadAll` + `ReadFrom` | 10 min | 45 |
| 49 | Implement `LogStore.LoadBackwards` + `Close` | 8 min | 45 |
| 50 | Write `LogStore` unit tests | 10 min | 46-49 |
| 51 | Rewrite `MemoryStore` (events) as wrapper | 10 min | 50 |
| 52 | Rewrite `MemoryCommandStore` as wrapper | 10 min | 50 |
| 53 | Rewrite `MemoryQueryStore` as wrapper | 10 min | 50 |
| 54 | Update memory store tests | 10 min | 51-53 |
| 55 | Run memory store tests | 3 min | 54 |
| 56 | Add `sql.Inserter[T]` generic struct | 10 min | — |
| 57 | Implement `Inserter.Save` + `Inserter.AppendBatch` | 10 min | 56 |
| 58 | Rewrite `SQLCommandStore` insert using `Inserter[T]` | 10 min | 57 |
| 59 | Rewrite `SQLQueryStore` insert using `Inserter[T]` | 10 min | 57 |
| 60 | Rewrite SQL event store insert using `Inserter[T]` | 10 min | 57 |
| 61 | Run SQL store tests | 5 min | 58-60 |
| 62 | Extract `system.AdapterCore[T]` | 10 min | — |
| 63 | Rewrite `CommandAdapter` to embed `AdapterCore[T]` | 10 min | 62 |
| 64 | Rewrite `QueryAdapter` to embed `AdapterCore[T]` | 10 min | 62 |
| 65 | Rewrite `EventAdapter` to embed `AdapterCore[T]` | 10 min | 62 |
| 66 | Run system tests | 5 min | 63-65 |
| 67 | Regenerate API stability golden | 5 min | all |
| 68 | Run API stability meta-tests | 3 min | 67 |
| 69 | Update AGENTS.md | 5 min | all |
| 70 | Run doc-check | 3 min | 69 |
| 71 | Run `nix run .#verify` | 10 min | all |
| 72 | Fix any issues from verify | Variable | — |

## Execution Graph

```mermaid
graph TD
    subgraph "Phase 1: Metadata Unification"
        M1[Add metadata.Metadata[K] generic]
        M2[Alias command.Metadata → metadata.Metadata[K]]
        M3[Alias query.Metadata → metadata.Metadata[K]]
        M4[Update event.Metadata to embed Metadata[K]]
        M5[Run all metadata-dependent tests]
        M1 --> M2 --> M5
        M1 --> M3 --> M5
        M1 --> M4 --> M5
    end

    subgraph "Phase 2: Store Middleware + Transform"
        S1[Add SinkTransform/SourceTransform types]
        S2[Implement DecorateStore]
        S3[Write DecorateStore tests]
        S4[Create encryption transforms]
        S5[Rewrite NewEncryptedStore]
        S6[Delete old encryptedStore]
        S7[Create schema UpcastSourceTransform]
        S8[Rewrite NewVersionedStore]
        S9[Delete old VersionedStore]
        S10[Add Rejecting* to event/]
        S11[Update signing + encryption nil-guards]
        S1 --> S2 --> S3
        S2 --> S4 --> S5 --> S6
        S2 --> S7 --> S8 --> S9
        S10 --> S11
    end

    subgraph "Phase 3: Record Unification"
        R1[Add query.AsRecord adapter]
    end

    subgraph "Phase 4: Memory Store Unification"
        L1[Design LogStore[T, ID] generic]
        L2[Implement core methods]
        L3[Implement optional interfaces]
        L4[Write LogStore tests]
        L5[Rewrite MemoryStore wrapper]
        L6[Rewrite MemoryCommandStore wrapper]
        L7[Rewrite MemoryQueryStore wrapper]
        L8[Update + run memory tests]
        L1 --> L2 --> L3 --> L4
        L4 --> L5 --> L8
        L4 --> L6 --> L8
        L4 --> L7 --> L8
    end

    subgraph "Phase 5: SQL Store Dedup"
        Q1[Add sql.Inserter[T] generic]
        Q2[Rewrite SQLCommandStore]
        Q3[Rewrite SQLQueryStore]
        Q4[Rewrite SQL event store]
        Q5[Run SQL tests]
        Q1 --> Q2 --> Q5
        Q1 --> Q3 --> Q5
        Q1 --> Q4 --> Q5
    end

    subgraph "Phase 6: System Adapter Dedup"
        A1[Extract AdapterCore[T]]
        A2[Rewrite CommandAdapter]
        A3[Rewrite QueryAdapter]
        A4[Rewrite EventAdapter]
        A5[Run system tests]
        A1 --> A2 --> A5
        A1 --> A3 --> A5
        A1 --> A4 --> A5
    end

    subgraph "Phase 7: Verify + Ship"
        V1[Regen API stability]
        V2[Run API meta-tests]
        V3[Update AGENTS.md]
        V4[Run doc-check]
        V5[Run nix run .#verify]
        V6[Fix issues]
        V1 --> V2
        V3 --> V4
        V2 --> V5
        V4 --> V5
        V5 --> V6
    end

    M5 --> S1
    S3 --> V1
    S6 --> V1
    S9 --> V1
    S11 --> V1
    R1 --> V1
    L8 --> V1
    Q5 --> V1
    A5 --> V1
```

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| Metadata alias changes break consumers | Low | Type aliases are fully compatible in Go |
| LogStore generic breaks interface assertions | Medium | Thin wrappers preserve exact interface conformance |
| DecorateStore doesn't cover edge case | Medium | Comprehensive tests for all optional interfaces |
| SQL Inserter[T] dialect differences | Medium | Start with SQLite, verify PG/MySQL separately |
| System adapter refactor breaks metaengine integration | Medium | Run full system + integration tests |
| API stability golden needs regen | Expected | Part of the plan |

## What This Plan Does NOT Do

- **Does not merge `Event`/`Command`/`Query` domain types** — different semantics
- **Does not merge dispatch/handler layers** — genuinely different (one vs many vs return)
- **Does not merge `event.Bus` with command/query dispatchers** — pub/sub vs dispatch-to-one
- **Does not remove `PublishMiddleware`** — producer/consumer split is real
- **Does not touch `event.Metadata` extra fields** — they're event-specific
- **Does not merge query's global-only reads** — genuinely different from stream-scoped
- **Does not break any public API** — all changes are internal refactoring or type aliases
