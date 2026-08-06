# metaengine/irohengine — Iroh CRDT Replication Wrapper

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4)

Adds CRDT convergence to any local [metaengine](../README.md) Engine via a pluggable `Transport`. Reads always hit the local engine; CRDT-safe writes replicate to peers. Non-CRDT operations (like `MapUpdate`) stay local.

```bash
go get github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4
```

## CRDT-Safe Operations

| Operation       | CRDT Type         | Convergence guarantee                    |
| --------------- | ----------------- | ---------------------------------------- |
| `MapSet`        | LWW (last-writer-wins) | Highest timestamp wins              |
| `SetAdd`        | OR-Set (add-wins) | Union of all adds                        |
| `CounterIncrement` | PN-Counter     | Sum of all increments/decrements         |
| `MultiAdd`      | OR-Set per key    | Union of all adds per multimap key      |
| `LogAppend`     | Append-only log   | Union of all appended entries            |

Non-CRDT operations (`MapUpdate`, `MapDelete`, `SetRemove`) stay local — they
do NOT replicate. This matches the CALM theorem constraint.

## Quick Start

```go
import (
    "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
    "github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

// 1. Create a local engine
local := metaengine.NewMemoryEngine()

// 2. Create an in-process transport network (fastest, no CGo)
network := irohengine.NewInProcessNetwork()
transport := network.Join("node-a")

// 3. Wrap with replication
engine := irohengine.Replicated(local,
    irohengine.WithTransport(transport),
    irohengine.WithAuthor("node-a"),
)

// 4. Use as a normal metaengine.Engine — CRDT-safe writes replicate
engine.MapSet(ctx, "users", "alice", map[string]any{"name": "Alice"})
```

## Three-Tier Transport Testing Pyramid

| Tier | Transport              | CGo | What It Catches                                           |
| ---- | ---------------------- | --- | --------------------------------------------------------- |
| 0    | `InProcessNetwork`     | No  | CRDT merge logic, subscriber dispatch, ordering           |
| 1    | `loopback.LoopbackTransport` | No | Serialization bugs, TCP framing, connection lifecycle |
| 2    | `quic.QuicTransport`   | Yes | NAT traversal, QUIC ACK timing, connection migration      |

Tier 0 is in this module. Tiers 1 and 2 are separate submodules:

- [**irohengine/loopback**](loopback/README.md) — Real TCP, no CGo
- [**irohengine/quic**](quic/README.md) — Real QUIC via iroh-go (CGo)

## API

| Symbol                       | Description                                              |
| ---------------------------- | -------------------------------------------------------- |
| `Replicated(local, opts...)` | Wraps a local engine with CRDT replication.              |
| `NewInProcessNetwork(opts)`  | Creates a goroutine-based transport network (fastest).   |
| `WithTransport(t)`           | Sets the transport (default: InProcessNetwork).          |
| `WithAuthor(author)`         | Sets the node author ID for LWW tie-breaking.            |
| `WithNamespace(ns)`          | Isolate replication state between applications.          |

### In-Process Network Options

| Option                      | Description                                  |
| --------------------------- | -------------------------------------------- |
| `WithNetworkDelay(max)`     | Simulate random network latency.             |
| `WithNetworkDropRate(rate)` | Simulate packet loss (0.0–1.0).              |

## Design

- **Reads are local**: All reads hit the local engine directly — zero replication latency.
- **Writes are fire-and-forget**: CRDT-safe writes apply locally, then replicate
  asynchronously via the transport. Eventual convergence is guaranteed by the CRDT properties.
- **LWW timestamps**: `MapSet` resolves conflicts by timestamp. Ties broken by author ID.
- **Separate module**: Lives outside metaengine core to preserve the zero-dependency boundary.

## Related Modules

- [**metaengine**](../README.md) — Core planner and `Engine` interface
- [**irohengine/loopback**](loopback/README.md) — TCP loopback transport (no CGo)
- [**irohengine/quic**](quic/README.md) — Real QUIC transport (CGo)
