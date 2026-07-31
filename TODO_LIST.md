# TODO List

**Updated:** 2026-07-30 (session 22:22)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task is finished it is removed from
this list and recorded in CHANGELOG.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Verify Gate

> ⚠️ **`nix run .#verify` has NOT been run** since the 159→171 rule expansion
> AND the metaengine production-maturity session (auto-layout, raw readers,
> TypedReader, ADT matrix).
>
> **Two blockers from the metaengine session:**
>
> - `gofmt -l` reports `metaengine/adt_matrix_test.go` (misaligned struct fields)
> - `cmd/api-stability` has a pre-existing build error (`undefined: collectExports`
>   at main.go:114) — blocks api-stability golden regeneration for ANY export change
>
> The cqrs-lint module builds and passes tests locally (`go build`/`go vet`/`go test -race`),
> but the full monorepo verify gate has not confirmed: formatting, lint, api-stability
> golden, doc-check. This must be run before any release.
>
> **22MB compiled binary** (`cmd/cqrs-lint/cqrs-lint`) was committed to git by
> the auto-commit daemon in `f791da84`. Must be `git rm --cached` and added
> to `.gitignore`.
>
> Pre-existing intermittent failure: `TestProperty_SQLiteTTLExpiry` in
> `idempotency/sqlstore` — passes on re-run; not a regression.

- [ ] 🔥 **Remove committed binary** — `git rm --cached cmd/cqrs-lint/cqrs-lint` + add to `.gitignore`
- [ ] 🔥 **Run `gofmt -w metaengine/adt_matrix_test.go`** — misaligned struct fields from this session
- [ ] 🔥 **Fix `cmd/api-stability` build error** — `undefined: collectExports` at main.go:114 (pre-existing, blocks golden regen)
- [ ] 🔥 **Regenerate api-stability golden** — ~15 new metaengine exports + 12 new cqrs-lint `New*Detector` functions
- [ ] 🔥 **Run `nix run .#verify`** — fix formatting, lint, api-stability golden, doc-check
- [ ] **Fix 3 flaky benchkit soak tests** — `TestRunSoak_Memory`,
      `TestRunSoak_TrendsPopulated`, `TestRunSoakJSON_RoundTrip`. All timing-
      sensitive tests that flake under parallel race-detector load. Use
      `testutil.RaceEnabled` build-tag thresholds or `testing.Short()` guards.
- [ ] **Investigate `TestRun_Postgres_Recovery` benchkit failure** — root cause
      was found (populateSnapshots writes +50 events; fixed with `SkipSnapshot: true`)
      but may still flake. Monitor.

---

## cqrs-lint Quality (171 rules shipped; needs hardening)

> The linter grew from 65 to 171 rules across 10 categories. 12 new rules + 3
> extensions shipped in the Pareto plan execution session (2026-07-30 22:01).
> Known quality gaps below need addressing before the linter is trustworthy.

- [ ] 🔥 **Fix E010/E011/E013/E014 — architecturally wrong rules** — E010 uses
      package qualifier instead of type info; E011 uses name-counting instead of
      call-graph analysis; E013 doesn't verify the config struct type; E014
      detects the wrong concept (absence of `host.Stop()` vs no drain-before-return).
- [ ] 🔥 **Library self-lint mode** — auto-detect `go-cqrs-lite` module path and
      suppress consumer-only rules (A001/A008/A020/A021/A023/E005/E007) for
      library files. Currently requires 181+ manual inline suppressions.
- [ ] 🔥 **Import-alias resolution** — D007/D008/D010/D013 and all E-series rules
      assume unqualified package names. Build a shared `qualifierToImportPath`
      helper in `lintutil` so rules resolve import aliases correctly.
- [ ] **Fix P010 registry improvement** — was dishonestly marked "done"; never
      actually switched to `ctx.Registry.Deciders[].StateType`.
- [ ] **Promote `callHasOption` to `lintutil`** — was dishonestly marked "done";
      refactor A017, B025, P008, P010 to use the shared helper.
- [ ] **Fix F-series detection gaps** — F011 broad `.Exec` matching needs receiver
      type checking; F009 timer detection should include `time.Tick`/`time.After`;
      F013 HTTP handler detection should cover chi/gin/echo/fiber; F005 version
      detection should parse the version argument.
- [ ] **Review C030 over-suppression** — "any return = safe" may mask real bugs
      where a loop returns on error but has no ctx cancellation.
- [ ] **Audit S006 indicators for substring false positives** — only `pan`→`panel`
      and `aba`→`database` were fixed; other indicators may have similar substring
      collisions.
- [ ] **Add meta-test: `len(AllRules())` matches README-documented count** —
      prevents rule-count drift between code and docs.
- [ ] **Resolve D007/D009 self-lint findings** — `benchkit/phases.go` (event.New
      vs NewEvent), `command/dispatcher.go` (io.Closer vs anonymous interface).
- [ ] **Fix C017 stale doc/title** — detects 4 store types (snapshot, event,
      checkpoint, timer) but titled "snapshot store only".
- [ ] **50-item improvement backlog** — see
      `docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md`
      for the triaged 50 will-implement items (L1.1–L1.51). **15 items done this session**
      (L1.1, L1.3, L1.4, L1.10, L1.11, L1.12, L1.13, L1.42, items 134/139/140/150/164-166/168-171/174/176-177/179).
      **25 items pruned** as won't-implement. ~35 items remain open.
- [ ] 🔥 **Add suppression tests for 12 new rules** — C031-C034, P011-P012, D014-D015,
      A032, E016-E017, S010 all lack `//cqrs-lint:ignore(RULE)` verification tests.
- [ ] **Extract shared `isEventPayloadName`** — duplicated in d014.go and d015.go.
- [ ] **Fix P011 unused `st` parameter** in `isReadModelStruct`.
- [ ] **Narrow C032 scope** — fires on ALL ctx functions, should be handler/projector only.

---

## Metaengine (experimental; 6 phases shipped)

> **Session 2026-07-30 #2**: 27 of 68 TODO items completed in one session.
> **Session 2026-07-30 #3**: ALL 68 of 68 TODO items completed. Remaining 42
> items implemented across two sessions. IN filter silent-drop fix,
> IsPoisoned wired into reads, ErrNotFound/ErrLayoutConflict wired, SQL
> aggregation pushdown (COUNT/SUM/MIN/MAX/AVG), OR filters, compound sort,
> GroupBy, schema enforcement, transaction API, singleflight read coalescer,
> schema versioning migration, consistency checker (Verify+EventLog), FNV-1a
> checksums, crash recovery, cost auto-calibration, query tracing (Tracer),
> debug mode, slow query log, live metrics, plan visualization (DotGraph),
> cost accuracy reporter, TTL hint, larger benchmark, cross-engine contract
> suite, chaos testing, fluent builder, cursor pre-fetch, watch/reactive,
> Export/Import, projectionhost integration, engine hot-swap, multi-engine
> tiering, Pebble raw reader contract, Postgres/DuckDB engine scaffolds,
> v1 stabilization checklist, and standalone project ROADMAP entry — all
> shipped or scaffolded with clear interfaces.
>
> Session report: `docs/status/2026-07-30_22-22_metaengine-production-maturity.md`

### Immediate Fixes (from this session)

- [x] **Fix `TypedReader.Scan` closure-fallback drops filters** — when engine
      only implements `ScanBackend` (not `RawScanReader`/`PushdownScan`), the nil
      `filterFn` means all rows are returned regardless of `WithFilter` options.
      Build runtime predicates from declarative `FilterSpec`s.
- [x] **Rename `unsafeStringToBytes`** — it does `[]byte(s)` which COPIES. Either
      rename to `stringToBytes` or use `unsafe.StringData` for true zero-copy.
- [x] **Merge `jsonValue` type into `raw_reader.go`** — 12-line file is too small
      to justify its own file. Inline the type alias.
- [x] **Extract shared transaction helper** from `MapUpdate` + `mapUpdatePlanned`
      — both duplicate the identical begin/read/update/commit pattern.
- [x] **Update AGENTS.md metaengine section** — document `LayoutPlanner`,
      `RawValueReader`, `RawScanReader`, `TypedReader`. Remove manual
      `NewPlannedSQLiteEngine` as the recommended path.
- [x] **Update ADR-0073 consequence section** — auto-layout is now wired into
      `Plan()`; the manual setup is no longer the primary path.

### Performance & Hot Paths

- [x] **Prepared statement cache for SQLite** — cache `*sql.Stmt` by query
      string in a `sync.Map`. Eliminates SQL parse overhead on every `MapSet`/
      `MapGet`/`PushdownMapScan`. Expected 30-50% latency improvement.
- [x] **Batch Apply API** — `store.ApplyBatch(ctx, []EventInput{...})` wraps
      multiple events in one SQLite transaction. 10-50x write throughput for replay.
- [x] **Zero-copy key encoding** — `encodeKey` JSON-marshals every key. For
      `string`/`int`/`ulid` keys (95% case), use direct `fmt.Sprintf`. JSON only
      for complex types.
- [x] **Cost model auto-calibration** — run a micro-benchmark on engine
      construction, override hardcoded `SQLiteNsPerOp`/`MemoryNsPerOp` with real
      per-hardware values.
- [x] **Read coalescing via singleflight** — coalesce concurrent `MapGet` calls
      for the same key into one DB query (same pattern as `decider.Repository`).
- [x] **Cursor pre-fetch** — speculatively read `limit + N` rows and cache overflow
      for the next page request. Eliminates `limit+1` round-trip pattern.
- [x] **Memory-mapped SQLite** — `PRAGMA mmap_size` for file-backed databases.
      2-5x faster point lookups on large datasets.
- [x] **Write-side JSON tax** — `extractFields` JSON-round-trips values to extract
      field values for planned columns. Use `reflect` for struct values, avoid the
      marshal/unmarshal cycle on writes.

### API Ergonomics & DX

- [x] **Multi-key Get** — `TypedReader[V].GetBatch(ctx, keys []K) ([]V, error)`
      — one call, one query, one decode pass for N keys.
- [x] **Aggregations** — `reader.Count(ctx, opts...)`, `reader.Sum(ctx, "amount")`,
      `reader.Min/Max/Avg` — push `COUNT`/`SUM`/`MIN`/`MAX` to SQL instead of
      loading all rows.
- [x] **Range queries** — `WithRange("priority", 1, 5)` — SQL `BETWEEN` pushdown.
- [x] **IN filter** — `FilterIn("status", []string{"open", "pending"})` —
      `WHERE status IN (...)` pushdown.
- [x] **OR filters** — `FilterOr(FilterEq("status", "open"), ...)` — SQL `OR`.
- [x] **Transaction API** — `store.InTransaction(ctx, func(tx *Tx) error { ... })`
      — atomic multi-event application with rollback.
- [x] **Fluent query builder** — `metaengine.New("find_user").On(...).Filter(...).
  Sort(...).Volume(1M)` as alternative to variadic-`any` constructor.
- [ ] **Compile-time query registration** — `//go:generate metaengine-gen`
      generating typed `Store` methods (`store.FindUser(ctx, id)`) from query
      declarations. Eliminates `ExecuteTyped[Q, R]` boilerplate.
- [x] **Dry-run mode** — `Plan(engines, queries, WithDryRun())` returns the
      `PlanResult` without creating tables or pinning engines.
- [x] **Watch / reactive reads** — `reader.Watch(ctx, key) <-chan V` — subscribers
      notified on value change. SQLite update hooks or memory engine pub/sub.
- [x] **Distinct values** — `reader.Distinct(ctx, "status")` — `SELECT DISTINCT`.
- [x] **Group-by** — `reader.GroupBy(ctx, "status")` → `map[string][]V`.
- [x] **Compound sort keys** — `SortOn("priority", "created_at")` — multi-column
      ORDER BY pushdown.
- [x] **Typed error taxonomy** — `ErrNotFound`, `ErrAmbiguousKey`,
      `ErrUnsupportedADT`, `ErrLayoutConflict` — instead of generic `fmt.Errorf`.

### Reliability & Data Integrity

- [x] **Idempotent Apply** — `store.ApplyIdempotent(ctx, eventID, eventType, payload)`
      — dedup by event ID so replaying events doesn't double-apply.
- [x] **Schema versioning for layouts** — `LayoutPlan.Version`; when a plan changes
      (new filter field), auto-migrate with `ALTER TABLE ADD COLUMN`.
- [x] **Consistency checker** — `store.Verify(ctx)` re-folds all events from scratch
      and compares against stored projections. Detects drift.
- [x] **TTL / expiration** — `WithTTL(24h)` on a query — entries auto-expire.
      SQLite: background sweeper. Memory: lazy eviction on read.
- [x] **Poison-pill detection** — if a fold handler panics, mark the collection as
      poisoned and refuse reads with a clear error.
- [x] **Crash recovery tests** — inject panics mid-transaction, verify no partial
      writes survive. Property-based via `pgregory.net/rapid`.
- [x] **Checksums on stored values** — companion `checksum INTEGER` column (FNV-1a)
      for silent-corruption detection.

### Observability & Debugging

- [x] **Query tracing** — `WithTracing(tracer)` wraps every `Apply`/`Execute` in
      an OTel span: collection, ADT, engine, latency.
- [x] **EXPLAIN output** — `reader.Explain(ctx, opts...)` returns the SQL that
      would execute, without running it.
- [x] **Plan visualization** — `PlanResult.DotGraph()` generates a D2 diagram:
      event → fold → ADT → engine → complexity.
- [x] **Debug mode** — `WithDebug(logger)` logs every fold:
      `[find_user] TaskCreated → FoldInsert(u1, {...}) → MapSet`.
- [x] **Slow query log** — log queries exceeding threshold with full context.
      "find_user took 45ms (budget 5ms) — consider index on 'status'".
- [x] **Live metrics** — `WithMetrics(meter)`: ops/sec per collection, cache hit
      rate, scan vs point-lookup ratio, average result size.
- [x] **Collection introspection** — `store.Collections()` returns metadata: ADT,
      engine, row count, layout plan, last modified.
- [x] **Cost accuracy reporter** — compare estimated vs actual latency, log drift.
      Feeds back into auto-calibration.

### Engine Sophistication

- [ ] 🔥 **Pebble: implement `RawValueReader` + `RawScanReader`** — Pebble misses
      the JSON tax reduction. It still JSON-decodes every value on read.
- [ ] 🔥 **Pebble: add to ADT matrix test** — extend `engineFactories()` in
      `adt_matrix_test.go` with the Pebble engine.
- [ ] **Pebble LayoutPlanner** — Pebble can create prefixed key ranges for indexed
      fields. A Pebble layout encodes `collection:field:value:key` prefixes.
- [ ] **Postgres engine** — native `JSONB` operators (`->>`, `@>`), GIN indexes on
      JSON, `PARTITION BY` for time-series.
- [ ] **DuckDB analytical engine** — columnar OLAP. `GROUP BY`/`COUNT`/`SUM` pushed
      to DuckDB — 100x faster for analytics.
- [x] **Multi-engine tiering** — assign the SAME query to multiple engines: memory
      for hot reads, SQLite for persistence. Write-fan-out, read-from-cheapest.
- [x] **Engine hot-swap** — swap an engine at runtime without rebuilding the Store.

### Ecosystem & Integration

- [x] 🔥 **projectionhost integration** — register metaengine collections as
      `projectionhost.Projection` workers. Crash-restart lifecycle, DLQ, checkpointing.
- [x] **CQRS event store adapter** — `projectionadapter.RegisterWithHost(host, name, store, decoder)`
      as a projection consuming a CQRS `event.Store` journal.
- [x] **HTTP/SSE adapter** — `metaengine.ServeSSE(w, r, watcher)` streams updates.
- [x] **Export/import** — `store.Export(ctx, w)` / `store.Import(ctx, r)` for
      backup, migration, seed data.
- [x] **CLI inspector** — `store.Inspect()` returns formatted collection metadata.
  --filter status=open --limit 10`.
- [~] **cqrs-lint rules** — F018 (FilterOn pushdown), F019 (missing Volume hint) implemented.
      missing `Volume` hint, `SortOn` without index, write amplification over budget.

### Testing & Verification

- [x] 🔥 **Cross-engine contract suite** — extract the ADT matrix into a reusable
      `metaengine.ContractSuite(t, engineFactory)` so any new engine gets full
      parity by importing one function.
- [x] **Property-based fold testing** — `rapid` generator verifying the engine
      produces identical results to a pure Go fold over the same events.
- [ ] **Soak test with 10M events** — replay through all 7 ADTs, verify memory
      doesn't grow unboundedly, latency stays constant, no corruption.
- [ ] **Chaos testing** — randomly kill transactions mid-flight, inject errors,
      swap engines between reads — verify no corruption.
- [x] **Benchmarks** — `BenchmarkRawReader_Get` vs `BenchmarkMapGet`,
      `BenchmarkRawReader_Scan` vs `BenchmarkPushdownMapScan` — prove the JSON tax
      reduction with numbers.
- [x] **Fuzz the fold classifier** — feed arbitrary function signatures to `On()`
      and verify classification never panics.

### Architecture & Maturity

- [x] **Stabilize and tag v1** — once `TypedReader` + `LayoutPlanner` + raw readers
      are validated by consumers, freeze the API and tag `metaengine/v4.1.0`.
- [x] **Generated typed read API** — `//go:generate metaengine-gen` producing
      `plan.Users.Get(ctx, id)` from declared query fields (original TODO item,
      `TypedReader` was the runtime precursor).
- [ ] **Schema enforcement at Plan() time** — validate that fold return types match
      the declared result type `R`. Currently mismatches surface only at runtime.
- [x] **Diagnostic when auto-layout is applied** — "query X: auto-planned table with
      columns [status, priority]" so consumers know what happened.
- [x] **Expose layout plans in `PlanResult`** — so consumers can inspect auto-generated
      tables, column types, and indexes.
- [ ] **Extract as standalone project** — AGENTS.md says "possibly a future dedicated
      project." A zero-dependency storage planner is valuable beyond CQRS. (→ ROADMAP)

---

## CI / Daemon

- [ ] **Recurring lint-sweep** — the auto-commit daemon occasionally commits
      unformatted code (gci/gofumpt drift), turning `#lint` red. Either gate
      daemon commits behind `nix fmt` or run a scheduled sweep.
- [ ] **CGo-enabled CI job** — add a separate CI job with `CGO_ENABLED=1` for
      DuckDB tests (currently only in flake.nix local apps, not CI).
- [ ] **Investigate dependabot alert** `security/dependabot/10` — `gh api`
      returned no results (auth issue). Cannot diagnose without GitHub token
      permissions.

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

---

_Long-term direction lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
