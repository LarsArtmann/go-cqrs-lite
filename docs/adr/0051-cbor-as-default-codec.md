# ADR 0051: CBOR as Default Codec for event.New()

> **Status:** ACCEPTED
> **Date:** 2026-07-11
> **Related:** ADR-0015 (CBOR codec introduction), ADR-0044 (blind store encoding stamps), ADR-0050 (envelope JSON fallback)

## Context

ADR-0015 introduced the `CBORCodec` alongside the existing `JSONCodec`, but kept
JSON as the default for `event.New()`. Since then, the codec infrastructure has
matured significantly:

1. **ADR-0044 envelopes** — All blind stores (kv, snapshot, command, query) now
   stamp their encoding on write and auto-detect on read via `WrapEncode`/
   `UnwrapDecode`. The `UnwrapDecode` fallback path uses `JSONCodec` for
   backward compatibility with pre-envelope data (ADR-0050).

2. **`DecodePayloadAuto[T]`** — Events are self-describing: each event stamps
   its encoding via `ImmutableEvent.Encoding()`. Mixed JSON+CBOR event streams
   decode correctly with a single call. Consumers do not need to know which
   codec was used at creation time.

3. **Symmetric encoding validation** — Cross-decode mismatches (JSON event
   decoded with CBOR codec, and vice versa) are rejected explicitly.

4. **Performance** — CBOR produces ~19-35% smaller payloads and encodes ~32%
   faster than JSON for typical event payloads.

The remaining question was whether to keep JSON as the default (conservative,
zero surprise) or flip to CBOR (better defaults for production, aligns with
the blind-store defaults that already use CBOR).

## Decision

**`event.DefaultCodec` is now `codec.CBORCodec{}`.**

Events created via `event.New()` with no explicit `WithCodec` option are
CBOR-encoded. Events created via `event.NewEvent()` (raw `[]byte` constructor)
are unaffected — raw bytes are assumed JSON unless an explicit `WithCodec`
option stamps a different encoding.

Consumers who need JSON as the process-wide default can revert:

```go
event.DefaultCodec = codec.JSONCodec{}
```

Or override per-event:

```go
event.WithCodec(codec.JSONCodec{})
```

## Consequences

- **New events are CBOR by default.** Existing stored events retain their
  stamped encoding — `DecodePayloadAuto` handles mixed streams transparently.
- **`NewEvent()` (raw bytes) still defaults to JSON.** This is intentional:
  raw `[]byte` payloads are almost always JSON in existing code, and changing
  this would be a silent behavioral shift with no self-describing safety net.
- **Blind stores already default to CBOR** via ADR-0044 envelopes. This change
  aligns `event.New()` with that default, creating a consistent codec story
  across the library.
- **Frontend/transport boundary is not affected.** SSE and other HTTP transports
  that need JSON output use `WithPayloadTransform` at the boundary — this is
  a transport concern, not a codec-default concern. SSE's text framing makes
  CBOR structurally counterproductive (base64 overhead negates size savings).
- **Tests updated** to expect CBOR encoding for `event.New()` events unless
  an explicit `WithCodec(codec.JSONCodec{})` is provided.
