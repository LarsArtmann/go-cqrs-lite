# Metaengine Remaining Work: Master Plan

> **Date:** 2026-07-31 20:30
> **Trigger:** Post-execution gap analysis after completing all 5 phases (A-E) of the MVP plan
> **Scope:** ALL remaining TODOs from the status report, deduplicated, sorted by impact/customer-value/effort
> **Task size:** Every subtask ≤ 12 minutes
> **Total:** 7 tiers, 52 tasks, 143 subtasks, ~18.5 hours estimated

---

## Sorting Methodology

Each task scored on three axes:
- **Impact** (H/M/L): How much does this improve the system?
- **Customer Value** (H/M/L): How much does a consumer of the library benefit?
- **Effort** (S/M/L): How long to complete?

Sort priority: `Tier → Impact → Customer Value → Effort (ascending)`

---

## TIER 0: BLOCKING — Decisions Requiring User Input

These gate downstream work. Cannot proceed without answers.

| ID  | Decision | Gates | Impact |
|-----|----------|-------|--------|
| D1  | Tag metaengine/v4 + projectionadapter/v4 as v4.3.0 and push? | F12 (tagging), consumer adoption | H |
| D2  | Remove old kv.Materialize projection (`mat`) from taskmanager? | A1 (counter removal), cleanup | M |
| D3  | Make EventDecoder the default decoder (deprecate PayloadDecoder)? | A3 (API change), X8 (docs) | M |

---

## TIER 1: P0-CRITICAL — Correctness, Safety, Data-Loss Prevention

| ID  | Task | Impact | Cust.Val | Effort | Subtasks | Est.Min | Files |
|-----|------|--------|----------|--------|----------|---------|-------|
| C1  | Add PRAGMA busy_timeout=5000 + journal_mode=WAL to metaengine SQLite connection | H | H | S | 3 | 20 | example/taskmanager/metaengine.go |
| C2  | Add EventDecoder unit test in projectionadapter | H | M | S | 4 | 35 | metaengine/projectionadapter/adapter_test.go |
| C3  | Run go mod tidy on example/taskmanager (verify kv dep is clean) | M | L | S | 2 | 10 | example/taskmanager/go.mod |
| C4  | Fix benchkit race condition (TestRunSoak, TestMixedWorkload, TestCompare) | H | M | M | 6 | 60 | benchkit/ |
| C5  | Verify mapupdate_fuzz_test.go compiles + runs clean (gopls false positive?) | L | L | S | 2 | 10 | metaengine/mapupdate_fuzz_test.go |
| C6  | Verify stack/memory go.mod metaengine dep is correct | L | L | S | 1 | 5 | stack/memory/go.mod |

### C1: Add SQLite Pragmas to Metaengine Connection

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| C1.1 | Add `PRAGMA busy_timeout=5000` exec after sql.Open in setupMetaEngine | 5 | Compiles |
| C1.2 | Add `PRAGMA journal_mode=WAL` exec after sql.Open | 5 | WAL mode confirmed |
| C1.3 | Run taskmanager tests to verify no regression | 10 | Tests pass |

### C2: EventDecoder Unit Test

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| C2.1 | Read adapter_test.go to understand existing PayloadDecoder test pattern | 8 | Understand pattern |
| C2.2 | Write test: New() with WithEventDecoder option, verify Handle calls EventDecoder | 10 | Test compiles |
| C2.3 | Assert EventDecoder receives full event.Event (StreamID, Version, Metadata) | 10 | Test passes |
| C2.4 | Assert EventDecoder takes precedence over PayloadDecoder when both set | 7 | Test passes |

### C3: Go Mod Tidy Taskmanager

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| C3.1 | Run `cd example/taskmanager && GOWORK=off go mod tidy -e` | 5 | No errors |
| C3.2 | Verify build still passes: `go build -tags "goexperiment.jsonv2" ./...` | 5 | Builds clean |

### C4: Fix Benchkit Race Condition

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| C4.1 | Run `go test -race -run TestRunSoak -v ./benchkit/` and capture race trace | 10 | Race stack trace captured |
| C4.2 | Analyze race: identify shared mutable state (likely goroutine + map/slice) | 12 | Root cause identified |
| C4.3 | Read benchkit Run/Compare/Soak code to find the unsynchronized access | 10 | Code located |
| C4.4 | Apply fix: mutex, channel, or sync.Map depending on pattern | 10 | Fix applied |
| C4.5 | Run benchkit tests with -race -count=3 | 10 | No race detected |
| C4.6 | Run full benchkit suite without -race to verify no regression | 8 | All pass |

### C5: Verify Mapupdate Fuzz Test

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| C5.1 | Run `go test -tags "goexperiment.jsonv2" -run TestMapUpdate -v ./metaengine/` | 5 | Test passes |
| C5.2 | If gopls error is a false positive, document it; if real, fix the type assertion | 5 | Resolved |

### C6: Stack/Memory Go.Mod Check

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| C6.1 | Run `cd stack/memory && GOWORK=off go mod tidy -e`, verify metaengine dep status | 5 | Correct (direct from test, or indirect) |

---

## TIER 2: P1-HIGH — Prove the Value (Performance, Benchmarks)

| ID  | Task | Impact | Cust.Val | Effort | Subtasks | Est.Min | Files |
|-----|------|--------|----------|--------|----------|---------|-------|
| V1  | Benchmark: metaengine filtered scan vs Materialize.List + Go filter | H | H | M | 5 | 50 | metaengine/ (new bench file) |
| V2  | Add SortOnField("priority", true) to task_views query | M | H | S | 3 | 20 | example/taskmanager/metaengine.go |
| V3  | Cost model calibration benchmarks for task_views query | M | M | M | 4 | 40 | metaengine/ |
| V4  | Metaengine stress test: 100K events, verify scan perf + correctness | H | M | M | 5 | 55 | metaengine/ (new test) |

### V1: Benchmark Metaengine vs Materialize

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| V1.1 | Read existing metaengine benchmarks (if any) and Materialize.List code | 10 | Understand both paths |
| V1.2 | Write BenchmarkMetaEngineFilteredScan: seed 1K/10K tasks, Scan with FilterOnField | 12 | Benchmark compiles |
| V1.3 | Write BenchmarkMaterializeListAndFilter: same data, mat.List() + Go filter | 10 | Benchmark compiles |
| V1.4 | Run both benchmarks, capture ns/op + allocs/op | 10 | Results captured |
| V1.5 | Write results summary (expected: metaengine O(logN) vs Materialize O(N)) | 8 | Summary documented |

### V2: Add SortOnField to Task Views

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| V2.1 | Add SortOnField("priority", true) to task_views Query declaration | 5 | Compiles |
| V2.2 | Update handleListTasks to use WithSort("priority", true) when no explicit sort | 8 | Compiles |
| V2.3 | Run taskmanager tests, verify sorted results | 7 | Tests pass |

### V3: Cost Model Calibration

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| V3.1 | Read cost model code (cost_assignment.go or similar) | 10 | Understand cost formula |
| V3.2 | Write calibration benchmark: vary N (100, 1K, 10K, 100K) for Map.Get, Map.Scan | 12 | Benchmark compiles |
| V3.3 | Run benchmarks, capture cost coefficients | 10 | Results captured |
| V3.4 | Compare actual costs to model predictions, document accuracy | 8 | Summary written |

### V4: Stress Test 100K Events

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| V4.1 | Write TestStress_100KEvents: seed 100K tasks via Apply loop | 12 | Test compiles |
| V4.2 | Verify point-lookup Get on random IDs returns correct data | 10 | All correct |
| V4.3 | Verify filtered Scan by status returns correct count | 8 | Count matches |
| V4.4 | Capture latency: p50/p99 for Get and Scan | 10 | Latency recorded |
| V4.5 | Assert no memory leaks (runtime.MemStats before/after) | 10 | Stable memory |

---

## TIER 3: P1-HIGH — Superb DX (Documentation, Discoverability)

| ID  | Task | Impact | Cust.Val | Effort | Subtasks | Est.Min | Files |
|-----|------|--------|----------|--------|----------|---------|-------|
| X1  | Update AGENTS.md metaengine section (EventDecoder, QueryBuilder, task_views, FilterOnField recipe) | H | H | M | 4 | 45 | AGENTS.md |
| X2  | Add metaengine recipes to SKILL.md (recipes.md) | H | H | M | 4 | 40 | .agents/skills/go-cqrs-lite/references/recipes.md |
| X3  | Document eventWithID wrapper pattern as a recipe | M | M | S | 2 | 15 | .agents/skills/go-cqrs-lite/references/recipes.md |
| X4  | Write migration guide: kv.ViewStore → metaengine (step-by-step) | M | H | M | 4 | 45 | metaengine/MIGRATION.md |
| X5  | Run doc-check verification on new README content | M | L | S | 2 | 15 | cmd/doc-check |
| X6  | Mark EventDecoder as recommended decoder in projectionadapter docs | M | M | S | 2 | 10 | metaengine/projectionadapter/adapter.go (doc), README.md |
| X7  | Document TieredStore and SwapEngine in README | L | M | S | 3 | 25 | metaengine/README.md |
| X8  | Add metaengine cookbook/recipes doc (beyond README examples) | L | M | M | 4 | 40 | metaengine/COOKBOOK.md |

### X1: Update AGENTS.md Metaengine Section

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| X1.1 | Add EventDecoder pattern + WithEventDecoder to AGENTS.md metaengine bullet | 12 | Updated |
| X1.2 | Add QueryBuilder example to Key Patterns section | 10 | Updated |
| X1.3 | Add task_views migration example (eventWithID wrapper, FilterOnField) | 12 | Updated |
| X1.4 | Add FilterOnField migration recipe (kv.Materialize → metaengine Map) | 11 | Updated |

### X2: Add Metaengine Recipes to SKILL.md

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| X2.1 | Read current recipes.md to find insertion point and style | 8 | Pattern understood |
| X2.2 | Write "Filtered Scan with Metaengine" recipe (Map + FilterOnField + SQLite pushdown) | 12 | Recipe written |
| X2.3 | Write "Multi-Engine Distribution" recipe (Counter→Memory, Map→SQLite) | 10 | Recipe written |
| X2.4 | Write "Point Lookup via TypedReader" recipe | 10 | Recipe written |

### X3: Document eventWithID Pattern

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| X3.1 | Write recipe: "Bridging Stream IDs to Map Keys" (eventWithID wrapper) | 10 | Written |
| X3.2 | Add to recipes.md under metaengine section | 5 | Added |

### X4: Migration Guide kv→Metaengine

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| X4.1 | Write "When to Migrate" section (query complexity threshold) | 10 | Written |
| X4.2 | Write step-by-step: declare Query, write folds, register adapter, wire reader | 12 | Written |
| X4.3 | Write "Parallel Run" section (keep both projections during migration) | 10 | Written |
| X4.4 | Write "Cutover" section (switch handlers, remove old projection) | 10 | Written |

### X5: Doc-Check Verification

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| X5.1 | Run `cd cmd/doc-check && GOWORK=off go run . ../../metaengine/README.md` | 8 | Passes or errors listed |
| X5.2 | Fix any invalid import paths or symbol names in README | 7 | All valid |

### X6: Mark EventDecoder as Recommended

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| X6.1 | Update WithEventDecoder doc comment: "recommended for Map ADT queries" | 5 | Updated |
| X6.2 | Update PayloadDecoder doc comment: "sufficient for Counter/Set queries" | 5 | Updated |

### X7: Document TieredStore and SwapEngine

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| X7.1 | Add "TieredStore" section to README (read from hot engine, fallback to cold) | 10 | Written |
| X7.2 | Add "SwapEngine" section to README (live engine migration) | 10 | Written |
| X7.3 | Run doc-check on new content | 5 | Passes |

### X8: Metaengine Cookbook

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| X8.1 | Write "Counter Patterns" cookbook section (dedup, rate limiting, stats) | 10 | Written |
| X8.2 | Write "Map Patterns" cookbook section (CRUD projections, filtered scans) | 12 | Written |
| X8.3 | Write "Multi-Query Patterns" section (fan-out, denormalized views) | 10 | Written |
| X8.4 | Write "Engine Selection Patterns" section (when SQLite vs Memory vs Pebble) | 8 | Written |

---

## TIER 4: P2-MEDIUM — Test Coverage (Integration Tests)

| ID  | Task | Impact | Cust.Val | Effort | Subtasks | Est.Min | Files |
|-----|------|--------|----------|--------|----------|---------|-------|
| T1  | Cursor-based pagination test through TaskReader | M | M | S | 3 | 25 | example/taskmanager/integration_test.go |
| T2  | Watcher integration test in taskmanager | M | M | S | 3 | 30 | example/taskmanager/integration_test.go |
| T3  | ServeSSE integration test | M | M | S | 3 | 30 | example/taskmanager/integration_test.go |
| T4  | WithPrefetch integration test | L | L | S | 2 | 15 | metaengine/ |
| T5  | TypedReader.GetBatch test | L | L | S | 2 | 15 | metaengine/ |
| T6  | StreamScan integration test | L | L | S | 2 | 15 | metaengine/ |
| T7  | Metaengine + projectionhost replay (crash recovery) test | M | M | M | 4 | 45 | metaengine/projectionadapter/ |
| T8  | Metaengine golden test for plan output (stable cost estimates) | M | L | S | 3 | 25 | metaengine/ |
| T9  | SwapEngine live migration test | L | L | M | 3 | 30 | metaengine/ |

### T1: Cursor-Based Pagination Test

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| T1.1 | Read TypedReader.Scan pagination options (WithCursor, WithLimit) | 8 | Understand API |
| T1.2 | Write test: seed 50 tasks, paginate with limit=10, verify all returned | 10 | Test passes |
| T1.3 | Write test: paginate with filter + cursor (status=active, limit=5) | 7 | Test passes |

### T2: Watcher Integration Test

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| T2.1 | Read metaengine Watcher API (Watch, channel semantics) | 10 | Understand API |
| T2.2 | Write test: start watcher on task_views, apply event, verify notification | 12 | Test passes |
| T2.3 | Write test: watcher filters by key (only specific task ID changes) | 8 | Test passes |

### T3: ServeSSE Integration Test

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| T3.1 | Read metaengine ServeSSE API (handler, event format) | 10 | Understand API |
| T3.2 | Write test: start SSE handler, apply event, verify SSE stream receives update | 12 | Test passes |
| T3.3 | Write test: SSE with filter (only status=active changes stream) | 8 | Test passes |

### T4: WithPrefetch Test

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| T4.1 | Read WithPrefetch option (cache warming semantics) | 7 | Understand API |
| T4.2 | Write test: WithPrefetch loads keys into cache, verify faster subsequent Get | 8 | Test passes |

### T5: TypedReader.GetBatch Test

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| T5.1 | Read GetBatch API (multi-key point lookup) | 5 | Understand API |
| T5.2 | Write test: seed 10 tasks, GetBatch with 5 IDs, verify all returned | 10 | Test passes |

### T6: StreamScan Integration Test

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| T6.1 | Read StreamScan API (iter.Seq2 lazy iteration) | 7 | Understand API |
| T6.2 | Write test: seed 1K tasks, StreamScan, count + verify all iterated | 8 | Test passes |

### T7: Crash Recovery Replay Test

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| T7.1 | Read projectionhost replay semantics (checkpoint + journal) | 12 | Understand replay |
| T7.2 | Write test: apply 100 events, save checkpoint, simulate crash (new host) | 12 | Test compiles |
| T7.3 | Verify replay from checkpoint rebuilds metaengine state correctly | 10 | State matches |
| T7.4 | Verify no duplicate applies (idempotency through checkpoint) | 11 | No duplicates |

### T8: Plan Output Golden Test

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| T8.1 | Read Plan() output structure (engine assignment, cost estimate) | 8 | Understand output |
| T8.2 | Write golden test: fixed queries → stable plan output (engine, ADT, complexity) | 10 | Golden matches |
| T8.3 | Add UPDATE_SNAPS instructions in test comment | 5 | Documented |

### T9: SwapEngine Migration Test

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| T9.1 | Read SwapEngine API (live engine swap semantics) | 10 | Understand API |
| T9.2 | Write test: seed Memory engine, swap to SQLite, verify data migrates | 12 | Test passes |
| T9.3 | Verify reads continue during swap (no downtime) | 8 | Reads succeed |

---

## TIER 5: P2-MEDIUM — Observability

| ID  | Task | Impact | Cust.Val | Effort | Subtasks | Est.Min | Files |
|-----|------|--------|----------|--------|----------|---------|-------|
| O1  | Add OpenTelemetry tracing to metaengine Apply/Scan hot paths | M | M | M | 4 | 45 | metaengine/ |
| O2  | Add Prometheus metrics for metaengine (query latency, engine distribution) | M | M | M | 4 | 45 | metaengine/, prometheus/ |
| O3  | Add metaengine.Explain(query) for plan visualization | M | H | M | 4 | 40 | metaengine/ |
| O4  | Add metaengine.Doctor() diagnostic function | L | M | M | 3 | 30 | metaengine/ |

### O1: OTel Tracing

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| O1.1 | Read otel/ module helpers (Tracer, Span creation patterns) | 10 | Understand pattern |
| O1.2 | Add span to Store.Apply (name: "metaengine.apply", attrs: query name, event count) | 12 | Compiles |
| O1.3 | Add span to TypedReader.Scan (name: "metaengine.scan", attrs: query, filter, engine) | 10 | Compiles |
| O1.4 | Add span to TypedReader.Get (name: "metaengine.get", attrs: query, key) | 8 | Tests pass |

### O2: Prometheus Metrics

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| O2.1 | Add Int64Counter "metaengine.query.count" (attrs: query, engine) | 10 | Compiles |
| O2.2 | Add Float64Histogram "metaengine.query.latency" (attrs: query, engine) | 10 | Compiles |
| O2.3 | Wire counters into Scan/Get/Apply paths via TypedMetricsRecorder pattern | 12 | Compiles |
| O2.4 | Write test: verify metrics registered + incremented | 10 | Test passes |

### O3: Explain(query) Plan Visualization

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| O3.1 | Design Explain() API: returns human-readable plan string or struct | 10 | API designed |
| O3.2 | Implement Explain: format engine assignment, ADT, cost, complexity, filter specs | 12 | Compiles |
| O3.3 | Write test: Explain output is stable + readable | 10 | Test passes |
| O3.4 | Document Explain in README | 8 | Documented |

### O4: Doctor() Diagnostic

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| O4.1 | Design Doctor() API: prints plan, engine profiles, table sizes, diagnostics | 10 | API designed |
| O4.2 | Implement Doctor: gather stats from each engine (row counts, memory usage) | 12 | Compiles |
| O4.3 | Write test: Doctor output includes all registered queries + engines | 8 | Test passes |

---

## TIER 6: P2-MEDIUM — Architecture Cleanup

| ID  | Task | Impact | Cust.Val | Effort | Subtasks | Est.Min | Files |
|-----|------|--------|----------|--------|----------|---------|-------|
| A1  | Consider removing Counter query (task_counts_by_status) — Map can derive | L | L | S | 2 | 15 | example/taskmanager/metaengine.go |
| A2  | Consolidate taskCountsInput and listTasksInput | L | L | S | 2 | 10 | example/taskmanager/metaengine.go |
| A3  | ApplyEncoded path for EventDecoder (avoid double decode) | M | L | M | 3 | 30 | metaengine/projectionadapter/ |
| A4  | Investigate GraphBackend delegation to graph.GraphDriver (per ADR-0077) | L | L | M | 3 | 35 | metaengine/ |

### A1: Consider Removing Counter Query

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| A1.1 | Evaluate: is Counter O(1) worth the redundancy? Document decision | 8 | Decision documented |
| A1.2 | If removing: delete counter query + folds, update handleGetStats to scan+count | 7 | Compiles |

### A2: Consolidate Input Structs

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| A2.1 | Check if taskCountsInput and listTasksInput can merge into one struct | 5 | Decision made |
| A2.2 | If yes: merge, update all references | 5 | Compiles |

### A3: ApplyEncoded Path for EventDecoder

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| A3.1 | Analyze: does Handle decode event twice (EventDecoder + ApplyEncoded)? | 10 | Understand flow |
| A3.2 | If yes: add ApplyEncoded path that skips re-decode when EventDecoder already decoded | 12 | Compiles |
| A3.3 | Write test: verify single-decode path produces same results | 8 | Test passes |

### A4: GraphBackend Delegation Investigation

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| A4.1 | Read GraphBackend interface + graph.GraphDriver interface | 12 | Understand both |
| A4.2 | Evaluate: can GraphBackend delegate to graph.GraphDriver via adapter? | 12 | Feasibility assessed |
| A4.3 | Document findings: update ADR-0077 with recommendation | 11 | ADR updated |

---

## TIER 7: P3-LOW — Future Features (Research, Nice-to-Have)

| ID  | Task | Impact | Cust.Val | Effort | Subtasks | Est.Min | Files |
|-----|------|--------|----------|--------|----------|---------|-------|
| F1  | Add Pebble engine to taskmanager setup (third engine option) | L | L | S | 2 | 20 | example/taskmanager/ |
| F2  | Pebble engine FilterOnField support (currently SQLite-only pushdown) | M | L | L | 4 | 50 | metaengine/pebbleengine/ |
| F3  | metaengine.RegisterQuery for runtime query registration | L | M | M | 3 | 35 | metaengine/ |
| F4  | cqrs-lint rule: warn when Map query has filterable fields but no FilterOnField | L | M | M | 4 | 45 | cmd/cqrs-lint/ |
| F5  | Versioned schema migration for metaengine tables (meta_map DDL versioning) | M | L | L | 4 | 50 | metaengine/ |
| F6  | Multi-tenant isolation (collection prefixing) | L | M | L | 3 | 40 | metaengine/ |
| F7  | metaengine.Snapshot() for backup | L | L | M | 3 | 30 | metaengine/ |
| F8  | CLI inspector tool (cqrs-meta: print plan, stats, health) | L | M | L | 4 | 50 | cmd/cqrs-meta/ |
| F9  | Postgres engine for metaengine | L | L | L | 5 | 60 | metaengine/postgresengine/ |
| F10 | Apply layout planning (ADR-0073) to task_views for indexed-column perf | M | L | M | 3 | 40 | metaengine/, example/taskmanager/ |
| F11 | Auto-denormalization research spike (detect federated reads across queries) | L | L | L | 3 | 40 | docs/planning/ |
| F12 | Tag metaengine/v4 + projectionadapter/v4 as v4.3.0 (if D1 approved) | H | H | S | 3 | 20 | git tags |

### F1: Pebble Engine in Taskmanager

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F1.1 | Add pebbleengine to setupMetaEngine (third engine in Plan() call) | 10 | Compiles |
| F1.2 | Verify planner can assign queries to pebble (point-lookup benchmark) | 10 | Planner works |

### F2: Pebble FilterOnField Support

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F2.1 | Read pebbleengine ScanBackend interface + FilterOp/FilterSpec types | 12 | Understand gap |
| F2.2 | Design: how to push filter to Pebble (prefix scan + closure filter, or key encoding) | 12 | Design done |
| F2.3 | Implement FilterOnField in pebbleengine scan path | 12 | Compiles |
| F2.4 | Write test: filtered scan on Pebble engine returns correct results | 10 | Test passes |

### F3: RegisterQuery Runtime Registration

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F3.1 | Design RegisterQuery API: add query to Store at runtime (after Plan) | 12 | API designed |
| F3.2 | Implement: add to query map, assign engine, create fold dispatcher | 12 | Compiles |
| F3.3 | Write test: RegisterQuery after Plan, then Apply + Scan | 10 | Test passes |

### F4: cqrs-lint Rule for FilterOnField

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F4.1 | Read cqrs-lint rule structure (detector interface, rule registration) | 12 | Understand pattern |
| F4.2 | Write detector: find metaengine.Query declarations, check for filterable struct fields | 12 | Detector compiles |
| F4.3 | Write rule: warn "struct has string field X but no FilterOnField" | 10 | Rule fires |
| F4.4 | Write test: verify rule fires on code without FilterOnField, silent with it | 10 | Test passes |

### F5: Versioned Schema Migration

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F5.1 | Design: version stamp in meta_map DDL, migration on engine open | 12 | Design done |
| F5.2 | Implement: ALTER TABLE migration path for schema version bump | 12 | Compiles |
| F5.3 | Write test: open v1 DB, migrate to v2, verify data intact | 12 | Test passes |
| F5.4 | Document migration strategy in README | 10 | Documented |

### F6: Multi-Tenant Isolation

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F6.1 | Design: collection prefix (tenantID + "_" + collection) for key isolation | 12 | Design done |
| F6.2 | Implement: WithTenant option on Store, prefix all keys | 12 | Compiles |
| F6.3 | Write test: two tenants with same collection name, verify isolation | 10 | Test passes |

### F7: Snapshot for Backup

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F7.1 | Design Snapshot() API: export all engine state to serializable format | 10 | API designed |
| F7.2 | Implement: iterate all engines, export key-value pairs | 12 | Compiles |
| F7.3 | Write test: snapshot → restore → verify data matches | 8 | Test passes |

### F8: CLI Inspector Tool

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F8.1 | Create cmd/cqrs-meta/ module with go.mod | 8 | Module created |
| F8.2 | Implement: connect to SQLite DB, read meta_map/meta_counter, print stats | 12 | Compiles |
| F8.3 | Implement: print plan (query → engine → ADT → cost) | 12 | Compiles |
| F8.4 | Implement: health check (table sizes, index status) | 10 | Tool works |

### F9: Postgres Engine

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F9.1 | Design PostgresEngine: same Engine interface, PG-specific DDL + queries | 12 | Design done |
| F9.2 | Implement Map/Counter/Set backends with Postgres SQL | 12 | Compiles |
| F9.3 | Implement FilterOnField pushdown (json_extract → jsonb ->> operator) | 12 | Compiles |
| F9.4 | Write test: integration with testcontainers-go | 12 | Test passes |
| F9.5 | Benchmark: compare SQLite vs Postgres for 100K rows | 10 | Results captured |

### F10: Layout Planning for Task Views

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F10.1 | Read ADR-0073 (layout planning) + LayoutPlan API | 12 | Understand API |
| F10.2 | Declare LayoutPlan for task_views (indexed columns: status, priority) | 12 | Compiles |
| F10.3 | Benchmark: json_extract vs indexed column for filter+sort | 10 | Speedup measured |

### F11: Auto-Denormalization Spike

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F11.1 | Research: detect when multiple queries read overlapping data (federated reads) | 12 | Research done |
| F11.2 | Prototype: suggest denormalized query that pre-joins at write time | 12 | Prototype sketched |
| F11.3 | Document: feasibility, complexity, NP-hardness analysis | 10 | Documented |

### F12: Tag + Push (if D1 approved)

| Sub-ID | Task | Max min | Verification |
|--------|------|---------|--------------|
| F12.1 | Verify git tag -l shows latest: `git tag -l 'metaengine/v4*' \| sort -V \| tail -1` | 5 | Version confirmed |
| F12.2 | Tag: `git tag metaengine/v4 v4.3.0` + `git tag projectionadapter/v4 v4.3.0` | 5 | Tags created |
| F12.3 | Push: `git push origin metaengine/v4 projectionadapter/v4` | 5 | Pushed |

---

## Summary by Tier

| Tier | Description | Tasks | Subtasks | Est.Hours | Blocking? |
|------|-------------|-------|----------|-----------|-----------|
| T0 | Decisions requiring user input | 3 | 0 | 0 | YES |
| T1 | P0-Critical: correctness, safety | 6 | 18 | 2.5 | No |
| T2 | P1-High: prove the value (perf) | 4 | 17 | 2.8 | No |
| T3 | P1-High: superb DX (docs) | 8 | 23 | 3.5 | No |
| T4 | P2-Medium: test coverage | 9 | 24 | 3.3 | No |
| T5 | P2-Medium: observability | 4 | 15 | 2.7 | No |
| T6 | P2-Medium: architecture cleanup | 4 | 10 | 1.5 | No |
| T7 | P3-Low: future features | 12 | 36 | 5.6 | No |
| **TOTAL** | | **52** | **143** | **~21.9** | |

---

## Recommended Execution Order

### Wave 1: Unblock + Critical Fixes (do first, ~2.5h)
C1 → C2 → C3 → C5 → C6 → C4

### Wave 2: Prove the Value (~2.8h, can parallelize with Wave 3)
V1 → V2 → V3 → V4

### Wave 3: Superb DX (~3.5h, can parallelize with Wave 2)
X1 → X2 → X3 → X6 → X5 → X4 → X7 → X8

### Wave 4: Test Coverage (~3.3h)
T1 → T2 → T3 → T7 → T8 → T4 → T5 → T6 → T9

### Wave 5: Observability (~2.7h)
O3 → O1 → O2 → O4

### Wave 6: Architecture Cleanup (~1.5h, after D2/D3 decisions)
A1 → A2 → A3 → A4

### Wave 7: Future Features (~5.6h, after D1 decision for F12)
F12 → F10 → F1 → F2 → F3 → F4 → F5 → F6 → F7 → F8 → F9 → F11

---

## Verification Gate

After each wave, run:
```bash
go build -tags "goexperiment.jsonv2" ./...
go test -tags "goexperiment.jsonv2" -count=1 -race ./metaengine/... ./example/taskmanager/... ./stack/sqlite/...
nix fmt
cd cmd/api-stability && GOWORK=off go run .
```

After final wave:
```bash
nix run .#verify
```
