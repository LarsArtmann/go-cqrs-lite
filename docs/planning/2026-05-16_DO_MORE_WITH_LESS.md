# Do More With Less — Smarter Architecture

**Date:** 2026-05-16 | **Status:** Proposal | **Impact:** ~730 lines eliminated (~6% of codebase)

---

## Current State

```
Production LOC: 12,353
Test LOC:       25,438
Total:          37,791
```

After 62 sessions of incremental improvements, the codebase has accumulated structural duplication that no amount of per-file refactoring can fix. The opportunities below are not about cutting features — they're about finding the right abstraction so the same behavior is expressed once, not twice.

---

## 1. SQL Dialect Abstraction — ~250 lines saved

### Problem

`storage/` has **6 near-identical file pairs**:

| PostgreSQL | SQLite | Diff |
|-----------|--------|------|
| `event_store.go` (223 lines) | `sqlite_event_store.go` (238 lines) | `$1/$2` vs `?` |
| `outbox.go` (217 lines) | `sqlite_outbox.go` (~120 lines) | `$N` vs `?`, time format |
| `snapshot.go` (169 lines) | `sqlite_snapshot.go` (140 lines) | `$N` vs `?`, `EXCLUDED` vs `excluded` |
| `checkpoint.go` | `sqlite_checkpoint.go` | `$N` vs `?` |
| `transactional_store.go` | `sqlite_transactional_store.go` | `$N` vs `?` |
| `helpers.go` (scanEvent) | `sqlite_helpers.go` (sqliteScanEvent) | `time.Time` vs `string` parse |

Every method body is copy-pasted. The `Save/Load/Delete/Append` boilerplate (begin tx, defer rollback, call helpers, commit) is identical. The only semantic differences:

- SQL placeholder syntax: `$1`, `$2` vs `?`
- Timestamp handling: PostgreSQL stores `time.Time` natively, SQLite stores `string` (RFC3339Nano)
- DDL column types: `BYTEA/JSONB/TIMESTAMP` vs `BLOB/TEXT/TEXT`
- Upsert syntax: `ON CONFLICT ... DO UPDATE SET ... EXCLUDED.col` vs `... excluded.col` (case)

### Solution

A `Dialect` struct captures the differences:

```go
// storage/dialect.go
type Dialect struct {
    Placeholder func(n int) string
    FormatTime  func(time.Time) any
    ScanEvent   func(rows *sql.Rows) (event.Event, error)
    ScanSnapshot func(row *sql.Row, aggType event.AggregateType, aggID id.AggregateID) (*event.Snapshot, error)
    UpsertExcluded string // "EXCLUDED" or "excluded"
}

var PostgreSQL Dialect
var SQLite Dialect
```

Then **one** implementation per store type with a `dialect Dialect` field replaces both PostgreSQL and SQLite versions:

```go
// storage/event_store.go (single file, not two)
type SQLEventStore struct {
    db      *sql.DB
    dialect Dialect
}

func NewSQLEventStore(db *sql.DB) (*SQLEventStore, error) { ... }
func NewSQLiteEventStore(db *sql.DB) (*SQLEventStore, error) { ... }
```

### What changes

| Before (12 files) | After (7 files) |
|-------------------|-----------------|
| `event_store.go` + `sqlite_event_store.go` | `event_store.go` (single) |
| `outbox.go` + `sqlite_outbox.go` | `outbox.go` (single) |
| `snapshot.go` + `sqlite_snapshot.go` | `snapshot.go` (single) |
| `checkpoint.go` + `sqlite_checkpoint.go` | `checkpoint.go` (single) |
| `transactional_store.go` + `sqlite_transactional_store.go` | `transactional_store.go` (single) |
| `helpers.go` + `sqlite_helpers.go` | `helpers.go` + `dialect.go` (new) |

### Effort & Risk

- **Effort:** 4 hours
- **Risk:** MEDIUM — touches every file in storage/. But the interface (`event.Store`, `event.Bus`, etc.) doesn't change at all. Only the internal implementation.
- **Verification:** Existing tests cover all paths. Run tests before/after.

---

## 2. Merge aggregate/decider Persistence Logic — ~60 lines saved

### Problem

`core/aggregate/repository.go` (279 lines) and `core/decider/decider.go` (265 lines) share identical persistence orchestration:

```
aggregate.Save:                          decider.Execute:
  outbox? → TransactionalStore.SaveWithOutbox    identical branching
  else → store.Save + publisher.Publish          identical branching
  shouldSnapshot? → saveSnapshot                 identical

aggregate.Load:                          decider.Load:
  snapshotStore? → loadFromSnapshot              identical fallback
  else → loadFromStore                           identical

aggregate.Delete:                        decider.Delete:
  store.Delete + opError                         identical
```

The only real difference: aggregate uses a mutable `Root` interface (OO), decider uses a pure `Fold` function (functional). But the persistence orchestration around them is the same.

### Solution

Extract the shared persistence logic into `event.PersistenceHelper`:

```go
// core/event/persist_helper.go (already partially exists as publish_helper.go)
type PersistenceHelper struct {
    Store          Store
    Publisher      Publisher
    SnapshotStore  SnapshotStore
    Outbox         Outbox
    Codec          Codec
    SnapshotStrategy SnapshotStrategy
}

func (h *PersistenceHelper) Persist(ctx, aggType, aggID, events, expectedVersion) error { ... }
func (h *PersistenceHelper) LoadEvents(ctx, aggType, aggID) ([]Event, Version, error) { ... }
func (h *PersistenceHelper) Delete(ctx, aggType, aggID) error { ... }
```

Both `aggregate.Repository` and `decider.Repository` embed `PersistenceHelper` and delegate to it.

### Effort & Risk

- **Effort:** 2 hours
- **Risk:** LOW — internal refactoring only, no interface changes
- **Bonus:** Future repository patterns (Saga, Process Manager) get persistence for free

---

## 3. Unify CatalogMeta — ~30 lines saved

### Problem

Three nearly identical structs:

```go
// core/event/catalog.go
type CatalogMeta struct { Name, Version, Summary string }

// core/command/catalog.go
type CatalogMeta struct { Name, Version, Summary string }

// core/query/catalog.go
type CatalogMeta struct { Name, Version, Summary string }
```

Each package defines its own `Catalogable` interface returning its own `CatalogMeta`. The `catalog` package doesn't know about any of them — adapters translate.

### Solution

One canonical `catalog.Meta` in the catalog package:

```go
// catalog/meta.go
type Meta struct {
    Name    string
    Version string
    Summary string
}
```

Re-export from each core package for backward compatibility:

```go
// core/command/catalog.go
type CatalogMeta = catalog.Meta
```

### Effort & Risk

- **Effort:** 30 minutes
- **Risk:** LOW — type alias preserves backward compatibility

---

## 4. Delete `catalog/internal/cattest` — ~220 lines eliminated

### Problem

`cattest` is a test helper package with **0% coverage**. It contains:
- `assertions.go` (155 lines) — 13 assertion functions
- `builders.go` (220 lines) — 12 builder functions
- `catalog.go` (67 lines) — `BuildTestCatalog`

Only used by catalog's own tests. The `internal` package convention means external consumers can't use it anyway.

### Solution

Move the actually-used helpers into `catalog_test` or `catalog/asyncapi_test` directly. Delete the `internal/cattest` package entirely.

Alternatively, if the helpers are valuable for external consumers, make them public: `catalog/testutil`.

### Effort & Risk

- **Effort:** 1 hour
- **Risk:** LOW — only affects test code

---

## 5. Consolidate testhelpers Fake Boilerplate — ~80 lines saved

### Problem

Every fake (`FakeStore`, `FakeBus`, `FakeOutbox`, `FakeSnapshotStore`) has identical boilerplate:

```go
type FakeStore struct {
    mu     sync.RWMutex
    closed bool
    saveFn func(...) error
    // ...
}

func (f *FakeStore) Close() error {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.closed = true
    return nil
}
```

~15 lines of lifecycle boilerplate per fake × 4 fakes = ~60 lines. Plus the `SaveFn()` setter pattern repeats.

### Solution

A `fakeBase` struct:

```go
type fakeBase struct {
    mu     sync.RWMutex
    closed bool
}

func (f *fakeBase) Close() error {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.closed = true
    return nil
}

func (f *fakeBase) checkClosed(err error) error {
    f.mu.RLock()
    defer f.mu.RUnlock()
    if f.closed { return err }
    return nil
}
```

Each fake embeds `fakeBase` instead of duplicating the lifecycle code.

### Effort & Risk

- **Effort:** 1 hour
- **Risk:** LOW — internal test helpers only

---

## 6. Generic Catalog Adapters — ~50 lines saved

### Problem

`catalog/adapters/command.go`, `event.go`, `query.go` are structurally identical:

```go
// command.go
func (b *CatalogBuilder) AddCommand(serviceID string, cmd Catalogable) {
    meta := cmd.CatalogInfo()
    msg := buildCommandMessageFromReflect(string(cmd.Type()), meta, cmd)
    b.addMessageToService(serviceID, catalog.CommandMessage, msg)
}

// query.go
func (b *CatalogBuilder) AddQuery(serviceID string, qry Catalogable) {
    meta := qry.CatalogInfo()
    msg := buildQueryMessage(string(qry.Type()), meta, schema)
    b.addMessageToService(serviceID, catalog.QueryMessage, msg)
}
```

Same pattern, different kind and builder function.

### Solution

A single generic helper:

```go
func addFromReflect[T catalog.Catalogable](
    b *CatalogBuilder,
    serviceID string,
    v T,
    kind catalog.MessageKind,
    buildMsg func(string, catalog.Meta, *catalog.Schema) catalog.Message,
) {
    meta := v.CatalogInfo()
    schema := catalog.SchemaFromReflect(reflect.TypeOf(v).Elem())
    msg := buildMsg(string(v.Type()), meta, schema)
    b.addMessageToService(serviceID, kind, msg)
}
```

Then each `Add*` method becomes a one-liner delegation.

### Effort & Risk

- **Effort:** 1 hour
- **Risk:** LOW — internal adapter code

---

## Summary

| # | Opportunity | Lines Saved | Effort | Risk | Priority |
|---|------------|-------------|--------|------|----------|
| 1 | SQL Dialect abstraction | **~250** | 4h | MEDIUM | HIGH — biggest win |
| 2 | Merge aggregate/decider persist | **~60** | 2h | LOW | HIGH — architectural clarity |
| 3 | Unify CatalogMeta | **~30** | 30min | LOW | LOW — easy win |
| 4 | Delete cattest package | **~220** | 1h | LOW | MEDIUM — dead weight |
| 5 | Consolidate fake boilerplate | **~80** | 1h | LOW | LOW — nice cleanup |
| 6 | Generic catalog adapters | **~50** | 1h | LOW | LOW — nice cleanup |
| | **Total** | **~730** | ~10h | | |

### Recommended Execution Order

1. **#3 CatalogMeta** (30min warmup, zero risk)
2. **#4 Delete cattest** (1h, test-only, zero production risk)
3. **#5 Fake boilerplate** (1h, test-only)
4. **#6 Generic adapters** (1h, warmup for bigger refactors)
5. **#2 Merge persist logic** (2h, moderate complexity)
6. **#1 SQL Dialect** (4h, highest impact, requires all tests to pass)

### Anti-Goals (What NOT to Do)

- **Don't merge Pebble into the dialect abstraction** — Pebble is key-value, not SQL. It's fundamentally different.
- **Don't create a "generic Store[T]"** — The `event.Store` interface is already the right abstraction boundary.
- **Don't eliminate the OO aggregate package** — It stays for backward compatibility. Just share the persistence logic.
- **Don't over-abstract the catalog** — The current 3-file split (command/event/query) is readable. Only collapse the structural duplication.

---

_"Perfection is achieved not when there is nothing more to add, but when there is nothing left to take away." — Antoine de Saint-Exupéry_
