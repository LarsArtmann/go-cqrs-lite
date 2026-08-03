# TODO List

**Updated:** 2026-08-02
**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Metaengine

> 5 engines (Memory, SQLite, Pebble, DuckDB, Postgres), 10 ADTs, pushdown +
> layout planning, rule pipeline, materialize-vs-replay, VersionedStorage,
> streaming, and watcher/SSE delivery are shipped. Remaining work is
> verification, typed APIs, and advanced indexing.

- [ ] 🔥 **10M soak test verification & hardening**
  - Add a 100K-event fast smoke variant that runs even when `SOAK_SKIP_10M=1`
    (`metaengine/soak_10m_test.go`).
  - Document `SOAK_SKIP_10M` in `AGENTS.md` / `CONTRIBUTING.md`.
  - Add `runtime.MemStats.TotalAlloc` delta measurement in addition to heap.
  - Run `TestSoak_MemoryBounded_10M` 3× with `-race` and record variance.
  - Evidence: `docs/status/2026-08-02_19-47_10M-soak-test.md`.

- [ ] **Watcher typed-channel design** — `Watcher[V]` still exposes `chan any`,
      which forces engine-specific reification in `metaengine/dx.go:163`. The
      SQLite silent-drop bug is fixed via `reifyWatcherValue`, but a typed channel
      design would eliminate the runtime type assertion entirely. Evidence:
      `metaengine/dx.go:163`, `metaengine/watcher_typesafe_test.go:252`.

- [ ] **SSE + SQLite Last-Event-ID reconnect test** — verify `ServeSSE` replay
      works end-to-end with the SQLite-backed `WatchWithSeq` path after the watcher
      reification fix. Evidence: `metaengine/sse_replay.go:128`,
      `docs/status/2026-08-02_19-58_metaengine-watcher-reification-fix.md`.

- [ ] **Boundary keys-type validation at Store boundary** — `query.keyType` is
      enforced during fold registration (`metaengine/fold_classify.go:86`) but not
      when a caller passes a key directly to `Store.Execute`/`ExecuteTyped`. Add a
      boundary check that returns `ErrKeyTypeMismatch`. Evidence:
      `metaengine/execute.go`, `docs/planning/2026-08-01_19-40_metaengine-data-model-refactor.md:271`.

- [ ] **Postgres GIN containment indexes** — add `@>` operator support for JSONB
      path queries; currently only B-tree expression indexes are implemented.
      Evidence: `metaengine/pgengine/pushdown.go`, `metaengine/pgengine/engine.go`.

- [ ] **DuckDB LayoutPlanner follow-ups**
  - Add `explainScan` for planned and standard DuckDB paths (`metaengine/sqlite_engine.go`
    has it; DuckDB returns placeholder).
  - ~~Verify the `coerceForColumn` fix resolves float truncation for planned columns~~
    DONE: `sqlTypeOf` now maps float64→DOUBLE (not REAL), `coerceForColumn` handles DOUBLE/REAL/FLOAT.
    Regression tests: `TestDuckDBEngine_ColumnarDoublePrecision`, `TestDuckDBEngine_ColumnarAggregation`
    ([ADR-0092](docs/adr/0092-duckdb-columnar-native-storage.md)).
  - Centralize planned-table helpers (`extractFields`, `jsonFieldName`,
    `quoteIdent`, `plansColumnCompatible`) that are currently duplicated between
    `metaengine/planned_sqlite.go` and `metaengine/duckdbengine/layout_planner.go`.
  - Document the no-backfill semantics of `ApplyLayout` (existing rows in
    `meta_map` remain invisible to planned-table queries).
  - Add a DuckDB layout benchmark.
  - Add `adttest` matrix coverage for the `LayoutPlanner` capability.
  - Evidence: `docs/status/2026-08-02_19-47_DuckDB-LayoutPlanner.md`.

- [ ] **Document `metaengine` watcher delete semantics** — delete notifications
      now deliver the zero value of `V` after the reification fix; this contract change
      should be documented in `metaengine/README.md` or `metaengine/COOKBOOK.md`.
      Evidence: `docs/status/2026-08-02_19-58_metaengine-watcher-reification-fix.md`.

> Long-term metaengine work (`generic ScanResult[T]`, `metaengine-gen` code
> generator, Vector/Search/Spatial engine backends, DuckDB columnar-native storage)
> lives in [ROADMAP.md](ROADMAP.md).

---

## cqrs-lint

> 181 rules across 10 categories. Config-level disabling, block-level suppression,
> A033/C037, import-alias resolution, and self-lint mode are shipped. Remaining
> work is validation and finishing the Pareto backlog.

- [ ] 🔥 **Run cqrs-lint against real consumer projects** — validate false-positive
      rates against Kernovia, Standup-Killer, bank-sync, cqrs-htmx, DiscordSync.
      This is the single highest-value non-coding task for cqrs-lint trustworthiness.
      Evidence: `docs/status/2026-08-02_16-29_cqrs-lint-rules-and-metaengine-verification.md:162`.

- [ ] **~14 remaining backlog items** — see the
      [Pareto plan](docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md).
      Highest impact: L1.29 event-type string typo detection, L1.30–L1.33 deep pattern
      detection, L1.18 config inheritance, L1.47–L1.51 new rule categories (DOC/OBS/RES/DI).

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).
- [BLOCKED] **Push `stack/duckdb/v4.0.0`, `metaengine/pgengine/v4.0.0`,
  `metaengine/duckdbengine/v4.0.0` tags** — all three tags created locally but
  not pushed (per safety rules). Consumers get 404 from Go proxy until pushed.

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

---

## Integration Test Infrastructure

> Session 2 (2026-08-03) built and verified the Nix-based integration test
> infrastructure: ephemeral PG, NixOS VM tests (PG+MySQL), VM launcher scripts,
> CI integration, and ADR-0095. See
> [execution plan](docs/planning/2026-08-03_04-24_nix-integration-test-execution-plan.md).

- [ ] **systemd-nspawn container type for MySQL VM** — Could make VM test 10x
  faster (~131s → ~15s). The NixOS test driver supports `NspawnMachine`. Needs
  prototyping. (M14)
- [ ] **Shellcheck linting on VM scripts** — `shellcheck` not available on host.
  Add as a flake check or devShell dependency. (M22)
- [ ] **Connection retry logic with backoff in VM scripts** — More robust than
  simple polling. (M28)
- [ ] **Health check SQL verification before running tests** — Run `SELECT 1`
  before Go tests, fail fast with clear message. (M29)
- [ ] **macOS verification of ephemeral PG** — Script claims cross-platform but
  never tested on Darwin. (M34)
- [ ] **Cache ephemeral PG data dir** — Skip `initdb` on repeated runs. (M35)
- [ ] **DuckDB CGo VM test** — Hermetic DuckDB testing with GCC in VM. (M38)
- [ ] **SQLite WAL concurrency VM test** — Concurrent access patterns. (M39)
- [ ] **Turso sync VM test** — Real libSQL server. (M40)
- [ ] **Go test binaries inside QEMU VM** — Deeper coverage. (M41)
- [ ] **`projectionhost` crash-restart PG integration test** — Verify checkpoint
  replay after crash. (M43)
- [ ] **`scheduling` durable timers across restarts test** — Timer survives
  process restart. (M44)
- [ ] **Contract test suite across ALL backends in VMs** — SQLite, PG, MySQL,
  DuckDB simultaneously. (M46)

---

_Long-term direction lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
