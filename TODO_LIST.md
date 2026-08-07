# TODO List

**Updated:** 2026-08-08 (code-verified audit: 9 stale items marked done)
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
> deprecation. ADRs 0111-0119 written. Verify gate GREEN (all 17 steps pass).
> Tags pushed: sqliteengine, graphadapter, dgraphengine, storage/bbolt,
> idempotency. Remaining work is completeness and edge-case coverage.

### Publishability (blocks external consumers)

- [ ] **Tag `metaengine/bench/v4.0.0`** — module exists in `go.work` but has no
      git tag. External consumers cannot import it.
- [ ] **Tag `metaengine/pebbleengine/v4.0.0`** — module exists in `go.work` but
      has no git tag. External consumers cannot import it.
- [ ] **Tag drifted modules for GOWORK=off CI** — `retry/v4`, `middleware/v4`,
      `benchkit/v4`, `stack/*` have API changes since their last tag. Per-module
      CI (GOWORK=off) fails because it resolves to stale versions.

### Test coverage

- [ ] **Record-aware integration test through Pebble engine** — Record-aware
      pipeline through `metaengine/pebbleengine` (SQLite engine test exists).
- [ ] **Soak test with `AutoCRUDByConvention`** — existing soak tests use
      manual folds; verify the auto-projection path under sustained load.
- [ ] **Add `RunTransactionalTest` to sqliteengine/badgerengine tests** —
      DuckDB + PG have it; SQLite + Badger do not.
- [ ] **Add concurrent `RunInTx` test** — two goroutines, verify isolation.
- [ ] **Add `MultiAdd` + `LogAppend` transactional tests** —
      `RunTransactionalTest` exercises MapBackend + CounterBackend + StreamLog
      inside `RunInTx`; Multimap and Log backends are untested in transactions.

### Module health

- [ ] **Add `metaengine/keycodec`, `metaengine/enginetest`,
      `testutil/pgtestcontainer` to api-stability modules list** —
      `TestEveryGoModDirIsInModulesList` will fail without these.
- [ ] **Add same 3 modules to AGENTS.md module list** — currently missing from
      the Quick Reference table and Monorepo Structure tree.
- [ ] **Fix 16 COVERAGE GAPs in `check-module-layers.sh`** — newer modules
      (badgerengine, dgraphengine, graphadapter, sqliteengine, metaengine/bench,
      testutil/pgtestcontainer, record) missing from LAYER/DEP_BUDGET maps.

> Long-term metaengine work (`metaengine-gen` code generator, Vector/Search/
> Spatial engine backends, generic `ScanResult[T]`, operator-driven engine
> selection) lives in [ROADMAP.md](ROADMAP.md).

---

## System Package (EXPERIMENTAL — P0/P1 fixes shipped, hardening continues)

> The `system/` module implements the operator-configured CQRS topology.
> Driver registry wired, SQLite working through `New()`, projections E2E
> proven, MultiBus/SnapshotBackend/scream store wired, introspection real.
> koanf YAML config, DuckDB/PG Transactional, bus driver registry, scream store
> plan-drift detection, CommandAdapter/QueryAdapter serialization, and
> example/taskmanager migration all shipped. Tagged `system/v4.0.0`. Remaining
> work is documentation, edge-case tests, and completeness.

### P2 — Hardening (makes the design production-ready)

- [ ] **`system/README.md` Quick Start doesn't compile** — missing imports,
      types. Needs to be copy-pasteable as a standalone program.
- [ ] **Fix `cmd/doc-check` cmdguard arg-parsing** — `cobra.ArbitraryArgs` is a
      band-aid; doc-check should properly accept file args via cmdguard.
- [ ] **Add `system.HealthCheck(ctx)` method** — delegates to registered
      resources that implement `stack.HealthChecker`.
- [ ] **Add `system.GracefulClose(ctx)`** — bounded `Close()` with timeout.
- [ ] **Add `system.ResetProjection(name)`** — delegates to
      `projectionhost.Host.Reset()`.
- [ ] **Wire checkpoint store as configurable** — currently hardcoded to
      `memoryCheckpointStore`; needs `WithCheckpointStore` option.

---

## bbolt Storage Backend

> Full storage backend shipped: EventStore, SnapshotStore, CheckpointStore,
> KVAdapter, CommandStore, QueryStore, Backend facade. Streaming iterators
> (`StreamingSource`/`StreamingJournal`), OTel span instrumentation, contract
> test suite (16 tests), durability tiers via `stack/bbolt`.
> `storage/bbolt/v4.0.0` tagged and pushed.

- [ ] **Add CommandStore contract tests** — Save/AppendBatch/Load/ReadAll/
      ReadFrom/duplicate detection.
- [ ] **Add QueryStore contract tests** — SaveQuery/LoadQueries/ReadAll/
      ReadFrom/duplicate detection.
- [ ] **Add same-stream concurrency contention test** — 10 goroutines racing
      for same stream (verify optimistic concurrency).
- [ ] **Add bbolt to `stack/bench/`** — zero benchmarks exist for bbolt.
- [ ] **Consider `WithBatchSize` option for `AppendBatch`** — currently
      appends one event per tx.

---

## Irohengine

> Level 2 prototype shipped with CRDT-safe operations. Three transports:
> InProcessNetwork (goroutine, no CGo), loopback (real TCP, no CGo), QUIC
> (`iroh-go` C bindings, CGo required). loopback/v4.0.0 + quic/v4.0.0 tagged.

- [ ] **QUIC transport integration with `adttest.RunMatrix`** — the in-process
      mock + loopback pass the full matrix; verify the QUIC transport also
      passes parity tests (LWW resolution, PN-Counter, MapUpdate-does-not-replicate).
- [ ] **Non-CRDT op rejection on QUIC path** — verify `MapUpdate` operations
      stay local-only and are NOT sent over QUIC (would break CRDT convergence).
- [ ] **Fix `TestQuicSetConvergence` flakiness** — pre-existing, network-dependent.
      Consider `//go:build integration` tag or relaxed timing bounds.
- [ ] **Fix `TestLoad_ConcurrentLoadsCoalescedBySingleflight` flake** —
      pre-existing race-condition test that flakes under parallel load.

---

## cqrs-lint

> 192 rules across 10 categories. Config presets, `--adoption`/`--scorecard`/
> `--group-by` flags, SARIF + Markdown output, self-lint mode, block-level
> suppression, metaengine-aware detection (F018-F026), scorecard metaengine
> section, cross-format consistency tests. v4.4.0 tagged. Self-lint clean
> (0 CRITICAL, 0 ERROR, 0 load errors, 0 stale suppressions).

- [ ] 🔥 **Run cqrs-lint against real consumer projects** — validate
      false-positive rates against Kernovia, Standup-Killer, bank-sync,
      cqrs-htmx, DiscordSync, timesheets, crush-daily, KeyHolderAI. This is the
      single highest-value non-coding task for cqrs-lint trustworthiness.

- [ ] **Fix remaining false positives** — C001 (read-only bbolt transactions),
      D012 (CLI tools should be excluded), C008 (non-monetary floats).

- [ ] **Triage remaining ~199 self-lint WARNING/INFO findings** — D007 (8
      `event.NewEvent` → `event.New`), D014 (15 missing json tags), C034 (8
      `go func()` without ctx), C033 (~15 bare `return err`), C023 (~10
      unchecked `Close()`), P012/P013 (~6 SQLite without WAL/busy_timeout),
      A032 (~8 string/int fields instead of branded ID).

- [ ] **Missing regression tests** — S006 fix (WEAK suppression), A018 fix
      (dispatch activity check), B004 fix (constructor check) — 3 of 7
      KeyHolderAI fixes have no regression tests.

- [ ] **Migrate global detectors to per-module evaluation** —
      `ProfileForFile` infrastructure exists. 13+ detectors migrated. Remaining:
      8 detector files still use `ctx.FeatureProfile` directly (6 in `adoption/`,
      1 in `api/`). F-series rules are intentionally project-level. High
      false-positive risk for multi-module workspaces.

- [ ] **Scorecard SARIF `logicalLocations`** — SARIF output represents adoption
      metrics as `notifications` (not `results`). The `logicalLocations` half
      is still pending.

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

---

## Code Quality

- [ ] **`metadata.CustomData[K]` immutability gap** — `command.Metadata` and
      `query.Metadata` migrated to `WithCustom()` (functional), but
      `metadata.CustomData[K]` still has pointer-receiver `EnsureCustom()`.
      Decision needed: complete the immutability sweep or accept the exception.

- [ ] **`query.WithCustomMetadata` missing** — `command` has
      `WithCustomMetadata(key, value string) Option` but `query` does not.
      Asymmetry between the two modules.

- [ ] **Stale `metadata/README.md`** — still documents `EnsureCustom` (removed
      from command/query). Needs update to `WithCustom`.

- [ ] **Fix `.golangci.yml` exclusion sprawl** — ~30 blocks, ~50% undocumented.
      Add comments explaining WHY for every exclusion. Consider per-module
      config split.

---

## Dedup

> Clone groups driven to **0 at threshold 3** (was 65). All thresholds (7, 4,
> 3) reduced to 0 through shared helper extraction. `.art-dupl-baseline.json`
> baseline: 0 groups. `nix run .#check-duplication` gate enforces no-new-clones.

- [ ] **Investigate threshold-2 clone groups** — 92 remaining at t=2. Some are
      intentional (cross-module isolation, table-driven); others may be
      extractable (`capitalizeFirst`, `truncateString`, `isCBORData`,
      `recordErr`, `startStreamSpan` patterns).
- [ ] **Extract `renderTable(b, headers, rows)` helper** — `cmd/cqrs-lint/explain.go`
      still has repeated table-rendering patterns.
- [ ] **`deferClose(closer)` helper** — metaengine engines have repeated
      `if c := eng.Close(); c != nil ...` patterns across 7 engines.

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
- [ ] **Fix 16 COVERAGE GAPs** — newer modules (badgerengine, dgraphengine,
      graphadapter, sqliteengine, metaengine/bench, testutil/pgtestcontainer,
      record, example/metaengine-quickstart) missing from LAYER/DEP_BUDGET maps.
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
