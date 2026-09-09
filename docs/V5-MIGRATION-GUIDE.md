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

### 2a. Per-tier before/after (the three migrations that touch code)

**Composition: stack presets → system composition root (wave A)**

```go
// v4 (deprecated at v5):
bundle, _ := sqlite.New(dsn, stack.WithEventCodec(codec.CBORCodec{}))
defer bundle.Close()
bus := bundle.EventBus()

// v5 (see example/getting-started for the full runnable version):
deployment := system.DeploymentConfig{
	Engines:   map[string]system.EngineConfig{primary: {Driver: "sqlite", DSN: dsn}},
	Instances: []system.InstanceConfig{
		{Role: system.RoleSourceOfTruth, Engine: primary},
		{Role: system.RoleProjections, Engine: primary},
	},
}
domain := system.DomainConfig{ Commands: …, Projections: projection }
sys, _ := system.New(ctx, domain, deployment)
defer sys.Close()
```

**Read models: `stack.Materialize` → projectionhost (wave A)**

```go
// v4 (deprecated at v5):
mat := stack.NewMaterialize[TaskView](bundle.KV(), adapter)
go stack.RunProjections(ctx, bundle.Journal(), mat)

// v5 — inside a system deployment the host is wired for you:
go func() { _ = sys.ProjectionHost().Start(ctx) }()

// v5 — standalone:
adapter := projectionadapter.New("tasks", store, decoder)
host, _ := projectionhost.New(journal, checkpointStore)
_ = host.Register(adapter)
_ = host.Start(ctx)
```

**Transforms: ADR-0126 shells → Decorate composition (wave B)**

```go
// v4 (deprecated at v5):
store := schema.NewVersionedStore(base, upcasters)

// v5 — one store, both transform directions, capabilities preserved:
store := event.DecorateStore(base,
	encryption.EncryptSinkTransform(encrypter),   // event.SinkTransform
	schema.UpcastSourceTransform(upcasters...),   // event.SourceTransform
)
```

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
  `event.aggregate_not_found`, …) renamed to the stream vocabulary in ONE
  batch. If you alert on family codes, update dashboards at the cut; the
  6-family taxonomy itself is unchanged.
  DONE 2026-09-08: all `aggregate_*` family codes renamed (see the mapping
  table in CHANGELOG `[Unreleased]`); deprecated `ErrAggregate*` symbol
  aliases were already forwarding to the `ErrStream*` sentinels, so
  `errors.Is` matching never depended on the code strings.
- **bbolt/pebble CBOR envelopes** (events + commands) write
  `stream_id`/`stream_type` with decode-only legacy fallbacks — pre-rename
  journals stay readable; the fallbacks die at v6. DONE 2026-09-09, pinned
  by legacy-row tests + the re-blessed bbolt event golden.
- **watermill message metadata** renames `aggregate_id`/`aggregate_type` →
  `stream_id`/`stream_type` behind a DUAL-WRITE window: fresh messages carry
  BOTH spellings (pre-rename readers keep working during rolling upgrades),
  readers prefer `stream_*` and fall back. Both sides drop at v6 — plan the
  rolling upgrade so no pre-rename reader is still consuming at v6.
- **benchkit result JSON** schema `2.0.0`: workload keys
  `aggregates`/`eventsPerAggregate` → `streams`/`eventsPerStream`.
  Dashboard/automation consumers must switch at the bump (no fallback —
  output contract, not stored data).
- The complete per-surface table (formats, legacy windows, v6 deletion
  points, and the still-open SQL events/commands columns with the
  expand-contract recommendation) lives in
  [WIRE-FORMAT-KEYS.md](WIRE-FORMAT-KEYS.md).
- **encryption envelope v2 (consumer note)**: `encryption.MarshalEnvelope`
  writes the raw JSON object (`{"v":"v2","ct":…}`) — storable in JSON/JSONB
  columns on every SQL dialect. The v1 base64-wrapped form is still READ;
  only writers changed. Consumers parsing envelope strings with their own
  JSON schema must accept both shapes (`{` prefix → v2; otherwise
  base64url-decode first). Pinned by `envelope_wire_golden_test.go` and the
  v1↔v2 decode-symmetry property test.

### 3a. Operator verification snippets

Prove a database took the snapshots rename (run before first v5 boot for
peace of mind, or after to confirm):

```sql
-- PostgreSQL / MySQL / MariaDB / DuckDB:
SELECT column_name FROM information_schema.columns
WHERE table_name = 'snapshots' ORDER BY ordinal_position;
-- expect stream_type, stream_id; aggregate_* gone.

-- SQLite:
PRAGMA table_info(snapshots);
```

Live-verified 2026-09-09: MariaDB 11.4 (SKIP LOCKED dialect family) via
`TestMigrateSnapshotColumnsToStream_MariaDB` (`-tags integration`,
`MYSQL_TEST_DSN`), DuckDB via the same probe + `ALTER TABLE … RENAME
COLUMN` sequence. Concurrent InitSchema is safe: simultaneous migrations
re-probe after a lost rename race and treat a fully-migrated table as
success (`TestMigrateSnapshotColumns_ConcurrentInitIsSafe`). A crash
BETWEEN the two renames leaves a half-mixed table that the next boot
rejects loudly (`storage.snapshot_column_mixed` Corruption) instead of
silently renaming the survivor — reconcile manually once, then boot.

Watermill rolling upgrade: deploy v5 writers while v4 readers drain —
messages carry both key spellings during the window, so no consumer sees a
missing field. Verify a message's shape with `watermill.EventToMessage`
output or a broker inspection tool: both `stream_id` and `aggregate_id`
must be present until the fleet is fully on v5.

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
