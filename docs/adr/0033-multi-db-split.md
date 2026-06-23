# ADR-0033: Multi-Database Split for Concern Isolation

**Date:** 2026-06-23
**Status:** Accepted

## Context

Event-sourced applications have three distinct I/O patterns:

1. **Event store** — append-only writes, sequential reads for replay
2. **Command/query audit** — low-frequency writes, occasional reads
3. **Read models** — random-access reads and writes, scan-heavy

When all three share one database, write-heavy event appends compete with
read-model scans for I/O. Under load, event appends can block behind long
view queries, causing write latency spikes.

The goal explicitly requires: _"ideally multiple SQLite DBs (1 for
Command+ES, 1 for Query, 1 for views)."_

## Decision

Provide `WithEventDB`, `WithQueryDB`, and `WithViewDB` options on all SQL
presets (SQLite, Turso, Postgres). Each opens a separate `*sql.DB`
connection to a different database file (SQLite/Turso) or database name
(Postgres).

**Routing:**

| Option        | Stores routed                  | Rationale                                                 |
| ------------- | ------------------------------ | --------------------------------------------------------- |
| `WithEventDB` | events, snapshots, checkpoints | The event-sourcing write model — append-heavy, sequential |
| `WithQueryDB` | commands, queries              | Operational audit log — low-frequency, independent of ES  |
| `WithViewDB`  | read models (`cqrs_kv`)        | Read-model scans — random-access, scan-heavy              |

When an option is not set, those stores remain in the primary database.

## Alternatives Considered

### Schema-based separation (Postgres only)

Use separate Postgres schemas (`events_schema.events`, `audit_schema.commands`)
within one database. Pros: single connection, simpler lifecycle. Cons:
schema-prefixing every query, no true I/O isolation (same WAL), Postgres-only.

**Rejected:** I/O isolation is the primary goal — schemas don't provide it.

### Table-level separation (single DB, different tables)

All stores in one database with different table names. This is the default
(no multi-DB options). Pros: simplest. Cons: no I/O isolation.

**Kept as default:** Multi-DB is opt-in for deployments that need isolation.

## Consequences

- **More connections:** Each additional DB opens a separate `*sql.DB`. For
  SQLite, this is cheap (file handles). For Postgres, each connection
  consumes a backend process — size connection pools accordingly.
- **No cross-DB transactions:** Events and commands cannot be in the same
  transaction. This is by design — CQRS separates these concerns.
- **Close handles everything:** `bundle.Close()` closes all registered DBs,
  deduplicated by pointer.
- **Contract tested:** `contracttest.RunMultiDBSuite` verifies routing
  correctness for any preset that supports multi-DB.
