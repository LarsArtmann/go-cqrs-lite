# ADR-0013: Zero-Copy Payload Access via payloadForDecode

**Date:** 2026-06-09
**Status:** Accepted

## Context

`ImmutableEvent.Payload()` returns `slices.Clone(e.payload)` to enforce immutability — callers receive an independent copy they can safely mutate without affecting the event's internal state. This costs 37–1165 ns + 1 alloc per call depending on payload size.

Most internal callers (decoding, signing, serialization) only **read** the bytes. The clone is pure overhead for these paths. In `DecodePayload`, which is called on every event in a projector, this clone adds measurable allocation pressure.

`*ImmutableEvent` is the only production `Event` implementation. One test-only stub exists, which the fallback path handles.

## Decision

Introduce an unexported `payloadForDecode(evt Event) []byte` function that type-asserts to `*ImmutableEvent` for direct field access, falling back to `Payload()` for any custom implementation:

```go
func payloadForDecode(evt Event) []byte {
    if ie, ok := evt.(*ImmutableEvent); ok {
        return ie.payload
    }
    return evt.Payload()
}
```

Use this function in all internal read-only paths:

- `DecodePayload` — payload is decoded (read) then discarded
- `DecodePayloads` — batch version, same read-only semantics
- `copyWithMetadata` (tombstone/rebirth) — copies into a new event via `make+copy`

The public `Payload()` method continues to clone for third-party safety.

## Consequences

- **+** Eliminates 1 alloc per `DecodePayload` call (verified by benchmarks)
- **+** Public API unchanged — `Payload()` still clones
- **+** Fallback path ensures correctness for any `Event` implementation
- **-** Internal coupling to `*ImmutableEvent` concrete type — adding a second production `Event` impl requires updating this function
- **-** Pattern must be applied consistently; new internal paths should use `payloadForDecode` instead of `Payload()`

## Benchmark Evidence

| Path | 16B payload | 256B payload |
|------|-------------|--------------|
| `Payload()` (clone) | ~37 ns, 1 alloc | ~65 ns, 1 alloc |
| Direct field access | ~1 ns, 0 allocs | ~1 ns, 0 allocs |
| `DecodePayload` (optimized) | Matches direct | Matches direct |

## Updated Files

- `event/codec.go` — `payloadForDecode()` function + usage in `DecodePayload`, `DecodePayloads`
- `event/tombstone.go` — `copyWithMetadata` uses `payloadForDecode` to avoid double-clone
- `event/benchmark_clone_test.go` — benchmarks confirming the optimization
