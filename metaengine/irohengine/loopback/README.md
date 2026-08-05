# irohengine/loopback

TCP-based transport for the Irohengine CRDT replication layer.

> **No CGo required.** Pure Go standard library (`net` package).
> The middle tier of the transport testing pyramid.

## What This Is

This module provides `LoopbackTransport`, a real TCP networking implementation of the
`irohengine.Transport` interface. Messages are framed with a 4-byte big-endian length
prefix and serialized with CBOR (preserving `int64` types and sub-second timestamp
precision for LWW resolution).

Unlike `InProcessNetwork` (goroutine calls, no serialization) and `QuicTransport`
(real QUIC via CGo), this transport exercises:

- Real serialization round-trips (CBOR encode/decode of `WriteOp`)
- Length-prefix framing (partial reads, message boundary errors)
- Connection lifecycle (accept loop, concurrent read/write, close)
- Real goroutine scheduling effects on message ordering

**No NAT traversal, no real P2P.** Use `QuicTransport` for that.

## Transport Testing Pyramid

| Tier | Transport         | CGo | Speed  | Catches                                      |
| ---- | ----------------- | --- | ------ | -------------------------------------------- |
| 0    | InProcessNetwork  | No  | ~0.6us | CRDT merge logic, subscriber dispatch        |
| 1    | LoopbackTransport | No  | ~10us  | Serialization, framing, connection lifecycle |
| 2    | QuicTransport     | Yes | ~86us  | NAT traversal, real QUIC ACK timing          |

Each tier catches a different bug class. None subsumes another.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
    "github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
    "github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/loopback/v4"
)

func main() {
    ctx := context.Background()

    // Create two nodes.
    tA, _ := loopback.New()
    tB, _ := loopback.New()
    defer tA.Close()
    defer tB.Close()

    // Wire engines with replication.
    nodeA := irohengine.Replicated(
        metaengine.NewMemoryEngine(),
        irohengine.WithAuthor("node-a"),
        irohengine.WithTransport(tA),
    )
    nodeB := irohengine.Replicated(
        metaengine.NewMemoryEngine(),
        irohengine.WithAuthor("node-b"),
        irohengine.WithTransport(tB),
    )

    // Connect B to A.
    _ = tB.Connect(tA.Addr())

    // Write on A, read on B.
    _ = nodeA.(metaengine.MapBackend).MapSet(ctx, "users", "u1", "Alice")
    time.Sleep(100 * time.Millisecond)

    val, _, _ := nodeB.(metaengine.MapBackend).MapGet(ctx, "users", "u1")
    fmt.Println(val) // "Alice"
}
```

## CRDT-Safe Operations

| Op                 | CRDT Type     | Convergent? |
| ------------------ | ------------- | ----------- |
| `MapSet`           | LWW-Map       | Yes         |
| `MapDelete`        | LWW tombstone | Yes         |
| `SetAdd`           | OR-Set        | Yes         |
| `CounterIncrement` | PN-Counter    | Yes         |
| `MultiAdd`         | PN-Map        | Yes         |
| `LogAppend`        | Append-only   | Yes         |

Non-CRDT operations (`MapUpdate`) are local-only and not replicated.

## Encoding

Operations are serialized with [CBOR](https://github.com/fxamacker/cbor) using:

- `TimeUnixDynamic` encoding for `time.Time` (preserves sub-second precision for LWW)
- `DefaultMapType = map[string]any` (matches JSON semantics for `any` fields)

This preserves `int64` values and timestamp ordering across the wire, unlike JSON
which truncates timestamps to whole seconds and decodes numbers as `float64`.

## API

```go
// Create a transport (binds a real TCP listener).
t, err := loopback.New(loopback.WithAddr("127.0.0.1:8080"))

// Or with simulated network delay for convergence testing.
t, err := loopback.New(loopback.WithSimulatedDelay(20 * time.Millisecond))

t.Addr()     // returns the listen address (host:port)
t.Connect(addr) // dials a remote transport
t.Close()    // closes all connections and the listener

// Implements irohengine.Transport and irohengine.LatencyProvider.
```
