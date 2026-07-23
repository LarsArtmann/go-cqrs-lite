# Analytical vs Transactional Queries in CQRS/Event Sourcing

**Date:** 2026-07-23
**Context:** go-cqrs-lite architecture — how to serve both OLTP and OLAP efficiently in query time and disk storage

---

## The Core Distinction

| Dimension           | Transactional (OLTP)                   | Analytical (OLAP)                 |
| ------------------- | -------------------------------------- | --------------------------------- |
| **Question shape**  | "Get user X" / "Update cart"           | "Revenue by month, by category"   |
| **Data accessed**   | 1 row, or N rows by index              | Millions of rows scanned          |
| **Latency target**  | Sub-ms to low-ms                       | Seconds to minutes                |
| **Concurrency**     | High (thousands of users)              | Low (few batch/report jobs)       |
| **Consistency**     | Read-after-write often needed          | Eventually consistent is fine     |
| **Storage layout**  | Row-oriented, B-tree indexed           | Column-oriented, compressed       |
| **Disk efficiency** | Good for point lookups, poor for scans | 5-10x compression, scan-optimized |
| **Who triggers it** | User request, real-time                | Scheduled job, dashboard, analyst |

---

## Why One Engine Can't Win at Both

This is a physical reality, not a software limitation:

**Row stores** (Postgres, SQLite, Pebble KV) optimize for "fetch this one record fast." Data is laid out contiguously per row. A point lookup touches one disk page. But aggregating `SUM(revenue)` over 10M rows means reading every byte of every row — including columns you don't need.

**Column stores** (ClickHouse, DuckDB, Parquet) lay out data per-column. `SUM(revenue)` reads _only_ the revenue column — 10x less I/O. But fetching "user X's full profile" means jumping across N column files, which is slower than one row read.

You literally cannot optimize for both access patterns in one storage format. The physics of disk I/O forces a choice.

---

## The Traditional Solutions (and their costs)

### 1. One database for everything (naive)

Postgres handles both. Works until analytics queries lock tables, pollute the buffer cache, and starve transactional queries of CPU. Disk storage is row-oriented — analytics scans are 10x slower than they need to be.

### 2. Two databases + ETL (classic data warehouse)

OLTP database (Postgres) serves users. Nightly ETL copies data to OLAP warehouse (Snowflake/BigQuery). Solves query interference. But: storage duplication (2-3x), ETL pipeline complexity, data freshness lag (hours), sync bugs.

### 3. HTAP databases (TiDB, SingleStore, CockroachDB)

One engine with dual storage (row + columnar under the hood). Elegant but: expensive, vendor lock-in, and the columnar tier is usually less mature than dedicated OLAP engines.

---

## How CQRS/Event Sourcing Changes the Game

This is where go-cqrs-lite has a **structural advantage** over traditional architectures:

### The event log is already the perfect integration point

Traditional systems need CDC (Change Data Capture) tools — Debezium, Maxwell, WAL parsing — to extract changes for analytics. Event sourcing systems **already have an append-only log of every state change**. There's nothing to extract. The `event.SeekableJournal` IS the CDC stream.

### Different projections, same events, different shapes

This is the key insight. A single `OrderPlaced` event can be materialized into:

```
OrderPlaced event
    │
    ├──► KV projection (stack.Materialize)     → {"id":"ord-123", "total":49.99}  [point lookup]
    │
    ├──► Relational projection                 → INSERT INTO orders VALUES(...)   [routed queries]
    │    (storage.RelationalProjection)            INSERT INTO daily_sales VALUES(...)  [pre-aggregated]
    │
    ├──► Graph projection (graph.GraphProjection) → MERGE (Order)-[:PLACED_BY]->(User)  [traversal]
    │
    └──► Columnar export (future)              → Parquet/Arrow/ClickHouse          [heavy analytics]
```

Each projection consumes the **same events** but materializes a **different shape optimized for its query type**. No ETL pipeline. No CDC. Just: subscribe to the journal, write to the right store.

### go-cqrs-lite already has 3 of the 4 tiers

| Projection tier | Query type                       | go-cqrs-lite module                   |
| --------------- | -------------------------------- | ------------------------------------- |
| KV/document     | Point lookup                     | `stack.Materialize` + `kv.TypedStore` |
| Relational      | Routed, indexed, pre-aggregated  | `storage.RelationalProjection`        |
| Graph           | Traversal, adjacency, paths      | `graph.GraphProjection`               |
| **Columnar**    | **Aggregation, scan, analytics** | **Not yet — this is the gap**         |

---

## How to Make Both Efficient

### Strategy 1: Pre-aggregate in projections (the biggest win)

Don't compute `SUM(revenue) BY day` at read time. Compute it at **event-processing time**. When `PaymentProcessed` arrives, the analytical projection updates:

```
daily_revenue[2026-07-23][electronics] += 49.99
```

Read time becomes **O(1)** — you're just reading a pre-computed row. The tradeoff: write amplification (one event touches multiple projections) and you must know your analytical queries upfront. But for dashboards and reports, queries ARE known upfront.

**In go-cqrs-lite terms:** `RelationalProjection` with an append-only history table + an aggregation table, both updated atomically in the same handler:

```go
func(ctx, evt, sink) error {
    var p PaymentProcessed
    json.Unmarshal(evt.Payload(), &p)
    sink.Ensure(ctx, "daily_sales", Row{
        "date": p.Date, "category": p.Category,
        "revenue": p.Amount,  // this is an UPSERT that adds
    })
    return nil
}
```

### Strategy 2: Tiered projections — KV for hot path, relational for analytics

The multi-DB presets already enable this:

```go
bundle, _ := sqlite.New(dsn,
    sqlite.WithDSN(
        sqlopt.WithEventDB("events.db"),    // append-optimized
        sqlopt.WithViewDB("views.db"),      // KV: point lookups
        sqlopt.WithAnalyticsDB("olap.db"),  // analytical tables (future)
    ),
)
```

OLTP reads hit `kv.Cache` (sub-microsecond). Analytical reads hit a separate database that doesn't compete for memory or CPU.

### Strategy 3: Embedded columnar for OLAP (DuckDB)

DuckDB is an embedded analytical database — the "SQLite of analytics." A future `storage/duckdb/` module could:

- Subscribe to the event journal
- Materialize events into columnar tables
- Run `SUM`, `GROUP BY`, `JOIN` over millions of rows in milliseconds
- No external service needed (embedded, like SQLite)

This would give go-cqrs-lite a complete HTAP story: KV for OLTP, relational for routing, graph for traversal, columnar for analytics — all from the same event log.

### Strategy 4: Right-size latency expectations

Not every dashboard needs real-time. A nightly materialized view is **10x cheaper** than streaming aggregation. Be smart:

| Query freshness need        | Mechanism                           | Cost      |
| --------------------------- | ----------------------------------- | --------- |
| Real-time (user-facing)     | KV cache, in-process                | Low       |
| Near-real-time (dashboards) | Live projection + pre-aggregation   | Medium    |
| Batch (reports)             | Nightly export to columnar          | Low       |
| Ad-hoc (analyst)            | Direct journal scan or DuckDB query | On-demand |

### Strategy 5: The event log as warehouse feed

For teams that already have a data warehouse (BigQuery, Snowflake, Redshift), the event journal exports directly — no Debezium, no CDC, no WAL parsing:

```
event.SeekableJournal.ReadAll() → JSON Lines → warehouse
```

This is Phase 6 in the benchkit design report (production replay). The export is trivial because events are already self-describing (`Encoding()` stamp, typed payload).

---

## Tradeoffs Worth Considering

| Decision                 | Pro                                  | Con                                  | When to choose                                  |
| ------------------------ | ------------------------------------ | ------------------------------------ | ----------------------------------------------- |
| **Pre-aggregate**        | O(1) reads, no scan cost             | Write amplification, query lock-in   | Known, repeated analytical queries (dashboards) |
| **Scan on demand**       | Flexible, no write cost              | Slow reads, CPU spikes               | Rare, ad-hoc queries                            |
| **Separate columnar DB** | Best OLAP performance                | Extra component, storage duplication | Heavy analytics, data team exists               |
| **Embedded DuckDB**      | No external service, great analytics | CGo or pure-Go maturity              | Single-process apps needing both                |
| **Single row store**     | Simplest, no duplication             | Analytics is 10x slower              | Small scale, light analytics                    |
| **Event log export**     | Leverages existing warehouse         | Freshness lag, pipeline maintenance  | Enterprise with existing BI stack               |

---

## How to Be Smart (Design Principles)

1. **Accept the physics.** One storage format can't optimize for both point lookups and full scans. Design for multiple projections from the start.

2. **The event log is your integration bus.** Every new analytical need = a new projection. No ETL refactoring. This is CQRS/ES's killer feature for analytics.

3. **Pre-aggregate the known 80%.** Most analytical queries are known in advance (dashboards, KPIs). Pay the cost once at write time. Keep raw events for the ad-hoc 20%.

4. **Separate by freshness, not just shape.** Real-time KV, near-real-time pre-aggregated tables, batch columnar exports. Different tiers for different latency budgets.

5. **Compress aggressively for analytical storage.** Columnar formats get 5-10x compression. CBOR (already used for the event log) helps. Disk space is cheap but I/O bandwidth isn't — compression improves both.

6. **Benchmark both dimensions.** The benchkit profiles should measure both transactional patterns (point writes, point reads) and analytical patterns (scan N events, aggregate by dimension, group-by latency). A deployer should be able to answer: "I have 60% point lookups and 40% aggregation queries — which backend minimizes total cost?"

---

## What This Means for benchkit

The current benchkit benchmarks **transactional** workloads (write events, read by aggregate ID, KV get/set). To serve the "which backend for my workload?" question fully, it should also benchmark:

- **Projection catch-up speed**: How fast can a backend replay 100K events into a projection? (This is the analytical write path.)
- **Scan/aggregate latency**: Given a relational projection with 100K rows, how fast is `SELECT category, SUM(total) GROUP BY category`?
- **Pre-aggregation overhead**: What's the write amplification when a projection updates both a KV store and an aggregation table per event?
- **Storage efficiency at scale**: After 1M events, how much disk does each backend use? (Columnar should win here, if we add it.)

This would let a deployer answer: "I have 60% point lookups and 40% aggregation queries — which backend minimizes total cost?"

---

## Conclusion

go-cqrs-lite's projection architecture is already the right foundation. The event log feeds any shape. The gap is the columnar/analytical tier (DuckDB adapter) and benchkit benchmarks that measure both OLTP and OLAP patterns. The deployer-first philosophy means the tool should surface this tradeoff explicitly: "Memory backend wins on point lookups by 3x, but SQLite with WAL wins on journal scans by 8x — choose based on your read ratio."
