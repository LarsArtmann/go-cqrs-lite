# Metaengine: How to Make It Actually Superb

> **Revised 2026-07-28** — This is v2. V1 was feature-driven ("implement pushdown, then Pebble, then layout planning"). V2 is **hypothesis-driven**: every phase tests the core research claim, with kill criteria. If the data says the hypothesis is wrong, we stop before wasting effort.
>
> Based on deep review of all design docs, all 21 non-test `.go` files, all 26 test files, and calibration benchmarks.

---

## The Core Hypothesis

> **Cross-engine, deployment-time, cost-based layout optimization for event-sourced data produces measurably better query performance than single-engine or naive multi-engine approaches — and is tractable to compute.**

Everything in this plan serves proving or disproving this hypothesis. If it's false, we need to know *before* investing 4 weeks of implementation.

---

## Part 1: The Brutal Diagnosis

### 1.1 What is genuinely embarrassing

#### E1: The query optimization engine does not optimize queries

Every filtered/sorted query on SQLite loads **all rows** into Go, then filters/sorts/paginates in Go:

```
sqlite_engine.go:222-307 — MapScan:
  query: "SELECT value FROM meta_map WHERE collection = ?"
  → loads every row in the collection
  → json.Unmarshal each into any
  → passesFilters() per row (closure + reflect.Call)
  → sort.Slice in Go
  → cursor window in Go
  → limit+1 truncation in Go
```

No `WHERE`, no `ORDER BY`, no `LIMIT` reach the database. **O(N) for every scan regardless of selectivity.** The engine's own cost profile admits this (`engine.go:124-129`).

For a project whose entire value proposition is "make event-sourced data query-optimal," this is not eating your own dog food.

#### E2: The "cross-engine" claim is theoretical with 2 engines

Memory (volatile, O(1) map) + SQLite (persistent, O(log N) B-tree). These aren't "heterogeneous engines with different cost profiles" — they're "RAM vs disk." The interesting research claim requires at least 3 engines with genuinely different optimization profiles. **Pebble is in every design doc and has zero code.**

#### E3: Every read pays a triple JSON tax

The SQLite read path: `TEXT → json.Unmarshal → map[string]any → reify[R] → json.Marshal → json.Unmarshal → R`. Three JSON operations per row. ADR-0066 calls this "acceptable" — it is not for a performance-focused engine.

#### E4: The "two-level optimization model" is only Level 1

The design docs promise Level 1 (engine assignment) AND Level 2 (within-engine layout: one table + N indexes, not one projection per filter combo). **Only Level 1 exists.** The novel part — layout planning — is eloquently argued in the docs and completely absent from the code.

#### E5: The cost model is unvalidated

Calibration benchmarks measure raw op latency, but **no benchmark validates that the cost model's PREDICTIONS match actual query performance.** Without this, the cost model is a formula, not a model.

---

### 1.2 What is genuinely good (preserve this)

| Strength | Evidence |
| --- | --- |
| **Greedy engine assignment works** | `engine_test.go:58-112` — fakeEngine proves cost-sort + tiebreaker, not first-match |
| **7 ADTs are a clean abstraction** | Map, Set, Counter, Graph, Multimap, Log, SortedMap — ISP-compliant interfaces |
| **Zero-dependency core** | ADR-0062 — stdlib-only. Projection adapter in its own module |
| **Concurrency hardening is real** | ADR-0067 (tx-atomic MapUpdate), ADR-0068 (restart-safe multimap seq) |
| **Degradation diagnostics are honest** | `planner.go:158-194` — graph-scan and filtered-scan degradation detected |
| **Cross-engine parity testing** | `cross_engine_meta_test.go` — deep-equal Map/Set/Counter/SortedMap |
| **Cost model honesty** | `cost.go:3-11` — explicit HONESTY NOTE, no false claims |
| **Projection adapter integration** | Full event flow through projectionhost, tested end-to-end |

---

### 1.3 Ghost systems (dead code)

| Ghost | Location | Verified? | Action |
| --- | --- | --- | --- |
| `EventTypeNames()` | `encoded.go:68` | ✅ Zero non-test consumers | Delete |
| `DiagLevelInfo` | `plan_types.go:13` | ✅ Never emitted by planner | Delete |
| `graphNeighbors` CTE query | `sqlite_engine.go:90` | ✅ Never executed | Delete |
| `DefaultNsPerOp` (exported) | `cost.go:56` | ✅ No external consumer | Unexport |

---

### 1.4 Split brains

| Split Brain | Locations | Fix |
| --- | --- | --- |
| `EventTypes()` vs `EventTypeNames()` | `store.go:40` vs `encoded.go:68` | Delete the dead one |
| Two struct-field introspection systems | `reflectField`/`reflectFields()` vs `colResultInfo`/`collectionResultInfo()` | Consolidate |
| Three reflection deref entry points | `derefType()`, `structType()`, `structValue()` | Consolidate |
| Filter logic split across files | `passesFilters` in `compare.go`, `filterPredicate` in `execute.go` | Co-locate |

---

### 1.5 Vaporware inventory (promised but missing)

| Feature | Design doc promise | Code status | Blocks hypothesis? |
| --- | --- | --- | --- |
| **PushdownScan** (WHERE/ORDER BY/LIMIT → SQL) | ADR-0063 Phase 1 | Zero code | ✅ Yes — cost model lies without it |
| **Streaming reads** | Project definition Phase 3 | Zero code | No (safety, not correctness) |
| **Pebble engine** | Every design doc | Zero code | ✅ Yes — "cross-engine" is hollow without it |
| **Within-engine layout planning** | Assumptions doc, 5-phase planner | Zero code | ✅ Yes — this IS the novelty |
| **Index DDL generation + dedup** | "Don't Be Stupid" rules | Zero code | ✅ Yes — part of layout planning |
| **Cost model validation** | Implicit in "research-grade" | Zero code | ✅ Yes — proves the model |
| **Generated typed read API** | Phase 3 | Zero code | No (ergonomics) |
| **Auto-denormalization** | Phase 2 | Zero code | No (needs 3+ remote engines) |

---

## Part 2: The Revised Plan (Hypothesis-Driven)

### Design principle

> **Test the hypothesis with minimum effort. Kill early if wrong.**

Each phase has:
- **Goal** — what we're testing
- **Deliverable** — concrete artifact
- **Kill criterion** — when to stop
- **Depends on** — prerequisite phases

---

### Phase 0: Cleanup — Zero-risk debt removal

| | |
| --- | --- |
| **Goal** | Remove dead code + split brains so future changes are clean |
| **Deliverable** | Passing tests with 4 ghosts deleted, 2 split brains fixed |
| **Kill criterion** | N/A (always do this) |
| **Depends on** | Nothing |
| **Estimated** | Half a day |
| **Leverages** | N/A |

Tasks:
1. Delete `EventTypeNames()` (`encoded.go:68`) + fix the test in `execution_test.go`
2. Delete `DiagLevelInfo` (`plan_types.go:13`) + fix the test in `planner_test.go:188`
3. Delete `graphNeighbors` query + struct field (`sqlite_engine.go:46,90`)
4. Unexport `DefaultNsPerOp` (`cost.go:56`)
5. Consolidate reflection helpers (`derefType`/`structType`/`structValue` → one)
6. Co-locate filter logic (`filterPredicate` → `compare.go`)

---

### Phase 1: Make measurement honest

| | |
| --- | --- |
| **Goal** | The SQLite engine must actually use SQL optimization so benchmarks measure reality, not the absence of pushdown |
| **Deliverable** | `PushdownScan` interface implemented on SQLite; existing tests pass; new tests prove WHERE/ORDER BY/LIMIT reach the DB |
| **Kill criterion** | N/A — this is a prerequisite, not a hypothesis test |
| **Depends on** | Phase 0 |
| **Estimated** | 2–3 days |
| **Leverages** | `storage/sql/where.go` `BuildWhereClause` (pattern reference, can't import — zero-dep boundary), SQLite `json_extract()` function |

**What to build:**

```go
// New optional interface — engines implement if they support SQL pushdown
type PushdownScan interface {
    PushdownMapScan(ctx, collection string, filters []FilterSpec,
        sort *SortSpec, cursor any, limit int) ([]any, error)
}

type FilterSpec struct {
    Column string   // JSON path: json_extract(value, '$.field')
    Op     FilterOp // Eq, Lt, Gt, Lte, Gte, Ne
    Value  any
}

type SortSpec struct {
    Column string
    Desc   bool
}
```

SQLite `PushdownMapScan` generates:
```sql
SELECT value FROM meta_map
WHERE collection = ? AND json_extract(value, '$.status') = ?
ORDER BY json_extract(value, '$.created_at') DESC
LIMIT ?
```

**Key insight:** `meta_map.value` is a JSON TEXT column. SQLite has `json_extract()` — we push filters into SQL without a schema change. This is a stopgap until Phase 3 generates typed columns.

**Why this phase gates everything:** Without pushdown, the cost model claims O(logN) for SQLite but the engine does O(N). Any benchmark comparing engines would measure the ABSENCE of optimization, not the presence. You can't validate a cost model that's lying.

---

### Phase 2: Make "cross-engine" real

| | |
| --- | --- |
| **Goal** | Add a 3rd engine with a genuinely different cost profile so the planner has a REAL choice |
| **Deliverable** | `pebbleEngine` implementing all 7 ADT backends; cross-engine parity tests pass |
| **Kill criterion** | If Pebble can't beat SQLite on point-lookup writes at any scale, it adds no value — drop it |
| **Depends on** | Phase 0 (clean interfaces) |
| **Estimated** | 3–4 days |
| **Leverages** | `storage/pebble/` PebbleDB wrapper (pattern reference — can't import directly due to zero-dep boundary, but same `cockroachdb/pebble` dependency) |

**Pebble's genuinely different cost profile:**

| ADT | SQLite | Pebble | Memory |
| --- | --- | --- | --- |
| Map | O(log N) B-tree | **O(1) LSM point read** | O(1) hash |
| SortedMap | O(NlogN) load-all-sort | **O(log N) LSM range scan** (but no secondary index!) | O(N) sort |
| Counter | O(1) upsert | **O(N) full scan** (degraded) | O(1) hash |
| Set | O(log N) | O(1) | O(1) |
| Graph | O(N^d) BFS | O(N^d) BFS | O(d) adjacency |

The critical difference: Pebble has **no native secondary indexes**. Point lookups → Pebble wins. Filtered scans → SQLite wins (it has indexes). Counters → Memory wins (Pebble can't aggregate). **This is what makes the planner's choice meaningful.**

**How to test the kill criterion:** Run `BenchmarkCalibration_*` on Pebble. If `PebbleMapGet` is not faster than `SQLiteMapGet` at N≥10K, Pebble adds no value as a Map engine.

---

### Phase 3: THE hypothesis test — Within-engine layout planning

| | |
| --- | --- |
| **Goal** | Test the core novelty: does deployment-time layout planning produce measurably better query performance than naive single-projection-per-query? |
| **Deliverable** | (a) Layout planner that generates DDL from declared query patterns (one table + N deduped indexes). (b) Benchmark comparing planned vs unplanned on the same workload. |
| **Kill criterion** | **If planned layout does not beat unplanned by ≥2× on at least one real workload, the hypothesis is FALSE. Stop and rethink.** |
| **Depends on** | Phase 1 (pushdown — so indexes are actually used), Phase 2 (Pebble — so cross-engine choices exist) |
| **Estimated** | 5–7 days |
| **Leverages** | `storage/view/auto.go` (struct-tag → column inference pattern), `storage/sql/dialect.go` (DDL patterns) |

**What "layout planning" means concretely:**

```
Input: Declare[UserID, UserView]("users",
    On(UserCreated, ...),
    PointLookup[UserID](),
    FilterOn(func(v V) string { return v.Status }),
    SortOn(func(v V) time.Time { return v.JoinedAt }),
    Count(),
)

Planner output (Level 2 — the novel part):
  SQLite: CREATE TABLE users_data (
            key TEXT PRIMARY KEY,
            value TEXT,
            status TEXT,           -- extracted from JSON for indexability
            joined_at TEXT          -- extracted from JSON for indexability
          )
          CREATE INDEX idx_users_status ON users_data(status)
          CREATE INDEX idx_users_joined ON users_data(joined_at)
          -- ONE table, TWO indexes — not three separate projections
          -- RangeFilter + OrderBy on joined_at SHARE idx_users_joined

  Pebble: keyspace "users:by_id"     (Map O(1))
          -- No secondary index possible; filtered scans DEGRADED → reassigned to SQLite
```

**The "Don't Be Stupid" rules, implemented as code:**
1. One table + N indexes, not one projection per (filter, sort) combo.
2. Don't index a column never filtered.
3. Deduplicate indexes (RangeFilter + OrderBy on same column → one index).
4. Don't split across engines when one suffices.
5. Let the engine's own planner handle index intersection for AND queries.

**The benchmark (the kill criterion test):**

```
Workload: 100K users, 5 query patterns (point lookup, status filter, date range,
          sort by date, count by status)

Variant A (naive):     One meta_map collection, all queries use MapScan (O(N) scan)
Variant B (planned):   Layout-planned table with 2 indexes, queries use pushdown

Measure: p50/p99 latency for each query pattern, total write amplification

PASS if: Variant B is ≥2× faster on at least 3 of 5 patterns
FAIL if: Variant B is not meaningfully faster → layout planning adds complexity for no gain
```

---

### Phase 4: Prove the cost model

| | |
| --- | --- |
| **Goal** | Validate that the planner's cost PREDICTIONS correlate with ACTUAL performance across all 3 engines |
| **Deliverable** | Benchmark matrix: {memory, sqlite, pebble} × {7 ADTs} × {100, 10K, 100K, 1M} with predicted vs actual latency |
| **Kill criterion** | If prediction error > 2× for more than 20% of cells, the cost model needs fundamental recalibration — not tuning |
| **Depends on** | Phase 1 (honest SQLite), Phase 2 (Pebble), Phase 3 (layout-planned tables) |
| **Estimated** | 2–3 days |
| **Leverages** | `benchkit/` (benchmark harness — pattern reference, factory-driven suite) |

**What this proves:** The difference between a research paper and a toy. If the model predicts "SQLite O(logN)" and actual is O(logN), the model is sound. If it predicts O(logN) but actual is O(N) (because pushdown wasn't used), the model is lying. Phase 1 fixes the lying; Phase 4 proves it.

---

### Phase 5: Polish to superb (only if Phase 3 passes)

| | |
| --- | --- |
| **Goal** | Make the validated engine production-grade |
| **Depends on** | Phase 3 kill criterion PASSED |
| **Estimated** | 1–2 weeks ongoing |

| Task | Estimated | Why |
| --- | --- | --- |
| Streaming reads (`iter.Seq2`) | 1–2 days | OOM safety for large collections |
| Kill the JSON tax (single-pass decode) | 1 day | Performance — 3 JSON ops → 1 per row |
| Generated typed read API | 2–3 days | Ergonomics — `plan.Users.Get(ctx, id)` instead of `ExecuteTyped[Q,R]` |
| `FilterOnField(name, op)` declarative API | 1 day | ADR-0063 Phase 2 — closure-free specs for pushdown |
| Unified 7-ADT × 3-engine test matrix | 1 day | Test confidence — one parameterized harness |
| Auto-denormalization | 3–5 days | DEFER — only matters with 3+ REMOTE engines (local cross-engine = cheap syscalls) |

---

### The extraction question

The design docs say: *"the meta-engine is a new project. It is not a module within go-cqrs-lite."*

**When to extract:** AFTER Phase 3 validates the core hypothesis. Extracting a concept that doesn't work is worse than keeping a working concept in-repo.

**Extraction criteria (all must be met):**
1. ✅ Core hypothesis validated (Phase 3 passes)
2. ✅ Cost model validated (Phase 4 passes)
3. ✅ API is stable (no breaking changes expected)
4. ✅ At least one real consumer (the case study from Phase 3)
5. ✅ Zero-dependency core proven (already true — ADR-0062)

**Don't extract until all 5 are met.** Until then, the workspace gives free cross-module testing and zero versioning friction.

---

## Part 3: Dependency Graph

```
Phase 0: Cleanup (½ day)
    │
    ├──→ Phase 1: Pushdown (2-3 days) ←── makes SQLite honest
    │         │
    │         └──→ Phase 3: Layout Planning (5-7 days) ←── THE hypothesis test
    │                   │
    │                   └──→ Phase 4: Cost Validation (2-3 days)
    │                             │
    │                             └──→ Phase 5: Polish (ongoing)
    │
    └──→ Phase 2: Pebble Engine (3-4 days) ←── makes cross-engine real
              │
              └──→ Phase 3: Layout Planning (needs 3 engines)
```

**Critical path:** Phase 0 → Phase 1 → Phase 3 → Phase 4 = ~10–13 days to hypothesis validation.
Phase 2 (Pebble) runs in parallel with Phase 1 — it's independent after Phase 0.

**Total to "validated research contribution":** ~2.5–3 weeks.
**Total to "superb engineering product":** ~4–5 weeks (add Phase 5).

---

## Part 4: Type Model Improvements

### 4.1 The `any` problem

`MapSet(ctx, collection, key any, value any)` — deliberate (ADT-generic) but causes the JSON tax and eliminates compile-time safety.

**Improvement (Phase 5):** Typed at declaration, erased at engine boundary:

```go
// Today: untyped
Declare[any, any]("users", On(UserCreated, ...), PointLookup[any]())

// Better: typed declaration, any-erased at engine boundary
Declare[UserID, UserView]("users", On(UserCreated, ...), PointLookup[UserID]())
```

### 4.2 The `FilterOn` closure problem

`FilterOn(func(r R) T)` is opaque — the planner can't inspect it, can't push it down. ADR-0063 rejected reflection-based inspection.

**Improvement (Phase 5):** `FilterOnField(name, op)` as the declarative, closure-free alternative:

```go
// Today: opaque closure
FilterOn(func(v UserView) string { return v.Status })

// Better: declarative spec → generates FilterSpec directly
FilterOnField("status", OpEq)
```

### 4.3 Consolidate introspection

Merge `reflectField`/`reflectFields()` and `colResultInfo`/`collectionResultInfo()` into one `fieldInfo` type (Phase 0).

---

## Part 5: Self-Review

### What did I forget in v1?

V1 was feature-driven. It listed what to BUILD, not what to TEST. The metaengine's value is a research claim; the plan should serve proving that claim with minimum effort. V2 fixes this: Phase 3 is an explicit hypothesis test with a kill criterion.

### What is stupid that we do anyway?

Loading every row from SQLite into Go to filter it. O(N) for every scan. A `WHERE` clause makes it O(log N + k). Phase 1 fixes this.

### Did v1 lie?

V1 presented 4 weeks of implementation as if success were guaranteed. It isn't. The layout planning hypothesis might be FALSE — maybe one table + indexes isn't meaningfully better than naive scan for real workloads at realistic scale. V2 is honest: Phase 3 has a kill criterion.

### How to be less stupid?

Test the hypothesis with minimum effort before investing in polish. Phase 3's benchmark IS the test. If it fails, we saved 2 weeks of Phase 5 work.

### Are we building ghost systems?

5 ghosts found (Section 1.3). Phase 0 kills them.

### Split brains?

4 found (Section 1.4). Phase 0 fixes 2; Phase 5 addresses the rest.

### Tests?

Strong: engine assignment, cost mechanics, degradation, cross-engine parity, concurrency, adapter integration.
Weak: no streaming/pushdown/layout tests (features don't exist), no cost validation, no comparative benchmarks, Graph/Log/Multimap lack cross-engine deep-equal.
Fix: Phase 4 adds cost validation; Phase 5 adds the unified test matrix.

---

## Appendix: Evidence Index

### Embarrassing facts
- `sqlite_engine.go:222-307` — MapScan: full-table scan, no pushdown
- `engine.go:124-129` — cost profile admits O(NlogN) for SortedMap
- `reify.go:18-67` — JSON round-trip on every SQL read (ADR-0066)
- `planner.go:31` — "Each query gets its own independent projection" (no Level 2)

### Ghosts (verified dead)
- `encoded.go:68` — `EventTypeNames()`: zero non-test consumers
- `plan_types.go:13` — `DiagLevelInfo`: never emitted by planner
- `sqlite_engine.go:46,90` — `graphNeighbors`: struct field + query string, never executed
- `cost.go:56` — `DefaultNsPerOp`: exported, no external consumer

### Split brains
- `store.go:40` + `encoded.go:68` — `EventTypes()` vs `EventTypeNames()`
- `reflect.go:76-107` + `collection.go:7-49` — two introspection systems
- `reflect.go:16,35,44` — three deref entry points
- `compare.go:12` + `execute.go:252` — filter logic split

### What's good
- `planner.go:89-156` — real greedy engine selection with cost sorting
- `engine_test.go:58-112` — proves cost-sort, not first-match
- `cost.go:3-11` — honest HONESTY NOTE
- `sqlite_engine.go:172-218` — tx-atomic MapUpdate (ADR-0067)
- `sqlite_backends.go:154-166` — restart-safe multimap seq (ADR-0068)
- `projectionadapter/adapter_integration_test.go:144` — full pipeline test
- `calibration_bench_test.go` — real benchmarks producing real numbers

### Design doc references
- Project definition: `docs/planning/meta-engine-project-definition.md`
- Design/vision: `docs/planning/meta-engine-design.md`
- Assumptions & query planning: `docs/planning/meta-engine-assumptions-and-query-planning.md`
- ADRs: `docs/adr/0061-0068`
