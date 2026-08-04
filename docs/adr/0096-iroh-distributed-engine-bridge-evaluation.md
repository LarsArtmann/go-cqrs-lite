# ADR-0096: Iroh Distributed Engine Bridge Evaluation

Date: 2026-08-03

## Status

**Real QUIC Transport Available** — Level 2 replication wrapper implemented (`metaengine/irohengine/`) with two transports:

- `InProcessNetwork`: goroutine-based delivery for unit tests (no CGo)
- `quic.QuicTransport`: real QUIC streams via `iroh-go` CGo bindings (Linux, CGo required)

Real bidirectional CRDT convergence verified between separate OS processes over UDP/QUIC.

## Context

The metaengine's replication model (ADR-0093) established DDIA-canonical
`EngineProfile` fields for `Replication`, `ReplicationLag`, and `NetworkRTT`.
All current engines (Memory, SQLite, Pebble, DuckDB, Postgres) are
`ReplicationNone` — single-node, local-only. The natural next step is a
distributed engine that provides coordination-free convergence across nodes.

The design doc
([meta-engine-eventual-consistency-and-iroh.md](../planning/meta-engine-eventual-consistency-and-iroh.md))
proposes [Iroh](https://iroh.computer) (by n0-computer) as the distributed
backend. Iroh is a Rust-implemented P2P data protocol with QUIC transport,
NAT traversal, and CRDT-based `iroh-docs` key-value store. The CALM theorem
guarantees that the metaengine's monotonic fold operations (MapInsert, SetAdd,
CounterIncrement, etc.) converge without coordination when replicated via CRDTs.

This ADR records the findings of a binding-availability evaluation and the
recommended bridge approach.

## Research Findings

### Iroh Go Binding Availability (as of 2026-08-03)

| Source                                      | Status                  | Platforms                                 | Notes                                                                   |
| ------------------------------------------- | ----------------------- | ----------------------------------------- | ----------------------------------------------------------------------- |
| Official Rust crate (`iroh`)                | v0.97.0                 | All                                       | Described as "stabilized 1.0 surface" despite 0.x version               |
| Official Python/Swift/Kotlin/JS             | Official                | All                                       | Via `iroh-ffi` (uniffi)                                                 |
| Official C bindings (`iroh-c-ffi`)          | Official                | Linux x86_64, macOS arm64, Windows x86_64 | Raw C API via `safer-ffi`; no prebuilt binaries (must `cargo build`)    |
| Go (`decentral1se/iroh-go`)                 | Community, experimental | Linux x86_64 + aarch64 (musl)             | Not built by n0; may lag behind releases                                |
| `n0-computer/iroh-ffi/iroh-go` (pkg.go.dev) | **Abandoned**           | —                                         | Legacy v0.12.0 from pre-1.0 era; directory removed from `iroh-ffi` repo |

### Critical Gap: `iroh-docs` Not in C FFI

The design doc's entire Level 2 architecture depends on `iroh-docs` (the CRDT
key-value store). However, `iroh-ffi` explicitly states: _"higher-level
protocols not yet at 1.0 (`iroh-blobs`, `iroh-docs`, `iroh-gossip`) are out of
scope."_ The `iroh-c-ffi` covers only the networking layer (endpoints,
connections, tickets, relays) — not `iroh-docs`.

This means:

- **CGo FFI over `iroh-c-ffi`**: Can access QUIC networking, NAT traversal,
  and blob transfer — but NOT the CRDT document store that the design requires.
- **CGo FFI over raw Rust crate**: Would require building a custom C shim over
  `iroh-docs` (significant effort, fragile coupling to internal APIs).
- **Community `iroh-go`**: Experimental, Linux-only, unclear `iroh-docs`
  coverage. Not production-grade.

## Decision

### Short-term: Sidecar process (when needed)

If a distributed engine prototype is needed before `iroh-docs` stabilizes in
the C FFI, use a **sidecar process** approach:

1. Run an Iroh node as a separate process (Rust binary)
2. Communicate via local gRPC or Unix domain socket
3. Implement `metaengine.EngineProfile` with the sidecar as the backing store

**Pros**: No CGo required, language-agnostic, clean process isolation.
**Cons**: Operational complexity (process lifecycle, health checks), latency
overhead of IPC (~100us per call on Unix socket), no shared-memory optimizations.

### Long-term: CGo FFI (when `iroh-docs` reaches C FFI)

Once `iroh-docs` is available in `iroh-c-ffi` (or a community Go binding matures
to production quality), switch to **CGo FFI**:

1. Isolate in `metaengine/irohengine/` with its own `go.mod`
2. `//go:build cgo` guard (same pattern as `stack/duckdb`)
3. Static link the C library (~same binary size impact as DuckDB's C++ engine)

**Pros**: Direct API calls (no IPC overhead), same module isolation pattern as
DuckDB, no separate process to manage.
**Cons**: CGo required, Rust toolchain needed for building the C library.

### Architecture: Level 2 (Iroh as replication layer)

The design doc's Level 2 approach is recommended over Level 1:

```go
// Future API (not implemented):
distributed := irohengine.Replicated(
    sqliteEngine,            // or pebbleEngine, memoryEngine
    irohengine.WithNamespace(docNamespace),
    irohengine.WithAuthor(authorKeypair),
)
```

- **Reads** always hit the local engine (full query power retained)
- **Writes** apply locally, then replicate via `iroh-docs` CRDT
- **Sync** applies incoming CRDT updates to the local engine
- **Planner** is unchanged — it picks the local engine; replication is transparent

This is architecturally superior to Level 1 (Iroh as direct engine) because:

1. Full query power retained (SQLite pushdown, DuckDB vectorization, Pebble point reads)
2. No new EngineProfile to teach the planner about
3. CRDT-safe operations replicate automatically; non-CRDT operations (MapUpdate)
   are caught by the existing `mapUpdateReplicationRule` WARN diagnostic

### No Implementation Now

This ADR records the decision and research findings. No code is written.
The `mapUpdateReplicationRule` (T18) and `WithReplication`/`WithNetworkRTT`
plan options (T14-T15) already provide the planner infrastructure for
distributed engines. When `iroh-docs` C bindings mature, the implementation
can proceed without planner changes.

## Prototype Implementation (2026-08-04)

A Level 2 replication wrapper prototype is now available in
`metaengine/irohengine/`. It validates the architecture from this ADR using a
pure-Go in-process `Network` that simulates P2P delivery — no CGo, no Rust,
no real Iroh dependency required.

### What the prototype proves

1. **Level 2 architecture works** — `Replicated(localEngine, ...)` wraps any
   existing Engine and transparently replicates CRDT-safe writes
2. **CALM theorem in practice** — monotonic folds (MapSet, SetAdd,
   CounterIncrement, MultiAdd, LogAppend) converge across nodes without coordination
3. **Non-CRDT detection** — MapUpdate (atomic RMW) executes locally but does NOT
   replicate (verified by test)
4. **LWW resolution** — concurrent MapSet from different nodes resolves to the
   latest timestamp
5. **PN-Counter** — concurrent CounterIncrement from multiple authors sums correctly
6. **Cross-engine parity** — `adttest.RunMatrix` passes (identical results to Memory)

### Transport interface

The `Transport` interface is the swap point for a real Iroh integration:

```go
type Transport interface {
    Publish(ctx context.Context, op WriteOp) error
    Subscribe(handler func(op WriteOp)) error
    Close() error
}
```

The in-process `Network` implements this today. When `iroh-docs` C bindings
mature, an `IrohTransport` struct implements the same interface via CGo FFI —
zero wrapper changes needed.

### What the prototype does NOT do

- ~~Real QUIC networking (mock transport is in-process, synchronous by default)~~ **DONE** — see `metaengine/irohengine/quic/` for real QUIC transport
- Real Iroh `iroh-docs` CRDT range reconciliation (our Go CRDT logic handles convergence; `iroh-docs` is a separate optional track)
- ~~Actual NAT traversal / P2P discovery~~ **DONE** — `iroh-go` provides NAT traversal via relay servers (`PresetN0()`); `WithLocalOnly()` disables it for tests
- ~~`WithNetworkDelay` / `WithNetworkDropRate` simulate adverse conditions~~ Still available on `InProcessNetwork` for CI without CGo; `tc netem` can shape real traffic for `QuicTransport`

## Real QUIC Implementation (2026-08-04)

### What was built

`metaengine/irohengine/quic/` is a separate Go module (own `go.mod`, `//go:build cgo` guard)
that implements `irohengine.Transport` over real Iroh QUIC streams:

```go
transport, _ := quic.New(quic.WithLocalOnly())
engine := irohengine.Replicated(
    metaengine.NewMemoryEngine(),
    irohengine.WithTransport(transport),
)
ticket, _ := transport.Ticket()
// Share ticket with another process...
otherTransport.Connect(ticket)
```

### How it works

1. Each node binds a real Iroh QUIC endpoint (`iroh.EndpointBind`)
2. Node A generates a ticket (`EndpointTicketFromAddr`) and shares it (base32 string)
3. Node B parses the ticket and connects (`endpoint.Connect`)
4. Writes serialize `WriteOp` as JSON and send over `conn.OpenBi()` BiStreams
5. The accept loop (`conn.AcceptBi()`) deserializes and dispatches to subscribers
6. RTT is measured from QUIC's own ACK timing (`conn.Rtt()`) — not `time.Sleep`

### What was verified

- 7 convergence tests pass (Map, Bidirectional, PN-Counter, Set, LWW, RTT, Log)
- All tests pass 3x with `-race`
- Multi-process demo: coordinator + node in separate processes see all keys
- Both nodes see 6/6 keys (3 from each side) with real QUIC delivery

## Consequences

1. **No immediate action required** — the planner infrastructure is in place
2. **Sidecar is the fallback** if distributed engine is needed urgently
3. **CGo FFI is the target** once `iroh-docs` stabilizes in C FFI
4. **Module isolation** (`metaengine/irohengine/` with own `go.mod`) ensures
   consumers who don't import it never need CGo or the Rust toolchain
5. **The CALM theorem connection** ensures monotonic folds converge without
   coordination — the planner only reasons about cost, not consistency
