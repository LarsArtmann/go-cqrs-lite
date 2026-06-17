# ADR-0023: Pebble KV Store Adapter

**Date:** 2026-06-17  
**Status:** Accepted  
**Deciders:** Lars Artmann, Crush

## Context

The `kv/` module (introduced in ADR-0022) defines a generic key-value store interface (`kv.Store`) with `Reader`, `Writer`, `Iterator`, and `Batch` sub-interfaces. Until this ADR, `kv/` had zero consumers — it was a ghost system with an unvalidated public API.

The `pebble/` module wraps CockroachDB Pebble (`*pebble.DB`) for CQRS event sourcing. Pebble already provides a rich LSM-tree KV API, but the CQRS stores (`EventStore`, `SnapshotStore`, `CheckpointStore`) use the raw Pebble API directly.

## Decision

Implement `pebble.KVAdapter` — a thin adapter that maps `*pebble.DB` to `kv.Store`. This makes `pebble/` the first real consumer of the `kv/` abstraction.

### Design choices

1. **Ownership semantics:** The adapter owns the `*pebble.DB` by default (`Close()` calls `db.Close()`). The `WithBorrowedDB()` option makes `Close()` a no-op for shared-DB scenarios (e.g., `PebbleBackend` where one DB backs multiple stores).

2. **Iterator positioning:** Pebble's `NewIter` returns an unpositioned iterator. The adapter's `Next()` calls `First()` on the first invocation, then `Next()` thereafter, matching the `kv.Iterator` contract.

3. **Prefix bounds:** `NewIterator(prefix)` computes the exclusive upper bound by incrementing the last byte of the prefix (or dropping overflow bytes), matching the existing pattern in `pebble/iteration.go`.

4. **Value safety:** `Get()` and iterator `Key()`/`Value()` return `slices.Clone()` of the underlying bytes, as Pebble's returned slices are only valid until the next call.

5. **Error mapping:** `pebble.ErrNotFound` → `kv.ErrNotFound`. All other Pebble errors are wrapped with `fmt.Errorf("...: %w", err)`.

6. **Batch:** `pebbleBatch` wraps `*pebble.Batch` with atomic `Commit()` semantics. `Close()` discards uncommitted operations. `Close()` after `Commit()` is a no-op.

## Consequences

### Positive

- **kv/ module validated:** The interface is proven fit for purpose against a real storage engine.
- **Ghost system eliminated:** `kv/` now has a production-grade consumer.
- **Future storage backends:** Any module that accepts `kv.Store` can use Pebble directly.
- **Contract testing:** The kv/ test suite can be run against the pebble adapter to verify conformance.

### Negative

- **+1 direct dependency** for pebble (7 → 8). Budget was consciously raised.
- **Two iterator APIs:** Pebble's native API and kv.Iterator coexist. The adapter does NOT replace existing pebble store internals — they continue to use `*pebble.DB` directly for performance-critical paths.

### Neutral

- The existing `EventStore`, `SnapshotStore`, and `CheckpointStore` do NOT use `kv.Store` internally. This is by design — those stores have specialized needs (CBOR serialization, journaling, OTel spans) that would be awkward to express through the generic interface. The adapter is for consumers who want a simple KV API over Pebble.

## Alternatives Considered

1. **Delete kv/ entirely** — Rejected. The abstraction is useful for testing, contract verification, and future storage backends.

2. **Refactor pebble stores to use kv.Store internally** — Rejected. Would add an abstraction layer to performance-critical paths for no consumer benefit. The adapter is additive.

3. **Create a separate `kvpebble/` module** — Rejected. The adapter is small enough to live in `pebble/` alongside the existing stores.
