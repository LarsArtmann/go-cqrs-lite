# v5 Deprecation Sweep — Census Artifact

> Generated 2026-08-22 by plan task T19 (core data-model v5 execution plan).
> Single source of truth for every v4-era symbol, wire tag, and error code
> that dies or renames at the v5 cut. Feed: `grep -rn "Deprecated:"` over
> id/command/event/query/listing/snapshot/metadata/record + wire-tag and
> error-code greps (`aggregate_id|aggregate_type` over code + SQL).
> Every entry below MUST be either deleted or explicitly re-confirmed at
> the v5 cut. When an entry is executed, strike it here and cite the commit.

## 1. Aggregate-vocabulary aliases (42 symbols)

### id (12) — `id/aggregate_id.go`, `id/aggregate_type.go`

| Deprecated symbol                | Replacement                   |
| -------------------------------- | ----------------------------- |
| `AggregateMarker`                | `StreamMarker`                |
| `AggregateID`                    | `StreamID`                    |
| `NewAggregateID()`               | `NewStreamID()`               |
| `ParseAggregateID(s)`            | `ParseStreamID(s)`            |
| `ParseAggregateIDStrict(s)`      | `ParseStreamIDStrict(s)`      |
| `IsAggregateIDULID(id)`          | `IsStreamIDULID(id)`          |
| `AggregateTimestamp(id)`         | `StreamTimestamp(id)`         |
| `DeriveAggregateID(ns, keys...)` | `DeriveStreamID(ns, keys...)` |
| `AggregateType`                  | `StreamType`                  |
| `ParseAggregateType(s)`          | `ParseStreamType(s)`          |
| `AggregateRef`                   | `StreamRef`                   |
| `ErrEmptyAggregateType`          | `ErrEmptyStreamType`          |

### event (8) — `v3_compat_aliases.go`, `event.go`, `errors.go`

| Deprecated symbol                   | Replacement                              |
| ----------------------------------- | ---------------------------------------- |
| `event.AggregateType`               | `id.StreamType` (via `event.StreamType`) |
| `event.AggregateID`                 | `id.StreamID` (via `event.StreamID`)     |
| `event.AggregateRef`                | `id.StreamRef` (via `event.StreamRef`)   |
| `(*ImmutableEvent).AggregateID()`   | `StreamID()`                             |
| `(*ImmutableEvent).AggregateType()` | `StreamType()`                           |
| `ErrNilAggregateID`                 | `ErrNilStreamID`                         |
| `ErrEmptyAggregateType`             | `ErrEmptyStreamType`                     |
| `ErrAggregateNotFound`              | `ErrStreamNotFound`                      |

### command (6) — `aggregate_ref.go`, `errors.go`

| Deprecated symbol        | Replacement                   |
| ------------------------ | ----------------------------- |
| `command.AggregateType`  | `command.StreamType`          |
| `command.AggregateRef`   | `command.StreamRef`           |
| `ParseAggregateType(s)`  | `command.ParseStreamType(s)`  |
| `NewAggregateRef(t, id)` | `command.NewStreamRef(t, id)` |
| `ErrNilAggregateID`      | `ErrNilStreamID`              |
| `ErrEmptyAggregateType`  | `ErrEmptyStreamType`          |

### query (2) — `errors.go`

| Deprecated symbol       | Replacement          |
| ----------------------- | -------------------- |
| `ErrNilAggregateID`     | `ErrNilStreamID`     |
| `ErrEmptyAggregateType` | `ErrEmptyStreamType` |

### listing (5) — `aggregate_reader.go`, `types.go`, `in_memory.go`

| Deprecated symbol               | Replacement                  |
| ------------------------------- | ---------------------------- |
| `AggregateReader`               | `StreamReader`               |
| `AggregateListing`              | `StreamListing`              |
| `AggregateStatus`               | `StreamStatus`               |
| `InMemoryAggregateReader`       | `InMemoryStreamReader`       |
| `NewInMemoryAggregateReader(j)` | `NewInMemoryStreamReader(j)` |

### misc (9) — singles across modules

| Deprecated symbol                  | Module   | Replacement                                                                                                                                               |
| ---------------------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `StreamRef` alias + `NewStreamRef` | command  | use `id.StreamRef` / `id.NewStreamRef`                                                                                                                    |
| `EnsureCustom()`                   | metadata | `WithCustom` (immutability contract)                                                                                                                      |
| `CustomData[K]`                    | metadata | `Metadata[K]`                                                                                                                                             |
| `SaveSnapshot(...)`                | snapshot | `NewSnapshot` + `SnapshotSink.Save` (encoding stamp)                                                                                                      |
| `ParseType(s)`                     | event    | `record.ParseType(s, ErrEmptyEventType)`                                                                                                                  |
| `ParseType(s)`                     | command  | `record.ParseType(s, ErrEmptyCommandType)`                                                                                                                |
| `ParseType(s)`                     | query    | `record.ParseType(s, ErrEmptyQueryType)`                                                                                                                  |
| `Execute(ctx, id, type, ...)`      | decider  | `ExecuteRef(ctx, ref, ...)`                                                                                                                               |
| `Load(ctx, id, type)`              | decider  | `LoadRef(ctx, ref)`                                                                                                                                       |
| `record.StreamRef` (string)        | record   | `record.StreamKey` — same name as the id.StreamRef struct pair is a collision (plan Appendix D.2); separator convergence `/` rides the v5 migration guide |

## 2. Record-bridge Deprecated fields (5) — `record/record.go`

| Deprecated field   | Replacement | Note                     |
| ------------------ | ----------- | ------------------------ |
| `CausationID`      | `Cause`     | kind implied vs explicit |
| `ActorID`          | `Actor`     | no kind:raw parse tax    |
| `ClientCreatedAt`  | `Created`   | presence-explicit Stamp  |
| `ServerReceivedAt` | `Received`  | presence-explicit Stamp  |
| `StoredAt`         | `Stored`    | presence-explicit Stamp  |

Bridges (`event/asrecord.go`, `command/asrecord.go`, `query/asrecord.go`)
populate both forms in lockstep until the cut.

## 3. Tombstone metadata API — ADR-0114 completion

`event/tombstone.go` + `Metadata.Tombstone` + `listing` metadata-triggered
paths (see TODO_LIST §v5 "Delete deprecated tombstone metadata API"):
`DetectTombstone`, `MarkTombstone`, `MarkRebirth`, `TombstoneStatus` (+ the
`listing/in_memory.go:155` call site), `MetadataKeyTombstone`,
`MetadataKeyRebirth`. Deletion is purely event-type-driven after the cut
(`listing.StatusMiddleware(deleteTypes, rebirthTypes)` is the v4 bridge).

## 4. Wire tags + error codes carrying aggregate vocabulary

**On-disk / wire tags (rename = migration, see T18 audit in TODO_LIST §v5):**

- `snapshot.Snapshot` JSON tags `aggregateId`/`aggregateType`
- pebble `serializableSnapshot` CBOR tags `aggregate_id`/`aggregate_type`
- bbolt `command_serialization.go` CBOR tags `aggregate_id`/`aggregate_type`
  (bbolt snapshot struct already uses `stream_*`)
- SQL schema columns `aggregate_type`/`aggregate_id` (postgres/sqlite/
  mysql/duckdb `snapshots` table + any commands table variants)
- `transport/grpc/proto` fields `aggregate_id`/`aggregate_type` (module is
  deleted at v5 anyway, ADR-0127)
- `benchkit/result.go` JSON key `aggregates` (benchmark output contract)

**Stale error-code strings (family codes, not symbols — renaming them is a
consumer-visible observability change; batch at v5 with a changelog note):**

- `event.nil_aggregate_id`, `event.empty_aggregate_type`,
  `event.aggregate_not_found`
- `command.nil_aggregate_id`, `command.empty_aggregate_type`
- `memory.aggregate_not_found`
- `storage.parse_aggregate_id`, `storage.parse_aggregate_type`,
  `storage.aggregate_type_mismatch`, `storage.aggregate_id_mismatch`,
  `storage.stream_by_aggregate`, `storage.delete_by_aggregate`
- `pebble.aggregate_type_mismatch`, `pebble.aggregate_id_mismatch`
- `listing.aggregate_projection` (projection name, not an error code —
  consumer-visible in metaengine collections)

## 5. Deprecated modules deleted wholesale at v5

From TODO_LIST §v5 + ADRs 0123/0126/0127: `stack` Bundle/Materialize/
RunProjections + 8 presets, `storage/view`, `storage/relational`,
`graph.GraphProjection`, `transport/http`, `transport/grpc`,
`schema.VersionedStore`/`VersionedSeekableJournal`, `signing.Rejecting*`,
`encryption.ErrInnerStoreNot*`, `storage/sql.BuildWhereClause`.

## Execution rules

1. Delete alias families per module in one commit each; the lockstep alias
   tests (cross-type comparison form) fail the build if an alias is dropped
   while still referenced internally.
2. Wire-tag renames ONLY together with their per-backend migration (T18
   audit); never rename a tag and a code string in the same wave as a
   schema change.
3. Error-code strings rename in ONE batch with a migration note in the v5
   guide — dashboards key on them.
4. After each family: regen api-stability golden, run
   `bash scripts/check-changelog-symbols.sh`, update this artifact.
