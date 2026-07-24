# idempotency/kvstore — KV-Backed Idempotency Store

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4)

Adapts any `kv.Store`-compatible backend into an `idempotency.Store` with TTL-based key expiry. Provides durable, backend-agnostic idempotency for at-least-once delivery.

```bash
go get github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4
```

## Why?

The base `idempotency/` package ships an in-memory `MemoryStore` for development. For production, you need persistence: if a service restarts, already-processed command IDs must still be recognized. This subpackage bridges any `kv.Store` implementation (Pebble, SQL-backed, Redis adapter) into the `idempotency.Store` interface.

## Quick Start

```go
package main

import (
    "context"
    "time"

    "github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4"
    cqrskv "github.com/larsartmann/go-cqrs-lite/kv/v4"
)

func main() {
    // Use any kv.Store that implements KVBackend
    backend := cqrskv.NewMemStore()
    store := kvstore.New(backend)

    ctx := context.Background()

    // Check-and-record atomically
    err := store.CheckAndRecord(ctx, "cmd-123", 10*time.Minute)
    if err != nil {
        // duplicate — already processed
    }

    // Or check and record separately
    seen, _ := store.Seen(ctx, "cmd-456")
    if !seen {
        _ = store.Record(ctx, "cmd-456", 10*time.Minute)
    }
}
```

## API

| Symbol                          | Kind      | Description                                                        |
| ------------------------------- | --------- | ------------------------------------------------------------------ |
| `KVBackend`                     | Interface | `kv.Reader` + `kv.Writer` + `kv.ConditionalWriter` + `io.Closer`.  |
| `Store`                         | Struct    | Wraps a `KVBackend` as an `idempotency.Store`.                     |
| `New(backend)`                  | Func      | Constructor.                                                       |
| `Seen(ctx, key)`                | Method    | `(bool, error)` — lazy-deletes expired entries.                    |
| `Record(ctx, key, ttl)`         | Method    | Stores an expiry timestamp unconditionally.                        |
| `CheckAndRecord(ctx, key, ttl)` | Method    | Atomic check-then-set. Returns `idempotency.ErrDuplicate` if seen. |
| `Close()`                       | Method    | Passes through to the backend.                                     |

## Design

- **Expiry as value**: The TTL is stored as a Unix-nano timestamp encoded as a string. Expired entries are lazily reclaimed on read.
- **Atomic `CheckAndRecord`**: Uses `SetIfAbsent` (conditional write) for atomicity. On a rare race (key deleted between `SetIfAbsent` and `Get`), it retries once.
- **Expired key reclaim**: If `SetIfAbsent` fails but the existing entry has expired, the store overwrites and claims it.
- **Error classification**: Infrastructure errors are classified as `Transient`; decode failures as `Corruption` via `go-error-family`.
- **Returns `idempotency.ErrDuplicate`** from the parent `idempotency/v4` package for duplicates.

## Related Modules

- [**idempotency**](../README.md) — Core `Store` interface, `MemoryStore`, `ErrDuplicate`
- [**kv**](../../kv/README.md) — The `Store`, `MemStore`, `ConditionalWriter` interfaces
- [**middleware**](../../middleware/README.md) — `CommandIdempotency`, `EventIdempotency`, `QueryIdempotency` middleware that consume idempotency stores
