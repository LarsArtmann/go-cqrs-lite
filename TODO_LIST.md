# TODO List

**Updated:** 2026-08-04 (system/ first pass + iroh QUIC + cqrs-lint per-module)
**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## System Package (EXPERIMENTAL — first pass shipped, wiring incomplete)

> The `system/` module implements the operator-configured CQRS topology from
> the [metaengine redesign](docs/planning/metaengine-redesign.md). 14 files,
> 2925 lines, 15 tests — types exist but critical wiring gaps prevent
> production use. Full audit:
> [design-vs-reality](docs/status/2026-08-04_22-32_metaengine-redesign-audit-design-vs-reality.md).
> Execution plan:
> [Pareto breakdown](docs/planning/2026-08-04_22-34_metaengine-system-pareto-execution-plan.md).

### P0 — Critical (blocks ALL production use)

- [ ] 🔥 **Replace `createEngine()` with `createEngineFromDriver()`** — the
      constructor at `system/constructor.go:39` calls a hardcoded switch
      supporting only `"memory"`. The entire driver registry (`RegisterDriver`,
      `lookupDriver`, `createEngineFromDriver`) is dead code. SQLite is
      completely unreachable through `system.New()` despite being fully
      implemented and tested at the metaengine level.
      Evidence: `system/constructor.go:39,216-229`, `system/driver_registry.go`.

- [ ] 🔥 **Register SQLite driver in `init()`** — open `*sql.DB` from DSN, call
      `metaengine.NewSQLiteEngine(db)`, register via `RegisterDriver("sqlite",
      factory)`. Only Memory is currently registered.
      Evidence: `system/driver_registry.go`.

- [ ] 🔥 **Auto-detect serialization for SQL engines** — `NewEventAdapter` at
      `system/constructor.go:73` never passes `WithSerialization()`. SQL-
      persisted events lose typed reconstruction. Detect non-Memory engine and
      pass the option.
      Evidence: `system/adapter_event.go` (`WithSerialization`, `serializedEvent`).

- [ ] 🔥 **SQLite-through-System integration test** — full CQRS roundtrip:
      construct System with `Driver: "sqlite"`, dispatch command, verify event
      persisted, restart simulation (new System, same DSN), verify events
      survive. Blocked by the constructor bypass above.

- [ ] 🔥 **Projection E2E test** — dispatch command → `host.Start(ctx)` →
      verify projection store updated. The projectionadapter wiring compiles
      but the full event→projection flow is unproven.
      Evidence: `system/constructor.go` (projection wiring).

- [ ] **Split `constructor.go` (369→<350 lines)** — extract projection wiring
      into `system/projections.go`. CI-enforced 350-line limit will fail.
      Evidence: `wc -l system/constructor.go`.

- [ ] **Split `adapter_event.go` (372→<350 lines)** — extract serialization
      (`serializedEvent`, `encodeEvent`, `decodeEvent`) into
      `system/adapter_event_sql.go`.
      Evidence: `wc -l system/adapter_event.go`.

- [ ] **Add `system/` to api-stability modules list + regen golden** —
      `cmd/api-stability/main.go` does not include `"system"`.
      `TestEveryGoModDirIsInModulesList` will fail. All new exported symbols
      (`NewMemorySnapshotBackend`, `NewMultiBus`, `NewCommandAdapter`,
      `NewQueryAdapter`, `WithSerialization`, `Op.StreamID`, `Op.StreamType`,
      `AtomicAppender`, `ErrVersionConflict`, etc.) need golden entries.

### P1 — High value (makes the design actually work)

- [ ] **Fix `simpleBus` handler independence** — `dispatch` chains ALL handlers
      into a single sequential chain; if one handler errors, subsequent
      handlers are skipped. Standard `event.Bus` calls each handler
      independently. Behavioral correctness issue for projections and
      side-effects. Evidence: `system/bus.go`.

- [ ] **Wire MultiBus into `New()`** — MultiBus exists and is tested in
      isolation, but the constructor always creates a single `simpleBus`.
      Fan-out to multiple publishers (D9) doesn't work through `New()`.
      Evidence: `system/multi_bus.go`, `system/constructor.go`.

- [ ] **Wire SnapshotBackend into `New()` + System lifecycle** —
      `memorySnapshotBackend` exists and is tested but not connected to the
      System lifecycle. Snapshots don't persist through System.
      Evidence: `system/snapshot.go`.

- [ ] **Fix introspection hardcoded values** — `Snapshot()` returns
      `HealthStatus: "ok"` (hardcoded), `Handlers: 0` (hardcoded). No actual
      health checks (no `db.PingContext`, no `projHost.Status()`). An admin UI
      consuming this data would display false information.
      Evidence: `system/introspection.go:77,92`.

- [ ] **Wire scream store into `New()`** — call `CheckSafety(ctx, deployment)`
      on startup; return `ErrUnsafeChange` on SCREAM-tier violations. Currently
      `New()` never calls `CheckSafety()` at all.
      Evidence: `system/scream_store.go`.

### P2 — Important for completeness

- [ ] **Scream store: PlanDiff / PlanFingerprint / Manifest** — the scream
      store's value proposition is detecting unsafe runtime changes by diffing
      the current `SerializablePlan` against a pinned manifest. Without these,
      it's a config validator, not a scream store. Design ref: §9.3–9.5.
- [ ] **koanf YAML + env config loading** — `LoadConfig(path)` has a stub
      `parseYAML` (returns nil). Only reads env vars. G2 ("deployer decides")
      is unmet.
- [ ] **Bus driver registry** — types exist, zero bus drivers registered.
      `BusConfig{Driver: "gochannel"}` does nothing.
- [ ] **Pebble/DuckDB/Postgres StreamLogBackend** — only Memory + SQLite
      implement it (2 of 5 engines). Each needs 5 stream-keyed methods. The
      design doc claims "all 5 engines" — false until implemented.
- [ ] **CommandAdapter + QueryAdapter serialization** — both adapters compile
      but need a `serializedCommand`/`serializedQuery` envelope for SQL engines
      (same pattern as `serializedEvent`).
- [ ] **Migrate example/taskmanager to System** — proves the consumer
      experience end-to-end. Depends on SQLite working through System.
- [ ] **System.Verify/Plan/Explain methods** — cross-instance consistency
      check, combined plan, human-readable explanation. Design ref: §8.3.

---

## Metaengine

> 5 engines (Memory, SQLite, Pebble, DuckDB, Postgres), 10/10 ADTs on all
> engines (Universal ADT Phase 3 shipped, ADR-0094), replication model
> (ADR-0093), persistence enum (ADR-0098), WatchTyped, SSE reconnect test,
> boundary key validation, CalibrateEngine, ReadCosts (per-read-pattern costs),
> and inspect.go extraction are all shipped. metaengine v4.4.0 tagged.

- [ ] **Postgres GIN containment indexes** — add `@>` operator support for
      JSONB path queries; currently only B-tree expression indexes are
      implemented. Needs `FilterContains`/`FilterExists` operators.
      Evidence: `metaengine/pgengine/pushdown.go`.

- [ ] **DuckDB LayoutPlanner follow-ups**
  - Add `explainScan` for planned and standard DuckDB paths.
  - Centralize planned-table helpers (`extractFields`, `jsonFieldName`,
    `quoteIdent`) duplicated between `metaengine/planned_sqlite.go` and
    `metaengine/duckdbengine/layout_planner.go`.
  - Add a DuckDB layout benchmark.
  - Add `adttest` matrix coverage for the `LayoutPlanner` capability.
  - Document the no-backfill semantics of `ApplyLayout` (existing rows in
    `meta_map` remain invisible to planned-table queries).

- [ ] **CalibrateEngine for external engines** — `calibratable` interface is
      unexported (`metaengine/reliability.go:47`); pebbleengine/duckdbengine/
      pgengine can't implement it. CalibrateEngine silently does nothing for
      these engines. Needs export as `Calibratable` + extended signature to
      accept `ReadCosts`. See
      [Read Costs problem analysis](docs/planning/2026-08-04_07-00_READ-COSTS-PER-OPERATION-VARIANCE.md#remaining-work).

- [ ] **Serialize `ReadCosts` into `SerializablePlan`** — `ReadCosts` is NOT
      in the plan JSON; plan diffing between deploys won't show what ReadCosts
      values were active. Add `read_costs` field to `SerializableQuery`.
      Evidence: `metaengine/engine.go:89` (`type ReadCosts struct`).

- [ ] **ADR for ReadCosts design** — no ADR documents the per-read-pattern cost
      model decision. Should cover: why 11 ReadPatterns → 4 cost fields, the
      conservative-margin methodology, calibration approach.

- [ ] **10M soak test verification & hardening**
  - Run `TestSoak_MemoryBounded_10M` 3× with `-race` and record variance.
  - Investigate the 10→12MB heap threshold bump (102KB/key expected?).
  - Add `TotalAlloc` tracking to the 10M variant.
  - Add engine parity soak tests (pgengine/duckdbengine/pebbleengine 1M/10M).

- [ ] **`sse.go` over 350-line CI limit** — `metaengine/sse.go` is 369 lines
      after the `Inspect()` extraction. Extract `sseMainLoop`/`forwardWithDropOld`
      into `sse_loop.go` to get under 350. Evidence: `wc -l metaengine/sse.go`.

- [ ] **Document `metaengine` watcher delete semantics** — delete notifications
      deliver the zero value of `V` after the reification fix; this contract
      should be documented in `metaengine/README.md` or `metaengine/COOKBOOK.md`.

> Long-term metaengine work (`metaengine-gen` code generator, generic
> `ScanResult[T]`, Vector/Search/Spatial engine backends, DuckDB
> columnar-native storage, Iroh distributed engine, `System` topology redesign)
> lives in [ROADMAP.md](ROADMAP.md).

---

## Irohengine

> Level 2 prototype shipped with CRDT-safe operations. Real QUIC FFI transport
> (`metaengine/irohengine/quic/`) now uses `iroh-go` C bindings for real
> networking — NOT the in-process mock. CGo required.

- [ ] **Evaluate `iroh-go` C binding stability** — the QUIC transport depends
      on `git.coopcloud.tech/decentral1se/iroh-go`, a third-party Go binding for
      Iroh (Rust). Assess upstream maintenance, API stability, and whether to
      vendor/fork. Evidence: `metaengine/irohengine/quic/transport.go:14`.
- [ ] **QUIC transport integration with `adttest.RunMatrix`** — the in-process
      mock passes the full matrix; verify the QUIC transport also passes parity
      tests (LWW resolution, PN-Counter, MapUpdate-does-not-replicate).
- [ ] **Non-CRDT op rejection on QUIC path** — verify `MapUpdate` operations
      stay local-only and are NOT sent over QUIC (would break CRDT convergence).

---

## cqrs-lint

> 186 rules across 10 categories. Config presets, `--adoption`/`--scorecard`/
> `--group-by` flags (text + JSON + Markdown output), changelog subcommand,
> self-lint mode, block-level suppression, C038-C040 (event-type mismatch/
> dead-fold-case detection), per-module feature profiles (S002/S003/S006/S007/
> C017/C036 migrated), E006 fold-aware, JSONC config loader, `explain` command,
> doctor overhaul, and `init` SHOWSTOPPER fix are shipped. v4.3.0 tagged;
> v4.4.0 pending.

- [ ] 🔥 **Publish cqrs-lint v4.4.0** — v4.3.0 tagged but post-v4.3.0 work
      (init SHOWSTOPPER fix, C038-C040 rules, scorecard + markdown, group-by
      aggregate, per-module detection, JSONC config loader, `explain` command,
      doctor overhaul, E006 fold-aware, E009 cqrs-htmx transport detection)
      remains unreleased. Also: published Nix binary is stale (v0.2.2).
      **BLOCKED on user approval**.

- [ ] 🔥 **Run cqrs-lint against real consumer projects** — validate
      false-positive rates against Kernovia, Standup-Killer, bank-sync,
      cqrs-htmx, DiscordSync, timesheets, crush-daily. This is the single
      highest-value non-coding task for cqrs-lint trustworthiness.

- [ ] **Migrate global detectors to per-module evaluation** —
      `ProfileForFile` infrastructure exists. S002, S003, S006, S007, C017,
      C036 are migrated. A015 (global mutable state), B014 (missing otel
      middleware), E008/E011 (transport detection), A009/A013 (soft-delete
      detection) still use `ctx.FeatureProfile` directly. F-series rules are
      intentionally project-level (they coach the whole project). High
      false-positive risk for multi-module workspaces.

- [ ] **Scorecard SARIF output** — scorecard already has text + JSON + Markdown
      output. SARIF could represent adoption metrics as `notifications` (not
      `results`). Worth revisiting for CI integration.

- [x] **B025 cross-package helper tracing** — DONE. The detector now scans ALL
      loaded packages (not just CQRS-importing ones) for function declarations
      and resolves cross-package helper calls via the import graph. Import
      aliases are supported. No `go/callgraph`/SSA dependency needed — the
      existing `packages.Load(NeedSyntax)` already had the syntax; the index
      just wasn't scanning it.

- [ ] **L1.5 domain severity calibration** — highest-impact open Pareto item.
      Add `DomainBias` to `FeatureProfile`, detect financial/security projects,
      escalate severity. Makes ALL 186 rules smarter instead of adding more.

- [ ] **~14 remaining Pareto backlog items** — see the
      [Pareto plan](docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md).
      Highest impact: L1.30–L1.33 deep pattern detection, L1.47–L1.51 new rule
      categories (DOC/OBS/RES/DI).

---

## SSE Consolidation

> ADR-0097 documented that both `transport/http.SSEBroker` and
> `metaengine.ServeSSE` now delegate wire-format serialization to `go-sse`
> (shipped). ADR-0091's rationale stands: do NOT merge the two implementations
> — they serve different layers (event-bus-to-client vs collection-watch).

- [ ] **Resolve metaengine SSE layer-leak (ADR-0062 violation)** —
      `metaengine/sse.go` pulls `go-sse` + `dedup` as **production** deps into
      a module whose core is documented as "zero production deps (stdlib +
      `database/sql` only)" (ADR-0062). Three options: (a) move SSE to
      `transport/http` behind a source adapter, (b) split into
      `metaengine/sse` sub-module with own go.mod, (c) amend ADR-0062 to
      acknowledge the exception. Needs a decision + ADR. See
      [2026-08-03 status report](docs/status/2026-08-03_21-43_sse-layering-and-watermill-fitness.md).
      **BLOCKED on user input**: is `metaengine.ServeSSE` stable public API
      that external consumers import?

- [ ] **Measure SSE loop duplication** — run `art-dupl` between
      `transport/http/sse*.go` and `metaengine/sse.go` to quantify actual
      shared logic (heartbeat, timeout, flush, drop-old, replay handoff).
      Input to whether a shared `sseloop` internal package is worth
      extracting. Decision required: the two implementations stream
      different data models (`event.Event` vs `SeqValue[V]`) — do NOT force
      a shared source interface. See
      [2026-08-03 status report](docs/status/2026-08-03_21-43_sse-layering-and-watermill-fitness.md).

---

## Code Quality

- [ ] **Encryption double-clone** — `encryption/crypto_helpers.go:66`:
      `evt.Metadata().Clone()` is a redundant double-clone (`Metadata()`
      already returns a clone). Wasted allocation on every decrypt hot path.
      Remove the extra `.Clone()`.

- [ ] **Metadata immutability** — `command.Metadata` and `query.Metadata`
      still use `EnsureCustom()` (mutable map access) instead of `WithCustom()`
      (functional). Decision needed: make Metadata fully immutable (all value
      receivers, `WithCustom` instead of `EnsureCustom`). Currently suppressed
      with `//nolint:recvcheck`.

- [ ] **Fix flaky `idempotency/kvstore` TTL test** — has blocked the verify
      gate multiple times. Needs `testutil.RaceEnabled` threshold or longer
      TTL.

- [ ] **Benchmark audit for 10 skipped modules** — benchmark assertion sweep
      covered 18 files but skipped: codec, command, dispatcher, query,
      middleware, snapshot, listing, watermill, transport/http, storage/view.
      These likely have the same `_, _ =` result-discarding pattern that was
      fixed in the other modules. Evidence:
      `docs/status/2026-08-04_06-49_benchmark-assertions-brutal-self-review.md`.

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).

- [x] **Push go-retry + go-idempotency to GitHub** — DONE. Both repos pushed
  with annotated tags (go-retry v0.1.0, go-idempotency v0.1.0 + v0.1.1).
  go-cqrs-lite `retry/go.mod` + `idempotency/go.mod` use real versioned
  `require` directives (no local replaces). Sub-modules (`kvstore`, `sqlstore`)
  also resolved: they build, test, and `go mod verify` clean standalone
  (GOWORK=off) against tagged kv/v4.2.0, codec/v4.2.0, idempotency/v4.2.0.

- [ ] **Tag `stack/mysql/v4`** — source is stable but tag doesn't exist.

- [ ] **Tag `system/v4`** — new module, no tag exists yet. Blocked on P0
      wiring fixes (constructor bypass, file-size limit).

- [ ] **Pin GitHub Actions to commit SHAs** — 72+ unpinned actions
      (supply-chain risk).

- [ ] **Update CONTRIBUTING.md** — JSONC config loader, `explain` subcommand,
      `scorecard` feature, and `--group-by` flag are undocumented in the
      contributor guide. Consumers learn about these only from the README.

- [ ] **Update AGENTS.md module list** — now 68 `go.mod` files (was 65).
      `system/` and `metaengine/irohengine/quic/` are missing from the module
      table, build command, and test command.

---

## Integration Test Infrastructure

> Nix-based integration test infrastructure shipped: ephemeral PG, NixOS VM
> tests (PG+MySQL), nspawn MySQL (M14, ~15s), projectionhost PG crash-restart
> (M43), scheduling/sqlstore durable timers (M44).

- [ ] **macOS verification of ephemeral PG** — script claims cross-platform but
      never tested on Darwin. (M34)
- [ ] **Cache ephemeral PG data dir** — skip `initdb` on repeated runs. (M35)
- [ ] **Performance profiling: ephemeral PG vs testcontainers** — measure
      speedup and document. (M36)
- [ ] **Explore `nixos-container` as lighter-weight VM alternative** (M37)
- [ ] **DuckDB CGo VM test** — hermetic DuckDB testing with GCC in VM. (M38)
- [ ] **SQLite WAL concurrency VM test** — concurrent access patterns. (M39)
- [ ] **Turso sync VM test** — real libSQL server. (M40)
- [ ] **Go test binaries inside QEMU VM** — deeper coverage. (M41)
- [ ] **Pebble backup/restore lifecycle VM test** (M42)
- [ ] **Contract test suite across ALL backends in VMs** — SQLite, PG, MySQL,
      DuckDB simultaneously. (M46)
- [ ] **Ephemeral Redis/NATS for future integration tests** — Watermill adapter
      testing with real brokers. (M47)
- [ ] **`scripts/test-integration.sh` aggregator** — auto-detect best strategy
      (ephemeral, VM, or testcontainers). (M48)

---

## Deferred Debt (ADR-committed)

Four items explicitly committed to in the 2026-08-03 ADR review as "the next
real roadmap." Each has a clear ADR with rationale.

- [ ] **Ghost bus removal** (ADR-0028) — delete `memory/bus.go`,
      `memory/command_bus.go`, `storage/pg_bus.go`. Largest blast radius — audit
      ALL consumer repos first.
- [ ] **Metadata aliases completion** (ADR-0031) — `command.Metadata` /
      `query.Metadata` → standalone structs (currently repointed aliases).
- [ ] **Extract `retry/` → `go-retry`** (ADR-0064) — standalone repo created +
      tagged, needs push + update go-cqrs-lite replace directive to published tag.
- [ ] **Extract `idempotency/` → `go-idempotency`** (ADR-0065) — standalone repo
      created + tagged, needs push. Sub-modules (`kvstore`, `sqlstore`) blocked
      on kv/ and codec/ dependencies.

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
  ORM creep. Principle #1: "Library, not framework."
- **Unify VersionedStore + VersionedSeekableJournal** — different interfaces. YAGNI.
- **RollupSpec / RollupProjection** — premature abstraction. `sink.Increment` is
  the composable primitive. See analytics rollup review.
- **Redis adapter** — see ROADMAP Non-Goals (ValKey/NATS/Kafka preferred).
- **`idempotency.RefreshTTL(ctx, key, ttl)`** — dropped 2026-07-26 (YAGNI).
  Sliding window is unsafe (unbounded TTL under retry storms).
- **Centralized cross-module error-wrapping helper** — ADR-0069 decided:
  per-module helpers, capped at 3 modules.
- **Move 3-way idempotency contract test to `integration/`** — dropped
  2026-07-26. Would add 3 new direct deps to integration/.
- **Stack preset `stackpreset` builder** — dropped 2026-07-26. ~45 lines of
  trivial Go idiom; real SQL consolidation lives in `stack/sqlopt`.
- **Test infra helpers (catalogtest, storagetest, codectest)** — dropped
  2026-07-26. `idtest`, `eventtest`, `cattest` already cover all real needs.
- **`filterDetectors` extraction in cqrs-lint** — dropped 2026-07-27
  (over-engineering).
- **Split `event/` module** — 27 importers, real cohesion. Decided in v4.
- **Extract metaengine as standalone project** — → ROADMAP.
- **`FluentBuilder` in metaengine** — deleted (ghost code, broken doc example).
  See ADR-0077.
- **C033 middleware-chain awareness** — HIGH risk. Data-flow tracing through
  `.Use()` chains is fragile and silences real bugs. Declined 2026-08-02.
- **A032 framework deserialization awareness** — HIGH risk. Couples linter to
  specific frameworks (Huma, Gin) that rot. Declined 2026-08-02.
- **A017/B025 stream-length awareness** — MEDIUM risk. "1-event-per-stream"
  detection is an unreliable heuristic. Declined 2026-08-02.
- **D005 multi-module version detection** — LOW risk. Making it smarter risks
  the simple case. Declined 2026-08-02.
- **Merge the two SSE implementations** — different semantics (collection-watch
  vs bus-to-client). ADR-0091 rationale is correct. Declined 2026-08-03.
- **`systemd-nspawn` container type (near-term)** — Implemented in M14.
  `containers.machine` in `runNixOSTest` is stable in nixpkgs. Requires
  `uid-range` system feature + `auto-allocate-uids` on the host.
  See `scripts/enable-nspawn-support.sh` for one-shot setup.

---

_Long-term direction lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
