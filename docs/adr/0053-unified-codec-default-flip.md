# ADR 0053: Unified Codec Default Flip (JSON → CBOR)

> **Status:** ACCEPTED
> **Date:** 2026-07-11
> **Related:** ADR-0015 (CBOR codec introduction), ADR-0044 (blind store encoding stamps), ADR-0050 (envelope JSON fallback), ADR-0051 (event.DefaultCodec flip), ADR-0052 (transport boundary codec strategy)

## Context

Historically, every codec default in the library was `JSONCodec`. This was the
safe, conservative choice — JSON is human-readable and universally understood.

Over time, CBOR proved superior for machine-to-machine serialization:
~19-35% smaller payloads, ~32% faster encoding, deterministic output. But
flipping defaults is a breaking change: consumers who upgrade without noticing
the flip get silent data corruption when the new codec tries to decode old data.

Three prior ADRs built the safety net that makes the flip safe:

1. **ADR-0044** — Envelope wrapping for blind stores (`kv`, `snapshot`,
   `command`, `query`). Every write stamps the encoding; every read
   auto-detects it.
2. **ADR-0050** — The `UnwrapDecode` fallback path permanently uses
   `JSONCodec`. Pre-envelope data (raw JSON) is transparently handled.
3. **ADR-0051** — `event.DefaultCodec` flipped to `CBORCodec`. Events are
   self-describing via `evt.Encoding()`, so mixed streams decode correctly.

The remaining step: flip the blind store defaults. This is the umbrella ADR
that documents the unified state — **every codec default in the library is now
CBOR** — and the backward-compatibility guarantees that protect consumers.

## Decision

All codec defaults are `CBORCodec`:

| Layer                             | Old Default | New Default | Override                                                         |
| --------------------------------- | ----------- | ----------- | ---------------------------------------------------------------- |
| `event.New()`                     | JSONCodec   | CBORCodec   | `event.DefaultCodec = codec.JSONCodec{}` or `event.WithCodec(c)` |
| `kv.NewTypedStore`                | JSONCodec   | CBORCodec   | `kv.WithTypedCodec(c)`                                           |
| `snapshot.NewTypedStore`          | JSONCodec   | CBORCodec   | positional arg: `NewTypedStore(store, c)`                        |
| `command.NewTypedStore`           | JSONCodec   | CBORCodec   | positional arg                                                   |
| `query.NewTypedStore`             | JSONCodec   | CBORCodec   | positional arg                                                   |
| `stack.ReadModel` / `Materialize` | JSONCodec   | CBORCodec   | `stack.WithDefaultCodec(c)`                                      |

### Backward Compatibility

**Events:** Each event stamps its encoding at creation time via
`ImmutableEvent.Encoding`. `event.DecodePayloadAuto[T]` dispatches on this
stamp. A mixed stream of JSON and CBOR events decodes correctly with no
configuration.

**Blind stores:** Every write goes through `codec.WrapEncode`, which produces
a JSON envelope (`{"$":"cqrs","enc":"cbor","dat":"..."}`). Every read goes
through `codec.UnwrapDecode`, which detects the envelope and uses the stamped
codec. Old pre-envelope data (raw JSON) is caught by the fallback path, which
permanently uses `JSONCodec` (ADR-0050).

**What this means for consumers upgrading to v4:** stored data from v3 decodes
correctly with zero migration. New writes are CBOR. The codec transition is
gradual — old keys decode as JSON, new keys decode as CBOR, both coexist.

### Testing

The migration path is verified by integration tests in `kv/typed_store_test.go`:
`TestTypedStore_Migration_OldRawJSON_ReadByNewCBORDefault` and
`TestTypedStore_Migration_MixedOldAndNewData`. These simulate the exact
scenario every consumer walks: old raw JSON data read through a new
CBOR-default store.

## Consequences

- **Payloads are smaller and faster by default.** CBOR's binary encoding
  produces ~19-35% smaller payloads with faster encoding/decoding.
- **No data migration required.** The envelope + fallback safety net handles
  the transition transparently.
- **Override is always available.** Consumers who need JSON (e.g., for
  debugging, external tooling compatibility) can revert per-store or
  process-wide.
- **Transport boundary is separate (ADR-0052).** SSE and HTTP transports
  use `WithPayloadTransform` to convert payloads to JSON at the wire
  boundary. This is independent of the storage codec default.
- **`BackfillHandler` uses the broker's transform** — the same transform
  configured for SSE applies to REST backfill responses, so transport-layer
  transcoding is unified.
