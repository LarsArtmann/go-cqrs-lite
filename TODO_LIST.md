# TODO List

**Updated:** 2026-08-04
**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Metaengine

> 5 engines (Memory, SQLite, Pebble, DuckDB, Postgres), 10/10 ADTs on all
> engines (Universal ADT Phase 3 shipped, ADR-0094), replication model
> (ADR-0093), WatchTyped, SSE reconnect test, boundary key validation,
> CalibrateEngine, ReadCosts (per-read-pattern costs), and inspect.go extraction
> are all shipped. metaengine v4.4.0 tagged.

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

## cqrs-lint

> 186 rules across 10 categories. Config presets, `--adoption`/`--scorecard`/
> `--group-by` flags, changelog subcommand, self-lint mode, block-level
> suppression, C038-C040 (event-type mismatch/dead-fold-case detection),
> per-module feature profiles, and `init` SHOWSTOPPER fix are shipped.
> v4.3.0 tagged; v4.4.0 pending.

- [ ] 🔥 **Publish cqrs-lint v4.4.0** — v4.3.0 tagged but post-v4.3.0 work
      (init SHOWSTOPPER fix, C038-C040 rules, scorecard, group-by aggregate,
      per-module detection, JSONC config loader, `explain` command, doctor
      overhaul, E009 cqrs-htmx transport detection) remains unreleased.
      Also: published Nix binary is stale (v0.2.2).
      **BLOCKED on user approval**.

- [ ] 🔥 **Run cqrs-lint against real consumer projects** — validate
      false-positive rates against Kernovia, Standup-Killer, bank-sync,
      cqrs-htmx, DiscordSync, timesheets, crush-daily. This is the single
      highest-value non-coding task for cqrs-lint trustworthiness.

- [ ] **Scorecard follow-ups**
  - Eliminate category-priority split brain (`categoryPriorityFor` in
    `scorecard.go` duplicates `ModuleEntry.CategoryPriority()` in catalog).
  - Render `Evidence` field in text output (show which import path triggered
    detection).
  - Expand catalog: `middleware`, `storage`, `stack/memory`, `scenario` are
    adoptable but excluded.
  - Add `--scorecard-threshold N` CI gate flag (exit non-zero below N%).
  - Add SARIF + markdown output formats.

- [ ] **Doctor/explain test coverage** — doctor command was completely
      rewritten, `explain.go` is 468 lines — both have **zero** unit tests.
      Refactor `renderDoctor*` to accept `io.Writer`, add output assertions.

- [ ] **Migrate global detectors to per-module evaluation** —
      `ProfileForFile` infrastructure exists but only `C017` uses it. The
      other 26 global `FeatureProfile` reads still use the primary profile,
      not per-module. High false-positive risk for multi-module workspaces.

- [ ] **`commentTextStart` multi-line string literal bug** — the block
      suppression parser resets state per line; a raw string literal spanning
      multiple lines causes false matches. Fix with `go/scanner` or carry
      string-literal state across lines. Evidence: `cmd/cqrs-lint/run.go`.

- [ ] **B025 cross-package helper tracing** — only same-package helpers are
      traced. Cross-package wiring functions (e.g. `pkg.helper(...)`) are
      invisible. Needs import-graph tracing via `golang.org/x/tools/go/callgraph`.

- [ ] **JSONC trailing comma support** — the `stripJSONComments` parser does
      not support trailing commas (allowed by JSONC spec). Edge case for
      hand-edited `.cqrs-lint.json` files.

- [ ] **~14 remaining Pareto backlog items** — see the
      [Pareto plan](docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md).
      Highest impact: L1.29 event-type string typo detection, L1.30–L1.33 deep
      pattern detection, L1.47–L1.51 new rule categories (DOC/OBS/RES/DI).

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

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).

- [BLOCKED] **Push go-retry + go-idempotency to GitHub** — repos created +
  annotated tags cut locally, but not pushed. go-cqrs-lite go.mod still uses
  `replace` directives pointing to local paths. Sub-modules (`kvstore`,
  `sqlstore`) blocked on kv/ and codec/ dependency complexity.

- [ ] **Tag `stack/mysql/v4`** — source is stable but tag doesn't exist.

- [ ] **Pin GitHub Actions to commit SHAs** — 72+ unpinned actions
      (supply-chain risk).

- [ ] **Regenerate API-stability golden** — `docs/api_surface.txt` at 3186
      exports; C040/C039/scorecard/group-by added exported symbols since last
      golden regen. Run `cd cmd/api-stability && GOWORK=off go run main.go -update`.

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
- **`systemd-nspawn` container type (near-term)** — not stable in nixpkgs.
  Research only for now.

---

_Long-term direction lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
