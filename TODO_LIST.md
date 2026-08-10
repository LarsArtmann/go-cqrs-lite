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
>
> Pareto session 3 (2026-08-09): 16 of 27 tasks done. All 9 FP regression tests
> verified to exist. Taskmanager integration test (31-finding golden) added.
> E003 map-iteration non-determinism fixed (`slices.Sort`). Library-framework
> preset wired. `http.Server{}` literal detection + DSN pragma detection added.
> Dgraph VM test + retry + pool tuning + CounterIncrement filter fix +
> Multimap/Log edge tests added. Aggregate NULL/large-dataset tests + SQLite
> Doctor test added. 4 `.go-arch-lint.yml` configs added. SHA pinning policy +
> view-store README + ADR-0122 + cqrs-lint v4.6.0 release notes documented.
> See `docs/status/2026-08-09_07-25_pareto-execution-session-2-report.md`.

---

## cqrs-lint — Consumer Feedback (2026-08-04 → 2026-08-09)

> Concrete detector improvements requested by real consumers (KeyHolderAI,
> cqrs-htmx, DiscordSync, file-renamer). See `docs/feedback/new/` for full
> context. Each is a verified false positive or missing detection in a real
> consumer codebase.

- [ ] **Per-module feature profiles** — when analyzing a multi-`go.mod`
      workspace, detect features per-module and apply each module's profile
      only to its own packages. (cqrs-htmx)
      _(Effort: L)_

---

## Code Quality / Dedup

> Dedup session 2026-08-09: 11→3 clone groups at threshold 4. 8 fixed, 3
> accepted. 5 test helpers + 2 production helpers extracted. Verify gate
> NOT run (individual module tests only). See
> `docs/status/2026-08-09_01-02_dedup-threshold-4-cleanup.md`.
>
> Pareto session 3: `DistinctValues` already shared via
> `metaengine.ScanDistinctValues()` (4 call sites). `deferClose` already
> shared via `metaengine.DeferClose()` (47 prod + 17 test sites). AGENTS.md
> dedup section + testModules coupling documented.

- [x] **Extract bbolt/pebble backup lifecycle test suite** — DONE 2026-08-10.
      New `storage/backuptest/v4` module with `Backend` interface, `Factory`
      struct, `RunFullLifecycle()`, `RunIncrementalCheckpoints()`. bbolt: 255→75
      lines. pebble: 235→59 lines. Both backends now use thin adapters. Dedup
      baseline updated; `art-dupl check` passes with 0 new clones.
      See `docs/status/2026-08-10_14-20_backuptest-extraction-and-pebbleengine-scan.md`.
- [x] **Scan remaining engine modules for setup boilerplate** — DONE 2026-08-10.
      pebbleengine: 18 of 23 test files now use helpers. The 4 remaining files
      (`format_index_test.go`, `nextkey_test.go`, `disk_backed_test.go`,
      `restart_safety_test.go`) cannot use helpers by design (pure functions or
      custom close/reopen lifecycle). All applicable files are refactored.
      See `docs/status/2026-08-10_14-20_backuptest-extraction-and-pebbleengine-scan.md`.
- [ ] **Audit `.golangci.yml` exclusion blocks** — `system/` (20 linters
      disabled), `cmd/cqrs-lint/` (17), `metaengine/` (21) have the broadest
      exclusions. Audit done 2026-08-09: `staticcheck` disabled for `system/`
      is the most concerning (correctness linter). Test shows no violations
      when enabled, but full lint gate verification needed before removing.
      Narrow where safe.
      _(Effort: M — needs full lint run to verify narrowing)_
- [BLOCKED] **Wire backuptest into bbolt/pebble go.mod for GOWORK=off** —
      The new `storage/backuptest/v4` module works in workspace mode but
      bbolt and pebble `go.mod` lack the `require` directive + go.sum entries.
      `nix run .#test` / `nix run .#build` (GOWORK=off) will FAIL. Requires:
      (1) tag `storage/backuptest/v4.0.0` (annotated, not lightweight),
      (2) add `require backuptest/v4 v4.0.0` to both go.mod files,
      (3) `go mod tidy` with GOWORK=off. Also: delete the lightweight tag
      created via `git update-ref` during development.
- [ ] **Run `nix run .#verify` after backuptest wiring** — AGENTS.md mandates
      verify gate before claiming GREEN. Not yet run this session.
      Includes: build + vet + test + race + lint + doc-check + check-arch +
      check-depguard + check-coverage + check-duplication.
- [ ] **Register `storage/backuptest` in docs and configs** — Not yet added to:
      AGENTS.md Module Map table, AGENTS.md Module Tiers, `.golangci.yml`
      depguard allow list, `.agents/skills/go-cqrs-lite/references/modules.md`,
      `docs/architecture-understanding/SEVEN-TIER-MODEL.md`.
- [ ] **Reduce `.art-dupl-baseline.json` diff noise** — `art-dupl baseline`
      reformatted the entire file from compact to pretty-printed JSON,
      causing a 400+ line diff. Investigate compact output or a JSON formatter.
---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must
  v0.1.2).
- [x] 🔥 **Cut CHANGELOG `[Unreleased]` → `[v4.7.0]`** — DONE 2026-08-10.
      `TestTagContentMatchesChangelog` passes (metaengine/v4.7.0 tag exists).
      [Unreleased] section moved to [v4.7.0]. Additional module tags can be
      cut as needed for consumer visibility.

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

---

## Layer Enforcement / Architecture

> 11 `.go-arch-lint.yml` configs exist (root + catalog, cmd/cqrs-lint, command,
> decider, event, kv, metaengine, middleware, projectionhost, signing, stack,
> storage). Meta-tests in `cmd/api-stability/main_test.go` verify all configs
> are valid and that every module with 3+ production packages has a config.

---

## Storage Backends

---

## System Package

> Lifecycle edge-case tests + DuckDB/Postgres/ShutdownDependency integration tests
> completed 2026-08-09. See
> `docs/status/2026-08-09_00-49_system-test-coverage-expansion.md`.
>
> Pareto session 3: PG integration test PASS (0.01s). GracefulClose
> drain-error-no-close + concurrent Close race tests added (both PASS under
> `-race`).

- [ ] **Add per-test database isolation for Postgres integration test** —
      parallel tests sharing one DSN will collide on table names. Wire in the
      `pgtestcontainer` per-test-database pattern. Currently only one PG test
      exists so this is not urgent.
      _(Effort: M)_
- [ ] **Consolidate driver registration into a `TestMain`** — each integration
      test file registers drivers in `init()` (pebble, duckdb, postgres,
      badger). No conflicts exist (unique names), but a shared `TestMain`
      would prevent future accidental duplicates. Complicated by CGo build
      tag on the DuckDB driver.
- [ ] **Consider moving CGo DuckDB test to a sub-module** — `duckdbengine` adds
      ~20 indirect deps to `system/go.mod` (Arrow, FlatBuffers, 6 platform
      DuckDB binding packages). A `system/integration/` sub-module follows the
      `testutil/pgtestcontainer` precedent and keeps the system module lean.
      _(Effort: M)_
- [ ] **Add bbolt source-of-truth integration test** — bbolt needs a
      `metaengine/bboltengine/` module first (v5 Phase 4 dependency). Badger
      and Pebble already covered.

---

## Metaengine Coverage Gaps

- [ ] **ADR-0117 command lifecycle implementation** — DLQ as event streams,
      retries as event streams (no status fields). Design complete in ADR;
      no code yet.
      _(Effort: L)_
- [ ] **Run calibration benchmarks against baseline** — verify
      `calibration-baseline.md` accuracy; add CI regression check.
      _(Effort: M)_

---

## Live Cost Measurement (dynamic NetworkRTT / per-op latency)

> Design: `docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md` — the critique that
> `NetworkRTT` is a compile-time constant on `EngineProfile` while real latency
> is a runtime observation. ADR-0093 defers the fix; the ReadCosts doc labels
> per-op costs "compile-time, do not evolve at runtime." This section tracks
> the phased remediation. Long-term vision: ROADMAP → Themes.

- [ ] 🔥 **P1: Prober + LatencyTracker** — optional `Prober`/`TransactMeasurer`
      interfaces, `LatencyTracker` (window + EWMA + percentiles), `ProbeEngine()`
      helper, `CalibrationCosts` gains `NetworkRTT` + measured read/write fields,
      live `Profile()` composition. Test-double engine proves a live RTT shift
      changes `Profile()`. Wire PG (`SELECT 1`) + Dgraph (healthcheck) probes.
      _(Effort: M)_
- [ ] **P2: Live planner view + Store stats** — Store keeps runtime profile
      snapshot; refresh on plan / `GetStats()` / background interval;
      `EngineStats` {profile, measured RTT, samples, lastProbe, stale};
      `EXPLAIN` shows `rtt=live … (p95, n)`; WARN diagnostic when routing on
      stale/prior RTT; optional live re-scoring of near-tied queries with
      hysteresis. `WithNetworkRTT` documented as a prior, not a constant.
      _(Effort: M)_
- [ ] **P3: Open measurement ingress for external engines** — exported
      `StatSink` so external engines (future fdbengine, pgengine, dgraphengine)
      push live measurements without a hard core dependency. Test: fake prober
      drives planner decisions with/without live stats.
      _(Effort: M — independent of P1/P2)_
- [ ] **Wire live latency into `GetStats`/Doctor UX** — Doctor + EXPLAIN show
      measured RTT per remote engine with freshness (samples, last probe) and
      stale labeling.
      _(Effort: S — P2 dependent)_

---

## Documentation

> taskmanager metaengine.go already uses `Register` + `NewTypeDecoder` DX helpers
> (verified 2026-08-09 — zero references to old `eventWithID`/`taskEventDecoder` patterns).

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
