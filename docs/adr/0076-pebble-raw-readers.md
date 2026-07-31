# ADR-0076: Pebble raw value readers (single-pass JSON decode)

|             |                                                                                                  |
| ----------- | ------------------------------------------------------------------------------------------------ |
| **Status**  | Accepted                                                                                         |
| **Date**    | 2026-07-31                                                                                       |
| **Context** | The Pebble LayoutPlanner's filter/sort index path decoded JSON 3x per row (filter, sort, cursor) |

## Context

The Pebble engine's `ScanRawValues` has three scan paths:

1. **Sort index path** — ordered iteration via the `o\x00` prefix index
2. **Filter index path** — range lookup via the `i\x00` prefix index
3. **Full scan path** — iterate all keys in the collection

Paths 1 and 2 return raw JSON bytes (`[][]byte`) that must be decoded for
sorting, cursor comparison, and limit application. The original implementation
decoded the same JSON blob up to **three times** per row:

1. `decodeJSON(raw)` in `sortIndexedResults` for the sort comparator
2. `decodeJSON(raw)` in `paginateIndexedResults` for the cursor comparator
3. `decodeJSON(raw)` in the caller's fold/apply function (via `ExecuteTyped`)

For 10K-item collections, this is 30K JSON decodes where 10K would suffice.

## Decision

Introduce two **optional interfaces** on the engine that allow callers to
request raw or pre-decoded values, avoiding redundant JSON parsing:

```go
// RawValueReader returns stored values as raw bytes (no decode).
type RawValueReader interface {
    RawValue(ctx context.Context, col, key string) ([]byte, bool, error)
}

// RawScanReader returns raw JSON bytes for scan operations.
type RawScanReader interface {
    ScanRawValues(ctx context.Context, col string, filters []FilterSpec,
        sortSpec *SortSpec, cursor any, limit int) ([][]byte, error)
}
```

The `Store.ExecuteTyped` and `TypedReader` callers check for these interfaces
at runtime. When available, they skip the map→struct reify step and decode
directly from raw bytes into the target type (single-pass decode).

The filter index path was also fixed to **reuse decoded values** between sort
and cursor pagination (ADR-0073 follow-up), reducing per-row decodes from 3 to 1.

## Consequences

- **Performance**: Single-pass decode eliminates 2/3 of JSON parsing on the
  indexed paths. For 10K-item scans with filter+sort, this is a measurable
  throughput improvement.
- **Interface segregation**: The raw reader interfaces are **optional** —
  engines that don't implement them still work via the default map→struct path.
  This preserves backward compatibility.
- **Cursor correctness**: The `paginateIndexedResults` function (added in the
  same change) ensures cursor pagination works on the filter index path,
  fixing a bug where every page returned the same first N items.
- **LayoutPlanner coupling**: The raw readers are most valuable when combined
  with a LayoutPlan (ADR-0073), which creates secondary indexes that avoid
  full-collection scans. Without a plan, the full scan path already decoded
  each value once.
