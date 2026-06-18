# Bundle Layer — Comprehensive Execution Plan

> **Date:** 2026-06-18 | **Status:** Draft for execution
>
> Every task is designed to be ≤12 minutes of focused work.
> Sorted by impact/effort ratio (highest first within each phase).

## What Changed After Deep Research

| Discovery | Impact on Plan |
|---|---|
| `kv.Store` already IS `readmodel.Backend` | readmodel module becomes a thin typed wrapper, not a new interface |
| `pebble.KVAdapter` already implements `kv.Store` | Pebble read models need zero new code |
| `kv.MemStore` already exists | Memory read models need zero new code |
| All 6 `memory/*` stores are complete | Memory preset is assembly, not implementation |
| Option pattern is `func(*T)` without error | Bundle options must match this convention |
| `snapshot.State []byte` is untyped | `TypedSnapshot[State]` eliminates the biggest type safety hole |
| Bundle should have typed accessors | `bundle.Repository[State]()` and `bundle.ReadModel[T,K]()` |

---

## Phase 1: readmodel Module (Foundation — Unblocks Everything)

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 1.1 | Create `readmodel/` module skeleton (`go.mod`, `doc.go`) | High | XS | `readmodel/go.mod`, `readmodel/doc.go` |
| 1.2 | Define `readmodel.Backend = kv.Store` type alias | High | XS | `readmodel/backend.go` |
| 1.3 | Define `readmodel.Store[T any, K ~string]` generic wrapper struct | High | S | `readmodel/store.go` |
| 1.4 | Implement `Store[T,K].Get(ctx, id K) (*T, error)` with codec decode | High | S | `readmodel/store.go` |
| 1.5 | Implement `Store[T,K].Set(ctx, id K, val *T) error` with codec encode | High | S | `readmodel/store.go` |
| 1.6 | Implement `Store[T,K].Delete(ctx, id K) error` | Medium | XS | `readmodel/store.go` |
| 1.7 | Implement `Store[T,K].Scan(ctx, prefix []byte) ([]*T, error)` helper | Medium | S | `readmodel/store.go` |
| 1.8 | Define `readmodel.Option` and `readmodel.New[T,K](backend, opts...)` constructor | High | S | `readmodel/store.go` |
| 1.9 | Add `WithCodec(c codec.Codec)` option | Medium | XS | `readmodel/options.go` |
| 1.10 | Add `WithKeyPrefix(prefix string)` option for namespacing | Medium | XS | `readmodel/options.go` |
| 1.11 | Add `WithKeyTransform(fn func(K) []byte)` option for custom key encoding | Low | XS | `readmodel/options.go` |
| 1.12 | Write unit tests: `Store[T,K]` Get/Set/Delete roundtrip with memory backend | High | S | `readmodel/store_test.go` |
| 1.13 | Write unit tests: Scan with prefix filtering | Medium | S | `readmodel/store_test.go` |
| 1.14 | Write example test showing `Store[Todo, TodoID]` usage | Medium | S | `readmodel/example_test.go` |
| 1.15 | Add `readmodel/errors.go` with `ErrNotFound` (alias to `kv.ErrNotFound`) | Low | XS | `readmodel/errors.go` |

---

## Phase 2: Type Safety Improvements (Independent, High Value)

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 2.1 | Define `snapshot.TypedSnapshot[State any]` struct | High | XS | `snapshot/typed.go` |
| 2.2 | Define `snapshot.TypedSnapshotSink[State]` and `TypedSnapshotSource[State]` interfaces | High | S | `snapshot/typed.go` |
| 2.3 | Implement typed↔untyped adapter: wraps `SnapshotStore` + `codec.Codec` | High | M | `snapshot/typed_adapter.go` |
| 2.4 | Write tests: `TypedSnapshot[State]` roundtrip | High | S | `snapshot/typed_test.go` |
| 2.5 | Define `command.PersistedCommand[T any]` with typed `Payload() T` | Medium | S | `command/typed.go` |
| 2.6 | Define `query.PersistedQuery[T any]` with typed `Payload() T` | Medium | S | `query/typed.go` |
| 2.7 | Write tests for typed command/query payloads | Medium | S | `command/typed_test.go`, `query/typed_test.go` |

---

## Phase 3: Bundle Core (The Composition Root)

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 3.1 | Create `stack/` module skeleton (`go.mod`, `doc.go`) | High | XS | `stack/go.mod`, `stack/doc.go` |
| 3.2 | Define `Bundle` struct with segregated interface fields | High | S | `stack/bundle.go` |
| 3.3 | Define `Option = func(*Bundle)` (no error, matching codebase convention) | High | XS | `stack/bundle.go` |
| 3.4 | Implement `New(opts ...Option) (*Bundle, error)` with validation | High | S | `stack/bundle.go` |
| 3.5 | Implement closer tracking: `registerCloser(io.Closer)` internal method | High | XS | `stack/bundle.go` |
| 3.6 | Implement `Close() error` with pointer-deduplicated closer list | High | S | `stack/bundle.go` |
| 3.7 | Implement partial-failure rollback in `New()` | High | M | `stack/bundle.go` |
| 3.8 | Define all `With*` option functions: `WithEventSink`, `WithEventSource`, etc. | High | M | `stack/options.go` |
| 3.9 | Define `WithEventStore(event.Store)` convenience (sets both Sink+Source+Journal if asserted) | Medium | S | `stack/options.go` |
| 3.10 | Define `WithBus(event.Bus)` convenience (sets Publisher+Subscriber) | Medium | XS | `stack/options.go` |
| 3.11 | Define `WithReadModels(kv.Store)` option | Medium | XS | `stack/options.go` |
| 3.12 | Add validation: `validate()` checks required fields aren't nil | Medium | S | `stack/bundle.go` |
| 3.13 | Write tests: Bundle construction and field assignment | High | S | `stack/bundle_test.go` |
| 3.14 | Write tests: Close() deduplicates shared resources | High | M | `stack/bundle_test.go` |
| 3.15 | Write tests: Partial failure closes already-opened resources | High | M | `stack/bundle_test.go` |

---

## Phase 4: Bundle Typed Accessors (Type Safety Layer)

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 4.1 | Implement `bundle.Repository[State](d decider.Decider[State], opts...)` accessor | High | M | `stack/accessors.go` |
| 4.2 | Implement `bundle.ReadModel[T any, K ~string](codec) *readmodel.Store[T,K]` accessor | High | S | `stack/accessors.go` |
| 4.3 | Implement `bundle.ProjectionRunner(opts...) *projection.Runner` helper | Medium | S | `stack/accessors.go` |
| 4.4 | Write tests: typed accessors produce correctly-wired types | High | S | `stack/accessors_test.go` |
| 4.5 | Write example: using `bundle.Repository[State]()` in app code | Medium | S | `stack/example_test.go` |

---

## Phase 5: Memory Preset (First Working Preset)

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 5.1 | Create `stack/memory/` sub-package skeleton | High | XS | `stack/memory/go.mod`, `stack/memory/doc.go` |
| 5.2 | Implement `memory.New() (*stack.Bundle, error)` — assembles all memory stores | High | S | `stack/memory/preset.go` |
| 5.3 | Implement `memory.Bus() stack.Option` — returns WithBus option | Medium | XS | `stack/memory/preset.go` |
| 5.4 | Implement `memory.Checkpoints() stack.Option` | Low | XS | `stack/memory/preset.go` |
| 5.5 | Write tests: memory preset produces a fully-wired Bundle | High | S | `stack/memory/preset_test.go` |
| 5.6 | Write integration test: memory preset end-to-end event roundtrip | High | M | `stack/memory/integration_test.go` |

---

## Phase 6: SQLite Preset (Most Common Deployment)

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 6.1 | Create `stack/sqlite/` sub-package skeleton | High | XS | `stack/sqlite/go.mod`, `stack/sqlite/doc.go` |
| 6.2 | Implement `sqlite.New(dir string, opts ...) (*stack.Bundle, error)` | High | M | `stack/sqlite/preset.go` |
| 6.3 | Implement `sqlite.AppendLog(dir string, opts ...) stack.Option` | High | M | `stack/sqlite/bundles.go` |
| 6.4 | Implement `sqlite.Views(dir string, opts ...) stack.Option` | High | M | `stack/sqlite/bundles.go` |
| 6.5 | Implement `sqlite.ForSequentialIO() Option` tuning profile | Medium | S | `stack/sqlite/tuning.go` |
| 6.6 | Implement `sqlite.ForRandomIO() Option` tuning profile | Medium | S | `stack/sqlite/tuning.go` |
| 6.7 | Implement `sqlite.WithAutoMigrate() Option` (calls InitSchema) | Medium | S | `stack/sqlite/options.go` |
| 6.8 | Implement `sqlite.WithWAL() Option` (enables WAL mode) | Low | XS | `stack/sqlite/options.go` |
| 6.9 | Write tests: SQLite preset produces working Bundle | High | M | `stack/sqlite/preset_test.go` |
| 6.10 | Write tests: AppendLog + Views split topology | High | M | `stack/sqlite/bundles_test.go` |
| 6.11 | Write tests: tuning profiles apply correct pragmas | Medium | S | `stack/sqlite/tuning_test.go` |

---

## Phase 7: Fill Pebble Gaps (Backend Completeness)

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 7.1 | Implement `pebble.CommandStore` (append-only, like EventStore) | Medium | M | `pebble/command_store.go` |
| 7.2 | Implement `pebble.QueryStore` (append-only, like CommandStore) | Medium | M | `pebble/query_store.go` |
| 7.3 | Add `pebble.Backend.CommandStore()` accessor | Medium | XS | `pebble/backend.go` |
| 7.4 | Add `pebble.Backend.QueryStore()` accessor | Medium | XS | `pebble/backend.go` |
| 7.5 | Add `pebble.Backend.ReadModels() kv.Store` accessor (returns KVAdapter with WithBorrowedDB) | High | XS | `pebble/backend.go` |
| 7.6 | Write tests: Pebble command store CRUD | Medium | S | `pebble/command_store_test.go` |
| 7.7 | Write tests: Pebble query store CRUD | Medium | S | `pebble/query_store_test.go` |

---

## Phase 8: Pebble Preset

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 8.1 | Create `stack/pebble/` sub-package skeleton | Medium | XS | `stack/pebble/go.mod`, `stack/pebble/doc.go` |
| 8.2 | Implement `pebble.New(dir string, opts ...) (*stack.Bundle, error)` | Medium | S | `stack/pebble/preset.go` |
| 8.3 | Write tests: Pebble preset produces working Bundle | Medium | S | `stack/pebble/preset_test.go` |

---

## Phase 9: Postgres Preset

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 9.1 | Create `stack/postgres/` sub-package skeleton | Medium | XS | `stack/postgres/go.mod`, `stack/postgres/doc.go` |
| 9.2 | Implement `postgres.New(dsn string, opts ...) (*stack.Bundle, error)` | Medium | M | `stack/postgres/preset.go` |
| 9.3 | Write tests: Postgres preset produces working Bundle (integration) | Medium | M | `stack/postgres/preset_test.go` |

---

## Phase 10: Rewrite Example

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 10.1 | Port `example/todo/cmd/api/main.go` to use `sqlite.New()` | High | M | `example/todo/cmd/api/main.go` |
| 10.2 | Replace hand-rolled `storage/PebbleStore` with `readmodel.Store[T,K]` | High | M | `example/todo/storage/` |
| 10.3 | Replace hand-rolled `storage/MemoryStore` with `readmodel.Store[T,K]` over `kv.MemStore` | High | S | `example/todo/storage/` |
| 10.4 | Verify example compiles and tests pass | High | S | — |
| 10.5 | Add split-topology example (2 SQLite DBs) | Medium | M | `example/todo/cmd/api/main_split.go` |
| 10.6 | Measure and document line-count reduction | Medium | XS | `example/todo/README.md` |

---

## Phase 11: Benchmarks & Contract Tests

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 11.1 | Create `stack/contract/` — contract test suite parameterised by Bundle | High | M | `stack/contract/contract_test.go` |
| 11.2 | Contract test: EventSink.Save + EventSource.Load roundtrip | High | S | `stack/contract/events.go` |
| 11.3 | Contract test: Journal.ReadAll ordering | Medium | S | `stack/contract/journal.go` |
| 11.4 | Contract test: ReadModel Get/Set/Delete/Scan | High | S | `stack/contract/readmodels.go` |
| 11.5 | Contract test: Close() releases all resources | Medium | S | `stack/contract/lifecycle.go` |
| 11.6 | Create `stack/bench/` — benchmark suite parameterised by Bundle | Medium | M | `stack/bench/bench_test.go` |
| 11.7 | Benchmark: EventSave (ns/op, allocs/op) | Medium | S | `stack/bench/events.go` |
| 11.8 | Benchmark: ReadModelGet (ns/op, allocs/op) | Medium | S | `stack/bench/readmodels.go` |
| 11.9 | Zero-overhead test: Bundle field access vs direct store | High | S | `stack/bench/zero_overhead_test.go` |

---

## Phase 12: Documentation

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 12.1 | Write `docs/PRESETS.md` — preset catalog with recommendation matrix | High | M | `docs/PRESETS.md` |
| 12.2 | Update `docs/STORAGE_GUIDE.md` to lead with presets | Medium | S | `docs/STORAGE_GUIDE.md` |
| 12.3 | Update `SKILL.md` with Bundle composition recipes | Medium | S | `SKILL.md` |
| 12.4 | Update `AGENTS.md` with stack module in module list | Medium | XS | `AGENTS.md` |
| 12.5 | Write `stack/README.md` | Medium | S | `stack/README.md` |
| 12.6 | Write `readmodel/README.md` | Medium | S | `readmodel/README.md` |
| 12.7 | Update `FEATURES.md` with Bundle layer features | Low | XS | `FEATURES.md` |
| 12.8 | Update proposal HTML with kv.Store discovery and typed accessors | Medium | M | `docs/brainstorming/2026-06-18_stack-layer-proposal.html` |

---

## Phase 13: CI & Quality Gates

| # | Task | Impact | Effort | Files |
|---|---|---|---|---|
| 13.1 | Add stack modules to `flake.nix` build/test/lint | High | S | `flake.nix` |
| 13.2 | Add stack modules to CI workflow | High | S | `.github/workflows/ci.yml` |
| 13.3 | Add stack modules to layer check script | Medium | XS | `scripts/check-module-layers.sh` |
| 13.4 | Add `.go-arch-lint.yml` rules for deployer/app separation | Low | S | `.go-arch-lint.yml` |

---

## Summary Statistics

| Metric | Value |
|---|---|
| Total tasks | 97 |
| Phases | 13 |
| Estimated total effort | ~40 hours |
| XS tasks (< 5 min) | 22 |
| S tasks (5-12 min) | 44 |
| M tasks (12-30 min) | 31 |

## Priority Order (Top 15 by Impact/Effort)

| Priority | Task | Impact | Effort | Ratio |
|---|---|---|---|---|
| 1 | 1.2: `readmodel.Backend = kv.Store` alias | High | XS | ★★★★★ |
| 2 | 1.3-1.8: `readmodel.Store[T,K]` wrapper | High | S | ★★★★★ |
| 3 | 3.2-3.4: Bundle struct + New() | High | S | ★★★★★ |
| 4 | 5.2: Memory preset assembly | High | S | ★★★★★ |
| 5 | 7.5: Pebble ReadModels() accessor | High | XS | ★★★★★ |
| 6 | 4.1-4.2: Typed accessors | High | M | ★★★★☆ |
| 7 | 2.1-2.3: TypedSnapshot[State] | High | M | ★★★★☆ |
| 8 | 6.2-6.4: SQLite preset + bundles | High | M | ★★★★☆ |
| 9 | 3.6-3.7: Close() + rollback | High | M | ★★★★☆ |
| 10 | 10.1-10.3: Rewrite example | High | M | ★★★★☆ |
| 11 | 11.1-11.2: Contract tests | High | M | ★★★☆☆ |
| 12 | 12.1: PRESETS.md | High | M | ★★★☆☆ |
| 13 | 12.8: Update proposal HTML | Medium | M | ★★★☆☆ |
| 14 | 7.1-7.2: Pebble command/query stores | Medium | M | ★★☆☆☆ |
| 15 | 9.2: Postgres preset | Medium | M | ★★☆☆☆ |
