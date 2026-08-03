# TODO List

**Updated:** 2026-08-03
**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Benchmark Trust (cross-cutting — highest leverage)

> The ADR review session flagged this as the single highest-leverage next move.
> 29 of 43 benchmarks discard results; DuckDB and Postgres cost constants have
> zero empirical backing.

- [ ] 🔥 **Add correctness assertions to 29 unasserted benchmarks** — without
      assertions, a benchmark can silently measure empty stores (the ADR-0090
      lesson: the metaengine benchmark measured empty counters for sessions).
- [ ] 🔥 **Create DuckDB + Postgres engine benchmarks** — 0 exist today. Cost
      constants for these engines are completely fabricated.

---

## Metaengine

> 5 engines (Memory, SQLite, Pebble, DuckDB, Postgres), 10/10 ADTs on all
> engines (Universal ADT Phase 3 shipped, ADR-0094), replication model
> (ADR-0093), WatchTyped, SSE reconnect test, boundary key validation, and
> CalibrateEngine are all shipped. metaengine v4.4.0 tagged.

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
      unexported; pebbleengine/duckdbengine/pgengine can't implement it.
      CalibrateEngine silently does nothing for these engines.

- [ ] **10M soak test verification & hardening**
  - Run `TestSoak_MemoryBounded_10M` 3× with `-race` and record variance.
  - Investigate the 10→12MB heap threshold bump (102KB/key expected?).
  - Add engine parity soak tests (pgengine/duckdbengine/pebbleengine 1M/10M).

- [ ] **Document `metaengine` watcher delete semantics** — delete notifications
      deliver the zero value of `V` after the reification fix; this contract
      should be documented in `metaengine/README.md` or `metaengine/COOKBOOK.md`.

> Long-term metaengine work (`metaengine-gen` code generator, generic
> `ScanResult[T]`, Vector/Search/Spatial engine backends, DuckDB
> columnar-native storage, Iroh distributed engine) lives in
> [ROADMAP.md](ROADMAP.md).

---

## cqrs-lint

> 185 rules across 10 categories (correctness 39, API 31, boilerplate 28,
> adoption 21, architecture 17, consistency 16, security 10, performance 9,
> testing 8, version 6). Config-level disabling, block-level suppression,
> import-alias resolution, self-lint mode, TLS detection, `--adoption` flag,
> changelog subcommand, and config presets are shipped. v4.3.0 tagged.

- [ ] 🔥 **Publish cqrs-lint v4.4.0** — v4.3.0 tagged but post-v4.3.0 fixes
      (init SHOWSTOPPER fix, E009 cqrs-htmx transport detection) remain
      unreleased. Also: published Nix binary is stale (v0.2.2).
      **BLOCKED on user approval**.

- [ ] 🔥 **Run cqrs-lint against real consumer projects** — validate
      false-positive rates against Kernovia, Standup-Killer, bank-sync,
      cqrs-htmx, DiscordSync, timesheets. This is the single highest-value
      non-coding task for cqrs-lint trustworthiness.

- [ ] **C036 library function recognition** — fires on `storage.OpenSQLite*`
      functions that ARE from the library. Should recognize `go-cqrs-lite/`
      imports as library-internal. Reported by 4 of 5 consumers.

- [ ] **E009/E016 cqrs-htmx awareness** — linter doesn't recognize
      `cqrs-htmx` as satisfying transport (E009) or health-check (E016)
      requirements. Reported by timesheets + crush-daily.

- [ ] **D007 auto-fix** (`event.NewEvent` → `event.New`) — `--fix`
      infrastructure exists; needs payload-type heuristic.

- [ ] **F013/C009/C016 feature-profile fixes** — recognize `cqrshtmx.New` as
      transport (F013), `New*` constructor + panic as must-pattern (C009),
      exempt `context.Background()` in graceful-shutdown paths (C016).

- [ ] **~14 remaining Pareto backlog items** — see the
      [Pareto plan](docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md).
      Highest impact: L1.29 event-type string typo detection, L1.30–L1.33 deep
      pattern detection, L1.47–L1.51 new rule categories (DOC/OBS/RES/DI).

---

## SSE Consolidation

> The 2026-08-03 ADR review discovered `go-sse` exists as a standalone SSE
> library. `go-cqrs-lite` reimplements SSE wire format in TWO places instead
> of consuming it. ADR-0097 documents the three-repo finding. ADR-0091's
> rationale was written as if `go-sse` didn't exist. Do NOT merge the two SSE
> implementations — different semantics (ADR-0091). Instead, both should
> consume `go-sse` internally.

- [ ] **SSE refactor: `transport/http.SSEBroker`** → consume `go-sse`
      internally (~300 LOC dedup). Preserve external API (filter, transform,
      budget, backfill).
- [ ] **SSE refactor: `metaengine.ServeSSE`** → consume `go-sse` (~200 LOC
      dedup). Preserve Watcher-based semantics.
- [ ] **Add SSE decision matrix to SKILL.md** — route consumers between
      raw-events SSE vs read-model SSE.

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).

- [ ] **Pin GitHub Actions to commit SHAs** — BuildFlow flagged 72+ unpinned
      actions (supply-chain risk).

- [ ] **gopls hint cleanup in cmd/cqrs-lint** — 6 `infertypeargs` + 1
      `writestring` hints remain.

---

## Integration Test Infrastructure

> Session 2 (2026-08-03) built and verified the Nix-based integration test
> infrastructure: ephemeral PG, NixOS VM tests (PG+MySQL), VM launcher scripts,
> CI integration, and ADR-0095.

- [ ] **systemd-nspawn container type for MySQL VM** — could make VM test 10x
      faster (~131s → ~15s). The NixOS test driver supports `NspawnMachine`.
      Needs prototyping. (M14)
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
- [ ] **`projectionhost` crash-restart PG integration test** — verify
      checkpoint replay after crash. (M43)
- [ ] **`scheduling` durable timers across restarts test** — timer survives
      process restart. (M44)
- [ ] **Contract test suite across ALL backends in VMs** — SQLite, PG, MySQL,
      DuckDB simultaneously. (M46)
- [ ] **Ephemeral Redis/NATS for future integration tests** — Watermill adapter
      testing with real brokers. (M47)
- [ ] **`scripts/test-integration.sh` aggregator** — auto-detect best strategy
      (ephemeral, VM, or testcontainers). (M48)

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
- **`systemd-nspawn` container type (near-term)** — not stable in nixpkgs.
  Research only for now.

---

_Long-term direction lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
