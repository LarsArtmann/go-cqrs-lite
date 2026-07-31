# TODO List

**Updated:** 2026-07-31 (session 05:44)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task is finished it is removed from
this list and recorded in CHANGELOG.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Metaengine — Critical Bugs & Quality (from session 2026-07-31 05:44)

> Session report: `docs/status/2026-07-31_05-44_metaengine-quality-pass-comprehensive-status.md`

- [ ] 🔥 **Fix PrefetchCache key mismatch** — `trimAndCache` generates cache keys via
      `cursorKeyFor(item, cfg)` but the prefetch-lookup at `typed_reader.go:140` uses
      `fmt.Sprintf("%s:%v", collection, cfg.cursor)`. The two formats don't match, so
      auto-populated cache entries are never served. Unify both key paths or use an
      opaque cursor protocol. (`metaengine/typed_reader.go:140` vs `:663`)
- [ ] 🔥 **Dedicated unit tests for F018-F021** — all 4 new cqrs-lint rules only have
      meta-test (count) and integration (taskmanager) coverage. Need tests that verify
      each rule fires on its anti-pattern and does NOT fire on clean code.
- [ ] **Add MapUpdateTyped[V]** — the `MapUpdate` callback receives `any` which is
      engine-dependent typed (MemoryEngine preserves Go types, SQLite returns
      `map[string]any` from JSON). A typed variant that auto-reifies `prev` eliminates
      this footgun.
- [ ] **Document MapUpdate type contract** — if not making it typed, at least document
      that callbacks receive engine-dependent `any` values.
- [ ] **Lint cleanup pass for metaengine** — 72 lint issues (err113: 8, wrapcheck: 18,
      varnamelen: 14, gocyclo: 3, nestif: 7, nlreturn: 4). Mechanical but time-consuming.
      Highest priority: err113 in `sse.go`, wrapcheck in `sse.go`/`export_import.go`.
- [ ] **SSE Last-Event-ID reconnection** — no replay support. Clients can't reconnect
      after disconnect. Needs either a journal mechanism or Watcher event tracking.
- [ ] **Integrate Cursor.Encode/ParseCursor with PrefetchCache** — opaque cursor
      strings for HTTP-safe pagination.
- [ ] **Refactor ContractSuite** — cyclomatic complexity 41 (threshold 30). Split by ADT.
- [ ] **Refactor applyFold** — cyclomatic complexity 33 (threshold 30). Split by FoldKind.
- [ ] **Refactor TypedReader.Scan** — cyclomatic complexity 39 (threshold 30). Extract
      pushdown/raw/closure scan paths.

---

## Metaengine — Engine Sophistication

- [ ] 🔥 **Pebble: implement `RawValueReader` + `RawScanReader`** — Pebble still
      JSON-decodes every value on read. Eliminates the JSON tax for Pebble.
- [ ] 🔥 **Pebble: add to ADT matrix test** — extend `engineFactories()` in
      `adt_matrix_test.go` with the Pebble engine.
- [ ] **Pebble LayoutPlanner** — prefixed key ranges for indexed fields.
- [ ] **Postgres engine** — native JSONB operators (`->>`, `@>`), GIN indexes.
- [ ] **DuckDB analytical engine** — columnar OLAP, GROUP BY/COUNT/SUM pushdown.
- [ ] **Soak test (10M events)** — verify memory doesn't grow unboundedly.
- [ ] **Chaos testing** — random transaction kills, error injection, engine swaps.
- [ ] **`metaengine-gen` code generator** — typed Store methods from query declarations.
- [ ] **Schema enforcement at Plan() time** — validate fold return types match `R`.

---

## Metaengine — Testing Gaps

- [ ] **SQLite PrefetchCache test** — verify auto-population works with SQLite engine.
- [ ] **PrefetchCache end-to-end pagination test** — multi-page scan using cursor flow.
- [ ] **SSE multi-subscriber fan-out test** — verify N clients all receive updates.
- [ ] **Export/Import cross-engine test** — export from Memory, import to SQLite, verify.
- [ ] **Multi-engine tiering test** — TieredStore fan-out with SQLite + memory.
- [ ] **SwapEngine data migration test** — verify data survives engine swap via replay.
- [ ] **MigrateLayout ALTER TABLE test** — verify column addition preserves data.
- [ ] **WithTTL functional test** — verify entries actually expire.

---

## cqrs-lint Quality (175 rules shipped; needs hardening)

> The linter grew from 65 to 175 rules across 10 categories. Known quality gaps
> need addressing before the linter is trustworthy.

- [ ] 🔥 **Fix E010/E011/E013/E014 — architecturally wrong rules** — E010 uses
      package qualifier instead of type info; E011 uses name-counting instead of
      call-graph analysis; E013 doesn't verify the config struct type; E014
      detects the wrong concept.
- [ ] 🔥 **Library self-lint mode** — auto-detect `go-cqrs-lite` module path and
      suppress consumer-only rules for library files. Currently requires 181+ manual
      inline suppressions.
- [ ] 🔥 **Import-alias resolution** — D007/D008/D010/D013 and all E-series rules
      assume unqualified package names. Build a shared `qualifierToImportPath` helper.
- [ ] **Fix F-series detection gaps** — F011 broad `.Exec` matching needs receiver type
      checking; F009 timer detection should include `time.Tick`/`time.After`; F013 HTTP
      handler detection should cover chi/gin/echo/fiber.
- [ ] **Review C030 over-suppression** — "any return = safe" may mask real bugs.
- [ ] **Audit S006 indicators for substring false positives**.
- [ ] **Fix C017 stale doc/title** — detects 4 store types but titled "snapshot only".
- [ ] **Narrow C032 scope** — fires on ALL ctx functions, should be handler/projector only.
- [ ] **50-item improvement backlog** — see
      `docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md`.
      ~35 items remain open.
- [ ] **Add suppression tests for new rules** — C031-C034, P011-P012, D014-D015,
      A032, E016-E017, S010, F018-F021 all lack `//cqrs-lint:ignore(RULE)` verification.

---

## CI / Daemon

- [ ] **Fix 3 flaky benchkit soak tests** — `TestRunSoak_Memory`,
      `TestRunSoak_TrendsPopulated`, `TestRunSoakJSON_RoundTrip`. Timing-sensitive
      under parallel race-detector load. Use `testutil.RaceEnabled` build-tag thresholds.
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
