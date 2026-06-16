# KV Store Abstraction Research

> **Date:** 2026-05-29 | **Updated:** 2026-06-16
> **Status: ✅ IMPLEMENTED** — The `kv/` module now exists as a Layer 0 leaf module
> with zero production dependencies. It provides the `Store`, `Reader`, `Writer`,
> `Iterator`, and `Batch` interfaces exactly as designed in §5, plus an in-memory
> implementation (`MemStore`) for testing. Future Pebble/BadgerDB/bbolt adapters
> can implement these interfaces.
> **Question:** Is there a meta-API for key-value stores that go-cqrs-lite should adopt instead of depending directly on Pebble?
>
> **History:** Originally descoped on 2026-06-15 because Pebble's native API was
> sufficient. The `kv/` module was subsequently implemented on 2026-06-16 as a
> standalone, zero-dependency interface module — the interface design from this
> research proved correct and was implemented verbatim.

---

## TL;DR

**No existing Go KV meta-API fits our needs.** All popular abstraction libraries lack iteration, range scans, and atomic batch writes — the three operations our event store requires. The original recommendation was to **define our own minimal `kv/` module** with ~15 methods across 3 interfaces.

> **Implementation (2026-06-16):** **Implemented.** The `kv/` module now exists
> as a Layer 0 leaf module with zero production dependencies. It provides the
> `Store`, `Reader`, `Writer`, `Iterator`, and `Batch` interfaces exactly as
> designed in §5 below, plus `MemStore` — an in-memory implementation for testing.
> Future Pebble/BadgerDB/bbolt adapters can implement these interfaces to provide
> backend interchangeability.

---

## 1. What Our Event Store Needs from a KV Backend

Looking at what `pebble/` actually does with its underlying store:

| Operation             | Usage in Event Store                                           |
| --------------------- | -------------------------------------------------------------- |
| **Get(key) → value**  | Load a single snapshot, checkpoint, or outbox entry            |
| **Set(key, value)**   | Save snapshot, checkpoint, outbox entry                        |
| **Delete(key)**       | Delete snapshot, ack outbox entry                              |
| **Prefix scan**       | Load all events for an aggregate: `cqrs_event:{type}:{id}:*`   |
| **Ordered iteration** | Events must come out in version order (lexicographic key sort) |
| **Atomic batch**      | Save multiple events + outbox entry in one write               |
| **Range delete**      | (Optional) Delete all events for a decommissioned aggregate    |
| **Has/Exists**        | Check if snapshot/checkpoint exists without reading value      |
| **Close**             | Graceful shutdown                                              |

The critical operations that disqualify most meta-APIs: **prefix scan** and **atomic batch**.

---

## 2. Survey of Go KV Meta-APIs

### 2.1. gokv (philippgille/gokv) — 828★

```go
type Store interface {
    Set(k string, v any) error
    Get(k string, v any) (found bool, err error)
    Delete(k string) error
    Close() error
}
```

| Criteria              | Verdict                                                                                                       |
| --------------------- | ------------------------------------------------------------------------------------------------------------- |
| Backends              | **30+** — Redis, Consul, etcd, bbolt, BadgerDB, LevelDB, DynamoDB, S3, PostgreSQL, MongoDB, CockroachDB, etc. |
| Iteration/range scans | ❌ Not supported. `List()`/`GetAll()` planned for v1.0 but not implemented                                    |
| Atomic batch writes   | ❌ Not supported                                                                                              |
| Transactions          | ❌ Not supported                                                                                              |
| Generics              | Uses `any` but no generic type parameters                                                                     |
| Go version            | 1.20+                                                                                                         |
| Maintenance           | Active (v0.7.0, Jan 2024; commits through Jun 2025)                                                           |
| **Verdict**           | ❌ **Too simple.** No iteration = can't load events. No batch = can't save atomically.                        |

### 2.2. valkeyrie (kvtools/valkeyrie) — 307★

```go
type Store interface {
    Put(key string, value []byte, options *WriteOptions) error
    Get(key string) (*KVPair, error)
    Delete(key string) error
    Exists(key string) (bool, error)
    Watch(key string, stopCh <-chan struct{}) (<-chan *KVPair, error)
    WatchTree(directory string, stopCh <-chan struct{}) (<-chan []*KVPair, error)
    NewLock(key string, options *LockOptions) (Locker, error)
    List(directory string) ([]*KVPair, error)
    DeleteTree(directory string) error
    AtomicPut(key string, value []byte, previous *KVPair, options *WriteOptions) (bool, *KVPair, error)
    AtomicDelete(key string, previous *KVPair) (bool, error)
    Close()
}
```

| Criteria              | Verdict                                                                                                                                        |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Backends              | 6 — Consul, etcd v2/v3, ZooKeeper, Redis, BoltDB, DynamoDB                                                                                     |
| Iteration/range scans | ⚠️ `List(directory)` returns all keys under a prefix — crude but works                                                                         |
| Atomic batch writes   | ❌ `AtomicPut` is CAS on a single key, not multi-key batch                                                                                     |
| Transactions          | ❌ No multi-key transactions                                                                                                                   |
| Watch                 | ✅ `Watch`/`WatchTree` for change notification                                                                                                 |
| Locking               | ✅ Distributed locks via `NewLock`                                                                                                             |
| Maintenance           | Maintenance mode (v1.0.0, Sep 2022; chore commits through Apr 2026, no new features)                                                           |
| **Verdict**           | ❌ **Distributed coordination focus.** Wrong abstraction level for embedded KV. Heavy API, no batch writes, `List()` is not ordered iteration. |

### 2.3. libkv (docker/libkv) — 850★

Hard fork of valkeyrie's predecessor. **Unmaintained** (archived). Same interface as valkeyrie. Not viable.

### 2.4. hyddenio/kv

```go
type KV interface {
    Get(ctx context.Context, pk lexkey.PrimaryKey) (*Item, error)
    GetBatch(ctx context.Context, keys ...lexkey.PrimaryKey) ([]*Item, error)
    Insert(ctx context.Context, item *Item) error
    Put(ctx context.Context, item *Item) error
    Remove(ctx context.Context, pk lexkey.PrimaryKey) error
    RemoveBatch(ctx context.Context, keys ...lexkey.PrimaryKey) error
    RemoveRange(ctx context.Context, rangeKey lexkey.RangeKey) error
    Query(ctx context.Context, queryArgs QueryArgs, sort SortDirection) ([]*Item, error)
    Enumerate(ctx context.Context, queryArgs QueryArgs) enumerators.Enumerator[*Item]
    Batch(ctx context.Context, items []*BatchItem) error
    Close() error
}
```

| Criteria              | Verdict                                                                                                                                                                    |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Backends              | 3 — Pebble, Azure, Redis                                                                                                                                                   |
| Iteration/range scans | ✅ `Enumerate` + `Query` with range support                                                                                                                                |
| Atomic batch writes   | ✅ `Batch` method                                                                                                                                                          |
| RemoveRange           | ✅ `RemoveRange` for bulk deletion                                                                                                                                         |
| Maintenance           | Very new, small community                                                                                                                                                  |
| **Verdict**           | ⚠️ **Closest to what we need**, but too new and too few backends. Depends on `lexkey` package for key encoding. Interesting design but not battle-tested enough to bet on. |

### 2.5. chenyanchen/kv

```go
type KV[K comparable, V any] interface {
    Get(ctx context.Context, k K) (V, error)
    Set(ctx context.Context, k K, v V) error
    Del(ctx context.Context, k K) error
}
```

Generic but no iteration, no batch. Too simple.

---

## 3. Native KV Store APIs Compared

Since no meta-API fits, let's compare the native APIs to understand what a common denominator would look like:

### 3.1. Pebble (cockroachdb/pebble) — 5.9k★ (v1.1.5 used by go-cqrs-lite)

```
Reader:  Get(key) → (value, closer, error)
         NewIter(*IterOptions) → (*Iterator, error)
         Close() error

Writer:  Set(key, value, *WriteOptions)
         Delete(key, *WriteOptions)
         DeleteRange(start, end, *WriteOptions)
         Merge(key, value, *WriteOptions)
         Apply(*Batch, *WriteOptions)

Iterator: SeekGE(key) → bool
          SeekLT(key) → bool
          SeekPrefixGE(prefix) → bool
          First() → bool
          Last() → bool
          Next() → bool
          Prev() → bool
          Valid() → bool
          Key() → []byte
          Value() → []byte
          Error() → error
          Close() → error

Batch:   Set(key, value, *WriteOptions)
         Delete(key, *WriteOptions)
         Commit(*WriteOptions) → error
         Close() → error
```

### 3.2. BadgerDB (dgraph-io/badger) — 15.7k★

```
DB:      Get(key) → (ValueStruct, error)   [note: on DB, not Txn]
         NewIterator(IteratorOptions) → *Iterator
         NewTransaction(update bool) → *Txn
         NewWriteBatch() → *WriteBatch
         Close() → error

Txn:     Get(key) → (*Item, error)
         Set(key, val []byte) → error
         Delete(key []byte) → error
         Commit() → error
         Discard()

Iterator: Seek(key)
          Rewind()
          Next()
          Valid() → bool
          ValidForPrefix(prefix) → bool
          Item() → *Item
          Close() → error

WriteBatch: Set(k, v []byte) → error
            Delete(k []byte) → error
            Flush() → error
            Cancel()
```

### 3.3. bbolt (etcd-io/bbolt) — 9.6k★

```
DB:      Open(path, mode, options) → (*DB, error)
         Close() → error
         Update(fn func(*Tx) error) → error
         View(fn func(*Tx) error) → error
         Batch(fn func(*Tx) error) → error

Tx:      Bucket(name) → *Bucket
         CreateBucketIfNotExists(name) → (*Bucket, error)
         DeleteBucket(name) → error
         Commit() → error
         Rollback() → error

Bucket:  Get(key) → []byte
         Put(key, value []byte) → error
         Delete(key []byte) → error
         Cursor() → *Cursor
         ForEach(fn) → error

Cursor:  First() → (key, value []byte)
         Last() → (key, value []byte)
         Seek(seek []byte) → (key, value []byte)
         Next() → (key, value []byte)
         Prev() → (key, value []byte)
         Delete() → error
```

### 3.4. Comparison Matrix

| Feature               | Pebble                                    | BadgerDB                                       | bbolt                                       |
| --------------------- | ----------------------------------------- | ---------------------------------------------- | ------------------------------------------- |
| **Data model**        | Flat key-value                            | Flat key-value                                 | Bucket-based (nested namespaces)            |
| **Get**               | `Get(key) → ([]byte, Closer, error)`      | `Get(key) → (ValueStruct, error)`              | `bucket.Get(key) → []byte`                  |
| **Set**               | `Set(key, val, opts)`                     | `txn.Set(key, val)`                            | `bucket.Put(key, val)`                      |
| **Delete**            | `Delete(key, opts)`                       | `txn.Delete(key)`                              | `bucket.Delete(key)`                        |
| **Prefix scan**       | `SeekPrefixGE(prefix)` + `Next()`         | `ValidForPrefix(prefix)` + `Seek()` + `Next()` | `Cursor.Seek(prefix)` + manual prefix check |
| **Range scan**        | `SeekGE`/`SeekLT` + bounds in IterOptions | `IteratorOptions.Prefix` + `Seek`              | `Cursor.Seek` + manual bounds               |
| **Ordered iteration** | ✅ LSM-tree (sorted)                      | ✅ LSM-tree (sorted)                           | ✅ B+tree (sorted)                          |
| **Atomic batch**      | `Batch.Set/Delete/Commit`                 | `WriteBatch.Set/Delete/Flush`                  | `DB.Update(fn)` (whole TX)                  |
| **Transactions**      | ❌ Single write TX at a time              | ✅ MVCC concurrent TX                          | ✅ Concurrent read TX, single write TX      |
| **Delete range**      | ✅ `DeleteRange(start, end)`              | ❌ Must iterate + delete                       | ❌ Must iterate + delete                    |
| **Sync/Async writes** | ✅ `WriteOptions{Sync bool}`              | ✅ `Options.SyncWrites`                        | ⚠️ `Options.NoSync` (default: synced)       |
| **Pure Go**           | ✅                                        | ✅                                             | ✅                                          |
| **Write performance** | Very high (LSM)                           | Very high (LSM)                                | Moderate (B+tree)                           |
| **Read performance**  | High (bloom filters)                      | High (bloom filters)                           | Very high (direct B+tree lookup)            |

---

## 4. Analysis: Should We Use a Meta-API?

### Why NOT gokv

1. **No iteration** — Can't load events by prefix. This is non-negotiable for event store.
2. **No batch writes** — Can't atomically save multiple events.
3. **String keys** — Our keys are byte slices for lexicographic ordering.
4. **Auto-marshalling** — We handle our own serialization (codec/ module).

### Why NOT valkeyrie

1. **Distributed focus** — Watch, Lock, WatchTree are irrelevant for embedded KV.
2. **String keys** — Same problem as gokv.
3. **No ordered iteration** — `List()` returns unordered entries.
4. **No batch writes** — `AtomicPut` is single-key CAS.
5. **Heavy dependency** — Brings in Consul/etcd/ZooKeeper abstractions we don't need.

### Why NOT hyddenio/kv

1. **Too new** — No community, no battle-testing.
2. **lexkey dependency** — Introduces a key encoding scheme we don't control.
3. **Only 3 backends** — No meaningful ecosystem advantage.
4. **Interesting design** — `Enumerate` + `Query` + `Batch` + `RemoveRange` is close to what we need, but the project is too early to depend on.

### Why NOT define our own meta-API

Actually — we SHOULD. Here's why:

1. **No standard exists** — There is no widely-adopted Go KV interface with iteration + batch.
2. **Our needs are narrow** — Only ~15 methods across 3 small interfaces.
3. **Adapters are trivial** — Each backend adapter is <100 lines.
4. **Independently importable** — Matches our multi-module pattern.
5. **Controls our semantics** — Byte-slice keys, lexicographic ordering, no marshalling.
6. **Aligns with "data store aware but independent"** — Principle from your architecture docs.

---

## 5. Recommendation: Define Our Own `kv/` Module

### Proposed Interface

```go
package kv

// Store is the core key-value store interface.
// Keys are byte slices with lexicographic ordering.
type Store interface {
    Reader
    Writer
    io.Closer
}

// Reader provides read access to the store.
type Reader interface {
    // Get retrieves the value for the given key.
    // Returns ErrNotFound if the key does not exist.
    Get(key []byte) ([]byte, error)

    // Has checks whether a key exists without reading the value.
    Has(key []byte) (bool, error)

    // NewIterator returns an iterator over keys matching the prefix.
    // Keys are yielded in lexicographic order.
    // Caller must call Close() on the returned iterator.
    NewIterator(prefix []byte) (Iterator, error)
}

// Writer provides write access to the store.
type Writer interface {
    // Set stores the value for the given key.
    Set(key, value []byte) error

    // Delete removes the value for the given key.
    // Deleting a non-existent key is a no-op.
    Delete(key []byte) error

    // Batch returns a new batch for atomic writes.
    // All operations in the batch are committed atomically.
    Batch() (Batch, error)
}

// Iterator yields key-value pairs in lexicographic order.
type Iterator interface {
    // Next advances to the next key-value pair.
    // Returns false when exhausted or on error.
    Next() bool

    // Key returns the current key. Only valid after Next() returns true.
    Key() []byte

    // Value returns the current value. Only valid after Next() returns true.
    Value() []byte

    // Error returns any error encountered during iteration.
    Error() error

    // Close releases iterator resources.
    Close() error
}

// Batch collects write operations for atomic commit.
type Batch interface {
    // Set queues a set operation.
    Set(key, value []byte) error

    // Delete queues a delete operation.
    Delete(key []byte) error

    // Commit applies all queued operations atomically.
    Commit() error

    // Close releases batch resources. Uncommitted operations are discarded.
    Close() error
}
```

**Total: 14 methods across 5 interfaces.** Every method has a clear purpose tied to what our event store actually does.

### Adapter Complexity Estimates

| Backend   | Adapter Size | Complexity                                                                              |
| --------- | ------------ | --------------------------------------------------------------------------------------- |
| Pebble    | ~80 lines    | Trivial — nearly 1:1 mapping (Pebble's API is the design target)                        |
| BadgerDB  | ~100 lines   | Easy — prefix via `ValidForPrefix`, batch via `WriteBatch`, iteration via `NewIterator` |
| bbolt     | ~120 lines   | Easy — bucket-based, cursor for iteration, `DB.Update` for atomic batch                 |
| In-memory | ~100 lines   | Easy — `sortedmap` or `btree` from Go stdlib                                            |

### Module Structure

```
kv/
├── kv.go              # Store, Reader, Writer, Iterator, Batch interfaces + errors
├── mem.go             # In-memory implementation (for testing)
├── mem_test.go        # Tests
├── doc.go             # Package doc
├── errors.go          # ErrNotFound, ErrClosed sentinels
└── go.mod             # Zero production dependencies
```

Adapters for specific backends live in their respective modules:

```
pebble/
├── adapter.go         # kv.Store adapter around pebble.DB
├── event_store.go     # event.Store implementation using kv.Store
├── snapshot.go        # event.SnapshotStore using kv.Store
├── checkpoint.go      # event.CheckpointStore using kv.Store
├── outbox.go          # event.Outbox using kv.Store
└── go.mod             # depends on kv/ + cockroachdb/pebble

badger/                # Future module
├── adapter.go         # kv.Store adapter around badger.DB
├── event_store.go     # Same event store logic, different kv.Store
└── go.mod             # depends on kv/ + dgraph-io/badger
```

### Why This Design Beats the Alternatives

| Criterion                        | gokv        | valkeyrie      | hyddenio/kv | **Our kv/** |
| -------------------------------- | ----------- | -------------- | ----------- | ----------- |
| Iteration/range scans            | ❌          | ⚠️ (List only) | ✅          | ✅          |
| Atomic batch writes              | ❌          | ❌             | ✅          | ✅          |
| Byte-slice keys                  | ❌ (string) | ❌ (string)    | ⚠️ (lexkey) | ✅          |
| Lexicographic ordering guarantee | ❌          | ❌             | ✅          | ✅          |
| Zero external deps               | ❌          | ❌             | ❌          | ✅          |
| Event store semantics            | ❌          | ❌             | ⚠️          | ✅          |
| No marshalling opinion           | ❌ (auto)   | ✅             | ✅          | ✅          |
| Independent module               | N/A         | N/A            | N/A         | ✅          |
| Go generics where useful         | ❌          | ❌             | ⚠️          | ✅ (future) |

---

## 6. Impact on Existing Modules

### pebble/ module changes

Currently `pebble/` depends directly on `cockroachdb/pebble` for:

- `pebble.Open` — database creation
- `pebble.DB.Get/Set/Delete/NewIter/Apply/Flush/Close` — CRUD
- `pebble.Iterator.SeekGE/SeekPrefixGE/Next/Valid/Key/Value/Error/Close` — iteration
- `pebble.Batch.Set/Delete/Commit/Close` — atomic writes

With `kv/` module:

1. `pebble/adapter.go` implements `kv.Store` by wrapping `*pebble.DB` (~80 lines)
2. All event store logic (`event_store.go`, `snapshot.go`, etc.) depends only on `kv.Store`, never on `pebble.DB` directly
3. `pebble/` becomes: `adapter.go` (Pebble-specific) + `store.go` (generic, depends only on `kv.Store`)

### Future: badger/ module

Same event store logic as `pebble/store.go`, different adapter. Could share a common `kvstore/` or `eventkv/` module containing the event store implementation built on `kv.Store`.

### Future: bbolt/ module

Same pattern. bbolt adapter is slightly more complex (bucket-based) but still ~120 lines.

---

## 7. What NOT to Abstract

These belong in the specific backend module, NOT in `kv/`:

| Feature                                 | Why                                                         |
| --------------------------------------- | ----------------------------------------------------------- |
| Database opening/configuration          | Backend-specific options (cache size, WAL mode, compaction) |
| Compaction triggers                     | Pebble-specific                                             |
| Metrics/statistics                      | Backend-specific                                            |
| Checkpoints/snapshots (Pebble-specific) | Not a general KV concept                                    |
| Merge operators                         | Pebble/Badger-specific; bbolt doesn't have them             |
| Range key operations                    | Pebble-specific MVCC feature                                |
| Watch/subscribe                         | Only some backends support this                             |

---

## 8. Decision Matrix

| Option                      | Pros                                     | Cons                                                     | Verdict             |
| --------------------------- | ---------------------------------------- | -------------------------------------------------------- | ------------------- |
| **Use gokv**                | 30+ backends, actively maintained        | No iteration, no batch, string keys, auto-marshalling    | ❌ Reject           |
| **Use valkeyrie**           | Watch, Lock, AtomicPut, 6 backends       | No batch, string keys, distributed focus, heavy          | ❌ Reject           |
| **Use hyddenio/kv**         | Closest fit, modern API                  | Too new, lexkey dep, 3 backends, no community            | ❌ Reject           |
| **Define own `kv/` module** | Exact fit, zero deps, controls semantics | More work (~300 lines total)                             | ✅ **Implemented**  |
| **Keep direct Pebble dep**  | Zero work, already works                 | Locked to one backend, violates "data store independent" | Still in use        |

---

## 9. Resolution (2026-06-16): Implemented

**The `kv/` module was implemented** as a Layer 0 leaf module with zero
production dependencies. The interface design from §5 was implemented verbatim.

### What was built

The `kv/` module provides:

- **5 interfaces** (Store, Reader, Writer, Iterator, Batch) — 14 methods total
- **MemStore** — in-memory implementation with sorted iteration, atomic batches,
  and defensive cloning (safe for concurrent use)
- **Zero production dependencies** — stdlib only
- **94.4% test coverage** with 16 tests covering all operations, closed-state
  behavior, defensive cloning, and interface conformance
- **6 benchmarks** (Set, Get, Has, Delete, BatchCommit, Iterator)

### Current status

The `pebble/` module still depends directly on `*pebble.DB`. A future refactor
could add a `pebble/adapter.go` implementing `kv.Store`, then rewrite the event
store logic to depend only on `kv.Store`. This is deferred until a second KV
backend (BadgerDB, bbolt) is actually needed.

### When to revisit

If a second KV backend (BadgerDB, bbolt) is ever added, this research provides the interface design and adapter estimates needed to do it properly. The recommended interface in §5 remains valid.

### References

- [KV Module Agent Plan (Descoped)](../planning/2026-06-15_07-33_KV_MODULE_AGENT_PLAN.md)
- [ADR-0009: Pebble Module Scope](../adr/0009-pebble-scope-event-store-only.md)
- [Comprehensive Remaining Work](../planning/2026-06-16_22-30_COMPREHENSIVE_REMAINING_WORK.md)

---

_Research sources: GitHub, Sourcegraph, pkg.go.dev. Libraries surveyed: gokv (828★), valkeyrie (307★), libkv (850★, archived), hyddenio/kv, chenyanchen/kv, Pebble (5.9k★), BadgerDB (15.7k★), bbolt (9.6k★). Star counts as of 2026-06-16._
