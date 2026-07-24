# kv — Backend-Agnostic Key-Value Store and Typed Stores

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/kv/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/kv/v4)

Minimal interface for embedded key-value stores with ordered iteration and atomic batch writes. Plus typed stores, caches, and view stores built on top.

```bash
go get github.com/larsartmann/go-cqrs-lite/kv/v4
```

## Why?

No existing Go KV meta-API (gokv, valkeyrie) provides all three operations an event store needs: iteration, batch, and byte-slice keys. This package defines the interface and provides typed, generic layers on top.

## Layer-0 Interfaces

| Interface           | Methods                                  | Purpose                                |
| ------------------- | ---------------------------------------- | -------------------------------------- |
| `Store`             | `Reader` + `Writer` + `Closer`           | Full read-write access                 |
| `Reader`            | `Get`, `Has`, `NewIterator`              | Read-only access                       |
| `Writer`            | `Set`, `Delete`, `Batch`                 | Write access                           |
| `ConditionalWriter` | `SetIfAbsent`                            | Atomic conditional write (idempotency) |
| `Iterator`          | `Next`, `Key`, `Value`, `Error`, `Close` | Ordered key-value iteration (snapshot) |
| `Batch`             | `Set`, `Delete`, `Commit`, `Close`       | Atomic multi-key writes                |

## Quick Start

### Raw KV

```go
s := kv.NewMemStore()
defer s.Close()

s.Set([]byte("user:1"), []byte("alice"))
val, _ := s.Get([]byte("user:1"))

// Atomic batch
batch, _ := s.Batch()
batch.Set([]byte("a"), []byte("1"))
batch.Delete([]byte("old"))
batch.Commit()

// Prefix iteration (lexicographic order)
iter, _ := s.NewIterator([]byte("user:"))
defer iter.Close()
for iter.Next() {
    fmt.Printf("%s = %s\n", iter.Key(), iter.Value())
}
```

### TypedStore[V, K]

Type-safe typed store with automatic codec serialization:

```go
store := kv.NewTypedStore[UserView, UserID](kvBackend, kv.WithTypedCodec(codec.JSONCodec{}))
store.SetTyped(ctx, userID, UserView{Name: "Alice"})
user, _ := store.GetTyped(ctx, userID)
```

### Cache[V, K]

LRU-bounded cache wrapping a TypedStore (ADR-0032):

```go
cache, _ := kv.NewCache[UserView, UserID](store, kv.WithCacheCapacity(500))
// Cache transparently fronts the store — hit returns cached, miss delegates
```

### ViewStore[V, K] Interface

Read-model store interface with queryable columns (for SQL-backed views):

```go
type ViewStore[V any, K fmt.Stringer] interface {
    Get(ctx, key) (*V, error)
    Set(ctx, key, *V) error
    Delete(ctx, key) error
    // Optional: Query, Count, BatchSet, DeleteAll (checked at runtime)
}
```

## Design

- **Defensive cloning**: `MemStore.Get` and `Set` defensively clone byte slices. Callers can't mutate internal state.
- **Point-in-time snapshots**: `NewIterator` returns a consistent snapshot view, safe for concurrent writes.
- **ConditionalWriter**: Enables atomic check-then-set for idempotency stores.
- **TypedStore codec default**: CBOR (override with `kv.WithTypedCodec(c)`).

## Implementations

| Implementation      | Module                                            |
| ------------------- | ------------------------------------------------- |
| `MemStore`          | This package (reference in-memory implementation) |
| Pebble KV adapter   | [storage/pebble](../storage/pebble/README.md)     |
| `SQLKVStore`        | [storage](../storage/README.md)                   |
| `SQLViewStore[V,K]` | [storage](../storage/README.md)                   |

## Related Modules

- [**storage/pebble**](../storage/pebble/README.md) — Implements `kv.Store` via `NewKVStore`
- [**storage**](../storage/README.md) — `SQLKVStore` and `SQLViewStore[V,K]`
- [**stack**](../stack/README.md) — `Materialize[V,K]` consumes `ViewStore[V,K]`
- [**idempotency/kvstore**](../idempotency/kvstore/README.md) — KV-backed idempotency store
