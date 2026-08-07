# TODO List

**Updated:** 2026-08-06 (post metaengine v2 completion)
**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Metaengine v2 (Record-aware ES-native architecture)

> Metaengine v2 is **feature-complete**: `record/` module, 9 engines (Memory,
> SQLite, Pebble, DuckDB, Postgres, Badger, Dgraph, GraphAdapter, Iroh),
> `OnRecord`/`ApplyRecord` Record-aware folds, `AutoInsert`/`AutoCRUD`/
> `AutoCRUDByConvention` auto-projection, tombstone deprecation. ADRs
> 0111-0119 written. Remaining work is hardening, completeness, and
> publishability.

### Publishability (blocks external consumers)

- [ ] **Tag untagged modules** — `metaengine/sqliteengine/v4`,
      `metaengine/graphadapter/v4`, `metaengine/dgraphengine/v4`,
      `storage/bbolt/v4` are implemented and in `go.work` but have **no git
      tags**. External consumers running `go mod tidy` (GOWORK=off) get
      `unknown revision`. Evidence: `git tag -l 'metaengine/sqliteengine/*'`
      returns empty.
- [ ] **Run `nix run .#verify`** — the full gate (build+vet+test+race+lint+
      doc-check) was never confirmed GREEN across the v2 sessions. Targeted
      tests pass; the full gate is the only source of truth.
- [ ] **Run `nix run .#check-layers`** — verify dependency budgets after
      adding `record/v4` to `event/`, `command/`, `metaengine/` modules.

### Code quality

- [ ] **`auto_naming.go` dedup refactor** — `autoInsertByType`,
      `autoUpdateByType`, `autoDeleteByType` duplicate logic from the generic
      `AutoInsert[E,R]`/`AutoUpdate[E,R]`/`AutoDelete[E]`. Refactor the generic
      versions to delegate to the `ByType` core. Evidence:
      `metaengine/auto_naming.go:14`, `metaengine/auto_fold.go:80`.
- [ ] **`record.FromCommand()` adapter** — mirrors `event.AsRecord()`.
      Completes the Record vision for both events AND commands. Evidence:
      `event/asrecord.go:41` (event side done; command side missing).
- [ ] **AutoCRUDByConvention naming convention documentation** — the function
      matches Go struct names (`"TaskCreated"`) not dot-separated event types
      (`"task.created"`). This differs from the rest of go-cqrs-lite and must
      be documented in the godoc. Evidence: `metaengine/auto_naming.go:145`.

### Test coverage

- [ ] **Projectionhost lifecycle test** — Record-aware folds through the
      full `projectionhost.Host.Start/Stop/checkpoint` lifecycle (current
      integration tests call `Handle()` directly).
- [ ] **SQLite engine integration test** — Record-aware pipeline through
      `metaengine/sqliteengine`, not just Memory engine.
- [ ] **Soak test** — 100K events through the Record-aware pipeline, verify
      no memory leaks.
- [ ] **Benchmark `ApplyRecord` overhead** — measure `Handle()` before/after
      the `ApplyRecord` switch.

### Documentation

- [ ] **`metaengine/README.md`** — document `OnRecord`, `AutoCRUDByConvention`,
      Record stamping, `AsRecord`. Current README predates v2.
- [ ] **AGENTS.md test command** — add `./record/...` to the `go test` command
      in the Quick Reference table (currently missing the new module).

> Long-term metaengine work (`metaengine-gen` code generator, Vector/Search/
> Spatial engine backends, generic `ScanResult[T]`, operator-driven engine
> selection) lives in [ROADMAP.md](ROADMAP.md).

---

## System Package (EXPERIMENTAL — first pass + Pareto P0/P1 fixes shipped)

> The `system/` module implements the operator-configured CQRS topology.
> Driver registry wired, SQLite working through `New()`, projections E2E
> proven, MultiBus/SnapshotBackend/scream store wired, introspection real.
> Tagged `system/v4.0.0`. Remaining work is hardening and completeness.

### P1 — Hardening (makes the design production-ready)

- [ ] **Scream store: PlanDiff / PlanFingerprint / Manifest** — the scream
      store's value proposition is detecting unsafe runtime changes by diffing
      the current `SerializablePlan` against a pinned manifest. Without these,
      it's a config validator, not a scream store.
- [ ] **CommandAdapter + QueryAdapter serialization** — both adapters compile
      but need a `serializedCommand`/`serializedQuery` envelope for SQL engines
      (same pattern as `serializedEvent`).
- [ ] **`example/taskmanager` full migration to System** — metaengine.go DX
      rewrite done (372→193 lines), but the app still wires manually via
      `system.New()`. Full migration proves the consumer experience end-to-end.

### P2 — Important for completeness

- [x] **koanf YAML config** — koanf integration done: YAML + structured env
      merge (`CQRS_ENGINES__PRIMARY__DRIVER=sqlite`), eliminated YAML intermediate
      structs, backward-compatible legacy env vars. (ADR-0105)
- [x] **DuckDB/PG Transactional** — DuckDB + Postgres implement
      `Transactional` (`RunInTx`) with tx routing via `conn()`/`activeTx`.
      Compile-time assertions added. `RunTransactionalTest` in enginetest.
- [x] **Bus driver registry** — registry functional: gochannel special-case
      removed, unknown drivers error (not silent fallback). Fixed latent
      `RLock`/`Unlock` bug in `lookupBusDriver`.

---

## bbolt Storage Backend

> Full storage backend shipped: EventStore, SnapshotStore, CheckpointStore,
> KVAdapter, CommandStore, QueryStore, Backend facade. Durability tiers via
> `stack/bbolt`. `storage/bbolt` is **untagged**.

- [ ] **Tag `storage/bbolt/v4.0.0`** — implemented, in `go.work`, but no git
      tag. External consumers cannot import it.
- [ ] **Contract test suite** — `contract_test.go` exists but covers only
      smoke tests. Needs full `eventtest.TestStore*` contract coverage
      (LoadFromVersion/LoadToVersion/ReadFrom/AppendBatch/concurrency).
- [ ] **Streaming iterators** — `event.StreamingSource` (LoadStream/ReadStream)
      and `event.StreamingJournal` (streaming ReadAll/ReadFrom) not implemented.
- [ ] **OTel spans wired** — context parameters are discarded (`_ context.Context`)
      in `ReadAll`/`ReadFrom`. Tracing spans are dead code.

---

## Irohengine

> Level 2 prototype shipped with CRDT-safe operations. Three transports:
> InProcessNetwork (goroutine, no CGo), loopback (real TCP, no CGo), QUIC
> (`iroh-go` C bindings, CGo required). loopback/v4.0.0 + quic/v4.0.0 tagged.

- [ ] **Evaluate `iroh-go` C binding stability** — the QUIC transport depends
      on a third-party Go binding for Iroh (Rust). Assess upstream maintenance,
      API stability, and whether to vendor/fork. Evidence:
      `metaengine/irohengine/quic/transport.go:14`.
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

> 192 rules across 10 categories. Config presets, `--adoption`/`--scorecard`/
> `--group-by` flags, SARIF + Markdown output, self-lint mode, block-level
> suppression, metaengine-aware detection (F018-F026), drift-prevention
> meta-tests. v4.4.0 tagged.

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
      intentionally project-level. High false-positive risk for multi-module
      workspaces.

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

- [ ] **Benchmark audit for 10 skipped modules** — benchmark assertion sweep
      covered 18 files but skipped: codec, command, dispatcher, query,
      middleware, snapshot, listing, watermill, transport/http, storage/view.
      Evidence:
      `docs/status/2026-08-04_06-49_benchmark-assertions-brutal-self-review.md`.

---

## Dedup

> Clone groups reduced 69 → 65. `art-dupl` baseline golden + `nix run
.#check-duplication` gate enforce no-new-clones.

- [ ] **Review remaining 65 clone groups** — duckdb↔pgengine parity code,
      testcontainer setup patterns, table-driven test boilerplate. Some are
      intentional (cross-module isolation, table-driven); others are
      extractable (`renderTable`, `testutil/pgtest` shared module).
- [ ] **`renderTable` extraction** — `cmd/cqrs-lint/explain.go` still has
      repeated table-rendering patterns (partially extracted to
      `renderKeyTable` but not fully DRY'd).
- [ ] **`deferClose(closer)` helper** — metaengine engines have repeated
      `if c := eng.Close(); c != nil ...` patterns across 7 engines.

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

> `check-module-layers.sh` has a self-enforcing coverage guard. 77/77 modules
> covered. ADR-0046 updated to the seven-tier model.

- [ ] **Rename `FOUR-TIER-MODEL.md` → `SEVEN-TIER-MODEL.md`** — filename lies
      about content (H1 says "seven-tier" but filename says "four-tier").
- [ ] **Remove dead `EXCEPTIONS[storage]="listing"`** — listing moved to
      Layer 3, the exception is no longer needed.
- [ ] **Expand go-arch-lint to remaining modules** — only 6 modules have
      per-module go-arch-lint configs. The bash script is the enforcement
      mechanism for the rest.
- [ ] **Consider rewriting `check-module-layers.sh` as `cmd/check-layers`** —
      330 lines of bash. A Go program would be more maintainable and testable.

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
