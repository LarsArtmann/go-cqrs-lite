# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

#### cqrs-lint post-v4.3.0: scorecard, group-by aggregate, C038-C040, config UX

- **Scorecard subcommand** — `cqrs-lint --scorecard` / `cqrs-lint scorecard`.
  Module adoption scorecard: detects used/missing go-cqrs-lite modules,
  computes coverage %, recommends top-3 modules. `ModuleCatalog` with 28
  modules across 7 categories. Profile-relative filtering. Text + JSON output.
- **Group-by aggregate** — `--group-by aggregate` infers aggregate names from
  event-type prefixes (`user.created` → `user`) and decider/fold state types
  (`CounterState` → `counter`). Groups findings by aggregate (most issues first).
- **C038/C039/C040 rewritten** — C038 now detects near-miss event type strings
  in `switch evt.Type()` blocks via Levenshtein distance. C039 flags event types
  emitted but never handled. C040 detects dead `case` branches in fold switch
  statements. Rule count: 185→186.
- **Per-module feature detection** — `ProfileForFile` evaluates feature profiles
  per-module in multi-module workspaces. C017 migrated; 26 detectors still on
  primary profile.
- **JSONC config loader** — `.cqrs-lint.json` now supports comments (`//` line
  comments and `/* */` block comments) via `stripJSONComments` parser.
- **`explain` subcommand** — interactive documentation explorer for config keys,
  presets, rules, and feature flags.
- **`doctor` overhaul** — now shows active preset, resolved feature overrides,
  disabled rules, suppression counts, and parent-config inheritance chain.
- **`init` SHOWSTOPPER fix** — `cqrs-lint init` no longer produces a broken
  config (array vs string parser mismatch). Now generates valid JSONC via
  `generateInitConfig`.
- **Module catalog extraction** — `cmd/cqrs-lint/pkg/analyzer/module_catalog.go`
  - `module_catalog_data.go` extracted from monolithic analyzer.

#### Metaengine: ReadCosts, DuckDB+PG calibration, inspect.go extraction

- **`ReadCosts` per-read-pattern cost model** — `EngineProfile.ReadCosts` adds
  separate cost fields for point-lookup, scan, and aggregation reads. Exposes
  the 4000× gap between DuckDB point lookups (~133 ns) and aggregations.
  DuckDB + Postgres engines calibrated. `metaengine/readcost_selection_test.go`
  validates planner selection with ReadCosts.
- **DuckDB + Postgres calibration benchmarks** — 4 benchmarks per engine
  (batch insert, pushdown scan, vectorized aggregation, full scan). Exposed
  the single-scalar cost model flaw that led to ReadCosts.
- **Benchmark correctness assertions** — 50+ benchmarks across 18 files now
  assert results. Found 3 real bugs: (1) `BenchmarkMemoryStore_Save` used
  expectedVersion=1 on empty stream; (2) `BenchmarkMemoryStore_ReadFrom_Scale`
  read from LAST event ID (always empty); (3) JSON map decode silently failed.
  `benchkit.RunSuite` now `b.Fatalf`s on integrity errors.
- **`Store.Inspect()` / `InspectJSON()` extraction** — moved from `sse.go` to
  `metaengine/inspect.go` for file cohesion. Collection introspection (key
  count, engine, ADT) has nothing to do with SSE.
- **Persistence enum (ADR-0098)** — `EngineProfile.Persistence` field
  (`PersistenceVolatile`/`PersistencePersistent`, DDIA Ch1 reliability axis).
  Per-engine `Profile()` sets persistence dynamically (Memory=volatile,
  SQLite/Pebble/DuckDB/PG=persistent). `durabilityRule` emits WARN when volatile
  engines hold materialized projections across restarts. `CollectionInfo` +
  `SerializablePlan` include persistence. `Doctor()` has `--- Persistence ---`
  section. `ExplainPlan()` shows persistence on engine lines.

#### MySQL/MariaDB support: stack preset, dialect methods, classifier, docs

- **`stack/mysql` preset** — full MySQL/MariaDB stack bundle with
  auto-migration, multi-DB topology support (`mysql.WithDSN`), and the same
  API surface as `stack/postgres` and `stack/sqlite`. Uses the pure-Go
  `go-sql-driver/mysql` driver (no CGo). Contract test suite and multi-DB
  routing tests pass against `mysql:8.0` via testcontainers.
- **Dialect upsert methods** (ADR-0080) — 4 new methods on `Dialect`:
  `UpsertEventSQL`, `UpsertSnapshotSQL`, `UpsertKVSQL`, `QuoteIdentifier`.
  MySQL uses `ON DUPLICATE KEY UPDATE` + `IF()` conditional logic instead
  of `ON CONFLICT ... WHERE`. The existing Postgres/SQLite paths are
  unchanged.
- **MySQL error classifier** — `storage/sql` now classifies MySQL error
  numbers: 1062 (duplicate key) → Conflict, 1205/1213 (lock timeout/deadlock)
  → Transient. Tested with 7 cases covering numeric codes and string fallbacks.
- **MySQL multi-statement DDL** — `MySQLInitSchema` splits the embedded schema
  into individual `CREATE TABLE` statements, avoiding the need for
  `multiStatements=true` in the DSN (a SQL-injection risk).
- **MySQL testcontainer pattern** — `ctr.Exec` for root privilege escalation
  (GRANT ALL PRIVILEGES) instead of host-side root DSN string replacement.
  Reliable, no `caching_sha2_password` auth issues.
- **`idempotency/sqlstore` MySQL support** — `NewMySQLStore` with
  `ON DUPLICATE KEY UPDATE` + `IF()` conditional TTL reclaim. Unit tests
  verify MySQL-specific SQL syntax correctness.
- **`cqrs-lint` MySQL detection** — `StoreMySQL` feature detection + A009
  suggestion + T007/T008 production-store rules. Test coverage for
  detection via `stack/mysql` import.
- **Documentation** — ADR-0080, `stack/mysql/README.md`, updated skill
  references (core/modules/recipes/faq), FEATURES.md, ROADMAP.md.

#### Self-review execution: goroutine leak, fuzz tests, StreamScan, lint fixes

- **Goroutine leak fix in Watch/WatchWithSeq** — the adapter goroutine in
  `metaengine.Watcher.Watch` and `WatchWithSeq` exited without closing the
  consumer-facing channel, leaving consumers blocked forever. Added
  `defer close(ch)` in both adapter goroutines. Regression tests verify
  channels close on context cancel AND Watcher.Close.
- **Pebble StreamScan** — implemented `StreamingScan` interface on
  `pebbleEngine` with `iter.Seq2` lazy iteration. Unsorted scans are O(1)
  memory per row; sorted scans materialize internally (documented tradeoff).
  Tested with unsorted, filtered, and early-exit scenarios.
- **Pebble ScanCount** — new `ScanCounter` optional interface. Counts
  collection items with O(1) memory (no JSON decode for unfiltered path).
  Filtered path decodes only to evaluate predicates.
- **formatIndexInt regression tests** — pin the 20-digit zero-pad encoding
  that ensures lexicographic ordering matches numeric ordering for ALL
  integers (negative, mixed-digit, max/min int64).
- **SortSpec/FilterSpec validation** — `extractDeclarativeFields` now returns
  an error when any FilterSpec or SortSpec has an empty Column name. Caught
  at Plan() time, not at scan time.
- **MapUpdate fuzz tests** — `FuzzMapUpdate_ConcurrentCounter` verifies no
  lost updates under concurrent MapUpdate (2-200 goroutines).
  `FuzzMapUpdate_CreateOrUpdate` verifies the create-if-absent pattern.
- **Cross-engine property-based parity test** — `TestProperty_MapSetGetParity`
  uses `pgregory.net/rapid` to generate random MapSet/MapGet/MapDelete
  sequences and verifies memory engine and SQLite engine agree on existence
  after every operation.
- **100K cursor pagination benchmark** — extends scan benchmarks from 10K to
  100K items for filter-indexed cursor pagination.
- **memory.New(metaengine) integration test** — verifies that
  `memory.New(stack.WithMetaEngine(store))` produces a fully wired bundle with
  both default capabilities AND the metaengine store.
- **Doc rule count CI check** — `scripts/check-rule-count.sh` + `nix run
.#check-rule-count` verifies FEATURES.md, ROADMAP.md, AGENTS.md rule counts
  match `rules.RegisterAll()` length. Prevents doc drift.
- **E012 alias-awareness** — migrated from raw `projectCalls(ctx, "flag", ...)`
  to alias-aware `projectCallsImportPathBool`. Removed dead code (`projectCalls`
  and `projectCallsAny` — superseded by alias-aware versions).
- **financialKeywords lint fix** — added `//nolint:gochecknoglobals` (constant
  lookup table belongs at package level).
- **Sweep app auto-fix** — sweep now runs `golangci-lint --fix` before lint
  check, not just formatting + report.
- **Float encoding limitation documented** — `encodeIndexValue` doc now
  explicitly warns that floats with fractional parts do NOT preserve
  lexicographic ordering and recommends integer-scaled values for indexed
  columns.
- **API surface** — updated from 2911 to 2965 exports (StreamScan, ScanCount,
  ScanCounter interface, property test types).

#### Pareto plan execution: correctness, tests, docs, release prep

- **scanWithIndex cursor pagination fix** — the Pebble LayoutPlanner's filter
  index path silently dropped cursor values, returning the same first N items
  on every page request. Added `paginateIndexedResults` + `processFilterIndex`
  helpers. Ascending and descending cursor pagination now verified by
  `TestPebbleLayoutPlanner_FilterIndexCursor{Ascending,Descending}`.
- **Fuzz test for ScanRawValues** — `FuzzScanRawValues` exercises the filter
  index path with arbitrary threshold values, including edge cases (0, negative,
  large numbers). Seeds cover known tricky values.
- **Pebble LayoutPlanner edge case tests** — empty filter results, concurrent
  read/write (race-detector clean), key collision (update doesn't duplicate),
  no-layout full scan with filter+sort.
- **Scan benchmarks** — filter index, sort index, and full scan paths benchmarked
  at 100/1K/10K items.
- **D007/D008/D010/D013 import-alias migration** — consistency rules migrated
  from variable-name heuristics to `lintutil.QualifierResolvesTo` (alias-aware
  import path resolution). Rules now work with aliased imports like
  `import ev "github.com/larsartmann/go-cqrs-lite/event/v4"`.
- **`memory.New` accepts extra options** — `stack/memory.New(extra ...stack.Option)`
  allows callers to extend the default wiring (e.g. `stack.WithMetaEngine(store)`).
- **ADR-0075: ADT test harness extraction** — documents why `adttest` was
  extracted as an exported sub-package for cross-engine parity testing.
- **ADR-0076: Pebble raw value readers** — documents the single-pass JSON decode
  optimization and optional interface design.
- **Tag `stack/duckdb/v4.0.0`** — created locally (push pending). First release
  of the DuckDB analytical engine preset.
- **Enhanced `sweep` app** — now runs `nix fmt` + build check + golangci-lint
  in one command for post-daemon cleanup.
- **4 stale status reports annotated** — inline corrections on 05:02, 05:44,
  22:22, and 23:22 reports with current rule counts (179), API surface (2911),
  and resolution status.
- **Per-category rule counts restored** — FEATURES.md, ROADMAP.md, and AGENTS.md
  updated with verified counts: correctness 36, API 30, boilerplate 28, adoption
  21, architecture 17, consistency 15, performance 9, security 9, testing 8,
  version 6.

#### cqrs-lint: quality hardening (171 → 179 rules)

- **4 new architecture rules** (E008–E011) — stack preset bypass detection,
  missing HTTP integration, capture without domain validation, excessive
  adapter layers. Brings total to **179 rules across 10 categories**.
- **Type-aware rule rewrites** — E010 rewritten with `projectCallsMethodOnType`
  (go/types receiver matching), E014 rewritten with type-aware projection-host
  matching. E013 already used type-aware composite literal matching.
- **Library self-lint mode** — `IsLibrarySelfLint()` auto-detects when linting
  the go-cqrs-lite source itself and skips 29 consumer-coaching rules (8
  architecture + 21 adoption). Eliminates need for 181+ manual inline suppressions.
- **Import-alias resolution** — `QualifierToImportPath` + `ImportQualifierMap`
  helpers in `lintutil.go`. `projectCallsImportPath` wrapper added to
  architecture/helpers.go. E008 migrated as proof of concept.
- **Type-aware F011/F013** — F011 `countSQLExec` now uses type info to verify
  receiver is `*sql.DB`/`*sql.Tx`/`*sql.Conn`. F013 now detects chi/gin/echo/
  fiber/gorilla/httprouter web framework imports.
- **22MB committed binary removed** — `git rm --cached` + `.gitignore` entry.
- **Suppression tests** for C031-C034, P011-P012, D014-D015, A032, E016-E017, S010.
- **Flaky benchkit soak tests fixed** — `soakTestScale` with `raceEnabled`
  build-tag multiplier for `-race`-inflated thresholds.

#### Metaengine: production hardening (6 sessions, 2026-07-30 to 2026-07-31)

- **Transaction API** — fully threaded `*sql.Tx` through engine operations for
  atomic multi-collection updates. Prior implementation was broken with a
  weakened test masking the failure.
- **SQL injection fix** — `quoteIdent` wraps identifiers; `MigrateLayout` no
  longer accepts raw user input in SQL.
- **Hooks fire on errors** — `OnApply` hook now receives the error parameter;
  removed success-only nil guard.
- **SSE event delivery** — `ServeSSE` with Last-Event-ID reconnection,
  backpressure via `BlockPublishUntilSubscriberAck`, `dedup.Ring` for
  replay→live overlap dedup, byte-budgeted replay, subscribe-before-replay
  ordering. Full rewrite of replay path.
- **PrefetchCache** — cursor-encoded auto-population cache for paginated reads.
  Thread-safe via `sync.RWMutex`. `prefetchCursorKey` uses `Cursor.Encode()`.
- **Watcher** — reactive change notifications with per-key filtering.
- **ADT test harness** — `metaengine/adttest` package with `RunMatrix` for
  cross-engine parity testing across all 7 ADTs (Map, Set, Counter, Multimap,
  Log, Graph, Scan). Reflect-based `backendInterfaces` for automatic
  capability detection. 12 harness self-tests.
- **Pebble LayoutPlanner** — secondary index with O(matches) prefix scan
  (108x speedup over full scan, 6ms→56μs). MapDelete cleans up index entries
  atomically. MapUpdate reindexes. Range filters (FilterGt/Ge/Lt/Le) use
  index bounds. Sort index stored but not yet used for ordering.
- **Raw value readers** — `RawValueReader`/`RawScanReader` interfaces skip
  JSON decode for filter/sort/cursor paths. `GetRawValue` returns raw bytes.
  Triple-decode bug fixed (filter + sort + cursor each decoded separately →
  now single-pass decode). Shared `sortAndPaginate` helper extracted from
  duplicated code in engine.go + raw_reader.go. `kvPair` struct unifies the
  key+value+raw representation.
- **Aggregate pushdown** — `AggregateReader` interface. SQL COUNT/SUM/MIN/MAX/
  AVG pushdown via the engine. `TypedReader.Count` prefers pushdown, falls
  back to `Scan + len()`.
- **Error sentinels wired** — `ErrNotFound` returned from `ExecuteTyped` for
  nil results. `ErrLayoutConflict` from conflicting column sets in `ApplyLayout`.
  `IsPoisoned` wired into all read paths.
- **ContractSuite expanded** — all 7 ADTs tested. Gocyclo reduced from 41 to ~15.
- **Data race fix** — `sync.Mutex` protecting `results` map in `RunMatrix`
  parallel subtests.
- **Exported helpers** — `PassesFilterSpecs`, `ItemFieldByName`, `CompareValues`,
  `EvalFilterOp` with full godoc.

#### cqrs-lint: Pareto plan execution (159 → 171 rules, +12 new + 3 extensions)

- **12 new detector rules** from the Pareto improvement backlog:
  - **C031** — error swallowing in `RegisterTyped` handlers (`return nil` on error)
  - **C032** — context propagation gaps (`context.Background()`/`TODO()` in ctx functions)
  - **C033** — missing error wrapping (bare `return err` after CQRS method calls)
  - **C034** — goroutine without ctx (`go func()` without context propagation)
  - **P011** — unbounded map growth in read models (OOM risk)
  - **P012** — missing SQLite WAL mode (lock contention risk)
  - **D014** — event payloads without json tags (Go field names in JSON)
  - **D015** — nullable pointer fields in event payloads (nil-deref panic risk)
  - **A032** — string/int IDs instead of branded `id.Of[T]` (type safety loss)
  - **E016** — missing health checks in server-mode projects (K8s survival)
  - **E017** — missing graceful shutdown on SIGTERM (in-flight events lost)
  - **S010** — bus encryption/signing without store wrapper (cleartext storage)
- **3 existing rules extended**:
  - **C008** — now detects `float32` money fields + added `rate` keyword
  - **C010** — now detects SQL error swallowing (`Exec`/`Query`/`Scan`/`Get`/`Select`)
  - **B008** — now detects bitshift backoff bug in retry loops (escalates to error)
- **Bug fix: suppression snippet fallback** (item 130) — `extractRuleID` returned
  only the first ID for comma-separated suppressions; replaced with
  `ParseSuppressions` which handles all IDs.
- **Backlog pruning**: 25 improvement ideas marked won't-implement with rationale
  (tutorial system, premature optimization, scope creep, cqrs-htmx-specific).
  Open backlog reduced from 75 to ~42 items.

#### cqrs-lint: massive rule expansion (65 → 159 rules across 10 categories)

- **94 new detector rules** across 8 new and existing categories. The linter grew
  from 65 rules in 6 categories (v4.2.0) to **159 rules in 10 categories**.
  New categories: testing (T-series), adoption (F-series), architecture (E-series),
  version (V-series expanded). Existing categories expanded: correctness (+13),
  API (+14), boilerplate (+13), consistency (+7), security (+5), performance (+5).
  Every rule has unit tests and is registered via the catalog meta-test
  (`TestCatalogCountMatchesRegister`).
- **F-series adoption coaching rules** (17 rules, F001–F017) — proactively coach
  consumers toward unused features: tombstone soft-delete, catalog docs, OTel,
  Prometheus, encryption, CBOR, scheduling, graph/relational projections, deriver,
  transport, kv.Cache, metaengine, listing, dedup.
- **T-series testing-quality rules** (8 rules, T001–T008) — detect missing test
  helpers, t.Parallel coverage gaps, snapshot store mock misuse, replay-mode test
  isolation issues.
- **E-series architecture rules** (8 rules, E008–E015) — detect consumer design
  issues: stack preset bypass, missing HTTP integration, capture without domain
  validation, excessive adapter layers, dual-write without completion, signing
  disabled by default, no read-your-writes, ordered delivery disabled.
- **V-series version rules** (5 new, V002–V006) — detect unpinned go.mod versions,
  version lag behind latest tag, vendored third-party modules, eventtest version
  mismatch, mixed version pins across modules.
- **Architecture refactor of cqrs-lint cmd** — `AllRules()` memoized via
  `sync.OnceValue`, `detectorCategory` cached as O(1) map lookup, `run()` god
  function split into 6 stages (applyConfigOverrides, handleLoadErrors,
  selectDetectors, runPipeline, filterFindings, printSummary). `toolName`
  consolidated to `lintutil.ToolName`. 3 new meta-tests (severity/confidence
  valid, critical detectors, detector names match catalog).
- **Self-lint suppression** — 181 inline suppressions across 83 files for
  library self-referential false positives. Suppression parser extended to
  handle space after `//` and comma-separated rule IDs.
- **Pareto improvement backlog** (`docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md`)
  — 50 will-implement items triaged from 75 open ideas, with 25 pruned with
  rationale.

#### Metaengine: pushdown, layout planning, Pebble engine, streaming

- **SQL pushdown** (`metaengine`) — `FilterOnField`/`SortOnField` declarative
  specs push WHERE/ORDER BY/LIMIT into SQLite via `json_extract()` (ADR-0072).
  Reduces SortedMap scan from O(N) to O(log N) for filtered/sorted queries.
- **Layout planning** (`metaengine`) — `LayoutPlan`/`BuildLayoutPlanFromType[R]`
  generate indexed-column DDL from declared query fields (ADR-0073). Planned
  tables use indexed columns instead of `json_extract` — 10x speedup on
  filter+sort. `plannedSQLiteEngine` for deployment-time table creation.
- **Pebble engine** (`metaengine/pebbleengine`) — LSM point reads (~7x faster
  than SQLite on MapGet). Separate module with `cockroachdb/pebble` dependency
  (ADR-0074). All 7 ADT backends implemented: Map, Set, Counter, Multimap, Log,
  Graph, MapUpdater/Scan. 10 parity tests.
- **Streaming reads** (`metaengine`) — `StreamingScan`/`StreamScan` interface
  for OOM-safe lazy iteration via `iter.Seq2`.
- **Cost model calibration** — `EngineProfile.NsPerRead`/`NsPerWrite` split
  (backward-compat fallback to `NsPerOp`). Pebble calibration: MapGet=708ns,
  MapSet=1785ns. SQLite=7000ns. Memory=500ns.
- **`OnTyped(eventType, handler)`** — bind a fold to an explicit CQRS event-type
  string, decoupling from the Go struct name.
- **Pebble engine `nextKey` regression test** (`metaengine/pebbleengine`) —
  pinning the exclusive-upper-bound helper behind every prefix scan, plus a
  concurrent `MapUpdate` test (100 goroutines) proving atomicity.
- **Metaengine → taskmanager integration** (`example/taskmanager`) — Counter ADT
  query (`task_counts_by_status`) with `/api/stats` endpoint via
  `projectionadapter`. First proof that metaengine works in a real CQRS app.

#### DuckDB analytical SQL backend

- **DuckDB analytical helpers** (`stack/duckdb.SQLViewModel`) — creates a real
  columnar view table backed by DuckDB, enabling server-side WHERE/ORDER BY and
  native GROUP BY / window-function aggregations. Integration test proves the
  OLAP path (revenue + avg-price per category) end-to-end. DuckDB is now a
  benchmarkable backend in `stack/bench` (`BenchmarkBenchkitSuite_DuckDB`,
  CGo-gated) and `cmd/cqrs-bench` (`--backend duckdb`, CGo-isolated so the CLI
  stays pure-Go otherwise).
- **DuckDB helpers** — `OpenDuckDB()`, `OpenDuckDBInMemory()`,
  `ConfigureDuckDBPool()`, `appendDuckDBOptions` unit test (6 cases), golden
  schema tests (4 tests), `TestMultiDBContract` in `stack/duckdb`.
- **ADR-0071** — documents the DuckDB CGo introduction decision and isolation
  strategy (`stack/duckdb/` is the only module requiring CGo; `//go:build cgo`
  on `drivers.go`).

#### Library adoption: otter, failsafe-go, testcontainers-go, go-snaps

- **Otter TinyLFU cache** (`decider/cache.go`) — replaced hand-rolled LRU
  (131→87 LOC) with `maypok86/otter/v2` TinyLFU cache. Same API surface; internal
  cache implementation changed.
- **Failsafe-go circuit breaker** (`middleware/circuit_breaker.go`) — replaced
  hand-rolled circuit breaker (243→175 LOC) with `failsafe-go/circuitbreaker`.
  Note: half-open semantics differ (limits trial executions to SuccessThreshold
  count, not unlimited). The `CircuitBreakerConfig` API is preserved.
- **testcontainers-go** (v0.43.0) — Postgres integration tests now use
  `testcontainers-go/modules/postgres` (postgres:16-alpine). Each test gets its
  own fresh database within a shared container for isolation. First time
  `storage/relational` Postgres tests ever ran. `stack/postgres` coverage went
  from 0% to tested.
- **go-snaps** (v0.5.23) — golden/snapshot testing adopted across `eventtest`
  (`AssertGolden`), `cattest`, catalog sub-packages, `otel`, `codec`. 38 golden
  files converted to `.snap` format. `snaps.Clean(m)` in TestMain for
  obsolete-snapshot cleanup across 16 modules. Update snapshots with
  `UPDATE_SNAPS=true go test ./...`.

#### Documentation & CI

- **Module count** — 58 → 60 `go.mod` files (added `stack/duckdb`,
  `metaengine/pebbleengine`). Verify: `find . -name go.mod -not -path './vendor/*' | wc -l`.
- **ADRs 0071–0074** — DuckDB CGo introduction, SQL pushdown, layout planning,
  Pebble engine.
- **CGo CI** — `CGO_ENABLED=1` added to 8 flake.nix test/test-race/verify apps;
  `pkgs.gcc` added to devShell for DuckDB CGo compilation.

### Fixed

- **`cqrs-lint init` SHOWSTOPPER** — the default preset generated
  `"exclude": []` (JSON array) but the `Exclude` config field is a `string`.
  Every new user's `.cqrs-lint.json` failed to load. Fixed: `"exclude": ""`.
  Regression test (`TestPresetConfigsLoadIntoAppConfig`) verifies all presets
  unmarshal into `AppConfig` without error. Reported by timesheets + Cyberdom.

- **Pebble engine `nextKey` + `MapUpdate`** (`metaengine/pebbleengine`) — the
  auto-commit daemon reverted the `nextKey` fix to the broken `slices.Backward`
  form THREE TIMES (range yields copies, so the increment was discarded and every
  prefix scan returned empty). Re-applied direct-index access each time and
  guarded `MapUpdate`'s read-modify-write with the engine mutex.
- **`TestRun_Postgres_Recovery` benchkit failure** — root cause: `populateSnapshots`
  writes +50 events; fixed with `SkipSnapshot: true`.
- **Metaengine declarative filter bug** — declarative `FilterOnField` filters were
  silently dropped in the closure fallback path. Fixed: both declarative and
  closure filters now apply.
- **`NewPebbleEngine(dir)` ignoring dir** — the disk-backed mode constructor
  silently used `vfs.NewMem()` instead of the provided directory. Fixed.
- **Dead deprecated error aliases** — removed 4 unused aliases
  (`ErrAggregateTypeMismatch`/`ErrAggregateIDMismatch`) from `storage/sql` and
  `storage/pebble`.
- **Data race in cross-engine tests** — `t.Parallel()` subtests had concurrent
  map/slice writes; fixed with `sync.Mutex`.
- **17 pebbleengine lint issues** — wrapcheck (9), gosec (2), makezero (1),
  modernize (1), prealloc (1), varnamelen (3). Resolved via targeted
  `.golangci.yml` path exclusion.
- **`decider/doc.go`** — corrected "LRU cache" to "TinyLFU cache" (otter).
- **Broken flake input** — daemon changed `cmdguard` ref to `v4.0.0`; the
  `github:` shorthand couldn't resolve the tag via SSH. Fixed `flake.lock`.

### Changed

- **SSE wire-format delegation to `go-sse`** (ADR-0097) — both SSE
  implementations now consume [`go-sse`](https://github.com/larsartmann/go-sse)
  v0.4.0 internally for wire-format serialization instead of reimplementing it:
  - `transport/http.SSEBroker` — `sse_event.go` shrank from 190→113 LOC.
    `SSEEventID` is now a type alias for `sse.EventID`; `NewSSEEventID`,
    `ParseSSEEventID`, `MustParseSSEEventID`, `WriteSSEEvent`,
    `WriteSSEHeartbeat`, and `WriteSSERetry` all delegate to go-sse. All
    external APIs unchanged (filter, transform, budget, backfill, OTel).
  - `metaengine.ServeSSE` — `sse.go` delegates event/heartbeat/replay writes to
    `sse.WriteEvent` / `sse.WriteHeartbeat`; headers via `sse.SetHeaders`.
    Watcher-based semantics preserved.
  - The two implementations remain **separate** (ADR-0091): different layers
    (event-bus-to-client vs collection-watch), different data models
    (`event.Event` vs `SeqValue[V]`). Only the wire-format serializer was shared.
  - New production dependency: `github.com/larsartmann/go-sse v0.4.0` (zero
    non-stdlib deps except `go-branded-id` + `go-error-family`).

- **Module extraction: `retry/` → `go-retry`** (ADR-0064) — the retry module
  is now a thin alias shim re-exporting `github.com/larsartmann/go-retry`.
  Backward-compatible: existing imports of `go-cqrs-lite/retry/v4` continue
  to work. The canonical code lives in the standalone repo.
- **Module extraction: `idempotency/` → `go-idempotency`** (ADR-0065) — the
  core idempotency types (`Store`, `MemoryStore`, `ErrDuplicate`) are now
  aliases re-exporting `github.com/larsartmann/go-idempotency`. The
  `kvstore/` and `sqlstore/` submodules remain local. Backward-compatible.
- **`command.Metadata` / `query.Metadata` are standalone structs** (ADR-0031) —
  no longer type aliases for `event.Metadata`. Each module owns its own shape
  (no Tombstone/Causation fields on command/query metadata). The JSON shape
  is identical to the previous alias, so serialized data is unaffected.

- **`storage/memory` read-path dedup** — extracted a generic `withReadLock[T]`
  helper that centralises the `wrapClosed` + `RLock` preamble for the three
  remaining read sites (`getEvents`, `ReadAll`, `ReadFrom`), mirroring the
  existing `withWriteLock` write-side pattern. Clone groups 34→19.
- **Dedup consolidation** — extracted `stack/sqlopt.OpenPrimaryBackend`
  (collapsing ~30 lines of identical postgres/sqlite openBackend control flow)
  and `catalog/eventcatalog.writeBuilderFile` (3 writer methods consolidated).
  `.art-dupl-baseline.json` updated. Full dedup triage at `-t 2` (48 groups)
  confirmed zero actionable extraction targets remain.
- **AGENTS.md updated** — metaengine canonical design docs marked, cqrs-lint
  description updated (159 rules, 10 categories), Pebble `slices.Backward`
  footgun documented, otter/failsafe-go adoption noted, `snaps.Clean` convention
  documented.

#### Flight recorder module (ADR-0089)

- **`flightrecorder/` module** — zero-dependency Go 1.25 `runtime/trace`
  wrapper. `Recorder` type with `New`/`Start`/`Stop`/`Enabled`/`Snapshot`/
  `SnapshotToFile`/`SnapshotIf`/`Reset`. Once-semantics (first trigger captures,
  rest no-ops). Thread-safe (`sync.Mutex`). Process-global (one active recorder
  per process; `ErrAlreadyEnabled` on double Start).
- **Trigger functions** — `OnLatency(d)`, `OnError()`, `OnErrorOrLatency(d)`,
  `OnAlways()`, `OnAny(...)`, `OnAll(...)`. Composable predicates.
- **Options** — `WithMinAge`, `WithMaxBytes`, `WithWriter`, `WithFile`.
  Configurable for slow/error/always capture.
- **CQRS middleware** — `CommandFlightRecorder`, `EventFlightRecorder`,
  `QueryFlightRecorder` in `middleware/`. Triggered on slow/error dispatch.
- **Decider integration** — `decider.WithFlightRecorder[State]` — captures on
  slow/error `Execute` calls.
- **Projection host integration** — `projectionhost.WithFlightRecorder` —
  captures on terminal worker failure.
- **Stack bundle integration** — `stack.WithFlightRecorder` — lifecycle
  management + discovery via `Bundle`.
- **Coverage:** 92.5%. 35 tests, `-race` clean. `io.Closer` on `Recorder` (fixes
  file handle leak). `ctx` pre-check in `Snapshot`/`SnapshotToFile`.
- **API surface:** 29 new symbols (3117→3161).

#### Benchkit evidence-grade metrics (ADR-0090)

- **7 new metric families** — statistical reliability (`RepeatStdDev`,
  `RepeatCoV`, `RepeatMean`, `RepeatIsReliable`), GC pause metrics
  (`GCMaxPause`), allocation metrics (`AllocsPerOp`, `BytesPerOp`), data
  integrity verification (`IntegrityErrors`), write amplification
  (`Disk.WriteAmplification`), cold/warm read distinction
  (`ColdReadLatency`), environment enrichment (`CPUModel`, `TotalRAMBytes`).
- **Derived rate metrics** — `GCPercent`, `TailRatio` (P99/P50 ratio)
  computed in `finalizeResult`.
- **Soak test drift** — `SoakSample.GCMaxPause`, `AllocBytes`;
  `SoakResult.GCMaxPauseDriftPct`, `AllocGrowthPct`. Memory-boundedness
  verification for sustained load.
- **Counter workload correctness** — `ErrMEEmptyCounter`, `ErrMEPointMiss`,
  `ErrMEEvent` sentinels. Benchmarks now assert non-empty counters (previous
  event-type mismatch bug silently measured empty stores).
- **Metaengine SQLite benchmark phase** — `phases_metaengine_sqlite.go`.
  Memory-vs-SQLite comparison with `MetaEngineSQLiteApplyThroughput`,
  `MetaEngineSQLiteScanLatency`, `MetaEngineSQLitePointReadLatency`.
- **PrintComparison expanded** — 11→13 columns (+TailR, +A/op).
- **API surface:** updated to 3162 exports.

#### Metaengine: Tier 4 expansion — new ADTs, engines, planner evolution

- **3 new ADTs** (ADR-0085) — `ADTVector` (k-NN similarity search,
  cosine/euclidean/dot), `ADTSearch` (full-text search, TF-IDF inverted index),
  `ADTSpatial` (geo range queries, haversine distance). Classification priority:
  Vector → Search → Spatial → Graph → Counter → Map → Set → Multimap → Log →
  SortedMap → Scan. Currently Memory-only (brute-force); future: DuckDB VSS,
  Postgres tsvector, PostGIS.
- **DuckDB metaengine engine** (`metaengine/duckdbengine/`) — MapBackend,
  CounterBackend, `LayoutColumnar`, `PushdownScan` (json_extract filter/sort
  pushdown). CGo required, separate module (ADR-0086).
- **Postgres metaengine engine** (`metaengine/pgengine/`) — MapBackend,
  CounterBackend, ScanBackend, `PushdownScan` (JSONB `->>` operator filter/sort
  pushdown), `LayoutPlanner` (expression indexes on JSONB paths). Pure Go (pgx
  driver, no CGo) (ADR-0087).
- **`ScanResult{Items []any; HasMore bool}`** — explicit `HasMore` contract
  across all 3 scan interfaces and all 5 engine implementations. Engines
  return at most `limit` rows; `HasMore` signals existence of more.
- **`RawScanResult{Items [][]byte; HasMore bool}`** — raw-bytes variant.
- **Postgres testcontainer tests** — `pgengine/testcontainer_test.go`. Shared
  `postgres:16-alpine`, per-test isolation. First time pgengine tested against
  real Postgres. ScanBackend tests (no filter, filter, sort+limit, keyset
  pagination). Batch `CounterIncrement` (N Exec → 1 multi-row
  INSERT...VALUES...ON CONFLICT).
- **DuckDB ScanBackend tests** — 4 sub-cases verified with CGo.
- **5 engines, cross-engine parity** — Memory, SQLite, Pebble, DuckDB,
  Postgres. `adttest.RunMatrix` extended to 10 ADTs (Vector, Search, Spatial
  added). Auto-skip for unsupported backends via reflect-based interface
  checking.

#### Metaengine: planner rule pipeline + materialize-vs-replay (ADR-0083)

- **`PlanRule` interface + `RulePipeline`** — composable planning rules.
  `planner.go` dissolved from 279→226 lines. 4 extracted rules: `schemaRule`,
  `layoutRule`, `writeAmpRule`. Rules applied sequentially after engine
  assignment.
- **Statistics + materialize-vs-replay** — `WithWorkloadStats`,
  `ReplayCost(stats)`, `MaterializeCost(stats)`, `ShouldMaterialize(stats)`.
  The ES-specific killer feature: advisory INFO/WARN diagnostic in `PlanResult`
  when materialization is cheaper than replay. Cost formula:
  `replay = read_rate × stream_length × fold_cost`;
  `materialize = write_rate × fold_cost + read_rate × query_cost`.
- **`StorageLayout` + cost matrix** — `Layout{Row, Columnar, LSM, KV}`,
  `(ADT × Layout) → Complexity` mapping, `EngineProfile.Layouts`,
  `RuleTrace`. Engines declare their physical layout; planner matches ADT
  requirements to engine capabilities.
- **`SerializablePlan`** — JSON-serializable `PlanResult` for diff/pin/round-trip
  testing. `Serialize(result, engines)` + `Deserialize`.
- **`VersionedStorage`** — temporal queries (`ExecuteAsOf`) on Memory engine.
  Version chains + binary search for point-in-time reads. Property-based test
  (100 rapid iterations, reference model).
- **`workloadMeter` / `poisonTracker` / `idempotencyTracker` / `subscriberHub`**
  — Store god-object decomposed into focused collaborators (17→13 fields).

#### Metaengine: data model refactor (Fold sealed interface)

- **`Fold` sealed interface** — 12 concrete unexported fold types replace the
  11-field `any` god-struct. Zero nil-panic risk (`reflect.Value` captured once
  at construction). `applyFold` dispatch is a type switch (zero
  `reflect.ValueOf` per event). 11 `callXxx` reflect helpers deleted.
- **`queryRuntime` deleted** — merged into `QueryDecl[Q,R]` with 3 unexported
  runtime fields. `queryMeta` interface gained `assignPlan()`/`setEngine()`.
- **Enum validation** — all 6 enum families (`ADT`, `StorageLayout`,
  `FilterOp`, `CursorKind`, `EngineKind`, `SortDirection`) have `Valid()` +
  registries. `AllStorageLayouts()`, `AllADTs()` helpers.
- **Branded unit types** — `NsPerRead`, `NsPerWrite`, `ByteSize` defined and
  wired into `planQuery()` via `Valid()` checks.
- **`ApplyError`** — structured error type for fold failures, wired into
  `applyFold()` via deferred wrapping.
- **README "Internal Architecture" section** added.

#### Metaengine: pgengine + duckdbengine PushdownScan

- **pgengine PushdownScan** — `FilterOnField`/`SortOnField` push
  WHERE/ORDER BY/LIMIT into Postgres via JSONB `->>` operators. Partial
  expression indexes (`LayoutPlanner`) for indexed JSONB paths.
- **duckdbengine PushdownScan** — `json_extract()` filter/sort pushdown for
  DuckDB. JSON stored as VARCHAR (columnar optimizations not yet leveraged).
- **Documentation overclaim fixes** — pgengine/duckdbengine doc.go and scan.go
  comments corrected to match actual capabilities.

#### Backend tradeoff framework

- **`DurabilityTier`** — unified vocabulary: `DurabilityStrict` (fsync per
  commit), `DurabilityNormal` (safe against app crash), `DurabilityRelaxed`
  (data loss possible on crash). Translated to per-backend pragmas:
  SQLite `synchronous`, Postgres `synchronous_commit`, Pebble `DisableWAL`.
  Default: `DurabilityNormal`.
- **`Capabilities`** — machine-checkable tradeoff matrix:
  `Persistent`, `Embedded`, `Distributed`, `OLAP`, `CGoRequired`, `SyncEnabled`.
  Every stack preset implements `Capabilities()`.
- **Mixed workload benchmark** — `BenchmarkMixedWorkload_ReadsDuringWrites`
  phase. Concurrent reads + writes for realistic contention profiling.
- **Backend options** — `sqlite.WithCacheSize`, `sqlite.WithBusyTimeout`,
  `postgres.WithPoolSize`, `postgres.WithStatementTimeout`.
- **`BACKEND_TRADEOFFS.md`** — 228-line guide documenting per-backend
  tradeoffs (durability, performance, embeddability).
- **5-backend benchmark** — `docs/benchmarks/2026-07-31_backend-comparison.md`.
  Pebble: 100K/s writes (7x SQLite, 50x DuckDB). Postgres: 137µs P50 reads,
  387x speedup with `synchronous_commit=off`. DuckDB: OLAP not OLTP.

#### cqrs-lint: A033 + C037 (179 → 181 rules)

- **A033** — branded-ID string roundtrip detection. Flags code that converts a
  branded `id.Of[T]` to `string` and back, which breaks compile-time type
  safety. 4 tests.
- **C037** — snapshot/event codec mismatch. Detects when a snapshot store and
  event store use different codecs (e.g., CBOR events + JSON snapshots), which
  causes deserialization failures on load. 5 tests.
- **Self-lint clean** — both rules fire 0 findings on the linter's own code.

#### cqrs-lint: block-level suppression (ADR-0088)

- **`//cqrs-lint:ignore-start` / `//cqrs-lint:ignore-end`** — suppress findings
  across a range of lines instead of per-line. Pairs with existing
  `//cqrs-lint:ignore <rule-id>` single-line suppression.
- **Stale block detection** — `DetectStaleSuppressions` flags
  `ignore-start`/`ignore-end` pairs where no finding fires between them
  (the code was fixed but the suppression wasn't removed).
- 5 tests covering start/end pairing, stale detection, nested blocks.

#### Pebble sort index (1,233x speedup)

- **Sort index implementation** — `'o'` prefix key structure for sort fields.
  `writeIndexEntries`/`deleteIndexEntries` maintain the index atomically.
  `scanWithSortIndex` iterates in sort order with cursor pagination and early
  termination. 9 tests, race-detector clean.
- **Benchmark:** 8,145µs → 6.6µs (1,233x speedup) for sorted scans.
- **Numeric range filter fix** — `encodeIndexValue` with sign-offset to uint64
  domain + `%020d` zero-pad ensures lexicographic ordering matches numeric
  ordering for ALL integers (negative, mixed-digit, max/min int64).

#### Verify gate repair: Stale GREEN → actual GREEN

- **Soak test stabilization** — `race_on.go`/`race_off.go` build-tag constants
  for `-race`-inflated thresholds (10x under race detector). Verified 3x with
  `-count=3 -race`. Benchkit soak test skip heap leak assertion when <5 iterations.
- **15 pre-existing lint issues fixed** across 6 modules (stack/sqlopt
  exhaustive+wrapcheck, stack/memory exhaustruct, stack/sqlite exhaustruct+mnd,
  stack/duckdb exhaustruct, storage goconst, storage/sql goconst, benchkit
  gocognit+nilerr+modernize).
- **ADR numbering collision resolved** — two files numbered 0081; renamed
  store-redesign to 0082. Cross-references updated.
- **First clean verify gate** — EXIT:0, 0 test failures, 0 lint issues, 56
  modules, 1105 doc-check refs, API stability passed.

#### MySQL polish session 2

- **Testcontainer privilege fix** — `ctr.Exec` for root GRANT instead of
  host-side DSN string replacement. Reliable, no `caching_sha2_password` issues.
- **Multi-statement DDL fix** — `splitMySQLDDL()` via `strings.SplitSeq` avoids
  `multiStatements=true` DSN (SQL-injection risk).
- **CHANGELOG entry** for MySQL/MariaDB support.

#### Documentation: ADRs 0080–0090

- **ADR-0080** — Dialect interface upsert methods (MySQL support)
- **ADR-0081** — Metaengine runtime casts
- **ADR-0082** — Metaengine store redesign analysis
- **ADR-0083** — Metaengine planner rule pipeline
- **ADR-0084** — Metaengine layered architecture
- **ADR-0085** — Metaengine new ADTs (Vector/Search/Spatial)
- **ADR-0086** — Metaengine DuckDB engine
- **ADR-0087** — Metaengine Postgres engine
- **ADR-0088** — Block-level suppression
- **ADR-0089** — Flight recorder
- **ADR-0090** — Benchkit evidence-grade metrics
- **ADR-0091** — SSE consolidation decision (keep `metaengine.ServeSSE` and
  `transport/http.SSEBroker` separate — they serve different layers)

#### Metaengine: DuckDB LayoutPlanner, dead code wiring, reification tracking

- **DuckDB LayoutPlanner** — `LayoutPlanApplier` interface + `WithColumnarLayout`
  query option. Reflection-derived column planning, type coercion
  (`coerceForColumn`), and aggregation support. Columnar view table enables
  server-side WHERE/ORDER BY on DuckDB.
- **Dead code wiring** — branded unit types (`NsPerRead`, `NsPerWrite`,
  `ByteSize`) have `Valid()` called in `planQuery()`. `ApplyError` wraps fold
  failures in `applyFold()` via deferred error wrapping.
- **Exhaustiveness guard** — `TestApplyFoldExhaustiveness` verifies all 12 fold
  types are handled: count check against `AllFoldKinds()` + mirror type switch
  with `default: t.Fatalf` catches unhandled new fold types.
- **Reification failure tracking** — `workloadMeter.IncReificationFailure()` /
  `ReificationFailures()` surface type mismatches between planned value type and
  engine stored shape. Non-zero values indicate an engine or planning bug.
- **gocritic fix** — single-case type switch in `duckdbengine/layout_planner.go`
  rewritten to `if v, ok := value.(string)` idiom.
- **10M memory soak test** — `TestSoak_MemoryBounded_10M` verifies O(keys) heap
  bound: 10M events into 1000 keys → 0.1 MB heap growth, flat growth curve.
  Skippable via `SOAK_SKIP_10M=1`.

#### cqrs-lint: C037 expansion, D007 auto-fix, config presets

- **C037 scope expansion** — now detects codec mismatches across all typed
  stores: snapshot, command, query, and kv (previously snapshot/event only).
  Tests cover each store type independently.
- **D007 `--fix` support** — auto-fix transforms `event.NewEvent(` →
  `event.New(` at call sites where both constructors are used inconsistently.
  Uses `go-finding` builder-based `FixStrategyDirect` with before/after code.
- **Config presets** — `init --preset` generates `.cqrs-lint.json` from named
  presets: `local-cli`, `library`, `server`, `full-stack`. Each tailors the
  feature profile and disabled rules for a common project type.
- **MySQL testcontainer retry** — `waitForMySQLReady` deadline-bounded polling
  (500ms interval, 10s timeout) replaces fragile host-side DSN manipulation.

#### Metaengine: replication model (ADR-0093) — DDIA Ch5 foundation

- **EngineProfile replication fields** — `Replication` (none/single-leader/
  multi-leader/leaderless), `ReplicationLag` (staleness, diagnostic-only),
  `NetworkRTT` (additive latency). All current engines are `ReplicationNone`
  (zero value). Foundation for future distributed engines (Iroh, CockroachDB).
- **`replicationRule`** — emits INFO diagnostic when routing to a replicated
  engine with non-zero lag. `mapUpdateReplicationRule` emits WARN when
  Map ADT with update folds is routed to a replicated engine.
- **CollectionInfo exposure** — `store.Collections()` now includes
  `Replication`/`ReplicationLagMs`/`NetworkRTTMs` per collection.
- **`store.ReplicationMode(queryName)`** — returns the topology for a single
  query.
- **Plan options** — `WithReplication(r)` and `WithNetworkRTT(d)` override
  engine profiles for "what-if" cost analysis.
- **SerializablePlan** — includes `Replication`/`ReplicationLagMs`/
  `NetworkRTTMs` per query.
- **ExplainPlan / Doctor** — replication suffix on engine lines; Doctor has a
  `--- Replication ---` section.
- **EngineProfile.String()** — readable format: `iroh-sync: map@O(1)
(replication=leaderless, lag=200ms, rtt=5ms)`.

#### Metaengine: Universal ADT Phase 3 (ADR-0094) — 10/10 ADTs on all engines

- **`DegradedADTs`** — engines now declare support for ALL 10 ADTs. Non-native
  ADTs run in O(N) degraded mode (full scan + filter). Eliminates
  `ErrUnsupportedADT` — every query routes to the best available engine.
- **`degradedADTRule`** — SCREAM diagnostics: emits WARN when a query is routed
  to a degraded ADT on an engine, including estimated cost-at-scale.
- **All 5 engines extended to 10/10 ADTs** — Memory (native all), SQLite, Pebble,
  DuckDB, Postgres now support Vector/Search/Spatial/Graph/Set/Multimap/Log
  via degraded fallback where not natively implemented.

#### Metaengine: replication Phase 2 polish

- **Visibility→Replication rename** — committed code used rejected "Visibility"
  naming (`31f26b8c`); fully rewritten to `Replication`/`NetworkRTT`/
  `ReplicationLag` per DDIA Ch5 model. `visibility.go` deleted.
- **Redundant diagnostic removed** — replicationRule and mapUpdateReplicationRule
  had overlapping messages; consolidated.
- **`EngineProfile.String()` pre-allocated** — avoids string concatenation
  allocations.
- **5 replication tests** — replicationRule WARN/INFO, mapUpdateReplicationRule,
  EngineProfile.String(), ExplainPlan output, Doctor output.

#### Metaengine: WatchTyped, SSE reconnect, boundary validation, calibration

- **`WatchTyped[V]` / `WatchTypedWithSeq[V]`** — typed watcher convenience
  functions. Returns `chan V` instead of `chan any`, eliminating the need for
  engine-specific reification at call sites.
- **SSE reconnect with SQLite reify fallback** — end-to-end test verifying
  `ServeSSE` replay works with the SQLite-backed `WatchWithSeq` path after the
  watcher reification fix.
- **`ErrKeyTypeMismatch` at Store boundary** — `Store.Execute`/
  `ExecuteTyped` now validates that the input struct's key field type matches
  the declared `keyType`. Catches type mismatches before dispatch.
- **CalibrateEngine copy-discard bug fixed** — `reliability.go` was discarding
  the calibrated values. Rewritten with `calibratable` interface; test rewritten
  to verify values persist.

#### Metaengine: execute.go refactor

- **Point-lookup and membership helpers extracted** — `lookupQuery` helper
  consolidates shared query lookup logic. Module dependencies promoted to
  v4.3.0.

#### cqrs-lint v4.3.0 — 185 rules, TLS detection, config features

- **4 new rules** (181→185): C038 (fold-case collection detection; rewritten
  post-v4.3.0 — see Unreleased section above), C039
  (consistency receiver-method guard), S011 (PII without encryption), D017
  (nullable pointer fields in event payloads).
- **TLS-aware server detection** — `projectCallsMethodOnType` detects
  `tls.Config`, `ListenAndServeTLS`, `http.Server.TLSConfig`. Eliminates false
  positives on TLS-enabled servers.
- **`ConfigFeatures` override** — consumers can override detected features via
  config or CLI flags. Resolves feature-profile misdetection.
- **C008 struct-level ignore** — `c008-ignore-fields` (case-insensitive) and
  `c008-ignore-structs` (skip entire structs) config options.
- **E016 narrowed + F015 gating** — E016 (missing health checks) now checks for
  `cqrshtmx.HealthHandler` and alternative endpoints. F015 gated on
  `StoreSQLite` feature profile.
- **Version management** — `TestVersionMatchesLatestTag` CI gate. `version
--verbose` shows build date + commit hash. `changelog` subcommand. `ldflags`
  version stamping in Nix build.
- **Config presets** — `init --preset {local-cli|library|server|full-stack}`.
  Each tailors feature profile and disabled rules for a common project type.
- **`--adoption` flag** — separate F-level adoption suggestions from health
  score deduction.
- **Transport feature flags** — `TransportHTTP`/`TransportGRPC` detection +
  `ServerLocal` heuristic for local-only servers.
- **Version-tag drift guard** — CI verifies cqrs-lint version constant matches
  the latest `cmd/cqrs-lint/v*` tag.
- **`scripts/bump-cqrs-lint.sh`** — automated version bump for downstream
  SystemNix integration.

#### Nix-based integration test infrastructure (ADR-0095)

- **Ephemeral PG** (`nix run .#integration-pg`) — starts a `pg_ctl` process
  from nixpkgs in a temp dir, runs all PG integration tests, cleans up. No VM,
  no Docker. Fast (~3s startup). Works on macOS.
- **NixOS VM tests** — `nix run .#integration-pg-vm` and `nix run
.#integration-mysql-vm` boot QEMU VMs with `services.postgresql` /
  `services.mariadb`. VM tests live in `nix/vm/postgres.nix` + `nix/vm/mysql.nix`.
- **CI integration** — `nixos-vm-tests` CI job runs the VM tests. Parallelized
  PG + MySQL VM tests.
- **`integration-all` / `verify-integration`** — nix apps that run all
  PG+MySQL tests or the integration gate only.
- **AGENTS.md + CONTRIBUTING.md** — testing-guide decision matrix updated
  with Nix-based approaches.

#### ADR review findings (ADR-0096)

- **ADR-0096: Iroh distributed engine bridge evaluation** — evaluates CGo FFI
  vs sidecar approaches for bridging Iroh (Rust CRDT) into the Go metaengine.
  Documents maturity assessment: `iroh-docs` NOT in C FFI, blocks direct
  integration. PN-Counter via Iroh identified as the killer feature.
- **SSE three-repo finding** — discovered `go-sse` exists as a standalone
  library. `go-cqrs-lite` reimplements SSE wire format in two places. ADR-0091
  rationale needs revisiting. SSE refactor deferred to TODO_LIST.
- **Benchmark trust deficit** — 29 of 43 benchmarks discard results. DuckDB/
  Postgres cost constants hand-picked with zero empirical backing. Flagged as
  "highest-leverage next move." → TODO_LIST.

### Fixed

#### Metaengine watcher reification and delete notifications

- **Watcher delete notifications no longer silently dropped** — `Watcher[V]`
  now delivers the zero value of `V` on `Remove[V]()` folds instead of dropping
  the notification. The old path used `nil.(V)` which always panicked and was
  recovered into a silent drop, so consumers never saw deletes.
- **Cross-engine watcher reification** — `reifyWatcherValue[V]` handles three
  cases: typed Go values (Memory engine), nil deletes, and engine-specific
  representations such as `map[string]any` (SQLite/Postgres/DuckDB JSON decode)
  and raw `jsonValue` (pushdown paths). This eliminates the silent type-
  assertion failures that could cause lost events in the replay journal and
  SSE reconnect path.
- **`replayShim.recordValue` uses reification** — the replay journal no longer
  records seq=0 when a SQL engine returns a different representation than the
  watcher type `V`.
- **Regression tests** added for memory, SQLite, DuckDB, Postgres, and Pebble
  engines covering both delete notifications and `WithReplay` typed-value
  capture. Added a `jsonValue` fast-path test for `reifyWatcherValue`.
- **Documentation** updated in `metaengine/README.md` and `metaengine/COOKBOOK.md`
  explaining delete-notification semantics and cross-engine value
  representation.

## [v4.2.0] — 2026-07-27

### Added

- **Coverage-drift checker** (`scripts/check-coverage.sh` + `nix run
.#check-coverage`) — mechanically detects when actual module coverage drifts
  from the numbers documented in AGENTS.md. Resolves the 4-session
  "coverage-verification gap" pattern where coverage claims were trusted from
  prior reports instead of re-measured. Supports `--update` to recompute and
  print the AGENTS.md-ready values. ±2% tolerance for refactor noise.

- **cqrs-lint: 3 new rules (now 65 total)** —
  `C015` (unchecked `Close()` — resource leak detection, flags bare
  `x.Close()` statements and `_ = x.Close()` assignments),
  `C016` (`context.Background()`/`context.TODO()` in handlers — flags detached
  contexts inside functions that receive a `context.Context` parameter, which
  discards the caller's cancellation, timeouts, and tracing),
  `D006` (missing `errorfamily.New*` — flags `errors.New` and `fmt.Errorf`
  without `%w` in production code, which bypasses the 6-family error taxonomy;
  package-level sentinel `var ErrXxx = errors.New(...)` declarations are
  exempt because they are matched by `errors.Is`, not classified).

- **CI gates expanded** — `.github/workflows/ci.yml` now runs 4 additional
  quality gates: `#check-api-stability`, `#check-duplication`,
  `#check-layers`, `#check-coverage`. These existed as local nix apps but were
  never wired into CI — the red cqrs-lint gate hidden for 3+ sessions is the
  proof that local-only checks rot silently.

- **wrapClosed consolidation** (`storage/memory`) — extracted `withWriteLock`
  and `withReadLock[T]` helper pairs across `store.go`, `command_store.go`,
  `query_store.go`, and `snapshot.go` (12 of 17 sites). Clone groups dropped
  34 → 19. Remaining: `checkpoint.go` (2) + `store_load.go` (3), same pattern.

- **Property-based tests** — `kv/property_test.go` (6 rapid tests for
  TypedStore + Cache round-trip, concurrent Set/Get, TTL expiry),
  `snapshot/property_test.go` (4 rapid tests for TypedStore save/load/version
  round-trip), `metaengine/cross_engine_adt_test.go` (Counter + Set parity
  across Memory vs SQLite engines). SortedMap parity deferred — see
  [TODO_LIST.md](TODO_LIST.md) "Module Health & Tooling".

- **Testing and release documentation** — `docs/testing-guide.md` (comprehensive
  testing patterns: table-driven, BDD/Ginkgo, property-based/rapid, scenario
  DSL, race-aware thresholds, golden files, coverage goals) and
  `docs/release-checklist.md` (step-by-step release process: tag-release.sh,
  batch-release.sh, golden regen, verify gate, push sequence).

- **CBOR→JSON transcoding helpers** — `codec.TranscodeToJSON(payload, enc)` and
  `transport/http.CBORToJSONTransform`. A schema-free, ready-made bridge for
  consumers that store events in CBOR but must serve JSON to browsers via SSE.
  Deletes the per-consumer transcode logic (~50 LOC) that every compact-codec
  deployment otherwise duplicates. The transform receives raw payload bytes +
  the event's encoding stamp; non-CBOR payloads pass through unchanged (zero
  overhead); decode/encode failures fall back to the raw payload and log at Warn
  (graceful degradation, ADR-0070 documents the slog.Default vs OTel counter
  decision). `CBORToJSONTransform` is the one-liner for `WithPayloadTransform`.
  Also fixes the `WithPayloadTransform` doc example, which previously swallowed
  errors (`jsonBytes, _ := json.Marshal(p)`).
- **Metaengine module** (`metaengine/v4`) — cost-based storage planner for
  event-sourced data. Derives projections and engine assignments from two
  primitives: Events (mutations) and Queries (read intent). 7 ADTs inferred
  from fold return types (Map, Set, Counter, Graph, SortedMap, Multimap, Log).
  Typed `FilterOn`/`SortOn` closures, cursor-based pagination, formal cost model,
  write amplification budget. SQLiteEngine shipped (ADR-0061); projection adapter
  integrated (ADR-0062); Phase 2 SQL pushdown deferred (ADR-0063). Zero production
  dependencies in core. 174 BDD specs, 86.2% coverage (verified 2026-07-27).
- **Benchkit module** (`benchkit/v4`) — factory-driven benchmarking suite with
  7 named workload profiles (Dev, Small, Medium, Large, Stress, WriteHeavy,
  ReadHeavy) plus an analytical profile, 9-phase runner (setup → warmup → write
  → read → readmodel → projection → durability → rawsink → teardown), concurrent
  workers, latency percentiles, resource sampling, codec-aware payload sizing,
  errorfamily error classification, SkipPhases, Config validation. 88 benchkit
  - 12 CLI test functions (`-race`). First real benchmark run executed across
    memory/pebble/sqlite — see
    [benchmark results](docs/status/2026-07-24_17-54_benchmark-first-real-run.md).
    Full feature detail in [FEATURES.md](FEATURES.md).
- **cqrs-bench CLI** (`cmd/cqrs-bench`) — benchmark any backend with named
  workload profiles. `run`, `compare`, `sweep`, and `--repeat N` subcommands.
  Uses `runtime/debug.ReadBuildInfo()` for version (was hardcoded `v4.1.0`).
- **Incremental rollups** (`storage/relational`) — `ProjectionSink.Increment`
  for atomic counter maintenance via `INSERT ... ON CONFLICT DO UPDATE`.
  `RelationalProjection.Reset` implements `projectionhost.Resettable` for
  zero-based replay. 11 tests.
- **example/readme-quickstart** — compile-verified Quick Start example testing
  every API pattern from the main README.
- **Error taxonomy migration** — 13 `errors.New` sentinels migrated to
  `errorfamily.New*` constructors across 7 modules (codec, decider, schema,
  middleware, catalog, prometheus, stack/postgres). 6 previously-unexported
  sentinels now exported. All external sentinels classified (e.g.
  `pebble.ErrNotFound` → Rejection).
- **Aggregate→Stream rename** (ADR-0058) — identity types renamed from
  `Aggregate*` to `Stream*` (`StreamID`, `StreamType`, `StreamRef`, `StreamMarker`)
  across `id/`, `event/`, `command/`, `listing/`, `otel/`, `storage/`. Deprecated
  type aliases preserve backward compatibility. Wire formats (JSON tags, SQL
  columns, proto fields) preserved.
- **Comprehensive README coverage** — 24 new module READMEs created, 9 existing
  rewritten, 19 code example bugs fixed. All 58 modules with go.mod have READMEs.
  248 Go symbol references verified by `doc-check`.

### Fixed

- **cqrs-lint build break (hidden red gate)** — the auto-commit daemon bumped
  `go-output` root to v0.33.0 (commit 85ac81f1), but `go-output/table` has no
  v0.33.0 release (maxes at v0.32.0). The v0.33.0 root removed `NewTableBuilder`,
  `Table`, and `RegisterTableMarshaler`, breaking `cmd/cqrs-lint` entirely. The
  verify gate's "GREEN" claim across 3+ sessions was stale: `go test` failed for
  `cmd/cqrs-lint` in both workspace and `GOWORK=off` modes. Downgraded
  `go-output` back to v0.32.0. Gate is now genuinely green (verified exit 0).
- **cqrs-lint lint fixes** — `main.go` struct tag had 3-space gap (golines) and
  non-alphabetical tag order (tagalign); both were masked by the build break.
- **AGENTS.md coverage drift (4-session pattern resolved)** — coverage claims were
  trusted from prior reports and had drifted: dispatcher claimed 98.0% (actual
  81.5%), id claimed 97.6% (actual 86.4%), codec claimed 76.0% (actual 70.2%),
  decider claimed 98.3% (actual 95.9%), event claimed 91.3% (actual 88.3%). All
  numbers re-verified via `go test -cover` (workspace mode, `goexperiment.jsonv2`
  tag) and corrected in AGENTS.md with a "verified 2026-07-27" citation.
- **`idempotency/kvstore.Record` no longer extends the TTL on an existing key.**
  `Record` now uses `SetIfAbsent` instead of `Set`, making it a no-op when the
  key already exists (the expiry is not refreshed). This aligns the KV store
  with the documented `idempotency.Store` contract shared by `MemoryStore` and
  the SQL store. Previously, a retried `Record` call silently extended the
  dedup window; consumers relying on at-least-once delivery could see a longer
  dedup window than requested. Behavior change: bug fix toward contract.
- **`stack/pebble` disk-usage metric** (`safeInt64`) now clamps `uint64→int64`
  to `math.MaxInt64` instead of wrapping to a negative value on overflow.

### Added (Pareto execution plan — consumer trust + production maturity)

- **Consistency model document** (`docs/CONSISTENCY_MODEL.md`) — documents
  single-process scope, write→read eventual consistency, projection lag,
  read-after-write patterns, and bounded-staleness semantics. The #1 doc gap
  for consumers reasoning about read correctness.
- **SQL-backed idempotency.Store** (`idempotency/sqlstore/`) —
  `NewSQLiteStore` and `NewPostgresStore` implementing `idempotency.Store`
  via `INSERT ON CONFLICT DO NOTHING` for exactly-one-winner dedup. Includes
  TTL sweep and concurrent race tests. The #1 horizontal-scaling blocker resolved.
- **WaitForVersion helper** (`decider/`) — polls `store.LoadFromVersion` until
  the target version is visible or a deadline hits. Default 2s timeout, 10ms
  poll interval. Enables read-your-writes consistency in request/response flows.
- **CheckStaleness / WithMaxStaleness** (`projectionhost/`) — projection read
  option that rejects/flags reads whose projection lag exceeds a threshold.
  Wired into `Host.LagDuration()` check.
- **Metaengine SQLite engine** (`metaengine/`) — `SQLiteEngine` wrapping
  `storage/view.SQLViewStore` as a metaengine backend. First production engine
  validates the interface design. Cost-based engine selection between Memory
  and SQLite (ADR-0061).
- **Metaengine projection adapter** (`metaengine/projectionadapter/`) — adapter
  implementing `projection.Projection` so a metaengine Store can be registered
  with `projectionhost.Host`. Integration tested with full host lifecycle
  (ADR-0062).
- **Metaengine cost calibration** (`metaengine/`) — `EngineProfile.NsPerOp`
  field replaces arbitrary `nsPerOp=100` constant with benchmark-driven numbers.
  Memory=500ns, SQLite=7000ns (14x ratio). Calibration benchmarks measure
  per-engine per-op cost.
- **Store.EventTypes()** (`metaengine/`) — returns sorted unique event types
  from registered queries' fold maps. Enables integration adapters to declare
  event interests without depending on event-sourcing packages.
- **FilterOn/SortOn pushdown ADR** (ADR-0063) — decision: Phase 1 keeps
  in-memory closures + adds `PushdownScan` interface seam (zero breaking
  change). Phase 2 defers declarative `FilterSpec`/`SortSpec`.
- **Module extraction ADRs** (ADR-0064, ADR-0065) — design for extracting
  `retry/` → `go-retry` and `idempotency/` → `go-idempotency` as standalone
  repos with re-export aliases for backward compatibility.
- **NATS transport design doc** (`docs/planning/nats-transport-design.md`) —
  JetStream stream configuration, durable consumers, subject mapping, and
  CatchUpSubscriber integration recipe via the existing `watermill/` bridge.
- **Parquet journal design doc** (`docs/planning/parquet-journal-design.md`) —
  Phase 1 design for `storage/parquet` segment-based SeekableJournal using
  pure-Go `parquet-go`. Columnar compressed archival with 5-10x compression.

### Fixed (Pareto execution plan)

- **flake.nix testModules gap** — added `metaengine`, `metaengine/projectionadapter`,
  `retry`, `idempotency/kvstore`, `idempotency/sqlstore`, `cmd/api-stability`, and
  `cmd/doc-check` to CI test module list. These modules were silently untested in CI.
- **Module count** — 56 → 58 `go.mod` files (added `metaengine/projectionadapter`
  and `idempotency/sqlstore`). All three formerly-untagged modules
  (`metaengine`, `metaengine/projectionadapter`, `idempotency/sqlstore`) are now
  tagged; `metaengine/projectionadapter/v4.0.0` is orphaned (points to a commit
  not in HEAD) and needs re-tagging — see [TODO_LIST.md](TODO_LIST.md).

### Fixed (benchkit hardening session)

- **SQLite concurrent-write failure (SQLITE_BUSY)** — `stack/sqlite/preset.go`
  was missing `storage.ConfigureSQLitePool(sqlDB)` after WAL enable. SQLite now
  handles 4+ goroutines correctly (was limited to Concurrency=1).
- **Compare-mode disk always 0B** — `compareCmd` discarded per-backend disk
  paths. New `compareWithDiskPaths()` injects `DiskPath` so disk columns
  populate in comparison tables.
- **`--version` hardcoded** — Replaced hardcoded `v4.1.0` with
  `runtime/debug.ReadBuildInfo()` + VCS revision fallback.
- **DiskSizer interface was dead code** — Implemented 3-layer DiskSizer:
  `storage/pebble.Backend.DiskUsage()` (computed from Metrics level sizes + WAL),
  `stack.WithDiskSize()` option + `Bundle.DiskSize()`, wired in `stack/pebble`
  preset. `durabilityPhase` checks `>= 0` before using, falls back to filesystem.
- **CPU measurement returned n/a for fast benchmarks** — Replaced
  `/proc/self/stat` parsing (10ms tick resolution) with `syscall.Getrusage`
  (microsecond resolution). Split into `cpu_unix.go` (`//go:build unix`) and
  `cpu_other.go` (`//go:build !unix`).
- **Projection benchmark showed 0 events** — Added polling loop (10ms ticker,
  30s deadline) in `projectionPhase` so workers process events before `Stop()`.

### Added (benchkit hardening session)

- **DiskSizer interface** — `stack.WithDiskSize(fn)` option + `Bundle.DiskSize()`
  method. Pebble preset wires `backend.DiskUsage()` automatically. Returns -1
  when not registered; runner falls back to filesystem walk.
- **Mixed payload-size distributions** — `NewMixedGenerator(seed, sizes, codec)`
  picks a size uniformly at random per event. CLI flag `--payload-sizes
64,256,4096`. Result reports mean + full distribution. See
  [scaling report](docs/status/2026-07-24_19-30_event-size-scaling-benchmark.md).
- **Projection benchmark phase** — Projection catch-up throughput now measured
  in default profiles (was always 0). Polls until all events processed before
  reporting.
- **ADR-0060** — Documents 5 benchkit design decisions: codec-aware padding,
  warmup isolation, ReadRatio-as-passes, SkipPhases, DiskSizer -1 sentinel.

### Added (benchkit — full benchmark suite)

> Post-hardening sessions completed the full benchmark evidence plan. All items
> below shipped unreleased; see [TODO_LIST.md](TODO_LIST.md) "Benchkit" for the
> one remaining open item (`benchkit/v0.1.0` tag).

- **Durability / recovery phase** (`Config.Recovery`) — closes the bundle, reopens
  it via the factory, and reloads all streams. Reports `Result.RecoveryTime` and
  `RecoveredEvents`. CLI: `--recovery`.
- **Production replay phase** (`Config.ReplayOnly`) — skips writes, discovers
  streams from `Journal`/`SeekableJournal`, and benchmarks reads + projections
  on existing data. CLI: `--replay`.
- **`benchtest.RunSuite`** — `RunSuite(b *testing.B, config, factory)` wraps the
  benchkit pipeline into a Go `testing.B` with `b.ReportMetric`. Wired into
  `stack/bench` with 3 backend suites.
- **Analytical profile** — `ProfileAnalytical` (10K streams, 90% reads, 5x journal
  scans) + `Profile.JournalScans` field for multi-pass journal scanning.
- **Postgres backend** — `postgres` added to `cqrs-bench`; benchkit tests skip
  without `POSTGRES_TEST_DSN`.
- **kv.Store projection handler** — projection phase exercises a real `kv.Store`
  (Get+Set per event on `bundle.ReadModels`); falls back to an atomic counter when
  no kv.Store is available.
- **Scaling sweeps** — `WorkerSweep`, `BatchSizeSweep`, `StreamLengthSweep`,
  `GOMAXPROCSSweep` for systematic parameter exploration. CLI: `sweep` subcommand.
- **benchstat output** — `WriteBenchstat` emits benchstat-compatible lines for
  statistical comparison across runs/backends.
- **Suite manifest** — `WriteManifest` serializes config + environment + result as
  JSON for reproducibility.
- **JSON schema stability** — `Result.SchemaVersion` + `ExpectedJSONFields` /
  `VerifyJSONFields` guards against silent result-schema changes.
- **CPU profiling** — `--cpuprofile file` and `--memprofile file` emit pprof output.
- **CI workflow** — profiling hooks and a benchmark interpretation guide
  (`docs/benchmarking/`).

### Added (metaengine hardening + error-wrapping convention — 2026-07-26)

- **Metaengine fold-classify** (`metaengine/`) — `classifyFold` inspects fold
  return types to assign ADT patterns, shared across engines for consistency.
  Eliminates divergent classification between MemoryEngine and SQLiteEngine.
- **Cross-engine meta-test** (`metaengine/cross_engine_meta_test.go`) — 150 specs
  run identical Apply → ExecuteTyped sequences on Memory + SQLite, asserting
  identical typed results. Guards the contract that engine choice must not
  affect query output.
- **End-to-end signature/ciphertext verification** (`metaengine/`) — integrated
  across Memory and SQLite engines so signed/encrypted events are verified at
  the engine boundary.
- **`metaengine/v4.1.1`** — supersedes v4.1.0's panicking `MapUpdate` (the
  `map[string]any` → struct reification path). `reifyReflect` co-located with
  `reify[R]` in `metaengine/reify.go`.
- **Error-wrapping helpers convention** (ADR-0069) — documents the per-module
  helper pattern (`wrapInfraOrOK`, `wrapTransientOrOK`, `MarshalBase64JSONWithModule`)
  used across storage/pebble, storage/kv_sql, codec, encryption, and signing.
  Capped at 3 modules per helper to preserve multi-module isolation.
- **Dedup acceptance documentation** (`docs/dedup-acceptance.md`) — documents
  the clone-group reduction methodology, thresholds (`-t 2` primary, `-t 5`
  secondary), and accepted groups with rationale.

### Fixed (metaengine hardening — 2026-07-26)

- **`cmd/api-stability/main.go` split** (353 → 238 + 123 lines) — the last
  file-size-gate violation. AST collection functions moved to `collect.go`.
  File-size gate now GREEN across all production files.

- **Metaengine `SQLiteEngine` reification** — `reifyReflect` helper handles
  `map[string]any` → struct conversion across all engine methods that return
  `any` from SQL scans. Co-located with the generic `reify[R]` function.
- **Metaengine tx-atomic `MapUpdate`** (ADR-0067) — SQLite `MapUpdate` wraps
  read-modify-write in a single transaction, preventing lost updates across
  concurrent calls.
- **Metaengine multimap seq-seed** (ADR-0068) — lazy `sync.Once` seeding from
  `MAX(seq)` on first use ensures safe restart without sequence collisions.

### Added (dedup extractions + UP1 test hardening + verify-gate GREEN — 2026-07-27)

- **Verify gate GREEN end-to-end** — `nix run .#verify` exits 0 for the first
  time across all 58 modules (build + vet + test + race + lint 0 issues +
  api-stability + doc-check 947 refs + doc-assertions). Previously flaky
  benchkit timing tests resolved via race-aware thresholds, DSN-level SQLite
  `busy_timeout`, and `soakTestScale` consolidation.
- **Code deduplication pass** — 17→3 clone groups at threshold 3 via type-aware
  art-dupl. 14 groups eliminated across 11 modules: `selectorNameAndPkg`
  (cqrs-lint/analyzer), `journalReadSpan` (pebble), `setEnabled` (turso),
  `findValueByType` (metaengine), `runLocked` (kv), `withWriteLock` (storage/memory
  command + snapshot stores), `parallelTimeoutCtx` (benchkit, 17 sites),
  `parallelViewStore` (storage/view, 21 sites), variadic `NewTestRegistry(svc...)`
  (catalog, 23 sites), `parallelExportEnv` (eventcatalog, 9 sites),
  `parallelBundle` (stack/contracttest, 4 sites), `newTestStreamEvent` (eventtest).
  `dedup-acceptance.md` documents the 3 accepted groups (mutex lock/unlock,
  named-return cleanup defer, strings.Builder with different content).
- **Race-aware test thresholds for transport/grpc** — new
  `race_on_test.go`/`race_off_test.go` build-tag files; `settleDelay` changed
  from `const` to race-aware `var` (100ms → 500ms under `-race`). 3 pubsub
  tests pass 3x under `-race`.
- **Race-aware soak test thresholds for benchkit** — `soakTestScale(base)` scales
  durations 3x under `-race`. Applied to all 5 soak tests. Consolidated the
  duplicate `soakTestDuration`/`soakTestTimeout` helpers into one function.
- **UP1 test hardening** — `FuzzCBORToJSONTransform` end-to-end (1.5M executions,
  0 panics), edge-case tests (`[]byte`→base64, float specials, duplicate CBOR map
  keys, CBOR tag 0 date/time, bignum/tagged values), `ExampleCBORToJSONTransform`
  (runnable godoc example), `BenchmarkTranscodeToJSON_NestedDeep` (7.2µs/op),
  `BenchmarkCBORToJSONTransform_FanOut_100Clients` (208µs/op — confirmed transform
  runs once per client, not memoized).
- **SQLite DSN-level busy_timeout** — `storage.EnsureSQLiteDSNBusyTimeout(dsn, ms)`
  injects `_pragma=busy_timeout(N)` at the DSN level, ensuring every pooled
  connection inherits the busy timeout (PRAGMA via `db.Exec` only applies to the
  executing connection). Wired into `stack/sqlite/preset.go` and `multidb.go`.
  Resolves the SQLITE_BUSY errors that made benchkit tests flaky under parallel load.
- **Documentation updates** — Two-Layer Pattern (primitive + adapter) added to
  CONTRIBUTING.md Code Standards; `CBORToJSONTransform` usage added to
  MIGRATION-GUIDE.md; SSE delivery / encoding projection added to
  CONSISTENCY_MODEL.md; json v2 key-ordering non-determinism documented in
  `codec/transcode.go`; `jsonBytes, _ :=` anti-pattern fixed in 2 doc examples.
- **AGENTS.md updated** — transport/grpc added alongside benchkit in the
  race-aware test threshold list (local `_test.go` suffix variant).

### Fixed (dedup + UP1 hardening — 2026-07-27)

- **cmd/cqrs-lint build failure** — `go-output` v0.32.1 renamed types
  (`Table`, `GraphBuilder`, `NewTableBuilder`) that its own submodules at v0.32.0
  still reference. Downgraded to v0.32.0. This was a REAL build failure producing
  compiler errors, not a phantom gopls issue.
- **benchkit `mustRun` timeout** — hardcoded 30s caused `context deadline
exceeded` under the verify gate's 42+ parallel packages. Changed to
  `soakTestScale(90*time.Second)` (270s under `-race`).
- **`TestRun_AnalyticalJournalScans` timing assertion** — "5 scans > 1 scan"
  timing comparison was race-gated. Made it a soft check (`t.Logf`) since timing
  comparisons are unreliable under any parallel load, not just `-race`.
- **`echo -e` → `printf`** in `check-module-layers.sh` — POSIX portability fix.
  Replaced bashism with `printf` and `$'\n'` literal newlines.
- **`FuzzCBORToJSONTransform` t.Skip → return** — standard Go fuzz pattern
  (bare return, not t.Skip which hides issues in the setup path).
- **cmd/cqrs-lint struct tag alignment** — triple-space tag fixed to
  alphabetical key order (tagalign requirement).

### Added (full TODO-list execution — 2026-07-26)

> 25 of 27 Pareto-plan tasks completed; 2 declined with documented rationale.
> See [TODO_LIST.md](TODO_LIST.md) "Declined" for the rationale.

- **`#verify-fast` nix app** — passes `-short` to skip soak tests (35s → 0.05s
  in benchkit). The rapid-iteration gate; full `#verify` remains for nightly.
- **`#verify-parallel` nix app** — splits module tests into N batches (default:
  nproc) for concurrent execution. Cuts ~4min sequential → ~1-2min.
- **`#check-duplication` nix app + `.art-dupl-baseline.json`** — CI gate that
  fails on newly introduced code clones (34 accepted groups at threshold 3).
  Local-only; CI wiring pending.
- **`#sweep` nix app** — runs `nix fmt` (gofumpt + goimports + golines) for
  auto-commit daemon drift recovery.
- **`#vulncheck` nix app** — runs `govulncheck` across all modules.
- **Taxonomy-consistency CI check** (`scripts/verify-docs.sh`) — greps for stale
  "5-family" / "5 Error Families" patterns in living docs. Prevents future
  split-brain when go-error-family adds families.
- **Idempotency property tests across all 3 implementations** — 4 rapid-based
  property tests (RecordIsIdempotent, CheckAndRecordExactlyOnce,
  KeysAreIndependent, TTLExpiry) now run against MemoryStore + KVStore +
  SQLiteStore. Each SQLite test gets a unique named in-memory DB to prevent
  parallel-test state leakage.
- **Cursor round-trip tests for non-numeric keys** — string + time keys across
  memory + SQLite engines. Verifies lexicographic/chronological ordering
  survives Encode → ParseCursor.
- **`TestTagContentMatchesChangelog` meta-test** — guards against tag/CHANGELOG
  drift. Verifies every `## [vX.Y.Z]` in CHANGELOG.md has ≥1 git tag.
- **Metaengine SQLite soak tests** — sustained writes (8 writers × 500 + 4
  readers × 200) and multimap growth (1000 writes across 10 keys), verifying
  grand-total integrity. Skip in `-short` mode.
- **`startReadSpan` consolidation in pebble** — extracted helper matching the
  existing `startLimitSpan` / `startStreamSpan` pattern. Applied to 5 bare
  `StartSpan` sites; consolidated 3 `ReadFrom` error arms. Net: -20 lines.
- **API stability golden regenerated** — 2675 exports (was 2637). New exports
  from property tests, cursor tests, soak tests, meta-test.
- **go-error-family v0.10.0 CHANGELOG entry** — records the upgrade, Orchestration
  family addition, 3 exhaustive-switch fixes, and "5-family" → "6-family" doc
  updates across error-taxonomy.md, README.md, FEATURES.md.
- **4 historical files annotated** + **2 HTML dashboards hand-edited** —
  analytics-rollup-review (rejected), NEXT-LEVEL-EXECUTION-STATUS (verify
  GREEN), meta-engine-design (shipped as metaengine/v4), benchkit-implementation
  (shipped). HTML: PARETO-EXECUTION-STATUS (Superseded badge),
  cqrs-ecosystem-audit (All Issues Resolved).
- **`metaengine/projectionadapter/v4.0.0` tag pushed** to origin.

### Changed

- **go-error-family upgraded v0.9.0 → v0.10.0** across all 50 modules. The new
  `Orchestration` family (6th family) classifies internal coordination failures
  (e.g., projection host lifecycle errors, dead-letter orchestration bugs).
  Three exhaustive `switch` statements updated with the new case: `projectionhost`,
  `middleware`, and `benchkit`. Not a breaking API change for consumers — the
  new family is additive. Documentation updated: "5-family" → "6-family"
  everywhere (error-taxonomy.md, README.md, FEATURES.md, AGENTS.md).
- **README rewrite** — restructured as 3-step Quick Start (define domain, event-source
  with decider, go to production). Added Install section, trimmed module catalog to 12
  key modules (links to AGENTS.md for full 58). Moved "Why" section before catalog.
- **Docs compile-verification** — `docs_compile_test.go` in `example/getting-started/`
  tests every API pattern from `docs/getting-started.md` to catch drift in CI.
- **Module count** — 56 → 58 `go.mod` files (metaengine, benchkit, cmd/cqrs-bench,
  example/readme-quickstart, metaengine/projectionadapter, idempotency/sqlstore).
- **Storage error var rename** — `ErrAggregateTypeMismatch` → `ErrStreamTypeMismatch`
  and `ErrAggregateIDMismatch` → `ErrStreamIDMismatch` in `storage/sql` and
  `storage/pebble`. Deprecated aliases preserve backward compatibility. Error code
  strings (wire format) unchanged.
- **Benchkit number formatting** — replaced 25-line hand-rolled `insertCommas()` with
  `humanize.Comma()` from `go-humanize`.
- **api-stability module list** — fixed 3 dead entries (`memory`/`pebble`/`turso`),
  corrected `event/eventtest` → `event/v4/eventtest`, added `metaengine`, `benchkit`,
  `stack/bench`, `cmd/cqrs-bench`. Golden file regenerated: 2637 exports (was 2340).
- **Doc accuracy fixes** — `error-taxonomy.md` and `DOMAIN_LANGUAGE.md` updated to use
  `errorfamily.*` constructors instead of removed `event.*` error functions.
  `CHANGELOG.md` migration code block switched to `diff` fence to avoid false
  doc-check warnings.
- **FEATURES.md cleanup** — removed dead "Known Code Quality Issues" section
  (6 resolved items).

### Removed (Breaking — targets v4.1)

- `middleware.NewMetrics`, `CommandMetrics`, `EventMetrics`, `QueryMetrics` — entire
  `metrics.go` deleted. Use `NewTypedMetrics`, `CommandTypedMetrics`, `EventTypedMetrics`,
  `QueryTypedMetrics` instead.
- `middleware.MetricsRecorder` interface and `OTelMetricsRecorder.Observe` — use
  `TypedMetricsRecorder` and `ObserveTyped` instead.
- `catalog.ErrorExporter` — use `Exporter[error]` instead.
- `storage/sql.NewOwnedDBHandle` and `SetOwnership` — use `NewBorrowedDBHandle` or
  `NewOwningDBHandle` instead.
- `eventtest.FakeMetrics` and `eventtest.AssertMetricRecord` — removed with the
  deprecated `MetricsRecorder` interface they implemented.

### Migration Guide: Aggregate→Stream Rename (ADR-0058)

Identity types renamed across `id/`, `event/`, `command/`, `listing/`,
`otel/`, `storage/`. All old names remain as deprecated aliases.

**Rename map** (old → new):

| Old                        | New                     |
| -------------------------- | ----------------------- |
| `AggregateID`              | `StreamID`              |
| `AggregateType`            | `StreamType`            |
| `AggregateRef`             | `StreamRef`             |
| `AggregateMarker`          | `StreamMarker`          |
| `NewAggregateID`           | `NewStreamID`           |
| `DeriveAggregateID`        | `DeriveStreamID`        |
| `NewAggregateRef`          | `NewStreamRef`          |
| `ParseAggregateType`       | `ParseStreamType`       |
| `ErrAggregateTypeMismatch` | `ErrStreamTypeMismatch` |
| `ErrAggregateIDMismatch`   | `ErrStreamIDMismatch`   |

**Intentionally kept as "aggregate" (wire-format stability):**

- JSON struct tags (`aggregate_id`, `aggregate_type`) — on-disk serialization
- SQL column names (`aggregate_type`, `aggregate_id`) — schema migrations
- Error classification codes (`event.nil_aggregate_id`, `pebble.aggregate_type_mismatch`) — `errors.Is` match keys
- OTel attribute string values (`cqrs.aggregate.*`) — dashboard/alert schema
- `AggregateAwareStrategy` interface, `catalog.AggregateRoot` field — DDD concepts

## [v4.0.4] - 2026-07-23

### Batch release — 49 modules tagged

> **Note:** `cmd/cqrs-lint` was NOT tagged in this release — its `go-finding`
> dependency is still local-only (unpublished). It will be tagged separately
> once `go-finding` is published to the Go module proxy.

**Added:**

- **COSE Sign1 signing** (`signing`) — RFC 9052 COSE Sign1 implementation for
  event signature verification, replacing the previous ad-hoc signature format.
- **COSE encryption support** (`encryption`) — COSE-compatible ciphertext
  handling for at-rest event encryption with improved key management.
- **Event encryption/signing integration** (`event`) — `MultiBatchEntry` and
  `MultiSink` interface for multi-aggregate atomic writes with encryption
  and signing support.
- **OTel instrumentation** (`storage`) — OpenTelemetry spans for event store
  operations (append, load, query) with attribute enrichment.
- **Multi-batch event store** (`storage`) — `SaveMultiBatch` for writing
  events across multiple aggregates in a single atomic operation.
- **Nix flake support** — development environment reproducibility via
  `flake.nix` with pre-commit hooks and Go workspace integration.
- **Getting started guide** (`docs/getting-started.md`) — step-by-step
  onboarding for new users with quickstart example.
- **Architecture documentation** (`docs/architecture-understanding/`) —
  book insights vs codebase comparison and four-tier model diagrams.

**Changed:**

- **gRPC transport refactored** (`transport/grpc`) — improved developer
  experience with cleaner event handler registration and error propagation.
- **Storage journal reader** (`storage`) — improved performance and error
  handling in journal read paths with better memory allocation patterns.
- **Command bus enhancements** (`watermill`) — improved message routing and
  delivery guarantees for command dispatch.
- **Stack presets** (`stack/*`) — multi-database support improvements with
  unified configuration across SQLite, Postgres, Pebble, and Turso backends.
- **Dependency alignment** — all 52 modules aligned with workspace revisions;
  internal pseudo-version requires resolved to published tags.

**Tags:**

| Module                      | Version                          | Module                       | Version                 |
| --------------------------- | -------------------------------- | ---------------------------- | ----------------------- |
| `catalog/v4.0.4`            | `cmd/api-stability/v4.0.2`       | `cmd/cqrs-gen/v4.0.2`        | `cmd/doc-check/v4.0.1`  |
| `codec/v4.0.4`              | `command/v4.0.2`                 | `decider/v4.0.3`             | `dedup/v4.0.1`          |
| `deriver/v4.0.2`            | `dispatcher/v4.0.2`              | `encryption/v4.0.3`          | `event/v4.0.4`          |
| `event/v4/eventtest/v0.2.1` | `example/getting-started/v4.0.2` | `example/taskmanager/v4.0.2` | `graph/v4.0.3`          |
| `id/v4.0.3`                 | `idempotency/v4.0.2`             | `idempotency/kvstore/v4.0.2` | `integration/v4.0.2`    |
| `kv/v4.0.3`                 | `listing/v4.0.3`                 | `metadata/v4.0.2`            | `middleware/v4.0.3`     |
| `otel/v4.0.3`               | `projection/v4.0.2`              | `projectionhost/v4.0.3`      | `prometheus/v4.0.2`     |
| `query/v4.0.2`              | `retry/v4.0.2`                   | `scenario/v4.0.3`            | `scheduling/v4.0.3`     |
| `schema/v4.0.3`             | `signing/v4.0.3`                 | `snapshot/v4.0.3`            | `stack/v4.0.2`          |
| `stack/bench/v4.0.2`        | `stack/memory/v4.0.2`            | `stack/pebble/v4.0.2`        | `stack/postgres/v4.0.2` |
| `stack/sqlite/v4.0.2`       | `stack/turso/v4.0.2`             | `storage/v4.0.3`             | `storage/memory/v4.0.2` |
| `storage/pebble/v4.0.3`     | `storage/turso/v4.0.2`           | `testutil/v4.0.2`            | `transport/grpc/v4.0.2` |
| `transport/http/v4.0.3`     | `watermill/v4.0.4`               |                              |                         |

## [v4.0.3] - 2026-07-22

### Batch release — 48 modules tagged

> **Note:** `cmd/cqrs-lint` was NOT tagged in this release — its `go-finding`
> dependency is still local-only (unpublished). It will be tagged separately
> once `go-finding` is published to the Go module proxy.

**Fixed:**

- **Turso `LoadToTimestamp` test was flaky** — used `time.Sleep(10ms)` + `time.Now()`
  (racy wall-clock with nanosecond precision). Rewritten to use explicit
  `event.WithOccurredAt` timestamps with large intervals, matching the pattern
  used by every other `LoadToTimestamp` test in the codebase.

**Changed:**

- **SQL dialect abstraction** (`storage/sql`) — refactored to support
  multi-database compatibility. All SQL stores now flow through a typed
  `Dialect` interface with SQLite and Postgres implementations.
- **Stack preset centralization** (`stack/`) — options consolidated into
  `sqlopt` package, eliminating three harmful clones across stack presets.
- **Harmful duplication eliminated** — shared helpers extracted across 10+
  modules (codec, dispatcher, signing, encryption, command, query, catalog,
  storage, watermill, scenario, retry, event).
- **JSON v2 migration** — codec, event, and middleware migrated to
  `encoding/json/v2` via `goexperiment.jsonv2` build tag.
- **Dependency alignment** — all 52 modules aligned with workspace revisions.

**Added:**

- **View store: transactional support** — `InTx` and `Executor` interface for
  atomic view operations (`storage/view`).
- **View store: keyset pagination** — multi-column `ORDER BY`, partial indexes,
  `IS NULL` operators, `RawWhere` escape hatch, `ViewUpdater`, BLOB support.
- **Catalog: REST helper shortcuts** — composite `WithOperation`, duplicate
  detection, golden tests with CI freshness check.
- **cqrs-lint improvements** (not tagged — pending `go-finding` publication) —
  scanner accuracy overhaul (handler→struct link recovery across 5 patterns,
  reducing consumer false positives 44→8), output rendering with source
  snippets, monorepo support, `--strict-load` flag, loader error surfacing.
- **C014 lint rule** — detects `time.Local` usage in event payload structs.

**Tags:**

| Module                           | Version                      |
| -------------------------------- | ---------------------------- |
| `catalog/v4.0.3`                 | `cmd/api-stability/v4.0.1`   | `cmd/cqrs-gen/v4.0.1`   |
| `codec/v4.0.3`                   | `command/v4.0.1`             | `decider/v4.0.2`        | `deriver/v4.0.1`            |
| `dispatcher/v4.0.1`              | `encryption/v4.0.2`          | `event/v4.0.3`          | `event/v4/eventtest/v0.2.0` |
| `example/getting-started/v4.0.1` | `example/taskmanager/v4.0.1` | `graph/v4.0.2`          | `id/v4.0.2`                 |
| `idempotency/v4.0.1`             | `idempotency/kvstore/v4.0.1` | `integration/v4.0.1`    | `kv/v4.0.2`                 |
| `listing/v4.0.2`                 | `metadata/v4.0.1`            | `middleware/v4.0.2`     | `otel/v4.0.2`               |
| `projection/v4.0.1`              | `projectionhost/v4.0.2`      | `prometheus/v4.0.1`     | `query/v4.0.1`              |
| `retry/v4.0.1`                   | `scenario/v4.0.2`            | `scheduling/v4.0.2`     | `schema/v4.0.2`             |
| `signing/v4.0.2`                 | `snapshot/v4.0.2`            | `stack/v4.0.1`          | `stack/bench/v4.0.1`        |
| `stack/memory/v4.0.1`            | `stack/pebble/v4.0.1`        | `stack/postgres/v4.0.1` | `stack/sqlite/v4.0.1`       |
| `stack/turso/v4.0.1`             | `storage/v4.0.2`             | `storage/memory/v4.0.1` | `storage/pebble/v4.0.2`     |
| `storage/turso/v4.0.1`           | `testutil/v4.0.1`            | `transport/grpc/v4.0.1` | `transport/http/v4.0.2`     |
| `watermill/v4.0.3`               |                              |                         |                             |

## [v4.0.2] - 2026-07-18

### CBOR time encoding fix + timezone-safe types

**Fixed:**

- **CBOR time encoding loses nanosecond precision** — `CanonicalEncOptions()`
  defaulted `Time` to `TimeUnix` (epoch seconds, no nanos, no timezone). Changed
  to `TimeUnixDynamic` (float64, preserves nanoseconds, ~165ns drift).
  Affects all user-defined payload structs with `time.Time` fields.
- **`event.defaultClock` returned local time** — changed to return UTC.
- **Pebble storage and Watermill protocol** did not normalize timezone on
  deserialization. Now explicitly call `.UTC()`.

**Added:**

- **`event.Instant`** — UTC-normalized timestamp type for event payloads.
  Wraps `time.Time`, enforces UTC at construction, marshals to int64 UnixNano
  via CBOR (exact precision, no timezone loss).
- **`event.WallTime`** — Time-of-day type tied to an IANA timezone.
  DST-aware `NextOccurrence` and `PreviousOccurrence` methods.
- **`event.Date`** — Calendar date type (year, month, day) without time.
  Timezone-agnostic; prevents off-by-one-day bugs.
- **`event.Zero`** constant — zero-value `Instant` for "no timestamp".
- **`Instant.Sub`, `Instant.Add`** — UTC-preserving arithmetic.
- **`WallTime.IsValid`, `WallTime.MarshalCBOR/UnmarshalCBOR`**.
- **C013 lint rule** — detects `time.Time` fields in event payload structs.
  Now detects nested anonymous struct fields and gives specific suggestions.
- **`docs/TIMEZONE_HANDLING.md`** — comprehensive timezone handling guide.
- **`event/doc.go`** — package-level documentation with time handling conventions.

**Tags:** `codec/v4.0.2`, `event/v4.0.2`, `storage/pebble/v4.0.1`, `watermill/v4.0.2`

### cqrs-lint v0.2.2 — loader error surfacing + --strict flag

**Fixed:**

- **Silent "Nothing to lint" on broken builds** — when `go/packages` failed to
  load packages (e.g., unresolvable dependencies from the go-cqrs-lite v4.0.0
  publish bug), the loader silently `continue`d past errors and produced an
  empty `AnalysisContext`. cqrs-lint then reported "No Go files importing
  go-cqrs-lite found. Nothing to lint." — a clean bill of health on a broken
  project. Now `BuildContext` collects per-package and per-module errors into
  `AnalysisContext.LoadErrors` and the main `run()` function surfaces them
  with a clear diagnostic and a non-zero exit code.
- **Doctor/lint split-brain on errored packages** — `doctor` read
  `ctx.Packages` (which includes errored packages) for feature detection while
  `lint` read `ctx.GoFiles` (which excludes them). The two commands could
  disagree on the project's feature profile. Now `DetectFeatures` skips
  packages with errors, making `lint` and `doctor` read the same data.
- **Doctor silently ignored `BuildContext` errors** — the `doctor` command
  called `BuildContext` but discarded the `LoadErrors` field. Now prints a
  "WARNING: package loading was partial" block with per-package error details.
- **Feature detection used unreliable import metadata** — errored packages
  may have incomplete `Imports` slices. `DetectFeatures` now skips packages
  with `len(pkg.Errors) > 0` in its import-based detection pass.

**Added:**

- **`--strict-load` flag** — exits non-zero if any packages failed to load, even
  when some packages were analyzed successfully. Without `--strict-load`, cqrs-lint
  proceeds with partial analysis and prints a warning.
- **Partial analysis warning** — in non-strict mode, when some packages loaded
  with errors but others succeeded, cqrs-lint prints a warning with the error
  count and suggests `--verbose` or `--strict-load`.
- **Message split for empty analysis** — "No Go files found. Nothing to lint."
  (no `.go` files at all) vs "Found Go files but none import go-cqrs-lite.
  Nothing to lint." (packages loaded, none import go-cqrs-lite). The old
  message conflated both cases.
- **`--verbose` load error display** — the `--verbose` output now includes a
  "Load errors (N):" section with per-package details.
- **Integration tests for loader error handling** — tests for broken modules,
  clean modules, syntax errors, strict mode, message split, and exit codes.

**Changed:**

- **`AnalysisContext.LoadErrors` field** — new `[]PackageLoadError` field holds
  per-package load errors collected during `BuildContext`. The
  `PackageLoadError` struct carries `Module`, `PkgPath`, and `Errors` fields.
- **`printLoadErrors` helper** — shared between `run()` and `doctor` for
  consistent error formatting.

### cqrs-lint v0.2.1 — health score correctness + show-suppressed

**Fixed:**

- **Health score computed on filtered findings** — the health score was
  computed on display-filtered findings (post severity/confidence filtering)
  instead of all unsuppressed findings. This meant `--min-severity`,
  `--min-confidence`, and `--fp-suspects` would change the health score,
  making it an unreliable project-health metric. Now computed on all
  unsuppressed findings.
- **InfoCapped display showed wrong cap value** — the health score display
  hardcoded the default cap (20) instead of showing the actual configured cap
  from `{"health": {"info-cap": N}}`. Added `InfoCapApplied` to HealthScore.
- **Suppressed findings shown in output** — `//cqrs-lint:ignore(RULE)`
  comments marked findings as suppressed but they still appeared in all
  output formats. Now properly filtered from output, health score, and the
  error-exit check. A summary count is printed to stderr.
- **scanner.Err() unchecked** — the suppression parser's `bufio.Scanner`
  errors (e.g., buffer overflow on >1MB lines) were silently dropped. Now
  logged to stderr as a warning.

**Added:**

- **`--show-suppressed` flag** — lists suppressed findings with their file
  location, rule ID, and suppression reason. An audit view beyond the
  counts-only `cqrs-lint doctor`.
- **`--fp-suspects` flag** — surfaces only low-confidence findings (below
  Medium confidence), which are the most likely false positives. Advisory
  mode: exit code is always 0.
- **Suppression count in output** — the main lint run now reports how many
  findings were suppressed by inline comments.

**Changed:**

- **`filterSuppressed` returns both active and suppressed slices** — enables
  the `--show-suppressed` feature and eliminates the need for a second pass.
- **Pre-allocated filter result slices** — `filterBySeverity`,
  `filterByConfidence`, `filterSuppressed`, `filterFPSuspects`, and
  `collectFindings` dedup now pre-allocate to avoid reallocation.
- **Extracted `shouldExitWithError`** — the exit-code decision is now a
  testable function instead of inline logic in `run()`.

### Documentation Health

- **Historical artifact banners** — All 41 `docs/*2026-07-1*.md` session reports
  now carry a banner at the top stating they are point-in-time snapshots, with
  links to this CHANGELOG and TODO_LIST.md for current status. This prevents
  readers from acting on stale TODO/Open/Not Started items in old reports.

### Fixed — Documentation Health Audit (2026-07-16)

- **README.md license section** — Said "MIT" but actual LICENSE file is
  PROPRIETARY. Corrected to match reality. This was a Critical documentation
  lie — consumers could have assumed MIT when the code is proprietary.
- **Module count across all docs** — AGENTS.md, README.md, FEATURES.md,
  CONTRIBUTING.md, docs/README.md, docs/v4-WISHLIST.md said "48" or "49"
  but the actual count is 52 `go.mod` files. All references now say 52 with
  a verify command (`find . -name go.mod -not -path './vendor/*' | wc -l`).
- **ROADMAP.md** — Was frozen at v3.6.0 ("Current State: v3.6.0 released")
  despite v4.0.0 being shipped. Rebuilt from scratch: current state reflects
  v4.0.0, release history table added, Long Term Vision cleaned (completed
  items removed — they belong in CHANGELOG, not ROADMAP).
- **TODO_LIST.md** — 13 completed items were still listed as open (middleware
  ordering guide, SQL TimerStore, SQL AggregateReader, lint-clean scheduling
  and scenario, ADR numbering fix, CONTRIBUTING agent rules, DeadLetterStoreAdmin
  docs, per-projection lag, session archiving, module graph, dependency model
  consolidation, event go.mod tidy). Removed; remaining items are genuinely open.
- **README.md migration reference** — Said "Migrating from v2?" but the current
  version is v4. Updated to reference both v3 and v4 migration guides.
- **FEATURES.md missing cqrs-lint** — Shipped feature (60 rules, 159+ tests)
  absent from feature inventory. Added full feature table.
- **FEATURES.md missing SQLTimerStore** — `storage.SQLTimerStore[T]` shipped
  but only MemoryTimerStore listed. Added row.
- **docs/v4-WISHLIST.md** — Said "PREP COMPLETE" and "Ready to cut" despite
  v4.0.0 already shipped. Updated to SHIPPED with correct module count.
- **docs/README.md module count** — Said "28 modules" (very stale). Updated
  to 52.

### Resolved During July 2026 Sessions

> The items below were flagged as TODO/Open/Not Started/Broken in the
> `docs/*2026-07-1*.md` session reports. They have ALL been resolved.
> This section exists so readers of those historical reports can verify
> resolution without grepping through code.

**v4 Release (2026-07-11):**

- ✅ Module path migration `/v3` → `/v4` (49 go.mod, ~750 .go files)
- ✅ Codec defaults flipped to CBOR (event, kv, snapshot, command, query)
- ✅ Deprecated alias removal (8 event/ aliases, WithNewCodec, WithReplay)
- ✅ BackfillHandler consolidation (`BackfillHandler(*SSEBroker)`)
- ✅ Storage split (eventstore/, readmodel/, sql/, relational/, view/, migrations/)
- ✅ HealthCheck on OwnedDBHandle (all SQL stores inherit)
- ✅ WithShutdownDependency (topological close ordering)
- ✅ ADR-0044 blind store envelopes (WrapEncode/UnwrapDecode)
- ✅ JSON quality audit (Deterministic + MatchCaseInsensitiveNames on all calls)
- ✅ ADRs 0047-0054 written (json/v2, transport codec, dispatch middleware, etc.)
- ✅ eventtest tag created locally (`event/v4/eventtest/v0.1.0`)
- ✅ v3 git tag backfill (v3.0.0–v3.7.1)

**DiscordSync Feedback Gaps (2026-07-10/11):**

- ✅ Gap 1: `schema.VersionedSeekableJournal` (upcasters for projectionhost)
- ✅ Gap 2: `transport/http.WithPayloadTransform` (all 3 SSE paths: live, replay, backfill)
- ✅ Gap 3: `projectionhost.SQLiteDeadLetterStore` (persistent DLQ)
- ✅ Gap 4: `prometheus.WithViews` (CQRS histogram boundaries applied)
- ✅ `DeadLetterStoreAdmin` interface (Count, ListPaged, PurgeBefore)
- ✅ `BackfillHandlerWithTransform` → consolidated into BackfillHandler(broker)

**Post-v4 Cleanup (2026-07-12):**

- ✅ 287 session artifacts archived to `docs/*/archive/`
- ✅ `docs/getting-started.md` rewritten + compile-verified
- ✅ ADR index regenerated (0032 → 0054)
- ✅ Module dependency graph (mermaid) in README
- ✅ `docs/middleware-ordering.md` (recommended order + rationale)
- ✅ CONTRIBUTING.md: Four-Tier Model (replaced stale 7-layer)
- ✅ CONTRIBUTING.md: AI Agent safety rules
- ✅ ADR numbering fix (duplicate 0047 → 0054)
- ✅ Lint-clean scheduling (11→0) and scenario (4→0)
- ✅ DiscordSync feedback doc reconciled (Gaps 3-5: REJECTED → SHIPPED)
- ✅ API surface regenerated (2209 exports)
- ✅ `fire_at` → `fireAt` JSON tag (tagliatelle compliance)
- ✅ `nix fmt` clean, `nix run .#lint` clean (0 issues)
- ✅ Doc-check passed (880+ references valid)

**cqrs-lint (2026-07-16) — Built from scratch across 6 sessions:**

- ✅ 60 rules with real detectors (correctness 12, API 19, boilerplate 15, consistency 4, architecture 7, security 3)
- ✅ 159+ tests + 3 benchmarks, 0 lint issues
- ✅ Scanner accuracy fixes (6 critical bugs: capturePayloadType, detectFoldFunc, isLikelyDecider, isOOAggregate, C004 dead rule, Type/AggregateID scanning)
- ✅ Source snippets on 34/60 detectors
- ✅ CLI overhaul: fang, go-output tables, monorepo support, module grouping
- ✅ Dead code eliminated (13 items: TypeResolver, HandlerInfo, nodeString, etc.)
- ✅ Severity filter bug fixed (alphabetical comparison → Severity.Compare)
- ✅ Catalog consolidated (3 files → 2, organized by category)
- ✅ Finding location improvements (go.mod:1:1 → real source lines on 7 rules)
- ✅ SARIF golden file test
- ✅ JSON v2 test determinism (10 non-deterministic tests fixed)
- ✅ Race detection verified (zero data races across all 50+ modules)
- ✅ Pipeline metrics wired (--verbose shows per-detector timing)
- ✅ CONTRIBUTING.md rule development guide

**Documentation Health Audit (2026-07-16):**

- ✅ README.md license corrected (MIT → PROPRIETARY)
- ✅ Module count corrected across all docs (48 → 52)
- ✅ ROADMAP.md rebuilt (v3.6.0 → v4.0.0)
- ✅ TODO_LIST.md cleaned (13 stale items removed)
- ✅ FEATURES.md updated (cqrs-lint + SQLTimerStore added)
- ✅ Historical artifact banners added to all session reports

### Fixed — Prior Session Fixes

- **README.md and docs/getting-started.md code examples** — Command examples
  were missing the required `ID()` method (inherited via `*command.BasicCommand`
  embedding). Getting-started also used `event.NewEvent` instead of `event.New`
  for typed payloads, referenced nonexistent `event.NewMemoryBus()`, and used
  `Fold` instead of `Apply` on `decider.Decider`. All examples now compile-verified.
- **api-stability golden file** — `docs/api_surface.txt` was stale (missing
  `kv.OpIsNull`, `kv.OpIsNotNull`, `kv.ViewUpdater` shipped in v4.0.0). Regenerated.
- **kv/benchmark_test.go** — Replaced `[]byte(fmt.Sprintf(...))` with
  `fmt.Appendf(nil, ...)` to clear `fmtappendf` diagnostics.

### Changed

- **scheduling: JSON tag `fire_at` → `fireAt`** — Tagliatelle (camelCase)
  compliance. The scheduling module is new in v4; no pre-v4 data exists to break.
- **scheduling: Magic numbers extracted to named constants** —
  `defaultPollInterval`, `defaultMaxRetries`, `defaultRetryDelay`,
  `jitterHalfDivisor`. All `ctx.Err()` returns wrapped per wrapcheck.
- **scenario: Lint cleanup** — `exhaustruct` nolints on builder-pattern structs,
  `errname`/`varnamelen` fixes.

### Documentation

- **287 session artifacts archived** — `docs/{status,planning,research,reviews,
quality,architecture-understanding,brainstorming,modularization}/` timestamped
  files moved to `archive/` subdirectories with explanatory READMEs.
- **Feedback doc reconciled** — DiscordSync round 3 appendix Gaps 3-5 changed
  from "REJECTED" to "SHIPPED" to match actual code.
- **ADR index extended** — From 0032 to 0054 (21 new entries, duplicate 0047
  renumbered to 0054).
- **CONTRIBUTING.md** — 7-layer model replaced with Four-Tier Model (ADR-0046).
  Added "Working with AI Agents" section.
- **New docs/middleware-ordering.md** — Recommended middleware application order
  for all 30+ middlewares.

## [cmd/cqrs-lint/v0.2.0] - 2026-07-17

### cqrs-lint — DiscordSync feedback: false-positive reduction & fairer scoring

These changes close the feedback loop on the DiscordSync cqrs-lint report
(18 of 34 findings were D002 false positives). Rule heuristics are now shaped
by what real consumers do, and the health score no longer lets info-level noise
drown real bugs.

**Fixed — rule false positives (prior session, now changelogged):**

- **C001 closure-escape** — `txVarEscapesToArg` now skips the false positive
  where the tx variable is passed to a callback that contractually owns the
  commit (suggesting `return tx.Commit()` there would double-commit). The old
  test encoded the false positive; replaced with a genuine missing-commit case.
- **C008 money corroboration** — money field names split into strong
  (amount/price/cost/balance/fee — fire alone) and weak
  (value/total/charge/payment/salary — need a money struct/package name).
  Eliminates the observability/metrics false positives.
- **D005 version-token regex** — `HasPrefix("v") && len >= 3` replaced with
  `^v\d+\.\d+`, rejecting prose words ("via", "version") and bare major
  versions ("v3") while keeping real semver references.
- **A005 broadcast vs projection** — `classifyCallbackBody` inspects the
  SubscribeAll callback and suppresses fire-and-forget fan-out (SSE broadcasters,
  stats notifiers) that has broadcast calls and no persistence calls.

**Added — D002 external-API opt-out (biggest remaining false-positive source):**

- D002 now excludes structs that mirror an external API (Discord/Stripe/GitHub),
  whose snake_case JSON tags are dictated upstream and aren't a local style
  choice. Two complementary opt-outs:
  - **Config** (`.cqrs-lint.json`): `"rules": {"external-api-struct-prefixes": ["Discord", "Stripe"]}`
    marks every struct whose name starts with a listed prefix.
  - **In-source marker**: `//cqrs-lint:external-api` on a struct's doc comment
    marks a single struct (works on both single `type Foo struct{}` and grouped
    `type ( ... )` blocks).
- `cqrs-lint doctor` now prints the loaded `rules` overrides so consumers can
  verify their prefix list was picked up.

**Improved — recall & context-awareness:**

- **C001 tx-use signal** — a function that uses the tx (`tx.Exec`, `tx.QueryRow`,
  any non-lifecycle method) now flags even without a bare `return nil`. tx usage
  is a stronger bug signal than the return shape; the old gate missed functions
  that return a sentinel/wrapped error after using the tx.
- **A005 widened broadcast signals** — added `Publish`, `Emit`, `Forward`,
  `Dispatch`, `WriteTo`, `Flush` to the fan-out detector (safe: a callback that
  both broadcasts and persists still flags). Catches deriver/republish patterns
  that aren't projections.
- **C008 project-aware downgrade** — when no package or struct anywhere in the
  project looks monetary, strong-field findings downgrade to Info/Low
  ("maybe money") instead of Warning/Medium. Non-payments codebases no longer
  get full-severity money warnings on coincidental `amount`/`balance` fields.

**Changed — fairer health score:**

- **Confidence weighting** — each finding's deduction is scaled by its
  confidence: High/Full = full deduction, Medium = 75%, Low = 50%. A flood of
  Low-confidence heuristic matches no longer costs the same as confirmed bugs.
  (No-confidence findings keep full weight, preserving prior behavior.)
- **Info cap** — total Info-severity deductions are capped at 20 points so a
  chatty style rule can't outweigh a Critical correctness bug.

> **Migration impact:** both scoring changes shift health scores on re-run.
> Projects that previously lost many points to info-level findings (especially
> D002 mixed-casing) will see their score rise. The relative ranking of findings
> is unchanged; only the aggregate penalty is fairer. Pin to `v0.1.0` if you
> depend on the old absolute score values.

## [cmd/cqrs-lint/v0.1.0] - 2026-07-16

First release of the domain-aware linter for CQRS/ES Go projects.

### Added

- **60 rules across 6 categories** — correctness (12), API misuse (19), boilerplate
  (15), consistency (4), architecture (7), security (3). Each rule has a real detector
  backed by AST analysis.
- **CLI** — struct-tag flags, `--min-confidence`, `--health-score`, `--verbose`,
  `--color`, `--format` (text/json/sarif/markdown). Monorepo support with per-module
  output grouping.
- **Config file** — `.cqrs-lint.json` via cmdguard for project-specific rule
  configuration, exclusions, and severity overrides.
- **Suppression comments** — `//cqrs-lint:ignore(rule-id) reason` for inline false-positive
  suppression.
- **Health score** — `--health-score` computes a 0–100 code health metric weighted by
  finding severity and category.
- **Source snippets** — 34/60 detectors include the offending source line in findings.
- **SARIF output** — GitHub Code Scanning integration via `--format sarif`.
- **165 tests** — positive tests (rule fires), negative tests (rule doesn't fire),
  scanner accuracy tests, CLI output tests, SARIF golden file test.
- **Auto-fix** — 4 rules support `--fix` for automatic remediation.

### Fixed

- **A011 detector compile bug** — `slices.Contains()` was called with zero arguments
  (incomplete refactor in `b3931503`). Fixed to `slices.ContainsFunc` with a suffix
  predicate. Detector now correctly identifies event payload structs.

## [retry/v4.0.0] - 2026-07-16

First release of the zero-dependency retry module.

### Added

- **`retry.Do(ctx, fn)`** — executes `fn` with exponential backoff and jitter until
  success, context cancellation, or max retries exhausted.
- **`retry.Config`** — configurable max attempts, initial delay, max delay, multiplier,
  and jitter factor. Sensible defaults via `retry.DefaultConfig`.
- **`retry.ErrExhausted`** — sentinel error returned when all retries are spent.
- **`retry.ErrCanceled`** — sentinel error returned when context is cancelled mid-retry.
- **Zero dependencies** — no CQRS, no OTel, no external imports. Pure Go stdlib.

## [idempotency/kvstore/v4.0.0] - 2026-07-16

First release of the KV-backed idempotency subpackage.

### Added

- **`kvstore.KVStore`** — idempotency store backed by the `kv.Store` interface.
  Works with any KV implementation (memory, Pebble, SQL-backed).
- **`kvstore.KVBackend`** — interface for pluggable KV backends.
- **Atomic check-and-set** — prevents duplicate processing under concurrent access.

## [v4.0.1 patches] - 2026-07-16

Per-module patch releases for bug fixes and additive features since v4.0.0.

### Fixed (projectionhost/v4.0.1)

- **Stagger-shutdown leak** — `defer wg.Done()` moved to `Start()` goroutine to
  prevent WaitGroup underflow during graceful shutdown.
- **Status() non-deterministic ordering** — `Host.Status()` output now sorted by
  worker name for deterministic test assertions and dashboard output.
- **purgeDLQ field initialization** — `purgeDLQ` explicitly initialized in constructor
  to prevent zero-value ambiguity.

### Fixed (watermill/v4.0.1)

- **EventBus dispatch deadlock** — `Close()` now releases `b.mu` before calling
  `backend.Close()`, preventing deadlock with background dispatch goroutine.

### Added (storage/v4.0.1, kv/v4.0.1)

- **IS NULL / IS NOT NULL operators** — `kv.OpIsNull`, `kv.OpIsNotNull` for NULL-aware
  view queries. Supported in `SQLViewStore.Query` and `Count`.
- **`RawWhere` escape hatch** — raw SQL WHERE clause injection for complex predicates
  not expressible via the `Condition` struct.
- **`ViewUpdater` interface** — incremental view updates (read-modify-write) for
  projections that need to merge event data with existing row state.
- **BLOB support** — `ViewColumn` now supports `BLOB` type for binary payload storage.

### Changed (metadata normalization batch)

- Module dependency references normalized across 15 modules (event, decider, middleware,
  signing, encryption, listing, codec, otel, schema, snapshot, id, graph, scenario,
  scheduling, transport/http). `metadata/v4` pinned to v4.0.0 in all go.mod files.
  No API changes — internal go.mod housekeeping only.

## [4.0.0] - 2026-07-11

**Major version cut — CBOR defaults, API cleanup, BackfillHandler consolidation.**

This release flips all codec defaults from JSON to CBOR (with full backward
compatibility via envelope wrapping), removes deprecated APIs, consolidates
the SSE backfill API, and migrates module paths to `/v4`.

See [`docs/migration/MIGRATION-GUIDE.md`](docs/migration/MIGRATION-GUIDE.md)
for step-by-step upgrade instructions.

### Breaking Changes

1. **Module path migration `/v3` → `/v4`** — All 49 `go.mod` files and every
   import path updated. Consumers must update `go.mod` require directives and
   all import statements. `go mod tidy` resolves most of this automatically.

2. **Codec defaults flipped to CBOR** — `event.DefaultCodec`, `kv.NewTypedStore`,
   `snapshot.NewTypedStore`, `command.NewTypedStore`, `query.NewTypedStore`,
   and `stack.ReadModel`/`Materialize` all default to `CBORCodec` instead of
   `JSONCodec`. **No data migration required** — envelope wrapping (ADR-0044)
   stamps the encoding on every write and auto-detects on read. Old JSON data
   is transparently handled via the permanent JSONCodec fallback (ADR-0050).
   See ADR-0053 for the full rationale.

3. **Deprecated aliases removed** — 8 event/+schema/ aliases deleted
   (`AggregateRef`, `Tracing`, `CustomData`, etc.). `event.WithNewCodec`
   removed (use `WithCodec`). `event.WithReplay` removed (use
   `WithProcessingMode`). `query.Handler` deprecation notice removed.

4. **`BackfillHandler` signature changed** — Now takes `*SSEBroker` instead of
   `event.SeekableJournal`. The broker's journal and payload transform are used
   directly, unifying SSE and REST backfill under a single codec configuration.
   `BackfillHandlerWithTransform` removed (consolidated — configure the
   transform on the broker via `WithPayloadTransform`).

### Migration

```diff
// Before (v3):
- import "github.com/larsartmann/go-cqrs-lite/event/v3"
- handler := http.BackfillHandler(journal)

// After (v4):
+ import "github.com/larsartmann/go-cqrs-lite/event/v4"
+ handler := http.BackfillHandler(broker) // broker must have WithReconnectJournal
```

Codec defaults: no action needed. Old data reads correctly. New data is CBOR.
To revert process-wide: `event.DefaultCodec = codec.JSONCodec{}`.

### Added

- **`HealthCheck` on `OwnedDBHandle`** — all SQL stores (event, snapshot,
  checkpoint, command, query) now inherit `HealthCheck(ctx)` via embedding.
  Previously only `*SQLEventStore` implemented it.
- **`SSEBroker.Journal()` and `SSEBroker.PayloadTransform()` accessors** —
  exposed for `BackfillHandler` and consumer introspection.
- **ADR-0053** — Unified codec default flip rationale and backward-compat
  guarantees.
- **Envelope backward-compat integration tests** — `kv.TestTypedStore_Migration_*`
  verify old raw JSON data reads through new CBOR-default stores, and mixed
  old+new data coexists correctly.
- **`storage/eventstore/` sub-package** — SQLEventStore, SQLSnapshotStore,
  SQLCheckpointStore extracted into focused package. Full backward compat via
  type aliases and constructor re-exports in `storage/`.
- **`storage/readmodel/` sub-package** — SQLKVStore extracted into focused
  package. Full backward compat via type aliases and constructor re-exports.
- **`WithShutdownDependency` integration tests** — now tested through real
  `stack.New()` constructor path with close-order tracking, not just struct
  literals.

#### Dead Code Cleanup

- **Deleted `event/arena_experiment.go`** — 36-line stub with zero consumers, no tests,
  and no real GC benefit (arena-allocating a struct header while its fields remain
  heap-allocated saves nothing). Removed `goexperiment.arenas` from `flake.nix`,
  `scripts/check-module-isolation.sh`, and all living docs.

#### Projectionhost Observability

- **`Host.LagPerProjection() map[string]time.Duration`** — per-worker lag keyed by
  projection name, for Prometheus dashboards with `WithLabelValues`. Returns 0 for
  workers that haven't processed any event yet (item 38).
- **`WorkerState.Lag` field** — `Lag time.Duration` populated in `snapshot()` via the
  new `worker.lagDuration()` method. Previously only available via the aggregate
  `Host.LagDuration()` (item 39).
- **`Reset(ctx, name, opts...)` with `WithPurgeDeadLetters()`** — projection reset
  now optionally purges dead-letter entries from the configured `DeadLetterStore`.
  Backward compatible: `Reset(ctx, name)` still works without purging (item 46).
- **`Host.LagDuration()` refactored** — now delegates to `worker.lagDuration()` for
  consistency (returns max lag across all workers).
- **6 new tests** — lag before/after processing, per-projection map, DLQ purge
  with/without flag, WorkerState.Lag in Status().

#### Scenario Projection Tests

- **`scenario.GivenProjection` tests** — added `ThenError`, multiple-events, and
  empty-events tests covering the projection DSL more thoroughly (item 48).

#### Race Detector Coverage

- **Full `-race` suite** — all 48 modules pass with race detector. Only
  `cmd/api-stability` fails (pre-existing: subprocess doesn't inherit
  `goexperiment.jsonv2` tags) (item 13).

#### Documentation

- **ADR-0043 Part B** — consumer operational guide for the two `DeadLetterEntry`
  types: decision tree, code examples for dispatch-side vs projection-side DLQ,
  and structural comparison table explaining why they can't merge (item 45).
- **README docs freshness** — fixed stale `testutil` API references
  (`MustNewCmd`→`NewCmd`, removed `ParseAggID`, `NoopCommandHandler{}`→`NoopCommandHandler()`)
  and `v2`→`v3` paths in `testutil/README.md`.
- **AGENTS.md** — updated projectionhost key patterns with `LagPerProjection()`,
  `WorkerState.Lag`, and `WithPurgeDeadLetters()` examples.
- **`projectionhost/README.md`** — added Status & Lag section with dashboard examples,
  Reset section with purge option.

#### P3 Polish & Cleanup

- **Restored bundle.go architectural comment** — documented the Bundle↔CatchUpSubscriber
  relationship (SeekableJournal + Subscriber + CheckpointStore fields compose into the
  replay-then-live projection pipeline) after dead `var _` code was removed.
- **Fixed histogram test hard-coded values** — `prometheus/exporter_test.go` now references
  `cqrsotel.CQRSHistogramBoundaries` directly instead of duplicating the literal. If boundaries
  change in `otel/`, the test tracks the real value.
- **Verified `nix flake check`** — passes after `scripts/check-module-layers.sh` changes.
- **Race detector verified** on `stack/` and `example/taskmanager/` — both pass with `-race`.
- **CBOR→JSON SSE e2e test** — `TestSSEHandler_PayloadTransform_CBOR_ToJSON_BrowserFlow`
  in `transport/http/sse_options_test.go` verifies CBOR events transform to JSON for browser
  consumption across all SSE delivery paths.
- **Fixed taskmanager integration test failures** — `example/taskmanager` now uses JSON codec
  (`event.DefaultCodec = codec.JSONCodec{}`) via `codec_init.go` to fix CBOR decode failures
  in the projection pipeline. Events are also human-readable in the database and SSE stream.

#### DLQ Admin Operations & SQLite Dead-Letter Store (`projectionhost`)

- **`DeadLetterStoreAdmin` interface** — production management operations for dead-letter stores:
  `Count(ctx) (int64, error)`, `ListPaged(ctx, projectionName, offset, limit)`,
  `PurgeBefore(ctx, before time.Time) (int64, error)`.
- **`SQLiteDeadLetterStore`** — persistent SQLite-backed dead-letter store (survives restarts).
  Full column layout, index strategy, and reconstruction docs in `projectionhost/doc.go`.
- **DLQ index optimization** — replaced redundant `idx_pdl_projection` with
  `idx_pdl_projection_time(projection_name, failed_at)` (covers List + pagination + ORDER BY)
  and `idx_pdl_failed_at(failed_at)` (covers List-all + PurgeBefore).
- **DLQ test coverage** — stress test (10k entries: Count, ListPaged, PurgeBefore), concurrent
  store test (20 goroutines × 50 entries = 1000 writes), corrupt-payload test (surfaces error
  with event ID, no panic).

#### VersionedSeekableJournal (`schema`)

- **`schema.VersionedSeekableJournal`** — wraps `event.SeekableJournal` with upcaster chains,
  enabling schema evolution for `projectionhost.New()` (which requires `SeekableJournal`).
  Cross-module integration test with `projectionhost.New()` included.
- **Property tests** (rapid, 100 iterations each) — upcaster chain (random depth + events),
  passthrough (unregistered types), ReadFrom (position-based seek with upcasting).
- **Mid-stream upcast error test** — 10 events, upcaster fails on event 5, error propagates
  from both ReadAll and ReadFrom (no panic, no partial results).
- **Benchmarks** — ReadAll no-upcasters (140µs), ReadAll 3-chain (7.5ms), ReadFrom 3-chain
  500 events (536µs).

#### SSE Transform & Replay Safety (`transport/http`)

- **`WithPayloadTransform`** — wire-format transcoding (e.g., CBOR→JSON for browsers) applied
  uniformly across all three SSE paths: live, replay, and backfill.
- **`BackfillHandlerWithTransform`** — REST backfill endpoint with the same payload transform.
- **`SSEReplayBudgetDisabled = -1`** sentinel — `WithReplayByteBudget(0)` now auto-defaults to
  the 8MB safety budget; pass -1 to explicitly disable budgeting.
- **Large-payload byte-budget test** — 100KB × 5 events under 250KB budget boundary verification.

#### Blind Store Encoding Envelopes (`codec`, `kv`, `snapshot`, `command`, `query`)

- **`codec.WrapEncode` / `codec.UnwrapDecode`** — ADR-0044 encoding stamps on blind stores.
  All four blind stores (kv, snapshot, command, query) are now self-describing: the codec is
  stamped on write and auto-detected on read. `UnwrapDecode` falls back to JSONCodec for
  backward compat with pre-envelope data.

#### Prometheus Custom Views (`prometheus`)

- **`WithViews(views ...metric.View) Option`** — custom metric views for the Prometheus exporter.
  Compose with `cqrsotel.NewCQRSViews()` to apply CQRS histogram boundaries.

#### Stack Health Checks & Shutdown Ordering (`stack`)

- **`HealthChecker` interface + `Bundle.HealthCheck(ctx)`** — pings the database and calls
  `HealthCheck` on every registered closer that implements the interface. Enables Kubernetes
  liveness/readiness probes.
- **`WithShutdownDependency(before, after string) Option`** — topological sort (Kahn's algorithm)
  for close-time dependency ordering. Projections drain before the event store closes. Cycles
  fall back to registration order.

#### Decider Hot-State Cache (`decider`)

- **`StateCache[State]` interface + LRU implementation** — incremental loads: on cache hit,
  `LoadFromVersion(cachedVer)` + fold delta → O(new events) instead of O(total events).
  `WithStateCache[State]` option enables it. Cache updated on every Execute, invalidated on
  fold/store errors. Benchmark: 7.4x faster Load (2090→283 ns/op) with 500-event history.
  Process-local, best-effort, zero new dependencies.

#### Read-Pressure Snapshot Strategy (`snapshot`)

- **`ReadPressure` strategy** — triggers snapshots based on read count (hot-read, cold-write
  aggregates). `AggregateAwareStrategy` and `ReadTracker` optional interfaces.
  Composable with `EveryNEvents` via `WithInnerStrategy`. Wired into decider Repository via
  optional interface checks. Fully backward compatible.

#### id/ + metadata/ Package Extraction

- **`id/` package** — branded IDs (`AggregateRef`, `EventID`, markers) extracted from `event/`
  into a standalone, zero-event-dependency module.
- **`metadata/` package** — `Tracing`, `CustomData[K]`, and shared metadata types extracted from
  `event/` for cross-module reuse (command, query, event).

#### SQL Error Classification Auto-Registration (`storage/sql`)

- **`errorfamily.RegisterStdlibDefaults()`** called via `init()` — registers stdlib error
  classifications automatically on import.
- **Database driver classifiers** — SQLite BUSY/LOCKED→Transient, CONSTRAINT→Conflict;
  Postgres SQLSTATE class mappings. Registered via `init()` in `storage/sql/classify_init.go`.

#### Idempotency Middleware — Generic Factory (`middleware/v4`)

- **`middleware.NewIdempotency[M]`** — generic idempotency middleware factory following the
  `NewValidation[M]` / `NewTracing[M]` pattern. Works for all 3 CQRS message types:
  - **`middleware.CommandIdempotency(store, ttl, keyExtractor)`** — command dedup using the
    command's minted ID by default (pass `nil` for keyExtractor).
  - **`middleware.EventIdempotency(store, ttl, keyExtractor)`** — event dedup using the event's
    minted ID by default (pass `nil` for keyExtractor). For ordered event consumption (projections),
    checkpoint-based dedup (`projectionhost`) is structurally stronger — use this when you don't own
    the checkpoint (webhooks, external sinks, cross-system delivery).
  - **`middleware.QueryIdempotency(store, ttl, keyExtractor)`** — query dedup. Requires a non-nil
    keyExtractor (queries have no built-in identity). Panics at construction if nil.
- Store errors are classified as `Transient` via `errorfamily.Wrapf`. Duplicate keys return
  `idempotency.ErrDuplicate` (a `Conflict` family error).

#### Documentation & ADRs

- **ADR-0043** — Dead-letter store design (dispatch-side vs projection poison entries).
- **ADR-0044** — Blind store encoding stamps (envelope wrapper).
- **ADR-0047** — json/v2 case-insensitive decode.
- **ADR-0048** — Deterministic encoding.
- **ADR-0049** — Dispatch-time middleware ordering.
- **SECURITY.md** — vulnerability reporting process.
- **Consumer migration guide** — `docs/migration/MIGRATION-GUIDE.md` for id/ + metadata/ extraction.
- **SKILL.md** updated — `VersionedSeekableJournal`, `BackfillHandlerWithTransform`, `WithViews`
  added to decision matrix + cheat sheet. doc-check passes (868 refs).
- **metadata/ + id/** added to AGENTS.md module table.
- **Deprecated alias cleanup** — 8 deprecated aliases deleted from `event/` + `schema/`
  (AggregateRef, Tracing, CustomData, etc.). Internal usage migrated to `id.` and `metadata.`.
  `event.WithNewCodec` removed (use `WithCodec`). `event.WithReplay` removed (use `WithProcessingMode`).
  `query.Handler` deprecation notice removed — it is the dispatch core, not deprecated.

### Changed

#### CBOR is the Default Codec (`event`, `codec`, `stack`)

- **`event.DefaultCodec`** is now `codec.CBORCodec{}` (was `JSONCodec{}`). Events are
  self-describing (`evt.Encoding()` stamp on every event), so mixed JSON+CBOR streams decode
  correctly via `DecodePayloadAuto`. Blind stores are self-describing via ADR-0044 envelopes.
  Blind store defaults (kv, snapshot, command, query) also flipped to CBOR.

#### Deprecated Alias Cleanup

- **~200 usages across 42 files** updated from `event.AggregateRef` → `id.AggregateRef`,
  `event.Tracing` → `metadata.Tracing`, etc. All internal code now uses `id.` and `metadata.`
  directly. SA1019 deprecated alias warnings eliminated across all modules.

#### JSON Quality Audit

- **`Deterministic(true)`** added to all `Marshal` calls in signing, encryption, event, storage,
  transport, listing, catalog.
- **`MatchCaseInsensitiveNames(true)`** added to all `Unmarshal` calls across all modules.
  Implements ADR-0047 (case-insensitive decode) and ADR-0048 (deterministic encoding).

#### errorfamily.HTTPStatus() Adoption (`example/taskmanager`)

- **`writeCQRSError`** simplified from 15-line switch statement to a 1-line
  `errorfamily.HTTPStatus(err)` call.

#### Dispatcher Middleware-at-Dispatch-Time Fix (`dispatcher`)

- Middleware can now be added in any order — the chain is rebuilt at dispatch time, not
  construction time. Documented in `dispatcher/doc.go`.

#### CI Sync Scripts

- **`scripts/check-workspace-sync.sh`** — verifies go.work ↔ flake.nix module sync. 8 missing
  modules added to flake.nix testModules.
- **`scripts/check-api-stability-sync.sh`** — verifies go.work ↔ api-stability tracking sync.
  12 missing modules added to api-stability tracking.
- **`scripts/check-module-layers.sh`** — dependency budget violations fixed (deriver=4,
  stack=14). projectionhost raised 7→9, watermill raised 8→9 (SQLite DLQ + metadata extraction).

#### Idempotency Module Slimmed Down (`idempotency/v4`)

- Removed `idempotency.CommandIdempotency`, `idempotency.KeyExtractor`, and
  `idempotency.CommandIDKey` — replaced by the generic `middleware.CommandIdempotency` factory.
- Module dependencies reduced: `command/v4` and `id/v4` dropped from direct deps. Now depends on
  `kv/v4` + `go-error-family` only.
- Layer changed from Layer 2 (→command, event, id, kv) to Layer 1 (→kv).
- Added to `flake.nix` testModules and `cmd/api-stability` module tracking (was missing from both
  since module creation).
- Pre-existing lint issues fixed: `exhaustruct`, `nestif`, `revive` (unused ctx), `wrapcheck`.

### Fixed

- **`WithReplayByteBudget(0)` semantics** — 0 now auto-defaults to the 8MB safety budget;
  `SSEReplayBudgetDisabled = -1` explicitly disables budgeting.
- **`api_surface.txt`** — removed dead `JSONCodecV2` entry. Regenerated golden with all new
  modules tracked (2212 exports).
- **File-size violations** — 3 production files split under the 350-line CI limit:
  `signing/cose.go` → `cose_sign1.go`, `cmd/doc-check/main.go` → `exports.go`,
  `catalog/eventcatalog/frontmatter_render.go` → `frontmatter_convert.go`.
- **Dead code removed** — `codec/jsonv2_experiment.go` (dead Go experiment tag gated zero files).
  All 4 `var _ =` hacks removed (`sse_backfill.go`, `example/taskmanager/http.go`,
  `stack/bundle.go`, `example/taskmanager/setup.go`).

### Security

- **SECURITY.md** — documents the vulnerability reporting process.

## [3.7.1] - 2026-07-07

**Release documentation completeness — all 48 modules synced to v3.7.1.**

v3.7.0 was published with 46 modules tagged (otel skipped as unchanged). This
patch releases all 48 modules at a uniform version for consumer dependency
alignment, and adds the CHANGELOG/version-string updates that v3.7.0 shipped
without.

### Fixed

- **CHANGELOG.md** — added [3.7.0] section (was missing from the v3.7.0 release).
- **flake.nix** — package version bumped to 3.7.0 (was stale at 3.6.0).
- **v4-WISHLIST.md** — "Current major" updated to v3.7.0 (was stale at v3.4.0).
- **otel/v4.7.0** tagged for version-line consistency (module unchanged since v3.5.0).

### Verified

- **govulncheck**: 0 vulnerabilities across all 48 modules.
- **All gates green**: build, test, lint, isolation (GOWORK=off), version drift.

## [3.7.0] - 2026-07-07

**Dedup module extraction, SSE production hardening, go-error-family direct adoption, SQLTimerStore.**

### Added

#### Dedup — Bounded Dedup Ring Buffer (`dedup/v4`, first release)

- **`dedup.Ring`** — O(1) fixed-capacity ID deduplication for stream boundaries.
  Extracted from the inline SSE and watermill implementations into a reusable
  module. Used by `projectionhost`, `watermill`, and `transport/http` (SSE).

#### SSE Production Hardening (`transport/http`)

- **Fanout and drop policies** for high-fanout deployments — configurable behavior
  when subscriber count exceeds budget.
- **Backfill REST endpoint** — query missed events by aggregate or timestamp range.
- **Auth middleware** — pluggable authentication for SSE connections.
- **Offline reconnection example** — reference pattern for resilient clients.
- **Byte-budget replay** — stops mid-batch when a configurable byte limit is
  exceeded (prevents memory blowups on large replays).
- **Replay timeout** — caps replay duration; sends an advisory event on timeout
  before live streaming begins.

#### ProjectionHost Graceful Teardown (`projectionhost`)

- **`WorkerDraining` status** — workers transition through Draining before Stopped,
  enabling graceful shutdown that respects in-flight events.

#### SQLTimerStore (`storage`)

- **`SQLTimerStore`** — persistent `scheduling.TimerStore` backed by SQL, enabling
  durable deadline timers that survive restarts.

#### Watermill Batched Replay (`watermill`)

- **CatchUpSubscriber replay** now batches historical events into fixed-size chunks
  instead of loading the entire backlog at once.

#### Pebble GracefulClose (`stack/pebble`, `storage/pebble`)

- **`GracefulClose(ctx)`** — bounds `Close()` with a timeout, preventing hung
  shutdowns on slow flushes.

### Changed

#### Go-Error-Family Direct Adoption

- All modules now import `go-error-family` directly instead of through the `event/`
  package facade. The `event/` package retains type aliases (`event.Family`,
  `event.Error`) for backward compatibility, but error construction and
  classification functions now use `go-error-family` directly.
- **`go-error-family` bumped to v0.6.1.**

#### Turso Database Rebrand

- "LibSQL" terminology replaced with "Turso Database" across the codebase and
  documentation.

### Fixed

- **dedupRing panic** — removed panic from constructor on invalid capacity; returns
  error or falls back to default.
- **Prometheus provider shutdown** — now returns nil on successful shutdown.
- **Tombstone projection** — persists correctly across KV store roundtrip.
- **gRPC test nil-deref** — guard added.
- **Pattern B sentinels** — replaced placeholder sentinels with real versions for
  external consumption.

### Infrastructure

- **47 modules tagged at v3.7.0** (including first-ever `dedup/v4.7.0` and
  version-line-consistency tag for `otel/v4.7.0`).
- Replace directives completed across all modules for GOWORK=off build correctness.
- Go toolchain at 1.26.4.

## [3.6.0] - 2026-07-05

**Error-family taxonomy full sweep, deriver module, flagship example consolidation.**

### Added

#### Deriver — Event→Command Derivation (`deriver/v4`, `example/taskmanager`)

- **`deriver.Deriver`** — reacts to events by deriving new commands. Chainable `Then`,
  `Filter`, `Idempotent`, and `AsHandler` operators for declarative event→command
  pipelines. Implements ADR-0040.
- **Taskmanager example** — auto-assigns new tasks via a `user.created` →
  `task.assign` derivation, demonstrating real-world usage.

#### Flagship Example Consolidation

- **9 examples → 2**: the scattered `deployer-first`, `deployer-first-multidb`,
  `deployer-first-heterogeneous`, `encryption`, `deriver`, `graph-demo`,
  `projectionhost`, `todo`, and `user` examples are consolidated into:
  - **`example/taskmanager`** — the complete reference: event sourcing, projections
    (KV + tombstone), SSE streaming, snapshot strategy, signing, ProjectionHost with
    DLQ, deriver integration.
  - **`example/getting-started`** — minimal getting-started guide.

### Changed

#### Error Family Taxonomy — Full Sweep

Adopted the 5-family error taxonomy (Rejection / Conflict / Transient /
Infrastructure / Corruption via `go-error-family`) across all production modules:

| Module                 | Classification                                                                |
| ---------------------- | ----------------------------------------------------------------------------- |
| `storage`              | `WrapInfrastructure` for event store streams, memory streams, PG bus listener |
| `storage/pebble`       | `WrapInfrastructure` for backend, command read, iteration paths               |
| `storage/relational`   | `WrapInfrastructure` for projection, schema, sink                             |
| `storage` (KV SQL)     | `WrapTransient` for idempotency KV store                                      |
| `middleware`           | `WrapInfrastructure` for dead-letter SQL store                                |
| `catalog/eventcatalog` | `WrapCorruption` for frontmatter marshal                                      |
| `projectionhost`       | `WrapInfrastructure` for dead-letter list                                     |
| `cmd/cqrs-gen`         | `WrapInfrastructure` for scan/walk/parse                                      |
| `stack/sqlite`         | `WrapInfrastructure` for preset errors                                        |
| `stack/postgres`       | `WrapInfrastructure` for preset + `WrapRejection` for bad DSN                 |
| `stack/pebble`         | `WrapInfrastructure` for preset errors                                        |
| `stack/turso`          | `WrapInfrastructure` for preset errors                                        |
| `idempotency`          | `WrapTransient` for KV store                                                  |
| `command`              | Taxonomy for memory bus + typed store                                         |
| `graph`                | Taxonomy for memory driver                                                    |

### Fixed

- **Tombstone projection persistence** — tombstone marks now survive KV store
  roundtrips correctly (`example/taskmanager/projection.go`).
- **Event signing middleware wiring** — signing middleware now correctly wired via
  EventBus type assertion instead of direct `UsePublish`.
- **eventtest module path** — moved to `event/v4/eventtest/` to match the Go module
  path spec for VCS resolution (ADR-0045). Fixes `go mod tidy` warnings.
- **Invalid v0 pseudo-versions** — corrected pseudo-versions for `/v4` module paths
  in cross-module `go.mod` dependencies.
- **go.mod/go.sum stabilization** — convergence tidy across all modules; workspace
  replace directives aligned for consistent local resolution.

## [3.5.0] - 2026-07-01

**CBOR promoted to first-class default, encoding-aware validator, symmetric validation.**

### Added

#### CBOR Adoption Primitives — `event/v4`, `stack/v4`

- **`event.DefaultCodec`** — mutable package-level variable (like `http.DefaultClient`)
  that controls the codec used by `event.New()` when no `WithCodec` option is passed.
  Defaults to `JSONCodec{}` for backwards compatibility. Set to `CBORCodec{}` for
  process-wide CBOR adoption: `event.DefaultCodec = codec.CBORCodec{}`.
- **`stack.WithEventCodec(c codec.Codec) Option`** — one-call adoption for both event
  payloads and read models. Sets `bundle.EventCodec()` and also `bundle.DefaultCodec()`.
  Consumers use `bundle.EventCodec()` in decide functions via `event.WithCodec()`.
- **`Bundle.EventCodec()`** — accessor for the event payload codec. Falls back to
  `event.DefaultCodec` when unset.

#### Codec Utilities — `codec/v4`

- **`AutoDetect(data []byte) Encoding`** — sniffs the serialization format from raw
  bytes by examining structural first-byte patterns. Distinguishes JSON from CBOR.
  Best-effort heuristic for diagnostics and tooling, not a security boundary.
- **`Size(v any) (jsonSize, cborSize int)`** — encodes v with both codecs and returns
  the byte sizes. Useful for evaluating CBOR adoption before committing.
- **`keyasint` example** — `ExampleCBORCodec_keyasint` demonstrating CBOR integer keys
  (CWT claim registry pattern) for 22% size reduction over string keys.

#### gRPC Codec Injection — `transport/grpc/v4`

- **`WithCodec(c codec.Codec) Option`** — shared functional option for
  `RegisterQueryService`, `NewQueryClient` (and future command/event transport).
  Defaults to JSON for backwards compatibility. Both server and client must use the
  same codec.
- **`QueryServer.codec`** — query results are encoded with the configured codec
  instead of hardcoded `json.Marshal`.
- **`QueryClient.codec`** — query results are decoded with the configured codec
  instead of hardcoded `json.Unmarshal`.

#### Encryption Encoding Fix — `encryption/v4`

- **Encoding preservation through middleware** — `AttachEncryption` and
  `decryptEvent` now preserve the original event's `Encoding()` stamp. Previously,
  CBOR events lost their encoding during the encrypt → decrypt cycle, causing
  `DecodePayload` to fail. JSON events were unaffected (the default).
- **`NewCodec` doc comment** — warns that `encryption.NewCodec` is for non-event
  serialization. For event payloads, use `EncryptMiddleware`/`DecryptMiddleware`,
  which preserves the encoding stamp.

#### Encryption Validation Tests — `schema/v4`

- **`TestValidator_EncryptedEncoding_RejectedGracefully`** — encrypted events
  (encoding="encrypted") produce a clean Rejection error, not a panic.
- **`TestValidator_UnknownEncoding_FallsBackToJSON`** — unknown encodings fall
  back to the JSON decoder.
- **`TestValidator_EncryptedEncoding_WithCustomDecoder`** — consumers can register
  a custom decoder for the "encrypted" encoding.

#### Mixed-Stream Decode — `codec/v4`, `event/v4`

- **`codec.ForEncoding(enc Encoding) (Codec, error)`** — resolves the built-in codec
  for a given encoding stamp. Returns `JSONCodec` for JSON, `CBORCodec` for CBOR,
  and an error for unknown encodings. The codec-level counterpart to `AutoDetect`.
- **`event.DecodePayloadAuto[T](evt) (T, error)`** — decodes an event's payload by
  dispatching to the codec matching the event's `Encoding()` stamp via `ForEncoding`.
  This fulfills the mixed-stream promise: JSON and CBOR events in the same store
  decode correctly without the caller knowing or passing the codec. Previously,
  `DecodePayload` rejected events whose encoding didn't match the caller-provided
  codec — making JSON→CBOR migration impossible without manual branching.

#### gRPC Query Tests — `transport/grpc/v4`

- **Query round-trip test coverage** — the query gRPC service had ZERO test coverage.
  Added tests for JSON round-trip, CBOR round-trip (with `WithCodec`), handler error
  propagation, and codec mismatch detection.

#### Encryption Integration Test — `integration/v4/encryption`

- **CBOR event through encrypt→decrypt** — integration test verifying CBOR events
  survive the encrypt→bus→decrypt cycle with encoding stamp preserved, and
  `DecodePayloadAuto` dispatches correctly post-decryption.

#### Documentation

- **`docs/migration/JSON_TO_CBOR.md`** — comprehensive migration guide with
  step-by-step instructions, decision matrix, and encryption guidance.
- **`docs/adr/0044-blind-store-encoding-stamps.md`** — design doc for v4 envelope
  wrapper to add encoding stamps to blind stores.
- **AGENTS.md codec default asymmetry table** — documents which layer defaults to
  which codec and how to override each.
- **`example/deployer-first`** — refactored to use `event.New()` with typed payloads
  (instead of pre-marshaled JSON bytes) and `stack.WithEventCodec(CBORCodec{})`.

#### CBOR as Recommended Default — `codec/v4`

- **CBOR listed first** in README, doc.go, and examples with "Recommended" badge.
  JSON remains fully supported as the interop/debugging codec.
- **`CBORCompactCodec`** — stricter CBOR (RFC 8949 Core Deterministic) with
  unknown-field rejection on decode, enabling schema drift detection.
- **`BufferEncoder` interface** — zero-allocation encoding via `EncodeToBuffer(v, buf)`.
  Implemented by `JSONCodec`, `CBORCodec`, and `CBORCompactCodec`.
- **Streaming CBOR** — `NewCBOREncoder`/`NewCBORDecoder` for batch encoding without
  materializing the full byte slice.
- **`Diagnose(data)`** — converts CBOR bytes to human-readable diagnostic notation
  for debugging.
- **Exported `CBOREncMode()`/`CBORDecMode()`** — shared canonical encoding modes so
  storage backends use one deterministic CBOR configuration.
- **6 new runnable examples** — CBORCompactCodec, toarray, BufferEncoder, streaming,
  Diagnose, CBOREncMode.
- **Realistic benchmarks** — `realisticOrder` struct with nested items. Results:
  CBOR 19% smaller than JSON, CBOR+toarray 43% smaller. Decode: CBOR 66% faster,
  CBOR+toarray 72% faster.
- **Property-based roundtrip tests** (`pgregory.net/rapid`) — 4 tests proving
  JSON, CBOR, CBORCompact all roundtrip correctly, plus CBOR determinism property.

#### Stack-Level Default Codec — `stack/v4`

- **`WithDefaultCodec(c codec.Codec) Option`** — set a bundle-level default codec.
  Defaults to `CBORCodec{}` (changed from JSON).
- **`Bundle.DefaultCodec()`** — returns the configured default codec.
- **`ReadModel()` and `NewMaterialize()`** — use `DefaultCodec()` instead of
  hardcoded `JSONCodec{}` when the caller passes nil codec.

#### Encoding-Aware Validator — `schema/v4`

- **`WithCodec(c codec.Codec) ValidatorOption`** — replaces the old
  `func([]byte, any) error` parameter with a type-safe `codec.Codec` interface.
  The codec's `Encoding()` determines which encoding the decoder handles.
- **`WithDecodeFunc(fn) ValidatorOption`** — backward-compatible deprecated alias
  for the old `WithCodec` raw-function API. Will be removed in v4.
- **`WithDecoder(enc, fn) ValidatorOption`** — register a decode function for a
  specific encoding.
- **Auto-detected CBOR** — the validator now auto-detects event payload encoding
  via `evt.Encoding()` and picks the matching decoder. JSON and CBOR work
  out of the box with no configuration.

### Changed

#### Symmetric Encoding Validation — `event/v4`

- **`validateEncodingMatch` is now symmetric.** Previously, JSON events got a free
  pass — a JSON event decoded with CBORCodec would bypass validation and fail with
  a confusing corruption error. Now ALL encodings are compared equally:
  `evtEnc != codecEnc`. Mismatches in either direction produce a clear
  `event.encoding_mismatch` Rejection error immediately.

### Documentation

- **`codec/README.md`** — full rewrite. CBOR listed first with "Recommended" badge,
  "When to Use" decision table, struct tag guide (toarray/keyasint/omitzero),
  BufferEncoder, streaming, shared CBOR modes, diagnostic notation.
- **`codec/doc.go`** — updated from "Three implementations" to "Four implementations".
  Added "Choosing a Codec" section.
- **`AGENTS.md`** — added toarray, BufferEncoder, streaming, and `WithDefaultCodec`
  code patterns.
- **`SKILL.md`** — cheat sheet changed from `JSONCodec{}` to `CBORCodec{}` with
  "recommended" note.
- **`kv/typed_options.go`** — `WithTypedCodec` doc mentions `stack.Bundle.DefaultCodec`.

### Migration Notes

- **`schema.WithCodec` signature changed** from `func([]byte, any) error` to
  `codec.Codec`. The old function signature is preserved as `schema.WithDecodeFunc`
  (deprecated). Migrate by replacing `WithCodec(json.Unmarshal)` with
  `WithCodec(codec.JSONCodec{})`.

## [3.4.0] - 2026-06-29

**Managed projection host maturity, durable scheduling, scenario-testing DSL, go mod tidy sweep.**

### Added

#### Managed Projection Host — `projectionhost/v4`

- **`Host`** — managed lifecycle for projection workers: per-projection
  goroutines, crash auto-restart with exponential backoff, checkpoint
  persistence, and a poison-message dead-letter queue. The "last loop every
  consumer rewrites", now a library module (framework gap A1).
- **`ReplayDeadLetters`** — re-feeds dead-letter entries to the matching
  projection after a handler fix; purges successful replays. `DeadLetterEntry`
  now carries the original `event.Event` so replay is possible.
- **`WithLogger(*slog.Logger)`** — inject a structured logger for worker
  lifecycle events (crashes, restarts, DLQ captures). Default: `slog.Default()`.
- **`MemoryDeadLetterStore`** — in-memory `DeadLetterStore` for dev/test.

#### Scenario-Testing DSL — `scenario/v4`

- Fluent BDD harness: `Given[Cmd,State](t, apply, initial, events...).When(cmd,
decide).Then(types...)`, plus `ThenError`, `ThenState`, and projection
  `GivenProjection/ThenNoError` (framework gap A5).

#### Scheduling — `scheduling/v4`

- Durable deadline timers: `TimerStore` (`Schedule`/`Due`/`MarkFired`/`Cancel`),
  `MemoryTimerStore`, and `Scheduler` with configurable poll interval and retry.
  Idempotent scheduling (framework gap A6) — "cancel order after 30 min unpaid".

#### Pebble `kv.ConditionalWriter`

- **`KVAdapter.SetIfAbsent`** — atomic compare-and-set on the Pebble KV adapter,
  unlocking `idempotency.KVStore` support on the Pebble backend. Serialized via
  a per-adapter mutex (process-local guarantee, matching `kv.MemStore`).

#### Brutal Self-Review Pass (2026-06-29)

- **`projectionhost.MetricsRecorder`** — zero-dependency metrics interface
  with `WithMetrics()` option. Five lifecycle methods: EventProcessed,
  EventErrored, EventDeadLettered, WorkerRestarted, CheckpointAdvanced.
  Consumers wire Prometheus/OTel/Datadog; host stays backend-agnostic.
- **`projectionhost.DeadLetterStore.Delete`** — entry-scoped removal
  (`Delete(ctx, name, eventID)`); callers can now surgically clear
  successfully-replayed entries instead of purging the whole projection.
- **`projectionhost` jitter backoff** — worker restart backoff now uses full
  jitter (stdlib `math/rand/v2`) to prevent thundering-herd restarts. No new
  dependency.
- **`scheduling` retry backoff** — dispatch retries now use exponential
  backoff with full jitter between attempts, with a new `WithRetryDelay`
  option. Previously retried with zero delay.
- **`testutil.CapturingSlogHandler`** — shared slog test handler, replacing
  two near-identical copies (`capturingSlogHandler` in projectionhost and
  `capturingHandler` in scheduling).
- **`example/deriver`** — runnable demo of the stateless-saga derivation
  pattern (the deriver module previously had zero consumers/examples).
- **ADR-0042** (pure replay design) and **ADR-0043** (DLQ unification options).

### Changed

- **`testing/v4` renamed to `scenario/v4`** — avoids collision with Go's stdlib
  `testing` package in import paths. The package name is now `scenario`
  (`scenario.Given[...]`). Consumers importing `testing/v4` must update to
  `scenario/v4`.
- **`scheduling.WithLogger`** — previously a no-op (discarded the logger); now
  correctly wires the injected `*slog.Logger`.
- **`scenario.DecideFunc` doc** — corrected the false "import cycle" claim;
  the real reason for decoupling is dependency footprint, not a cycle.
- **`projectionhost/example` lint** — cleared 21 shipped golangci-lint warnings
  (sentinel error, named const, unused-param fix).

### Migration Notes

- **`scheduling.Timer` is now generic (`Timer[P any]`)** — `Timer`, `TimerStore`,
  `MemoryTimerStore`, `DispatchFunc`, and `Scheduler` all require a payload type
  parameter. Migrate by adding it at the call site:
  `scheduling.NewMemoryTimerStore()` → `scheduling.NewMemoryTimerStore[YourCmd]()`,
  `scheduling.Timer{...}` → `scheduling.Timer[YourCmd]{...}`.
- **`command.Command.ID()` (v3.1.0 → v3.3.0)** — the `command.Command`
  interface gained a mandatory `ID() id.CommandID` method for idempotency
  support. Consumers upgrading from v3.1.0 must add `ID()` to every command
  type implementing `command.Command`.

## [3.3.0] - 2026-06-28

**Three projection tiers, unified command identity, production dead-letter storage.**

### Added

#### SQL-Backed Dead-Letter Store

- **`middleware.SQLDeadLetterStore`** — persistent dead-letter handler backed by
  SQLite or PostgreSQL. Auto-creates the `dead_letters` table, survives process
  restarts. Implements `DeadLetterHandler` — drop-in replacement for
  `MemoryDeadLetterStore` in `RetryConfig.OnDeadLetter`.

#### Row Column-Name Validation

- **`storage.ProjectionSink`** methods (Upsert/Ensure/Update/DeleteWhere/QueryOne)
  now validate column and table names against `RelationalSchema` before SQL
  execution. Catches typos at the application boundary. New sentinel errors:
  `errSinkUnknownColumn`, `errSinkUnknownTable`.

#### Denormalization Guidance

- **`storage.RelationalStore`** documented decision: single-table queries only.
  For multi-table reads, denormalize FK columns in the projection handler.
  No JOIN API — intentional boundary (the projection tier's promise is "no raw SQL").

### Changed

#### Breaking: Command ID Unification

- **`command.Command` interface** now requires `ID() id.CommandID`. Every command
  gets a stable, auto-minted ID at construction time via `command.New()`.
  Override with the new `command.WithCommandID` option for idempotency-key replay.
- **`command.WithCommandID` (PersistOption)** renamed to
  `command.WithPersistedCommandID` to avoid name collision.
- **Migration:** any type implementing `command.Command` must add `ID()`.
  Embed `command.BasicCommand` to inherit it automatically.

#### Watermill Command Bridge

- **`watermill.CommandToMessage`** now uses `cmd.ID()` instead of minting an
  ephemeral ID per call. Same command instance → same message UUID (stable for
  dedup). Different instances → different UUIDs (auto-minted in `New()`).
- **`watermill.MessageToCommand`** now parses and preserves the command ID
  round-trip (previously discarded).

#### Transport/gRPC

- **`transport/grpc`** now carries `command_id` in envelope metadata. Server
  preserves the client's command ID through dispatch.

#### Zero Lint Findings

- All 46 modules now lint clean. Previous 8 issues resolved:
  stack (contextcheck, errname, wrapcheck, unused), middleware (exhaustruct),
  transport/grpc (gosec G115, containedctx, nolintlint).

### Documentation

- **All research docs stamped** with status markers (RESOLVED/IMPLEMENTED/SUPERSEDED).
  Every doc in `docs/research/` now clearly indicates whether it's live or historical.
- **ROADMAP.md updated** — module count (43→46), transport adapters (NATS/Redis
  superseded by Watermill), three projection tiers marked done.
- **Graph tier scope documented** — MemoryDriver is the v3.x ship target.
- **`go.work` genproto replace** — explanatory comment added.

### Added

#### catalog/v4.2.0

- **`catalog/simple` sub-package** — single-service Builder facade (`New`,
  `Command[T]`, `Query[T]`, `Event[T]`, `Build`, `BuildValid`) with auto-kebab
  service ID via `internal/caseutil.ToKebab`. Streamlines the common case of
  documenting one service.
- **`catalog/docserver` standalone handlers** — `D2Handler` (D2 architecture
  diagram over HTTP), `HealthCheckHandler` (liveness probe verifying the
  catalog has services), `GenerateEventCatalog` (writes EventCatalog MDX files
  at startup). These complement the existing `DocsServer` for lighter use cases.

#### New Module: `projection/`

- **`projection.Projection`** interface and `projection.NewProjection` — extracted
  from `event/` to a dedicated module. The Projection interface is a consumer-side
  abstraction; it belongs with consumers, not with the event producer module.
  Implements proper dependency-direction: `projection → event` (consumer → producer),
  never the reverse.

#### New Module: `graph/`

- **`graph.GraphProjection`** — third projection tier (nodes + edges) for
  traversal-heavy read models. Merges events into graph structures via a
  transactional `GraphSink`. Writes are portable across openCypher backends
  (Neo4j, Memgraph, Apache Age). `MemoryDriver` provides a zero-dep reference
  implementation.

#### New Module: `storage.RelationalProjection`

- **`storage.RelationalProjection`** — multi-table, dialect-portable SQL projection
  with a transactional `ProjectionSink`. Atomic cross-table writes per event.
- **`storage.RelationalStore`** — read-side companion (Count/CountMany/Query).

#### Architecture Enforcement via go-arch-lint

- **`scripts/check-arch.sh`** — two-layer architecture enforcement:
  Layer 1 = cross-module rules via `check-module-layers.sh` (go.mod parsing);
  Layer 2 = intra-module package rules via go-arch-lint (per-module configs).
  Wired into flake.nix as `nix run .#check-arch`.
- **`.go-arch-lint.yml`** (workspace-level) — documents the 7-layer module model.
  Rewritten from stale config that referenced 6 deleted directories.
- **`storage/.go-arch-lint.yml`** — first per-module config, enforces intra-module
  package dependency rules.
- **Per-module configs for `event/`, `command/`, `middleware/`, `kv/`, `catalog/`** —
  extends Layer-2 architecture enforcement to the largest unchecked modules.

### Changed

#### Breaking: `event.Projection` moved to `projection/`

- `event.Projection` → `projection.Projection`
- `event.NewProjection` → `projection.NewProjection`
- **Migration:** change imports from `event/v4` to `projection/v4` for Projection
  types. All other event types (`Event`, `Type`, `Store`, etc.) remain in `event/`.
- **Rationale:** Projections are event CONSUMERS. The Projection interface had zero
  internal consumers in `event/` — it was a layering inversion. Moving it establishes
  correct dependency direction.

#### Relational Store Query Contract

- **`RelationalStore.Query` now accepts `kv.ViewQuery`** — removes the duplicate
  `storage.RelationalQuery` type. The relational read side now shares the same
  filtered/ordered/paginated query contract as `kv.ViewStore` implementations.

### DX Improvements

#### Bundle.RunProjections — One-Call Projection Runner

- **`bundle.RunProjections(ctx, projections...)`** — replays journal + subscribes to
  live + dispatches to all registered projections. Eliminates ~20 lines of
  CatchUpSubscriber + channel consumption + message decoding boilerplate.
- **`stack.Materialize` now implements `projection.Projection`** — added
  `Name()`, `Handle()`, `EventTypes()` methods. Fixes the split brain where
  Materialize returned Watermill's `NoPublishHandlerFunc` but bypassed the
  library's own `Projection` contract. All three projection tiers now satisfy
  the same interface.

### Tests & Infrastructure

#### Graph Contract Test Suite

- **`graph/graphtest/contract.go`** — shared behavioral contract test for
  `GraphDriver` implementations (mirrors `kv/viewstoretest/contract.go`).
  7 tests: MergeNodeCreates, MergeNodeUpdates, MergeEdgeCreatesEndpoints,
  MergeEdgeUpdatesProps, RemoveNodeDeletesIncidentEdges, RemoveEdgeLeavesEndpoints,
  AtomicRollbackOnError. MemoryDriver passes all 7.

#### Architecture Enforcement

- **`scripts/check-arch.sh`** — two-layer arch enforcement (cross-module via
  go.mod parsing + intra-module via go-arch-lint). Wired as `nix run .#check-arch`.
- **`storage/.go-arch-lint.yml`** — first per-module arch-lint config.
- Stack dep budget bumped from 12 to 13 (added `projection/v4` dependency).

#### ADRs

- **ADR-0037**: Projection interface extraction from `event/`
- **ADR-0038**: Graph projection tier design (writes portable, reads native)
- **`docs/projection-tiers.md`**: Decision guide for choosing between tiers

#### Quality

- **`projection/` module: 100% test coverage** (5 tests)
- **`graph/` module: 86.9% coverage** (9 tests + 7 contract tests)

#### Workspace Integration

- **`transport/grpc` is now wired into `go.work`** — resolves the long-standing
  `google.golang.org/genproto` ambiguous-import conflict via a workspace-level
  replace directive. The module builds and tests as a first-class workspace member.
- **BuildFlow pre-commit hook budget increased** from 60s to 300s — eliminates the
  need for `--no-verify` on commits.

#### RunProjections Test Coverage

- **`stack/run_projections_test.go`** — end-to-end test covering journal replay,
  live event handoff, materialized-view updates, and clean shutdown via context
  cancellation.

## [3.1.0] - 2026-06-25

**Feature release — 79 commits since v3.0.0, +69 API exports (1558 → 1627), zero breaking changes.**

### Added

#### SQL-Backed View Stores & Queryable Read Models

- **`storage.SQLViewStore`** — SQL-backed `kv.ViewStore` with column-mapped views. Supports `Query` (WHERE + ORDER BY + LIMIT/OFFSET), `Count`, `BatchSet` (chunked upsert, SQLite 999-param aware), `DeleteAll`, and `Scan`. Tombstone column support for server-side filtering.
- **`storage.ViewMapper[V]`** — declarative column mapping: table name, columns with extractors, `ScanRow`, optional `TombstoneColumn` and `Indexes`.
- **`storage.AutoMapper` / `AutoMapperWithTombstone`** — generates a `ViewMapper` from struct tags (field name → column name).
- **`storage.NewSQLiteViewStore` / `NewSQLViewStore` / `NewViewStoreWithDialect`** — constructors with auto-migration.
- **`kv.ViewStore` interface** — `ViewQuerier`, `ViewCounter`, `ViewBatchSetter`, `ViewResetter`, `TombstoneQuerier` optional interfaces checked at runtime.
- **`kv.ViewQuery` / `Condition` / `Operator`** — typed query DSL (`OpEq`, `OpNeq`, `OpGt`, `OpGte`, `OpLt`, `OpLte`, `OpIn`, `OpLike`).
- **Preset integration** — `sqlite.SQLViewModel[V,K]` and `postgres.SQLViewModel[V,K]` one-call constructors.
- **`storage.WithoutViewAutoMigrate`** / **`storage.SQLiteApplyOptimizations`** — production options.
- **`sqlite.WithForeignKeys()` / `sqlite.WithOptimizations()`** — referential integrity + cache/temp/mmap PRAGMAs.

#### Multi-Database Split

- **Postgres multi-DB split** — `WithEventDB`/`WithQueryDB`/`WithViewDB` options for the Postgres preset, mirroring SQLite and Turso. Routes events+snapshots+checkpoints, commands+queries, and read models to separate databases on the same Postgres server. (ADR-0033)
- **`stack/sqlopt` package** — shared option-assembly logic for SQL-backed presets. Keeps the base `stack` package free of a storage dependency.
- **`stack.WithDatabase` / `Bundle.Database()`** — expose the underlying DB handle for preset-specific constructors.
- **Multi-DB contract test** — `contracttest.RunMultiDBSuite` verifies routing correctness.
- **Multi-DB example** (`example/deployer-first-multidb/`) — runnable end-to-end demo.
- **ADR-0033** — Multi-database split design rationale.
- **ADR-0034** — Session store boundary.

#### Shared Metadata & Lifecycle Helpers

- **`event.CustomData[K]`** — shared generic base for `command.Metadata` and `query.Metadata` (ADR-0031). Carries tracing + custom map with shared `Clone`/`Merge`/`EnsureCustom`.
- **`event.MergeCustomMaps`** — generic zero-allocation merge for custom metadata maps.
- **`stack.MultiCloser` / `stack.FuncCloser`** — shared lifecycle helpers.
- **`Bundle.Debug()`** — prints which capability fields are set for wiring diagnostics.

#### CI & Tooling

- **API stability CI check** — `cmd/api-stability` golden file (1627 exports) verified on every push/PR.
- **Convenience flake apps** — `nix run .#test-grpc`, `.#check-wasm`, `.#check-api-stability`, `.#ci` (aggregate).
- **`nix run .#check-file-size`** — local mirror of the CI file-size gate.
- **Property-based tombstone tests** — 6 `rapid`-based tests (100 iterations each) covering empty stream, last-event-wins, no-mutation, transitions, unmarked, nil.
- **Zero lint findings** — golangci-lint config tuned to 0 findings across all 33 modules (down from 200).
- **12 design documents** (`docs/design/`) — NATS, Redis, secondary indexes, hot-state cache, read-pressure snapshots, compaction, archival, dashboard, distributed runner, blocked items, makezero eval, remaining ideas.

#### Storage & Production Tuning

- **`synchronous=NORMAL` in `SQLiteEnableWAL`** — 3-10x better write throughput without durability loss.
- **Turso WAL default** — Turso preset now enables WAL by default; disable with `WithoutWAL()`.
- **Turso sync contract test** — `TestNewSync_Contract` (skips without `TURSO_SYNC_URL`).
- **Schema migration caveat** documented in `storage/doc.go`.
- **Migration guide** (`docs/MIGRATION_TO_STACK.md`) — replacing hand-wired infrastructure with presets.

### Fixed

- **11 phantom doc references** — corrected stale type names across stack/doc.go, stack/errors.go, bundle.go, options.go, snapshot/doc.go.
- **FEATURES.md stale v2 import paths** — stack modules updated to v3.
- **ROADMAP.md module count** — corrected 38 → 43.
- **ADR-0026 stale WASM claims** — decider/ now compiles to WASM (fixed via `//go:build !js`); removed reference to deleted `wasm/main.go`.
- **9 dead `noinlineerr` references** — removed from `.golangci.yml` exclusion lists.
- **11 stale `//nolint:errcheck` directives** — removed from test files (errcheck excluded for `_test.go`).
- **`stack/go.mod` invalid `eventtest v3.0.0`** — fixed to `v0.0.0` (no major-version suffix).
- **storage/pebble test unchecked errors** — added error checks on constructor calls.

### Changed

- **go-error-family upgraded v0.4.0 → v0.5.1** — across all 12 direct-dep modules. `event.Compose` removed (use stdlib `errors.Join`). Upstream adds `Family.HTTPStatus()`, `Family.RetryPolicy()`, `Error.JSON()`, copy-on-write errors, severity-ordered multi-error classification, lock-free sentinel lookup, injectable `Registry`.
- **API surface** — 1558 → 1627 exports. Golden file regenerated.
- **Coverage documented** — real per-module numbers in AGENTS.md (decider 98.3%, event 91.4%, command 89.4%, workspace total 78.7%). — `WithEventDB`/`WithQueryDB`/`WithViewDB` options for the Postgres preset, mirroring SQLite and Turso. Routes events+snapshots+checkpoints, commands+queries, and read models to separate databases on the same Postgres server. (ADR-0033)
- **Multi-DB contract test** — `contracttest.RunMultiDBSuite` verifies routing correctness for any preset supporting multi-DB. Wired into sqlite and turso test suites; postgres test requires `POSTGRES_TEST_DSN` + `CREATE DATABASE` permission.
- **Migration guide** (`docs/MIGRATION_TO_STACK.md`) — Step-by-step guide showing how to replace 200–400 lines of hand-wired infrastructure with 5–10 lines of stack preset. Covers event store, projection runner (CatchUpSubscriber+Materialize), build-tag switching, and multi-DB split.
- **Turso sync contract test** — `TestNewSync_Contract` runs the full contract suite against a NewSync bundle (skips without `TURSO_SYNC_URL`).
- **ADR-0033** — Multi-database split design rationale.
- **ADR-0034** — Session store boundary (sessions are application-layer, not CQRS infrastructure).
- **Schema migration caveat** documented in `storage/doc.go` — raw constructors do NOT auto-migrate; use a stack preset or call `SQLiteInitSchema`/`PostgresInitSchema` manually.
- **`synchronous=NORMAL` in `SQLiteEnableWAL`** — WAL mode now sets `synchronous=NORMAL` instead of the default FULL, giving 3-10x better write throughput without durability loss (safe with WAL). Affects both SQLite and Turso presets.
- **SQLite `WithOptimizations()`** — applies `cache_size`, `temp_store=MEMORY`, and `mmap_size` PRAGMAs for production throughput. Parity with the existing Turso option.
- **Turso `WithoutWAL()`** — WAL mode is now the default for the Turso preset (was previously off). Disable with `WithoutWAL()`.

## [3.0.0] - 2026-06-22

**Major release — tagged.** All 38 modules migrated to `/v4` import paths. The 11 breaking changes are additive in nature (the new shapes existed in v2). See the **[v3 Migration Guide](docs/migration/V3_MIGRATION.md)** for step-by-step instructions.

### Breaking Changes

| #   | Change                                                                                                      | ADR                                                       |
| --- | ----------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| 1   | Delete ghost bus code (`event/reactive*.go`, `samber/ro` dep)                                               | [0028](docs/adr/0028-watermill-as-delivery-layer.md)      |
| 2   | Move `memory/` → `storage/memory/`                                                                          | [0029](docs/adr/0029-storage-consolidation.md)            |
| 3   | `event.Version`: `int` → `uint64`                                                                           | —                                                         |
| 4   | Break `command/query.Metadata = event.Metadata` alias (ADR-0031)                                            | [0031](docs/adr/0031-metadata-split.md)                   |
| 5   | Remove `io.Closer` from 9 core interfaces                                                                   | [0010](docs/adr/0010-remove-io-closer-from-interfaces.md) |
| 6   | Delete `readmodel/` (merged into `kv/` as `kv.TypedStore` + `kv.Cache`)                                     | [0032](docs/adr/0032-merge-readmodel-into-kv.md)          |
| 7   | Delete `projection/` (replaced by `bus.SubscribeAll` + `stack.Materialize` + `watermill.CatchUpSubscriber`) | [0030](docs/adr/0030-dissolve-projection.md)              |
| 8   | Move SSE → `transport/http/`; delete healthcheck/metrics_http/pprof                                         | [0025](docs/adr/0025-transport-adapter-strategy.md)       |
| 9   | `query.Handler`: `any` → generic `TypedHandler[Q, R]`                                                       | [0008](docs/adr/0008-typed-handler-signature.md)          |
| 10  | Rename `Decider.Fold` → `Apply`                                                                             | —                                                         |
| 11  | Make `event.Event` a concrete type (`= *ImmutableEvent`)                                                    | —                                                         |

### Added

- **Pebble backup and observability accessors** (`stack/pebble/`) — `pebble.Bundle` wraps `*stack.Bundle` with `Checkpoint(dir)` for point-in-time backups, `Metrics()` for LSM-tree health, `Flush()` for write durability, and `NewSnapshot()` for consistent reads.
- **Bundle.GracefulClose** (`stack/`) — Context-bounded `Close()` for production shutdown. Runs `Close()` in a goroutine; returns `ctx.Err()` if the deadline fires. Lets in-flight handlers drain without hanging forever.
- **SSE Last-Event-ID reconnection** (`transport/http/`) — `WithReconnectJournal(journal, limit)` option on `NewSSEBroker` enables standard SSE reconnection. When a client sends `Last-Event-ID`, the broker replays missed events from the journal before starting live delivery. Uses the same dedup strategy as `watermill.CatchUpSubscriber` (replayIDs set) to prevent duplicate delivery.
- **Streaming event reads** — `StreamingSource`/`StreamingJournal` now implemented on all three stores: `SQLEventStore` (cursor-based via `*sql.Rows`), Pebble `EventStore` (iterator-based with limit + skip), `MemoryStore` (SliceIterator-wrapped). Consumers can type-assert to streaming interfaces uniformly across backends.
- **DistributedRunner** — _Deleted with `projection/` (ADR-0030). The `watermill.CatchUpSubscriber` + `stack.Materialize` pattern replaces it with simpler semantics._
- **cqrs-gen event handler generation** — _Removed: `-type=event` generated `projection.On[T]()` calls, but `projection/` was deleted (ADR-0030). cqrs-gen now supports `command` and `query` only._
- **Postgres LISTEN/NOTIFY event bus** (`storage/`) — `PostgresBus` implements `event.Bus` using `SELECT pg_notify()` with lightweight JSON reference payloads (under 8KB). `NotificationListener` interface abstracts driver-specific LISTEN; the bus calls `Listen(channel)` itself so consumers don't need to pre-arm. Listener re-fetches full events from store with retry for visibility-gap handling. Uses `LoadByEventID` (indexed O(1) lookup) when the store implements `EventByIDLoader`. **Wired into `stack/postgres` preset** via `WithDistributedBus(listener)` option.
- **PgxListener** (`stack/postgres/`) — `PgxListener` implements `storage.NotificationListener` using `pgxpool`. Dedicated single-connection pool for LISTEN; channel-name allow-list defends against SQL injection. `NewPgxListener(pool)` wraps an existing pool; `NewPgxListenerFromDSN(ctx, dsn)` creates an owned single-conn pool.
- **PostgresBus otel spans** — `pg_bus.publish` (SpanKindInternal) and `pg_bus.handle_notification` (SpanKindConsumer) spans for distributed tracing of NOTIFY round-trips.
- **Real-Postgres integration tests** (`stack/postgres/`) — Three `-tags=integration` tests covering the full LISTEN/NOTIFY round-trip, channel validation, and preset wiring. Run in CI's `postgres-integration` job.
- **Documentation site content** — `docs/index.md` landing page with value proposition, quick start, module overview, presets comparison table.
- **PgxListener auto-reconnect** (`stack/postgres/`) — On connection loss, the listener automatically re-acquires a connection and re-issues LISTEN with exponential backoff (default: 10 attempts, 1s→30s). Configurable via `WithReconnect(maxAttempts)`, `WithReconnectBackoff(initial, max)`, `WithoutReconnect()`. A dropped connection no longer silently kills event delivery.
- **PgxListener deadlock regression test** — `TestPgxListener_CloseDoesNotDeadlock` asserts Close() returns within 2s when the receive loop is running, preventing regression of the critical cancelFn fix.
- **Property-based channel-name validation** — `rapid` property tests (3 properties × 100 inputs) covering valid identifiers, digit-first rejection, and no-panic-on-arbitrary-input.

### Changed

- **Module paths** — All 38 modules migrated from `…/v2` to `…/v4` import paths (e.g. `github.com/larsartmann/go-cqrs-lite/event/v4`). Consumers update `go get` targets and import statements. The `example/*` modules remain unversioned.
- **Zero-panic API migration** — All production `panic()` calls converted to error returns. Breaking signature changes:
  - `pebble.NewStore/NewSnapshotStore/NewCheckpointStore/NewKVStore/NewQueryStore/NewCommandStore` now return `(T, error)` — returns `ErrNilDatabase` (classified as `Rejection`) if db is nil.
  - `pebble.NewBackend` now returns `(*Backend, error)`.
  - `multisig.VerifierMap` now returns `(map, error)` — returns `ErrNilSigner` (`Rejection`) if any signer is nil.
  - `Version.Decrement()` and `Version.Sub(n)` now return `(Version, error)` — returns `ErrVersionUnderflow` (`Rejection`) on underflow.
  - `SchemaVersion.Decrement()`, `.Add(n)`, `.Sub(n)` now return `(SchemaVersion, error)` — returns `ErrSchemaVersionUnderflow` (`Rejection`) on underflow.
  - `codec.CBOREncMode()` and `codec.CBORDecMode()` return bare `cbor.EncMode`/`cbor.DecMode` via `sync.OnceValue` (no error — creation cannot fail with hardcoded valid options).
  - `cattest.StringSchema` now returns `(*Schema, error)` instead of panicking on odd-length props.
- **SSE moved to transport/http/** (`transport/http/`) — SSE broker moved from `middleware/` to new `transport/http/` module (ADR-0025). SSE wire format rewritten with proper `SSEEvent` struct, spec-correct multi-line `data:` handling, `SSEEventID` branded type, and 15s heartbeat to prevent proxy timeouts. Healthcheck, metrics_http, and pprof handlers deleted (generic utilities, zero CQRS deps, zero consumers).
- **Ghost streaming interfaces removed** — Consolidated the old `StreamLoader`/`EventStream` types (bool-based `Next()` + `Err()`) into the shipped `EventIterator` interface (standard Go `io.EOF` pattern). Dead code that never compiled against the real interface is gone.
- **WASM compilation** — All 7 core modules (id, codec, dispatcher, event, command, query, decider) now compile to `GOOS=js GOARCH=wasm`. Moved `NewCQRSViews()` behind `//go:build !js` to exclude the OTel SDK's `os/user` dependency.
- **notifyPayload type model** — Replaced 5 stringly-typed fields with branded domain types (`id.EventID`, `event.Type`, `event.AggregateType`, `id.AggregateID`, `event.Version`). Eliminates the manual `String()`→`Parse` roundtrip on the receive side.
- **pgx upgraded v5.7.1 → v5.10.0** — Patches critical memory-safety vulnerability (CVE) and SQL-injection via placeholder confusion.
- **API surface** — 1806 → 1852 exports.

## [2.7.0] - 2026-06-19

The **Bundle composition layer**: consumers stop deciding on infrastructure. A deployer picks a backend via one preset call; the application imports only `readmodel` and `stack` and never touches a storage driver. 8 new modules (~5,500 lines), persistent read models for every preset, a shared contract suite, and a zero-lint release gate.

### Added

- **Bundle composition root** (`stack/v2`) — `Bundle` with ISP-honest fields (EventSink/EventSource/Journal kept separate, not a fat Store), `Option = func(*Bundle)`, pointer-deduplicated `Close()`, and rollback-on-validation. Repository/ReadModel helpers are top-level generic functions (`stack.Repository[State]`, `stack.ReadModel[T,K]`) since Go forbids generic methods.
- **Bundle presets** — `stack/memory`, `stack/sqlite` (modernc, WAL, auto-migrate), `stack/pebble` (single PebbleDB for all stores via disjoint key prefixes), `stack/postgres` (pgx, auto-migrate). Each wires event store+bus, command/query/snapshot/checkpoint stores, and a read-model backend in one call.
- **Typed read-model store** (`readmodel/v2`) — `Store[T any, K fmt.Stringer]` over `kv.Store` with codec + key prefixing; `Backend` is an alias for `kv.Store`, so `kv.MemStore`, `pebble.KVAdapter`, and the new SQL KV store all satisfy it.
- **Read-model cache decorator** (`readmodel/cache/v2`) — Otter-backed `CachedStore[T,K]` (TinyLFU admission) with capacity + TTL, write-through.
- **Typed stores** — `snapshot.TypedSnapshot[State]` + `TypedStore` (closes the `[]byte` hole on snapshot state); `command.TypedCommandStore[P]` (with `AppendBatch`); `query.TypedQueryStore[P]`. Encode/decode happens once at the adapter boundary.
- **Pebble gaps closed** (`pebble/v2`) — `CommandStore`, `QueryStore`, and `ReadModels()` accessor on `Backend`; EventStore.Close() is now a no-op so the Backend owns the DB lifecycle (fixes a double-close).
- **SQL-backed kv.Store** (`storage/v2`) — `SQLKVStore` implements `kv.Store` over a `cqrs_kv` table (Get/Set/Has/Delete/streaming-Iterator/transactional-Batch), exposed via `SQLBackend.KVStore()`. SQLite and Postgres presets now **persist read models across restarts** instead of using `kv.MemStore`. Verified by an E2E reopen test.
- **Shared contract test suite** (`stack/contracttest`) — `RunSuite(t, factory)` runs 5 behavioural checks; 4 presets × 5 = 20 contract assertions.
- **Zero-overhead benchmarks** (`stack/bench/v2`) — proves Bundle field access is a direct struct read (~0.20 ns/op).
- **godoc example** (`stack/memory`) — `ExampleNew` renders the canonical Bundle entry point on pkg.go.dev.

### Changed

- **Dialect interface** (`storage/v2/sql`) — gained `KVSchema()` for the `cqrs_kv` table (BLOB for SQLite, BYTEA for Postgres). The only implementations are the in-package `PostgresDialect`/`SQLiteDialect`; upsert uses `ON CONFLICT(key) DO UPDATE … excluded.value`, identical across dialects.
- **Lint app resilience** (`flake.nix`) — `nix run .#lint` now reports every failing module instead of aborting on the first (it ran under `errexit`).
- **API surface** — 1351 → 1784 exports; golden file regenerated and the checker's module list expanded to 33 consumer-facing modules.
- **Example rewrite** (`example/todo`) — uses the pebble Bundle preset + `readmodel.Store`; dead `storage/` package deleted (7 files).

### Fixed

- **Postgres preset tests ran in CI** — the `postgres-integration` job set `DATABASE_URL` (read by `storage` tests) but not `POSTGRES_TEST_DSN` (read by `stack/postgres` tests), so preset tests were silently skipped despite a running container. Now sets both and runs the preset suite.
- **Zero lint violations** — 39 violations shipped under `--no--verify` in the first pass are now 0 across all 34 modules (readmodel, pebble, snapshot, stack presets cleaned up).
- **Workspace** — `go work sync` applied; dependency budgets reconciled (`DEP_BUDGET[storage]` 11→12 for the new kv dep).

### Infrastructure

- CI matrix, `flake.nix`, `check-module-layers.sh`, and `.golangci.yml` updated for the 8 new modules (otter + pgx added to depguard).

## [2.6.0] - 2026-06-19

27 commits since v2.5.0. Two new modules (schema validator, prometheus exporter), projection replay/live split, replay→live dedup pipeline, OTel correlation enricher, bounded dedup, streaming event reads, exported ID marker types, cqrs-gen struct tags, and leader election interface.

`pebble.DeleteEventsBefore` (added in v2.5.0, the immediately prior release ~24h earlier) is removed: it contradicted event-sourcing immutability and no consumer could depend on it between releases. No other existing API removed or renamed.

### Added

- **Schema registry validator** (`schema/v2`) — `Validator` with `RegisterType[T]()`, `RegisterTypeWithValidator[T]()`, strict/lenient modes, custom codec support. Returns `Rejection` errors on invalid payloads. ADR-0017 accepted
- **Prometheus metrics exporter** (`prometheus/v2`) — New module wrapping OTel Prometheus exporter. `Setup()` creates a `MeterProvider` backed by a Prometheus registry and an HTTP handler for `/metrics`. `WithRegistry()`, `WithHandlerOptions()`, `MustSetup()`
- **Bounded dedup** (`event/v2`, `projection/v2`) — `DistinctByEventIDBounded(cap)` with FIFO ring eviction for bounded memory in 24/7 projections. `DistinctByEventIDBoundedWith(cap, seen)` seeded variant. `WithDedupCapacity(n)` Runner option
- **Streaming event reads** (`event/v2`) — `EventIterator` interface for one-at-a-time event reading without materializing slices. `StreamingSource` and `StreamingJournal` opt-in interfaces. `SliceIterator` adapts pre-loaded slices
- **cqrs-gen struct tag scanning** (`cmd/cqrs-gen`) — Supports `cqrs:"command:CreateUser"` struct tags on `_ struct{}` fields in addition to `//cqrs:command CreateUser` comment markers. Comment markers take precedence
- **LeaderElection interface** (`projection/v2`) — `LeaderElection` interface + `AlwaysLeader` default for distributed projection coordination per ADR-0018. Consumers implement coordination (Redis, etcd, k8s); library provides interface and default
- **Projection replay/live split** (`projection/v2`) — `Runner.RunReplay(ctx)` replays historical events synchronously and returns once the read model is caught up (read-your-writes); `Runner.RunLive(ctx)` then tails live events in the background. `Run` remains as a convenience wrapper calling both. Eliminates `time.Sleep`-based catch-up hacks in consumers. Adds `ErrReplayRequired` when `RunLive` is called before `RunReplay`
- **Replay→live dedup pipeline** (`event/v2`, `projection/v2`) — Closes the duplicate-processing gap at the replay→live boundary. New `event.SubscriberToObservable` adapts callback-based `Subscriber` to `ro.Observable[Event]`; `event.DistinctByEventIDWith(seen)` seeds the dedup set with IDs from journal replay. The Runner's live path now builds `live → DistinctByEventIDWith(replayIDs) → handler`, suppressing overlap-window duplicates
- **OTel correlation enricher** (`middleware/v2`) — `OTelCorrelationEnricher` bridges OTel baggage correlation IDs into event metadata via `event.WithCustom`. Composes with `CommandCausalityEnricher` via `CompositeEnricher`. New `OTelCorrelationIDFromEvent` extractor and `MetadataKeyOTelCorrelationID` constant
- **Exported ID marker types** (`id/v2`) — All 8 phantom marker types are now exported (`AggregateMarker`, `UserMarker`, `CorrelationMarker`, `RequestMarker`, `CausationMarker`, `ClientMarker`, `CommandMarker`, `EventMarker`), enabling downstream `go-branded-id` `BrandNamer` integration and other type-parameterized tooling against the root module's ID types

### Removed

- **Pebble `DeleteEventsBefore`** (`pebble/v2`) — Removed. Events are immutable truth; automatic event deletion contradicts event sourcing principles. Introduced in v2.5.0 (immediately prior release) and removed before any consumer could adopt it. The `Flush()` method remains for durability control

## [2.5.0] - 2026-06-18

70 commits since v2.4.0. Pebble backup/retention/consistent reads, OpenTelemetry baggage correlation + metric views + propagator, load coalescing via singleflight, HKDF multi-tenant key derivation, CBOR streaming, reactive event dedup operators, Watermill middleware wrappers, and turso race fixes. No breaking API changes.

### Added

- **Pebble backup and consistent reads** (`pebble/`) — `PebbleBackend.Checkpoint(dir)` for point-in-time DB snapshots and `NewSnapshot()` for consistent read views via Pebble snapshots
- **OTel baggage correlation IDs** (`otel/`) — `WithCorrelationID(ctx, id)` and `CorrelationIDFromContext(ctx)` propagate correlation IDs across distributed service boundaries via W3C baggage
- **OTel TextMapPropagator** (`otel/`) — `NewTextMapPropagator()` implements W3C trace context + baggage propagation for inject/extract across transports
- **OTel CQRS metric views** (`otel/`) — `NewCQRSViews()` configures customized histogram boundaries (`CQRSHistogramBoundaries`) for CQRS latency ranges; `ServiceResourceAttributes()` for service identification; `CounterAddWithAttributes()` and `AddSpanEvent()` helpers for rate metrics and span events
- **Decider load coalescing via singleflight** (`decider/`) — `Repository[State]` now coalesces concurrent `Load` calls for the same aggregate into one `store.Load` query. Events are immutable (`*ImmutableEvent`), so sharing the loaded slice is safe. Disable via `WithLoadCoalescing[State](false)`
- **HKDF key derivation** (`encryption/`) — `DeriveKey(masterKey, info, length)` derives per-tenant/subscope keys via HKDF-SHA256, enabling multi-tenant encryption without separate master keys
- **SQLite foreign keys helper** (`storage/`) — `SQLiteEnableForeignKeys(ctx, db)` enables `PRAGMA foreign_keys=ON` for opt-in referential integrity
- **Codec BufferEncoder interface** (`codec/`) — `BufferEncoder` extension enables zero-allocation encoding directly into a caller-provided `*bytes.Buffer` via `EncodeToBuffer(payload, buf)`, bypassing intermediate allocations
- **Event stream deduplication operators** (`event/`) — `DistinctByEventID()` suppresses duplicate event IDs; `DistinctByAggregateID()` keeps only the first event per aggregate. Composable via `ro.Pipe1`
- **Watermill middleware wrappers** (`watermill/`) — `CorrelationIDMiddleware()` and `NewRetryMiddleware(config)` for Watermill routers, plus Router integration support
- **CBOR streaming and compact codec docs** (`codec/`) — `CBORCompactCodec` documentation (struct fields as positional array, ~35% smaller payloads); `Diagnose()` for human-readable CBOR debugging
- **Testutil seed control** (`testutil/`) — seed control helper and rapid testing generator patterns for reproducible randomized tests

### Changed

- **Dependency upgrades** — `go-error-family` v0.3.0 → v0.4.0; `go-branded-id` v0.3.0 → v0.3.1 across all consuming modules
- **API surface growth** — 1266 → 1289 exports (29 new public symbols), golden file updated
- **Testutil ghost API removal** (`testutil/`) — removed non-functional `EventSlice` and `SeedFromEnv` exports (dead code that never worked; technically a public surface reduction but no behavioral impact)

### Fixed

- **Turso CheckpointScheduler race** (`turso/indexing/`) — `Stop()` now drains the checkpoint goroutine via a `done` channel before returning, preventing goroutine leaks and races on repeated Start/Stop cycles
- **Turso parallel test flakiness** (`turso/`) — eliminated flaky parallel test failures by isolating state and increasing checkpoint test timing margins
- **Decider singleflight error passthrough** (`decider/`) — singleflight errors now pass through verbatim instead of being wrapped with `fmt.Errorf`, preserving error classification (Rejection/Conflict/etc.) via `errors.Is`
- **OTel NewCQRSViews wildcard** (`otel/`) — corrected view instrument name wildcard matching so all CQRS histograms receive custom boundaries
- **Production dependency budget accuracy** (`scripts/check-module-layers.sh`) — test-only packages (gomega, ginkgo, rapid) now excluded from the production dep count, reflecting true direct dependency budgets

### Infrastructure

- **Watermill Router integration test** — end-to-end test for CorrelationID + Retry middleware through a real Watermill Router

## [2.4.0] - 2026-06-17

15 performance optimizations across 7 modules. No public API changes, no disk format changes, no breaking behavior. Verified with 5-run benchmark averages (allocation deltas are deterministic and reliable; ns/op has ±15% variance), tests + race detector + lint.

### Performance

- **Pebble double serialization eliminated** (`pebble/`) — events serialized once, `batch.Set` called for both event and journal keys. Halves CPU and disk bytes per write
- **Event lazy metadata map initialization** (`event/`) — `NewMetadata()` returns zero-value struct instead of always allocating a map. Eliminates 1 heap allocation per event when no custom metadata is set
- **Projection handler Lookup zero-allocation** (`projection/`) — `lookupSlices()` returns pre-built handler slices directly instead of allocating a combined slice per event. Only benefits `projection.Builder`-created projections
- **Projection Runner event type caching** (`projection/`) — Runner caches `p.EventTypes()` once at `Register()` time, eliminating 10.5M per-event clone allocations (100K events × 100 projections) in the scale benchmark. This is the real fix for the projection allocation hotspot — the original T3/T4 `*builtProjection` type assertion was dead code for `event.NewProjection()` users. Also pre-allocates the candidates slice in `dispatchToProjections`
- **SQL template strings cached per dialect** (`storage/`) — INSERT SQL built once at `SQLEventStore` construction, eliminating `fmt.Sprintf` per call
- **MemoryStore Load double-copy eliminated** (`memory/`) — removed redundant `slices.Clone` wrapper on already-fresh slice from `getEvents()`
- **SSE vestigial goroutine removed** (`middleware/`) — removed useless `go func() { <-ctx.Done() }()` goroutine leak. Consolidated 3× `fmt.Fprintf` into single write
- **Event Merge EnsureCustom hoisted** (`event/`) — `EnsureCustom` called once before the Merge loop instead of per-iteration nil-check
- **Event FilterByTimestamp pre-sized** (`event/`) — result slice initialized with `make([]Event, 0, len(events))` to eliminate nil-slice append growth pattern
- **SQL ScanSlice pre-allocated** (`storage/`) — initial capacity hint of 64 reduces log₂(N) slice growth copies during large Loads
- **CircuitBreaker atomic state machine** (`middleware/`) — replaced `sync.Mutex` + `int` fields with `atomic.Int32`. Happy path (circuit closed) is now lock-free: single `state.Load()` check
- **MemoryBus middleware pre-computation** (`memory/`) — middleware chains pre-computed at `Use()`/`UsePublish()` registration time. `Publish()` reads cached chain under RLock — zero per-publish closure allocation
- **Pebble ReadFrom key-based skip** (`pebble/`) — during cursor skip phase, parse event ID from journal key via `journalKeyEventID()` instead of CBOR-deserializing every skipped event
- **SQL multi-VALUES INSERT batching** (`storage/`) — single `INSERT INTO events ... VALUES (..), (..), (..)` statement replaces N individual INSERTs. SQLite 999-parameter limit handled via automatic chunking (99 events/batch)

### Added

- **Reactive CommandBus and QueryBus** (`command/`, `query/`) — `NewCommandBus`, `NewQueryBus`, `FilterCommandType`, `FilterQueryType`, `HandlerToObserver`, plus replay/behavior variants. Mirrors the existing reactive event API for command and query streams
- **PebbleBackend facade** (`pebble/`) — `Open()` and `NewBackend()` provide a single shared-DB entry point for Pebble-backed EventStore, SnapshotStore, and CheckpointStore, with clear ownership semantics
- **SQLBackend lifecycle facade** (`storage/`) — `SnapshotStore()`, `CheckpointStore()`, and `Close()` methods complete the SQL backend full-stack facade
- **KV module** (`kv/`) — Layer-0 in-memory key-value store abstraction (`MemStore`) with snapshot iteration and atomic batch commit
- **`command.Compose` and `query.Compose`** — re-export `go-error-family.Compose` for classified multi-error composition in command and query modules
- **Integration tests** (`integration/`) — end-to-end tests for pebble-backed projection Runner (replay + live) and decider Repository with Pebble SnapshotStore
- **Pebble KV Store adapter** (`pebble/`) — `NewKVStore()` wraps `*pebble.DB` as `kv.Store`, making pebble the first real consumer of the kv/ abstraction. Supports owned and borrowed DB lifecycle, prefix-bounded iteration, atomic batch commit, and `ErrNotFound`/`ErrClosed` error mapping
- **Built-in pprof endpoints** (`middleware/`) — `ProfilingHandler()` and `RegisterProfiling()` expose Go runtime profiling (heap, goroutine, CPU, allocs, block, mutex) via standard `/debug/pprof/` paths
- **Pebble benchmarks** (`pebble/`) — 4 benchmarks (Save100, SaveLoad100, Save1, LoadEmpty) for performance regression tracking
- **KV contract tests** (`pebble/`) — 10-test contract suite run against both PebbleAdapter and MemStore, proving semantic equivalence
- **Compose tests** (`command/`, `query/`) — 5 tests each for `Compose` error composition (nil, single, multiple, classified, mixed)
- **PostgreSQL CI** (`.github/workflows/ci.yml`) — `postgres-integration` job with PostgreSQL 16 service container wired to storage integration tests

### Fixed

- **Turso error classification** (`storage/sql/query_engine.go`) — `QueryRows` no longer re-wraps classified errors as Infrastructure, preserving Rejection semantics for `LoadNonExistent`
- **Module layer budgets** (`scripts/check-module-layers.sh`) — budgets updated to reflect actual direct dependencies: codec 2, pebble 8, storage 11, turso 10, integration 19
- **Turso lint hygiene** (`turso/indexing/advisor_data.go`) — cleared 3 pre-existing `gochecknoglobals` findings on static advisor data tables

### Infrastructure

- **CI replace-directives check** — `scripts/check-replace-directives.sh` now runs in GitHub Actions to verify every module `replace` directive matches `go.work`
- **`cmd/api-stability` in CI matrix** — per-module-test job now tests the API stability checker in isolation

## [2.3.0] - 2026-06-12

231 commits since v2.2.0. Lint hygiene, coverage improvements, CBOR codec, encryption module, phantom types, and release readiness.

### Added

- **CBOR codec** (`codec/`) — `CBORCodec` with deterministic canonical encoding, sorted map keys, `DecMode` option
- **Pebble CBOR envelope** (`pebble/serialization.go`) — events serialized as CBOR with JSON backward compatibility layer
- **Encryption module** (`encryption/`) — XChaCha20-Poly1305, AES-256-GCM, `Algorithm` enum, `KeyID` phantom type, `KeyResolver` interface, composable `NewCodec` wrapper, `EncryptMiddleware`/`DecryptMiddleware`
- **Command store interfaces** (`command/`) — `CommandSink`, `CommandSource`, `Store` (Sink+Source) for persisted command logs
- **SQL CommandStore** (`storage/`) — `SQLCommandStore` with Save, AppendBatch, Load, LoadFromTimestamp, LoadToTimestamp
- **SQL Backend facade** (`storage/`) — `SQLBackend` returning EventStore, SnapshotStore, CheckpointStore, CommandStore
- **Phantom types** across library modules — `DbPath`, `RemoteURL`, `AuthToken` (turso); `KeyID` (encryption); `Algorithm` (encryption); `DisplayID` (catalog); type-safe domain IDs in examples
- **Event binary blob helpers** (`event/`) — `AttachBlob`, `ExtractBlob`, `HasBlob` for signing/encryption
- **`command.TypedHandler[Q, R]`** with `RegisterTyped[Q, R]` — type-safe command handler
- **`event.DecodePayloads[T]()`** — batch payload deserialization
- **Listing table schema** (`storage/`) — DDL + repository for aggregate status persistence
- **ADR-0008 through ADR-0015** — 8 new architecture decision records (TypedHandler, immutability, OTel re-exports, error taxonomy, CBOR, encryption, saga, config)
- **ADR index** (`docs/adr/README.md`) — complete index of all 15 ADRs with titles, dates, status
- **Comprehensive fuzz testing** — fuzz tests in codec, encryption, signing/multisig, integration
- **Property-based tests** — `pgregory.net/rapid` in command, query, event, decider, id modules
- **go-snaps snapshot tests** — catalog, integration, projection golden test coverage
- **Benchmark infrastructure** — realistic scale benchmarks, fuzz benchmarks, multisig concurrent benchmarks
- **gosec security scanning** in CI with SARIF upload
- **Module layer check** — `.go-arch-lint.yml` architecture rules enforced in CI
- **17 scale benchmarks** across modules (10K–1M events)
- **`pkg/config/`** — YAML config loader with env-specific overlays
- **`pkg/gracefulshutdown/`** — signal-aware shutdown with timeout and hook support
- **Docker packaging** for `example/user/` (multi-stage Dockerfile + docker-compose.yml)
- **SSE broker** (`middleware/sse.go`) — server-sent events over event bus
- **Health check middleware** (`middleware/healthcheck.go`) — `/health`, `/health/live`, `/health/ready`
- **Metrics HTTP handler** (`middleware/metrics_http.go`) — request count, error rate, avg response time
- **EventCatalog docserver** (`catalog/docserver/`) — embedded SPA with AsyncAPI + Scalar rendering
- **`integration/simulation/`** — event sequence generator + decider stress tests
- **Encryption integration** — end-to-end encrypt→sign→verify→decrypt round-trip tests
- **Test coverage:** storage/sql 37.4%→89.2%, otel 73.0%→97.3%, turso 26.8%→39.0%

### Changed

- **Pebble: migrated event envelope from JSON to CBOR encoding** — deterministic, compact binary format
- **Pebble: sharded mutex pool** (FNV-1a hash, 256 shards) replaces unbounded `sync.Map` — bounded memory, zero allocations
- **storage/sql: extracted generic `LoadWithSpan[T]` + `QueryRows[T]`** — eliminated event/command store load duplication
- **storage/sql: context-aware SQL methods** throughout — `BeginTx`, `ExecContext`, `QueryRowContext` (no more `noctx` lint)
- **storage/sql: `ClosableBase` extracted** — deduplicated store lifecycle boilerplate
- **OTel abstraction** — modules import `otel/` re-exports instead of `go.opentelemetry.io` directly (decider, storage, middleware, projection)
- **Error wrapping** — replaced `fmt.Errorf` wrapping classified errors with `WrapRejection`/`WrapCorruption` across memory, pebble, storage, listing
- **`command/command.go`** — added `Type.IsZero()`, `ParseType()`, `MustParseType()` to match `event.Type` API
- **`query/query.go`** — added `Type.IsZero()`, `ParseType()`, `MustParseType()` to match `event.Type` API
- **`event/types.go`** — `SchemaVersion.Cmp` now uses `cmp.Compare` (matches `Version.Cmp`)
- **`event/errors.go`** — doc comments on all 30 exported error symbols
- **`event/Clone()`** — deep-copies `eventOptions` pointer to prevent shared mutation
- **`event: Map/ScanState/Tap` reactive wrappers removed** (unused, no consumers)
- **`event: StreamKey` free function removed** (unused)
- **All 120 `//nolint` suppressions** now have documented `// reason` justifications
- **0 lint issues** across all 27 modules — first zero-lint release
- **`golang.org/x/exp`** bumped across all workspace modules
- **`storage/AggregateProjection`** uses `Dialect.Placeholder()` (Postgres-compatible)
- **`listing/AggregateRef` renamed to `AggregateListing`** with JSON tags
- **`catalog: ErrorExporter` deprecated** as type alias to `Exporter[error]`
- **`catalog: asyncapi.Info` and `openapi.Info` consolidated** into shared `DocumentInfo`
- **`snapshot: json tags`** added to `Snapshot` struct
- **Dissolved `core/` module** — all sub-packages are flat peer-level modules (v2.0.0, maintained in v2.3.0)
- **`event.Snapshot*` types moved to `snapshot/` package** — all consumers updated
- **`dispatcher/Lifecycle` field unexported** with method delegation added

### Fixed

- **SSE broker send-on-closed-channel race** — `handleEvent`/`RemoveClient` synchronization
- **SSE broker constructor** — `NewSSEBroker` now returns `(*SSEBroker, error)` instead of nil on error
- **Circuit breaker nil `IsFailure` guard** — defaults to `event.IsRetryable`
- **Circuit breaker error taxonomy** — `ErrCircuitBreakerOpen` uses error taxonomy instead of bare `errors.New`
- **Projection Runner double-wrapping classified errors** in `opError`
- **Projection Runner fresh done channel** per `Run` invocation
- **Projection Runner `Close()`** now waits for `Run` to complete
- **Clone shared opts pointer** — deep-copy `eventOptions` prevents shared mutation
- **Retry middleware** — `ErrRetryCanceled` sentinel actually used on context cancellation
- **Pebble `NewStore(nil, ...)` panics** with clear message instead of nil pointer dereference
- **Pebble `countEvents` uses `iter.Last()`** instead of full scan
- **Pebble `MarshalMetadataJSON` error** — handled instead of discarded
- **Decider `slog.WarnContext` fallback** for snapshot failures (previously OTel-only)
- **Multiple lint issues** — nlreturn, varnameld, noctx, errcheck, unconvert, nolintlint
- **`event.NewMetadata`** now initializes `Custom` map
- **`dispatcher/Lifecycle`** field unexported, added method delegation
- **`event: renamed `WithNewCodec`→`WithCodec`** (kept deprecated alias)
- **Config loader path traversal** — `filepath.Clean` sanitizes paths (gosec G304)
- **Graceful shutdown select guards** on errCh sends to prevent panic

### Performance

- **`catalog.SchemaFromType` cached by `reflect.Type`** — 553ns→8ns, 15→0 allocs
- **`event.New()` lazy-initializes metadata map** — 3→2 allocs per event
- **`event.New()` moves clock/newCodec/deadline to `eventOptions` pointer** — 48B saved per event
- **`event.PayloadReadOnly()` zero-copy** for internal paths (signing, pebble, storage, middleware)
- **`event.DecodePayload` bypasses `Payload()` clone** for zero-copy decoding
- **`listing` caches sorted aggregate index** — 25× faster listing
- **`memory` replaces O(n log n) `collectAllSorted`** with append-only global log
- **`signing.canonicalPayload()` eliminates alloc overhead**

### Security

- **gosec scanning** in CI with SARIF upload
- **Module layer check** enforced in CI
- **Config loader path traversal fix** (G304)
- **Constant-time ciphertext comparison** in encryption module

### Removed

- **`storage.PostgresBus` (LISTEN/NOTIFY)** — the Postgres LISTEN/NOTIFY bus
  implementation and all associated types (`PostgresListenNotifyBus`,
  `NewPostgresBus`, `PostgresBusOption`, `NotificationListener`, `PgxListener`)
  were removed. The `stack/postgres` preset now uses an in-process bus
  (watermill GoChannel). For cross-process pub/sub, wire a Watermill-backed
  bus externally. The NixOS VM test still verifies LISTEN/NOTIFY as a Postgres
  capability (foundation for future distributed-bus work).
- **`storage/options.go`** — deleted `NewSQLEventStoreWithOptions`, `WithOwnership`, `SQLEventStoreOption` (zero external consumers)
- **`storage/doc.go`** — removed 5 unused re-exports
- **`pebble/config.go`** — deleted entire config abstraction layer (`Backend`, `Config`, `NewConfig`, etc.)
- **`pebble/example_test.go`** — tested only deleted config API
- **`pebble/errors.go`** — removed `ErrPebbleProviderRequired`
- **`turso/errors.go`** — removed `ErrTursoMemorySync` backward-compat alias
- **All `MustParse`/`MustParseType` panic wrappers** removed from command, query, event test code
- **Deprecated backward-compat aliases** from `pebble/` module
- **Dead code and unused APIs** across multiple modules
- **`command/errors.go`** — removed unused `WrapTransient` re-export
- **`event/go.mod`** — removed `query/v2` direct dependency
- **`snapshot/go.mod`** — removed `memory/v2` dependency

## [2.2.0] - 2026-06-08

81 commits since v2.1.0. Operational readiness, testing rigor, and developer experience release.

### Added

- **Health check middleware** (`middleware/`) — `/health`, `/health/live`, `/health/ready` endpoints
- **Metrics HTTP handler** (`middleware/`) — request count, error rate, avg response time
- **SSE broker** (`middleware/`) — server-sent events over event bus with subscription management
- **Config loader** (`pkg/config/`) — YAML config with env-specific overlays
- **Graceful shutdown** (`pkg/gracefulshutdown/`) — signal-aware shutdown with timeout and hook support
- **Docker packaging** (`example/user/`) — multi-stage Dockerfile + docker-compose.yml
- **Production server example** (`example/user/server.go`) — operational endpoints demonstrating health, metrics, graceful shutdown
- **Property-based tests** (`decider/`, `event/`, `id/`) — `pgregory.net/rapid` for deterministic decide, version monotonicity, ULID validity
- **Snapshot tests** (`integration/`) — `go-snaps` for event JSON serialization, catalog exports
- **Simulation framework** (`integration/simulation/`) — event sequence generator + decider stress tests
- **Benchmark baseline** (`benchmark-baseline.txt`) — saved from all benchmarks for regression detection
- **Module READMEs** — 9 modules with usage and API surface documentation
- **Package doc.go** — 7 library modules with usage examples for pkg.go.dev
- **example_test.go** coverage — storage, otel, projection, watermill, schema, signing, snapshot, listing, pebble, turso, codec, dispatcher
- **docserver** (`catalog/docserver/`) — embedded EventCatalog SPA server with AsyncAPI + Scalar rendering

### Changed

- **Standardized flake configuration** — dev shell, test apps, benchmark apps unified
- **Command store split** — `storage/command_store.go` (387L → 3 focused files)
- **Snapshot errors extracted** — `snapshot/errors.go` with all sentinel errors
- **Projection replay refactored** — `loadReplayEvents` extracted (65L → 37L + 28L)
- **Dependencies bumped** — `golang.org/x/exp` across all workspace modules
- **Lint issues resolved** — all catalog, infrastructure, and pre-commit hook failures fixed

### Fixed

- **Catalog ToPascal byte underflow** — unicode boundary bug in case conversion
- **Duplicate package godoc** — removed from non-doc.go files in event, middleware, dispatcher
- **Broken example_test.go** — repaired in projection, schema, signing, watermill

### Security

- **gosec scanning** — Go security scanner integrated in CI with SARIF upload
- **Module layer check** — `.go-arch-lint.yml` architecture rules enforced in CI

## [2.1.0] - 2026-06-03

62 commits since v2.0.0. Performance-focused release with production bug fixes, new query types, and comprehensive benchmarking.

### Added

- `query.TypedHandler[Q Query, R any]` — typed query parameter + typed result via `RegisterTyped[Q, R]`
- `listing.CacheInvalidationMiddleware(reader)` — auto-invalidates `InMemoryAggregateReader` cache after publish
- `listing.CacheInvalidator` interface — decouples middleware from concrete reader type
- 17 scale benchmarks across event, memory, listing, storage, pebble, turso, watermill, and codec modules
- 6 new benchmark suites with `b.ReportAllocs` for allocation tracking
- `nix run .#bench` app and `benchstat-compare` script for regression detection
- Turso CRUD integration tests for event/snapshot/checkpoint stores
- Realistic scale benchmarks behind `-tags=scale` in integration module
- ADR-0008 for `TypedHandler[Q Query, R any]` dual type parameter signature
- `docs/STORAGE_GUIDE.md` — performance comparison across PostgreSQL/SQLite/Pebble/Turso backends

### Changed

- `MemoryStore` deduplicated event storage — single `globalLog` + `streamIndex` map of indices replaces per-stream event copies (2× memory reduction)
- `event.New()` inlined codec extraction — removed `findCodecOption` helper, fast path for empty opts avoids probe allocation
- `MemoryStore.ReadFrom` uses cursor-based pagination instead of linear scan
- `schema.VersionedStore` load methods deduplicated into shared `loadAndUpcast` helper
- Error wrapping migrated to `event.Wrap*` taxonomy across storage, watermill, command, query, schema, and listing
- Deprecated backward-compat aliases removed from `pebble/` module
- Dead code removed + Go idioms modernized across multiple modules
- `event.Metadata()` documented as returning a defensive copy

### Performance

- `catalog.SchemaFromType` cached by `reflect.Type` — 553ns→8ns, 15→0 allocs
- `event.New()` lazy-initializes metadata map — 3→2 allocs per event
- `event.New()` moves clock/newCodec/deadline to `eventOptions` pointer — 48B saved per event
- `event.Payload()` removes defensive clone — 1 fewer alloc per access
- `event.New()` skips redundant payload copy — 1 fewer alloc
- `event.New()` stamps encoding directly — 1 fewer alloc
- `signing.canonicalPayload()` eliminates alloc overhead
- `listing` caches sorted aggregate index — 25× faster listing
- `memory` replaces O(n log n) `collectAllSorted` with append-only global log

### Fixed

- HealthCheck OOM on large event stores
- `SQLAggregateReader` Postgres compatibility
- `SubscriberAdapter` race condition
- Pebble `Close` not releasing resources
- `Version.Sub` panic on zero value
- `codec.Raw` passthrough encoding
- `GetID` rename consistency
- `ToAny` error propagation
- `HasSignature` false negatives
- `errgroup` error propagation
- `projection.Runner` missing `ErrAlreadyRunning` guard
- `storage` closed state tracking, snapshot SQL filter, `createTable` context
- `subscribeLive` handler guard for nil handlers
- `eventtest.FakeStore` ReadFrom test for sorted ReadAll output

### Removed

- Deprecated backward-compat aliases from `pebble/` module
- Dead code and unused APIs across multiple modules

## [2.0.0] - 2026-06-01

### Added

- `schema/` module — Upcaster, UpcasterRegistry, VersionedSource for schema evolution (extracted from event/)
- `snapshot/` module — Snapshot, SnapshotStore, SnapshotStrategy, helpers, error sentinels (extracted from event/)
- `samber/ro` integration in `event/reactive.go` — EventBus, NewReplayEventBus, NewBehaviorEventBus, FilterEventType/Types, ReplayFilter, HandlerToObserver/WithContext, Map, ScanState, Tap, Observable type alias
- `samber/ro` integration in `command/reactive.go` — CommandBus, FilterCommandType, Observable type alias
- `samber/ro` integration in `query/reactive.go` — QueryBus, FilterQueryType, Observable type alias
- `event/reactive.go` uses context-aware `ro.NewObserverWithContext` API — handler errors terminate the observer via `ErrorWithContext`
- `projection/runner.go` replay uses direct loop filters (`filterByEventTypes`, `filterFromCheckpoint`) instead of ro.Pipe1/ro.Collect overhead — projection no longer depends on `samber/ro`
- `listing/` module added to flake.nix testModules
- `otel/`, `pebble/`, `turso/`, `codec/` modules added to flake.nix testModules

### Changed

- **Dissolved `core/` module** — All 8 sub-packages (event, command, query, decider, id, dispatcher, schema, snapshot) are now flat peer-level modules. Import paths changed from `go-cqrs-lite/core/{pkg}` to `go-cqrs-lite/{pkg}`.
- `event.Snapshot*` types moved to `snapshot/` package — all consumers updated (decider, memory, storage, testhelpers)
- `event.ErrSnapshotNotFound` / `event.ErrSnapshotStoreClosed` moved to `snapshot/store.go`
- `memory/snapshot.go` uses `snappkg` alias to avoid local variable shadowing
- Removed duplicate `EventHandler` type from `event/reactive.go` (identical to `Handler`)
- AGENTS.md fully rewritten with new monorepo structure, dependency graph, key patterns
- Removed self-referencing replace directives (`module => ./`) from 6 go.mod files

### Removed

- `command/reactive.go` — temporarily deleted (restored in this release)
- `event/reactive.go` — restored with context-aware ro API (NewObserverWithContext + ErrorWithContext)
- `core/` directory — all sub-packages promoted to workspace root
- `event.Context() context.Context` — Go anti-pattern removed; use `Event.Deadline()` instead
- `event/context.go` — `deadlineCtx` type deleted (only used by removed `Context()`)

### Fixed

- `flake.nix` now includes all library modules in testModules
- `go.work.sum` stale references cleaned via `go work sync`

### Added

- `event.DecodePayloads[T]()` batch decode helper for processing multiple events at once
- `middleware.WithLogger(*slog.Logger)` option for retry, recovery, and validation middleware
- `storage/tables.go` — 5 table name constants replacing inline SQL strings
- `dispatcher.LifecycleMixin` embedded in `memory/checkpoint` and `memory/outbox`
- Concurrent access tests for MemoryBus, MemoryStore, MemoryOutbox, MemoryCheckpoint, MemorySnapshot
- `CONTEXT.md` — Domain glossary (aggregate, decider, event, fold, projection, saga)
- `docs/adr/` — ADR-0001 (Decider), ADR-0002 (Error taxonomy), ADR-0003 (Multi-module monorepo)
- `docs/ARCHITECTURE_PATTERNS.md` — Time-travel API, state-is-disposable, determinism, versioned events
- `docs/STORAGE_GUIDE.md` — PostgreSQL/SQLite/Pebble/Turso backends, event store operations

### Changed

- `AGENTS.md` trimmed from 384→121 lines (all essential info preserved)
- TODO_LIST.md reconciled: 40+ stale items verified as already done

### Fixed

- `storage/sql_base.go` bare `%w` wrapping → direct sentinel error return
- LSP hints: `sync.WaitGroup.Go` simplification, `fmt.Appendf` replacing `[]byte(fmt.Sprintf(...))`
- `projection/filterEvents` optimized from O(n×k) to O(n+k) via typeSet map

## [1.0.0] - 2026-05-26

### Added

- **saga** — Saga / Process Manager with compensation, retry, and timeout support
- **watermill** — Watermill message bus adapter with metadata-based event serialization
- **stream loading** — Memory-efficient `EventStream` + `StreamLoader` iterator pattern
- **event versioning** — `VersionedStore` with registered `Upcaster`s for transparent legacy event upcasting
- Full CQRS pipeline integration test (Command → Decider → Store → Bus → Projection → Query → Stream)
- Watermill metadata protocol: 15 metadata keys preserving all event fields

### Changed

- Eventcatalog coverage: 85.7% → 92.8%
- Saga coverage: 70.5% → 93.8%
- Watermill coverage: 28.6% → 89.6%
- `go.work` expanded to 13 modules

### Fixed

- Watermill `toEvent` used broken `json.Unmarshal` into `ImmutableEvent` — replaced with metadata reconstruction

## [0.2.0] - 2026-04-05

### Added

- **Event catalog system** (`catalog/`): Three-layer architecture with reflection-based schema generation, custom YAML marshaler, AsyncAPI and EventCatalog exporters
- **SnapshotStrategy** (`core/event`): Canonical interface and `EveryNEvents(n)` extracted to `core/event/snapshot_strategy.go`
- **Publisher/Subscriber ISP** (`core/event`): Sub-interfaces extracted from `event.Bus` for Interface Segregation
- **Error classification** via `event.RegisterClassification()` in `init()` for aggregate, projection, storage sentinels
- **PublishChanges / SaveSnapshot** (`core/event`): Shared functions eliminating duplication in aggregate/decider repositories
- **Strong ID migration**: 62 bare `string`/`int` violations replaced with named types (`OperationID`, `NodeID`, `ServiceID`, `DomainID`, etc.)
- **Dialect tests** (`storage`): 15 tests for PostgresDialect, SQLiteDialect, `placeholders()`
- **OpenAPI coverage tests** (`catalog/openapi`)
- **Performance benchmarks**: 43 benchmarks across 12 files
- **Design documents**: Outbox transaction API, query handler generics, saga design

### Changed

- **ISP activation**: Repositories accept `Publisher`, projections accept `Subscriber` (backward-compatible)
- Root go.mod module path: `github.com/LarsArtmann/go-cqrs-lite` (consistent casing)
- Zero lint issues across all 8 linted modules (was 50+)
- File splits: all files under 250 lines
- `outboxEvent` fields: `Version`/`SchemaVersion` changed from bare `int` to strong types
- `gomodguard` → `gomodguard_v2`

### Fixed

- All linter issues resolved: exhaustruct, gosec G201, tagliatelle, wrapcheck, noinlineerr, prealloc, goconst, fatcontext
- `FakeSnapshotStore.Save` now records snapshots for verification (was no-op)
- Dispatcher lifecycle: `Register()` and `Dispatch()` on closed dispatcher return errors correctly

## [0.1.0] - 2026-01-01

### Added

- Initial release with core CQRS infrastructure (command, event, query dispatchers)
- Event sourcing with `Store`, `Bus`, `SnapshotStore` interfaces
- In-memory implementations (`memory/` module)
- Branded IDs via `go-branded-id`
- Middleware: logging, retry, recovery, validation
- Test helpers for fakes and mocks
