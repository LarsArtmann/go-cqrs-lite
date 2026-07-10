# ADR 0048: Deterministic JSON Encoding in Security-Critical Paths

> **Status:** ACCEPTED
> **Date:** 2026-07-10

## Context

`encoding/json/v2` Marshal produces non-deterministic output for types
containing maps: map iteration order is randomized. This means the same
value can produce different bytes on consecutive Marshal calls.

For most code paths, non-deterministic encoding is harmless. But for
**security-critical paths**, it breaks correctness:

1. **Event signing** (`signing/`): The HMAC-SHA256 / Ed25519 signature is
   computed over the event's JSON encoding. Non-deterministic encoding means
   the same event produces different signatures, breaking verification.

2. **Event encryption** (`encryption/`): The envelope and ciphertext are
   JSON-encoded before encryption. Non-deterministic encoding means the
   same plaintext produces different ciphertexts, which is correct for
   security (nonce randomization) but breaks any comparison or caching
   that assumes deterministic ciphertext.

3. **Catalog exports** (`catalog/`): API documentation output should be
   stable for diffing and caching. Non-deterministic JSON means every
   export produces a diff.

## Decision

All `json.Marshal` calls in security-critical and output-stable paths use
`json.Deterministic(true)`:

```go
json.Marshal(v, json.Deterministic(true))
```

Applied to:

- `signing/signature.go` — Signature MarshalJSON
- `signing/multisig/extract.go` — MultiSignature encoding
- `encryption/envelope.go` — Envelope MarshalEnvelope
- `encryption/ciphertext.go` — Ciphertext MarshalJSON
- `event/metadata_json.go` — Metadata MarshalJSON
- `catalog/` — All export paths (already had this since v3.5.0)
- `codec/envelope.go` — Blind store envelope (ADR-0044)

## Alternatives Considered

- **Make JSONCodec.Encode deterministic by default** — Rejected for v3:
  would change wire format for all JSON-encoded events, breaking consumers.
  v4 may do this since events are self-describing.
- **Use CBOR everywhere** — CBOR is deterministic by default, but switching
  all codecs is a v4 breaking change.

## Consequences

- Security-critical encoding is stable and reproducible.
- Signing and verification work correctly across multiple calls.
- Catalog exports are diffable.
- Slight performance cost for map key sorting (negligible for security paths).
