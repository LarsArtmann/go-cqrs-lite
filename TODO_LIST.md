# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Correctness Bugs

- [ ] 🔥 **Add length-mismatch guard to `DecodeFloatResults`** —
      `metaengine/scan.go:53` indexes `raws[i]` inside a `for i, s := range specs`
      loop with no bounds check. Will panic (`index out of range`) if
      `len(raws) < len(specs)`. Add `if len(raws) < len(specs) { return nil, fmt.Errorf(...) }`.
      _(Effort: S)_
      _(Source: `docs/status/2026-08-08_10-36_metaengine-aggregate-test-coverage-fill.md`)_
- [ ] 🔥 **Fix `context.Background()` in taskmanager handlers** —
      `example/taskmanager/handlers.go` (10 handlers, lines 118–175): every
      `RegisterCommand` handler discards the incoming `ctx` and calls
      `system.Execute(context.Background(), ...)`. Request-level timeouts,
      tracing spans, and correlation IDs are lost. Pass `ctx` instead.
      _(Effort: S)_
      _(Source: `docs/status/2026-08-07_22-12_system-p1-hardening-scream-store-serialization-taskmanager-migration.md`)_
- [ ] **Route DuckDB `plans` map reads through `lookupPlan()`** —
      `metaengine/duckdbengine/aggregations.go` (lines 173, 263, 384, 497, 655)
      and `pushdown.go:33` read `e.plans[col]` directly, bypassing the
      `layoutMu` RWMutex added to fix the race. Consolidate through the locked
      `lookupPlan()` helper.
      _(Effort: M)_
      _(Source: `docs/status/2026-08-08_08-34_metaengine-v2-coverage-gaps-duckdb-race-fix.md`)_
- [ ] **Delete `mustSQLiteEngine` zombie test** —
      `metaengine/concurrent_gaps_test.go:188` opens a SQLite DB but returns
      `NewMemoryEngine()` (gopls flags `impossible condition: nil != nil`). The
      opened DB leaks. Tests using this helper test the memory engine, not
      SQLite. Fix or delete.
      _(Effort: S)_
      _(Source: `docs/status/2026-08-08_02-50_es-native-graph-status-and-graphbackend-cleanup.md`)_
- [ ] **Delete `_skipped_sqlite_test_*` zombie functions** —
      `metaengine/features2_test.go:330,383`. Two underscore-prefixed dead
      functions with stale `sql.Open` code. Delete.
      _(Effort: S)_
      _(Source: `docs/status/2026-08-07_22-24_metaengine-v2-publishability-hardening.md`)_

---

## cqrs-lint

- [ ] 🔥 **Run cqrs-lint against real consumer projects** — validate
      false-positive rates against 8 identified repos (Kernovia,
      Standup-Killer, bank-sync, cqrs-htmx, DiscordSync, timesheets,
      crush-daily, KeyHolderAI). The linter has 192 rules but zero real-world
      false-positive data.
      _(Effort: L)_
- [ ] **Build type-checking test helper** — `BuildContextWithTypes` needed
      for testing type-aware rules (C023, C001 `Begin(false)` generalization).
      Current `BuildContextFromSource` provides empty `TypesInfo` maps, making
      type-aware paths untestable.
      _(Effort: M)_
- [ ] **Self-lint CI: tighten severity gate** — the CI job passes on
      warnings; `init.go:69` has a C025 warning (`fmt.Errorf` without `%w`).
      Either suppress inline or add `--min-severity warning`.
      _(Effort: S)_
- [ ] **10 genuinely-missing rules** — identified during DOC/OBS/RES/DI
      category evaluation. 80–90% of proposed patterns already covered by
      scattered rules. These are absent:
  - [ ] RES: Missing retry middleware — B008 detects manual retry, but no rule
        flags a bus/dispatcher that lacks retry entirely.
  - [ ] RES: Circuit breaker absence — zero circuit-breaker detection.
  - [ ] RES: Missing DLQ config — no rule detects projectionhost without
        dead-letter handling.
  - [ ] DOC: Stale catalog entries — reverse of E004.
  - [ ] DOC: AsyncAPI/OpenAPI freshness — F002 checks catalog usage but not
        generated-doc freshness.
  - [ ] OBS: Missing OTel SDK init — F003/B014 detect usage, not proper setup.
  - [ ] OBS: Missing `slog.SetDefault` — no structured-logging setup detection.
  - [ ] OBS: Missing span creation around handlers.
  - [ ] DI: Optimistic concurrency / expected-version check on Save/Append.
  - [ ] DI: Missing append-stream version precondition at store level.
        _(Source: `docs/status/2026-08-08_02-28_cqrs-lint-backlog-triage.md`)_
- [ ] **Tag cqrs-lint v4.6.0** — after CI self-lint job + remaining
      false-positive fixes are shipped. Version constant must match latest tag.
      _(Effort: S)_

---

## Metaengine v2 — Remaining Gaps

> Metaengine v2 is feature-complete: `record/` module, 9 engines, Record-aware
> folds, auto-projection, tombstone deprecation, GraphBackend cleanup, aggregate
> pushdown (5 interfaces on DuckDB/SQLite/PG). ADR-0120 written. All 14 tags
> created locally and pushed.

- [ ] **Dgraph engine: test against real Dgraph** — all `t.Skipf("Dgraph not
    available")` paths were taken. DQL queries, JSON mappings, upsert
      conditions completely unverified. Also: DQL injection risk (manual string
      interpolation via `dqlString()` — should use `QueryWithVars`), missing
      MultimapBackend/LogBackend/SnapshotBackend.
      _(Effort: L)_
      _(Source: `docs/status/2026-08-07_00-41_dgraph-metaengine-implementation.md`)_
- [ ] **Fix stale pebbleengine README** — claims GraphBackend support but
      pebbleengine does not implement it (removed during GraphBackend cleanup).
      `metaengine/pebbleengine/README.md:35`.
      _(Effort: S)_
- [ ] **Soak test for record-aware pipeline** — 100K events through
      `event.AsRecord()` → `projectionadapter.Handle()`. Verify memory growth
      stays bounded. No data exists beyond correctness assertions.
      _(Effort: M)_
      _(Source: `docs/status/2026-08-07_00-42_metaengine-v2-hardening-execution-status.md`)_
- [ ] **OTel span attributes from Record** — add `rec.StreamID`, `rec.Version`,
      `rec.Type` to `projectionadapter.Handle()` spans for traceability.
      _(Effort: S)_
- [ ] **Add `LayoutPlanApplier` support to SQLite engine** — SQLite only
      supports plans at construction time (`NewPlannedSQLiteEngine`); DuckDB
      supports post-construction registration (`ApplyLayoutPlan`).
      _(Effort: M)_
      _(Source: `docs/status/2026-08-08_09-27_metaengine-v2-coverage-gaps-and-aggregate-followup.md`)_

---

## Irohengine / Replicated Engine

- [ ] **Add `WithClock` option to `replicatedEngine`** — `MapSet` uses
      `time.Now()` for LWW timestamps (engine.go:136). Injectable clock
      eliminates timing assumptions in convergence tests.
      _(Effort: M)_
- [ ] **Add connection pooling to QuicTransport** — each `Publish` opens a new
      BiStream; reusing streams would reduce latency under high throughput.
      _(Effort: M)_
      _(Source: `docs/status/2026-08-08_02-50_irohengine-quic-parity-and-flake-fixes.md`)_
- [ ] **Add MapDelete LWW convergence test** — only MapSet LWW is tested;
      MapDelete convergence is unverified.
      _(Effort: S)_
- [ ] **Add graceful shutdown test** — verify in-flight ops complete during
      `Close()`.
      _(Effort: S)_

---

## Code Quality / Dedup

- [ ] **Per-module `.golangci.yml` split** — golangci-lint v2 `config-dirs`
      would give each module ownership of its own exclusions. The monolithic
      config is documented but sprawls across 30+ blocks.
      _(Effort: L)_
- [ ] **Extend `deferClose` to pebble production code** — 12
      `defer func() { _ = x.Close() }()` sites remain in `adapter.go`,
      `checkpoint.go`, `command_read.go`, `command_store.go`, `helpers.go`,
      `iteration.go`, `journal.go`, `query_read.go`, `save.go`. Requires
      promoting the helper to production code or creating shared
      `storage/internal/closeutil`.
      _(Effort: M)_
- [ ] **Deduplicate `deferClose` helper** — duplicated 3x across
      `storage/pebble/defer_close_test.go`, `defer_close_ext_test.go`,
      `storage/bbolt/defer_close_test.go`. Consolidate or accept the per-module
      idiom.
      _(Effort: S)_
- [ ] **Update `event/event_metadata_test.go:82` doc comment** — calls
      `event.EnsureCustom(&m)` but doc comment was not updated to match the
      backward-compat intent pattern used in `event/customdata_test.go`.
      _(Effort: S)_
- [ ] **Audit remaining `EXCEPTIONS` entries** — only
      `EXCEPTIONS[storage]` was checked and removed. The other ~10 entries
      (event, schema, snapshot, decider, query, command, listing,
      projectionhost, transport/http, metaengine) were not verified for dead
      rules. See `scripts/check-module-layers.sh`.
      _(Effort: M)_

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must
  v0.1.2).
- [ ] **Pin GitHub Actions to commit SHAs** — 72+ unpinned actions use mutable
      `@vN` tags (supply-chain risk). `actions/checkout@v4`,
      `DeterminateSystems/nix-installer-action@v17`, etc.
      _(Effort: M)_
- [ ] **Add `--fail-on-stale-suppressions` CI gate** — prevents stale
      `//cqrs-lint:ignore` directives from accumulating.
      _(Effort: S)_
- [ ] **Add CI check for API-version drift** — verify every exported symbol in
      a tagged module exists at that tag. Catches the `WithCustom`/`event/v4.3.0`
      class of drift before vulncheck fails.
      _(Effort: M)_
- [ ] **Add calibration benchmark regression baseline** — metaengine calibration
      benchmarks should run in CI and fail if cost constants drift >3× from
      baseline. Currently 0 of 43 benchmarks have CI regression tracking.
      _(Effort: M)_
      _(Source: `ROADMAP.md` Theme 8)_
- [ ] **Add `duckdb-vm` and `turso-vm` to CI `nixos-vm-tests` job** — wired as
      flake checks but not in dedicated CI job.
      _(Effort: S)_
      _(Source: `docs/status/2026-08-08_10-18_m35-m48-integration-test-infrastructure.md`)_

---

## Integration Test Infrastructure

- [ ] **macOS verification of ephemeral PG** — `scripts/ephemeral-pg.sh` claims
      cross-platform but was never tested on Darwin. (M34)
      _(Effort: M)_
- [ ] **Write actual Redis/NATS integration tests** — `ephemeral-redis.sh` and
      `ephemeral-nats.sh` exist but no tests use them. Watermill Redis Streams
      and NATS JetStream roundtrips untested.
      _(Effort: M)_
      _(Source: `docs/status/2026-08-08_10-18_m35-m48-integration-test-infrastructure.md`)_
- [ ] **Add bbolt backup/restore test** — Pebble has `backup_lifecycle_test.go`;
      bbolt should have equivalent coverage.
      _(Effort: S)_

---

## Layer Enforcement

- [ ] **Delete stale FOUR-TIER-MODEL.d2/.svg artifacts** — the `.md` was
      renamed to `SEVEN-TIER-MODEL.md` but the `.d2` and `.svg` diagram files
      in `docs/architecture-understanding/` still carry the old name.
      _(Effort: S)_
- [ ] **Add intra-module architecture config for `cmd/cqrs-lint`** — 16
      production sub-packages (`pkg/analyzer`, `pkg/rules`, etc.) with no
      intra-module architecture enforcement. Only `storage/` and `catalog/`
      have meaningful multi-package configs today.
      _(Effort: M)_
- [ ] **Consider rewriting `check-module-layers.sh` as `cmd/check-layers`** —
      348 lines of bash. A Go program would add testability but the script is
      stable and only runs in CI. Defer until significantly more complex.
      _(Effort: L)_

---

## Declined / Rejected (do not re-litigate)

> Full rationale in the linked ADRs/reviews.

- **Wire `#verify-parallel` into CI** — declined 2026-07-29. CI already has a
  per-module matrix strategy that provides better isolation.
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Use
  `RelationalProjection` (junction tables). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
- **Redis adapter** — the author is not a fan of Redis. See ROADMAP Non-Goals.
