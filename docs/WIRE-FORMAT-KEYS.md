# Wire-Format Keys — Stream Vocabulary Status

Every on-disk / on-the-wire key that ever carried the `aggregate` vocabulary,
its current spelling, the legacy-read window, and when the fallback dies.
This is the single source of truth for the v5 sweep §4 renames
(`docs/planning/v5-deprecation-sweep.md`); the golden tests cited here pin
each row.

## Status table

| Surface                          | Format | Current keys                                                                                                   | Legacy keys accepted                                                  | Legacy written?             | Fallback dies |
| -------------------------------- | ------ | -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------- | --------------------------- | ------------- |
| `snapshot.Snapshot`              | JSON   | `stream_id`, `stream_type`                                                                                     | `aggregateId`, `aggregateType` (decode-only)                          | no                          | v6            |
| pebble snapshots                 | CBOR   | `stream_id`, `stream_type`                                                                                     | pre-rename rows decode with zeroed identity, rebuilt from the key     | no                          | v6            |
| bbolt events                     | CBOR   | `stream_id`, `stream_type`                                                                                     | `aggregate_id`, `aggregate_type` (decode-only)                        | no                          | v6            |
| bbolt commands                   | CBOR   | `stream_id`, `stream_type`                                                                                     | `aggregate_id`, `aggregate_type` (decode-only)                        | no                          | v6            |
| pebble commands                  | CBOR   | `stream_id`, `stream_type`                                                                                     | `aggregate_id`, `aggregate_type` (decode-only)                        | no                          | v6            |
| watermill event/command metadata | KV     | `stream_id`, `stream_type`                                                                                     | `aggregate_id`, `aggregate_type` (dual-read)                          | **yes** (dual-write window) | v6            |
| SQL `snapshots` columns          | SQL    | `stream_id`, `stream_type`                                                                                     | migrated by `MigrateSnapshotColumnsToStream` (auto-run at InitSchema) | n/a                         | n/a           |
| SQL `events`/`commands` columns  | SQL    | `aggregate_type`, `aggregate_id` — **still**                                                                   | see assessment below                                                  | n/a                         | TBD (v5.x)    |
| benchkit result JSON             | JSON   | `streams`, `eventsPerStream` (schema `2.0.0`)                                                                  | none — output contract, old keys gone                                 | n/a                         | n/a           |
| error-family codes               | string | stream vocabulary (`*.stream_*`, `read_stream`, …)                                                             | none — observability rename, batched 2026-09-08                       | n/a                         | n/a           |
| transport/grpc proto fields      | proto  | `aggregate_id`, `aggregate_type` — module deleted wholesale at v5 (ADR-0127)                                   | n/a                                                                   | n/a                         | v5            |
| pebble slog keys                 | log    | `stream_type`, `stream_id`                                                                                     | none                                                                  | n/a                         | n/a           |
| `listing.aggregate_projection`   | string | projection name — consumer-visible in metaengine collections; still open (rename = collection identity change) | n/a                                                                   | n/a                         | TBD           |

## Pinning tests

- `snapshot/wire_test.go` — JSON + CBOR carry the new keys; legacy spellings decode.
- `storage/bbolt/serialization_golden_test.go` — envelope keys + byte golden (`testdata/golden-event.cbor`).
- `storage/bbolt/serialization_legacy_test.go` — legacy CBOR rows decode; fresh rows never carry the old keys.
- `storage/pebble/command_serialization_legacy_test.go` — pebble legacy rows decode; re-serialized rows carry only stream keys.
- `watermill/protocol_legacy_test.go` — legacy-only metadata decodes; fresh messages dual-write.
- `watermill/golden_test.go` — metadata snapshot (both spellings during the window).
- `encryption/envelope_wire_golden_test.go` — unrelated to the aggregate rename, same golden discipline for the encryption envelope.

## SQL events/commands columns — assessment (M25.2, 2026-09-09)

The `events` and `commands` tables still carry `aggregate_type` /
`aggregate_id` columns, indexes (`idx_events_aggregate`,
`idx_events_agg_time`, …), and every SELECT/INSERT references them across
all four dialects. Unlike the binary formats above, columns cannot get a
decode-only fallback — a rename is a schema migration, and on the events
table (the hot path, the largest table in most deployments) the naive
`ALTER TABLE ... RENAME COLUMN` is either a full table copy (MySQL/MariaDB)
or a lock-taking metadata change (PostgreSQL) — unacceptable as a
construction-time side effect.

**Recommendation (pending owner ruling on 5.0 vs 5.x):** ship the rename as
an expand-contract migration in a v5.x minor, NOT in the v5.0 cut:

1. **Expand**: add `stream_type`/`stream_id` columns (nullable) + dual-write
   in the INSERT paths; a backfill `UPDATE ... SET stream_* = aggregate_*`
   batched by primary key.
2. **Switch**: reads move to the stream columns once the backfill completes
   (checked like `MigrateSnapshotColumnsToStream`, idempotent, re-runnable).
3. **Contract**: drop the aggregate columns and rebuild indexes in a later
   minor, after one full release cycle.

This keeps v5.0's migration surface equal to the already-shipped snapshots
migration and defers the big-table churn to a deliberate operator-driven
step. The wire-key table above stays the tracking surface.

## Deletion discipline

Every legacy fallback in the table carries a `v6` marker at its definition
site (`v6: drop` / "deleted at v6" comments on `snapshotWireLegacy`, the
bbolt/pebble `*StreamKeysLegacy` shadows, and the watermill
`metaLegacyAggregate*` constants). At v6: delete the fallback types, the
dual-write block in watermill, and this table's Legacy column.
