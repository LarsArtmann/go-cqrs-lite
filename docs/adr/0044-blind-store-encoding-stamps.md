# ADR 0044: Blind Store Encoding Stamps

> **Status:** ACCEPTED — implemented in v3.8.0. Envelope wrapping is active for
> all blind stores (kv.TypedStore, snapshot.TypedStore, command.TypedStore,
> query.TypedStore). Old unenveloped data auto-detected via backward-compatible
> fallback decode. v4 codec default flip is now unblocked.
> **Date:** 2026-07-01
> **Related:** `docs/v4-WISHLIST.md` item #8, `docs/migration/JSON_TO_CBOR.md`

---

## Context

The "blind" stores — `kv.TypedStore`, `snapshot.TypedStore`, `command.TypedStore`,
`query.TypedStore` — serialize Go values to raw `[]byte` using a codec (default JSON),
then persist the bytes with **no format tag**. The encoding is a deployment-time
agreement between the writer and reader. This works today because:

1. The codec is set once at construction (`kv.WithTypedCodec(c)`) and both write
   and read paths use the same codec instance.
2. The data is ephemeral (read models, snapshots, audit logs) — it can be
   rebuilt from the event log if the codec changes.

**The problem:** Once a blind store has data written with codec A, switching to
codec B requires a full clear-and-rebuild. There is no way to detect that stored
data was written with a different codec. A consumer who changes the codec without
clearing data gets silent corruption: the new codec tries to decode JSON bytes as
CBOR (or vice versa), producing garbage values or cryptic unmarshal errors.

Events do NOT have this problem — `ImmutableEvent.Encoding` stamps the format on
every event, and `event.DecodePayload` validates the match.

## Decision

### Option A: Envelope Wrapper (RECOMMENDED)

Wrap every serialized value in a thin envelope with a format tag:

```go
type envelope struct {
    Encoding string `json:"enc"` // or cbor field
    Data     []byte `json:"dat"`
}
```

The TypedStore encodes the envelope (not the raw value). On decode, it reads
the envelope, picks the codec from `Encoding`, and decodes `Data`.

**Pros:**

- Simple to implement (2 lines in encode/decode)
- Backwards-compatible: old unenveloped data can be detected by
  `codec.AutoDetect(envelopeBytes)` falling back to raw decode
- Works for any codec, including custom ones

**Cons:**

- Adds 20-30 bytes overhead per value (the envelope wrapper)
- Changes the wire format (breaking for external readers of the raw KV store)

### Option B: Key Prefix (REJECTED)

Prefix the key with the encoding: `cbor:user:01J...` vs `json:user:01J...`.

**Rejected because:** Keys are opaque in some KV backends (Pebble sorts by key),
and the prefix pollutes the key space. Migration requires rewriting every key.

### Option C: Codec Auto-Detection on Read (PARTIAL)

Use `codec.AutoDetect(data)` to sniff the format on every read.

**Partial because:** AutoDetect is a heuristic, not a guarantee. Two codecs can
produce overlapping byte ranges for certain payloads. Safe for diagnostics,
unsafe as the sole mechanism for data integrity.

## Implementation Plan (v4)

1. Add `codec.Envelope` type with `Encode(v any, c codec.Codec) ([]byte, error)`
   and `Decode(data []byte) (codec.Codec, []byte, error)`.
2. TypedStore uses `Envelope.Encode` for writes and `Envelope.Decode` for reads.
3. Decode falls back to raw mode (no envelope) for backwards compatibility:
   - If `AutoDetect` + raw decode succeeds, use the result.
   - If envelope is present, use the stamped encoding.
4. Migration tool: scan all keys, re-wrap raw data in envelopes.

## Alternatives Considered

- **Do nothing:** Keep blind stores blind, document the risk. This is the v3.x
  status quo — consumers who want CBOR must pass it explicitly per store.
- **Format byte prefix:** Prepend a single magic byte (`0x00` = JSON, `0x01` = CBOR)
  before the serialized data. Simpler than an envelope but less extensible.

## Consequences

- **v4 breaking change:** The wire format of blind store data changes.
  Consumers with existing data must migrate (clear and rebuild, or run the
  migration tool).
- **Forward-compatible:** New codecs (MessagePack, Protocol Buffers) can be
  added without changing the envelope structure.
- **Debugging:** `codec.AutoDetect(envelope.Data)` still works on the inner bytes.
