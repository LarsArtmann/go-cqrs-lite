# ADR-0013: Zero-Copy Payload Access

**Date:** 2026-06-09
**Status:** Accepted
**Updated:** 2026-06-09 — added cross-module `PayloadReadOnly` export

## Context

`ImmutableEvent.Payload()` returns `slices.Clone(e.payload)` to enforce immutability — callers receive an independent copy they can safely mutate without affecting the event's internal state. This costs 37–1165 ns + 1 alloc per call depending on payload size.

Most callers only **read** the bytes (decoding, signing, serialization, SQL inserts). The clone is pure overhead for these paths.

`*ImmutableEvent` is the only production `Event` implementation. One test-only stub exists, which the fallback path handles.

## Decision

### Phase 1: Internal zero-copy (unexported)

`payloadForDecode(evt Event) []byte` type-asserts to `*ImmutableEvent` for direct field access, falling back to `Payload()` for custom implementations. Used within `event/` package:

- `DecodePayload` / `DecodePayloads` — payload decoded then discarded
- `copyWithMetadata` (tombstone/rebirth) — copies into new event

### Phase 2: Cross-module zero-copy (exported)

`PayloadReadOnly(evt Event) []byte` exports the same pattern for read-only callers in other modules. The returned slice MUST NOT be mutated. Used in:

- `signing/payload.go` — SHA-256 canonical hashing
- `signing/event.go` — `CloneEvent` passes to `NewEvent` (which clones once)
- `pebble/serialization.go` — `json.Marshal` reads only
- `storage/sql/helpers.go` — `ExecContext` reads only
- `transport/http/sse.go` — string conversion for SSE data frame

`encodingForCopy(evt Event) codec.Encoding` preserves the raw encoding field without the `""`→`"json"` normalization that `Encoding()` applies. Used in `copyWithMetadata`.

### Builder optimization

`builder.Build()` calls `buildEvent()` directly instead of `NewEvent()`, since `WithPayload()` already cloned the payload. Eliminates one allocation per builder-built event.

## Consequences

- **+** Eliminates 1 alloc per `DecodePayload` call
- **+** Eliminates 5 wasted clones per event across signing/storage/pebble/middleware
- **+** Eliminates 1 alloc per builder-built event
- **+** Public `Payload()` API unchanged — still clones for third-party safety
- **+** Fallback path ensures correctness for any `Event` implementation
- **-** Internal coupling to `*ImmutableEvent` concrete type — adding a second production `Event` impl requires updating `payloadForDecode` and `PayloadReadOnly`
- **-** `PayloadReadOnly` is a trust-based API — callers must not mutate. Documented but not enforced at compile time.

## Benchmark Evidence

| Path                            | 16B payload       | 256B payload      |
| ------------------------------- | ----------------- | ----------------- |
| `Payload()` (clone)             | ~37 ns, 1 alloc   | ~65 ns, 1 alloc   |
| `PayloadReadOnly()` (zero-copy) | ~1 ns, 0 allocs   | ~1 ns, 0 allocs   |
| `DecodePayload` (optimized)     | Matches zero-copy | Matches zero-copy |

## Updated Files

- `event/codec.go` — `payloadForDecode()`, `PayloadReadOnly()`, `encodingForCopy()`
- `event/tombstone.go` — `copyWithMetadata` uses `payloadForDecode` + `encodingForCopy`
- `event/builder.go` — `Build()` calls `buildEvent()` directly
- `signing/payload.go` — `canonicalPayload` uses `PayloadReadOnly`
- `signing/event.go` — `CloneEvent` uses `PayloadReadOnly`
- `pebble/serialization.go` — `serializeEvent` uses `PayloadReadOnly`
- `storage/sql/helpers.go` — `SharedInsertEvents` uses `PayloadReadOnly`
- `transport/http/sse.go` — SSE data frame uses `PayloadReadOnly`
- `event/benchmark_clone_test.go` — benchmarks confirming optimizations
