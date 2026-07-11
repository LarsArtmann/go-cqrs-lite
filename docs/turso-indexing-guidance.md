# Turso Indexing & Health Check Guidance

This document provides platform-specific guidance for using the `turso/indexing`
module with different database backends, and shows how to integrate Turso health
checks with the `middleware` health check system.

---

## Platform-Specific Indexing Guidance

### Turso (Embedded Database with Sync)

Turso uses an embedded database with optional cloud sync. Indexing behavior:

- **CREATE INDEX** is fully supported and recommended for CQRS access patterns.
- Use `indexing.ApplyOptimizations(ctx, db)` to set Turso-specific pragmas
  (journal mode, synchronous, cache size).
- Indexes are created locally and synced to the cloud replica automatically.
- **Tip**: Run `advisor.AnalyzeQuery` during development to catch missing
  indexes before production. The advisor inspects `EXPLAIN QUERY PLAN` output.

```go
import "github.com/larsartmann/go-cqrs-lite/turso/v4/indexing"

auto := indexing.NewAutoIndexer(db,
    indexing.WithIndexingHooks(
        indexing.WithAfterCreateHook(func(_ context.Context, hctx indexing.HookContext) error {
            log.Printf("index created: %s", hctx.Index.DDL())
            return nil
        }),
    ),
)
auto.Enable()
_ = auto.ApplyRecommended(ctx)
```

### SQLite (Local Standalone)

Standard SQLite behaves the same as Turso for indexing. Key differences:

- No cloud sync — indexes live only in the local file.
- `PRAGMA journal_mode = WAL` is the single most impactful optimization.
- For write-heavy event stores, consider `PRAGMA synchronous = NORMAL` (not FULL).
- SQLite supports partial indexes: `CREATE INDEX ... WHERE aggregate_type = 'User'`.

### PostgreSQL (via storage/ SQLEventStore)

PostgreSQL has a different query planner than SQLite/Turso. Key considerations:

- PostgreSQL's planner is cost-based and collects statistics automatically.
  Run `ANALYZE` after bulk imports so the planner has fresh stats.
- PostgreSQL does NOT support `PRAGMA` statements — do not call
  `indexing.ApplyOptimizations` on a PG connection.
- PostgreSQL supports expression indexes and partial indexes natively.
- The `indexing.Advisor` targets SQLite's `EXPLAIN QUERY PLAN` format.
  For PostgreSQL, use `EXPLAIN (ANALYZE, BUFFERS)` and tools like `pg_stat_statements`.
- **Recommended indexes for CQRS on PostgreSQL**:

```sql
-- Aggregate load (most common query)
CREATE INDEX CONCURRENTLY idx_events_agg
    ON events (aggregate_type, aggregate_id, version);

-- Journal replay (position-based)
CREATE INDEX CONCURRENTLY idx_events_journal
    ON events (event_id);

-- Projection cursor lookup
CREATE INDEX CONCURRENTLY idx_events_cursor
    ON events (occurred_at, event_id);
```

- Use `CONCURRENTLY` to avoid blocking writes during index creation.
- Consider partitioning the events table by time range for high-volume systems.

---

## Health Check Integration

The `middleware` package provides a health check HTTP handler. To monitor a
Turso database connection opened via `turso.Open`:

```go
import (
    "context"
    "database/sql"
    "fmt"
    "net/http"

    "github.com/larsartmann/go-cqrs-lite/middleware"
    "github.com/larsartmann/go-cqrs-lite/turso/v4"
)

func tursoHealthChecker(db *sql.DB) middleware.HealthChecker {
    return func(ctx context.Context) middleware.Check {
        if err := db.PingContext(ctx); err != nil {
            return middleware.Check{
                Status: middleware.HealthStatusFail,
                Output: fmt.Sprintf("turso ping failed: %v", err),
            }
        }
        return middleware.Check{
            Status: middleware.HealthStatusPass,
        }
    }
}

// Usage:
db, _ := turso.Open(turso.DbPath("local.db"))
turso.ConfigurePool(db)

handler := middleware.HealthCheckHandler("1.0.0", tursoHealthChecker(db))
http.Handle("/health", handler)
```

### Health Check for SyncDB

For a sync-enabled database (`turso.OpenSync`), use the built-in
`SyncDB.HealthCheck` method directly:

```go
syncDB, _ := turso.OpenSync(ctx,
    turso.DbPath("local.db"),
    turso.RemoteURL("https://my-db.turso.io"),
    turso.AuthToken("token"),
)
defer syncDB.Close()

func tursoSyncHealthChecker(syncDB *turso.SyncDB) middleware.HealthChecker {
    return func(ctx context.Context) middleware.Check {
        if err := syncDB.HealthCheck(ctx); err != nil {
            return middleware.Check{
                Status: middleware.HealthStatusFail,
                Output: fmt.Sprintf("turso sync ping failed: %v", err),
            }
        }
        return middleware.Check{
            Status: middleware.HealthStatusPass,
        }
    }
}

handler := middleware.HealthCheckHandler("1.0.0", tursoSyncHealthChecker(syncDB))
http.Handle("/health", handler)
```

### Index Health

To check whether auto-indexing has run and how many indexes exist, use
`indexing.Stats` to list all indexes with their query-planner statistics:

```go
advisor := indexing.NewAdvisor(db)
_ = advisor.ExistingIndexes(ctx)

allStats, _ := indexing.Stats(ctx, db)
if len(allStats) < 3 {
    log.Warn("events table has fewer than expected indexes")
}

// Find indexes that the query planner has never used.
unused, _ := indexing.UnusedIndexes(ctx, db)
for _, idx := range unused {
    log.Printf("unused index: %s on %s", idx.Name, idx.Table)
}
```

---

## Comparison: Indexed vs Unindexed

The `indexing.Advisor` provides `AnalyzeQuery` which returns recommendations
with priority classifications. Use it to find queries that need indexes:

```go
advisor := indexing.NewAdvisor(db)

// Find missing indexes for a specific query.
recs, _ := advisor.AnalyzeQuery(ctx,
    "SELECT * FROM events WHERE aggregate_type = ? AND aggregate_id = ?",
    "User", "user-id")
for _, r := range recs {
    if r.Priority == indexing.PriorityCritical ||
        r.Priority == indexing.PriorityRecommended {
        log.Printf("MISSING INDEX (%s): %s — %s",
            r.Priority, r.Index.DDL(), r.Explanation)
    }
}

// Find missing indexes across all CQRS tables at once.
allRecs, _ := advisor.MissingIndexes(ctx)

// Apply all recommendations automatically.
auto := indexing.NewAutoIndexer(db)
auto.Enable()
_ = auto.Apply(ctx, allRecs)
```
