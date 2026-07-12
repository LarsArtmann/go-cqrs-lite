# Comprehensive Execution Plan: Saga Persistent Store & Outbox UX

**Date:** 2026-05-27
**Focus:** Close the two largest consumer-facing gaps in go-cqrs-lite
**Scope:** saga/ type model, storage/ SQL implementation, documentation

---

## 1. Executive Summary

From a consumer perspective, go-cqrs-lite has two critical gaps:

1. **No persistent saga store.** `saga/` exposes `saga.Store` but provides only `MemoryStore` (documented as "for testing"). The `Instance` type embeds function pointers (`Step.Action`, `Step.Compensate`) and an `error` interface, making it categorically unserializable. Every other major concern (events, snapshots, outbox, checkpoints) has persistent SQL/Pebble implementations.

2. **Outbox transaction footgun.** `SQLEventStore.Save` and `SQLOutbox.Append` each start their own transaction. A consumer wiring them independently does not get atomic save+outbox behavior. `SQLTransactionalStore.SaveWithOutbox` exists but is a separate, hard-to-discover type.

This plan fixes both, plus related polish (stale docs, dead code, defensive copies).

---

## 2. What Was Forgotten / What Could Be Better

### In the initial analysis:

- **`Instance.Err` is also unserializable** (interface type). Both `Steps` and `Err` must be split from the persistable surface.
- **`StatusStepCompleted` is dead code** — defined in `saga.go` but never assigned in `runner.go`.
- **`MemoryStore` lacks defensive copies** — stores raw pointers, so mutations to loaded instances leak back into the store.
- **No examples use saga** — neither `example/user/` nor `example/todo/` demonstrate saga usage, making discoverability poor.
- **`saga/go.mod` has a stale `replace` for `memory`** — saga does not actually import memory.
- **`docs/getting-started.md` references deleted `core/aggregate`** — replaced by `core/decider` in Session 99.

### In the broader codebase:

- **`Dialect` interface is missing `SagaSchema()`** — every other SQL store has a schema method.
- **`storage/` does not depend on `saga/`** — adding `SQLSagaStore` creates a new DAG edge (`storage → saga → core`), which is safe and correct.
- **No `io.Closer` on `saga.Store`** — inconsistent with every other store interface in `core/event/`.

---

## 3. Pareto Analysis

### 1% → 51% impact: Split `Instance` into `State` + runtime view

Without this type split, no persistent store is possible. It is the load-bearing foundation.

### 4% → 64% impact: Implement `SQLSagaStore` in `storage/`

Follows the exact `sqlBase` + `Dialect` + constructor + sqlmock + SQLite pattern already proven by `SQLSnapshotStore`, `SQLCheckpointStore`, etc. The consumer gets a production-grade saga store for PostgreSQL, SQLite, and Turso.

### 20% → 80% impact: Add `SQLBackend` unified constructor + fix docs

`SQLBackend` makes atomic save+outbox the default path. Fixing `getting-started.md` removes the #1 documentation trap for new consumers.

---

## 4. Type Model Redesign

### Before (unserializable)

```go
type Instance struct {
    ID          id.AggregateID
    SagaType    string
    Status      Status
    CurrentStep int
    Steps       []Step        // ← func fields, NOT serializable
    Err         error         // ← interface, NOT serializable
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Store interface {
    Save(ctx context.Context, instance *Instance) error
    Load(ctx context.Context, id id.AggregateID) (*Instance, error)
    LoadAllRunning(ctx context.Context) ([]*Instance, error)
}
```

### After (separation of concerns)

```go
// State is the fully serializable persistable state of a saga.
type State struct {
    ID          id.AggregateID
    SagaType    string
    Status      Status
    CurrentStep int
    ErrMsg      string        // ← replaces error interface
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Instance is the runtime view assembled by Runner.
type Instance struct {
    State
    Steps []Step              // ← hydrated from Definition registry
    Err   error               // ← rebuilt from ErrMsg
}

type Store interface {
    Save(ctx context.Context, state *State) error
    Load(ctx context.Context, id id.AggregateID) (*State, error)
    LoadAllRunning(ctx context.Context) ([]*State, error)
}
```

### Hydration in Runner

```go
func (r *Runner) hydrate(state *State) (*Instance, error) {
    def, ok := r.registry[state.SagaType]
    if !ok {
        return nil, fmt.Errorf("saga %s: %w", state.SagaType, ErrSagaNotRegistered)
    }
    inst := &Instance{State: *state, Steps: def.Steps()}
    if state.ErrMsg != "" {
        inst.Err = errors.New(state.ErrMsg)
    }
    return inst, nil
}
```

This makes impossible states unrepresentable: you cannot accidentally try to persist a function pointer.

---

## 5. Comprehensive Task List

| #   | Phase          | Task                                                                                    | Est. | Impact      | Effort | Why                                          |
| --- | -------------- | --------------------------------------------------------------------------------------- | ---- | ----------- | ------ | -------------------------------------------- |
| 1   | **Foundation** | Create `saga.State` type in new `state.go`                                              | 5m   | 🔴 Critical | Low    | Without this, no persistence possible        |
| 2   | **Foundation** | Redefine `saga.Instance` to embed `State` + `Steps` + `Err`                             | 5m   | 🔴 Critical | Low    | Runtime view separate from persistable state |
| 3   | **Foundation** | Update `saga.Store` interface to use `*State`                                           | 3m   | 🔴 Critical | Low    | Interface is the contract                    |
| 4   | **Foundation** | Update `MemoryStore` for new interface + add defensive copies                           | 8m   | 🔴 Critical | Med    | Tests and in-memory impl must work           |
| 5   | **Foundation** | Update `Runner` to hydrate `State` → `Instance` and dehydrate `Instance` → `State`      | 10m  | 🔴 Critical | Med    | Core orchestration logic                     |
| 6   | **Foundation** | Update all saga tests for new types (`errorStore`, field access, assertions)            | 10m  | 🔴 Critical | Med    | Must not break test suite                    |
| 7   | **Foundation** | Remove unused `StatusStepCompleted` constant                                            | 2m   | 🟡 Medium   | Low    | Dead code removal                            |
| 8   | **Foundation** | Run `go test ./saga/...` and fix any issues                                             | 5m   | 🔴 Critical | Low    | Verify foundation                            |
| 9   | **SQL Store**  | Add `SagaSchema()` to `Dialect` interface                                               | 3m   | 🔴 Critical | Low    | Required for SQL store                       |
| 10  | **SQL Store**  | Implement `PostgresDialect.SagaSchema()`                                                | 5m   | 🔴 Critical | Low    | PostgreSQL DDL                               |
| 11  | **SQL Store**  | Implement `SQLiteDialect.SagaSchema()`                                                  | 5m   | 🔴 Critical | Low    | SQLite DDL                                   |
| 12  | **SQL Store**  | Create `storage/saga_store.go` with `SQLSagaStore` type + `sqlBase` embed               | 5m   | 🔴 Critical | Low    | Follow existing pattern                      |
| 13  | **SQL Store**  | Implement `SQLSagaStore.Save()` with UPSERT                                             | 8m   | 🔴 Critical | Med    | Core persistence logic                       |
| 14  | **SQL Store**  | Implement `SQLSagaStore.Load()`                                                         | 8m   | 🔴 Critical | Med    | Core load logic                              |
| 15  | **SQL Store**  | Implement `SQLSagaStore.LoadAllRunning()`                                               | 8m   | 🔴 Critical | Med    | Core query logic                             |
| 16  | **SQL Store**  | Add constructors: `NewSQLSagaStore`, `NewSQLiteSagaStore`, `NewSQLSagaStoreWithDialect` | 5m   | 🔴 Critical | Low    | Consumer API                                 |
| 17  | **SQL Store**  | Add schema helpers: `SagaSchema()`, `SQLiteSagaSchema()`                                | 3m   | 🔴 Critical | Low    | Consistent with other stores                 |
| 18  | **SQL Store**  | Add `saga` dependency + replace directive to `storage/go.mod`                           | 3m   | 🔴 Critical | Low    | Module wiring                                |
| 19  | **SQL Store**  | Write `go-sqlmock` unit tests for `SQLSagaStore`                                        | 12m  | 🔴 Critical | Med    | Quality gate                                 |
| 20  | **SQL Store**  | Write SQLite integration tests for `SQLSagaStore`                                       | 12m  | 🔴 Critical | Med    | Quality gate                                 |
| 21  | **SQL Store**  | Run `go test ./storage/...` and fix issues                                              | 5m   | 🔴 Critical | Low    | Verify SQL store                             |
| 22  | **UX**         | Design `SQLBackend` type wrapping store + outbox                                        | 8m   | 🟡 Medium   | Med    | Unified constructor                          |
| 23  | **UX**         | Implement `NewSQLBackend()` + `TransactionalStore()` / `Outbox()` accessors             | 10m  | 🟡 Medium   | Med    | Consumer-friendly API                        |
| 24  | **UX**         | Write tests for `SQLBackend`                                                            | 8m   | 🟡 Medium   | Med    | Quality gate                                 |
| 25  | **Docs**       | Fix `docs/getting-started.md` stale `core/aggregate` → `core/decider`                   | 5m   | 🟡 Medium   | Low    | #1 doc trap for consumers                    |
| 26  | **Docs**       | Update `AGENTS.md` saga section with persistent store info                              | 5m   | 🟢 Low      | Low    | Memory maintenance                           |
| 27  | **Docs**       | Update `FEATURES.md` saga status from 🧪 to ✅/⚠️                                       | 3m   | 🟢 Low      | Low    | Honest inventory                             |
| 28  | **Docs**       | Update `TODO_LIST.md` — mark completed items                                            | 3m   | 🟢 Low      | Low    | Close the loop                               |
| 29  | **Verify**     | Run full test suite `go test ./saga/... ./storage/...`                                  | 5m   | 🔴 Critical | Low    | Cross-module verification                    |
| 30  | **Verify**     | Run lint / vet (`nix run .#lint` or `go vet`)                                           | 5m   | 🔴 Critical | Low    | Static analysis                              |
| 31  | **Verify**     | Check coverage for new files (target: >80%)                                             | 3m   | 🟡 Medium   | Low    | Coverage gate                                |
| 32  | **Verify**     | Git status, commit each phase, final push                                               | 5m   | 🔴 Critical | Low    | Delivery                                     |

**Total estimated time:** ~3.5 hours  
**Task count:** 32  
**Max task size:** 12 minutes  
**Breaking changes:** Yes (pre-v1.0, acceptable)

---

## 6. D2 Execution Graph

```d2
direction: down

Foundation: {
  label: |md
    **Phase 1: Foundation**
    Split Instance → State + runtime view
  |
  style.fill: "#ffcccc"

  t1: State type
  t2: Instance embed
  t3: Store interface
  t4: MemoryStore
  t5: Runner hydrate
  t6: Tests update
  t7: Dead code removal
  t8: Test run
}

SQLStore: {
  label: |md
    **Phase 2: SQL Saga Store**
    Persistent saga implementation
  |
  style.fill: "#ccffcc"

  t9: Dialect.SagaSchema
  t10: Postgres schema
  t11: SQLite schema
  t12: SQLSagaStore type
  t13: Save UPSERT
  t14: Load
  t15: LoadAllRunning
  t16: Constructors
  t17: Schema helpers
  t18: go.mod wiring
  t19: sqlmock tests
  t20: Integration tests
  t21: Test run
}

UX: {
  label: |md
    **Phase 3: Outbox UX**
    SQLBackend unified constructor
  |
  style.fill: "#ccccff"

  t22: Design SQLBackend
  t23: Implement SQLBackend
  t24: SQLBackend tests
}

Docs: {
  label: |md
    **Phase 4: Documentation**
    Fix stale refs, update status
  |
  style.fill: "#ffffcc"

  t25: Fix getting-started.md
  t26: Update AGENTS.md
  t27: Update FEATURES.md
  t28: Update TODO_LIST.md
}

Verify: {
  label: |md
    **Phase 5: Verification**
    Full test, lint, coverage, git
  |
  style.fill: "#ccffff"

  t29: Full test suite
  t30: Lint / vet
  t31: Coverage check
  t32: Git commit & push
}

Foundation -> SQLStore -> UX -> Docs -> Verify
```

---

## 7. Reuse Checklist

Before implementing anything from scratch, verify existing code fits:

| What we need         | Existing code that fits                         | Decision                                       |
| -------------------- | ----------------------------------------------- | ---------------------------------------------- |
| SQL store pattern    | `SQLSnapshotStore`, `SQLCheckpointStore`        | ✅ Reuse `sqlBase` + `Dialect` pattern         |
| Schema DDL pattern   | `EventSchema()`, `SnapshotSchema()`, etc.       | ✅ Reuse — add `SagaSchema()` to `Dialect`     |
| SQL constructors     | `NewSQLSnapshotStore`, `NewSQLiteSnapshotStore` | ✅ Reuse naming convention                     |
| SQL test pattern     | `snapshot_test.go`, `checkpoint_test.go`        | ✅ Reuse sqlmock + SQLite integration pattern  |
| Branded ID pattern   | `id.Of[T]`, `id.AggregateID`                    | ✅ Reuse — saga ID is already `id.AggregateID` |
| Error wrapping       | `fmt.Errorf` with `%w`                          | ✅ Reuse existing pattern                      |
| JSON serialization   | `encoding/json` (stdlib)                        | ✅ Reuse — `State` is simple enough for stdlib |
| Time formatting      | `Dialect.FormatTime` / `ParseTime`              | ✅ Reuse existing dialect time handling        |
| Transaction handling | `saveWithOutboxTx` pattern                      | ✅ Reuse existing tx helper pattern            |

**No new third-party libraries needed.** The existing `sqlBase` + `Dialect` + `go-sqlmock` + `modernc.org/sqlite` stack is sufficient.

---

## 8. Architecture Decisions

### ADR-1: `State` belongs in `saga/`, `SQLSagaStore` belongs in `storage/`

`saga/` is a pure orchestration module (depends only on `core`). `storage/` is the persistent implementation module. Adding `SQLSagaStore` to `storage/` requires `storage` → `saga` dependency. This is a DAG, no cycle. It follows the same pattern as `storage` implementing `core/event.Store`.

### ADR-2: `Store` interface omits `io.Closer`

The project has an open TODO to remove `io.Closer` from core interfaces (SESSION_60). `saga.Store` currently has no `Closer`. We keep it that way — the `*sql.DB` is borrowed, not owned, consistent with all other `storage/` stores.

### ADR-3: `ErrMsg string` replaces `Err error` in `State`

Error type information is lost on serialization. In the saga design, `Err` is for inspection only — `ExecuteStep` guards against re-executing failed sagas by checking `Status`, not `Err`. `ErrMsg` is sufficient.

### ADR-4: `SQLBackend` is additive, not replacing existing constructors

`NewSQLEventStore` and `NewSQLOutbox` remain for backward compatibility. `NewSQLBackend` is a convenience wrapper that returns the pre-wired atomic combination.

---

## 9. Risk Register

| Risk                                                                | Likelihood                         | Impact | Mitigation                                                            |
| ------------------------------------------------------------------- | ---------------------------------- | ------ | --------------------------------------------------------------------- |
| Breaking change to `saga.Store` affects external consumers          | Low (pre-v1.0, replace directives) | Medium | Document in CHANGELOG; project is pre-v1.0                            |
| `storage` → `saga` dependency bloats consumers who only want events | Low                                | Low    | Go modules prune unused dependencies at compile time                  |
| `Dialect` interface break affects custom dialect implementers       | Low                                | Medium | Add method with compile-time safety; implementers get clear error     |
| Test coverage drops below 80% for new files                         | Low                                | Medium | Target >90% with sqlmock + integration tests                          |
| `Runner.hydrate` fails if saga type not registered on restart       | Low                                | High   | Log error clearly; this is a code deployment issue, not a library bug |

---

## 10. Success Criteria

- [ ] `go test ./saga/...` passes with 0 failures
- [ ] `go test ./storage/...` passes with 0 failures
- [ ] `SQLSagaStore` coverage > 80%
- [ ] `saga` module tests updated for new `State`/`Instance` split
- [ ] `docs/getting-started.md` no longer references `core/aggregate`
- [ ] `TODO_LIST.md` updated with completed items
- [ ] No lint errors in modified files
- [ ] Git history shows clean, atomic commits per phase
