# ADR-0126: Metadata Canonical Generic, Store Transforms, and WAL Unification

**Date:** 2026-08-14
**Status:** Implemented
**Supersedes:** ADR-0031 Decision 3 (alias-to-struct conversion stance)
**Builds on:** ADR-0111 (Record type), ADR-0123 (v5 unification horizon)

---

## Context

Three duplications spanned the event/command/query triads:

1. **Metadata.** `command.Metadata` and `query.Metadata` aliased
   `metadata.CustomData[K]`, while `Clone`/`Merge`/`WithCustom` were
   hand-duplicated in both packages. ADR-0031 proposed standalone structs and
   rejected a generic `Metadata[T]` as over-engineered.
2. **Store wrapping.** `encryption.NewEncryptedStore` and
   `schema.NewVersionedStore` each hand-implemented full wrapper structs with
   every Store method forwarded. The encryption wrapper silently lacked
   `MultiSink`. Nil-guard middleware was duplicated in `signing` and
   `encryption`.
3. **Write-ahead-log mechanics.** The three memory stores
   (`MemoryStore`, `MemoryCommandStore`, `MemoryQueryStore`) carried three
   copies of lock/index/append/scan machinery. The SQL command/query stores
   duplicated INSERT boilerplate (placeholders, metadata marshal,
   duplicate-key wrap). The three `system` adapters duplicated backend
   fields, serialize dispatch, and value decoding.

## Decision

### 1. `metadata.Metadata[K ~string]` is the canonical type

`Metadata[K]` owns `Clone`, `Merge`, `WithCustom`, and `EnsureCustom`.
`command.Metadata` and `query.Metadata` are type aliases of
`metadata.Metadata[MetadataKey]` — fully source-compatible; their duplicated
methods are deleted. `metadata.CustomData[K]` remains only as a deprecated
generic type alias.

**Why not standalone structs (ADR-0031's proposal):** aliases to the generic
give identical type safety (the branded key parameter carries the domain)
without breaking any external composite literal, method set, or map
assignment. ADR-0031's residual risk (aliases hiding shape) is neutralized
because the aliased type now has real methods and its own home.

**`event.Metadata` does NOT embed `Metadata[K]`** — permanently for v4.
Embedding would move `Tracing`/`Custom` under the embedded field name and
break every external `event.Metadata{Tracing: ...}` composite literal. The
event-specific metadata stays a standalone struct; its `Clone`/`Merge` remain
local because its field set differs. Revisit only as part of a v5 major
break, if ever.

### 2. `event.DecorateStore` + transforms replace wrapper structs

`SinkTransform func(event.Event) (event.Event, error)` and
`SourceTransform func(event.Event) (event.Event, error)` wrap individual
events. `DecorateStore(store, sinkT, sourceT) *decoratedStore` composes them
once and implements Store + Journal + SeekableJournal + BackwardsSource +
MultiSink + io.Closer. Interface-wrapping middleware types
(`func(EventSink) EventSink`) were rejected: they cannot forward optional
capability interfaces.

- `encryption.EncryptSinkTransform` / `DecryptSourceTransform` are the
  composable forms; `NewEncryptedStore` keeps its signature and now returns
  `event.Store` via DecorateStore. Wrapped stores gain `MultiSink` (fixed
  capability gap).
- `schema.UpcastSourceTransform` is the composable upcasting form;
  `VersionedStore`/`NewVersionedStore` remain as deprecated compat shells.
- `event.RejectingPublishMiddleware` / `RejectingHandlerMiddleware` are the
  shared nil-guards; `signing.Rejecting*` are deprecated forwarders.

**Error-code migration:** unsupported-capability errors raised by wrapped
stores now carry `event.ErrInnerStoreNot*` codes instead of `encryption.*`.
Sentinel identity is preserved: `encryption.ErrInnerStoreNot*` are deprecated
aliases of the event sentinels, so existing `errors.Is` checks keep matching.

### 3. WAL write paths share one generic core per layer

- `storage/memory.LogStore[T, ID]` owns lock/index/log mechanics;
  `LogStoreConfig` injects per-store policy (duplicate and not-found errors,
  stream tracking, missing-position semantics). Event `ReadFrom` replays from
  stream start on a missing position; command/query return empty — encoded
  explicitly as a config flag rather than forked code.
- `storage/sql.Inserter[T]` is the write-side counterpart of
  `JournalReader[T]`: dialect placeholders, metadata-marshal Corruption wrap,
  duplicate-key routing to a per-entity Conflict sentinel. `InsertAll` stays
  row-by-row deliberately: command/query batches are small and per-row
  inserts keep duplicate errors naming the offending ID. Event batches keep
  the chunked multi-VALUES path (`SharedBatchInsertEvents`).
- `system.AdapterCore[T]` owns backend/collection/serialize plus the
  value dispatch (pointer | envelope string | re-marshaled JSON map) and the
  journal `ReadAll`/`ReadFromAfter` scans. EventAdapter keeps its
  version-conflict, temporal-reader, and sequence-cache logic; CommandAdapter
  and QueryAdapter become thin type layers.

## Deprecation window

Deprecated surfaces (`metadata.CustomData`, `schema.VersionedStore` /
`NewVersionedStore`, `signing.Rejecting*`, `encryption.ErrInnerStoreNot*`)
remain through v4 with `// Deprecated:` notices and are removed at v5 per the
ADR-0123 horizon. Internal code must not use them; compat tests pin their
behavior until removal.

## Consequences

- No external API breaks: aliases, shells, and forwarders keep old spellings
  source-compatible.
- One place to fix WAL bugs (lock discipline, duplicate detection, value
  dispatch) instead of three per layer.
- Wrapped stores now honestly expose their inner store's capabilities.
- Error text for unsupported capabilities on encrypted stores changes code
  prefix (`encryption.` → `event.`); `errors.Is` chains unaffected.

## Alternatives Considered

- **Interface-wrapping store middleware** — cannot forward optional
  interfaces; forces capability loss or reflection.
- **Multi-VALUES batch for command/query inserts** — loses per-item duplicate
  attribution for negligible round-trip savings at real batch sizes.
- **Embedding `Metadata[K]` in `event.Metadata`** — breaks external
  composite literals; rejected (see above).
