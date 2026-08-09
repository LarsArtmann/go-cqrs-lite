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
- [ ] **Fix CounterIncrement over-read** — queries ALL counters in collection
      even for 1-2 key deltas. Query only delta keys instead.
      _(Effort: M)_
- [ ] **Write dedicated unit tests for MultimapBackend** — empty key,
      limit=0, concurrent append ordering (currently only tested via
      `adttest.RunMatrix`).
      _(Effort: S)_
- [ ] **Write dedicated unit tests for LogBackend** — empty collection,
      limit > entries (the same-nanosecond collision case is a declined
      tradeoff — see Declined section).
      _(Effort: S)_

---

## cqrs-lint — Consumer Feedback (2026-08-04 → 2026-08-09)

> Concrete detector improvements requested by real consumers (KeyHolderAI,
> cqrs-htmx, DiscordSync, file-renamer). See `docs/feedback/new/` for full
> context. Each is a verified false positive or missing detection in a real
> consumer codebase.

- [ ] **Broaden `server` feature detection** — recognize `http.Server{}`
      struct literals, `.ListenAndServe()` calls, Gin's `engine.Run()`.
      (KeyHolderAI, DiscordSync)
      _(Effort: M)_
- [ ] **P012/P013 DSN-level pragma detection** — scan for
      `_pragma=journal_mode(WAL)` and `_pragma=busy_timeout(N)` in the
      `sql.Open` DSN string (modernc.org/sqlite users). (DiscordSync)
      _(Effort: M)_
- [ ] **Per-module feature profiles** — when analyzing a multi-`go.mod`
      workspace, detect features per-module and apply each module's profile
      only to its own packages. (cqrs-htmx)
      _(Effort: L)_
- [ ] **C034 context-derivation tracing** — recognize
      `context.WithCancel(ctx)` → variable → `<-variable.Done()` as satisfying
      the rule. (DiscordSync)
      _(Effort: M)_

---

## Code Quality / Dedup

> Dedup session 2026-08-09: 11→3 clone groups at threshold 4. 8 fixed, 3
> accepted. 5 test helpers + 2 production helpers extracted. Verify gate
> NOT run (individual module tests only). See
> `docs/status/2026-08-09_01-02_dedup-threshold-4-cleanup.md`.

- [ ] **Extract `DistinctValues` row-scan into shared SQL helper** — 23-line
      loop duplicated between `duckdbengine/aggregations.go` and
      `sqliteengine/aggregations_grouped.go`. `storage/sql/` exists for this
      purpose but both engine modules are Tier 4 (importing it inverts the
      dependency). Consider a new `metaengine/sqlutil/` (Tier 0/1) instead.
      _(Effort: M)_
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
- [ ] **Add CI check comparing `go.mod` requires vs depguard allow list** —
      dependencies are only added to `.golangci.yml` after lint fails.
      _(Effort: M)_
- [ ] **Investigate `gci` vs `goimports` disagreement** —
      `pgengine/testcontainer_test.go` has `gci` issues that `nix fmt`
      doesn't fix.
      _(Effort: S)_
- [ ] **Document `testModules` ↔ `lintModules` coupling in AGENTS.md** —
      they share the same list; adding a module requires updating both.
      _(Effort: S)_

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must
  v0.1.2).
- [ ] 🔥 **Cut CHANGELOG `[Unreleased]` → `[v4.7.0]`** — `TestTagContentMatchesChangelog`
      requires ≥1 git tag at the version. Needs ≥10 coordinated module tags via
      `scripts/tag-release.sh` first. Attempted + reverted 2026-08-09 (zero tags
      at v4.7.0). The `[Unreleased]` section is ~4800 lines.
      _(Effort: M — run tag-release.sh for changed modules, then cut)_

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

## Storage Backends

- [ ] **bbolt `ReadStreamFrom` O(N) linear scan** — journal keys are
      `{unixnano}:{eventID}`, so `ReadStreamFrom` scans from journal start to
      find the skip target. Pebble can Seek directly. A secondary index keyed
      by eventID would allow O(log N) Seek. Low priority: bbolt is single-writer
      and write-amplification of a secondary index may not be worth it.
      _(Effort: M)_

---

## System Package

> Lifecycle edge-case tests + DuckDB/Postgres/ShutdownDependency integration tests
> completed 2026-08-09. See
> `docs/status/2026-08-09_00-49_system-test-coverage-expansion.md`.

- [ ] 🔥 **Run Postgres integration test against live PG** — test compiles and
      skips without DSN, but has never been run against a real Postgres. Verify
      via `nix run .#integration-pg -- go test -run TestIntegration_PostgresSource ./system/...`.
      _(Effort: S)_
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

- [ ] **ADR-0117 command lifecycle implementation** — DLQ as event streams,
      retries as event streams (no status fields). Design complete in ADR;
      no code yet.
      _(Effort: L)_
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

- [ ] **Update `example/taskmanager/metaengine.go` to showcase new DX helpers**
      — the canonical reference consumers copy from still uses manual
      `eventWithID`/`taskEventDecoder` patterns (49 references). Should use
      `projectionadapter.Register` + `NewTypeDecoder`.
      _(Effort: M)_
- [ ] **Document `WithoutViewAutoMigrate`, `AutoMapper` as default, `Increment`
      non-clamping philosophy** in view-store README. (DiscordSync feedback)
      _(Effort: S)_
- [ ] **Add ADR for WithClock pattern** (injectable time for CRDT testing).
      _(Effort: S)_
- [ ] **Document GitHub Actions SHA pinning policy** in CONTRIBUTING.md.
      _(Effort: S)_
- [ ] **Write cqrs-lint v4.6.0 release notes** (202 rules, 10 new rules,
      resilience/observability/correctness categories).
      _(Effort: S)_

---

## v5 Unification

> Decision: [ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md).
> Vision: developers declare only Commands + Events + Queries; the system
> infers projections, storage layout, and engine routing. Operators pick
> infrastructure at deployment time. No dual paths, no escape hatches.
>
> Dependency chain — tasks are listed in execution order. Each phase gates the
> next. See ADR-0123 for the full rationale.

### Phase 1: Type Foundation

- [ ] 🔥 **Finish Record consolidation (ADR-0111 Phases 3-4)** — consolidate
      `event.Metadata`, `command.Metadata`, `metadata.Tracing` into
      `record.CommonMetadata`. Record becomes the single structural base for
      events + commands. Delete duplicate metadata types.
      _(Effort: L)_

### Phase 2: Quick Wins (dead code removal)

- [ ] **Delete `metaengine.GraphBackend` (ADR-0113)** — remove the interface
      (`engine.go:394`), remove implementations from all engines (memory engine
      assertion at `engine.go:560`). Graph operations route through
      `graphadapter` exclusively. Update `adttest.RunMatrix` to use graphadapter.
      _(Effort: M)_
- [ ] **Replace `simpleBus` with `watermill.EventBus` in `system/`** — delete
      `system/bus.go` (simpleBus), delete `BusDriverFactory` +
      `RegisterBusDriver` from `system/driver_registry.go`. Wire
      `watermill.EventBus` as the bus in `system/constructor.go:179-185`.
      Map `BusConfig.Driver` to watermill backend selection.
      _(Effort: M)_

### Phase 3: Self-Registration Infrastructure

- [ ] 🔥 **Move driver registry to `metaengine/`** — relocate `RegisterDriver`,
      `DriverFactory`, `EngineConfig`, `lookupDriver`, `createEngineFromDriver`
      from `system/driver_registry.go` to a new `metaengine/registry.go`.
      `system/` calls `metaengine.LookupDriver(name)` instead of its own map.
      All 9 engines already depend on `metaengine/`, so no new deps.
      _(Effort: M)_
- [ ] **Convert memory + sqlite to self-registration** — move their
      `RegisterDriver` calls from `system/init()` to their own packages.
      `metaengine/memory_engine.go` gets `register.go`; `sqliteengine/` gets
      `register.go`. Verify `system.New()` still works via blank imports in
      tests/examples.
      _(Effort: S)_

### Phase 4: Backend Porting (all 8)

- [ ] 🔥 **Port pebble driver** — `metaengine/pebbleengine/register.go` with
      `RegisterDriver("pebble", ...)`. Map `cfg.DSN` to directory path. Handle
      in-memory (`vfs.NewMem`) when DSN is empty. Verify through system tests.
      _(Effort: S)_
- [ ] **Port bbolt driver** — new `metaengine/bboltengine/` module (or extend
      existing `storage/bbolt` as a metaengine engine). Self-register as
      `"bbolt"`. Map DSN to file path.
      _(Effort: M)_
- [ ] **Port postgres driver** — `metaengine/pgengine/register.go` with
      `RegisterDriver("postgres", ...)`. Map `cfg.DSN` to pgx connection string.
      Handle pool config from `cfg.Pragmas` or new `EngineConfig` fields.
      _(Effort: S)_
- [ ] **Port duckdb driver** — `metaengine/duckdbengine/register.go` with
      `RegisterDriver("duckdb", ...)`. CGo isolation preserved (separate module).
      Map DSN to file path or `:memory:`.
      _(Effort: S)_
- [ ] **Port mysql driver** — new `metaengine/mysqlengine/` module or extend
      `pgengine` with MySQL dialect. Self-register as `"mysql"`.
      _(Effort: M)_
- [ ] **Port turso driver** — `metaengine/tursoengine/` or extend with libSQL.
      Self-register as `"turso"`. Handle sync config.
      _(Effort: M)_
- [ ] **Port badger driver** — `metaengine/badgerengine/register.go`.
      Self-register as `"badger"`.
      _(Effort: S)_
- [ ] **Port dgraph driver** — `metaengine/dgraphengine/register.go`.
      Self-register as `"dgraph"`.
      _(Effort: S)_

### Phase 5: Record-Typed Default Folds

- [ ] 🔥 **Make `OnRecord` the default fold constructor** — change examples,
      docs, and auto-projection to use `OnRecord`/`OnRecordTyped` instead of
      `On`. Fold handlers receive `record.Record` as the first parameter.
      Deprecate payload-only `On` (mark deprecated, remove in v5 cut).
      _(Effort: M)_

### Phase 6: Auto-Projection (the killer feature)

- [ ] 🔥🔥 **Planner-time fold inference (ADR-0116 Layer 1)** — the planner
      inspects event and query struct shapes at `Plan()` time and synthesizes
      folds automatically. Field-name matching (`event.ID` → result `ID`,
      `event.Status` → filter field `status`). Struct composition (nested
      structs, slices → separate collections). Convention detection
      (Created/Updated/Deleted suffixes → insert/update/delete folds). Consumer
      declares zero folds for the 80% case.
      _(Effort: XL)_
- [ ] **Struct-composition-driven multi-collection** — when an event has a
      `[]Attachment` field and a query requests `MessageView` (which has
      `Attachments`), auto-generate a second collection for attachments.
      Planners sees the relationship and generates a join-aware read path.
      _(Effort: L)_
- [ ] **Fold inference override API** — when auto-projection gets it wrong,
      consumer can override with an explicit `OnRecord` fold for a specific
      event/query pair. Override replaces (not supplements) the generated fold.
      _(Effort: M)_

### Phase 7: Universal Engine Coverage

- [ ] 🔥 **Multi-collection batch atomicity** — when one event triggers folds
      for multiple collections, all writes commit atomically in one engine
      transaction. Modify `store.ApplyRecord` to batch all fold operations and
      execute them in a single engine transaction. The batch boundary is the
      event, not the collection. Replaces RelationalProjection's per-event tx.
      _(Effort: L)_
- [ ] **Universal ADT coverage per engine** — audit each engine for missing
      ADT backends. Add degraded fallbacks where native support is impossible:
      graph traversal via recursive CTE on SQLite/PG/MySQL; brute-force vector
      search on Memory/Pebble; StreamLog on Dgraph (append-ordered nodes).
      Every engine must handle every ADT.
      _(Effort: XL)_
- [ ] **Capability-degradation planner rule** — new `PlanRule` that emits
      WARN/INFO when an ADT is routed to an engine whose `EngineProfile`
      declares that ADT as degraded. Shows estimated cost penalty + recommends
      a better engine if one is available. Integrates into `ExplainPlan()` and
      `Doctor()`.
      _(Effort: M)_

### Phase 8: Deletion + v5 Cut

- [ ] **Delete `stack.Materialize`** — auto-projection replaces it.
      _(Effort: S)_
- [ ] **Delete `storage.RelationalProjection` + `storage/view` (SQLViewStore)**
      — multi-collection batch atomicity + auto-projection replaces them.
      Absorb `ProjectionSink` operations as engine internals.
      _(Effort: M)_
- [ ] **Delete `graph.GraphProjection`** — auto-projection + graphadapter
      replaces it.
      _(Effort: S)_
- [ ] **Delete `stack.Bundle` + all 8 stack presets** — `system.System` is the
      only composition root. `stack/` module deleted entirely.
      _(Effort: M)_
- [ ] **Delete `stack.RunProjections`** — `projectionhost.Host` is the only
      projection runner.
      _(Effort: S)_
- [ ] **Write v5 migration guide** — document the path from v4 (stack presets,
      v1 tiers) to v5 (system.System, auto-projection). Include before/after
      examples for each v1 tier.
      _(Effort: L)_
- [ ] **Cut v5.0.0** — tag all modules. Update CHANGELOG, README, SKILL.md,
      examples. Run full verify gate.
      _(Effort: M)_

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
