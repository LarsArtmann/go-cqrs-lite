# irohengine/quic

Real QUIC transport for the Irohengine CRDT replication layer.

> **Requires CGo** and the Rust toolchain (via `iroh-go`). Linux-only.
> For in-process testing without CGo, use `irohengine.InProcessNetwork`.

## What This Is

This module provides `QuicTransport`, a real QUIC networking implementation of the
`irohengine.Transport` interface. It uses [iroh-go](https://git.coopcloud.tech/decentral1se/iroh-go)
(CGo bindings to Iroh's stable 1.0 QUIC stack) to establish real P2P connections
between separate OS processes.

Every operation goes through real network I/O:

- Serialization (JSON) over real QUIC BiStreams
- Real UDP packets between processes
- Real RTT measurement from QUIC's ACK timing (`conn.Rtt()`)
- Real NAT traversal (relay servers, STUN, etc.)

## Quick Start

### 2-Node Demo (separate processes)

Terminal 1 (coordinator):

```bash
cd metaengine/irohengine/quic
CGO_ENABLED=1 go run -tags "goexperiment.jsonv2" ./demo/ -mode coordinator -wait-nodes 1 -writes 5
```

Terminal 2 (node, after copying the ticket from terminal 1):

```bash
CGO_ENABLED=1 go run -tags "goexperiment.jsonv2" ./demo/ -mode node -ticket <ticket-from-terminal-1> -writes 5
```

Both processes will show all 10 keys (5 from each side) with real QUIC measurements.

### Programmatic Usage

```go
import (
    "github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/quic"
    "github.com/larsartmann/go-cqrs-lite/metaengine/irohengine"
    "github.com/larsartmann/go-cqrs-lite/metaengine"
)

// Node A: bind, create engine, print ticket
transportA, _ := quic.New(quic.WithLocalOnly())
defer transportA.Close()

engineA := irohengine.Replicated(
    metaengine.NewMemoryEngine(),
    irohengine.WithAuthor("node-a"),
    irohengine.WithTransport(transportA),
)

ticket, _ := transportA.Ticket()
fmt.Println("Ticket:", ticket)

// Node B: bind, create engine, connect to A
transportB, _ := quic.New(quic.WithLocalOnly())
defer transportB.Close()

engineB := irohengine.Replicated(
    metaengine.NewMemoryEngine(),
    irohengine.WithAuthor("node-b"),
    irohengine.WithTransport(transportB),
)

transportB.Connect(ticket)

// Write on A, read on B (over real QUIC)
engineA.(metaengine.MapBackend).MapSet(ctx, "users", "u1", "Alice")
// B sees "Alice" after async QUIC delivery
```

## How to Verify This Is Real Networking

1. **Check UDP ports**: `ss -ulnp | grep <process>` shows the QUIC UDP socket
2. **Kill one process**: The other keeps running, proving separate processes
3. **Use `tc netem`**: Apply kernel-level delay and watch latency increase
4. **Check `conn.Rtt()`**: Values are measured from QUIC ACK timing, not `time.Sleep`

## Architecture

```
Engine A (Go)              Engine B (Go)
    |                           |
    v                           v
QuicTransport A            QuicTransport B
    |                           |
    v                           v
Iroh Endpoint A            Iroh Endpoint B
    |                           |
    +------- QUIC/UDP --------+
              |
         Real Network
```

WriteOps are serialized as JSON and sent over QUIC BiStreams. Each op opens a
new bidirectional stream, sends the data, and reads an empty ack back. The
receiver deserializes and dispatches to all registered subscribers.

## CRDT Safety

Only CRDT-safe operations replicate over QUIC:

- MapSet (LWW-Map: latest timestamp wins)
- MapDelete (LWW tombstone)
- SetAdd (OR-Set: add-only)
- CounterIncrement (PN-Counter: per-author increments)
- MultiAdd (OR-Set per key)
- LogAppend (per-author append-only)

Non-CRDT operations (MapUpdate, Scan, Graph, Vector, Search, Spatial) execute
locally and do NOT replicate. This matches the CALM theorem constraint.

## Options

| Option               | Description                                                |
| -------------------- | ---------------------------------------------------------- |
| `WithLocalOnly()`    | Localhost-only (no relay, 127.0.0.1 bind). Best for tests. |
| `WithRelay()`        | Star-topology relay mode (forward ops to all peers).       |
| `WithALPN(bytes)`    | Custom ALPN protocol. All nodes must match.                |
| `WithBindAddr(addr)` | Override bind address.                                     |
