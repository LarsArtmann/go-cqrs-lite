# Metaengine: How to Make It Actually Superb

> A brutally honest gap analysis and Pareto-prioritized improvement plan.
> Based on deep review of all design docs (project-definition, design, assumptions, ADRs 0061–0068),
> all 21 non-test `.go` files, all 26 test files, and the calibration benchmarks.
>
> **Date:** 2026-07-28

---

## Part 1: The Brutal Diagnosis

### 1.1 What is genuinely embarrassing

#### E1: The query optimization engine does not optimize queries

The single most damning fact. Every filtered/sorted query on SQLite loads **all rows** into Go memory, then filters/sorts/cursors/paginates in Go:

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

No `WHERE`, no `ORDER BY`, no `LIMIT` reach the database. This is **O(N) for every scan regardless of selectivity**. The engine has a cost profile that honestly admits this:

```
engine.go:124-129 — "loads every row in the collection via the (collection, key)
PK index, then sorts in Go … until sort-column pushdown lands (ADR-0063)"
```

For a project whose **entire value proposition** is "make event-sourced data query-optimal," this is the definition of not eating your own dog food.

#### E2: The "cross-engine" claim is theoretical with 2 engines

Memory (volatile, O(1) map) + SQLite (persistent, O(log N) B-tree). These aren't "heterogeneous engines with different cost profiles" — they're "RAM vs disk." The interesting research claim ("nobody has solved cross-engine view selection") requires at least 3 engines with genuinely different optimization profiles. **Pebble is promised in every design doc but has zero code.**

#### E3: Every read pays a triple JSON tax

The SQLite read path is:

```
TEXT column → json.Unmarshal → map[string]any → reify[R] → json.Marshal → json.Unmarshal → R
```

Three JSON operations per row on the read path. For a performance-focused engine that brands itself on cost optimization. ADR-0066 acknowledges this but calls it an acceptable tradeoff — it is not acceptable for a "superb" engine.

#### E4: The "two-level optimization model" is only Level 1

The design docs (`assumptions-and-query-planning.md`) promise:

- **Level 1 — Engine Assignment** (across engines): which engine serves which ADT.
- **Level 2 — Within-Engine Layout** (inside one engine): what physical structures serve the patterns.

**Only Level 1 exists.** The planner picks WHICH engine but never decides HOW to structure data within it. No DDL generation from declared query patterns, no index planning, no structure selection. The "Don't Be Stupid" rules (one table + N indexes, not one table per filter combo) are eloquently argued in the docs and completely absent from the code. The SQLite schema is 6 hardcoded tables regardless of what queries are planned.

#### E5: The cost model is unvalidated

Calibration benchmarks (`calibration_bench_test.go`) measure raw op latency (MapSet ~466 ns, MapGet ~21 ns → rounded to 500 ns/op). But **no benchmark validates that the cost model's PREDICTIONS match actual query performance.** The whole point of a cost-based optimizer is that its predictions correlate with reality. Without validation, the cost model is a formula, not a model.

---

### 1.2 What is genuinely good (preserve this)

| Strength | Evidence |
| --- | --- |
| **Greedy engine assignment works** | `engine_test.go:58-112` — fakeEngine proves cost-sort + tiebreaker, not first-match. Real engine test assigns point-lookup to memory over SQLite. |
| **7 ADTs are a clean abstraction** | Map, Set, Counter, Graph, Multimap, Log, SortedMap — each with a well-defined interface (`engine.go:38-99`). ISP-compliant. |
| **Zero-dependency core** | ADR-0062 — metaengine core is stdlib-only. The projection adapter is in its own module. Excellent boundary discipline. |
| **Concurrency hardening is real** | ADR-0067 (tx-atomic MapUpdate, tested with 50 goroutines). ADR-0068 (restart-safe multimap seq via sync.Once + SELECT MAX). |
| **Degradation diagnostics are honest** | `planner.go:158-194` — graph-scan and filtered-scan degradation detected and reported. The code admits when it's doing O(N). |
| **Cross-engine parity testing** | `cross_engine_meta_test.go` — deep-equal Map/Set/Counter/SortedMap across memory + SQLite. |
| **Projection adapter integration** | `projectionadapter/adapter_integration_test.go` — full event flow through projectionhost, proves the pipeline works end-to-end. |
| **Cost model honesty** | `cost.go:3-11` — explicit "HONESTY NOTE" admitting it's first-order, not calibrated. No false claims. |

---

### 1.3 Ghost systems (code that serves no purpose)

| Ghost | Location | Action |
| --- | --- | --- |
| `EventTypeNames()` | `encoded.go:68` | Dead duplicate of `EventTypes()` (`store.go:40`). Same result, two methods. **Delete.** |
| `DiagLevelInfo` | `plan_types.go:13` | Defined, exported, **never emitted** by the planner. Only used in one test. **Delete.** |
| `graphNeighbors` recursive CTE query | `sqlite_engine.go:88-90` | Defined in the query struct, **never executed** — GraphNeighbors does BFS in Go via `scanNeighborKeys` instead. **Delete the dead query.** |
| `DefaultNsPerOp` | `cost.go:56` | Exported const, used only as internal fallback. No external consumer. **Unexport.** |
| `Execute()` (untyped) | `execute.go:10` | Test-only convenience wrapper. Real code uses `ExecuteCtx`/`ExecuteTyped`. **Keep but document as test-helper.** |

---

### 1.4 Split brains

| Split Brain | Locations | Fix |
| --- | --- | --- |
| `EventTypes()` vs `EventTypeNames()` | `store.go:40` vs `encoded.go:68` | Same result, two methods. Consolidate to `EventTypes()`. |
| Two struct-field introspection systems | `reflectField`/`reflectFields()` (`reflect.go:76-107`) vs `colResultInfo`/`collectionResultInfo()` (`collection.go:7-49`) | Both reimplement "iterate exported struct fields" with their own types. Extract a shared `fieldInfo` type. |
| Three reflection deref entry points | `derefType()` (`reflect.go:35`), `structType()` (`reflect.go:44`), `structValue()` (`reflect.go:16`) | Overlapping deref logic. Consolidate. |
| Filter logic split | `passesFilters` in `compare.go:12`, `filterPredicate` in `execute.go:252` | Same concept, two files. Move to one location. |

---

## Part 2: What's Promised but Missing (Vaporware Inventory)

| Feature | Design doc promise | Code status | Impact |
| --- | --- | --- | --- |
| **PushdownScan** (WHERE/ORDER BY/LIMIT → SQL) | ADR-0063 Phase 1 ("now") | **Zero code.** ADR pseudocode only. | 🔴 Critical — the engine is O(N) without it |
| **Streaming reads** (`iter.Seq2`) | Project definition Phase 3 | **Zero code.** Not even an interface stub. | 🔴 Critical — OOM risk on large collections |
| **Pebble engine** | Every design doc, deployment walkthroughs B/C/D | **Zero code.** No file, no profile. | 🔴 High — "cross-engine" claim is hollow without it |
| **Within-engine layout planning** (Level 2) | Assumptions doc, 5-phase planner algorithm | **Zero code.** Planner stops at engine assignment. | 🔴 High — the research-novel part is missing |
| **Index DDL generation + dedup** | Assumptions doc, "Don't Be Stupid" rule #1 | **Zero code.** Schema is hardcoded. | 🟡 Medium |
| **Generated typed read API** | Design doc Phase 3, `store.Users.Get(ctx, id)` | **Zero code.** `ExecuteTyped[Q,R]` does runtime switch dispatch. | 🟡 Medium — ergonomics |
| **Auto-denormalization** | Project definition Phase 2 | **Zero code.** Write-amp counting exists, but no cross-engine denorm. | 🟡 Medium — only matters with 3+ remote engines |
| **Cost model validation** | Implicit in "research-grade" claim | **Zero code.** Calibration measures raw ops, not prediction accuracy. | 🟡 Medium — research credibility |
| **Query expression tree** (OR, nested predicates) | Project definition Phase 1 | **Partial.** Only AND via `FilterOn`. `RawWhere` is the escape hatch. | 🟢 Low — AND covers most real queries |
| **Runtime degradation detection** | Design doc, future item | **Zero code.** Plan-time static analysis only. | 🟢 Low — nice-to-have |
| **Scale-dependent structure selection** | Assumptions doc, threshold tables | **Partial.** Emits warnings, does NOT select different structures. | 🟢 Low — warnings are honest enough for v1 |
| **Engine plugin registration** | Design doc, `Register()`/`Open(cfg)` | **Zero code.** Engines are passed as struct literals. | 🟢 Low — ergonomics |

---

## Part 3: The Pareto Plan (80/20 Impact Sort)

### Tier 0: Kill the ghosts (half a day, zero risk)

Before adding anything, clean the dead code:

1. **Delete `EventTypeNames()`** — dead duplicate of `EventTypes()`.
2. **Delete `DiagLevelInfo`** — never emitted, test-only.
3. **Delete the `graphNeighbors` recursive CTE query string** — dead code.
4. **Unexport `DefaultNsPerOp`** — no external consumer.
5. **Consolidate reflection helpers** — merge `derefType`/`structType`/`structValue` into one.
6. **Move `filterPredicate` next to `passesFilters`** — same concept, one file.

---

### Tier 1: Stop being embarrassing (highest ROI)

#### P0: Pushdown — THE priority ⭐

**Without this, the metaengine is a query optimizer that doesn't optimize queries.**

Implement ADR-0063 Phase 1:

```go
// New interface — engines implement if they support SQL pushdown
type PushdownScan interface {
    PushdownMapScan(ctx, collection string, filters []FilterSpec,
        sort *SortSpec, cursor any, limit int) ([]any, error)
}

type FilterSpec struct {
    Column string    // extracted from the closure's target field name
    Op     FilterOp  // Eq, Lt, Gt, Lte, Gte, Ne
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

**Key insight:** `meta_map.value` is a JSON TEXT column. SQLite has `json_extract()` — we can push filters into SQL without changing the schema. This is a stopgap until layout planning (P4) generates typed columns.

**Impact:** Transforms SQLite SortedMap from O(NlogN) → O(logN + k) for filtered queries. Makes the cost model honest.

**Estimated:** 2–3 days.

#### P1: Streaming reads

```go
type StreamBackend interface {
    MapStream(ctx, collection string, filters []FilterSpec,
        sort *SortSpec, cursor any) iter.Seq2[any, error]
}
```

Use Go 1.23+ `iter.Seq2` (the repo is on Go 1.26). Never materialize full result sets. The `StreamsResults` field already exists in `EngineProfile` — wire it.

**Impact:** Eliminates OOM risk. Makes the engine production-safe for large collections.

**Estimated:** 1–2 days.

#### P2: Kill the JSON tax

Three options, in increasing effort:

- **Option A (quick):** Single-pass decode. Currently `json.Unmarshal → any → json.Marshal → json.Unmarshal → R`. Collapse to `json.Unmarshal → R` directly when the target type is known (it is — `ExecuteTyped[Q,R]` has `R`).
- **Option B (medium):** CBOR instead of JSON. `fxamacker/cbor` is already in the repo's `codec/` module. CBOR is ~35% smaller and faster to decode.
- **Option C (proper):** Typed columns via layout planning (P4). Store `status`, `created_at` as real SQL columns, not JSON blobs. This is the real fix but depends on P4.

**Recommendation:** Do Option A now (1 day), Option C when P4 lands.

---

### Tier 2: Make the research claim real

#### P3: Pebble engine

The repo already has `storage/pebble/` (PebbleDB wrapper). The metaengine needs its own `pebbleEngine` following the same `Engine` interface pattern as `memoryEngine`/`sqliteEngine`.

Pebble's genuinely different cost profile:
- Map: O(1) point lookups (LSM-tree, faster than SQLite B-tree for writes)
- SortedMap: O(logN) but **no native secondary indexes** (unlike SQLite) — one keyspace per lookup key
- Counter: O(N) full scan (no aggregation) — **degraded** vs SQLite
- Graph: O(N^d) — **degraded**, same as SQLite

This is what makes "cross-engine" real: 3 engines, 3 different optimization surfaces. The planner now has a REAL choice: point lookups → Pebble O(1), filtered scans → SQLite O(logN), counters → Memory O(1) volatile or SQLite rollup O(1).

**Estimated:** 3–4 days (follow the sqliteEngine pattern, Pebble KV interface already exists).

#### P4: Within-engine layout planning (Level 2) — THE research contribution

This is the part that no existing tool does. The design docs argue it eloquently; the code doesn't implement it.

**What it does:** Given declared query patterns for a projection, generates the optimal physical layout:

```
Input: Declare[V,K]("users",
    On(UserCreated, ...),
    PointLookup[K](),
    FilterOn(func(v V) string { return v.Status }),
    SortOn(func(v V) time.Time { return v.JoinedAt }),
    Count(),
)

Planner output (Level 2):
  SQLite: CREATE TABLE users (key TEXT PRIMARY KEY, value TEXT,
            status TEXT, joined_at TEXT)  -- extracted columns
          CREATE INDEX idx_users_status ON users(status)
          CREATE INDEX idx_users_joined ON users(joined_at)
          -- ONE table, TWO indexes, not three projections
          -- RangeFilter + OrderBy on joined_at SHARE idx_users_joined

  Pebble: keyspace "users:by_id" (Map O(1))
          keyspace "users:by_status" (SortedMap — degraded, O(N))
          -- Pebble has no native secondary index; one keyspace per lookup key
```

**The "Don't Be Stupid" rules, implemented:**
1. Don't create separate projections when one table + indexes suffices (SQLite `NativeIndex: true`).
2. Don't index a column that's never filtered.
3. Deduplicate indexes (RangeFilter + OrderBy on same column → one index).
4. Don't split across engines when one suffices.

**This replaces the hardcoded `meta_map`/`meta_set`/etc. schema** with generated DDL. The ADT tables become the fallback (collection-namespaced generic tables); the layout-planned tables are the optimized path.

**Estimated:** 5–7 days. This is the hard part, but it's also THE differentiator.

#### P5: Cost model validation

Build a benchmark suite that answers: **"Does the cost model predict reality?"**

```
For each (engine, ADT, volume) ∈ {memory, sqlite, pebble} × {7 ADTs} × {100, 10K, 100K, 1M}:
    1. Measure actual query latency (ns)
    2. Compare to predicted latency (cost model's EstimatedLatencyMs)
    3. Compute prediction error: |predicted - actual| / actual
```

If prediction error > 2× for any cell, the cost model is wrong and needs recalibration. This is what separates a research paper from a toy.

The repo already has `benchkit/` — use it for the harness.

**Estimated:** 2–3 days.

---

### Tier 3: Polish to superb

#### P6: Generated typed read API

Replace the runtime switch dispatch in `executeQuery()` with function-field structs wired at `Plan()` time:

```go
// Today (runtime reflection dispatch):
result, err := metaengine.ExecuteTyped[GetUser, UserResult](ctx, store, GetUser{ID: id})

// Superb (Plan-time assembled, zero reflection on hot path):
type UserStore struct {
    Get     func(ctx context.Context, id UserID) (*UserView, error)
    ByStatus func(ctx context.Context, status string) ([]*UserView, error)
}
users := plan.Users  // wired at Plan() time
user, err := users.Get(ctx, id)
```

This is what the design docs promise. The `queryRuntime` already holds the routing decision — extend it to hold the function field.

**Estimated:** 2–3 days.

#### P7: Auto-denormalization

Only matters when you have 3+ **remote** engines (the docs correctly note local cross-engine reads are cheap syscalls). For v1 with local engines (memory + sqlite + pebble all in-process), this is YAGNI. Defer until someone deploys with ClickHouse + Neo4j.

**Estimated:** 3–5 days (defer).

#### P8: Unified ADT × engine test matrix

One table-driven test iterating all 7 ADTs through all available engines:

```go
DescribeTable("ADT cross-engine parity", func(adt adtCase, eng engineFactory) {
    // Apply events → ExecuteTyped → deep-equal against expected
},
    Entry("Map on memory", mapCase, memoryEngine),
    Entry("Map on sqlite", mapCase, sqliteEngine),
    Entry("Map on pebble", mapCase, pebbleEngine),
    Entry("Counter on memory", counterCase, memoryEngine),
    // ... 21 entries (7 ADTs × 3 engines)
)
```

Currently tests exist piecemeal but no single parameterized harness. Graph, Log, and Multimap lack full Apply→Execute cross-engine deep-equal.

**Estimated:** 1 day.

---

## Part 4: Type Model Improvements

### 4.1 The `any` problem

The metaengine uses `any` for keys and values throughout (`MapSet(ctx, collection, key any, value any)`). This is deliberate (ADT-generic) but causes:
- The JSON TEXT tax (can't store typed columns without knowing the type)
- The reify hack (JSON round-trip to bridge SQL→typed Go)
- No compile-time safety on key/value types

**Improvement:** Introduce a `Collection[K, V]` type parameter at the declaration level:

```go
// Today: untyped
Declare[any, any]("users", On(UserCreated, ...), PointLookup[any]())

// Better: typed at declaration, erased at the engine boundary
Declare[UserID, UserView]("users", On(UserCreated, ...), PointLookup[UserID]())
```

The engine boundary still uses `any` (SQLite stores JSON TEXT regardless), but the declaration API and the read API are fully typed. This eliminates `reify[R]` entirely — the type is known at compile time.

### 4.2 The `FilterOn` closure problem

`FilterOn(func(r R) T)` stores a typed closure that's invoked via `reflect.Call` per row. This is:
- Slow (reflection on every row)
- Opaque (the planner can't see what column is being filtered — it's inside a closure)
- The reason pushdown is hard (ADR-0063 rejected reflection-based closure inspection)

**Improvement:** Add `FilterOnField(name, op)` as the declarative, closure-free alternative:

```go
// Today: opaque closure (planner can't inspect, can't push down)
FilterOn(func(v UserView) string { return v.Status })

// Better: declarative spec (planner sees the column, pushes to SQL)
FilterOnField("status", OpEq)
```

This is ADR-0063 Phase 2. The closure API stays for in-memory engines; the field-based API generates `FilterSpec` directly.

### 4.3 The two introspection systems

Consolidate `reflectField`/`reflectFields()` and `colResultInfo`/`collectionResultInfo()` into one:

```go
type fieldInfo struct {
    Name     string
    Type     reflect.Type
    Index    []int
    JSONName string  // from struct tags
}

func inspectStruct(t reflect.Type) []fieldInfo { ... }
```

Both systems need the same thing: "iterate exported struct fields, get name + type + tag." One function, one type.

---

## Part 5: Self-Review (the skill's 11 questions)

### 1. What did you forget?

The metaengine forgot to implement its own value proposition. The design docs describe a cost-based **layout** optimizer; the code implements a cost-based **engine selector**. The layout part — the novel part — is vaporware. The pushdown gap means the engine doesn't even use the database's own query optimization.

### 2. What is something stupid that we do anyway?

Loading every row from SQLite into Go memory to filter/sort it. This is O(N) for every scan. A `WHERE` clause would make it O(log N + k). The fact that a "query optimization engine" does this is embarrassing.

### 3. What could you have done better?

The pushdown seam (`PushdownScan` interface, `FilterSpec`/`SortSpec`) should have been implemented alongside the SQLite engine, not deferred to a future ADR. Without it, the SQLite engine is a proof-of-concept, not a production backend.

### 4. What could you still improve?

Everything in the Pareto plan above. The priority order is: **pushdown → streaming → Pebble → layout planning → cost validation**. That sequence takes the metaengine from "working prototype" to "research contribution."

### 5. Did you lie to me?

The design docs are honest (the HONESTY NOTE in `cost.go`, the degradation diagnostics, the ADR candor). But the **project framing** oversells: "cross-engine view selection" with 2 engines (one volatile) is not yet cross-engine. "Cost-based optimizer" without pushdown is not yet optimizing. The gap is between the vision docs and the implementation, not between claims and reality.

### 6. How can we be less stupid?

Stop adding new ADT features until pushdown exists. The single highest-leverage change is making the SQLite engine actually use `WHERE`/`ORDER BY`/`LIMIT`. Everything else is secondary.

### 7. Is everything correctly integrated or are we building ghost systems?

Five ghost systems found (Section 1.3). The projectionadapter is correctly integrated and tested. The calibration benchmarks are real but don't validate the cost model's predictions.

### 8. Are we focusing on the scope creep trap?

Slightly. Auto-denormalization (P7) is scope creep for a 2-engine system with local engines. Runtime degradation detection is nice-to-have but not core. Focus should narrow to: pushdown, streaming, Pebble, layout planning.

### 9. Did we remove something that was actually useful?

No. The dead code (`EventTypeNames`, `DiagLevelInfo`, `graphNeighbors` CTE) was never useful — it was preemptively added API that never got consumers.

### 10. Did we create any split brains?

Four split brains found (Section 1.4). The worst is `EventTypes()` vs `EventTypeNames()` — two methods producing the same result.

### 11. How are we doing on tests?

**Strong:** engine assignment, cost model mechanics, degradation detection, cross-engine parity, concurrency hardening, projection adapter integration, cursor/pagination.

**Weak:** no streaming tests (feature doesn't exist), no pushdown tests (feature doesn't exist), no cost model validation, no comparative memory-vs-sqlite benchmarks, Graph/Log/Multimap lack full cross-engine deep-equal.

**Improve:** implement the unified 7-ADT × N-engine test matrix (P8). Add cost model validation benchmarks (P5). These two additions would transform test confidence from "the pieces work" to "the system's predictions match reality."

---

## Part 6: Execution Sequence

```
Week 1: Tier 0 (ghost cleanup, half a day)
        + P0 Pushdown (2-3 days) ⭐
        + P1 Streaming (1-2 days)
        → The engine now actually optimizes queries and doesn't OOM

Week 2: P2 Kill JSON tax — Option A (1 day)
        + P3 Pebble engine (3-4 days)
        → "Cross-engine" is now real: 3 engines, 3 profiles

Week 3: P4 Within-engine layout planning (5-7 days) ⭐
        → THE research contribution. One table + N indexes, not N projections.

Week 4: P5 Cost model validation (2-3 days)
        + P8 Unified test matrix (1 day)
        + P6 Generated typed read API (2-3 days)
        → Research-grade: predictions validated, API ergonomic, tests comprehensive
```

**Total: ~4 weeks of focused work to take the metaengine from "working prototype" to "actually superb."**

The two starred items (⭐) are the ones that matter most. Pushdown makes the engine honest. Layout planning makes it novel. Everything else is supporting infrastructure.

---

## Appendix: Evidence Index

### Embarrassing facts
- `sqlite_engine.go:222-307` — MapScan: full-table scan, no pushdown
- `engine.go:124-129` — cost profile admits O(NlogN) for SortedMap
- `reify.go:18-67` — JSON round-trip on every SQL read (ADR-0066)
- `planner.go:31` — "Each query gets its own independent projection" (no Level 2)

### Ghost systems
- `encoded.go:68` — `EventTypeNames()` dead duplicate
- `plan_types.go:13` — `DiagLevelInfo` never emitted
- `sqlite_engine.go:88-90` — `graphNeighbors` CTE query string, never executed
- `cost.go:56` — `DefaultNsPerOp` exported, no external consumer

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
