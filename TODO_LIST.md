# TODO List

**Updated:** 2026-08-08 (metaengine test coverage gaps closed, DuckDB/PG record-stamp + Pebble/DuckDB soak + concurrent -race verified)
**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Metaengine v2 (Record-aware ES-native architecture)

> Metaengine v2 is **feature-complete and publishable**: `record/` module, 9
> engines (Memory, SQLite, Pebble, DuckDB, Postgres, Badger, Dgraph,
> GraphAdapter, Iroh), `OnRecord`/`ApplyRecord` Record-aware folds,
> `AutoInsert`/`AutoCRUD`/`AutoCRUDByConvention` auto-projection, tombstone
> deprecation. ADRs 0111-0119 written. Tags created: sqliteengine,
> graphadapter, dgraphengine, storage/bbolt, idempotency, pebbleengine,
> metaengine/bench + 11 drifted-module bumps (retry, middleware, benchkit,
> stack/*). Remaining work is release hygiene and edge-case coverage.
>
> **WARNING:** Verify gate NOT confirmed GREEN after the tagging changes. See
> [status report](docs/status/2026-08-08_01-34_metaengine-v2-publishability-and-test-coverage.md).

### Release hygiene (blocks consumer trust)

- [x] 🔥 **Push 14 new tags to remote** — all 14 confirmed on `origin`
      (verified via `git ls-remote --tags origin`): `metaengine/pebbleengine/v4.0.0`,
      `metaengine/bench/v4.0.0`, `retry/v4.3.0`, `middleware/v4.3.0`,
      `benchkit/v4.3.0`, `stack/v4.3.0`,
      `stack/{memory,sqlite,turso,pebble,postgres}/v4.3.0`,
      `stack/{duckdb,bbolt,mysql}/v4.1.0`.
- [ ] 🔥 **Update CHANGELOG.md for all 14 new tags** — `TestTagContentMatchesChangelog`
      will fail without entries for each version section.
- [ ] 🔥 **Run `nix run .#verify` to completion** — verify gate was killed twice
      this session without confirming GREEN. 10 pre-existing lint findings in
      `system/` (wsl_v5, goconst, err113, prealloc) need fixing.
- [ ] **Run `nix run .#vulncheck`** — verify all tagged modules build under
      GOWORK=off (per-module consumer resolution).

### Test coverage (DONE — see [status report](docs/status/2026-08-08_02-30_metaengine-test-coverage-gaps-closed.md))

- [x] **Run new concurrent tests under `-race`** — SQLite, DuckDB, PG all pass
      `-count=3 -race` clean. 3 engines × 2 tests × 3 iterations = 18 green runs.
- [x] **Record-aware integration test through DuckDB engine** —
      `TestDuckDB_RecordStamping` in `duckdbengine/record_stamp_cgo_test.go`.
      Handles `map[string]any` scan return (DuckDB returns maps, not JSONValue).
- [x] **Record-aware integration test through PG engine** —
      `TestPostgres_RecordStamping` in `pgengine/record_stamp_test.go`.
      Uses testcontainers pattern. Completes 5-engine record-stamp coverage.
- [x] **`RunTransactionalTest` on Memory engine (baseline)** —
      `RunTransactionalBaselineTest` added to enginetest (pass-through tx: commit
      + error propagation, documents no-rollback limitation).
      `TestMemory_TransactionalBaseline` in `metaengine/memory_transactional_test.go`.
- [x] **Soak test with `AutoCRUDByConvention` through Pebble/DuckDB** —
      `enginetest.RunAutoCRUDSoak(t, eng)` extracted (220 lines → shared helper).
      Pebble: 4.0MB heap, 0 errors. DuckDB: 0.1MB heap, 0 errors. Both `-race` clean.
      Memory soak refactored to delegate to shared helper.

> **Prior session work (carried over):** Record-aware Pebble + SQLite tests,
> AutoCRUDByConvention Memory soak, RunTransactionalTest on SQLite, concurrent
> RunInTx test (enginetest + SQLite/DuckDB/PG), MultiAdd + LogAppend
> transactional subtests. badgerengine does NOT implement Transactional (no
> RunInTx method), so it is correctly excluded.

### Test coverage (follow-up items from this session)

- [ ] **Regen API stability golden** — `RunTransactionalBaselineTest` and
      `RunAutoCRUDSoak` are new exports in `enginetest`. Run
      `cd cmd/api-stability && GOWORK=off go run main.go -update`.
- [ ] **Run `nix run .#verify`** — verify gate not run this session. All tests
      verified per-module via `go test` + `-race`.
- [ ] **Add record-stamp test for badgerengine** — completes all-engine parity
      (currently: Memory, SQLite, Pebble, DuckDB, PG have it; Badger, Dgraph,
      GraphAdapter do not).
- [ ] **Add AutoCRUD soak for sqliteengine + pgengine** — currently only
      Memory/Pebble/DuckDB. SQLite and PG are the most-used SQL backends.
- [ ] **Consolidate `race_on.go`/`race_off.go` into `testutil/`** — pattern is
      now duplicated in 5 locations (benchkit, metaengine `_test`, transport/grpc
      `_test`, enginetest, metaengine `soak_autocrud_test.go`). Single canonical
      copy in testutil would eliminate the drift risk.
- [ ] **Extract `RunRecordStampTest(t, eng)` helper in enginetest** — record-stamp
      test body is copy-pasted across 4 engine modules (pebble, sqlite, duckdb, pg).
      A shared helper would eliminate ~100 lines of duplication.
- [ ] **DuckDB soak CI gating decision** — DuckDB soak takes 82-98s (vs Pebble
      0.27s, Memory 0.03s). Consider `testing.Short()` skip or nightly-only tag
      if it slows per-PR CI.
- [ ] **Add `// Caller owns engine Close.` doc comment to
      `RunTransactionalBaselineTest`** — matching the convention of
      `RunTransactionalTest` and `RunAutoCRUDSoak`.

### Module health (DONE this session)

- [x] **Add `metaengine/keycodec`, `metaengine/enginetest`,
      `testutil/pgtestcontainer`, `example/metaengine-quickstart` to AGENTS.md** —
      added to Quick Reference table, Test command, and Monorepo Structure tree.
      (api-stability `TestEveryGoModDirIsInModulesList` was already passing —
      keycodec/enginetest are packages within metaengine, not separate go.mod
      modules.)
- [x] **Fix COVERAGE GAPs in `check-module-layers.sh`** — script passes clean.
      No gaps remain (verified by running the script directly).

### ADR-0113: GraphBackend cleanup (DONE this session)

- [x] 🔥 **Removed GraphBackend from degraded engines** — SQLite, Pebble, Badger,
      and Iroh no longer implement `metaengine.GraphBackend`. These engines had
      O(N) BFS scan fallbacks (not real graph databases). Removed: assertion lines,
      graph method implementations, graph DDL/keycodec aliases, graph tests,
      dead `nextKey` function in badgerengine, unused `BFSNeighbors`/`GraphEdgeKey`/
      `GraphPrefixForward` from `keycodec`. Engines now return `ErrUnsupportedGraphOps`
      for graph queries — consumers use `graphadapter` or `dgraphengine` instead.
- [x] **Record-aware graphadapter integration test** — `TestAdapter_StoreIntegration_-
      RecordAware` in `graphadapter/adapter_test.go`. Proves the full ES-native
      pipeline: `Plan` → `ApplyRecord(Record)` → `Execute(Traversal)` → neighbors.
      Graph queries flow: Store → GraphBackend → graphadapter → graph.MemoryDriver.
- [x] **Kept GraphBackend on Memory (testing), Dgraph (native graph DB),
      GraphAdapter (canonical graph path)** — these three are the only engines
      that should support ADTGraph.

> **Remaining ADR-0113 work:** The `GraphBackend` interface itself is still
> exported (it's the capability-detection pattern, same as MapBackend/SetBackend).
> Fully deleting the interface type would require a different dispatch mechanism
> in Store/Execute — deferred as low-value churn. Dgraph engine still implements
> GraphBackend directly (intentional — it's a native graph DB).

> Long-term metaengine work (`metaengine-gen` code generator, Vector/Search/
> Spatial engine backends, generic `ScanResult[T]`, operator-driven engine
> selection) lives in [ROADMAP.md](ROADMAP.md).

---

## System Package (EXPERIMENTAL — P0/P1/P2/P3 shipped)

> The `system/` module implements the operator-configured CQRS topology.
> Driver registry wired, SQLite working through `New()`, projections E2E
> proven, MultiBus/SnapshotBackend/scream store wired, introspection real.
> koanf YAML config, DuckDB/PG Transactional, bus driver registry, scream store
> plan-drift detection, CommandAdapter/QueryAdapter serialization, and
> example/taskmanager migration all shipped. Tagged `system/v4.0.0`. **P2
> hardening + P2 test depth + P3 code quality all shipped.**
> HealthCheck, GracefulClose (with Drainer phase), ResetProjection,
> configurable checkpoint store, WithShutdownDependency, errors.Join in Close,
> SQLite engine HealthCheck, README Quick Start fix, doc-check arg validation.
> **Round 2 hardening shipped:** shutdown.go extraction (system.go 350→268),
> `namedEngine` struct (parallel slices eliminated), `ProjectionHostResource`
> removed (was a lie), HealthCheck on all 4 external-state engines (SQLite,
> DuckDB, Postgres, Pebble), 10 new tests (orderedEngines, Close, Drainer,
> ResetProjection restart-and-replay). See
> [status report](docs/status/2026-08-08_01-53_system-hardening-round2.md).

### P2 — Hardening (DONE — see [CHANGELOG.md](CHANGELOG.md))

- [x] **`system/README.md` Quick Start doesn't compile** — replaced with
      complete `package main` program using real API. Verified compiles + runs.
- [x] **Fix `cmd/doc-check` cmdguard arg-parsing** — `cobra.ArbitraryArgs`
      replaced with custom `fileArgs` validator (rejects non-existent files,
      directories, non-`.md` extensions).
- [x] **Add `system.HealthCheck(ctx)` method** — checks stopped state, pings
      engines implementing `metaengine.HealthChecker`, inspects projection host
      for `WorkerFailed` workers.
- [x] **Add `system.GracefulClose(ctx)`** — runs `Close()` in goroutine racing
      against `ctx.Done()`.
- [x] **Add `system.ResetProjection(ctx, name)`** — delegates to
      `projectionhost.Host.Reset()`. Returns `ErrNoProjectionHost` if not
      configured.
- [x] **Wire checkpoint store as configurable** — `DomainConfig.CheckpointStore`
      field; constructor uses it instead of hardcoded `memoryCheckpointStore`.

### P2 — Test depth (DONE)

- [x] **Deepen `TestSystem_CustomCheckpointStore`** — declares a real projection,
      produces events, starts host, waits for processing, asserts `saveCnt > 0`.
- [x] **Add `TestSystem_HealthCheck_FailedProjection`** — registers a projection
      with a failing decoder, `WithMaxRestarts(1)`, waits for `WorkerFailed`,
      asserts `HealthCheck` returns an error.
- [x] **Add `TestSystem_ResetProjection_Positive`** — configures projection,
      produces events, stops host, resets, verifies checkpoint is zero-value.
- [x] **Add `TestSystem_GracefulClose_SlowShutdown`** — creates system with
      projection host, produces events, GracefulClose completes within context.
- [x] **Add `TestSystem_HealthCheck_EngineUnhealthy`** — internal test injects
      a mock engine returning error from HealthCheck, verifies propagation.

### P3 — Code quality (DONE)

- [x] **Join errors in `System.Close()`** — uses `errors.Join` for all close
      errors (projection host + engines), matching `stack.Bundle.Close()`.
- [x] **Remove dead `s.closers` slice** — removed the `closers []func() error`
      field and its loop in Close (never populated in `New()`).
- [x] **Port `WithShutdownDependency`** — `DomainConfig.ShutdownDependencies`
      field + `ShutdownDependency` type + `orderedEngines()` topological sort
      (Kahn's algorithm, cycle fallback to creation order). Projection host
      always closes first (removed `ProjectionHostResource` — it was a lie).
- [x] **Add `Drainer` interface** — `GracefulClose` now drains via registered
      `Drainer`s before `Close`. `RegisterDrainer(d)` method on System.
- [x] **Document `DomainConfig.CheckpointStore` in README** — added
      DomainConfig Fields table with CheckpointStore + ShutdownDependencies.
- [x] **HealthCheck on all external-state engines** — SQLite (`db.PingContext`),
      DuckDB (`db.PingContext`), Postgres (`db.PingContext`), Pebble (lightweight
      point-read). All implement `metaengine.HealthChecker`.
- [x] **Extract shutdown logic to `shutdown.go`** — `Drainer`, `shutdownEdge`,
      `orderedEngines`, `RegisterDrainer` extracted from `system.go` (was at
      350-line CI limit). `namedEngine` struct replaces parallel slices.
- [x] **Tests for shutdown ordering + drainer + close joining** —
      `TestOrderedEngines_NoDeps`, `TestOrderedEngines_BasicOrdering`,
      `TestOrderedEngines_CycleFallback`, `TestOrderedEngines_UnknownNames`,
      `TestSystem_Close_ErrorJoining`, `TestSystem_Close_OrderMatchesOrderedEngines`,
      `TestSystem_RegisterDrainer_CalledBeforeClose`,
      `TestSystem_RegisterDrainer_ErrorPropagation`,
      `TestSystem_ResetProjection_RestartAndReplay` (SQLite persistence).

### Round 2 — Open follow-up items

#### HealthCheck completeness (high priority)

- [ ] 🔥 **Add `HealthCheck` to Badger engine** — `db.View(func(txn) error {
    return nil })` as a lightweight read-only probe.
- [ ] 🔥 **Add `HealthCheck` to Dgraph engine** — gRPC connection check via
      `dgo` client.
- [ ] **Add test for Pebble `HealthCheck`** — in-memory vfs, healthy + closed
      DB (closed DB returns non-`ErrNotFound` error).
- [ ] **Add test for SQLite `HealthCheck` on closed DB** — verify error
      propagation.
- [ ] **Add test for DuckDB `HealthCheck`** — CGo required.
- [ ] **Add test for Postgres `HealthCheck`** — testcontainers or nix
      ephemeral PG.
- [ ] **Add `HealthChecker` to `Store.HealthCheck` test matrix** — verify it
      delegates to all engines, not just the first.

#### System module improvements (medium priority)

- [ ] **Add `System.Drain(ctx)` standalone method** — expose drain without
      Close (for rolling deploys).
- [ ] **Add `System.EngineNames()` introspection** — returns engine names for
      diagnostics (`Explain()` prints count but not names).
- [ ] **Add `System.ShutdownOrder()` introspection** — returns resolved close
      order for debugging shutdown hangs.
- [ ] **Consider structured `HealthCheck` result** — `HealthCheckDetailed()`
      returning `[]EngineHealth{Name, Error}` while `HealthCheck()` stays
      first-error-only (non-breaking).
- [ ] **Add `System.LagPerProjection()`** — expose projection host lag via
      System.
- [ ] **Add `System.LagDuration()`** — same.
- [ ] **Add `System.WorkerStatus()`** — expose projection host worker status.
- [ ] **Add `System.RegisterCloser(name, closer)`** — let consumers register
      external resources for lifecycle management.
- [ ] **Test `GracefulClose` with context expiring during drain phase** —
      slow drainer + short context.
- [ ] **Test `GracefulClose` with context expiring during close phase** —
      slow engine + short context.

#### Shutdown ordering (medium priority)

- [ ] **Test `orderedEngines` with 5+ engines and multiple edges** — complex
      DAG topology.
- [ ] **Test `orderedEngines` with self-loop edge** — `{before: "a", after:
    "a"}` should be silently ignored.
- [ ] **Test `orderedEngines` with duplicate edges** — should not double-count
      inDegree.
- [ ] **Add dedup guard to `orderedEngines` cycle fallback** — currently
      correct but fragile if refactored; explicit guard prevents accidental
      breakage.

#### Documentation (low priority)

- [ ] **Update `AGENTS.md` system module section** — remove
      `ProjectionHostResource` from Modules list, mention all 4 engine
      HealthChecks.
- [ ] **Add `ShutdownDependency` example to README Quick Start**.
- [ ] **Add `Drainer` example to README**.
- [ ] **Update `metaengine` README** — list which engines implement
      `HealthChecker`.

#### Testing polish (low priority)

- [ ] **Split `system_internal_test.go` if it grows past ~250 lines** —
      currently 7 tests + 3 mock types.
- [ ] **Add `TestSystem_Close_NoEngines`** — Close on a system with zero
      engines.
- [ ] **Add `TestSystem_Close_ProjectionHostError`** — projection host Stop
      fails, engine close still runs.
- [ ] **Add `TestSystem_GracefulClose_NoDrainers`** — GracefulClose with zero
      registered drainers (should just Close).
- [ ] **Add `TestSystem_GracefulClose_MultipleDrainers`** — verify all
      drainers called in order.
- [ ] **Add comment to `TestSystem_ResetProjection_RestartAndReplay`** —
      explain SQLite shared-cache DSN pattern.

#### Release (when ready)

- [ ] **Tag `metaengine/sqliteengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `metaengine/duckdbengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `metaengine/pgengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `metaengine/pebbleengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `system/v4.1.0`** (removed `ProjectionHostResource`,
      `namedEngine` refactor, new tests).
- [ ] **Verify all module versions are monotonically increasing before
      tagging** — check `git tag -l '<module>/v4*' | sort -V | tail -1`.

#### Integration (future)

- [ ] **Integration test: SQLite source-of-truth + Memory projections +
      HealthCheck** — end-to-end system with real engines.
- [ ] **Integration test: Pebble source-of-truth + HealthCheck**.
- [ ] **Integration test: GracefulClose with real Watermill router as
      Drainer**.

---

## bbolt Storage Backend

> Full storage backend shipped: EventStore, SnapshotStore, CheckpointStore,
> KVAdapter, CommandStore, QueryStore, Backend facade. Streaming iterators
> (`StreamingSource`/`StreamingJournal`), OTel span instrumentation, contract
> test suite (29 tests), durability tiers via `stack/bbolt`.
> `storage/bbolt/v4.0.0` tagged and pushed. CommandStore/QueryStore contract
> tests, same-stream contention test, and contention benchmark all shipped
> (42 total tests, race-clean). See
> [status report](docs/status/2026-08-08_01-17_bbolt-test-and-benchmark-coverage.md).

### DONE this session

- [x] **CommandStore contract tests** — 8 tests: Save/Load, DuplicateDetection,
      AppendBatch, AppendBatchDuplicate, LoadEmptyStream, ReadAll, ReadFrom
      (pagination), LoadFromTimestamp.
- [x] **QueryStore contract tests** — 4 tests: SaveAndLoadQueries, DuplicateDetection,
      ReadAllQueries, ReadQueriesFrom (pagination).
- [x] **Same-stream concurrency contention test** — `TestContract_SameStreamContention`:
      10 goroutines race same stream with `expectedVersion=0`, exactly 1 wins.
- [x] **Add bbolt to `stack/bench/` contention benchmark** — added to
      `BenchmarkContention_Persistent_SameStream` alongside sqlite/pebble.
- [x] **`WithBatchSize` for `AppendBatch`** — dismissed: code already writes all
      events in a single atomic bbolt tx. Splitting into sub-batches would break
      atomicity.

### DONE this session (test suite consolidation)

- [x] **Add `stack/bbolt/contract_test.go`** — mirrors the pebble pattern:
      `contracttest.RunSuite(t, factory)`. All 5 subtests pass.
- [x] **Add bbolt to `durability_tiers_test.go`** in stack/bench —
      `BenchmarkDurabilityTiers_Bbolt` tests all 3 tiers (Strict/Normal/Relaxed).
- [x] **Extract `commandtest`/`querytest` shared packages** —
      `command/commandtest/store_suite.go` (RunStoreSuite + 6 subtests),
      `query/querytest/store_suite.go` (RunStoreSuite + 4 subtests). Pebble +
      bbolt consumer tests refactored to thin wrappers (892 → 136 lines).
- [x] **Modernize `for` loops in `storage/bbolt/contract_test.go`** — 4 instances
      of `for i := 0; i < N; i++` → `for i := range N` (gopls rangeint hints
      cleared).

### Remaining

- [ ] **Refactor `storage/memory/` command/query store tests** — 316+248 lines
      with the same ~90% duplication pattern. Now that `commandtest`/`querytest`
      shared suites exist, memory backend should adopt them too.
- [ ] **Add `command/commandtest` to AGENTS.md module list** — missing from the
      Quick Reference Modules row (lists `query/querytest` but not
      `command/commandtest`).
- [ ] **Add `command/commandtest` to `cmd/api-stability/main.go` modules list** —
      `query/querytest` is tracked; `command/commandtest` is not. The
      `TestEveryGoModDirIsInModulesList` gate will catch this once it's a
      separate package path.
- [ ] **Add `doc.go` to `command/commandtest/`** — `query/querytest` has one;
      `commandtest` puts the package doc in `store_suite.go` header instead.
      Consistency issue.
- [ ] **Add self-test to `commandtest`** — run the suite against
      `storage/memory.MemoryCommandStore` to validate the suite itself (mirrors
      how `eventtest` is tested). Currently `[no test files]`.
- [ ] **Remove unused `time` import in `durability_tiers_test.go`** — line 123
      `var _ = time.Second` suppresses an import that is genuinely unused.
      Pre-existing, not from this session.

---

## Irohengine

> Level 2 prototype shipped with CRDT-safe operations. Three transports:
> InProcessNetwork (goroutine, no CGo), loopback (real TCP, no CGo), QUIC
> (`iroh-go` C bindings, CGo required). loopback/v4.0.0 + quic/v4.0.0 tagged.

- [x] **QUIC transport integration with `adttest.RunMatrix`** — `TestQuicADTMatrix`
      runs the full 10-ADT matrix against a QUIC-backed replicated engine (StreamLog
      auto-skipped, not CRDT-safe). Added `TestLoopbackADTMatrix` for the loopback tier.
      Parity tests for LWW (`TestQuicLWWResolution`), PN-Counter (`TestQuicPNCounter`),
      and MapUpdate-does-not-replicate (`TestQuicMapUpdateDoesNotReplicate`) all pass.
- [x] **Non-CRDT op rejection on QUIC path** — `TestQuicMapUpdateDoesNotReplicate`
      verifies `MapUpdate` operations stay local-only over the QUIC transport.
      CBOR round-trip type handling (int→uint64) addressed with `BeEquivalentTo`.
- [x] **Fix `TestQuicSetConvergence` flakiness** — both SetAdd elements ("go" + "cqrs")
      now checked inside the same `Eventually` block (was: Eventually for first,
      direct assertion for second). Also fixed `TestQuicPNCounter` (was `time.Sleep`,
      now `Eventually` on both nodes).
- [x] **Fix `TestLoad_ConcurrentLoadsCoalescedBySingleflight` flake** — increased
      coalescing window from 50ms to 200ms, added `runtime.Gosched()` after barrier
      release to yield immediately to waiting goroutines.

---

## cqrs-lint

> 192 rules across 10 categories. Config presets, `--adoption`/`--scorecard`/
> `--group-by` flags, SARIF + Markdown output, self-lint mode, block-level
> suppression, metaengine-aware detection (F018-F026), scorecard metaengine
> section, cross-format consistency tests. v4.4.0 tagged. Self-lint clean
> (0 CRITICAL, 0 ERROR, 0 load errors, 0 stale suppressions).
>
> **Done 2026-08-08** (see [status report](docs/status/2026-08-08_02-28_cqrs-lint-backlog-triage.md)):
> C001/D012/C008 false-positive fixes, S006/A018/B004 regression tests,
> A034 per-module migration, SARIF logicalLocations, D007 self-lint fix
> (5 instances), C023 self-lint fix (1 instance).

### False-positive fixes (DONE)

- [x] **C001** — read-only bbolt `Begin(false)` + composite-literal escape
      (`&iter{tx: tx}`) no longer flagged. 2 regression tests added.
- [x] **D012** — `main` package files excluded (CLI tools use `fmt.Print*`
      intentionally). Regression test added.
- [x] **C008** — removed `"rate"` from weak fields + added
      `nonMonetaryFieldPatterns` denylist (latency, throughput, ratio,
      percentage, duration, seconds, qps, rps, fps). 2 regression tests.
- [x] **Removed stale `//cqrs-lint:ignore(C001)`** in `storage/bbolt/kv_adapter.go`
      (no longer needed after fix).

### Regression tests (DONE)

- [x] **S006** — weak-tier suppression for local-only projects + fires for
      server projects.
- [x] **A018** — suppressed by Save/Publish/Dispatch calls + FoldInfo.
- [x] **B004** — no finding for < 3 fields + suppressed by existing `New*`
      constructor.

### Per-module migration (PARTIALLY DONE)

- [x] **A034 migrated** — `ctx.FeatureProfile.HasMetaengine` →
      `ctx.ProfileForFile(gf.Path).HasMetaengine`.
- [ ] **F015/F016/F017, F022, F026** — intentionally project-level (F-series
      assess project-wide adoption). No migration needed unless they start
      false-positiving in multi-module workspaces.

### SARIF logicalLocations (DONE — test gap remains)

- [x] **SARIF logicalLocations populated** — `run.logicalLocations[]` from
      scored modules (used + missing), result-level index cross-references.
- [ ] **Dedicated SARIF logicalLocations test** — verify array is populated,
      index mapping is correct, `kind` is `"module"`.

### Self-lint triage (PARTIALLY DONE)

- [x] **D007** — 5 `event.NewEvent(` → `event.New(` instances fixed
      (metaengine-quickstart, encryption, watermill).
- [x] **C023** — 1 `engine.Close()` unchecked call fixed (irohengine demo).
- [ ] 🔥 **C023 false positive on void-return `Close()`** — `dgo` client's
      `Close()` returns void but C023 flags it. Needs type-awareness: check
      call expression returns an error before flagging. Requires `TypesInfo`
      (unavailable in `BuildContextFromSource` test helper).
- [ ] **~80 C033 bare `return err` findings** — across `metaengine/*engine/`
      and `benchkit/`. All INFO-level. Needs bulk-fix vs suppress decision.
- [ ] **~15 D014 missing json tags** findings.
- [ ] **~8 C034 `go func()` without context** findings.
- [ ] **~6 P012/P013 SQLite without WAL/busy_timeout** findings.
- [ ] **~8 A032 string/int fields instead of branded ID** findings.

### Open work (NOT STARTED)

- [ ] 🔥 **Run cqrs-lint against real consumer projects** — validate
      false-positive rates against Kernovia, Standup-Killer, bank-sync,
      cqrs-htmx, DiscordSync, timesheets, crush-daily, KeyHolderAI. This is the
      single highest-value non-coding task for cqrs-lint trustworthiness.
- [ ] **C008 word-boundary matching** — `TotalDays` matches `total`; add
      word-boundary regex to prevent substring false positives.
- [ ] **D007 auto-fix test** — `--fix` path (replaces `event.NewEvent` →
      `event.New`) is untested.
- [ ] **Generalize C001 `Begin(false)` check** — currently bbolt-specific;
      other DBs may use different read-only patterns.
- [ ] **Deferred P-series rules** — `metaengine.Query` without type parameter,
      `MapUpdate` on replicated engine, Store never Closed, `metaengine.On`
      wrong handler signature. Each needs advanced type inference.
- [ ] **L1.5 domain severity calibration** — `DomainKind` enum +
      `applyDomainBias` shipped; still needs broader testing against
      financial/security projects.
- [ ] **~14 remaining Pareto backlog items** — see the
      [Pareto plan](docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md).
      Highest impact: L1.30–L1.33 deep pattern detection, L1.47–L1.51 new rule
      categories (DOC/OBS/RES/DI).
- [ ] **Tag cqrs-lint v4.5.0** — with all false-positive fixes + regression
      tests from this session. Or wait for C023 fix + consumer validation.

---

## Code Quality

> **4 items resolved 2026-08-08** — metadata immutability sweep
> (`CustomData.WithCustom` added, `EnsureCustom` deprecated), query parity
> (`query.WithCustomMetadata` added), README updated (EnsureCustom →
> WithCustom, fixed false alias claims), `.golangci.yml` exclusion sprawl
> fixed (100% documented, 12 ireturn blocks → 1, duplicates removed). See
> [status report](docs/status/2026-08-08_02-01_metadata-immutability-query-parity-lint-cleanup.md).

- [x] **`metadata.CustomData[K]` immutability gap** — added value-receiver
      `WithCustom(key K, value string)` matching the command/query pattern.
      `EnsureCustom()` deprecated with `// Deprecated:` doc comment. 5
      immutability tests added.

- [x] **`query.WithCustomMetadata` missing** — added
      `query.WithCustomMetadata(key, value string) Option`, mirroring
      `command.WithCustomMetadata`. 2 tests added (single + accumulate).

- [x] **Stale `metadata/README.md`** — updated: methods table shows
      `WithCustom` + deprecation note, fixed false "command.Metadata IS
      CustomData" claims, added standalone-struct usage example.

- [x] **Fix `.golangci.yml` exclusion sprawl** — every exclusion now has a
      `#` rationale comment (was ~50% undocumented). Consolidated 12
      scattered `ireturn`-only blocks into 1 regex group. Removed 4
      duplicate-path/duplicate-linter entries. ~70 rules → ~45 documented.

### DONE 2026-08-08 (event.EnsureCustom + metadata.CustomData deprecation)

> See [status report](docs/status/2026-08-08_02-29_deprecate-ensurecustom-customdata.md).

- [x] **Deprecate `event.EnsureCustom()` free function** — added
      `event.Metadata.WithCustom(key, value) Metadata` value-receiver
      method. Migrated all 4 production call sites (`event.WithCustom`
      option, `watermill/protocol.go`, `event/tombstone.go` inline init,
      fuzz test). Marked `EnsureCustom` `// Deprecated:`. Signing/encryption
      had ZERO direct coupling — the "defer to dedicated session" framing
      overstated risk (was ~30 min mechanical migration).
- [x] **Soft-deprecate `metadata.CustomData[K]`** — added `// Deprecated:`
      doc directing consumers to standalone-struct pattern. Type kept for
      major version (library-first rule). `event.Metadata` is now the third
      standalone-struct example alongside `command.Metadata` and
      `query.Metadata`.

### Open follow-up items

- [ ] **Add `// Deprecated:` to `event.CustomData` v3-compat alias** —
      `event/v3_compat_aliases.go:31` re-exports `metadata.CustomData[K]`
      but the alias does not carry the deprecation notice. Should mirror
      the base type.
- [ ] **Migrate remaining test callers off deprecated `EnsureCustom`** —
      `event/customdata_test.go:177,190` and `metadata/metadata_test.go:252,267`
      still call `CustomData[K].EnsureCustom()`. Migrate to `WithCustom` or
      keep as backward-compat coverage (decision needed).
- [ ] **Per-module `.golangci.yml` split** — the monolithic config is now
      fully documented, but golangci-lint v2 `config-dirs` would give each
      module ownership of its own exclusions. Evaluate tradeoff: single
      source of truth vs locality.

---

## Pre-Existing Failures (discovered 2026-08-08)

> Found while verifying the `EnsureCustom` deprecation. Both confirmed
> pre-existing via `git stash` — NOT caused by the deprecation work.

- [ ] 🔥 **Fix `cmd/api-stability/main.go:172` — `collectExports` undefined** —
      the api-stability tool itself does not compile. Blocks ALL api-surface
      golden regeneration. A meta-test should ensure `cmd/api-stability`
      compiles (catches this class of breakage).
- [ ] **Regenerate api-stability golden** — after fixing the tool. The
      golden is stale: missing `event.Metadata.WithCustom` (added this
      session) and likely other symbols from intervening daemon commits.
- [ ] 🔥 **Fix 4 pre-existing watermill CBOR test failures** —
      `TestRoundTrip`, `TestMessageToEvent_DefaultsJSONWhenNoEncoding`,
      `TestEventToMessage_PreservesEncoding/json`, `TestEventPublisher_RoundTripCBOR`.
      Root cause: default codec changed to CBOR but watermill tests still
      expect JSON defaults. Needs systematic sweep of watermill tests for
      codec-default assumption drift.

---

## Dedup

> Clone groups driven to **0 at threshold 3** (was 65). All thresholds (7, 4, 3) reduced to 0 through shared helper extraction. `.art-dupl-baseline.json`
> baseline: 64 groups. `nix run .#check-duplication` gate enforces no-new-clones.

- [x] **Investigate threshold-2 clone groups** — investigated 2026-08-08. Findings:
  - `capitalizeFirst` / `titleCase`: extracted `benchkit.TitleCase` + `benchkit.Truncate`, eliminated dup in `cmd/cqrs-bench` (2 clone groups gone).
  - `isCBORData`: **accepted** — cross-module (bbolt vs pebble, separate go.mod), 4 lines, already documented.
  - `recordErr`: **accepted** — OTel boilerplate (100 occurrences), cross-module, 1-line calls. bbolt has local helper.
  - `startStreamSpan`: **accepted** — module-local helpers already exist. Call-site boilerplate (`span := ...; defer span.End()`).
  - Remaining t=2 clones are test boilerplate (`t.Parallel()` + `ctx := context.Background()`), cross-module isolation, and DuckDB aggregations internal patterns.
- [x] **Extract `renderTable(b, headers, rows)` helper** — generalized `renderKeyTable` into
      `renderTable` + `writeTableRow` + `writeTableSeparator` + `columnWidths`. Refactored
      `renderRulesConfig` to share the same primitives.
- [x] **`deferClose(closer)` helper** — added `metaengine.DeferClose(c Closer)`. Replaced
      `defer func() { _ = X.Close() }()` with `defer metaengine.DeferClose(X)` across 47
      production sites + 17 test sites in sqlite/pg/duckdb/pebble/badger/dgraph engines.

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).

- [ ] **Pin GitHub Actions to commit SHAs** — 72+ unpinned actions
      (supply-chain risk).
- [ ] **Add self-lint to CI** — `cqrs-lint --self-lint` works but no GitHub
      Actions step gates it.
- [ ] **Add `--fail-on-stale-suppressions` CI gate** — prevents stale
      `//cqrs-lint:ignore` directives from accumulating.

---

## Integration Test Infrastructure

> Nix-based integration test infrastructure shipped: ephemeral PG, NixOS VM
> tests (PG+MySQL), nspawn MySQL (~15s), projectionhost PG crash-restart,
> scheduling/sqlstore durable timers. All in ephemeral-pg.sh.

- [ ] **macOS verification of ephemeral PG** — script claims cross-platform but
      never tested on Darwin. (M34)
- [ ] **Cache ephemeral PG data dir** — skip `initdb` on repeated runs. (M35)
- [ ] **Performance profiling: ephemeral PG vs testcontainers** — measure
      speedup and document. (M36)
- [ ] **Explore `nixos-container` as lighter-weight VM alternative** (M37)
- [ ] **DuckDB CGo VM test** — hermetic DuckDB testing with GCC in VM. (M38)
- [ ] **SQLite WAL concurrency VM test** — concurrent access patterns. (M39)
- [ ] **Turso sync VM test** — real libSQL server. (M40)
- [ ] **Pebble backup/restore lifecycle VM test** (M42)
- [ ] **Contract test suite across ALL backends in VMs** — SQLite, PG, MySQL,
      DuckDB simultaneously. (M46)
- [ ] **Ephemeral Redis/NATS for future integration tests** — Watermill adapter
      testing with real brokers. (M47)
- [ ] **`scripts/test-integration.sh` aggregator** — auto-detect best strategy
      (ephemeral, VM, or testcontainers). (M48)

---

## Layer Enforcement

> `check-module-layers.sh` has a self-enforcing coverage guard. 79 go.mod
> files, 77+ modules in `go.work`. ADR-0046 updated to the seven-tier model.

- [ ] **Rename `FOUR-TIER-MODEL.md` → `SEVEN-TIER-MODEL.md`** — filename lies
      about content (H1 says "seven-tier" but filename says "four-tier").
- [ ] **Remove dead `EXCEPTIONS[storage]="listing"`** — listing moved to
      Layer 3, the exception is no longer needed.
- [x] **Fix 16 COVERAGE GAPs** — `check-module-layers.sh` passes clean. All
      newer modules are now in LAYER/DEP_BUDGET maps. (Verified 2026-08-08.)
- [ ] **Expand go-arch-lint to remaining modules** — only 6 modules have
      per-module go-arch-lint configs. The bash script is the enforcement
      mechanism for the rest.
- [ ] **Consider rewriting `check-module-layers.sh` as `cmd/check-layers`** —
      330 lines of bash. A Go program would be more maintainable and testable.

---

> **Deferred debt resolved.** Ghost bus removal (ADR-0028) and metadata
> aliases completion (ADR-0031) are both DONE. `retry/` → `go-retry` and
> `idempotency/` → `go-idempotency` extraction is DONE. `retry/` is now
> DEPRECATED (re-export shim, consumers should import `go-retry` directly).

---

## Declined / Rejected (do not re-litigate)

> Kept here so decisions are not re-litigated. Full rationale in the linked
> ADRs/reviews.

- **Wire `#verify-parallel` into CI** — declined 2026-07-29. CI already has a
  per-module matrix strategy that provides better isolation.
- **Add `#verify-fast` as a pre-merge CI gate** — done (already wired as
  `verify-fast-gate` at ci.yml:128).
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Use
  `RelationalProjection` (junction tables). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
- **Redis adapter** — the author is not a fan of Redis. See ROADMAP Non-Goals.
