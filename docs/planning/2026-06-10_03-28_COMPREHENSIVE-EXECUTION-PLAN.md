# Comprehensive Execution Plan — go-cqrs-lite

**Date:** 2026-06-10
**Source:** TODO_LIST.md (all open items) + code review findings + architecture audit
**Scope:** ALL open TODOs broken into tasks ≤12min each
**Sorting:** Impact × (1/Work) × Customer Value — highest first

## CRITICAL CORRECTION

The 2026-06-10 architecture review claimed "4 bidirectional cycles." This was **wrong**.

Verified by checking production imports (excluding `_test.go` and `eventtest/`):

| Claimed Cycle     | Reality                                                                |
| ----------------- | ---------------------------------------------------------------------- |
| event ↔ command   | **Test-only** — event production code imports ZERO from command        |
| event ↔ query     | **Test-only** — only `errors_taxonomy_test.go`                         |
| event ↔ memory    | **Test-only** — only test files + eventtest                            |
| event ↔ schema    | **Test-only** — only test files                                        |
| memory ↔ snapshot | **One-directional** — memory→snapshot exists, snapshot→memory does NOT |

**Production DAG is clean.** The `go.mod` listings are misleading because Go doesn't separate test-only deps.

The real issue: `event/go.mod` pulls in 5 deps (command, query, memory, schema, snapshot) that are only needed by test files and eventtest/. This is binary bloat for consumers, not a compile-time cycle.

---

## Tier 1: Quick Wins (≤30min each, HIGH impact)

| #   | Task                                                                                                    | Module              | Impact | Effort | File(s)                                     |
| --- | ------------------------------------------------------------------------------------------------------- | ------------------- | ------ | ------ | ------------------------------------------- |
| 1   | Fix README.md broken CI badge URLs (test.yml→ci.yml, lint.yml→ci.yml, Go Reference core→root)           | README              | HIGH   | 12min  | `README.md`                                 |
| 2   | Remove command error re-exports — delete ~60 lines of dead API from command/errors.go, update consumers | command             | HIGH   | 12min  | `command/errors.go`                         |
| 3   | Rename `WithNewCodec` → `WithCodec` in event (misleading "New" prefix) + update decider reference       | event               | MED    | 8min   | `event/options.go`                          |
| 4   | Add `IsReplay(ctx) bool` getter to complement `WithReplay`                                              | event               | MED    | 8min   | `event/replay.go`                           |
| 5   | Rename `ErrNilBus` → `ErrNilPublisher` in decider, `ErrNilBus` → `ErrNilSubscriber` in projection       | decider, projection | MED    | 8min   | `decider/errors.go`, `projection/errors.go` |
| 6   | Remove `StreamKey()` free function — duplicates `AggregateRef.StreamKey()`, update 1 caller in memory   | event               | LOW    | 8min   | `event/stream.go`, `memory/snapshot.go`     |
| 7   | Remove `Map`/`ScanState`/`Tap` reactive wrappers — test-only pass-throughs, pollute public API          | event               | LOW    | 10min  | `event/reactive.go`                         |
| 8   | Fix `NewMetadata()` to initialize `Custom: make(map[string]string)` instead of nil map                  | event               | LOW    | 5min   | `event/metadata.go`                         |

## Tier 2: Type Safety & API Quality (30-60min, HIGH impact)

| #   | Task                                                                                                                     | Module           | Impact | Effort | File(s)                                                                                                       |
| --- | ------------------------------------------------------------------------------------------------------------------------ | ---------------- | ------ | ------ | ------------------------------------------------------------------------------------------------------------- |
| 9   | Make `dispatcher.Lifecycle` unexported — add `IsClosed()`, `Close()` methods on Dispatcher, hide field                   | dispatcher       | HIGH   | 30min  | `dispatcher/dispatcher.go`, `dispatcher/lifecycle.go`, update command/dispatcher, query/dispatcher, memory/\* |
| 10  | Fix AggregateProjection hardcoded `?` placeholders — use `d.Placeholder(n)` for Postgres compat                          | storage          | HIGH   | 20min  | `storage/aggregate_projection.go`                                                                             |
| 11  | Consolidate `listRefsFromStatus` — move to listing/, have storage/ import it instead of duplicating                      | listing, storage | MED    | 25min  | `listing/aggregate_reader.go`, `storage/sql_aggregate_reader.go`                                              |
| 12  | Fix `decider/load.go` double-wrapping — use `event.WrapInfrastructure(err, ...)` directly instead of `fmt.Errorf` + wrap | decider          | MED    | 15min  | `decider/load.go`                                                                                             |

## Tier 3: Duplication Elimination (1-2hr, HIGH impact)

| #   | Task                                                                                                | Module     | Impact | Effort | File(s)                                                                                             |
| --- | --------------------------------------------------------------------------------------------------- | ---------- | ------ | ------ | --------------------------------------------------------------------------------------------------- |
| 13  | Extract generic `sql.QueryEngine[T]` in `storage/sql/` — shared loadParams, loadWithSpan, queryRows | storage    | HIGH   | 90min  | `storage/sql/query_engine.go` (new), `storage/event_store_load.go`, `storage/command_store_load.go` |
| 14  | Move HTTP code from middleware to `middleware/http/` sub-package — SSE, healthcheck, metrics_http   | middleware | MED    | 60min  | `middleware/sse.go` → `middleware/http/sse.go`, etc.                                                |

## Tier 4: Architecture Improvement (2-3hr, MED-HIGH impact)

| #   | Task                                                                                                               | Module | Impact | Effort | File(s)                                                        |
| --- | ------------------------------------------------------------------------------------------------------------------ | ------ | ------ | ------ | -------------------------------------------------------------- |
| 15  | Remove test-only deps from event/go.mod — extract eventtest as separate module with own go.mod                     | event  | HIGH   | 120min | New `event/eventtest/go.mod`, update `event/go.mod`, `go.work` |
| 16  | Add metadata support to `query.BasicQuery` — WithCorrelationID, WithCausationID, WithUserID, WithRequestID options | query  | MED    | 60min  | `query/query.go`, `query/metadata.go` (new)                    |
| 17  | Fix pebble unbounded lock map — replace `sync.Map` with sharded lock pool or sync.Pool pattern                     | pebble | MED    | 45min  | `pebble/store.go`                                              |

## Tier 5: Code Quality Polish (30-60min each, MED impact)

| #   | Task                                                                                             | Module     | Impact | Effort | File(s)                                           |
| --- | ------------------------------------------------------------------------------------------------ | ---------- | ------ | ------ | ------------------------------------------------- |
| 18  | Fix projection Runner.Close() — add sync.WaitGroup for graceful shutdown of in-flight handlers   | projection | MED    | 30min  | `projection/runner.go`                            |
| 19  | Fix README.md v2 import paths — add `/v2` suffix to all import examples                          | README     | MED    | 20min  | `README.md`                                       |
| 20  | Fix README.md stale "v2.0.0 pre-release" → reflect v2.2.0 status                                 | README     | LOW    | 5min   | `README.md`                                       |
| 21  | Fix README.md Turso formatting — stray `-db,` and `-syncDB,` in code blocks                      | README     | LOW    | 5min   | `README.md`                                       |
| 22  | Remove `event.SaveFunc` from public API — only used in eventtest, move there                     | event      | LOW    | 15min  | `event/store.go`, `event/eventtest/fake_store.go` |
| 23  | Rename reactive `EventBus` alias → `Stream` or `Subject` to avoid confusion with `Bus` interface | event      | LOW    | 15min  | `event/reactive.go`                               |

## Tier 6: Documentation & Tests (30-60min, MED impact)

| #   | Task                                                                              | Module    | Impact | Effort | File(s)                                                                     |
| --- | --------------------------------------------------------------------------------- | --------- | ------ | ------ | --------------------------------------------------------------------------- |
| 24  | Fix `event/eventtest/fake_store.go` — 273 lines, add tests for untested mock code | eventtest | MED    | 45min  | `event/eventtest/fake_store_test.go` (improve)                              |
| 25  | Update architecture review HTML — correct "4 cycles" to "test-only deps"          | docs      | MED    | 15min  | `docs/architecture-understanding/2026-06-10_01-27_architecture-review.html` |
| 26  | Update D2 diagrams — remove red "cycle" lines, add "test-only dep" labels         | docs      | LOW    | 15min  | `docs/architecture-understanding/2026-06-10_01-27_current-architecture.d2`  |
| 27  | Write ADR for eventtest module extraction decision                                | docs      | MED    | 20min  | `docs/adr/0014-extract-eventtest-module.md`                                 |

## Tier 7: Existing [v2] Deferred Items (BREAKING — do NOT execute now)

| #   | Task                                                                         | Module                     | Impact | Effort | Note                      |
| --- | ---------------------------------------------------------------------------- | -------------------------- | ------ | ------ | ------------------------- |
| 28  | [v2] Add global `TransactionID` branded type for cross-aggregate consistency | id                         | MED    | 60min  | ADR needed first          |
| 29  | [v2] Remove `io.Closer` from core interfaces                                 | event, snapshot, command   | HIGH   | 4hr    | ADR-0010 exists, proposed |
| 30  | [v2] Split event.Store into Writer/Reader/Deleter                            | event                      | HIGH   | 3hr    | Breaking change           |
| 31  | [v2] Make event Core truly immutable                                         | event                      | HIGH   | 2hr    | Breaking change           |
| 32  | [v2] Unify ErrDispatcherClosed (ADR-0011)                                    | dispatcher, command, query | MED    | 2hr    | ADR-0011 exists, proposed |

## Tier 8: [BLOCKED] / [FUTURE] (NOT ACTIONABLE NOW)

| #   | Task                                                                  | Why Blocked                 |
| --- | --------------------------------------------------------------------- | --------------------------- |
| 33  | [BLOCKED] Move example/todo to own repository                         | Manual repo creation        |
| 34  | [BLOCKED] Add PostgreSQL integration tests with testcontainers        | Docker/testcontainers setup |
| 35  | [BLOCKED] Remove cockroachdb/errors from go-localsync                 | Different repo              |
| 36  | [BLOCKED] Create go-branded-id v0.2.0                                 | Different repo              |
| 37  | [BLOCKED] Design ActaFlow event sourcing overlay                      | Different project           |
| 38  | [BLOCKED] Extract shared golangci.yml into larsartmann/library-policy | Different repo              |
| 39  | [BLOCKED] Change LICENSE from proprietary to MIT/Apache-2.0           | Owner decision              |
| 40  | [BLOCKED] Migrate ActaFlow build to flake.nix                         | Different project           |
| 41  | [BLOCKED] Integrate TypeSpec types → catalog.Registry                 | Different project           |
| 42  | [FUTURE] Add catalog diff/breaking-change detection tool              | Speculative                 |
| 43  | [FUTURE] Add high-level test utilities (AggregateTester, etc.)        | Speculative                 |
| 44  | [FUTURE] Add ServerReceivedAt/ServerStoredAt timestamps               | Speculative                 |
| 45  | [FUTURE] Add bi-temporal support (ValidAt, WithValidAt)               | Speculative                 |
| 46  | [FUTURE] Absorb projection/ into core/event                           | Speculative                 |
| 47  | [FUTURE] Add HLC (Hybrid Logical Clock)                               | Speculative                 |
| 48  | [FUTURE] Implement pull-before-push sync protocol                     | Speculative                 |
| 49  | [FUTURE] Implement rebase mechanism                                   | Speculative                 |
| 50  | [FUTURE] Build network simulator for testing                          | Speculative                 |
| 51  | [FUTURE] Build multi-client test harness                              | Speculative                 |
| 52  | [FUTURE] Build thin PostgreSQL store adapter (no Watermill)           | Speculative                 |
| 53  | [FUTURE] Build thin NATS bus adapter (no Watermill)                   | Speculative                 |
| 54  | [FUTURE] Add Filter, Predicate types for context queries              | Speculative                 |
| 55  | [FUTURE] Add ContextQuerier, ContextAppender, QueryResult interfaces  | Speculative                 |
| 56  | [FUTURE] Make transactional projection contract explicit              | Speculative                 |
| 57  | [FUTURE] Add multi-engine storage support via sqlc                    | Speculative                 |
| 58  | [FUTURE] Add schema migration tool                                    | Speculative                 |
| 59  | [FUTURE] Add hybrid service example                                   | Speculative                 |
| 60  | [FUTURE] Add distributed consensus capability (Raft/CRDT)             | Speculative                 |
| 61  | [FUTURE] Add time-series event query language                         | Speculative                 |
| 62  | [FUTURE] Create documentation site (Docusaurus/MkDocs/Hugo)           | Speculative                 |
| 63  | [FUTURE] Set up pkg.go.dev documentation hosting                      | Speculative                 |
| 64  | [FUTURE] Add Pebble Journal/SeekableJournal support                   | Feature gap                 |

---

## Summary Statistics

| Tier                   | Tasks  | Total Effort | Avg Impact      |
| ---------------------- | ------ | ------------ | --------------- |
| Tier 1: Quick Wins     | 8      | ~71min       | HIGH            |
| Tier 2: Type Safety    | 4      | ~90min       | HIGH            |
| Tier 3: Dedup          | 2      | ~150min      | HIGH            |
| Tier 4: Architecture   | 3      | ~225min      | MED-HIGH        |
| Tier 5: Polish         | 6      | ~130min      | MED             |
| Tier 6: Docs/Tests     | 4      | ~95min       | MED             |
| Tier 7: [v2] Breaking  | 5      | ~12hr        | HIGH (deferred) |
| Tier 8: Blocked/Future | 32     | N/A          | N/A             |
| **TOTAL ACTIONABLE**   | **32** | **~13hr**    |                 |

## Execution Order

Start with Tier 1 (tasks 1-8), then Tier 2 (tasks 9-12), then Tier 3 (tasks 13-14), then Tier 4-6 as time permits. Each task is independently committable. Skip Tier 7-8 entirely.

After each task: `nix run .#build && nix run .#test && git add -A && git commit`
