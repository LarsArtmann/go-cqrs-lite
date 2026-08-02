# TODO List

**Updated:** 2026-08-02
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Metaengine — Open Work

> The metaengine is production-ready: **5 engines** (memory, SQLite, Pebble,
> DuckDB, Postgres), 10 ADTs (7 original + Vector/Search/Spatial), rule pipeline,
> materialize-vs-replay cost model, StorageLayout + cost matrix, SerializablePlan,
> VersionedStorage (temporal), StreamScan, ScanResult explicit HasMore, SSE
> delivery, Watcher, PrefetchCache, TypedReader, QueryBuilder, property-based
> cross-engine parity testing, Pebble sort index (1,233x speedup), Fold sealed
> interface. All known bugs are fixed. See CHANGELOG `[Unreleased]` for full
> detail.

- [x] ~~**Wire dead code from data model refactor**~~ — **DONE** (2026-08-02).
  `ApplyError` now wraps all fold errors in `applyFold` (structured context:
  query, event type, fold kind). `Valid()` methods are called at `Plan()` time
  for ADT, ReadPattern, and each FoldKind. `NsPerRead`/`NsPerWrite` were
  already wired.
- [x] ~~**Exhaustiveness guard test**~~ — **DONE** (2026-08-02). Compile-time
  test ensuring all Fold concrete types are handled in `applyFold` type switch
  (prevents silent fallthrough when a new fold type is added).
- [x] ~~**10M-event soak test**~~ — **DONE** (2026-08-02). `TestSoak_MemoryBounded_10M`
  in `metaengine/soak_10m_test.go`: 10M events into 1000 keys → 0.1 MB heap growth,
  flat growth curve. Verifies O(keys) bound, correctness of accumulated totals,
  and no sustained segment growth. Skips in `-short` mode and with `SOAK_SKIP_10M=1`.
- `[ ]` **`metaengine-gen` code generator** — typed Store methods from query
  declarations (CLI tool, similar to `cqrs-gen`). Go AST parsing + template
  generation.
- `[ ]` **Generic `ScanResult[T]`** — replace `[]any` with generic typed slice
  (currently `ScanResult{Items []any}`). Breaking API change; needs major
  version bump.
- `[ ]` **Boundary keys-type validation** — enforce that map keys passed to
  engines match the declared key type at the Store boundary (not just at fold
  time).
- `[ ]` **Watcher typed channel** — `Watcher[V]` sends `any`, not typed `V`.
  SQLite engine type assertion can silently fail.
- [x] ~~**DuckDB LayoutPlanner**~~ — **DONE** (2026-08-02). DuckDB engine now
  implements `LayoutPlanner` via dedicated planned tables with extracted columns
  and ART indexes (same pattern as SQLite). `ApplyLayout` creates a per-collection
  table; `MapSet`/`MapGet`/`MapDelete`/`PushdownMapScan` dispatch to the planned
  table with direct column references instead of `json_extract`, enabling
  DuckDB's zone maps to prune data blocks. 8 tests in `layout_planner_cgo_test.go`.
- `[ ]` **Postgres GIN containment indexes** — `@>` operator for JSONB path
  queries. Currently only expression indexes (B-tree on JSONB paths).
- `[ ]` **DuckDB columnar-native storage** — DuckDB stores JSON as VARCHAR;
  columnar scans not leveraged. Vectorized GROUP BY for CounterGet would use
  DuckDB's native columnar engine.
- [x] ~~**SSE consolidation**~~ — **DONE** (2026-08-02). ADR-0091 documents
  the intentional split between `metaengine.ServeSSE` (read-model push) and
  `transport/http.SSEBroker` (event stream push). Different layers, different
  replay strategies, different module boundaries — merging would violate
  the metaengine zero-dependency principle.
- `[ ]` **Vector/Search/Spatial backends** — currently Memory-only (brute-force).
  Future: DuckDB VSS extension (vector), Postgres tsvector (search), PostGIS
  (spatial). See ROADMAP.

---

## cqrs-lint — Open Work

> The linter has **181 rules** across 10 categories. Import-alias resolution,
> block-level suppression, self-lint mode, and D/E-series migrations are complete.
> Recently added: A033 (branded-ID string roundtrip), C037 (snapshot/event codec
> mismatch), block-level suppression (ADR-0088).
>
> **Round-2 consumer feedback processed (2026-08-02):** B022 bug fixed,
> P012/P013 cross-file blindness fixed, config-level rule disabling +
> `--exclude-rules` shipped, suppression parser accepts Go-idiomatic `// cqrs-lint:`,
> S006 over-broad `"total"` keyword removed, C036 shared-backend detection added,
> unknown-rule-ID stale suppression detection, `init --preset`, `--help`
> suppression docs. See the
> [round-2 review](docs/feedback/reviewed/2026-08-02_cqrs-lint-round-2-review.md).

- `[x]` ~~**Config-level rule disabling**~~ — **DONE** (2026-08-02). Added
  `"rules": {"disable": [...]}` in `.cqrs-lint.json` + `--exclude-rules` CLI flag.
- `[ ]` 🔥 **Run cqrs-lint against real consumer projects** — validate FP rate
  against Kernovia, Standup-Killer, bank-sync, cqrs-htmx, DiscordSync. Consumer
  feedback round 2 (bank-sync + browser-history) processed; see
  [review](docs/feedback/reviewed/2026-08-02_cqrs-lint-round-2-review.md).
- `[x]` ~~**Fix B022 bug**~~ — **DONE** (2026-08-02). Suggestion text corrected to
  `event.CommandCausalityEnricher`; `WithEnricher(event.CommandCausalityEnricher)`
  now correctly exempted.
- `[x]` ~~**Fix P012/P013 cross-file blindness**~~ — **DONE** (2026-08-02). Only
  files with direct `sql.Open("sqlite",...)` are flagged; constructor wrappers
  (`sqlite.New`, `NewSQLiteBackend`) are excluded.
- [x] ~~**C037 scope expansion**~~ — **DONE** (2026-08-02). Now covers all 4
  typed stores: snapshot, command, query, and kv (via `WithTypedCodec`).
- `[ ]` **F009/F015/F017 feature-profile gating** — fire on CLI projects where
  modules are deliberately not used (missing feature-profile check). Requires
  adding `HasAsyncBus` to the feature profile.
- [x] ~~**`--fix` support for D007**~~ — **DONE** (2026-08-02). D007 now emits
  per-call-site findings with `FixStrategyDirect`. `--fix` replaces
  `event.NewEvent(` with `event.New(` via the existing `CQRSFixProvider`.
  Multiple occurrences handled via pipeline iteration (MaxIterations=5).
- `[ ]` **Domain-based severity calibration (L1.5)** — makes all rules smarter
  via domain context (financial aggregates get stricter rules). Strategic item;
  deferred since 2026-07-30.
- `[ ]` **~14 remaining backlog items** — see the
  [Pareto plan](docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md).
  Open: L1.18 (config inheritance), L1.29 (event-type string typo detection),
  L1.30–L1.33 (deep pattern detection), L1.47–L1.51 (new categories DOC/OBS/RES/DI).

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).
- [BLOCKED] **Push `stack/duckdb/v4.0.0`, `metaengine/pgengine/v4.0.0`,
  `metaengine/duckdbengine/v4.0.0` tags** — all three tags created locally but
  not pushed (per safety rules). Consumers get 404 from Go proxy until pushed.
- [x] ~~**MySQL testcontainer privilege fix**~~ — **DONE** (2026-08-02).
  Replaced fragile `ctr.Exec` GRANT with Go-side `database/sql` root connection
  + retry loop (`waitForMySQLReady`). go-sql-driver/mysql v1.10+ supports
  caching_sha2_password, eliminating the auth issue.
- [x] ~~**Investigate `TestRun_Postgres_Recovery` benchkit failure**~~ —
  **DONE** (2026-08-02). Investigation: the test is well-designed (per-test
  database isolation, 90s timeout, skips without Docker). The flake is a CI
  resource issue (Docker testcontainer startup, parallel contention), not a
  code bug. No code fix needed.
- [x] ~~**Investigate `TestProperty_SQLiteTTLExpiry` flake**~~ — **DONE**
  (2026-08-02). Root cause: (1) `newTestStore(t)` registered cleanup on `t`
  not `rt`, accumulating hundreds of open SQLite connections across rapid
  iterations; (2) 50ms TTL + 100ms sleep was too tight under -race. Fixed:
  per-iteration store creation/cleanup via `defer`, generous 200ms TTL +
  500ms sleep. Stale rapid failure files cleaned.

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

_Long-term direction lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
