# TODO List

**Updated:** 2026-07-31 (session 14:30)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task is finished it is removed from
this list and recorded in CHANGELOG.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## cqrs-lint Quality (175 rules shipped; hardening in progress)

> The linter grew from 65 to 175 rules across 10 categories. Most quality gaps
> are now addressed. Remaining items are polish + the 50-item improvement backlog.

- [x] 🔥 **Fix E010/E011/E013/E014** — E010 + E014 rewritten with go/types-aware
      receiver matching (`projectCallsMethodOnType`). E011 left as-is (name-counting
      is reasonable for low-confidence advisory with threshold 3). E013 already uses
      type-aware composite literal matching (`findKeyBoolLitInTypedComposite`).
- [x] 🔥 **Library self-lint mode** — `RegisterAll` now checks `IsLibrarySelfLint()`
      and skips 29 consumer-coaching rules (8 architecture E008-E015 + 21 adoption
      F001-F021) when linting the go-cqrs-lite source itself. Eliminates need for
      181+ manual inline suppressions.
- [x] 🔥 **Import-alias resolution** — `QualifierToImportPath` + `ImportQualifierMap`
      helpers exist in lintutil.go. `projectCallsImportPath` wrapper added to
      architecture/helpers.go. E008 migrated as proof of concept. Pattern documented
      for other rules to follow.
- [x] **Fix F-series detection gaps** — F011 countSQLExec now uses type info to
      verify receiver is `*sql.DB`/`*sql.Tx`/`*sql.Conn` (with variable-name fallback).
      F013 now detects chi/gin/echo/fiber/gorilla/httprouter web framework imports.
      F009 timer detection already expanded (prior session).
- [x] **Review C030 over-suppression** — "any return/break/.Done() = safe" heuristic
      is intentional. The concern about masking graceful-shutdown bugs represents a
      different rule category, not a C030 deficiency. No change needed.
- [x] **Audit S006 for substring false positives** — Three-tier system (STRONG/MEDIUM/WEAK
      with ≥2 compound threshold), serialization tag gate, and encryption module check
      minimize false positives. Short indicators (`swift`, `bic`) are financial-specific
      enough. No change needed.
- [x] **Fix C017 stale doc/title** — catalog description already covers all 4 store types.
- [x] **Narrow C032 scope** — already scoped to handler/projector function names.
- [x] **Fix F009 timer detection** — added time.Tick, time.After, time.NewTicker.
- [x] **Dedicated unit tests for F018-F021** — 8 tests covering fire + no-fire paths.
- [x] **Fix A032 test** — malformed Go source in test case fixed.
- [ ] **50-item improvement backlog** — see
      `docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md`.
      ~35 items remain open.
- [ ] **Add suppression tests for new rules** — C031-C034, P011-P012, D014-D015,
      A032, E016-E017, S010, F018-F021 all lack `//cqrs-lint:ignore(RULE)` verification.

---

## Metaengine — Engine Sophistication

> Multi-day engine creation work (Postgres, DuckDB, metaengine-gen) moved to
> [ROADMAP.md](ROADMAP.md) — each is a 2-4 day new-module effort.

- [x] **Schema enforcement at Plan() time** — validate fold return types match `R`.
      Already implemented and tested.
- [x] **Pebble LayoutPlanner** — secondary index with O(matches) prefix scan.
      MapDelete + MapUpdate now clean up index entries atomically.
      Benchmark: 108x speedup over full scan (6ms→56μs, 80K→311 allocs).
- [x] **Soak test** — concurrent correctness, multimap growth safety, memory boundedness.
- [x] **Chaos testing** — concurrent stress tests, error injection, engine swaps.
- [ ] **Pebble LayoutPlanner range filters** — FilterGt/FilterLt/FilterIn fall through
      to full scan (only FilterEq uses the index). Requires lexicographic value encoding.
- [ ] **Pebble LayoutPlanner sort index** — sortFields stored but unused for ordering.
      Requires separate sort-prefix index structure.

---

## CI / Daemon

- [x] **Fix 3 flaky benchkit soak tests** — mitigated via `soakTestScale` with
      `raceEnabled` build-tag multiplier.
- [ ] **Recurring lint-sweep** — the auto-commit daemon occasionally commits unformatted
      code. Either gate daemon commits behind `nix fmt` or run a scheduled sweep.
- [ ] **CGo-enabled CI job** — add a separate CI job with `CGO_ENABLED=1` for DuckDB tests.
- [ ] **Investigate `TestRun_Postgres_Recovery` benchkit failure** — may still flake.

---

## Release

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod replace
  directives are needed for dev; consumers resolving the published modules
  depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).

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
