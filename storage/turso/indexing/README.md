# turso/indexing — Auto-Smart Index Management

Production-grade auto-smart indexing for Turso CQRS event-sourcing
workloads. Detects full table scans, recommends indexes, and applies
CQRS-optimized defaults in one call.

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/turso/v3"
import "github.com/larsartmann/go-cqrs-lite/turso/v3/indexing"

// One-shot: schema + indexes + performance PRAGMAs
db, _ := turso.OpenInMemory()
turso.InitSchemaWithIndexesAndOptimizations(ctx, db)

// Or step-by-step:
turso.InitSchema(ctx, db)
turso.ApplyCQRSIndexes(ctx, db)
turso.ApplyTursoOptimizations(ctx, db)
```

## Components

### `Advisor` — Analyzes query plans

```go
advisor := indexing.NewAdvisor(db,
    indexing.WithExcludedTables("audit_log", "trace"))

recs, _ := advisor.AnalyzeQuery(ctx,
    "SELECT * FROM events WHERE aggregate_type = ? AND aggregate_id = ?",
    "User", "user-id")

for _, r := range recs {
    fmt.Println(r.Priority, r.Explanation, r.Index.DDL())
}
```

### `AutoIndexer` — Creates indexes automatically

```go
auto := indexing.NewAutoIndexer(db,
    indexing.WithAutoAnalyze(),   // run ANALYZE after creating
    indexing.WithDryRun(),         // collect DDL, don't execute
)
auto.Enable()

// Option 1: Apply the predefined CQRS-optimized indexes
_ = auto.ApplyCQRSIndexes(ctx)

// Option 2: Detect and apply missing indexes
_ = auto.ApplyRecommended(ctx)

// Option 3: Combine
_ = auto.RecommendAndApply(ctx)

// Inspect dry-run results
for _, ddl := range auto.LastDDL() {
    fmt.Println(ddl)
}

// Drop indexes you no longer need
_ = auto.Drop(ctx, indexing.Index{Name: "idx_old", Table: "events", Columns: []string{"x"}})

// Lifecycle
_ = auto.Close()
```

### `ApplyOptimizations` / `ApplyOptimizationsTraced` — Performance PRAGMAs

```go
// Standard (no tracing)
indexing.ApplyOptimizations(ctx, db)

// With OTel tracing
indexing.ApplyOptimizationsTraced(ctx, db)
```

Applies: WAL, synchronous=NORMAL, 64MB cache, memory temp store.

### `Stats` / `UnusedIndexes` — Observability

```go
stats, _ := indexing.Stats(ctx, db)
for _, s := range stats {
    fmt.Printf("%s: rows=%d, hasStats=%v\n", s.Name, s.RowEst, s.HasStats)
}

unused, _ := indexing.UnusedIndexes(ctx, db)
```

### `CheckpointScheduler` — WAL maintenance

```go
sched := indexing.NewCheckpointScheduler(db, 5*time.Minute)
sched.Start(ctx)
defer sched.Stop()
```

## CQRS-Optimized Indexes

`RecommendedCQRSIndexes()` returns pre-calculated indexes for the most
common CQRS access patterns:

| Index                    | Columns                                       | Purpose                                                |
| ------------------------ | --------------------------------------------- | ------------------------------------------------------ |
| `idx_events_cursor`      | `(occurred_at, id)`                           | Cursor pagination for `ReadFrom` / journal replay      |
| `idx_events_agg_ver`     | `(aggregate_type, aggregate_id, version)`     | Covering index for `LoadFromVersion` / `LoadToVersion` |
| `idx_events_type_time`   | `(event_type, occurred_at)`                   | Projection filters by event type                       |
| `idx_commands_agg_time`  | `(aggregate_type, aggregate_id, received_at)` | Command audit trail                                    |
| `idx_commands_type_time` | `(command_type, received_at)`                 | Command analytics                                      |

## Type Model

```go
type Index struct {
    Name    string
    Table   string
    Columns []string
    Unique  bool
    Partial bool
    Where   string
    Reason  string
}

type IndexSet []Index  // DDL(), DropDDL(), Filter(table), Names()

type Recommendation struct {
    Index        Index
    Explanation  string
    QueryPattern string
    Priority     Priority
    AdvisorVer   Version
}

type Priority int  // Optional, Recommended, Critical
```

## OpenTelemetry

All major operations are traced when an OTel TracerProvider is
configured. Spans emitted:

- `indexing.advisor.analyze_query`
- `indexing.auto_indexer.apply`
- `indexing.auto_indexer.apply_recommended`
- `indexing.auto_indexer.apply_cqrs`
- `indexing.auto_indexer.drop`
- `indexing.optimizations.apply`

No-op when no provider is configured (zero overhead).

## License

Same as the parent project.

## Related Modules

- [**turso/v2**](../README.md) — Parent module; Turso connector
- [**otel/v2**](../../otel/README.md) — Index analysis and auto-indexer emit OTel spans/metrics
- [**storage/v2**](../../storage/README.md) — SQL store implementations that indexing optimizes
