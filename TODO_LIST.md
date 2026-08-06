# TODO List

**Updated:** 2026-08-06 (post-SUPERB execution plan session 1)
**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## System Package (EXPERIMENTAL — first pass + Pareto P0/P1 fixes shipped)

> The `system/` module implements the operator-configured CQRS topology from
> the [metaengine redesign](docs/planning/metaengine-redesign.md). Driver
> registry wired, SQLite working through `New()`, projections E2E proven,
> MultiBus/SnapshotBackend/scream store wired, introspection real. All file-size
> violations resolved (constructor.go, system.go, adapter_event.go split). Tagged
> system/v4.0.0. Remaining work is hardening and completeness, not blocking.

### P1 — Hardening (makes the design production-ready)

- [ ] **Scream store: PlanDiff / PlanFingerprint / Manifest** — the scream
      store's value proposition is detecting unsafe runtime changes by diffing
      the current `SerializablePlan` against a pinned manifest. Without these,
      it's a config validator, not a scream store. Design ref: §9.3–9.5.
- [ ] **CommandAdapter + QueryAdapter serialization** — both adapters compile
      but need a `serializedCommand`/`serializedQuery` envelope for SQL engines
      (same pattern as `serializedEvent`).
- [ ] **`example/taskmanager` full migration to System** — metaengine.go DX
      rewrite done (372→193 lines, now uses `NewTypeDecoder`/`Register`/
      `PlanFromSQLite`), but the app still wires manually via `system.New()`.
      Full migration proves the consumer experience end-to-end.

### P2 — Important for completeness

- [ ] **koanf YAML config** — `yaml.v3` parsing added but `koanf` integration
      (G2 "deployer decides") is unmet. Env var overrides are basic.
- [ ] **Pebble AtomicAppender + DuckDB/PG Transactional** — Pebble lacks
      `StreamAppendExpected` (AtomicAppender). DuckDB + Postgres have
      AtomicAppender but lack `Transactional` (`RunInTx`). Both are
      compile-time assertions; Transactional enables cross-collection atomic
      writes.
- [ ] **Pebble restart safety test** — `seedSeqCounters()` implementation is
      complete but no direct restart test exists (only indirect E2E).
- [ ] **Bus driver registry** — types exist, gochannel driver registered, but
      `BusConfig{Driver: "nats"}` / `{Driver: "redis"}` do nothing.

---

## Metaengine

> 5 engines (Memory, SQLite, Pebble, DuckDB, Postgres), 10/10 ADTs on all
> engines (Universal ADT Phase 3 shipped, ADR-0094), replication model
> (ADR-0093), persistence enum (ADR-0098), consumer DX helpers
> (`NewSQLiteEngineFromDSN`, `PlanFromSQLite`, typed projection decoders),
> CalibrateEngine exported, ReadCosts (with SerializableReadCosts in plan JSON),
> SSE reconnect, WatchTyped, and `Inspect()` extraction are all shipped.
> ADR-0100 documents the per-read-pattern cost model. metaengine v4.5.0 tagged.

- [ ] **Postgres GIN containment indexes** — add `@>` operator support for
      JSONB path queries; currently only B-tree expression indexes are
      implemented. Needs `FilterContains`/`FilterExists` operators.
      Evidence: `metaengine/pgengine/pushdown.go`.

- [ ] **10M soak test verification & hardening**
  - Run `TestSoak_MemoryBounded_10M` 3× with `-race` and record variance.
  - Investigate the 12→15MB heap threshold bump — 13.6MB for 100 keys × 50K
    events needs root-causing (likely GC pressure under parallel test load).
  - Add `TotalAlloc` tracking to the 10M variant.
  - Add engine parity soak tests (pgengine/duckdbengine/pebbleengine 1M/10M).

- [ ] **Document `metaengine` watcher delete semantics** — delete notifications
      deliver the zero value of `V` after the reification fix; this contract
      should be documented in `metaengine/README.md` or `metaengine/COOKBOOK.md`.

- [ ] **Add SerializableReadCosts to ExplainPlan output** — calibrated costs
      are in the plan JSON but not shown in the human-readable `ExplainPlan()`
      or `Doctor()` output.

> Long-term metaengine work (`metaengine-gen` code generator, generic
> `ScanResult[T]`, Vector/Search/Spatial engine backends, DuckDB
> columnar-native storage, Iroh distributed engine, operator-driven engine
> selection) lives in [ROADMAP.md](ROADMAP.md).

---

## Irohengine

> Level 2 prototype shipped with CRDT-safe operations. Three transports:
> InProcessNetwork (goroutine, no CGo), loopback (real TCP, no CGo), QUIC
> (`iroh-go` C bindings, CGo required). CBOR encoding, latency measurement.
> loopback/v4.0.0 + quic/v4.0.0 tagged.

- [ ] **Evaluate `iroh-go` C binding stability** — the QUIC transport depends
      on `git.coopcloud.tech/decentral1se/iroh-go`, a third-party Go binding for
      Iroh (Rust). Assess upstream maintenance, API stability, and whether to
      vendor/fork. Evidence: `metaengine/irohengine/quic/transport.go:14`.
- [ ] **QUIC transport integration with `adttest.RunMatrix`** — the in-process
      mock + loopback pass the full matrix; verify the QUIC transport also
      passes parity tests (LWW resolution, PN-Counter, MapUpdate-does-not-replicate).
- [ ] **Non-CRDT op rejection on QUIC path** — verify `MapUpdate` operations
      stay local-only and are NOT sent over QUIC (would break CRDT convergence).
- [ ] **WriteOp.ID dedup ring** — `SetAdd`/`CounterIncrement` are NOT idempotent;
      double-delivery on redelivery corrupts state. QuicTransport has a 10K-bound
      `dedupSeen` set; loopback does not.

---

## cqrs-lint

> 186 rules across 10 categories. Config presets, `--adoption`/`--scorecard`/
> `--group-by` flags (text + JSON + Markdown + SARIF output), changelog subcommand,
> self-lint mode, block-level suppression, C038-C040 (event-type mismatch/
> dead-fold-case detection), per-module feature profiles (S002/S003/S006/S007/
> C017/C036/A015/A016/B014/E009/A012/A009/E016 migrated), E006 fold-aware,
> JSONC config loader, `explain` command, doctor overhaul, `init` SHOWSTOPPER
> fix, B025 cross-package tracing, KeyHolderAI feedback fixes shipped.
> feature_detect.go + output.go split under 350-line CI limit. v4.4.0 tagged.

- [ ] 🔥 **Run cqrs-lint against real consumer projects** — validate
      false-positive rates against Kernovia, Standup-Killer, bank-sync,
      cqrs-htmx, DiscordSync, timesheets, crush-daily, KeyHolderAI. This is the
      single highest-value non-coding task for cqrs-lint trustworthiness.

- [ ] **Missing regression tests** — S006 fix (WEAK suppression), A018 fix
      (dispatch activity check), B004 fix (constructor check) — 3 of 7
      KeyHolderAI fixes have no regression tests.

- [ ] **Migrate global detectors to per-module evaluation** —
      `ProfileForFile` infrastructure exists. 13+ detectors migrated. Remaining:
      ~20 detectors still use `ctx.FeatureProfile` directly. F-series rules are
      intentionally project-level (they coach the whole project). High
      false-positive risk for multi-module workspaces.

- [ ] **Scorecard SARIF `logicalLocations`** — SARIF output represents adoption
      metrics as `notifications` (not `results`). The `logicalLocations` half
      of IMPROVEMENT_IDEAS.md item 195 is still pending.

- [ ] **L1.5 domain severity calibration** — highest-impact open Pareto item.
      `DomainKind` enum + `applyDomainBias` shipped; still needs broader testing
      against financial/security projects.

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

- [ ] **Benchmark audit for 10 skipped modules** — benchmark assertion sweep
      covered 18 files but skipped: codec, command, dispatcher, query,
      middleware, snapshot, listing, watermill, transport/http, storage/view.
      These likely have the same `_, _ =` result-discarding pattern that was
      fixed in the other modules. Evidence:
      `docs/status/2026-08-04_06-49_benchmark-assertions-brutal-self-review.md`.

---

## Dedup

> Clone groups reduced from 69 → 65. All remaining groups classified into
> 6 categories: cross-module isolation (11), table-driven tests (8/109 clones),
> testcontainer setup (3), trivial boilerplate (11), within-module remnants (4),
> other (3). `art-dupl` baseline golden + `nix run .#check-duplication` gate
> enforce no-new-clones.

- [ ] **Review remaining 65 clone groups** — duckdb↔pgengine parity code,
      testcontainer setup patterns, table-driven test boilerplate. Some are
      intentional (cross-module isolation, table-driven); others are
      extractable (`renderTable`, `testutil/pgtest` shared module).
- [ ] **`renderTable` extraction** — `cmd/cqrs-lint/explain.go` still has
      repeated table-rendering patterns (partially extracted to
      `renderKeyTable` but not fully DRY'd).
- [ ] **`deferClose(closer)` helper** — metaengine engines have repeated
      `if c := eng.Close(); c != nil ...` patterns across 5 engines.

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).

- [ ] **Pin GitHub Actions to commit SHAs** — 72+ unpinned actions
      (supply-chain risk).
- [ ] **Add `go test` to CI for example/taskmanager** — currently only builds,
      no test step.

---

## Integration Test Infrastructure

> Nix-based integration test infrastructure shipped: ephemeral PG, NixOS VM
> tests (PG+MySQL), nspawn MySQL (M14, ~15s), projectionhost PG crash-restart
> (M43), scheduling/sqlstore durable timers (M44). All in ephemeral-pg.sh.

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

## Deferred Debt (ADR-committed)

Two items explicitly committed to in the 2026-08-03 ADR review as "the next
real roadmap." Each has a clear ADR with rationale.

- [ ] **Ghost bus removal** (ADR-0028) — delete `memory/bus.go`,
      `memory/command_bus.go`, `storage/pg_bus.go`. Largest blast radius — audit
      ALL consumer repos first.
- [ ] **Metadata aliases completion** (ADR-0031) — `command.Metadata` /
      `query.Metadata` → standalone structs (currently repointed aliases with
      functional `WithCustom`, but not yet fully standalone types).

> `retry/` → `go-retry` and `idempotency/` → `go-idempotency` extraction is
> DONE — both repos pushed with annotated tags. See CHANGELOG.

---

## Layer Enforcement

> `check-module-layers.sh` now has a self-enforcing coverage guard. 68/68
> modules covered with LAYER/DEP_BUDGET entries. ADR-0046 updated to the
> seven-tier model. FOUR-TIER-MODEL.md filename deliberately kept (H1 updated
> to "seven-tier" to avoid breaking inbound links). All layer exceptions verified
> valid.

- [ ] **Expand go-arch-lint to remaining 63 of 69 modules** — only 6 modules
      have per-module go-arch-lint configs. The bash script is the enforcement
      mechanism for the rest.
- [ ] **Consider rewriting `check-module-layers.sh` as `cmd/check-layers`** —
      330 lines of bash. A Go program would be more maintainable and testable.

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
