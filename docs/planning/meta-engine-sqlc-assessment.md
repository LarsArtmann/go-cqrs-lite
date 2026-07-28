# sqlc (sqlc.dev) in go-cqrs-lite — Honest Assessment

> **Question:** How could we leverage sqlc in this project, especially in the metaengine?
>
> **Status:** Analysis & recommendation (not yet an ADR). No code changed.
>
> **Verdict:** sqlc is an **architectural mismatch for the metaengine core** (the design docs already *rejected* codegen explicitly — ADR-0063 alt-B). There **is** one genuine fit in this repo: the **fixed-schema stores** (eventstore / command / query / snapshot / checkpoint / idempotency / timer). The generic ViewStore / Relational tier in between is a trap.

---

## TL;DR

- **Metaengine core:** Don't. It's a deployment-time, cost-based storage *layout planner*, not a query builder. Keys/values are `any` JSON-encoded into a `TEXT` column; one `meta_map` table namespaces N collections at runtime; the planner assembles the typed read API at `Plan()` time in-process. sqlc solves a *different* problem (type-safe SQL authoring) that the metaengine doesn't have. The 6 SQL strings it uses are already trivially correct and static.
- **Fixed-schema stores:** Candidate. Concrete, build-time-fixed DDL + a fixed, small query set = textbook sqlc profile. Biggest win: gen-time query verification (`db-prepare` vet) on the **event store** (your source of truth).
- **Generic ViewStore / Relational:** Trap. Columns come from `ViewMapper[V]` / `RelationalSchema` at runtime; sqlc only knows columns at gen time. Inverts the library's value proposition.
- **Planner-emitted sqlc (research angle):** Novel ("a CBO that emits its own query layer") but conflicts with runtime re-planning. Spike-only, needs an ADR.

---

## 1. Why sqlc fights the metaengine's core design

The metaengine is not a query builder. It's a **deployment-time, cost-based storage layout planner**. These properties — taken straight from the design docs — are why sqlc can't help:

| Metaengine property (from docs) | Why sqlc breaks it |
| --- | --- |
| **"runtime-composable, NOT codegen-dependent"** (ADR-0063 explicitly rejected codegen as alt-B) | sqlc is **build-time** codegen. Write `.sql` → run sqlc → emit Go. The metaengine assembles the typed read API at `Plan()` time in-process. |
| **Keys/values are `any`, JSON-encoded into a `TEXT` column** (ADR-0061) | sqlc maps a column to a **concrete Go type**. A `TEXT` column gives you `string`. You'd still `json.Unmarshal` afterward — sqlc adds a layer that returns the wrong type (`string`) and you re-do the work. |
| **One `meta_map` table + `collection TEXT` namespacing** serves N collections at runtime | sqlc models **one struct per table**. There is no table-per-collection. The whole `any`/collection design is invisible to sqlc. |
| **"We do NOT reimplement query planning — we set up structures and let the engine optimize"** (assumptions doc) | sqlc doesn't plan either — it just emits prepared calls. It solves a *different* problem (type-safe SQL authoring) that the metaengine doesn't have. |

### The pushdown path (ADR-0063) is where you'd *expect* sqlc to help — and it can't

The one place dynamic SQL matters is `PushdownScan` (`FilterSpec`/`SortSpec` → `WHERE`/`ORDER BY`). But **sqlc generates fixed-arity queries**. You cannot express "0-to-N dynamic `AND` conditions on arbitrary columns with arbitrary operators" in one sqlc annotation. `sqlc.slice('ids')` handles a single `IN (...)`, not a variable filter tree. You'd still need `BuildWhereClause` — sqlc would be dead code on this path.

And the metaengine's *existing* SQL is already trivially correct and static:

```sql
-- These 6 constant strings ARE the entire SQL surface. sqlc can't improve them.
INSERT OR REPLACE INTO meta_map (collection, key, value) VALUES (?, ?, ?);
SELECT value FROM meta_map WHERE collection = ? AND key = ?;
DELETE FROM meta_map WHERE collection = ? AND key = ?;
```

Generating these with sqlc means: adding a toolchain, a `sqlc.yaml`, a `.sql` file, generated files, a `go generate` step — to produce functions that call the identical 3-arg `ExecContext` you already hand-wrote. **Negative ROI.**

> The metaengine's value is the **planner and cost model**, not its SQL. The SQL is deliberately dumb. Investing sqlc here is polishing the part that doesn't matter.

---

## 2. Where sqlc GENUINELY fits: the fixed-schema stores

These modules have **fully-known, build-time-fixed schemas** and a **fixed, small set of queries** — the textbook sqlc profile:

| Store | Schema (fixed at build time) | Typical queries |
| --- | --- | --- |
| `storage/eventstore` | `events(id, stream_id, type, version, payload, encoding, metadata, timestamp)` | Load, LoadFromVersion, LoadToVersion, Append, AppendBatch |
| `storage/` `SQLCommandStore` | commands table (fixed DDL via `dialect.CommandSchema()`) | Save, Load, ReadFrom |
| `storage/` `SQLQueryStore` | queries table | SaveQuery, LoadQueries |
| `storage/eventstore` `SQLSnapshotStore` | snapshots table | Save, Load |
| `storage/eventstore` `SQLCheckpointStore` | checkpoints table | Get, Save |
| `idempotency/sqlstore` | idempotency table | SetIfAbsent, sweep |
| `scheduling` (if SQL) | timers table | Schedule, Cancel, Due, MarkFired |

**Why these are good sqlc candidates:**

- Schemas are concrete and embedded as DDL strings (`dialect.go`: `EventSchema()`, `CommandSchema()`, …) — sqlc can parse them directly.
- Queries are fixed shape — `:one`, `:many`, `:exec` map cleanly.
- You'd trade hand-rolled `fmt.Fprintf` SELECT/scan loops for type-safe `q.AppendEvent(ctx, ...)`.
- The current code scans into columns and re-assembles structs by hand; sqlc does that for you.

**The real benefit here isn't type safety — it's the query verification.** sqlc's `db-prepare` vet rule prepares every query against a live DB at gen time, catching typos / wrong-column bugs *before* runtime. That's genuinely valuable for the event store (your source of truth).

---

## 3. The middle ground that's a trap: the generic ViewStore / Relational tier

`storage/view/store.go` looks tempting (it has Get/Set/Query/Count SQL). **Don't.** It's generic:

```go
type SQLViewStore[V any, K fmt.Stringer] struct { ... }   // columns unknown until runtime
```

The columns come from `ViewMapper[V]` (reflection or `view:"col"` struct tags) **at runtime**. sqlc generates code for a table whose columns are known *at gen time*. To use sqlc here you'd have to run it **once per concrete view type** (`Todos`, `Users`, `Orders`, …) — which means either (a) the library can't ship the generic store, or (b) consumers must run sqlc themselves. That inverts the library's value proposition. The `relational/` sink is the same story (tables declared at runtime via `RelationalSchema`).

This tier is **dynamic-by-design** and that's correct — it's the escape hatch for arbitrary projections. Leave it on the dynamic builder it already has.

---

## 4. A research-grade idea worth a README, not a commitment

Since the metaengine *is* a research project, here's the one creative angle: **planner-emitted sqlc.**

```
metaengine.Plan(engines, declarations)
        │
        ├─ picks tables + indexes + keyspaces (the planner's job today)
        └─ EMITS  schema.sql + queries.sql   (new output)
                │
                └─ sqlc generate  →  typed read package
```

The planner already knows the optimal physical layout. Instead of runtime-assembling the read API from function fields, it could emit `.sql` files and hand them to sqlc, producing a **fully type-safe, zero-reflection read package** per deployment.

**Why this is interesting:** it's a genuine novelty — "a CBO that emits its own query layer." It could be a section in the research paper.

**Why it's probably wrong for v1 (honest caveats):**

- It forces a **rebuild after every re-plan** — kills runtime re-planning, which the docs list as a future goal ("adaptive re-planning by measuring actual cardinality").
- It couples the planner to sqlc's dialect coverage (SQLite is still "beta" in sqlc).
- The docs explicitly chose runtime assembly over codegen for the read API, with a documented fallback reason. Overriding that needs an ADR, not a whim.
- It only helps the SQL engine path; Pebble / Memory / Neo4j get nothing.

**Verdict:** worth a `docs/planning/` exploration note and a spike, not a core dependency.

---

## 5. Decision matrix

| Question | Answer |
| --- | --- |
| Adopt sqlc in **metaengine core**? | **No.** Architectural mismatch; design rejected codegen; SQL is already trivial + correct. |
| Adopt sqlc in **generic ViewStore / Relational**? | **No.** Generic-over-types defeats sqlc; columns known only at runtime. |
| Adopt sqlc in **fixed-schema stores** (eventstore / cmdstore / …)? | **Yes, candidate.** Fixed DDL + fixed queries = textbook fit. Biggest win: gen-time query verification on the event store. |
| Emit sqlc **from the planner** (research angle)? | **Spike only.** Novel but conflicts with runtime re-planning; needs an ADR. |

---

## 6. Recommendation (one alternative, dismissed)

**Start a focused spike: sqlc for `storage/eventstore` only.** It's the highest-value, lowest-risk target — the event store is your source of truth, its schema is fixed and embedded, and gen-time query verification would have real bug-catching value. If that pays off, extend to command / query / snapshot / checkpoint stores (they share `dialect.go` schemas, so it's cheap to replicate).

**Dismiss** the metaengine-core adoption: the design docs already did the analysis for us (ADR-0063 alt-B), and the actual SQL confirms it — 6 dead-simple constant strings that sqlc can only make worse by adding a toolchain.

---

## Appendix: Evidence (file:line references)

### Metaengine SQL — all static, all simple

- `metaengine/sqlite_engine.go:51-92` — `defaultSQLiteQueries()` struct of pre-built constant strings. No `fmt.Sprintf`, no string building. All `?`-parameterized.
- `metaengine/sqlite_engine.go:310-320` — compile-time interface assertions (`sqliteEngine` satisfies all backends).
- `metaengine/sqlite_engine.go:172-218` — `MapUpdate` read-modify-write transaction (ADR-0067). Only 2 methods use explicit transactions; everything else is autocommit.
- `metaengine/sqlite_backends.go:39-54` — `CounterIncrement` batch tx (ADR-0067).
- `metaengine/sqlite_backends.go:154-166` — `nextMultiSeq` lazy `SELECT MAX(seq)` + `sync.Once` seed (ADR-0068).

### No pushdown exists yet (ADR-0063 is a design seam, not implemented)

- `metaengine/execute.go:179-211` — `buildFilterPredicates`: closure-based, `reflect.Call` per row.
- `metaengine/sqlite_engine.go:222-307` — `MapScan`: fixed `SELECT value FROM meta_map WHERE collection = ?`, loads every row, filters/sorts/cursors/paginates all in Go.
- `metaengine/engine.go:124-129` — cost profile comment: "loads every row … until sort-column pushdown lands (ADR-0063)".

### `any`/JSON/`TEXT` design defeats sqlc's type mapping

- `metaengine/sqlite_engine.go:115-133` — `encodeJSON`: `json.Marshal` → TEXT.
- `metaengine/reify.go:18-67` — `reify[R]` / `reifyReflect`: JSON marshal→unmarshal round-trip (ADR-0066).
- `metaengine/execute.go:286-323` — `ExecuteTyped[Q,R]`: SQLite returns `map[string]any`, reified via JSON.

### Existing query infra the metaengine "builds on" (also dynamic, also sqlc-resistant)

- `storage/sql/where.go:14-55` — `BuildWhereClause`: dialect-injected placeholder, `strings.Join(parts, " AND ")`.
- `storage/view/query.go:22-95` — `Query`: `strings.Builder` + `fmt.Fprintf`, keyset cursor.
- `storage/view/query.go:149-200` — `buildKeysetClause`: recursive closure for nested `OR`/`AND` (row-value comparison without Postgres row syntax).
- `storage/view/count.go:19-34` — `Count`: `fmt.Fprintf(&b, "SELECT COUNT(*) FROM %s", ...)`.
- `storage/view/store.go:275-301` — `createTable` / `createIndexes`: `strings.Builder` DDL.
- `storage/relational/sink.go:174-250` — `Upsert` / `Update`: `fmt.Sprintf` with `ON CONFLICT(...) DO UPDATE SET`.
- `storage/relational/sink_advanced.go:92` — `Increment`: `ON CONFLICT DO UPDATE SET col = COALESCE(col, 0) + excluded.col`.
- `storage/sql/dialect.go:15-268` — `Dialect` interface: `Placeholder(int)`, `EventSchema()`, …
- `storage/sql/run_in_tx.go:17` — `RunInTx`.
- `storage/sql/reconstruction.go:47` — `ScanSlice[T]`.
- `storage/sql/duplicate.go:21` — `IsDuplicateKeyError` (3-tier detection).
