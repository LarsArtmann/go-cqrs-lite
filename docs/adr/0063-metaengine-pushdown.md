# ADR-0063: FilterOn/SortOn Pushdown Strategy

| Field      | Value                                             |
| ---------- | ------------------------------------------------- |
| Status     | Accepted                                          |
| Date       | 2026-07-25                                        |
| Deciders   | Lars Artmann                                      |
| Related    | ADR-0061 (SQLite engine), ADR-0062 (dep boundary) |
| Supersedes | —                                                 |

## Context

The metaengine `FilterOn[R, T](func(r R) T)` and `SortOn[R, T](func(r R) T)`
APIs declare filter and sort criteria using typed closures. Currently these
closures are executed **in-memory** by the `ScanBackend.MapScan` method:
the engine loads all rows from the collection, calls each filter closure on
every row, sorts the survivors, and applies cursor+limit.

This works correctly but is O(N) for every scan regardless of the engine.
When the SQLite engine backs the scan, this means:

1. Every row is serialized to JSON, sent across the database driver boundary,
   and deserialized to `any` — even rows that will be filtered out.
2. The sort is a Go-side `slices.SortFunc`, not a database `ORDER BY` with
   an index.
3. The cursor is a Go-side offset, not a database `WHERE id > ?`.

For large collections (>10K items), the in-memory approach wastes the SQL
engine's indexing and query optimization capabilities.

## Decision

**Phase 1 (now): Keep in-memory filtering. Add an optional `PushdownCapable`
interface for future SQL engines to implement.**

**Phase 2 (future, deferred): When a production SQL engine is needed, add
`FilterSpec`/`SortSpec` as alternatives to closures, enabling SQL pushdown.**

### Phase 1 — Interface seam (zero breaking change)

Add an optional interface that SQL engines can implement:

```go
// PushdownScan is an optional capability for engines that can execute
// filter and sort operations natively (e.g., SQL WHERE + ORDER BY).
// Engines that do not implement this interface fall back to in-memory
// filtering via the ScanBackend interface.
type PushdownScan interface {
    PushdownMapScan(
        ctx context.Context,
        collection string,
        filters []FilterSpec,
        sort *SortSpec,
        cursor any,
        limit int,
    ) ([]any, error)
}
```

Where `FilterSpec` and `SortSpec` are declarative, serializable descriptions:

```go
type FilterSpec struct {
    Column string
    Op     FilterOp // Eq, Lt, Gt, Lte, Gte, Ne
    Value  any
}

type SortSpec struct {
    Column string
    Desc   bool
}
```

The planner checks for `PushdownScan` at query planning time. If present, it
emits `FilterSpec`/`SortSpec` from the `QueryConfig`'s accessor metadata.
If not, it falls back to closure-based `ScanBackend`.

### Phase 2 — Declarative alternative (when needed)

Add `FilterOnField(name, op)` and `SortOnField(name)` as alternatives to the
closure-based `FilterOn`/`SortOn`. These produce `FilterSpec`/`SortSpec`
directly, without closures, enabling SQL pushdown from any engine.

The closure-based API stays for engines that can't push down (in-memory).

## Alternatives Considered

### A. Reflection-based closure inspection (DSL)

Turn `func(r R) T { return r.Status }` into `{Column: "status", Op: Eq}` by
inspecting the closure's captured fields via `reflect`.

**Rejected.** Go closures are opaque — `reflect` cannot reliably inspect
captured variables or their field paths. This would require fragile hacks
(reading closure memory layout) that break across Go versions. The typed
closure API is for **type safety**, not for serialization.

### B. Codegen from struct tags

Generate SQL filter/sort code from struct tags like
`filter:"status" sort:"joined_at"`.

**Rejected for now.** The metaengine is designed to be runtime-composable,
not codegen-dependent. Codegen adds a build step and tight coupling between
the generator and the struct definitions. This may be revisited if the
codegen module (`cqrs-gen`) adds metaengine support.

### C. Pure in-memory (status quo, no interface seam)

**Rejected.** Without the interface seam, adding SQL pushdown later requires
either breaking the `ScanBackend` interface or duplicating the scan path.
The optional interface approach costs nothing now and preserves future options.

## Consequences

- **Positive:** Zero breaking change to existing API. Memory engines keep
  using closures. The interface seam is opt-in.
- **Positive:** `FilterSpec`/`SortSpec` are serializable — they can be logged,
  cached, and used for query planning diagnostics.
- **Negative:** Phase 2 adds a parallel API (`FilterOnField` alongside
  `FilterOn`). Consumers must choose between type-safe closures and
  SQL-pushdown-capable specs.
- **Neutral:** The `ScanBackend` interface stays as-is — it's the fallback
  for all engines. The `PushdownScan` interface is purely additive.
