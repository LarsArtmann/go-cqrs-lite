# Duplicate Code Review — 2026-06-11

**Source:** `branching-flow dupe . --format markdown`
**Result:** 15 groups, 5 actionable, 10 false positives.

## Pareto-Sorted Action Plan

| Rank | Group | Impact | Effort | Customer Value | Decision | Rationale |
|------|-------|--------|--------|----------------|----------|-----------|
| 1    | 13 (storage: `AggregateProjection` / `SQLAggregateReader`) | HIGH | LOW (~12 min) | HIGH | **EXTRACT** | Two types must stay in sync — table name + prefix validation. Drift risk = broken write/read contract. |
| 2    | 14 (projection: `Builder` / `builtProjection`) | LOW | HIGH | LOW | **ACCEPT** | Builder pattern: builder is mutable config, builtProjection is immutable `event.Projection`. Different lifecycles, different invariants. |
| 3    | 2 (example: `CreateUserPayload` etc.) | NONE | N/A | NONE | **ACCEPT** | Example apps intentionally use distinct payload shapes to demo variation. |
| 4    | 9 (example: `ItemAdded` / `ItemRemoved`) | NONE | N/A | NONE | **ACCEPT** | Two semantically opposite event types in projection demo. |
| 5    | 12 (example: `CreateUserCmd` / `RebirthUserCmd`) | NONE | N/A | NONE | **ACCEPT** | Different commands for different use cases. |

## Refactor Plan: Group 13

**Problem:** `storage/aggregate_projection.go` and `storage/sql_aggregate_reader.go` both
compute `tablePrefix + "listing_aggregates"` and both call `validateListingTablePrefix`.
If the table name or validation rule changes, both must update in lockstep.

**Solution:** Introduce a tiny `listingTable` helper in a new file `storage/listing_table.go`:

```go
type listingTable struct {
    name string
}

func newListingTable(prefix string) (listingTable, error) {
    if err := validateListingTablePrefix(prefix); err != nil {
        return listingTable{}, err
    }
    return listingTable{name: prefix + "listing_aggregates"}, nil
}
```

Both `AggregateProjection` and `SQLAggregateReader` store a `listingTable` instead of
`(db, dialect, tableName)` plus a separate validation call. This makes drift impossible
because the validation and name derivation live in one constructor.

**Files touched:**
- NEW: `storage/listing_table.go`
- MODIFY: `storage/aggregate_projection.go` (remove prefix validation, use listingTable)
- MODIFY: `storage/sql_aggregate_reader.go` (same)

**Validation:**
- `cd storage && GOWORK=off go test ./... -count=1`
- `nix run .#build`
- `nix run .#lint`
- Re-run `branching-flow dupe . --format markdown` to confirm Group 13 gone.

## Accepted Duplications — One-Line Rationale

| Group | Members | Reason for acceptance |
|-------|---------|------------------------|
| 1     | Empty marker/brand types (`AggregateMarker`, `JSONCodec`, `*Marker`, etc.) | Distinct phantom types used for type branding. Shape similarity (all zero-size) is a Go language feature, not duplication. |
| 3     | `InventoryReleased`, `InventoryReserved`, `OrderConfirmed`, `PaymentCharged`, `PaymentRefunded` | Distinct domain events with intentionally similar scalar payloads — different aggregate contexts. |
| 4     | `ChangeStatusHandler`, `CreateTodoHandler`, `DeleteTodoHandler`, `UpdateTodoHandler` | Per-action handler types in a todo example. Local to one demo. |
| 5     | `ChangeUserNamePayload`, `Tag`, `UserNameChangedPayload` | Different payload types in different examples / different fields when inspected carefully. |
| 6     | `CountTodosHandler`, `GetTodoHandler`, `ListTodosHandler` | Per-query handler types in todo example. Local to one demo. |
| 7     | `SQLCommandStore`, `SQLEventStore` | Different stores with intentionally different fields (commands have different schema than events). |
| 8     | `SQLCheckpointStore`, `SQLSnapshotStore` | Different stores; different schemas. |
| 10    | `Ref`, `SchemaRef` | Different references in different packages (catalog vs catalog/schema). |
| 11    | `aes256gcm`, `xchacha20` | Different algorithm parameter types in encryption module — different fields, different roles. |
| 14    | `Builder` / `builtProjection` (projection/builder.go) | Builder pattern: builder is mutable config that grows; builtProjection is the immutable `event.Projection` returned by `Build()`. Different lifecycles, different contracts. Sharing fields is required. |
| 15    | `Dispatcher` (multiple modules) | Different dispatcher interfaces per module type (command, query, event). Coincidental name, distinct contracts. |
