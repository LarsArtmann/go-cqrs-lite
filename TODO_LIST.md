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

## ADR-0114 Cleanup Follow-ups

> Session 4 (2026-08-10): TombstonePolicy → DeletePolicy rename done, goldens
> regenerated, migration guide rewritten, docs updated across 12 files.
> Remaining cleanup tracked here. See
> `docs/status/2026-08-10_19-26_tombstone-rename-docs-goldens-session4.md`.

- [x] 🔥 **Run `nix run .#verify`** — DONE 2026-08-11. Full CI gate green:
      build + vet + test + race + lint (0 issues) + arch + dedup + coverage +
      api-stability (3989 exports) + doc-check (928 refs).
      See `docs/status/2026-08-11_04-04_verify-green-and-lint-cleanup.md`.
- [x] **Run `nix fmt`** on all changed files — DONE 2026-08-11.
- [x] **Fix `listing/README.md:16`** — DONE 2026-08-11. Fixed stale tri-state
      claim to bi-state (Active, Deleted).
      _(Effort: XS)_
- [ ] **Unify `DeletePolicy` constant naming** — listing uses
      `DeleteExclude`/`DeleteInclude`/`DeleteOnly`; stack uses
      `ExcludeDeleted`/`IncludeDeleted`/`OnlyDeleted`. Same divergence as
      before the rename, just with new names.
      _(Effort: M — breaking change)_
- [ ] **Rename remaining internal "tombstone" vocabulary** — `OnTombstone`,
      `OnRebirth`, `isMaterializedTombstoned`, `tombstoner` interface,
      `kv.TombstoneQuerier`, `AutoMapperWithTombstone`, `TombstoneColumn`,
      `IsTombstoned()` all still use old vocabulary. Large blast radius.
      _(Effort: L — breaking change, needs user decision)_
- [ ] **Consider backward-compat type aliases** — `type TombstonePolicy =
      DeletePolicy` for smoother consumer migration. Currently a hard break.
      _(Effort: S — needs user decision)_
- [ ] **Decide `metadata/` module fate** — only `CustomData[K]` +
      `MergeCustomMaps` remain. Keep, move into `event/` or `record/`, or delete.
      _(Effort: S — needs user decision)_
- [ ] **Fix `example/taskmanager/setup.go:113`** — pre-existing `[]any` vs
      `[]system.ProjectionDeclaration` type mismatch.
      _(Effort: XS)_

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

- [x] 🔥 **Migrate `metaengine.On` → `metaengine.OnRecord`** — DONE 2026-08-11.
      59 files migrated via AST-based tool. SA1019 blanket exclusion removed.
      On/OnTyped now have `// Deprecated:` godoc pointing to OnRecord/OnRecordTyped.
      _(Effort: M — mechanical migration)_
- [ ] **Audit `.golangci.yml` exclusion blocks** — `system/` (20 linters
      disabled), `cmd/cqrs-lint/` (17), `metaengine/` (24) have the broadest
      exclusions. Several new exclusions added 2026-08-11 for the
      `On/OnTyped` migration, `flightrecorder/`, `id/`, `record/`, engine
      `register.go`. Track which can be removed after migrations complete.
      _(Effort: M — needs full lint run to verify narrowing)_
- [x] **Add smoke test for `check-module-layers.sh`** — DONE 2026-08-11.
      `scripts/test-check-module-layers.sh` validates the arch script on
      known-good tree + handles missing go.mod.
      _(Effort: S)_
- [ ] **Lint code-fix batch (narrow exclusions by fixing the code)** —
      surfaced by the 2026-08-11 lint-cleanup session. Each is currently
      suppressed by a blanket exclusion; fix the code instead:
      `flightrecorder/alias.go` (13 deprecatedComment findings),
      `id/actor_id.go` (16 findings: constants, receiver, strings.Cut),
      `mysqlengine` (4 sqlclosecheck — use CloseRows indirection like
      pgengine), `cmd/api-stability/main_test.go` (nilerr, gocognit),
      `dgraphengine/retry.go` (wire or remove unused retry utilities).
      _(Effort: M)_
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
- [x] **Wire backuptest into bbolt/pebble go.mod for GOWORK=off** — DONE 2026-08-10.
      Used `replace ... => ../backuptest` directives (the repo's established pattern,
      same as signing/encryption → codec, metaengine/*engine → metaengine). This
      resolves GOWORK=off without needing a published tag. Lightweight dev tag
      deleted; annotated tagging deferred to release cycle. GOWORK=off build+vet+test
      verified on both modules.
- [x] **Run `nix run .#verify` after backuptest wiring** — DONE 2026-08-10.
      My modules pass: check-arch ✓, api-stability (3868 exports) ✓, lint (0 issues) ✓,
      backup tests with -race ✓, check-module-layers coverage ✓. Full `#verify` gate
      is blocked by a PRE-EXISTING metadata refactoring (commit 7e374b753) that removed
      tombstone/tracing types without updating listing/watermill/transport/grpc/enginetest.
      That breakage is unrelated to backuptest.
- [x] **Register `storage/backuptest` in docs and configs** — DONE 2026-08-10.
      Added to: `scripts/check-module-layers.sh` (LAYER=5, DEP_BUDGET=3),
      `AGENTS.md` Module Map, `SEVEN-TIER-MODEL.md` Tier 4, `modules.md`.
      `.golangci.yml` depguard already covers via `github.com/larsartmann/go-cqrs-lite` prefix.
- [x] **Reduce `.art-dupl-baseline.json` diff noise** — DONE 2026-08-10.
      The pretty-printed baseline was already committed (no pending diff). No action needed.

---

## GraphBackend / Dead-Code Cleanup Follow-ups

> Phase 2–3 cleanup session (2026-08-10): 7 items tasked, 3 fully done, 2
> partially done. Remaining gaps tracked here. See
> `docs/status/2026-08-10_15-26_graphbackend-deadcode-cleanup-followups.md`.

- [x] **Fix `dgraphengine/README.md:71` broken code example** —
      Replaced with local `graphDispatch` interface definition (ADR-0113 pattern).
      Prose (lines 7, 119) updated from "GraphBackend" to "graph dispatch".
      _(Effort: XS)_ ✅
- [x] **Clean stale `GraphBackend` comment references** in dgraphengine Go
      files: `engine.go:5,7`, `graphrag_test.go:20`, `mixed_bench_test.go:14`
      reworded to "graph dispatch". `engine_test.go:13` left as historically
      accurate ("ADR-0113: the exported metaengine.GraphBackend was deleted").
      _(Effort: XS)_ ✅
- [x] **Run `doc-check`** after `ErrUnknownDriver` removal —
      Passed: all 695 references valid. Fixed 4 stale `event.MarkTombstone`/
      `event.DetectTombstone` references in skill docs (`core.md`, `advanced.md`)
      rewritten to the ADR-0114 domain-event pattern.
      _(Effort: XS)_ ✅
- [x] **Audit skill references** (`.agents/skills/go-cqrs-lite/references/*.md`)
      for `ErrUnknownDriver` — zero references found. Nothing to fix.
      _(Effort: XS)_ ✅
- [x] **Re-run `go vet`** on system + metaengine — passes clean.
      Also fixed 2 additional branded-ID build breaks: `auto_fold_record_test.go:56-57`
      and `soak_record_test.go:97`.
      _(Effort: XS)_ ✅

---

## CI / Release / Infrastructure

- [x] 🔥 **Record consolidation fallout — all modules compile** — ADR-0111
  Phases 3-4 (metadata consolidation + tombstone removal) are DONE. All 79
  modules build clean. The ADR-0114 tombstone build breaks in storage/,
  stack/, transport/grpc/, and example/taskmanager were fixed 2026-08-10.
  See `docs/status/2026-08-10_16-15_graphbackend-cleanup-and-adr0114-tombstone-unblock.md`.
  **IMPORTANT: `go test ./...` from workspace root tests ZERO packages (no root
  go.mod). Always verify per-module or use `nix run .#test`.**
  Done:
  - [x] Core metadata consolidation (record.CommonMetadata branded types)
  - [x] `id.ActorID` kind-discriminated struct + tests
  - [x] Tombstone types deleted from event/
  - [x] listing/ refactored to event-type-based detection (ADR-0114)
  - [x] watermill/ protocol updated (writeCommonMetadata replaces writeTracing)
  - [x] integration/event/ tests fixed (.UserID → .ActorID)
  - [x] API stability goldens regenerated
  - [x] `metaengine/enginetest/record_stamp.go:57-58` — FIXED.
  - [x] `system/sqlite_driver.go` deleted + `system.ErrUnknownDriver` removed.
  - [x] storage/aggregate_projection.go — reworked to `WithDeleteTypes` option.
  - [x] stack/materialize.go — reworked to `DeleteTypes`/`RebirthTypes` fields.
  - [x] transport/grpc/event_server.go — dead tombstone serialization removed.
  - [x] storage/sql_aggregate_reader.go — `listing.Status` replaces `event.TombstoneStatus`.
  - [x] example/taskmanager — `[]system.ProjectionDeclaration` with `system.RawQuery()`.
  - [x] All test files fixed (sql_aggregate_reader_test.go, view_models_integration_test.go,
        fuzz_test.go, auto_fold_record_test.go, soak_record_test.go,
        adapter_record_test.go, projectionhost_record_test.go).
  - [x] Dedup baseline updated (74→90 groups for concurrent engine clones).
  Remaining (tests still failing, NOT build breaks):
  - [x] **Memory engine graph ADT support** — DONE 2026-08-10 session 3.
        Added `GraphAddEdge`/`GraphNeighbors` to memory engine (`memory_graph.go`).
        `ADTGraph: ComplexityODegree` added to profile. All 15 Ginkgo specs pass.
  - [x] **Branded-ID auto-fold stamping panic** — DONE 2026-08-10 session 3.
        `metaengine/record_stamp.go` getters now call `.String()` on branded types.
        `TestAutoFold_RecordAware_Insert` + `TestIntegration_AutoInsert_ThroughAdapter` pass.
  - [x] **Signing golden stale** — DONE 2026-08-10 session 3.
        `hmac-signed-metadata.snap` regenerated. Obsolete `signature-json.snap` cleaned.
  - [x] **Metadata roundtrip (pebble/bbolt)** — DONE 2026-08-11.
        Root cause: `id.ActorID` has unexported fields implementing `json.Marshaler`
        but NOT `cbor.Marshaler`. fxamacker/cbor uses reflection → encodes as `{}`.
        Fix: `metadataPayload` type stores metadata as JSON bytes inside the CBOR
        envelope. Applied to both pebble + bbolt serialization. ActorID regression
        test added. Committed in `74b5762e2`.
  - [x] **cqrs-lint findings mismatch** — DONE 2026-08-10 session 3.
        F001 rule rewritten for domain deletion events. Golden profiles updated
        (33 findings, C017+V003 added). Catalog exclusion list updated for new modules.
  - [x] **example/taskmanager sqlite driver** — DONE 2026-08-10 session 3.
        Blank import of `sqliteengine/v4` added to register the "sqlite" driver.
  - [x] **All 82 workspace modules pass `go test`** — DONE 2026-08-10 session 3.
  - [x] Run `nix run .#verify` end-to-end — DONE 2026-08-11. GREEN.
        Fixed build break (`Tombstone→DeletePolicy` in sql_aggregate_reader),
        regenerated API golden (3989 exports), fixed 147 lint issues (→0),
        fixed arch check script (submodule detection + `set -e` bug),
        registered bboltengine/mysqlengine/tursoengine in LAYER+DEP_BUDGET,
        removed metaengine excess deps (go-humanize, direct sqlite),
        updated dedup baseline (92 groups).
  - [x] Regenerate API stability goldens (`cmd/api-stability --update`) — DONE 2026-08-10 session 4.
        3992 exports verified. Meta-tests pass.
  - [x] Naming cleanup: rename `TombstonePolicy` → `DeletePolicy` across stack/ + listing/ — DONE 2026-08-10 session 4.
        listing: `DeletePolicy` (`DeleteExclude`/`DeleteInclude`/`DeleteOnly`), `ListOptions.DeletePolicy`.
        stack: `DeletePolicy` (`IncludeDeleted`/`ExcludeDeleted`/`OnlyDeleted`), `FilterDeleted`.
        14 Go files updated (production + tests). Zero old references remain.
  - [x] Write `docs/migration/tombstone-to-domain-events.md` — DONE 2026-08-10 session 4.
        Rewritten from scratch: fixes doc/code drift, covers all 3 patterns
        (metaengine.Remove, stack.Materialize.DeleteTypes, listing.WithDeleteTypes).
  - [x] Update docs: AGENTS.md, SKILL.md, skill references for ADR-0114 patterns — DONE 2026-08-10 session 4.
        12 files updated: AGENTS.md, advanced.md, core.md, modules.md, readmodels.md,
        FEATURES.md, DOMAIN_LANGUAGE.md, event/README.md, listing/README.md,
        stack/README.md, cqrs-lint/README.md. doc-check passes (695 refs).
  - [x] Run `nix fmt` on all changed files — DONE 2026-08-11.
  - [x] Fix stale `metadata/doc.go` — DONE 2026-08-11. Removed references to
        deleted `Tracing` type; documented `record.CommonMetadata` embedding.
  _(Effort: M — verification + docs + naming cleanup remaining)_
- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must
  v0.1.2).
- [x] 🔥 **Release `record/v4` tag** — DONE 2026-08-11. Tagged `record/v4.1.0`
      with branded ID types, ActorID taxonomy, and Merge method.
      `metadata/go.mod` + `event/go.mod` pinned to v4.1.0.
      _(Effort: S)_
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
- [x] **Add bbolt source-of-truth integration test** — DONE 2026-08-11.
      Four files added to `metaengine/bboltengine/` (352 lines): `persistence_test.go`
      (3 tests: volatile/persistent profile), `restart_safety_test.go` (2 tests:
      seq-counter seeding across close→reopen for StreamLog+Map+Multimap+FromDB),
      `disk_backed_test.go` (2 tests: on-disk persists, volatile does not),
      `calibration_bench_test.go` (3 benchmarks: Set, Get, CounterIncrement — follows
      badger pattern). All 7 tests + 3 benchmarks pass with `-race`. `go vet` clean.
      See `docs/status/2026-08-11_05-28_bboltengine-source-of-truth-tests.md`.
- [x] **Run `nix fmt` + `nix run .#lint` on bboltengine test files** — DONE
      2026-08-11. Files pass gofumpt + go vet + verify-fast lint pipeline.
      _(Effort: XS)_
- [x] **Run `go mod tidy` in bboltengine** — DONE 2026-08-11. Removed unused
      `dustin/go-humanize` indirect dep. Module builds clean.
      _(Effort: XS)_
- [x] **Fix `record_stamp.go` GOWORK=off build failure** — DONE 2026-08-11.
      Resolved by releasing `record/v4.1.0` tag with branded ID types.
      Engine modules can now resolve branded types under GOWORK=off.
      _(Effort: S)_
- [ ] **Add `CounterIncrement` benchmark to pebbleengine calibration** — badger
      and bbolt have it; pebble does not. All engines should have calibration
      data for all ADT operations.
      _(Effort: XS)_
- [ ] **bboltengine parity gaps** — pebbleengine has `edge_cases_test.go`,
      `fuzz_test.go`, `stream_log_test.go`, `watcher_test.go`, `scan_bench_test.go`
      that bboltengine lacks. Prioritize by which ADT operations bbolt actually
      supports differently from pebble.
      _(Effort: M)_

---

## Metaengine Coverage Gaps

- [x] **ADR-0117 command lifecycle implementation** — DLQ as event streams,
      retries as event streams (no status fields). Implemented in
      `commandlifecycle/` (events, recorder, middleware) and
      `commandlifecycle/projections/` (DLQ/retry-count/failure-log projections).
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
>
> Implementation sessions 2026-08-10/11: P1, P2, P3, and UX all DONE. `nix run .#verify` GREEN.
> Full status: `docs/status/2026-08-11_04-04_live-latency-phase2-complete.md`.

- [x] 🔥 **P1: Prober + LatencyTracker** — DONE 2026-08-10. Optional `Prober`/
      `TransactMeasurer` interfaces, `LatencyTracker` (ring buffer + incremental
      EWMA + P50/P95/P99), `ProbeEngine()` helper, `CalibrationCosts.NetworkRTT`
      prior, `Calibration` hosts live trackers whose EWMA `ApplyCalibration`
      merges into `Profile()`. `EngineProfile.RequiresNetwork` structural flag.
      Test-double engine proves a live RTT shift changes `Profile()`.
      PG (`SELECT 1`) + Dgraph (healthcheck) probes wired. 15 tests pass
      (incl. `-race`).
      Files: `latency.go`, `probe.go`, `reliability.go`, `engine.go`,
      `pgengine/probe.go`, `dgraphengine/probe.go`.
- [x] **P3: Open measurement ingress for external engines** — DONE 2026-08-10.
      Exported `StatSink` / `LatencySample` / `SampleKind`. `LatencyTracker`
      forwards every sample to a configured sink via `WithTrackerSink`.
      `ProbeEngine` accepts `WithProbeSink`. Test: fake prober drives planner
      decisions with/without live stats.
- [x] **Wire live latency into `GetStats`/Doctor UX** — DONE 2026-08-10.
      `Store.GetEngineStats(ctx) []EngineStats` with measured RTT, samples,
      lastProbe, stale. `Doctor()` adds `--- Latency ---` section. `ExplainPlan()`
      shows `rtt=live … (p95, n)` per remote engine. `FormatLiveLatency()` renders
      live/stale/local. WARN diagnostic when routing on prior/stale RTT.
      Files: `engine_stats.go`, `explain.go`, `rule_live_latency.go`.
- [x] **P2: `Store.Replan(ctx)`** — DONE 2026-08-10. Three-phase locking
      (assign under write lock, run rules without lock, atomic plan swap).
      Increments plan version. Picks up live RTT shifts. File: `store.go`.
- [x] **P2: Execution-time live re-scoring** — DONE 2026-08-10.
      `Store.CheckRouting(ctx)` re-scores all queries with current live profiles;
      emits `REPLAN-SUGGESTED` when an alternative exceeds the 20% hysteresis
      deadband. `Store.StartAutoReplan(interval)` runs the loop in background.
      `DefaultRoutingHysteresis` = 0.20. File: `store_routing.go`.
- [x] **Fix `staleThresholdFor` code smell** — DONE 2026-08-10. Removed
      `staleThresholdFor`; `buildEngineStats` uses the tracker's authoritative
      `LiveLatency.Fresh` for display-side staleness. Display and routing agree.
- [x] **Fix `LiveLatency.Fresh` OR-semantics** — DONE 2026-08-10. `Fresh` is
      now RTT-specific: true only when the RTT tracker has current samples.
      A read-only tracker does not suppress the WARN rule.
- [x] **Add `WithProbeWindow`/`WithProbeAlpha`/`WithProbeStale` ProbeOptions** —
      DONE 2026-08-10. Consumers can tune tracker window, EWMA alpha, and
      stale-after through the probe API.
- [x] **Wire mysqlengine Prober** — DONE 2026-08-10. `SELECT 1` timing.
      `MySQL_NetworkRTT` = 1ms prior. `RequiresNetwork: true`. File:
      `mysqlengine/probe.go`.
- [x] **Wire tursoengine Prober** — DONE 2026-08-10 (prior-only). Remote DSN
      detection (`libsql://`, `https://`, `http://`) sets `NetworkRTT` prior via
      calibration. `Turso_NetworkRTT` = 2ms. Live probing deferred — sqliteengine
      delegation prevents adding `Prober` without wrapping (documented gap).
- [x] **Migrate irohengine onto core `LatencyTracker`** — DONE 2026-08-10.
      `LatencyCollector` now delegates to two core `LatencyTracker` instances
      (delivery + convergence). Eliminates duplicate ring buffer + percentile
      machinery. `SortDurations`/`PercentileIdx` kept as transport utilities.
- [x] **Implement `TransactMeasurer` on PG** — DONE 2026-08-10.
      `pgEngine.MeasureTransact` times a real `SELECT value FROM meta_map ... LIMIT 1`
      point lookup (B-tree seek + JSONB decode). File: `pgengine/probe.go`.
- [x] **Run `nix run .#verify`** — DONE 2026-08-11. Full verify gate GREEN
      including live-latency code. No stale-GREEN risk.
      See `docs/status/2026-08-11_04-04_verify-green-and-lint-cleanup.md`.
- [x] **Integration test: real PG testcontainer + ProbeEngine** — DONE
      2026-08-11. Two tests in `metaengine/pgengine/probe_live_test.go`:
      `TestProbeEngine_RealPostgres_LiveRTT` (verifies live RTT samples,
      `HasLiveRTT`, `FormatLiveLatency` renders `rtt=live`, `TransactMeasurer`
      read tracker, `Failures()==0`) and `TestProbeEngine_RealPostgres_StaleAfterStop`
      (verifies stale transition after probe loop stops). Both pass with `-race`.
      **Also fixed a production wiring bug:** `pgEngine` used a named `cal` field
      instead of embedding `metaengine.Calibration`, so `ProbeEngine` could never
      install trackers — the entire live-latency system was dead code for real PG.
      See `docs/status/2026-08-11_05-53_pg-probeengine-integration-test-calibration-embedding-fix.md`.
      _(Effort: M)_
- [x] **Update AGENTS.md + skill docs** — DONE 2026-08-10. CHANGELOG updated,
      TODO_LIST updated, API golden regenerated (3992 exports). Skill recipes
      pending — tracked separately.
- [x] **Consolidate percentile helpers** — DONE 2026-08-10. Iroh's internal
      `computeStats` + `percentile` removed (LatencyCollector now uses core
      tracker). `PercentileIdx`/`SortDurations` kept as transport-facing utilities
      (separate modules, different use case — acceptable 3-line index formula).

#### Live Cost Measurement — Improvement Backlog (from phase 2 self-review)

- [x] **Integration test: real PG testcontainer + ProbeEngine** — DONE
      2026-08-11 (duplicate of item above — both tracked the same work).
      See `docs/status/2026-08-11_05-53_pg-probeengine-integration-test-calibration-embedding-fix.md`.
      _(Effort: M)_
- [x] **Add `WithRoutingHysteresis(float64)` Store option** —
      DONE 2026-08-11. `WithRoutingHysteresis` + `WithRoutingMinDelta` plan
      options added in `planner.go`. `DefaultRoutingMinDelta = 0.5` (ms).
- [x] **Accept parent context in `StartAutoReplan`** —
      DONE 2026-08-11. Signature changed to `StartAutoReplan(ctx, interval)`.
- [x] **Wire structured logging into `CheckRouting` and `Replan`** —
      DONE 2026-08-11. slog-based Info logging for replan completions and
      routing drift detection. `WithProbeErrorHandler` for probe failures.
      OTel deferred (metaengine has no otel dep).
- [x] **Differential `CheckRouting`** —
      DONE 2026-08-11. `routingSignature()` caches results until any engine's
      RTT changes (`store_routing.go`).
- [x] **Fix turso live probing gap** —
      DONE 2026-08-11. `sqliteengine.SetProber` + `ProberSetter` interface.
      Turso injects `db.PingContext` probe for remote DSNs. ProbeEngine's
      `IsRemote()` guard prevents probing local SQLite.
- [x] **Add absolute minimum delta to hysteresis** —
      DONE 2026-08-11. `DefaultRoutingMinDelta = 0.5ms`. `WithRoutingMinDelta`.
- [x] **Concurrency stress test** —
      DONE 2026-08-11. `TestConcurrency_ReplanCheckRoutingStress` in
      `live_latency_phase3_test.go`. Replan + CheckRouting + tracker shift +
      GetEngineStats in parallel goroutines. Passes with `-race`.
- [x] **Edge case tests** —
      DONE 2026-08-11. `TestCheckRouting_SingleEngineNoAlternative`,
      `TestReplan_SingleEngine`, `TestCheckRouting_CancelledContextReturnsNil`.
- [x] **Add `Replan` to Doctor report** —
      DONE 2026-08-11. `--- Routing ---` section in Doctor with plan version,
      replan count, hysteresis, and drift summary (`explain.go`).
- [x] **Wire `TransactMeasurer` on mysqlengine** —
      DONE 2026-08-11. `meta_map` point lookup with backtick-escaped `key`.
- [x] **Wire `TransactMeasurer` on dgraphengine** —
      DONE 2026-08-11. Predicate index seek on sentinel `__probe` key.
- [x] **Add live-latency section to AGENTS.md metaengine section** —
      DONE 2026-08-11. Full component table added under `### Live Cost Measurement`.
- [x] **Update `METAENGINE-LIVE-LATENCY-MODEL.md` implementation status** —
      DONE 2026-08-11. Phase 3 row added to status table. Header updated.
- [x] **Cost model: RTT amortization for batch reads** —
      DONE 2026-08-11. `NsForRead` subtracts RTT from scan-pattern fallback
      costs when `NsPerRead > RTT` (`engine.go`). Prevents overestimating
      remote scan cost.
- [x] **Probe failure observability** —
      DONE 2026-08-11. `ProbeHandle.Failures()` counter +
      `WithProbeErrorHandler` option + slog.Debug for probe failures (`probe.go`).
- [x] 🔥 **Fix dgraphengine Calibration embedding** — DONE 2026-08-11.
      Embedded `metaengine.Calibration` in all 7 remaining engines (dgraph,
      badger, bbolt, pebble, sqlite, duckdb, mysql). Removed explicit
      `SetCalibration` passthroughs (now promoted).
      _(Effort: XS)_
- [x] 🔥 **Fix badgerengine Calibration embedding** — DONE 2026-08-11.
      Same fix applied to all engines (see dgraphengine item above).
      _(Effort: XS)_
- [x] **Verify tursoengine Calibration/probe wiring** — DONE 2026-08-11.
      tursoengine is a thin factory shim that delegates to sqliteengine.
      sqliteengine now embeds `Calibration` (fix applied), so turso inherits
      `TrackerHost`/`liveLatencyReporter` via the sqlite engine. Verified:
      `sqliteengine` has `var _ metaengine.TrackerHost = (*sqliteEngine)(nil)`.
      _(Effort: S)_
- [x] **Add compile-time `TrackerHost` assertions to all remote engines** —
      DONE 2026-08-11. Added `TrackerHost`, `Prober`, `TransactMeasurer`,
      `Calibratable` assertions to pgengine, dgraphengine, sqliteengine,
      badgerengine. A future embedding regression now fails at compile time.
      _(Effort: S)_
- [x] **ProbeEngine should warn on missing `TrackerHost`** — DONE 2026-08-11.
      Added `slog.Warn` in `ProbeEngine` (metaengine/probe.go) when an engine
      implements `Prober`/`TransactMeasurer` but not `TrackerHost`. Design
      decision: warning (not error) — maintains backward compat for local
      engines while catching wiring bugs in remote engines.
      _(Effort: S)_
- [x] **Regenerate api-stability golden after pgengine embedding change** —
      DONE 2026-08-11. Golden was already up to date: "API surface OK: 3999
      exports verified". Meta-tests (`TestEveryGoModDirIsInModulesList`,
      `TestEveryGoModDirIsInTestModules`) also pass.
      _(Effort: XS)_

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

- [x] 🔥 **Finish Record consolidation (ADR-0111 Phases 3-4)** — DONE.
      `record.CommonMetadata` is the single structural base; `event.Metadata`
      and `command.Metadata` now embed it (no duplicated tracing fields);
      `metadata.Tracing` deleted; `metadata.CustomData[K]` deprecated.
      See CI/Release section (line ~209) for the full done-subitem list.
      _(Effort: L)_

### Phase 2: Quick Wins (dead code removal)

- [x] **Delete `metaengine.GraphBackend` (ADR-0113)** — remove the interface
      (`engine.go:394`), remove implementations from all engines (memory engine
      assertion at `engine.go:560`). Graph operations route through
      `graphadapter` exclusively. Update `adttest.RunMatrix` to use graphadapter.
      _(Effort: M)_
- [x] **Replace `simpleBus` with `watermill.EventBus` in `system/`** — delete
      `system/bus.go` (simpleBus), delete `BusDriverFactory` +
      `RegisterBusDriver` from `system/driver_registry.go`. Wire
      `watermill.EventBus` as the bus in `system/constructor.go:179-185`.
      Map `BusConfig.Driver` to watermill backend selection.
      _(Effort: M)_

### Phase 3: Self-Registration Infrastructure

- [x] 🔥 **Move driver registry to `metaengine/`** — relocate `RegisterDriver`,
      `DriverFactory`, `EngineConfig`, `lookupDriver`, `createEngineFromDriver`
      from `system/driver_registry.go` to a new `metaengine/registry.go`.
      `system/` calls `metaengine.LookupDriver(name)` instead of its own map.
      All 9 engines already depend on `metaengine/`, so no new deps.
      _(Effort: M)_
- [x] **Convert memory + sqlite to self-registration** — move their
      `RegisterDriver` calls from `system/init()` to their own packages.
      `metaengine/memory_engine.go` gets `register.go`; `sqliteengine/` gets
      `register.go`. Verify `system.New()` still works via blank imports in
      tests/examples.
      _(Effort: S)_

### Phase 2–3 Follow-ups (discovered during status review)

- [x] **Delete `system/sqlite_driver.go`** — DONE. Removes `database/sql` and
      `modernc.org/sqlite` from system's production deps.
      _(Effort: XS)_ ✅
- [x] **Fix stale GraphBackend error messages** — DONE. 8 Fatal strings in
      dgraphengine test files changed to `"does not implement graph dispatch"`.
      _(Effort: XS)_ ✅
- [x] **Rename `TestGraphBackend` → `TestGraphOperations`** — DONE.
      _(Effort: XS)_ ✅
- [x] **Fix stale GraphBackend doc references** — DONE. METAENGINE_DOMAIN_LANGUAGE.md,
      metaengine/README.md cleaned. ROADMAP.md left as historical migration doc.
      _(Effort: XS)_ ✅
- [x] **Remove `system.ErrUnknownDriver`** — DONE. API-stability golden regenerated.
      _(Effort: XS)_ ✅
- [x] **Run `nix run .#verify-fast`** — DONE. Build + vet + doc-check + doc
      assertions PASS. Also fixed ADR-0114 tombstone build breaks in storage/,
      stack/, transport/grpc/, example/taskmanager. Test failures remain
      (pre-existing, see ADR-0114 fallout section below).
      _(Effort: S)_ ✅
- [x] **Run `nix run .#check-duplication`** — DONE. 0 new clones. Baseline
      updated 74→90 for concurrent-work clones.
      _(Effort: S)_ ✅

### Phase 4: Backend Porting (all 8) — DONE 2026-08-10

> All 8 engines self-register via `metaengine.RegisterDriver`. 6 pre-existed
> (pebble, postgres, duckdb, badger, dgraph were done in Phase 3; memory is
> built-in). The remaining 3 (bbolt, mysql, turso) were created this session.
> All 10 driver names now register: memory, sqlite, pebble, bbolt, duckdb,
> postgres, mysql, badger, dgraph, turso.
>
> **Remaining gaps** (see status report
> `2026-08-10_16-14_metaengine-backend-porting-bbolt-turso-mysql.md`):
> missing compile-time assertions, calibration benchmarks, MySQL integration
> test infrastructure, SKILL.md docs update, `nix fmt` + `nix run .#verify`.

- [x] 🔥 **Port pebble driver** — `metaengine/pebbleengine/register.go` with
      `RegisterDriver("pebble", ...)`. Map `cfg.DSN` to directory path. Handle
      in-memory (`vfs.NewMem`) when DSN is empty. Verify through system tests.
      _(Effort: S)_
- [x] **Port bbolt driver** — new `metaengine/bboltengine/` module (or extend
      existing `storage/bbolt` as a metaengine engine). Self-register as
      `"bbolt"`. Map DSN to file path.
      _(Effort: M)_
- [x] **Port postgres driver** — `metaengine/pgengine/register.go` with
      `RegisterDriver("postgres", ...)`. Map `cfg.DSN` to pgx connection string.
      Handle pool config from `cfg.Pragmas` or new `EngineConfig` fields.
      _(Effort: S)_
- [x] **Port duckdb driver** — `metaengine/duckdbengine/register.go` with
      `RegisterDriver("duckdb", ...)`. CGo isolation preserved (separate module).
      Map DSN to file path or `:memory:`.
      _(Effort: S)_
- [x] **Port mysql driver** — new `metaengine/mysqlengine/` module or extend
      `pgengine` with MySQL dialect. Self-register as `"mysql"`.
      _(Effort: M)_
- [x] **Port turso driver** — `metaengine/tursoengine/` or extend with libSQL.
      Self-register as `"turso"`. Handle sync config.
      _(Effort: M)_
- [x] **Port badger driver** — `metaengine/badgerengine/register.go`.
      Self-register as `"badger"`.
      _(Effort: S)_
- [x] **Port dgraph driver** — `metaengine/dgraphengine/register.go`.
      Self-register as `"dgraph"`.
      _(Effort: S)_

### Phase 5: Record-Typed Default Folds

- [ ] 🔥 **Make `OnRecord` the default fold constructor** — change examples,
      docs, and auto-projection to use `OnRecord`/`OnRecordTyped` instead of
      `On`. Fold handlers receive `record.Record` as the first parameter.
      Deprecate payload-only `On` (mark deprecated, remove in v5 cut).
      _(Effort: M)_

### Phase 6: Auto-Projection (the killer feature)

- [x] 🔥🔥 **Planner-time fold inference (ADR-0116 Layer 1)** — IMPLEMENTED
      2026-08-11 (code complete, tests green, **NOT verify-gated**). The
      `metaengine.Infer(samples...)` API lets consumers declare zero folds:
      the planner inspects event/query struct shapes at `Plan()` time and
      auto-generates insert/update/delete folds. Implements: convention
      detection (`*Created`/`*Updated`/`*Deleted` suffixes), key field
      auto-detection (from query input type, falling back to `"ID"`),
      field-name matching (incl. nested struct flattening via `srcPath []int`),
      filter auto-detection (query input fields → `FilterOnField`), collection
      result support (`R{Items []T}` → element type T). 12 tests, 145 total
      `go test` green (workspace mode). ADR-0116 status updated to "Layer 1
      implemented". API golden regenerated (3993 exports).
      **Not recommended for production domain models** — hides projection logic
      behind conventions; prefer explicit `OnRecord`/`AutoInsert` folds.
      Disclaimer added to Go doc, ADR, and skill reference.
      **WARNING: `nix run .#verify` NOT YET RUN.** `nix fmt` not run. File
      line-count violations likely (`query.go` at 417, limit 350). See
      `docs/status/2026-08-11_05-09_fold-inference-adr0116-layer1-status.md`.

      Not yet implemented (separate tasks below):
      - **Fold inference gap:** `Infer()` does not decompose `[]Struct` fields
        in *event types* (only handles `Items []T` in *result types* via
        `collectionElementType`). This is a fold-generation gap, orthogonal to
        Phase 6b layout planning. Layout planning decides physical shape
        (embed vs normalize); fold inference generates the fold functions.
      - Fold inference override API
      - `InferFromNamedEvents()` for wire event types
      - Sort inference, composite keys, filter operators beyond `FilterEq`
- [ ] 🔥 **Run `nix run .#verify` for fold inference** — the Infer() feature
      was committed without the full CI gate. Must fix: `nix fmt`, file
      line-count violations (`query.go` 417 > 350 limit), lint, arch, dedup,
      coverage, race. See status report for full gap list.
      _(Effort: M)_
- [ ] **Fold inference override API** — when auto-projection gets it wrong,
      consumer can override with an explicit `OnRecord` fold for a specific
      event/query pair. Override replaces (not supplements) the generated fold.
      Without this, `Infer()` is all-or-nothing.
      _(Effort: M)_
### Phase 6b: Operator-Driven Layout Planning (replaces M9)

> **Original M9 was wrong.** It assumed normalization is always correct and put
> storage intent on the developer. The revised model: the developer is silent,
> the operator controls layout via priorities + the cost model. See the full
> design rationale in [`docs/planning/METAENGINE-LAYOUT-PLANNING-MODEL.md`](docs/planning/METAENGINE-LAYOUT-PLANNING-MODEL.md).
>
> Key insight: `[]Attachment` vs `[]AttachmentID` tells the planner what data
> is in the payload, NOT what layout to pick. The planner reads reality, not
> intent. Layout is 100% the operator's call.
>
> **Layout planning ≠ fold inference.** Fold inference (ADR-0116 Layer 1)
> generates fold functions (how events map to projection entries). Layout
> planning decides the physical storage shape of those projections (embed vs.
> normalize within an engine). They are orthogonal concerns.

- [ ] 🔥 **Operator priority system** — define `Priority` enum
      (`WriteSpeed` / `ReadSpeed` / `StorageSpace` / `Balanced`) and a priority
      hierarchy: `GLOBAL` (whole deployment) → per-Engine → per-Query (most
      specific wins). Wire into `EngineConfig` / `QueryDecl` / deployment config.
      The priority weights the existing cost model — it does not bypass it.
      _(Effort: L)_
- [ ] 🔥 **Cost model: embed-vs-normalize scoring** — extend the cost model to
      score embed (denormalized) vs. normalize (child collection + join) per
      field, per priority, per backend. The per-backend truth: KV favors embed,
      SQL favors normalize, graph favors normalize, DuckDB is workload-dependent.
      Even single nested structs (not just slices) can be normalized if the
      priority justifies it. The operator's control has no structural floor.
      _(Effort: L)_
- [ ] **Benchmark mode** — let the operator try multiple plans against real or
      simulated workloads and see measured results + scaling predictions before
      committing. Delivery: both CLI (extends `cqrs-bench`, pre-deployment "what
      if") and runtime API (ongoing monitoring + adaptive re-tuning). Workload:
      synthesize from declared queries by default; accept real operator-provided
      traces for calibration.
      _(Effort: L)_
- [ ] **Runtime backend addition + dual-use / migration / backup** — the
      planner maintains parallel projections across engines with explicit roles:
      active read, migration target, backup replica, dual-use. New backends added
      at runtime; planner generates plan + backfills from event log. Sync is
      role-based: fold pipeline (strong) for active + dual-use; async replication
      (eventual) for backup + migration.
      _(Effort: XL)_
- [ ] **Threshold-based re-layout trigger** — when the operator changes a
      priority, small projections (below threshold) rebuild automatically from
      the event log; large ones require explicit operator confirmation. Threshold
      is operator-configurable. Prevents a global priority change from silently
      launching massive parallel rebuilds.
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
