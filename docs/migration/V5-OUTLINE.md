# v4 → v5 Migration Guide — OUTLINE (pre-cut planning)

> **Status:** outline, not a migration document. Each entry becomes a full
> section (what breaks, why, mechanical steps, before/after) as the v5 cut
> approaches. Sources: ADR-0123 (storage tiers), ADR-0126 (metadata/store
> transforms), ADR-0127 (transport), ADR-0114 (tombstone), the v5 audit
> T18 (2026-08-22), and the TODO_LIST v5 section.

## 1. Module path migration `/v4` → `/v5`

Same mechanics as v3→v4 (see MIGRATION-GUIDE.md §1): go.mod require
directives + import paths, then `go mod tidy`. 80+ go.mod files today.

## 2. Deletions (ADR-0123: view + relational tiers, stack presets)

- `stack.Materialize`, `stack.Bundle`, all 8 stack presets, `stack.RunProjections`
- `storage.RelationalProjection`, `storage/view` (SQLViewStore)
- `graph.GraphProjection` (graphadapter keeps GraphDriver/GraphSink)
- `storage/sql.BuildWhereClause` (deprecated 2026-08-15)
- Replacement: `system.System` composition root + `projectionhost.Host`
  - auto-projection (ADR-0116).

## 3. Deletions (ADR-0127: deprecated transports)

- `transport/http` (SSE delivery) and `transport/grpc` (gRPC dispatch) modules
- Replacement: `watermill/` brokers, go-sse for SSE.

## 4. Deletions (ADR-0126: deprecated compat shells)

- `schema.VersionedStore`, `schema.VersionedSeekableJournal`
- `signing.Rejecting*`, `encryption.ErrInnerStoreNot*`
- `metadata.CustomData`
- Replacement: `event.DecorateStore` / `event.DecorateJournal` +
  `SinkTransform`/`SourceTransform` composition.

## 5. Deletions (ADR-0114 completion)

- Deprecated tombstone metadata API (`event.DetectTombstone`/`MarkTombstone`,
  `event.TombstoneMark` metadata triggers)
- Replacement: deletion as domain events (`listing.StatusMiddleware` bridging).

## 6. Semantic alignments (pending final decision before the cut)

- `event.EventSource` missing-stream shape: pebble/bbolt `(nil, nil)` →
  `(nil, ErrStreamNotFound)` (contract pinned on the interface godoc).
- `record.NewStreamRef` validation becomes breaking (invalid type/id rejected).
- Honest snapshot wire tags (T18 audit): rename `meta` wire keys.
- `id.ActorID` vs `record.Actor` zero-semantics unification.
- listing cursor keyed by (type, id) if cross-type ambiguity is not solved
  at the reader level.

## 7. Signal/noise decisions carried into v5

- kvstore SA1019 exclusion: kept permanent (see .golangci.yml comment,
  2026-08-29) — go-idempotency MemoryStore in test matrices is sanctioned.
- `boundedMap` FIFO cache semantics: documented as heuristic-only.

## 8. Cut order (draft)

1. Land all v5-marked removals behind the module-path bump.
2. Regenerate api-stability golden; refresh cqrs-lint goldens.
3. CHANGELOG: split every `[Unreleased]` section into the v5.0.0 block.
4. Per-module tags in dependency order (Tier 0 → 6), GOWORK=off build matrix
   between waves; see CONTRIBUTING "Pin-bump before tagging".
5. Write this outline into the full MIGRATION-GUIDE v5 document.
