# TODO List

**Updated:** 2026-07-31
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Metaengine — Remaining Work

> The metaengine is production-ready: **5 engines** (memory, SQLite, Pebble,
> DuckDB, Postgres), 7-ADT cross-engine parity tests, LayoutPlanner
> (filter + sort indexes), cursor pagination, SSE delivery, transaction API,
> ADT test harness, StreamScan (lazy iter.Seq2), ScanCount, property-based
> parity testing. All known bugs are fixed (including goroutine leak in Watch).
> pgengine and duckdbengine are shipped (PushdownScan + LayoutPlanner); see
> ROADMAP for their remaining sub-tasks (GIN containment indexes, vectorized
> GROUP BY).

- `[ ]` **10M-event soak test** — verify memory boundedness at scale (currently 50K).
- `[ ]` **`metaengine-gen` code generator** — typed Store methods from query
  declarations (CLI tool, similar to `cqrs-gen`).

---

## cqrs-lint — Remaining Work

> The linter has **181 rules** across 10 categories. Import-alias resolution,
> suppression tests, and D/E-series migrations are complete. C017 tracing
> (L1.9), migration paths in findings (L1.16), and doc links (L1.17) are done.
> Recently added: A033 (branded-ID string roundtrip), C037 (snapshot/event
> codec mismatch).

- `[ ]` 🔥 **~17 open items in the improvement backlog** — see the
  [Pareto plan](docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md).
  Top open items: domain-based severity calibration (L1.5), block-level
  suppression (L1.22), new categories (DOC/OBS/RES/DI, L1.47–L1.50),
  event-type string typo detection (L1.29).
- `[ ]` **Run cqrs-lint against real consumer projects** — validate FP rate
  against Kernovia, Standup-Killer, bank-sync, cqrs-htmx, DiscordSync.
- `[ ]` **cqrs-lint domain-based severity** — (L1.5) makes all rules smarter
  via domain context (financial aggregates get stricter rules).

---

## CI / Daemon / Release

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).
- [BLOCKED] **Push `stack/duckdb/v4.0.0` tag** — tag created locally but not
  pushed (per safety rules). Consumers get 404 from Go proxy until pushed.
- `[ ]` **Investigate `TestRun_Postgres_Recovery` benchkit failure** — may
  still flake under CI.

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

---

_Long-term direction lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
