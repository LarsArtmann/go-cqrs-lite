# Design Spike: Secondary Indexes for Read-Model Scan

**Status:** Proposed
**Module:** `storage/` (SQLViewStore), `kv/` (TypedStore)

## Problem

Today, `kv.TypedStore.Scan()` loads all items into memory and filters in Go. `storage.SQLViewStore.Scan()` uses a `WHERE` clause but only on the key column. Consumers with large read-model sets (10K+ items) need indexed queries on non-key fields (e.g., "find all users where status=active and created_at > X").

## Design

### Interface Extension

```go
// IndexSpec declares a secondary index on a view column.
type IndexSpec struct {
    Name   string   // index name for DDL
    Column string   // column to index
}

// ViewStoreOption to declare indexes at construction time.
func WithIndexes(specs ...IndexSpec) ViewStoreOption

// Query executes a filtered, indexed scan.
type ViewQuery struct {
    Conditions []Condition
    Limit      int
    Offset     int
}

type Condition struct {
    Column string
    Op     Operator  // Eq, Neq, Gt, Gte, Lt, Lte, In, Like
    Value  any
}
```

### SQL Implementation

`SQLViewStore` already has `ViewQuery` and `Condition` types (added by the concurrent session). The `CREATE INDEX` DDL runs during auto-migration when `WithIndexes` is specified:

```sql
CREATE INDEX IF NOT EXISTS idx_<table>_<column> ON <table>(<column>);
```

### KV Implementation

`kv.TypedStore` doesn't have SQL indexes. For KV-backed stores, secondary indexes are maintained as separate sorted sets:

```
index:<view>:<column>:<value> → SET of keys
```

This requires a write-time maintenance cost (update index on every Set/Delete) but enables O(log N) lookups instead of O(N) scans.

### Key Design Decisions

1. **Declarative, not imperative** — Indexes are declared at construction time, not created ad-hoc. This makes the schema explicit.
2. **SQL first, KV later** — The SQL path is straightforward (DDL indexes). The KV path is more complex and can be deferred.
3. **No query planner** — Conditions are applied as WHERE clauses directly. The SQL engine's own planner handles optimization.

### Effort

- SQL path: S (types already exist, just need DDL generation + auto-migrate integration)
- KV path: M (requires index maintenance logic, sorted set operations)
