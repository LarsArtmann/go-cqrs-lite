# Design Spike: Event Archival to Cloud Storage

**Status:** Raw Idea — No Design Yet
**Module:** `storage/` (new archival interface)

## Problem

Long-lived event stores grow indefinitely. For compliance and cost management, consumers may need to archive old events to cold storage (S3, GCS, Azure Blob) while keeping recent events in hot storage.

## Approaches

### Archive Interface

```go
type Archiver interface {
    Archive(ctx context.Context, before event.Version) error
    Restore(ctx context.Context, aggregateType string, id id.AggregateID) error
}
```

### Flow

1. `Archive(before=V)`: Export all events with version < V to cloud storage as JSON/CBOR files
2. Delete archived events from the hot store (or mark as archived)
3. `Restore`: Re-import events from cloud storage for deep replay

### Key Considerations

- **Partitioning**: Archive by `(aggregateType, aggregateID, versionRange)` for parallel upload/download
- **Format**: CBOR for compactness, or Parquet for analytics queryability
- **Integrity**: SHA-256 checksums per archive file; verify on Restore
- **Index**: Maintain a lightweight index in hot storage mapping aggregate IDs to their archive locations
- **Snapshot dependency**: Can only archive events below the oldest snapshot version for each aggregate (otherwise replay breaks)

## Recommendation

**Defer.** No consumer has requested this. When needed, the Archive/Restore interface is straightforward to implement on top of any cloud SDK. The snapshot dependency check is the only non-trivial part.
