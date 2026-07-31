# ADR-0080: Dialect interface expansion for cross-database upsert support

|             |                                                                                  |
| ----------- | -------------------------------------------------------------------------------- |
| **Status**  | Accepted                                                                         |
| **Date**    | 2026-07-31                                                                       |
| **Context** | The `Dialect` interface had no upsert/quoting methods; 11 SQL sites hardcoded `ON CONFLICT` syntax, preventing MySQL support |

## Context

The `storage/sql.Dialect` interface abstracted placeholder style, schema DDL,
and error classification across PostgreSQL, SQLite, and DuckDB. However, the
**upsert SQL generation** (`ON CONFLICT ... DO UPDATE SET col = excluded.col`)
was hardcoded at 11 call sites across the storage layer:

- `storage/eventstore/snapshot.go` — snapshot save
- `storage/sql/helpers.go` — checkpoint save
- `storage/readmodel/kv_sql.go` — KV store upsert
- `storage/timer_store.go` — timer schedule
- `storage/relational/sink.go` + `sink_advanced.go` + `sink_helpers.go` — relational projection upserts
- `storage/aggregate_projection.go` — stream projection handle
- `storage/view/crud.go` + `batch.go` — view store upserts

MySQL/MariaDB use fundamentally different upsert syntax:
`ON DUPLICATE KEY UPDATE col = VALUES(col)` instead of
`ON CONFLICT(cols) DO UPDATE SET col = excluded.col`.

Additionally, MySQL reserves the word `key` (used as a column name in `cqrs_kv`),
requiring backtick-quoted identifiers: `` `key` `` instead of `key`.

## Decision

**Expand the `Dialect` interface with 4 new methods** that centralize upsert
SQL generation and identifier quoting:

```go
type Dialect interface {
    // ... existing methods ...

    // ExcludedRef returns the reference to the "excluded"/"new" row in an
    // upsert's UPDATE clause. PostgreSQL/SQLite/DuckDB: "excluded.col".
    // MySQL: "VALUES(col)".
    ExcludedRef(col string) string

    // OnConflictDoNothing generates the conflict-resolution clause for an
    // INSERT that should silently ignore duplicates. PostgreSQL/SQLite:
    // "ON CONFLICT DO NOTHING". MySQL: "ON DUPLICATE KEY UPDATE col = col"
    // (self-assignment no-op, since MySQL lacks ON CONFLICT DO NOTHING).
    OnConflictDoNothing(noOpCol string) string

    // OnConflictDoUpdate generates the conflict-resolution clause for an
    // INSERT that should update on duplicate. PostgreSQL/SQLite:
    // "ON CONFLICT (cols) DO UPDATE SET setExprs". MySQL: no conflict
    // target needed — "ON DUPLICATE KEY UPDATE setExprs".
    OnConflictDoUpdate(conflictCols, setExprs []string) string

    // QuoteIdentifier quotes an identifier for the dialect. MySQL uses
    // backticks for reserved words (e.g. `key`). Other dialects return
    // the identifier unchanged (Postgres/SQLite don't reserve "key").
    QuoteIdentifier(name string) string
}
```

All 11 hardcoded sites were refactored to use these methods. The `MySQLDialect`
(now fully functional) implements them with MySQL-specific syntax.

## Consequences

**Breaking change for external `Dialect` implementors:** Any code that
implements the `Dialect` interface must now implement these 4 methods.
This is acceptable because:

1. The `Dialect` interface is internal infrastructure — consumers use
   `NewSQLiteBackend` / `NewSQLBackend` / `NewMySQLBackend` facades, not
   custom dialects.
2. Only 4 dialects exist in the codebase, all updated together.
3. The methods are trivially implementable (most return constant strings).

**Positive consequences:**
- MySQL and MariaDB are now fully supported across all SQL stores.
- Future database dialects (e.g., Oracle, SQL Server) can be added by
  implementing 4 methods instead of branching at 11 call sites.
- The `cqrs_kv` table's `key`/`value` columns now use
  `dialect.QuoteIdentifier("key")` everywhere, preventing reserved-word
  collisions in any dialect.

**Alternative considered:** A separate `UpsertDialect` sub-interface checked
at runtime via type assertion. Rejected because it would leave 11 sites with
dual code paths (old + new), increasing complexity instead of reducing it.

## MySQL-specific patterns

| Pattern | PostgreSQL/SQLite | MySQL |
|---------|-------------------|-------|
| Upsert update ref | `excluded.col` | `VALUES(col)` |
| Do-nothing | `ON CONFLICT DO NOTHING` | `ON DUPLICATE KEY UPDATE col = col` |
| Do-update | `ON CONFLICT (cols) DO UPDATE SET ...` | `ON DUPLICATE KEY UPDATE ...` |
| Reserved word `key` | `key` (not reserved) | `` `key` `` (backtick-quoted) |
| Idempotency conditional | `WHERE expires_at < excluded.expires_at` | `IF(expires_at < VALUES(expires_at), VALUES(expires_at), expires_at)` |

The last row is notable: MySQL's `ON DUPLICATE KEY UPDATE` does not support a
`WHERE` clause. The `idempotency/sqlstore` module's conditional TTL extension
uses MySQL's `IF()` function instead.
