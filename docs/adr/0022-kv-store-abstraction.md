# ADR-0022: KV Store Abstraction Module

**Status:** Accepted
**Date:** 2026-06-16

## Context

The `pebble/` module depends directly on `cockroachdb/pebble` for all KV operations. Research (`docs/research/kv-store-abstraction-research.md`) surveyed every Go KV meta-API (gokv, valkeyrie, libkv, hyddenio/kv) and found none that provides all three capabilities an event store needs:

1. **Ordered iteration** (prefix scan for loading aggregate events)
2. **Atomic batch writes** (multi-event Save)
3. **Byte-slice keys** (lexicographic ordering for version-encoded keys)

The recommendation was to define our own minimal interface. This was initially descoped (2026-06-15) because Pebble's native API was sufficient, then subsequently implemented (2026-06-16) as a standalone interface module.

## Decision

**Create a `kv/` module** with 5 interfaces (14 methods total) that captures the common denominator across Pebble, BadgerDB, and bbolt.

### Interface design

```
Store = Reader + Writer + io.Closer
Reader: Get, Has, NewIterator
Writer: Set, Delete, Batch
Iterator: Next, Key, Value, Error, Close
Batch: Set, Delete, Commit, Close
```

### Key properties

- **Byte-slice keys** with lexicographic ordering — no string keys, no marshalling opinion
- **Interface segregation** — callers can accept `Reader` or `Writer` individually
- **Snapshot iterators** — `NewIterator` returns a point-in-time view
- **Atomic batches** — all operations in a batch commit or none do
- **Defensive cloning** — `Get` returns cloned bytes, `Set` clones on intake
- **Layer 0 leaf module** — zero internal deps, 1 external dep (go-error-family)

## Rationale

1. **No standard exists** — there is no widely-adopted Go KV interface with iteration + batch
2. **Our needs are narrow** — 14 methods, not 30
3. **Future-proofing** — enables BadgerDB/bbolt adapters without touching event store logic
4. **Independently importable** — matches multi-module pattern
5. **Controls our semantics** — byte-slice keys, snapshot iteration, no auto-marshalling

## Consequences

- `pebble/` still depends directly on `*pebble.DB` — a future refactor could add a `pebble/adapter.go` implementing `kv.Store`
- The `kv/` module has no consumers yet — it's infrastructure for future backend interchangeability
- Adding a second KV backend (BadgerDB, bbolt) requires only a ~80-line adapter file

## References

- [KV Store Abstraction Research](../research/kv-store-abstraction-research.md)
- [ADR-0009: Pebble Module Scope](0009-pebble-scope-event-store-only.md)
- [KV Module Agent Plan](../planning/2026-06-15_07-33_KV_MODULE_AGENT_PLAN.md)
