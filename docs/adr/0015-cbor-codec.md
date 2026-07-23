# ADR-0015: CBOR Codec for Binary Payload Encoding

> **SUPERSEDED (2026-07-11):** This ADR introduced the CBOR codec but kept
> JSON as the default. That default was subsequently flipped to CBOR by
> [ADR-0051](0051-cbor-as-default-codec.md) (event.New default) and
> [ADR-0053](0053-unified-codec-default-flip.md) (all blind store defaults).
> The codec design and rationale below remain accurate; only the "JSON
> remains the default" conclusion is outdated.

**Status:** Superseded by [0051](0051-cbor-as-default-codec.md)/[0053](0053-unified-codec-default-flip.md)
**Date:** 2026-06-11

## Context

The `codec/` module provided `JSONCodec` (encoding/json) and `RawCodec` (passthrough). JSON is the default for all event payloads. However, JSON has drawbacks for event-sourced systems:

1. **Non-deterministic encoding**: JSON map key ordering is undefined in the Go spec. `encoding/json` happens to sort keys, but this isn't guaranteed across implementations. Two identical structs can produce different bytes, making HMAC/Ed25519 signatures non-deterministic.

2. **No native byte strings**: `[]byte` fields are base64-encoded, adding ~33% overhead for binary payloads (images, encrypted blobs, signatures).

3. **Numeric bloat**: JSON encodes all numbers as decimal strings. CBOR uses compact binary representations.

4. **Parse overhead**: JSON requires string parsing for every field name and value. CBOR's binary format avoids this.

The library needs an opt-in binary codec that consumers can use via `event.WithCodec(codec.CBORCodec{})`.

## Decision

Add `CBORCodec` using `fxamacker/cbor/v2` with canonical encoding (RFC 7049):

- **Canonical mode**: Sorted map keys, shortest float encoding. Produces deterministic output — same struct always produces same bytes. Critical for signing safety.
- **Zero-value struct**: `CBORCodec{}` follows the same pattern as `JSONCodec{}`. No configuration needed.
- **Opt-in**: Consumers must explicitly pass `event.WithCodec(codec.CBORCodec{})`. JSON remains the default.
- **Minimal dependency**: `fxamacker/cbor/v2` has zero transitive deps (only `x448/float16`). No `unsafe` usage.

### Why CBOR over alternatives

| Alternative       | Rejected because                                                                                        |
| ----------------- | ------------------------------------------------------------------------------------------------------- |
| **msgpack**       | No deterministic encoding mode (signing hazard). No IETF standard. Ambiguous "raw" type.                |
| **protobuf**      | Requires code generation per event type — wrong for a library where consumers define their own types.   |
| **FlatBuffers**   | Requires schema compilation per event type. Good for zero-copy reads but event sourcing is append-only. |
| **gob**           | Go-specific, includes type names in output (brittle), security concerns with untrusted input.           |
| **custom binary** | Reinventing the wheel. CBOR already solves this with IETF standardization.                              |

### Why `fxamacker/cbor` specifically

- IETF STD 94 compliant (RFC 8949)
- Used by Arm, Microsoft, Red Hat, IBM in production
- Zero `unsafe` usage (memory-safe)
- API mirrors `encoding/json` (`Marshal`/`Unmarshal`)
- Canonical encoding mode for deterministic output
- Only transitive dep: `x448/float16`

### Why Canonical (RFC 7049) instead of CoreDet (RFC 8949)

Both produce identical bytes for typical event payloads (maps with string keys, integers, strings, booleans). The difference only matters for edge cases (bignums, maps with bytestring keys). Canonical is more widely recognized and is sufficient for our signing safety requirements.

## Consequences

- Consumers can opt into CBOR by passing `event.WithCodec(codec.CBORCodec{})` when creating events.
- All storage layers (Pebble, SQL, memory, Turso) store raw bytes + encoding string — no changes needed.
- The `EncryptionCodec` wrapper works with CBOR (wraps any `Codec`).
- CBOR payloads are ~20-40% smaller than JSON for the same data.
- CBOR encode is ~17% faster with 50% fewer allocations. Decode is ~32% faster with 25% fewer allocations.
- Events encoded with CBOR have `Encoding() == "cbor"`. The `validateEncodingMatch` function enforces codec-encoding match on decode.
- Golden files and existing tests use `JSONCodec` by default — no disruption.
