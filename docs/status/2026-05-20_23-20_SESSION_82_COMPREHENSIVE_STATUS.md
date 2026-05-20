# Session 82 — Comprehensive Status Report

**Date:** 2026-05-20 23:20 · **Branch:** master · **Commits this session:** 7 (on top of Session 81)

---

## A) FULLY DONE

### Critical Fixes
| Item | Detail |
|------|--------|
| **example/todo build** | Added `replace` directives for core/memory/testhelpers in go.mod. Fixed infinite `MarshalJSON` recursion in 7 types (3 query + 4 command). Fixed `TestUpdateTodo_InvalidID` assertion. All 7 example/todo packages pass. |
| **Pebble corrupt event handling** | `iterateEvents` now returns error on corrupt data instead of silently skipping. Was producing incomplete results with zero visibility. |
| **OutboxPublisher observability** | Verified: `publishPending` already logs errors with `slog.Warn`. Was previously noted as "swallowed" but already fixed. |

### Zero Lint (All 8 Modules)
| Module | Issues |
|--------|--------|
| core | 0 (was 3: embeddedstructfieldcheck, godot×2) |
| memory | 0 (was 1: unparam) |
| catalog | 0 (was 5: exhaustruct, godoclint, varnamelen×3) |
| middleware | 0 |
| integration | 0 |
| projection | 0 |
| storage | 0 (was 3: errchkjson×2, prealloc) |
| testhelpers | 0 |

### Architecture Improvements
| Item | Detail |
|------|--------|
| **DDL on Dialect interface** | `EventSchema/SnapshotSchema/CheckpointSchema/OutboxSchema` added to `Dialect`. Both `PostgresDialect` and `SQLiteDialect` implement them. Free functions (`Schema()`, `SQLiteSchema()`, etc.) now delegate — backward compatible. |
| **Pebble deserialization split** | Extracted `deserializeMetadata` from 71-line `deserializeEvent`. Cleaner, easier to test. |
| **VectorClock.Cmp() enum** | New `Cmp()` returns `ClockOrder` enum: `OrderBefore/OrderAfter/OrderConcurrent/OrderEqual`. Distinguishes equal from concurrent (unlike `Compare()` which conflated them). `Compare()` deprecated. `String()` method on `ClockOrder`. 15 new tests. |
| **Clock injection** | Verified: already injectable via `WithOccurredAt(time.Time)` option on `NewEvent`. No change needed. |
| **Logger injection** | Verified: already standardized. Constructors accept `*slog.Logger`, optional via functional options. |
| **CatalogMeta** | Evaluated: intentional per-package. `event.CatalogMeta` has extra `AggregateType` field. No change needed. |

### Files Changed (35 files, +547/-374 lines)
- `storage/dialect.go` — DDL methods on both dialects (+119 lines)
- `storage/helpers.go` — DDL functions now delegate (-42 lines)
- `storage/checkpoint.go` — DDL delegation (-15 lines)
- `storage/snapshot.go` — DDL delegation (-23 lines)
- `storage/outbox.go` — DDL delegation (-23 lines)
- `storage/pebble_event_store.go` — Error on corrupt events
- `storage/pebble_serialization.go` — Extracted `deserializeMetadata`
- `sync/vectorclock.go` — `ClockOrder` enum + `Cmp()` + `String()` (+67 lines)
- `sync/vectorclock_test.go` — 15 new tests (+70 lines)
- `example/todo/` — 14 files: replace directives, MarshalJSON fixes, dead code removal
- `.golangci.yml` — Added `w` to varnamelen ignore-names

---

## B) PARTIALLY DONE

None. Everything started this session was completed.

---

## C) NOT STARTED

From TODO_LIST.md remaining items:

| Item | Priority | Effort |
|------|----------|--------|
| Decider Execute dual `%w` wrapping | LOW | trivial — works fine in Go 1.20+ |
| Remove replace directives from go.mod | MEDIUM | blocked — need to tag modules |
| Standardize version references across go.mod | MEDIUM | small |
| Unify ErrNilBus sentinels | MEDIUM | small — but intentional per-package |
| Add BDD tests for catalog, storage, sync | LOW | medium |
| Implement Saga/Process Manager | LOW | large — design done, impl pending |
| PostgreSQL integration tests for storage | LOW | medium |
| `event.Context` propagation to `time.Now()` calls | LOW | small |

---

## D) TOTALLY FUCKED UP

**Nothing.** All 34 packages pass (24 main + 7 example/todo + 3 no-test internal). Zero lint. Clean git state.

The only close call: the `noinlineerr` linter caught the refactored `deserializeMetadata` using `if parsed, err := ...; err == nil` pattern. Fixed immediately by separating assignment and error check.

---

## E) WHAT WE SHOULD IMPROVE

1. **Golden test drift** — Golden files in `catalog/testdata/golden/` keep drifting between sessions. The `-update` flag fixes them but it's a recurring annoyance. Consider making golden tests auto-refresh in CI or pinning go-faster/yaml version.

2. **example/todo external deps** — The example pulls in `cqrs-htmx`, `cockroachdb/pebble`, `stretchr/testify`, `casbin/casbin/v3`, `prometheus/client_golang` — 37 indirect deps. The `MarshalJSON` recursion bug was in example code. Could use a simpler example or at least a lint/test gate.

3. **testhelpers drift** — `fake_bus.go` and `fake_store.go` keep getting method-level overrides that diverge from their `memory/` counterparts. Consider code generation or shared test suites.

4. **replace directive explosion** — `example/todo/go.mod` now has 4 replace directives (core, memory, storage, testhelpers). This is fragile and breaks when any module adds new deps. The proper fix is tagging modules.

5. **No CI golden file safety net** — If golden tests fail in CI, there's no way to auto-fix. Should at least `git diff --exit-code` after test run.

---

## F) Top 25 Things We Should Get Done Next

### Critical / High Impact (1-10)

| # | Item | Why | Effort |
|---|------|-----|--------|
| 1 | **Tag all modules (v1.4.0 for core, v1.2.0 for memory, etc.)** | Eliminates ALL replace directives. Unblocks downstream consumers. | small |
| 2 | **PostgreSQL integration tests for storage** | go-sqlmock can't catch real SQL dialect issues. Storage is the riskiest module (87.6% coverage). | medium |
| 3 | **Consolidate testhelpers fakes with memory implementations** | FakeStore/FakeBus are diverging from MemoryStore/MemoryBus. Shared test suites would catch this. | medium |
| 4 | **CI pipeline hardening** — golden file auto-refresh, `go test -race`, coverage gates | Prevents golden drift and race conditions from landing on master. | small |
| 5 | **Add `event.Context` propagation** — thread `ctx` through `NewEvent`, `PublishChanges`, etc. | Enables tracing/cancellation in event pipeline. Currently `time.Now()` and ID generation ignore context. | medium |
| 6 | **Decider Execute dual `%w` wrapping** — `fmt.Errorf("...: %w; %w", err1, err2)` | Technically works in Go 1.20+ but surprising. Split into two errors or use `errors.Join`. | trivial |
| 7 | **Unify ErrNilBus sentinels** — 3 independent sentinels across memory/decider/projection | Each has different context. Consider shared `event.ErrNilBus` or accept as-is. | small |
| 8 | **Storage SQLEventStore error classification** — register sentinels with `event.RegisterClassification` | Currently storage errors aren't classified. `ErrOptimisticConcurrency` → Conflict, `ErrAggregateNotFound` → Rejection, etc. | small |
| 9 | **Add BDD tests for storage module** — Ginkgo/Gomega tests for event store lifecycle | Storage has the lowest coverage (87.6%). BDD would cover edge cases better. | medium |
| 10 | **Add BDD tests for sync module** — VectorClock, ConflictResolver, LWWResolver | Sync module is new and under-tested. BDD would validate concurrent scenarios. | medium |

### Medium Impact (11-20)

| # | Item | Why | Effort |
|---|------|-----|--------|
| 11 | **example/todo simplify** — remove cqrs-htmx, casbin, prometheus deps | Reduce external attack surface. Example should be minimal. | medium |
| 12 | **CatalogMeta shared base** — extract common Name/Version/Summary fields | Even though event has AggregateType, a shared base struct reduces 3× duplication. | small |
| 13 | **Add benchmarks for storage DDL path** — measure Schema() overhead | DDL generation is now on interface. Should be zero alloc. | trivial |
| 14 | **Projection Runner structured error wrapping** — wrap handler errors with projection name | Currently errors from handlers lose context about which projection failed. | small |
| 15 | **Add `event.Store.BatchLoad(ctx, []AggregateRef) map[string][]Event`** | For bulk state reconstruction (e.g., list view projections). Avoids N+1 queries. | medium |
| 16 | **Storage: add `LoadAllFromTimestamp` to event.Store interface** | Symmetric with `LoadToTimestamp`. Currently only `LoadAllFromPosition` exists. | small |
| 17 | **Middleware: add `WithLogger` to all middleware constructors** | Logging middleware uses `slog` directly. Retry/recovery/validation should accept logger for consistency. | small |
| 18 | **Add `event.Event.Clone()` method** — defensive copy for mutation safety | Currently consumers must be careful not to mutate event payloads. Clone() makes this safe. | trivial |
| 19 | **Add `decider.Decider[State].MustExecute` — panic variant of Execute** | Consistent with `MustNew*` pattern. Useful in bootstrapping/tests. | trivial |
| 20 | **Add `catalog.SchemaFromType[T]()` tests for edge cases** — embedded unexported, circular refs, time.Time, []byte | Schema reflection has no edge-case tests. Could break on complex types. | small |

### Lower Impact / Future (21-25)

| # | Item | Why | Effort |
|---|------|-----|--------|
| 21 | **Saga/Process Manager implementation** — design is done in `docs/planning/SAGA_DESIGN.md` | Full saga orchestration is the last major missing feature. | large |
| 22 | **Pebble store compaction/config tuning** — expose Pebble options | Currently hardcoded. Production use needs compaction tuning. | medium |
| 23 | **Add `event.Outbox.Ack(ctx, []EventID) error` batch variant** | More efficient than individual acks for high-throughput scenarios. | small |
| 24 | **Add `projection.Runner.WithMetrics(metrics.ProjectionMetrics)` option** | Currently no observability into projection lag/throughput. | medium |
| 25 | **Storage: add SQLite WAL mode configuration** — `PRAGMA journal_mode=WAL` | WAL mode is significantly faster for concurrent read/write. Should be default. | trivial |

---

## G) Top #1 Question I Cannot Figure Out Myself

**When should we tag module releases (v1.4.0, v1.2.0, etc.)?**

The `replace` directive problem is the #1 blocker for external consumers. But tagging is a one-way door — once published, we can't un-publish. Key unknowns:
- Is the `storage` module API stable enough? (DDL on Dialect just changed it)
- Should `sync` module be tagged separately or stay at v0.0.0?
- Should `example/todo` stay as a replace-directive module or get its own tagged release?
- Is there a downstream consumer waiting for tags, or is this premature optimization?

This is a product decision, not a technical one.

---

## Metrics Dashboard

| Metric | Value |
|--------|-------|
| Test packages passing | 24/24 (main) + 7/7 (example/todo) |
| Lint issues | 0 (all 8 modules) |
| Production LOC | 15,678 |
| Test LOC | 30,781 |
| Test files | 126 |
| Commits this session | 7 |
| Total coverage | ~93% |
| Lowest coverage module | storage (87.6%) |
| Highest coverage module | core/query (100%) |
| Sentinels classified | 38 across 7 modules |
| Benchmarks | 43 across 12 files |
| Files over 250 lines | 0 |
