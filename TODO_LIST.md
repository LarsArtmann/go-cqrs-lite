# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## cqrs-lint

> See [ROADMAP.md](ROADMAP.md) → Raw Ideas for long-term L-effort items
> (cqrs-lint consumer validation, per-module golangci split).
>
> FP elimination session (2026-08-08 → 2026-08-09): 32 FPs eliminated across
> 15 rules (128→96 findings on 8 consumer repos, 0 TPs lost, 0 critical/error
> FPs). v4.3.0 released. See
> `docs/status/2026-08-09_00-19_cqrs-lint-fp-elimination-execution.md`.

- [ ] 🔥 **Write regression unit tests for FP fixes** — 13 of 15 rule fixes
      lack dedicated tests (only C002 + E007 have them). Each test uses
      `BuildContextFromSource` + `ruletest.RunDetector` + `AssertRule`:
      A005 (non-event-bus receiver), C027 (non-event-bus receiver),
      S010 (requires Use() wiring), A032 (form-tag structs + display packages),
      C013 (json:"-" skip), C034 (HTTP shutdown pattern), C035 (serialization DTO),
      E009 (custom HTTP), D005 (code blocks + import paths), E007 (package
      has registration).
      _(Effort: M)_
- [ ] 🔥 **Replace `PackagesWithRegistration` with precise per-type tracing** —
      the current E007 fix suppresses ALL findings in a package that has ANY
      `RegisterTyped`/`RegisterQuery` call. Over-suppresses: if a package
      registers 9 of 10 queries, the 10th won't be flagged. Replace with
      per-type registration tracing (trace through generic wrapper functions
      like crush-daily's `register[Q]()`).
      _(Effort: M)_
- [ ] **Reclassify misclassified FPs in validation report** — at least 9 of the
      original 39 "FPs" were actually TPs: D005 x4 (genuinely stale docs),
      A005 x1 (DualWriteBus.SubscribeAll is real), A032 x5 (PluginID on domain
      commands). Update `docs/status/2026-08-08_cqrs-lint-false-positive-validation.md`.
      _(Effort: S)_
- [ ] **Improve B029-B031 `isBusName` heuristic** — require `.Use()`/`.Publish()`
      calls, not just suffix match on variable names. The `Use()`/`UsePublish()`
      argument-checking pattern was proven for S010/C027/A005 in the FP
      elimination session — port it to B029-B031.
      _(Effort: S)_
- [ ] **Improve D018 `collectEventNewTypes`** — use type info for precise
      `event.NewEvent` detection (currently matches any `NewEvent`).
      _(Effort: S)_
- [ ] **Raise C041 confidence to Medium (0.5)** — Save ignoring
      `expectedVersion` is a real bug, not low-confidence.
      _(Effort: S)_
- [ ] **Add integration test: lint `example/taskmanager`** — verify expected
      findings end-to-end through the full rule pipeline.
      _(Effort: M)_

---

## Metaengine v2 — Dgraph Engine

> Dgraph engine integration-tested against live Dgraph 25.4.0 (2026-08-08).
> `nix run .#ephemeral-dgraph` spins up Zero+Alpha from nixpkgs. All 46 tests
> pass (6.072s). DQL injection fixed (`QueryWithVars`), MapDelete fixed
> (Dgraph 25.x explicit null-predicate deletion), regression test added.
>
> 2026-08-09 session: lint sweep (0 issues across 73 modules), 8 missing modules
> added to `testModules`, depguard allowlist fixed, Dgraph added to
> `test-all-backends.sh` Phase 4, per-test cleanup via `main_test.go` DropAll.
> See `docs/status/2026-08-09_01-09_dgraph-lint-sweep-module-registration-full-verification.md`.

- [ ] **Add Dgraph VM test** (`nix/vm/dgraph.nix`) — NixOS VM test for CI
      reproducibility, matching postgres-vm/mysql-vm/duckdb-vm pattern.
      _(Effort: M)_
- [ ] **Add Dgraph retry logic** for transient `"Please retry again"` errors.
      _(Effort: S)_
- [ ] **Add Dgraph connection pool tuning** — gRPC `MaxCallRecvMsgSize` for
      large result sets.
      _(Effort: S)_
- [ ] **Fix LogBackend same-nanosecond collision** — `time.Now().UnixNano()`
      can collide under concurrency. Consider atomic counter per collection
      or ULID.
      _(Effort: S)_
- [ ] **Fix CounterIncrement over-read** — queries ALL counters in collection
      even for 1-2 key deltas. Query only delta keys instead.
      _(Effort: M)_
- [ ] **Write dedicated unit tests for MultimapBackend** — empty key,
      limit=0, concurrent append ordering (currently only tested via
      `adttest.RunMatrix`).
      _(Effort: S)_
- [ ] **Write dedicated unit tests for LogBackend** — same-nanosecond
      collision, empty collection, limit > entries.
      _(Effort: S)_
- [ ] **Add StreamLogBackend to dgraphengine** — currently skips (8/11 ADTs).
      _(Effort: M)_
- [ ] **Run full `-race` suite on dgraphengine with fresh Dgraph** — non-race
      run passed clean (46/46); clean `-race` run not yet re-attempted after
      confirming transient failure was stale-process related.
      _(Effort: S)_
- [ ] **Tag `dgraphengine/v4.0.2`** after verifying full gate passes.
      _(Effort: S)_

---

## Irohengine / Replicated Engine

- [ ] **Add runtime protocol-mismatch detection for QUIC stream pooling** — a
      pooled sender connected to a non-pooled receiver silently hangs (receiver
      calls `ReadToEnd` waiting for `Finish()` that never comes). Detect via a
      magic byte in the first frame and return a clear error.
      _(Effort: S)_
- [ ] **Add stream-reuse counter to `peerConn`** — increment each time
      `sendOpPooled` opens a new BiStream. Tests can assert that N ops over a
      pooled connection used only 1 stream (proving reuse, not just correctness).
      _(Effort: S)_
- [ ] **Extract shared framing constants** — `frameHeaderSize`, `errFrameTooLarge`
      are duplicated between `quic/frame.go` and `loopback/frame.go`. Move to
      `irohengine/framing.go` (protocol constants only; I/O stays per-transport).
      _(Effort: S)_
- [ ] **Port injectable-clock pattern to QUIC LWW tests** — `TestQuicLWWResolution`
      still relies on replication-time-gap for timestamp ordering. Could use
      `WithClock` for determinism (same pattern as the in-process tests).
      _(Effort: S)_
- [ ] **Extract `RunConvergenceSuite(t, factory)`** — shared test harness for
      all 3 transports (~200 lines dedup between in-process, loopback, QUIC).
      _(Effort: M)_

---

## Code Quality / Dedup

> Dedup session 2026-08-09: 11→3 clone groups at threshold 4. 8 fixed, 3
> accepted. 5 test helpers + 2 production helpers extracted. Verify gate
> NOT run (individual module tests only). See
> `docs/status/2026-08-09_01-02_dedup-threshold-4-cleanup.md`.

- [ ] 🔥 **Run `nix run .#verify`** — the dedup session changed ~25 files
      across 6 modules but only ran individual `go test`/`go vet`. Lint,
      doc-check, race detection, and cross-module integration were NOT
      verified.
      _(Effort: S — just run it and fix failures)_
- [ ] **Eliminate `newDuckDBPushdown` dead wrapper** — collapsed to a 1-line
      `return mustNewDuckEngine(t)` but left 5 callers pointing at it.
      Replace all callers with `mustNewDuckEngine` directly, delete function.
      _(Effort: S)_
- [ ] **Extract `DistinctValues` row-scan into shared SQL helper** — 23-line
      loop duplicated between `duckdbengine/aggregations.go` and
      `sqliteengine/aggregations_grouped.go`. `storage/sql/` exists for this
      purpose but both engine modules are Tier 4 (importing it inverts the
      dependency). Consider a new `metaengine/sqlutil/` (Tier 0/1) instead.
      _(Effort: M)_
- [ ] **Fix non-deferred `eng.Close()` in healthcheck tests** — both
      `pgengine/healthcheck_test.go:34` and
      `duckdbengine/healthcheck_cgo_test.go:36` call bare `eng.Close()`
      (not deferred) — leaks the engine on test failure.
      _(Effort: S)_
- [ ] **Refactor remaining engine-setup boilerplate** — 4 sites in
      `duckdbengine/stream_log_cgo_test.go` use `t.Fatalf` + `DeferClose`
      instead of the skip+defer pattern; 1 site each in
      `pgengine/healthcheck_test.go` and `duckdbengine/healthcheck_cgo_test.go`
      have non-deferred Close. Could add a `mustNewDuckEngineOrFatal`
      variant or standardize on skip+cleanup.
      _(Effort: S)_
- [ ] **Extract bbolt/pebble backup lifecycle test suite** — the 2 largest
      remaining clone groups (73 + 46 lines) are near-identical test files
      in `storage/bbolt/backup_lifecycle_test.go` and
      `storage/pebble/backup_lifecycle_test.go`. A shared `backuptest`
      module would eliminate both, but requires a new test-only Go module
      that both backends depend on.
      _(Effort: M)_
- [ ] **Rename `helper_test.go` → `helper_cgo_test.go`** in duckdbengine for
      naming consistency with sibling `_cgo_test.go` files.
      _(Effort: S)_
- [ ] **Update AGENTS.md "Dedup helper patterns" section** — document the
      5 new test helpers (`mustNewPgEngine`, `mustNewDuckEngine`,
      `setupSeededAggTest`, `assertTxCommitSetup`, `saveOneCommand`) and
      the `stdQueryInit` / `drainAll` production helpers.
      _(Effort: S)_
- [ ] **Scan remaining engine modules for setup boilerplate** —
      `metaengine/badgerengine/`, `metaengine/pebbleengine/`,
      `metaengine/dgraphengine/` likely have the same `New(...) + err +
      skip + defer Close` pattern that was fixed in pgengine/duckdbengine.
      _(Effort: M)_
- [ ] **Consolidate `deferClose` helper** — 3 copies across test packages
      (storage/pebble, storage/bbolt, metaengine). Consider shared
      `storage/internal/closeutil` package.
      _(Effort: M)_
- [ ] **Audit `.golangci.yml` exclusion blocks** — `system/` (20 linters
      disabled), `cmd/cqrs-lint/` (13), `metaengine/` (15) have the broadest
      exclusions. Narrow where safe.
      _(Effort: M)_
- [ ] 🔥 **Add meta-test enforcing `testModules == all go.mod dirs`** — 8
      modules were silently missing from `flake.nix` `testModules`, meaning
      they were never built, tested, or linted. Like
      `TestEveryGoModDirIsInModulesList` in api-stability.
      _(Effort: S)_
- [ ] **Add CI check comparing `go.mod` requires vs depguard allow list** —
      dependencies are only added to `.golangci.yml` after lint fails.
      _(Effort: M)_
- [ ] **Investigate `gci` vs `goimports` disagreement** — two test files
      (`pgengine/testcontainer_test.go`, `duckdbengine/helper_test.go`) have
      `gci` issues that `nix fmt` doesn't fix.
      _(Effort: S)_
- [ ] **Document `testModules` ↔ `lintModules` coupling in AGENTS.md** —
      they share the same list; adding a module requires updating both.
      _(Effort: S)_
- [ ] 🔥 **Fix benchkit timing flakes** — `TestRun_SQLite_DurationAborts`,
      `TestCompare_ThreeBackends`, `TestRun_CancelledContext` fail under
      parallel test load with hardcoded 5s thresholds. Apply the
      `testutil.RaceEnabled` relaxed-bound pattern or increase thresholds to
      account for system load.
      _(Effort: S)_
- [ ] **Remove unused `newSQLiteEngineForPath`** in
      `metaengine/bench/sqlite_factory_test.go:26` (gopls unusedfunc warning).
      _(Effort: S)_

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must
  v0.1.2).
- [x] **Wire `#check-arch` into the verify gate and CI** — replaced
      `#check-layers` with `#check-arch` in verify, verify-fast, ci app,
      and `.github/workflows/ci.yml`. `check-arch` is a strict superset
      (calls Layer 1 internally + Layer 2 go-arch-lint per-module).
- [x] **Add go-arch-lint as a nix dependency in `#check-arch`** — added
      `pkgs.go-arch-lint` (v1.17.0 from nixpkgs) + `findutils` + `gnugrep`
      to the app's runtimeInputs. No longer depends on system PATH.
- [x] **Document CHANGELOG release process** — added `tag-release.sh`
      workflow, CHANGELOG-to-tag constraint explanation, and two-layer
      architecture model docs to CONTRIBUTING.md.

---

## Integration Test Infrastructure

- [ ] **macOS verification of ephemeral PG** — `scripts/ephemeral-pg.sh` claims
      cross-platform but was never tested on Darwin.
      _(Effort: M)_
- [ ] **Write actual Redis/NATS integration tests** — `ephemeral-redis.sh` and
      `ephemeral-nats.sh` exist but no Go tests use them. Watermill Redis
      Streams and NATS JetStream roundtrips untested.
      _(Effort: M)_
- [ ] **Write actual Dgraph integration tests in Go** — ephemeral-dgraph script
      exists; ADT tests run but no system-level integration test.
      _(Effort: M)_
- [ ] **Add stale-process detection (PID file) to `ephemeral-dgraph.sh`** —
      orphaned Dgraph processes from prior sessions cause transient test
      failures.
      _(Effort: S)_

---

## Layer Enforcement / Architecture

- [ ] **Add `.go-arch-lint.yml` for `metaengine/`** — 16+ production files
      (planner.go, engine.go, dsl.go, etc.) with complex internal structure
      and no intra-module enforcement.
      _(Effort: M)_
- [ ] **Add `.go-arch-lint.yml` for `stack/`** — 11 production files, composition
      layer with clear internal dependencies, no enforcement.
      _(Effort: S)_
- [ ] **Add `.go-arch-lint.yml` for `decider/`** — core domain module, no
      intra-module enforcement.
      _(Effort: S)_
- [ ] **Add `.go-arch-lint.yml` for `projectionhost/`** — lifecycle module
      with complex internal structure.
      _(Effort: S)_
- [ ] **Add meta-test: every `.go-arch-lint.yml` is parseable and components
      match real packages** — no test today asserts configs are valid or that
      declared components resolve to actual Go packages. Prevents stale configs
      after package renames/deletes.
      _(Effort: S)_
- [ ] **Add meta-test: every module with 3+ production packages has a
      `.go-arch-lint.yml`** — prevents the intra-module enforcement gap from
      recurring as new modules are added.
      _(Effort: S)_

---

## System Package

> Lifecycle edge-case tests + DuckDB/Postgres/ShutdownDependency integration tests
> completed 2026-08-09. See
> `docs/status/2026-08-09_00-49_system-test-coverage-expansion.md`.

- [ ] 🔥 **Run Postgres integration test against live PG** — test compiles and
      skips without DSN, but has never been run against a real Postgres. Verify
      via `nix run .#integration-pg -- go test -run TestIntegration_PostgresSource ./system/...`.
      _(Effort: S)_
- [ ] 🔥 **Fix ShutdownOrder naming gap** — `ShutdownOrder()` returns
      `Profile().Name` (e.g., `"memory"`) but `ShutdownDependency.Before`/`After`
      reference config keys (e.g., `"event-store"`). Either document the
      discrepancy or change `ShutdownOrder()` to return config keys.
      _(Effort: S for doc, M for code change)_
- [ ] **Add per-test database isolation for Postgres integration test** —
      parallel tests sharing one DSN will collide on table names. Wire in the
      `pgtestcontainer` per-test-database pattern.
      _(Effort: M)_
- [ ] **Add `TestSystem_GracefulClose_DrainError_NoClose`** — verify `Close()`
      is NOT called when a drainer fails (resources may leak).
      _(Effort: S)_
- [ ] **Consolidate driver registration into a `TestMain`** — each integration
      test file registers drivers in `init()` (pebble, duckdb, postgres). A
      shared `TestMain` avoids silent last-wins conflicts on the global driver
      map.
      _(Effort: S)_
- [ ] **Consider moving CGo DuckDB test to a sub-module** — `duckdbengine` adds
      ~20 indirect deps to `system/go.mod` (Arrow, FlatBuffers, 6 platform
      DuckDB binding packages). A `system/integration/` sub-module follows the
      `testutil/pgtestcontainer` precedent and keeps the system module lean.
      _(Effort: M)_
- [ ] **Add Badger/bbolt source-of-truth integration tests** — DuckDB and
      Postgres now covered; Badger and bbolt implement StreamLogBackend but
      have no system-level integration test.
      _(Effort: M)_
- [ ] **Add concurrent Close/GracefulClose race tests** — concurrent `Close()`
      calls from multiple goroutines, concurrent `RegisterCloser` + `Close`.
      _(Effort: S)_

---

## Metaengine Coverage Gaps

- [ ] **Cross-engine parity test for all 5 aggregate interfaces** — like
      `adttest.RunMatrix` but for aggregate pushdown (AggregateReader/
      GroupedAggregateReader/MultiAggregateReader/MultiGroupedAggregateReader/
      ExplainableAggregate).
      _(Effort: M)_
- [ ] **Run full DuckDB test suite under `-race`** — only race regression test
      run so far; full suite (~5 min) never verified.
      _(Effort: S)_
- [ ] **Add aggregate tests with NULL values + large datasets (10K+ rows)**.
      _(Effort: S)_
- [ ] **Add SQLite engine Doctor test** (real engine, not fake).
      _(Effort: S)_
- [ ] **Run calibration benchmarks against baseline** — verify
      `calibration-baseline.md` accuracy; add CI regression check.
      _(Effort: M)_

---

## Documentation

- [ ] **Add ADR for ApplyLayoutPlan post-construction registration pattern**.
      _(Effort: S)_
- [ ] **Add ADR for WithClock pattern** (injectable time for CRDT testing).
      _(Effort: S)_
- [ ] **Document GitHub Actions SHA pinning policy** in CONTRIBUTING.md.
      _(Effort: S)_
- [ ] **Write cqrs-lint v4.6.0 release notes** (202 rules, 10 new rules,
      resilience/observability/correctness categories).
      _(Effort: S)_

---

## Declined / Rejected (do not re-litigate)

> Full rationale in the linked ADRs/reviews.

- **Wire `#verify-parallel` into CI** — declined 2026-07-29. CI already has a
  per-module matrix strategy that provides better isolation.
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Use
  `RelationalProjection` (junction tables). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
- **Redis adapter** — the author is not a fan of Redis. See ROADMAP Non-Goals.
- **Rewrite `check-module-layers.sh` as Go NOW** — deferred. The script is stable
  (348 lines). Revisit when complexity grows significantly.
- **Fix LogBackend same-nanosecond collision** — `time.Now().UnixNano()` could
  theoretically collide under extreme concurrency (1-in-a-billion). The
  performance cost of an atomic counter per collection is not worth the
  theoretical correctness gain. Accepted tradeoff: a counter may be off by 1.
