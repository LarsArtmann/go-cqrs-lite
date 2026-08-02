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

- `[ ]` 🔥 **Wire dead code from data model refactor** — branded unit types
  (`NsPerRead`, `NsPerWrite`, `ByteSize`), `ApplyError`, `Valid()` calls at
  `Plan()` time are defined but NOT wired. They are currently dead code.
- `[ ]` 🔥 **Exhaustiveness guard test** — compile-time test ensuring all Fold
  concrete types are handled in `applyFold` type switch (prevents silent
  fallthrough when a new fold type is added).
- `[ ]` **10M-event soak test** — verify memory boundedness at scale (currently
  50K events; stretch goal is 10M).
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
- `[ ]` **DuckDB LayoutPlanner** — DuckDB engine has no layout planner (JSON
  stored as VARCHAR; no expression indexes).
- `[ ]` **Postgres GIN containment indexes** — `@>` operator for JSONB path
  queries. Currently only expression indexes (B-tree on JSONB paths).
- `[ ]` **DuckDB columnar-native storage** — DuckDB stores JSON as VARCHAR;
  columnar scans not leveraged. Vectorized GROUP BY for CounterGet would use
  DuckDB's native columnar engine.
- `[ ]` **SSE consolidation** — `metaengine.ServeSSE` overlaps
  `transport/http.SSEBroker`. ADR needed: consolidate, or document the
  intentional split.
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
- `[ ]` **C037 scope expansion** — only covers snapshot store (1 of 5 typed
  stores). Missing: kv, command, query, stack.Materialize.
- `[ ]` **F009/F015/F017 feature-profile gating** — fire on CLI projects where
  modules are deliberately not used (missing feature-profile check). Requires
  adding `HasAsyncBus` to the feature profile.
- `[ ]` **`--fix` support for D007** — mechanical `event.NewEvent` → `event.New`
  migration could be auto-fixed.
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
- [BLOCKED] **Push `stack/duckdb/v4.0.0` tag** — tag created locally but not
  pushed (per safety rules). Consumers get 404 from Go proxy until pushed.
- [BLOCKED] **Tag `metaengine/pgengine/v4.0.0` + `metaengine/duckdbengine/v4.0.0`**
  — both modules shipped but untagged. Consumers cannot resolve them.
- `[ ]` **MySQL testcontainer privilege fix** — root password auth
  intermittently fails. Testcontainer GRANT pattern is fragile.
- `[ ]` **Investigate `TestRun_Postgres_Recovery` benchkit failure** — may
  still flake under CI.
- `[ ]` **Investigate `TestProperty_SQLiteTTLExpiry` flake** — pre-existing
  property test failure in `idempotency/sqlstore`.

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
