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
>
> Verify-green session 2026-08-11: 147 lint issues → 0, arch script fixed,
> `nix run .#verify` GREEN. See
> `docs/status/2026-08-11_04-04_verify-green-and-lint-cleanup.md`.

- [ ] **Audit `.golangci.yml` exclusion blocks** — `system/` (20 linters
      disabled), `cmd/cqrs-lint/` (17), `metaengine/` (24) have the broadest
      exclusions. Several new exclusions added 2026-08-11 for the
      `On/OnTyped` migration, `flightrecorder/`, `id/`, `record/`, engine
      `register.go`. Track which can be removed after migrations complete.
      _(Effort: M — needs full lint run to verify narrowing)_
- [ ] **Infrastructure polish (nix apps + shared helpers)** — add
      `#check-lint-config` (validate `.golangci.yml` + excluded paths exist),
      `#verify-ci` (mirror GH Actions GOWORK=off per-module), wire `#sweep`
      to pre-commit/cron, consolidate engine `register.go` boilerplate (7
      modules), audit/trim indirect deps in `metaengine/go.mod`
      (modernc sqlite chain), add property-based tests for `metadataPayload`
      CBOR roundtrip, extract `metadataPayload` to `storage/serialization/`
      if a 3rd KV engine is added. See plan
      `docs/planning/2026-08-11_04-12_pareto-comprehensive-plan.html` (M27).
      _(Effort: M)_

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must
  v0.1.2).

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

## Metaengine Coverage Gaps

> ADR-0117 command lifecycle implemented 2026-08-11. See
> `docs/status/2026-08-11_07-07_adr-0117-command-lifecycle.md`.

### ADR-0117 Follow-ups (from status report)

> Recorder version tracking, integration tests, projection query tests,
> processing-time projection, system wiring, lifecycle recipe, and verify gate
> all completed 2026-08-11 by `86458d36e`.

- [ ] **Tag `commandlifecycle/v4.0.0`** — publish the two new modules after
      version tracking fix and verify gate pass.
      _(Effort: S)_
      **BLOCKED (2026-08-11):** id.ActorID release gap. Published
      `record/v4.1.0`, `command/v4.4.0`, `metaengine/v4.8.0` reference
      `id.ActorID`, but newest `id/v4` tag is v4.2.0 (no ActorID) → consumer
      GOWORK=off builds fail. Fix: tag `id/v4.3.0` → re-tag record/v4.2.0,
      command/v4.5.0, metaengine/v4.9.0 → then commandlifecycle. See
      `docs/status/2026-08-11_17-05_actorid-release-gap-blocks-commandlifecycle.md`.
- [x] **Tag `benchkit/v4.4.0`** — already tagged (7d5cd10c7), contains
      `Truncate`/`TitleCase`. Closed 2026-08-11.
      _(Effort: XS)_

- [ ] **Run calibration benchmarks against baseline** — verify
      `calibration-baseline.md` accuracy; add CI regression check.
      _(Effort: M)_

---

## v5 Unification

> Decision: [ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md).
> Vision: developers declare only Commands + Events + Queries; the system
> infers projections, storage layout, and engine routing. Operators pick
> infrastructure at deployment time. No dual paths, no escape hatches.
>
> Dependency chain — tasks are listed in execution order. Each phase gates the
> next. See ADR-0123 for the full rationale.

### Phases 1–4: Type Foundation, Dead-Code Removal, Self-Registration, Backend Porting

DONE 2026-08-10/11. See [CHANGELOG.md](CHANGELOG.md) and status reports
`docs/status/2026-08-10_*.md` for details.

### Phase 5: Record-Typed Default Folds

- [x] 🔥 **Make `OnRecord` the default fold constructor** — change examples,
      docs, and auto-projection to use `OnRecord`/`OnRecordTyped` instead of
      `On`. Fold handlers receive `record.Record` as the first parameter.
      Deprecate payload-only `On` (mark deprecated, remove in v5 cut).
      _(Effort: M)_ — DONE 2026-08-11. All metaengine tests, cqrs-lint
      detectors/fixtures, examples, and living docs migrated. `On`/`OnTyped`
      carry `Deprecated:` godoc and will be removed in the v5 cut.

### Phase 6: Auto-Projection (the killer feature)

> Planner-time fold inference (ADR-0116 Layer 1) implemented 2026-08-11.
> `metaengine.Infer(samples...)` is code-complete but not verify-gated.
> See `docs/status/2026-08-11_05-09_fold-inference-adr0116-layer1-status.md`.

- [ ] 🔥 **Run `nix run .#verify` for fold inference** — fix `nix fmt`, file
      line-count violations (`query.go` > 350 limit), lint, arch, dedup,
      coverage, race.
      _(Effort: M)_
- [ ] **Fold inference override API** — when auto-projection gets it wrong,
      consumer can override with an explicit `OnRecord` fold for a specific
      event/query pair. Override replaces (not supplements) the generated fold.
      Without this, `Infer()` is all-or-nothing.
      _(Effort: M)_
- [ ] **Fold inference gaps** — `[]Struct` fields in event types,
      `InferFromNamedEvents()` for wire event types, sort inference,
      composite keys, filter operators beyond `FilterEq`.
      _(Effort: M)_

### Phase 6b: Operator-Driven Layout Planning (replaces M9)

> Core implemented 2026-08-11 (priority system, embed-vs-normalize scoring,
> `ReplanLayout`, `ConfirmRebuild`, runtime backend add/remove). See status
> reports `2026-08-11_07-23_layout-planning-implementation-comprehensive-status.md`
> and `2026-08-11_08-20_layout-planning-followups-safe-backfill-real-rebuilds.md`.
>
> Original M9 was wrong: storage layout is the operator's call, not the
> developer's. See
> [`docs/planning/METAENGINE-LAYOUT-PLANNING-MODEL.md`](docs/planning/METAENGINE-LAYOUT-PLANNING-MODEL.md).

- [ ] 🔥 **Run `nix run .#verify` clean for layout planning** — `nix fmt`,
      file line-count limits (`explain.go`), lint, arch, dedup, coverage, race.
      _(Effort: M)_
- [ ] 🔥 **`cqrs-bench layout` CLI subcommand** — pre-deployment "what if"
      exploration tool.
      _(Effort: M)_
- [ ] 🔥 **Calibrate cost model multipliers** — replace placeholder constants
      (0.5, 1.0, 1.3, 2.0) with measured values from real engine benchmarks.
      _(Effort: L)_
- [ ] **Fold-pipeline sync for Active+DualUse roles** — event → all
      Active+DualUse projections in one transaction (strong consistency).
      _(Effort: L)_
- [ ] **Async replication for Backup+Migration roles** — eventual consistency,
      failure-isolated.
      _(Effort: L)_
- [ ] **Role transition API** — Backup→Active promote, Migration→Active cutover.
      _(Effort: M)_
- [ ] **Multi-engine integration test with two real backends** — current test
      uses one MemoryEngine + Backfill replay. Need two engines with data,
      verify both serve correct query results after AddEngine + Backfill.
      _(Effort: M)_
- [ ] **Add e2e Store integration test for graph fallback** — graph_fallback_test.go
      tests helper functions directly; needs a full Store.Apply → Store.Execute
      test through a non-graph engine (e.g., SQLite/Pebble).
      _(Effort: S)_
- [ ] **Integrate `ReplanLayout` with `Store.Replan`/`CheckRouting`** —
      `SetPriority` calls `Replan` but layout diffs require separate
      `ReplanLayout` call. These should converge into one planning pass.
      _(Effort: M)_
- [ ] **Real workload trace format** — JSON-lines spec, trace recorder, trace
      player for benchmark calibration.
      _(Effort: M)_
- [x] **Wire `Priority` into deployment YAML** — `EngineConfig`/`DriverConfig`
      + `QueryDecl` builder options + config validation.
      _(Effort: M)_
- [ ] **Aggregate boundary config** — `WithSharedCollection("Attachment")`
      opt-in for shared-by-type collections.
      _(Effort: M)_
- [ ] **Layout audit trail** — plan version history, who changed what priority
      when, in `GetEngineStats()`.
      _(Effort: S)_
- [ ] **Update SKILL.md + skill references** — layout planning concepts,
      priority system, benchmark mode consumer docs.
      _(Effort: S)_
- [x] **Tag `benchkit/v4.4.0`** — `Truncate`/`TitleCase` added after v4.3.0 tag.
      `cmd/cqrs-bench` fails under GOWORK=off without it. Already tagged
      (7d5cd10c7). Closed 2026-08-11.
      _(Effort: XS)_
- [ ] **Document commandlifecycle in skill references** — modules.md, recipes.md,
      advanced.md do not mention commandlifecycle/. Only AGENTS.md has it.
      _(Effort: S)_
- [ ] **Test pgtestcontainer external DSN isolation** — M18 change is untested;
      verify per-test database creation with actual external Postgres.
      _(Effort: S)_
- [ ] **Consider per-fold mutex instead of global foldMu** — current foldMu
      serializes all fold execution; per-fold would allow parallel writes across
      different queries.
      _(Effort: M)_

### Phase 7: Universal Engine Coverage

- [ ] 🔥 **Multi-collection batch atomicity** — when one event triggers folds
      for multiple collections (e.g. parent + normalized child from Phase 6b
      layout planning), all writes commit atomically in one engine transaction.
      Modify `store.ApplyRecord` to batch all fold operations and execute them in
      a single engine transaction. The batch boundary is the event, not the
      collection. Replaces RelationalProjection's per-event tx. See
      [`METAENGINE-LAYOUT-PLANNING-MODEL.md`](docs/planning/METAENGINE-LAYOUT-PLANNING-MODEL.md) §6 (dual-use sync).
      _(Effort: L)_
- [ ] 🔥 **Universal ADT coverage per engine** — audit each engine for missing
      ADT backends. Add degraded fallbacks where native support is impossible:
      graph traversal via recursive CTE on SQLite/PG/MySQL; brute-force vector
      search on Memory/Pebble; StreamLog on Dgraph (append-ordered nodes).
      Every engine must handle every ADT.
      **Partial progress 2026-08-11:** Graph fallback via MultimapBackend BFS
      implemented (`graph_fallback.go`). ADTGraph added (degraded) to SQLite
      + MySQL profiles. 4 unit tests. Still missing: StreamLog on Dgraph,
      recursive CTE optimization, e2e Store integration test.
      _(Effort: XL)_
- [ ] **Capability-degradation planner rule** — new `PlanRule` that emits
      WARN/INFO when an ADT is routed to an engine whose `EngineProfile`
      declares that ADT as degraded. Shows estimated cost penalty + recommends
      a better engine if one is available. Integrates into `ExplainPlan()` and
      `Doctor()`.
      _(Effort: M)_
- [ ] **Engine test parity gaps (remaining)** — mysqlengine lacks
      `stream_log_test.go`, `pushdown_test.go`, `calibration_bench_test.go`,
      and `explain.go` (`ExplainableScan`/`ExplainableAggregate`); tursoengine
      lacks `record_stamp_test.go`, `soak_autocrud_test.go`,
      `healthcheck_test.go`. bboltengine still lacks `edge_cases_test.go`,
      `fuzz_test.go`, `scan_bench_test.go` (may be pebble-specific).
      _(Effort: M — bbolt partial done, mysql + turso remaining)_
- [ ] **Engine compile-time assertion gaps** — bboltengine missing
      `HealthChecker` and `StreamingScan` assertions; mysqlengine missing
      `Calibratable` and `HealthChecker` assertions.
      _(Effort: S)_
- [x] ✅ **Fix bench fold `reflect.Call` panic** — fixed in commit
      `7ba946377` ("reify prev value in OnRecord update folds"). OnRecord update
      folds now call `reifyReflect(prev, prevType)` to bridge
      `map[string]any` (SQL engine decode) → typed struct before
      `reflect.Call`. All 3 tests pass with `-race`.
      _(Effort: M)_
- [x] ✅ **Pebble `CounterIncrement` calibration benchmark** — done.
      `metaengine/pebbleengine/calibration_bench_test.go` has
      `BenchmarkCalibration_PebbleCounterIncrement` (parity with badger/bbolt).
      _(Effort: S)_
- [x] ✅ **Batch atomicity rollback test** — done.
      `metaengine/batch_atomicity_rollback_test.go` covers failure-path rollback
      semantics.
      _(Effort: S)_
- [x] ✅ **Engine test parity gaps (partial)** — bboltengine `stream_log_test.go`
      + `watcher_test.go` ported from pebble (2 of 5 files). mysqlengine and
      tursoengine gaps remain open. bboltengine `edge_cases_test.go`,
      `fuzz_test.go`, `scan_bench_test.go` still missing (pebble-specific,
      may not port cleanly).
      _(Effort: L — partial, ~40% done)_

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
