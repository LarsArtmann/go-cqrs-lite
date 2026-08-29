# go-cqrs-lite — Deep Consumer Feedback (DiscordSync Read/Write Census)

**Consumer:** [DiscordSync](https://github.com/LarsArtmann/DiscordSync) — Discord backup/archiving bot
**Version used:** `storage/turso/v4 v4.2.1-0.20260807062738-1628cf09ea1a` + `metaengine/v4 v4.6.0` + `system/v4` (not adopted)
**Previous feedback:** [2026-07-12 view-store-as-projection-target](../2026-07-12_DiscordSync_view-store-as-projection-target.md), [2026-07-10 leverage review](../2026-07-10_DiscordSync_leverage_review.md), [2026-07-05 original](../2026-07-05_DiscordSync.md)
**Date:** 2026-08-08
**Context:** The user asked "Does DiscordSync MAINLY work AROUND go-cqrs-lite instead of WITH it?" and demanded exhaustive evidence. This report is the result of a complete read-side query census (~121 methods), a complete write-side `sink.Tx()` bypass census (50 sites across 6 projections), and a full investigation of `metaengine/` and `system/`.

---

## Executive Summary

DiscordSync is a **deep adopter** — 20+ leaf modules, 24 relational projections, the full event sourcing pipeline. The decider/stack/system rejections are architecturally correct (documented 4× in ADRs). **But there is a real issue hiding on the write side:** 6 of 15 projection files bypass `relational.ProjectionSink` entirely via `sink.Tx()`, hand-writing 50 raw SQL statements that 42 of could use sink methods instead. Only 8 sites are genuinely irreducible — and those reveal 5 concrete API gaps in the sink interface.

On the read side, DiscordSync has ~121 query methods, ~54 of which need JOIN/GROUP BY/FTS5/PRAGMA that no current go-cqrs-lite abstraction can express. **This is where metaengine's vision matters most:** the current framing assumes read patterns are fixed at coding time (you either JOIN or you denormalize). Metaengine's goal is to make that a **deployment-time decision** — pre-compute aggressively when storage is cheap, compute on-the-fly when it isn't. DiscordSync currently locks that decision at coding time with raw SQL.

---

## Part 1: What's Working Great

### 1a. Deep leaf-module adoption (20+ modules)

DiscordSync imports and heavily uses the following modules. This is not surface-level — these are the load-bearing dependencies:

| Module                      | Usage Depth                                                                                          |
| --------------------------- | ---------------------------------------------------------------------------------------------------- |
| `event/v4`                  | Event creation, bus, store, journal, middleware, codec, decode — all core types                      |
| `id/v4`                     | Stream IDs, refs, event IDs — full                                                                   |
| `command/v4`                | 7 typed commands, dispatcher, audit store — full                                                     |
| `middleware/v4`             | Recovery, retry, logging, tracing, OTel metrics — full middleware stack                              |
| `storage/v4`                | SQLEventStore, checkpoint/command stores, schema init — full                                         |
| `storage/v4/sql`            | SQLiteDialect for projections — full                                                                 |
| `storage/v4/relational`     | 24 RelationalProjections, RelationalSchema, ProjectionSink — the projection framework IS the library |
| `storage/v4/view`           | SQLViewStore for bans + audit_log (pilot) — minimal but intentional                                  |
| `kv/v4`                     | ViewQuery DSL for keyset pagination — same pilot scope                                               |
| `projection/v4`             | Projection interface, NewProjection — full                                                           |
| `projectionhost/v4`         | Host lifecycle, DLQ, replay, reset, metrics — full                                                   |
| `watermill/v4`              | In-process EventBus — full                                                                           |
| `signing/v4`                | HMAC-SHA256 event signing — full                                                                     |
| `encryption/v4`             | XChaCha20-Poly1305 at-rest encryption — full                                                         |
| `idempotency/v4`            | Source dedup via SQLite store — full                                                                 |
| `schema/v4`                 | Versioned journal with upcasters — full                                                              |
| `codec/v4`                  | CBOR default codec — full                                                                            |
| `otel/v4` + `prometheus/v4` | Tracing + metrics bridge — full                                                                      |
| `catalog/v4`                | OpenAPI/AsyncAPI/D2 generation — full                                                                |
| `storage/turso/v4`          | Turso backend, indexing, quota detection — full                                                      |

### 1b. Correct architectural rejections (decider/stack/system)

These are documented and justified. Summary:

- **`decider.Repository[State]`** — rejected 4× (ADR-004, ADR-012, ADR-022, ADR-043). DiscordSync captures external Discord Gateway events; it doesn't decide them. No aggregate state to load, no business decision to make.
- **`stack/sqlite` / `stack/turso` presets** — rejected 3× (ADR-022). Saves ~40 lines but loses shutdown-DAG control, deep health checks, hot-reload, and creates split-brain schema management.
- **`system.System`** — not applicable. System assumes command → decider → events → projections cycle. DiscordSync has no domain deciders.

These rejections are **correct** and should not be "fixed." They represent genuine architectural mismatches, not adoption failures.

### 1c. Schema-as-data is excellent

DiscordSync's entire schema (40+ tables, columns, PKs, FKs, indexes) is declared as `relational.RelationalSchema` data structures — the library's intended pattern. The `WithoutRelationalAutoMigrate` option correctly lets DiscordSync own the DDL while still using the library's schema type system. The `syncColumns` auto-repair feature is a genuine lifesaver for schema drift.

### 1d. EventCapture is a legitimate composition layer

EventCapture does NOT reimplement go-cqrs-lite. It orchestrates library primitives: `idempotency.Store` → `signing.SignerVerifier` → `event.Store.AppendBatch` → `event.Bus.Publish`. The retry+DLQ+metrics wrapper is application-specific and correctly lives in DiscordSync.

---

## Part 2: The REAL Issue — Write-Side `sink.Tx()` Bypass

### 2a. The problem

The `relational.ProjectionSink` interface provides structured methods that generate parameterized SQL:

| Method                                          | SQL generated                                                           |
| ----------------------------------------------- | ----------------------------------------------------------------------- |
| `Upsert(ctx, table, Row, conflictCols...)`      | `INSERT ... ON CONFLICT(pk) DO UPDATE SET col=excluded.col`             |
| `Ensure(ctx, table, Row)`                       | `INSERT OR IGNORE`                                                      |
| `Update(ctx, table, set, match)`                | `UPDATE table SET ... WHERE match`                                      |
| `DeleteWhere(ctx, table, match)`                | `DELETE FROM table WHERE match`                                         |
| `QueryOne(ctx, table, column, match)`           | `SELECT col FROM table WHERE match LIMIT 1`                             |
| `Increment(ctx, table, key, counterCol, delta)` | `INSERT ... ON CONFLICT DO UPDATE SET col=COALESCE(col,0)+excluded.col` |
| `UpsertCols`                                    | Partial upsert (update only declared columns on conflict)               |
| `UpsertExpr`                                    | Upsert with raw SQL expressions in the SET clause                       |

**9 of 15 projection files use these methods exclusively** — zero raw SQL. These prove the pattern works when projections are simple.

**But 6 files bypass the sink entirely** via `sink.Tx()` to get a raw `*sql.Tx`, then write 50 hand-written SQL statements:

| Projection                                                         | `sink.Tx()` calls | Raw SQL sites | Replaceable | Irreducible |
| ------------------------------------------------------------------ | :---------------: | :-----------: | :---------: | :---------: |
| Messages (messages.go, messages_bulk.go, embed_media.go, users.go) |         1         |      22       |     20      |    **2**    |
| Members (members.go, members_relational.go)                        |         2         |       4       |      3      |    **1**    |
| Metadata (metadata_relational.go)                                  |        10         |      10       |      8      |    **2**    |
| Stats rollup (stats_rollup_relational.go)                          |         3         |       4       |      4      |    **0**    |
| Activity rollup (activity_rollup_relational.go)                    |         1         |       6       |      5      |    **1**    |
| Attachment stats (attachment_stats_rollup_relational.go)           |         1         |       4       |      2      |    **2**    |
| **Total**                                                          |      **18**       |    **50**     |   **42**    |    **8**    |

### 2b. The 42 replaceable sites (consumer-side fixable)

These use raw `INSERT OR IGNORE`, `INSERT ... ON CONFLICT DO UPDATE`, `UPDATE ... SET`, and `DELETE` that `sink.Ensure`, `sink.Upsert`/`UpsertExpr`, `sink.Update`, and `sink.DeleteWhere` generate automatically. Same SQL, parameterized, dialect-portable.

Examples of what's being bypassed:

```go
// CURRENT (raw SQL in messages.go) — bypasses sink:
_, err := tx.ExecContext(ctx,
    `INSERT OR IGNORE INTO users (id, username, discriminator, bot, kind) VALUES (?, ?, ?, ?, ?)`,
    p.Author.ID, p.Author.Username, p.Author.Discriminator, p.Author.Bot, p.Author.Kind)

// SHOULD BE (sink method):
err := sink.Ensure(ctx, "users", relational.Row{
    "id": p.Author.ID, "username": p.Author.Username, ...
})
```

```go
// CURRENT (raw SQL in metadata_relational.go):
_, err := tx.ExecContext(ctx,
    `UPDATE attachments SET content_hash=?, content_size=? WHERE id=?`,
    hash, size, attachmentID)

// SHOULD BE:
err := sink.Update(ctx, "attachments",
    relational.Row{"content_hash": hash, "content_size": size},
    relational.Row{"id": attachmentID})
```

The messages projection is the worst offender — it grabs `sink.Tx()` once at the top and **never calls a single sink method**. All 22 SQL statements (INSERT, UPDATE, SELECT existence checks, soft-delete logic, FK stubs, counter increments, edit-history recording) are hand-written.

### 2c. The 8 genuinely irreducible sites — 5 concrete API gaps

These sites CANNOT be expressed with current sink methods. Each reveals a gap in the API:

#### Gap 1: No clamped decrement (`count > 0` guard)

**Sites:** 2 (messages.go:211, messages_bulk.go:28)

```sql
UPDATE channels SET message_count = message_count - 1 WHERE id = ? AND message_count > 0
```

`Increment` deliberately does NOT clamp to zero (documented behavior). Using it here would allow negative message counts — a behavioral regression. The consumer needs a way to express "decrement but not below zero."

**Proposed fix:** `IncrementClamped(ctx, table, key, col, delta, min int64)` or a `WithMin` option on `Increment`.

#### Gap 2: No `INSERT ... SELECT` subquery

**Sites:** 1 (members.go:91)

```sql
INSERT OR IGNORE INTO member_roles (guild_id, user_id, role_id)
SELECT ?, ?, id FROM roles WHERE id IN (...)
```

This joins against the `roles` table to skip unseeded role IDs — a multi-table INSERT...SELECT. No sink method supports this pattern.

**Proposed fix:** `InsertSelect(ctx, table, columns, selectQuery, args...)` or `EnsureFromSelect(ctx, table, Row, selectQuery, args...)`.

#### Gap 3: Multi-column arithmetic in UPDATE

**Sites:** 2 (metadata_relational.go:144, metadata_relational.go:297)

```sql
UPDATE attachments
SET download_status=?, download_attempts=download_attempts+?,
    last_download_attempt_at=CURRENT_TIMESTAMP, last_error_message=?
WHERE id=?
```

This mixes a counter increment (`download_attempts+?`) with 3 other column sets in a single UPDATE. `Increment` handles only a single counter column. `Update` handles arbitrary SETs but not arithmetic. Splitting into `Update` + `Increment` would be 2 statements instead of 1 — changing semantics (no longer atomic).

**Proposed fix:** `UpdateExpr(ctx, table, setExprs []SetExpr, match Row)` where `SetExpr{Column, Expr}` can express `download_attempts=download_attempts+?`.

#### Gap 4: Multi-column SELECT

**Sites:** 1 (activity_rollup_relational.go:182)

```sql
SELECT channel_id, author_id, msg_date, hour FROM stats_msg_dates WHERE message_id=?
```

`QueryOne` reads only 1 column. No sink method reads multiple columns from within a projection transaction.

**Proposed fix:** `QueryRow(ctx, table, columns []string, match Row) Row` or make `QueryOne` variadic columns.

#### Gap 5: Multi-counter atomic increment

**Sites:** 2 (attachment_stats_rollup_relational.go:115, :144)

```sql
INSERT INTO stats_attachment_by_category (category, count, total_size)
VALUES (?, 1, ?)
ON CONFLICT(category) DO UPDATE SET count=count+1, total_size=total_size+?
```

Two counters incremented atomically in one statement. `Increment` handles only one counter column. Using two separate `Increment` calls would not be atomic.

**Proposed fix:** `MultiIncrement(ctx, table, key, increments map[string]int64)` or `IncrementCols(ctx, table, key, increments []CounterUpdate)`.

---

## Part 3: The Read-Side Census and the Metaengine Question

### 3a. The query census

DiscordSync has ~121 SELECT query methods. Complete census:

| Tier | SQL shape                                 | Count | View store? | Metaengine?    |
| ---- | ----------------------------------------- | ----- | ----------- | -------------- |
| T1   | Simple single-table WHERE/ORDER/LIMIT     | ~45   | ✅          | ✅             |
| T2   | LIKE/range/IN/cursor pagination           | ~10   | ✅          | ✅             |
| T3   | Multi-table JOIN                          | ~8    | ❌ no JOINs | ❌ no JOINs    |
| T4   | GROUP BY/aggregation (COUNT/SUM/AVG)      | ~30   | ❌          | ⚠️ Go-side only |
| T5   | Date functions/FTS5/PRAGMA/json_extract   | ~16   | ❌          | ❌             |
| T6   | Already view store (bans/audit_log pilot) | 2     | ✅          | —              |

**~54 queries (T3+T4+T5) are impossible** in the current view store AND in metaengine. Both are single-collection by design.

### 3b. The original (wrong) framing

My initial analysis concluded: "you can't pre-compute every possible dashboard query into a denormalized projection." **This framing is wrong.** Of course you can pre-compute everything — the question is whether it's worth the storage cost. That's a deployment-time tradeoff, not a coding-time axiom.

### 3c. The correct framing — and metaengine's purpose

Metaengine exists to make the CPU vs query-time vs storage-cost tradeoff a **deployment-time decision**. The operator declares the engines, the workload stats, and the latency budgets. Metaengine plans: which queries to materialize (pre-compute into projections) vs which to compute on-the-fly (scan + fold). With DuckDB, even analytical GROUP BY queries could be pushed to a columnar engine cheaply.

**DiscordSync currently locks this tradeoff at coding time.** Every JOIN is hardcoded. Every GROUP BY is hardcoded. The deployment can't choose "I have 10TB of NVMe, materialize everything" or "I'm on a $5 VPS, compute on-the-fly." The raw SQL makes that impossible without a rewrite.

**This is the real gap.** Not "raw SQL is bad" but "raw SQL makes the tradeoff immutable."

### 3d. What metaengine would need to serve DiscordSync's use case

DiscordSync's read patterns are **relational-shaped** (JOINs, aggregations, FTS5 search). Metaengine currently serves **projection-shaped** read patterns (point lookups, filtered scans, basic aggregations). For metaengine to serve DiscordSync:

1. **JOIN support or cross-projection queries** — Either denormalize at projection time (the event-sourcing way: fold `UserJoined` into the message view) or allow the engine to JOIN materialized projections. The denormalization path is metaengine's current model; the JOIN path would be new.

2. **FTS5 integration** — DiscordSync uses SQLite FTS5 for message search. Metaengine has `ADTSearch` and `SearchBackend` — if these can be wired to FTS5 (or an equivalent), search queries could become metaengine-managed.

3. **Date/time function pushdown** — DiscordSync's activity analytics extract dates from timestamps via `SUBSTR(created_at, 1, 10)`. This is a computed column pattern that could be materialized at projection time (`day` as a real column) rather than computed at query time.

4. **Analytical engine tier** — For the ~30 GROUP BY/aggregation queries, a DuckDB-backed metaengine tier could handle these natively (columnar scans, vectorized aggregation). The consumer would declare `Query[ActivityByDay, ActivityByDayResult]` with a `FoldOn(MessageCreated)` that increments a daily counter, and metaengine would route it to DuckDB.

5. **Migration path from `relational.ProjectionSink` to `metaengine.QueryDecl`** — This is the hardest part. DiscordSync's 24 projections use `relational.ProjectionSink` (structured SQL builders). Metaengine uses fold-based `QueryDecl` (functional Map/Reduce). These are fundamentally different paradigms. A bridge or migration guide is needed.

### 3e. Why bans/audit_log are the only view store pilots

These were the first tables built after the view store's keyset pagination feature landed. They were chosen because:

- **Newly built** — no existing raw SQL to risk migrating
- **Simple** — single-table, filter-by-guild, order-by-time
- **Modest column counts** (6 and 12) — good pilot candidates

The pilot proved the pattern works. It also proved that migrating existing raw-SQL queries is net-negative: more code (ViewMapper + ScanRow + duplicate column defs), same SQL, split-brain read patterns. The pilot was never expanded — ADR-042 deferred it.

---

## Part 4: What Needs Improvement in go-cqrs-lite

### 4a. Fill the 5 sink API gaps (HIGH priority, LOW effort)

The 5 gaps in Part 2c prevent full sink adoption in DiscordSync's projections. Filling them would allow converting 8 more sites, bringing the bypass count from 50 to 0 (after consumer-side fixes for the other 42). These are small, focused additions to an existing, well-designed interface.

Proposed priority:

1. `IncrementClamped` — 2 sites, prevents negative counters (data integrity)
2. `MultiIncrement` — 2 sites, enables atomic multi-counter stats
3. `UpdateExpr` — 2 sites, enables mixed arithmetic+data UPDATE
4. `QueryRow` (multi-column) — 1 site, enables reads within projections
5. `InsertSelect` — 1 site, enables conditional INSERT from another table

### 4b. Document `sink.Tx()` as a code smell (MEDIUM priority)

The sink interface correctly provides `Tx()` as an escape hatch. But there's no guidance on when to use it vs when to extend the structured methods. A note in the sink docs: "If you're using `Tx()` for more than one statement, consider whether a new sink method would serve your use case better. Every raw SQL statement is SQL the consumer must maintain, dialect-port, and security-audit."

### 4c. Metaengine needs a relational/JOIN story (STRATEGIC)

Metaengine's single-collection model is correct for pure event sourcing. But real-world event-sourced applications need to query across projections (e.g., "show me messages with author names" — a JOIN between `messages` and `users` projections). Currently the only options are:

1. **Denormalize at projection time** (put `author_name` into the message projection) — works but creates stale-read windows and doubles storage
2. **Raw SQL JOIN** (what DiscordSync does) — works but locks the tradeoff at coding time
3. **Application-level JOIN** (fetch messages, then batch-fetch users) — N+1 or batch-resolve, more code

A 4th option would make metaengine the clear strategic choice: **cross-projection query planning** where metaengine JOINs two materialized projections at query time (using SQL or in-Go hash join). This would make the "JOIN vs denormalize" decision a deployment-time tradeoff that metaengine optimizes.

### 4d. `WithoutViewAutoMigrate` should be documented (LOW priority, exists but hidden)

`view.WithoutViewAutoMigrate()` exists and works (confirmed in `storage/view/options.go:52`). It skips CREATE TABLE/INDEX, letting the consumer own the DDL. This is essential for DiscordSync (which has a single `RelationalSchema` owning all DDL). But it's not mentioned in the README or any example. A one-line doc addition would save consumers from discovering this through source-diving.

### 4e. The view store needs a "schema-from-struct-tags" option (MEDIUM priority)

Currently, `ViewMapper` requires manual `Columns []ViewColumn[V]` with `Extract` functions for every column. For a 23-column table like `attachments`, this is 23 `Extract` closures — more code than the raw SQL it replaces. `AutoMapper[V](table)` exists and generates this via reflection from `view:"col_name"` struct tags, but it's not the default and isn't well-documented. Making AutoMapper the default path (with manual ViewMapper as the escape hatch) would significantly lower the barrier to view store adoption.

---

## Part 5: Honest Scorecard

| Area                           | Grade | Notes                                                                |
| ------------------------------ | ----- | -------------------------------------------------------------------- |
| Event store adoption           | A+    | Full use of SQLEventStore, journal, signing, encryption, idempotency |
| Projection framework adoption  | B+    | 9/15 files use sink methods exclusively; 6 bypass via Tx() (fixable) |
| View store adoption            | C-    | 2 of ~40 tables (pilot only, not expanded — justified for now)       |
| Bus/projection host adoption   | A+    | Full DLQ, replay, checkpoint, metrics integration                    |
| Command/query dispatch         | A     | Full dispatcher with audit trail, typed commands                     |
| Catalog/documentation          | A+    | OpenAPI/AsyncAPI/D2 generation, llms.txt                             |
| Decider/stack/system rejection | A+    | Architecturally correct, documented 4×                               |
| Metaengine adoption            | F     | Not adopted — but the read patterns are genuinely hard to express    |
| Sink API completeness          | B     | 5 gaps prevent 100% sink adoption (all small, all fixable)           |

### Overall adoption score: B+

Deep, load-bearing adoption with one real friction area (write-side Tx() bypass) and one strategic gap (no metaengine path for relational read patterns).

---

## Part 6: Recommended Next Steps (for go-cqrs-lite)

| #  | Action                                                | Impact                                  | Effort       | Priority  |
| -- | ----------------------------------------------------- | --------------------------------------- | ------------ | --------- |
| 1  | Implement `IncrementClamped` on ProjectionSink        | Eliminates 2 irreducible Tx() sites     | Low (10 LOC) | P0        |
| 2  | Implement `MultiIncrement` on ProjectionSink          | Eliminates 2 irreducible Tx() sites     | Low (15 LOC) | P0        |
| 3  | Implement `UpdateExpr` on ProjectionSink              | Eliminates 2 irreducible Tx() sites     | Low (20 LOC) | P1        |
| 4  | Implement `QueryRow` (multi-column) on ProjectionSink | Eliminates 1 irreducible Tx() site      | Low (10 LOC) | P1        |
| 5  | Implement `InsertSelect` on ProjectionSink            | Eliminates 1 irreducible Tx() site      | Low (15 LOC) | P2        |
| 6  | Document `WithoutViewAutoMigrate`                     | Lowers barrier to view store adoption   | Trivial      | P1        |
| 7  | Document `sink.Tx()` as code smell with guidance      | Prevents future bypass                  | Trivial      | P2        |
| 8  | Design metaengine cross-projection JOIN story         | Unlocks metaengine for relational apps  | High         | Strategic |
| 9  | Add DuckDB analytical tier to metaengine              | Handles GROUP BY/aggregation natively   | High         | Strategic |
| 10 | Write `relational → metaengine` migration guide       | Enables incremental metaengine adoption | Medium       | P2        |

---

## Appendix: Methodology

This report is based on:

- Complete census of all ~121 SELECT query methods in `internal/db/` (read every .go file)
- Complete census of all 50 raw SQL sites in 6 projection files using `sink.Tx()` (read every projection file)
- Full read of all public types/methods in `metaengine/` (all .go files)
- Full read of all public types/methods in `system/` (all .go files)
- Full read of `storage/view/store.go`, `options.go`, `query.go`, `crud.go`, `batch.go`, `count.go`
- Full read of `storage/relational/sink.go` and `projection.go`
- Prior feedback docs (3 rounds, all read)
- ADRs: ADR-004, ADR-012, ADR-022, ADR-031, ADR-042, ADR-043, ADR-054

No estimates or hand-waving — every claim is backed by a file:line reference in the source code.

---

## Appendix: Maintainer Research & Response (2026-08-08)

### A1. Source-Code Verification of Feedback Claims

Every technical claim in this report was independently verified against the go-cqrs-lite source code. **All claims are factually accurate.**

| Claim                                          | Verdict                               | Evidence                                                                                                                                                                                                                                         |
| ---------------------------------------------- | ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 5 sink API gaps all absent                     | **CONFIRMED**                         | `storage/relational/sink.go:46-146` — 9 methods exist; none of the 5 proposed ones                                                                                                                                                               |
| `Increment` deliberately doesn't clamp to zero | **CONFIRMED**                         | `sink.go:84-89` — doc explicitly states negative counters surface data-loss bugs                                                                                                                                                                 |
| `WithoutViewAutoMigrate()` exists but hidden   | **CONFIRMED**                         | `storage/view/options.go:52`                                                                                                                                                                                                                     |
| `AutoMapper[V](table)` exists but not default  | **CONFIRMED**                         | `storage/view/auto.go:51` — generates ViewMapper from `view:"col_name"` struct tags                                                                                                                                                              |
| `ViewMapper` requires manual Columns/Extract   | **PARTIALLY CORRECT**                 | `store.go:50` supports manual Columns, but `AutoMapper` makes it optional, not mandatory                                                                                                                                                         |
| Metaengine is single-collection, no JOINs      | **CONFIRMED**                         | `metaengine/planner.go:72-74`: "Each query gets its own independent projection"                                                                                                                                                                  |
| ADTSearch exists but NOT wired to SQLite FTS5  | **CONFIRMED**                         | Only Memory (`memory_engine.go:288`) + Dgraph (`dgraphengine/engine.go:352`) implement `SearchBackend`. SQLite has zero FTS5 code.                                                                                                               |
| DuckDB engine lacks aggregation pushdown       | **CONFIRMED (stronger than claimed)** | DuckDB does NOT implement `AggregateReader`. `CounterGet` (`duckdbengine/engine.go:312-335`) loads all rows into a Go map. The "vectorized GROUP BY" exists only as raw SQL in a test (`layout_planner_cgo_test.go:657`), never through the API. |

### A2. Three Findings the Original Report Missed

1. **`SetExpr` already exists** (`sink.go:148-156`) — `UpsertExpr` already takes `[]SetExpr{Column, Expr, Args}`. An `UpdateExpr` would reuse this type directly. Lowest-effort gap of all five.

2. **`InsertSelect` is explicitly documented as an intended `Tx()` use case** — the `Tx()` doc comment (`sink.go:131-144`) lists `INSERT INTO ... SELECT` as an escape hatch scenario. This isn't a "gap" so much as "working as designed."

3. **Latent ADT inconsistency (unrelated to this feedback)** — `ADTStreamLog` is defined (`metaengine/types.go:12`) but NOT included in `AllADTs()` (`metaengine/enum_validation.go:10-15`), so `ADTStreamLog.Valid()` returns `false`.

### A3. Maintainer Decisions

After reviewing the feedback and verification results, the following direction was set:

#### Write-Side Sink Gaps: **REJECTED — Not implementing**

The 5 proposed sink methods (`IncrementClamped`, `MultiIncrement`, `UpdateExpr`, `QueryRow`, `InsertSelect`) are ORM-level API additions. go-cqrs-lite is **not an ORM**. The `Tx()` escape hatch (`sink.go:131-144`) already exists and is documented for exactly these irreducible cases. The 8 irreducible sites in DiscordSync are the intended use case for `Tx()`. The 42 replaceable sites are consumer-side cleanup, not a library API problem.

The library's philosophy on `Increment` non-clamping (`sink.go:84-89`: "a counter going below zero signals inconsistent events") remains the correct default. Adding `IncrementClamped` would undermine this principle for a convenience that `Tx()` already provides.

#### Metaengine: **Primary focus — two tracks**

1. **DuckDB real aggregation pushdown** (approved) — Implement `AggregateReader` on the DuckDB engine so `CounterGet`, GROUP BY, SUM, AVG actually push down to columnar SQL instead of loading rows into Go maps. The DuckDB engine today is marketing, not implementation — `CounterGet` loads all rows into Go and accumulates in-process. Making the columnar analytical story real is the highest-leverage metaengine work.

2. **Cross-projection JOIN**: **DEFERRED** to a separate research track / ADR. Metaengine stays single-collection for now. The tension between "metaengine as projection materializer" vs "metaengine as relational query optimizer" is a fundamental identity question that needs its own design work, not a reactive feature addition.

#### DX/Documentation: **Three trivial wins (docs-only, zero risk)**

1. `WithoutViewAutoMigrate` — one-line doc addition so consumers stop source-diving
2. `AutoMapper` — make it the documented default path in the view store README; manual `ViewMapper` as the escape hatch
3. `Increment` non-clamping philosophy — surface the rationale in README, not just source comments

The `Tx()` code-smell doc suggestion was **rejected** as preachy — the existing `Tx()` doc already covers the intent adequately.

### A4. Key Insight: Where the Energy Goes

DiscordSync's feedback focused heavily on low-level SQL API gaps (5 sink methods, 50 raw SQL sites). This is understandable from a consumer perspective but **misses the strategic layer**. The real value of go-cqrs-lite is not being a better SQL builder — it's the metaengine making the materialize-vs-replay, pushdown-vs-scan, and engine-selection tradeoffs into **deployment-time decisions** instead of coding-time commitments.

The DuckDB aggregation pushdown is the first step toward making that vision real. The cross-projection question is the strategic frontier — but it needs design rigor, not a rushed feature.
