# v5 Migration Guide

> Consumer-facing guide for the v4 → v5 major bump. Everything here is
> already marked `Deprecated:` in v4 — `go build` succeeds today, but
> SA1019/gopls flag the uses. Fix them now and the v5 cut is a version bump.
>
> Sources of truth: `docs/planning/v5-deprecation-sweep.md` (full symbol
> census + consumer scans), ADRs 0114/0123/0126/0127, ADR-0130 (durability
> mapping survives v5 unchanged).

## 0. Why the v5 cut exists

v4.x accumulates three layers of compatibility surface: the aggregate→stream
vocabulary rename (aliases), the stack preset era replaced by `system/` +
`projectionhost/` (ADR-0123), and the ADR-0126 transform shells that predate
`event.DecorateStore`/`DecorateJournal`. v5 deletes all three so there is ONE
way to say each thing.

## 1. Aggregate → Stream vocabulary (42 alias symbols)

Every `Aggregate*` name in `id`, `event`, `command`, `query` is an alias of
the `Stream*` name (e.g. `id.AggregateID` = `id.StreamID`,
`event.ErrAggregateNotFound` = `event.ErrStreamNotFound`). Mechanical fix:

```bash
# per repo, after bumping to v5:
gofmt -r 'id.AggregateID -> id.StreamID' -w .
# or: sed -e 's/AggregateID/StreamID/g; s/AggregateType/StreamType/g; ...'
```

There is no semantic change — the aliases ARE the new types. The lockstep
tests in v4 prove the aliases are identity (`event.Type("x") !=
record.Type("x")` does not compile).

## 2. Deletion waves at the cut (each lands as one commit family)

| Wave | Deleted                                                                                                                                                                                                                                                  | Replace with                                                                                                |
| ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| A    | `stack.Materialize`, `stack.Bundle` + all 8 presets, `stack.RunProjections`, `stack/bench`                                                                                                                                                               | `system.New` composition root + `projectionhost.Host`                                                       |
| B    | `storage/view`, `storage/relational`, `graph.GraphProjection`, `storage/sql.BuildWhereClause`, ADR-0126 shells (`schema.VersionedStore`, `schema.VersionedSeekableJournal`, `signing.Rejecting*`, `encryption.ErrInnerStoreNot*`, `metadata.CustomData`) | metaengine auto-projection; `event.DecorateStore`/`DecorateJournal` + `SinkTransform`/`SourceTransform`     |
| C    | `transport/http` (SSE), `transport/grpc`, tombstone metadata API (`event.DetectTombstone`, `MarkTombstone`, `MarkRebirth`, `TombstoneStatus`), `NewStreamRef` lenient validation, snapshot wire-tag legacy readers                                       | `watermill/` brokers or go-sse; domain-event deletion types + `listing.StatusMiddleware`; strict validation |

Consumer scans (v5-deprecation-sweep.md §6, 2026-08-30) confirmed the
deleted modules have no in-repo consumers outside themselves.

## 3. Wire formats and data

- **Snapshot JSON/CBOR tags** rename `aggregate_id`/`aggregateType` →
  `stream_id`/`stream_type` with DECODE-ONLY legacy fallback: pre-v5
  snapshots stay readable, no data migration. The fallback itself dies at v6.
  DONE 2026-09-06 (`snapshot/wire.go`): fallback covers JSON and CBOR
  (fxamacker/cbor v2.9 keys CBOR maps by the json tag when no cbor tag
  exists). Pebble's envelope tags were renamed in the same wave; old pebble
  rows keep loading because identity is rebuilt from the key.
- **SQL `snapshots` columns** (`aggregate_type`, `aggregate_id`): renamed via
  `ALTER TABLE ... RENAME` migrations shipped in `storage/migrations` in the
  same release. Apply the embedded DDL before first boot of v5.
  DONE 2026-09-06: `MigrateSnapshotColumnsToStream` runs automatically inside
  every `InitSchema` helper — existing databases upgrade on first boot, no
  manual step; data moves with the renamed columns (no backfill).
- **Error-code strings** (`event.nil_aggregate_id`,
  `storage.aggregate_not_found`, …) rename to the stream vocabulary in ONE
  batch. If you alert on family codes, update dashboards at the cut; the
  6-family taxonomy itself is unchanged.

## 4. What does NOT change

- The seven-tier module layout and every module path.
- `record/` as the structural base; `record.Type` aliases in
  event/command/query.
- Durability tiers (`metaengine.DriverConfig.Durability`) and their
  per-engine mappings (ADR-0130).
- Codec defaults (CBOR at every blind-store layer), the ADR-0044 envelope,
  and `DecodePayloadAuto`'s mixed-codec reads.

## 5. Cut checklist (release engineering)

1. `nix run .#verify` + `#vulncheck` + `#verify-ci` green; `check-depguard`,
   `check-duplication`, `check-coverage`, api-stability golden regenerated.
2. Waves A → B → C each in their own commit family, golden + changelog-symbols
   gate re-run after each; strike executed rows in
   `v5-deprecation-sweep.md` citing the commit.
3. Wire-tag renames (§3) land AFTER wave C's code deletions — never in the
   same commit as a code rename (execution rule 2).
4. Error-code batch rename last, with a CHANGELOG migration note.
5. `nix run .#verify-standalone` equivalent (`#verify-ci`) over ALL modules
   after every wave — unpublished-symbol pin traps surface here, not in
   workspace builds.
6. Tag wave per CONTRIBUTING "Pre-tag checklist"; `create-github-releases.sh`
   publishes changelog-accurate bodies.
7. Post-cut sweep: `grep -rn "Deprecated:"` over the tree must return EMPTY
   (everything deprecated in v4 is now either deleted or un-deprecated), and
   `v5-deprecation-sweep.md` must have every row struck.
