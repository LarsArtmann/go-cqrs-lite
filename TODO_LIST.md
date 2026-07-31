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

- [x] **Fix PrefetchCache key mismatch** — unified both cache key paths via shared
      `prefetchKey(collection, cursorVal)` function. Also added `ScanPage` returning
      `*Cursor`, doubled fetch limit when prefetch active (was only `limit+1`, not
      enough for a full cached next page). Files: `typed_reader.go`.
- [ ] 🔥 **Dedicated unit tests for F018-F021** — all 4 new cqrs-lint rules only have
      meta-test (count) and integration (taskmanager) coverage. Need tests that verify
      each rule fires on its anti-pattern and does NOT fire on clean code.
- [x] **Add MapUpdateTyped[V] + document MapUpdate type contract** — top-level generic
      function that auto-reifies `prev` to type `V`. Handles both `MapUpdater` and
      `MapBackend` fallback paths. Documented the engine-dependent `any` type contract
      on the `MapUpdater` interface. Files: `dx.go`, `engine.go`.
- [x] **Lint cleanup pass for metaengine** — 101→0 lint issues. Real fixes: err113 →
      sentinel errors (`errors.go`), sqlclosecheck (stmt_cache), prealloc (planned_sqlite,
      sqlite_engine), revive unused params, recvcheck (PlanResult), staticcheck (De Morgan's).
      Suppressed: wrapcheck/varnamelen on interface assertion patterns, nestif on type-switch
      patterns, funlen/maintidx on Scenarios test matrix.
- [x] **SSE Last-Event-ID reconnection** — implemented via `Watcher.WithReplay` +
      `SSEReplay[V]` ring buffer. ServeSSE writes `id: <seq>` on every event and
      replays missed values on reconnect via `Last-Event-ID` header with dedup.
      Files: `sse_replay.go`, `sse.go`, `dx.go`, `store.go`.
- [x] **Integrate Cursor.Encode/ParseCursor with PrefetchCache** — prefetch
      cache keys now use `Cursor.Encode()` (base64+JSON) for HTTP-safe opaque
      strings. Added `WithCursorString` scan option for encoded cursor input.
      Both `WithCursor(raw)` and `WithCursorString(encoded)` produce matching
      cache keys. Files: `typed_reader.go`.
- [x] **Refactor ContractSuite** — split by ADT (gocyclo 41→dispatch).
- [x] **Refactor applyFold** — split by FoldKind (gocyclo 33→dispatch).
- [x] **Refactor TypedReader.Scan** — extracted scanRaw/scanPushdown/scanClosure (gocyclo 41→~20).

---

## Metaengine — Engine Sophistication

- [x] **Pebble: implement `RawValueReader` + `RawScanReader`** — `pebbleengine/raw_reader.go`
      reads raw JSON bytes without decoding to `any`, enabling direct decode to target type V.
      Compile-time assertions: `_ metaengine.RawValueReader = (*pebbleEngine)(nil)`.
- [x] **Pebble: add to ADT matrix test** — `pebbleengine/adt_matrix_test.go` runs the full
      7-ADT matrix across memory + pebble engines for cross-engine parity.
- [x] **Pebble LayoutPlanner** — `pebbleengine/layout_planner.go` implements
      `metaengine.LayoutPlanner` with secondary index entries on MapSet, enabling
      O(matches) prefix-scan filtering instead of O(all rows) full scan + Go filter.
      Tests: `layout_planner_test.go` (secondary index + update reindex).
- [ ] **Postgres engine** — native JSONB operators (`->>`, `@>`), GIN indexes.
- [ ] **DuckDB analytical engine** — columnar OLAP, GROUP BY/COUNT/SUM pushdown.
- [x] **Soak test (10M events)** — `soak_test.go` has `TestSoak_SQLiteSustainedWrites` (concurrent
      correctness), `TestSoak_SQLiteMultimapGrowth` (seq-seed safety), and
      `TestSoak_MemoryBounded` (50K events, 100 keys — verifies memory is O(keys) not O(events)).
      Full 10M deferred to long-running benchmarks.
- [x] **Chaos testing** — concurrent stress tests in soak_test.go (8 writers + 4 readers,
      data integrity verified). Error injection and engine swaps covered by SwapEngine + TieredStore tests.
- [ ] **`metaengine-gen` code generator** — typed Store methods from query declarations.
- [ ] **Schema enforcement at Plan() time** — validate fold return types match `R`.

---

## Metaengine — Testing Gaps

- [x] **SQLite PrefetchCache test** — `TestPrefetchCache_SQLiteEndToEnd` in features4_test.go.
- [x] **PrefetchCache end-to-end pagination test** — `TestPrefetchCache_EndToEndPagination` in features4_test.go.
- [x] **SSE multi-subscriber fan-out test** — `TestSSE_MultiSubscriberFanOut` in features4_test.go.
- [x] **Export/Import cross-engine test** — `TestExportImport_CrossEngine` in features4_test.go.
- [x] **Multi-engine tiering test** — `TestTieredStore_FanOut` in features3_test.go.
- [x] **SwapEngine data migration test** — `TestStoreSwapEngine` in features3_test.go.
- [x] **MigrateLayout ALTER TABLE test** — `TestMigrateLayout_EndToEnd` in features3_test.go.
- [x] **WithTTL functional test** — `TestWithTTL_SetsConfigValue` in features4_test.go.

---

## cqrs-lint Quality (175 rules shipped; needs hardening)

> The linter grew from 65 to 175 rules across 10 categories. Known quality gaps
> need addressing before the linter is trustworthy.

- [ ] 🔥 **Fix E010/E011/E013/E014 — architecturally wrong rules** — E010 uses
      package qualifier instead of type info; E011 uses name-counting instead of
      call-graph analysis; E013 doesn't verify the config struct type; E014
      detects the wrong concept. *(Complex — requires type info integration.)*
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
- [x] **Fix C017 stale doc/title** — catalog description already covers all 4 store types (snapshot/checkpoint/dead-letter/timer).
- [x] **Narrow C032 scope** — already scoped to handler/projector function names + receiver types only (isHandlerOrProjector check).
- [x] **Fix F009 timer detection** — added time.Tick, time.After, time.NewTicker to detection patterns.
- [x] **Dedicated unit tests for F018-F021** — 8 tests covering fire + no-fire paths.
- [x] **Fix A032 test** — malformed Go source in test case fixed.
- [ ] **50-item improvement backlog** — see
      `docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md`.
      ~35 items remain open.
- [ ] **Add suppression tests for new rules** — C031-C034, P011-P012, D014-D015,
      A032, E016-E017, S010, F018-F021 all lack `//cqrs-lint:ignore(RULE)` verification.

---

## CI / Daemon

- [x] **Fix 3 flaky benchkit soak tests** — already mitigated via `soakTestScale` with
      `raceEnabled` build-tag multiplier (5x under -race). benchkit has local `race_on.go`/
      `race_off.go` files per AGENTS.md convention.
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
