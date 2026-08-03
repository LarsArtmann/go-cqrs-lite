# ADR-0097: SSE Three-Repo Finding and go-sse Consumption

**Date:** 2026-08-03
**Status:** Accepted
**Supersedes:** None (extends ADR-0091 with new context)
**Related:** ADR-0025 (Transport: HTTP SSE + gRPC), ADR-0091 (SSE Consolidation Decision)

## Context

ADR-0091 decided to keep `metaengine.ServeSSE` and `transport/http.SSEBroker`
as separate implementations, rejecting a shared SSE utility package because
"the shared code is trivial Go stdlib."

**That decision was made without knowledge that `go-sse` already exists.**

During the 2026-08-03 ADR review, a three-repo investigation revealed:

1. **`github.com/larsartmann/go-sse`** — a standalone SSE primitives library
   extracted from production. It provides:
   - `Event` (typed SSE event: ID, Type, Data, Retry)
   - `Stream` (wraps `http.ResponseWriter` + `http.Flusher`, writes SSE wire format)
   - `Broadcaster[T]` (fan-out to N concurrent subscribers with non-blocking send)
   - `EventStore` interface + `Replay` (replay missed events from a journal)
   - Zero non-stdlib dependencies

2. **`github.com/larsartmann/cqrs-htmx`** — a consumer project that correctly
   imports `go-sse` and uses `Broadcaster[sse.Event]` with ~110 lines of
   CQRS-specific glue code on top of go-sse primitives.

3. **`go-cqrs-lite`** — has TWO hand-rolled SSE implementations, neither of
   which imports `go-sse`:
   - `transport/http/sse.go` (~500 LOC): hand-rolled wire format (manual
     `fmt.Fprintf("data: %s\n\n")`), manual client map with `sync.Mutex`,
     manual fan-out loop, manual SSE header writing.
   - `metaengine/sse.go` (~200 LOC): similar hand-rolled wire format and
     client management for `Watcher[V]` push.

### The Duplication

Both implementations duplicate the same ~100 LOC of SSE wire-format code that
`go-sse.Stream` already provides:
- `Content-Type: text/event-stream` header
- `Cache-Control: no-cache` header
- `Connection: keep-alive` header
- `id:` / `event:` / `data:` / `retry:` field writing
- `\n\n` terminator
- `http.Flusher` flush after each event

### ADR-0091's Gap

ADR-0091's rejection of a shared SSE utility was based on two premises:

1. *"The shared code is trivial Go stdlib"* — True for the wire format, but
   `go-sse.Broadcaster[T]` provides non-trivial fan-out logic (subscriber
   lifecycle, non-blocking send, graceful close) that both implementations
   hand-roll independently.

2. *"Extracting it adds coupling without meaningful deduplication"* — The
   coupling already exists: both implementations depend on the same SSE
   wire format and fan-out semantics. `go-sse` as an external dependency adds
   zero internal coupling (it's not a go-cqrs-lite module).

## Decision

**Consume `go-sse` as an internal dependency for SSE wire-format and fan-out
primitives, while preserving the two-implementation architecture from
ADR-0091.**

### Scope

This ADR does NOT reverse ADR-0091's core decision (keep both implementations
separate). It supplements it: both implementations should consume `go-sse`
primitives internally to eliminate wire-format duplication.

### What changes

1. `transport/http.SSEBroker` — internally uses `sse.Stream` for wire format
   and `sse.Broadcaster` for fan-out. All existing features (event filtering,
   payload transform, byte budget, REST backfill, OTel spans, graceful
   shutdown) are preserved as SSEBroker-specific wrappers around go-sse
   primitives.

2. `metaengine.ServeSSE` — internally uses `sse.Stream` for wire format and
   `sse.Broadcaster[V]` for fan-out. All existing features (ring buffer
   replay, heartbeat keepalive) are preserved.

### What does NOT change

- The external API of both `SSEBroker` and `ServeSSE` remains identical.
- The two implementations remain in their respective modules.
- The module dependency boundary is preserved (metaengine gets go-sse as an
  external dep, not a go-cqrs-lite module).
- ADR-0091's layer comparison and rationale remain valid.

## Consequences

- ~300 LOC of duplicated SSE wire-format code eliminated.
- Both implementations benefit from go-sse's tested wire-format implementation.
- `go-sse` becomes a dependency of both `metaengine` and `transport/http`.
  This is acceptable: `go-sse` has zero non-stdlib dependencies and is
  versioned independently.
- Future SSE bug fixes (e.g., CRLF injection prevention) are fixed once in
  `go-sse` rather than twice in go-cqrs-lite.
- The refactor is internal-only: no consumer code changes required.

## Alternatives Considered

### Status quo (keep hand-rolled wire format)

Rejected: The wire-format code is duplicated and was written before go-sse
existed. Maintaining two copies of SSE wire-format logic is technical debt.

### Merge both implementations (reverse ADR-0091)

Rejected: ADR-0091's rationale remains valid — the two implementations serve
different layers with different data sources, replay strategies, and feature
sets. This ADR supplements ADR-0091, it does not reverse it.

### Extract a new `go-cqrs-lite/sse` module

Rejected: `go-sse` already exists and is the right home for SSE primitives.
Creating a parallel implementation within go-cqrs-lite would duplicate
go-sse's purpose.
