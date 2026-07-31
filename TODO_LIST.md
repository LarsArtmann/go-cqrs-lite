# TODO List

**Updated:** 2026-07-31
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- `🐛` = Known bug

---

## Metaengine — Open Bugs & Gaps

> The metaengine grew from a prototype to a production-ready multi-engine
> cost-based storage planner across 6 sessions (2026-07-30 to 2026-07-31).
> Most items are done and shipped. These remain open.

- 🐛 `[ ]` **Pebble LayoutPlanner range filter numeric bug** — lexicographic
  key ordering ≠ numeric ordering for values with different digit counts
  (e.g., `2` vs `10`). Test passed by accident (used only 2-digit values).
  Fix: zero-pad numeric keys or use a numeric-aware comparator.
  _(Source: `docs/status/2026-07-31_14-57_metaengine-pebble-layoutplanner-cqrs-lint-hardening.md` §D5)_
- 🐛 `[ ]` **Fix `TestSSE_DropOldSemantics` hang** — SSE goroutines
  (`forwardWithDropOld`) block on channel selects that never drain after
  `httptest.Server.Close()`. Blocks the full metaengine test suite from
  completing cleanly. Workaround: use `-run` filters to exclude it.
  _(Source: `docs/status/2026-07-31_17-19_metaengine-engine-sophistication-complete.md` §D1)_
- `[ ]` **Pebble LayoutPlanner sort index** — `sortFields` stored but unused
  for ordering. Requires a separate sort-prefix index structure.
  _(Source: `docs/status/2026-07-31_14-57_*.md` §B1)_
- `[ ]` **Add Pebble to metaengine `adt_matrix_test.go`** — currently only
  memory + SQLite. No triple-parity test exists.
  _(Source: `docs/status/2026-07-31_17-19_*.md` §G3)_
- `[ ]` **Fix `scanWithIndex` cursor pagination gap** — the index fast path
  in `ScanRawValues` doesn't apply cursor pagination.
  _(Source: `docs/status/2026-07-31_17-19_*.md` §F44)_

---

## cqrs-lint — Open Quality Items

> The linter has **175 rules across 10 categories**. Most quality gaps from the
> brutal review are addressed. These remain open.

- `[ ]` 🔥 **50-item improvement backlog** — ~35 items remain open in the
  [Pareto plan](docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md).
  Includes domain-based severity calibration (L1.5), C017 tracing (L1.9),
  migration paths in findings (L1.16), doc links (L1.17), block-level
  suppression (L1.22), new categories (DOC/OBS/RES/DI, L1.47–L1.50).
- `[ ]` **Add suppression tests for new rules** — C031-C034, P011-P012,
  D014-D015, A032, E016-E017, S010, F018-F021 all lack
  `//cqrs-lint:ignore(RULE)` verification.
- `[ ]` **Migrate import-alias resolution to remaining E-series rules** —
  `QualifierToImportPath` + `ImportQualifierMap` helpers exist and E008
  was migrated as proof of concept. D007/D008/D010/D013 and E009-E015
  still use variable-name heuristics.
  _(Source: `docs/status/2026-07-30_23-22_cqrs-lint-hardening-and-verify-gate-repair.md` §F1)_
- `[ ]` **Run cqrs-lint against real consumer projects** — validate against
  Kernovia, Standup-Killer, bank-sync, cqrs-htmx, DiscordSync.

---

## CI / Daemon / Release

- `[ ]` **Recurring lint-sweep** — the auto-commit daemon occasionally commits
  unformatted code. Either gate daemon commits behind `nix fmt` or run a
  scheduled sweep.
- `[ ]` **CGo-enabled CI job** — add a separate CI job with `CGO_ENABLED=1`
  for DuckDB tests (stack/duckdb requires CGo).
- `[ ]` **Investigate `TestRun_Postgres_Recovery` benchkit failure** — may
  still flake.
- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).
- [BLOCKED] **Tag `stack/duckdb/v4`** — the module exists but is untagged.
  Consumers resolving the latest version get a 404 from the Go proxy. The
  `govalid-generate` step was patched via SSH redirect in `flake.nix`, but the
  root cause (no published tag) remains.

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
