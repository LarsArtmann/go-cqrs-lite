# ADR-0047: COSE Support for Signing, Encryption, and Codec

**Status:** Accepted
**Date:** 2026-07-10

## Context

The library already provides event signing (`signing/`) and payload encryption (`encryption/`) using a custom wire format:

- Signatures attach as URL-safe base64 in event custom metadata (`event.signature`).
- Ciphertext attaches as URL-safe base64 in event custom metadata (`event.encrypted`).
- Algorithm names are custom strings (`HMAC-SHA256`, `Ed25519`, `aes-256-gcm`, `xchacha20-poly1305`).

This works well internally, but it is not interoperable with other COSE-aware systems. The [IANA COSE registries](https://www.iana.org/assignments/cose/cose.xhtml) define standard algorithm identifiers, header parameters, and binary envelope formats (RFC 9052/9053) that are widely used in IoT, WebAuthn, CWT, and mobile ecosystems.

Adding COSE support lets consumers:

- Exchange signed or encrypted events with non-Go systems that already understand COSE.
- Use standard algorithm identifiers instead of custom strings.
- Benefit from CBOR's compact binary representation for signatures and encrypted envelopes.
- Feed events into COSE-based audit, attestation, or telemetry pipelines.

## Decision

Add COSE building blocks across `codec/`, `signing/`, and `encryption/`:

1. **`codec/cose.go`** — Low-level COSE structure serialization and helper functions:
   - Constants for COSE header parameter labels (`alg`, `kid`, `iv`, etc.).
   - Constants for COSE algorithm identifiers relevant to the project:
     - HMAC-SHA256 = 5
     - Ed25519 = -19
     - AES-256-GCM = 3
     - ChaCha20/Poly1305 = 24
   - `COSESign1` and `COSEEncrypt0` types.
   - `MarshalCOSESign1` / `UnmarshalCOSESign1`.
   - `MarshalCOSEEncrypt0` / `UnmarshalCOSEEncrypt0`.
   - `SigStructure` and `EncStructure0` builders for RFC 9052 canonical signing/encryption inputs.

2. **`signing/cose.go`** — COSE_Sign1 signing and verification:
   - `COSESigner` / `COSEVerifier` interfaces that sign/verify raw bytes and report a COSE algorithm ID.
   - `NewCOSEHMAC` and `NewCOSEEd25519Signer` / `NewCOSEEd25519Verifier` constructors.
   - `SignCOSE1(evt, signer, opts...)` — produces a COSE_Sign1 message signing the event's canonical representation.
   - `VerifyCOSE1(evt, verifier, coseBytes, opts...)` — verifies a COSE_Sign1 message against the event.
   - Optional `WithCOSEKeyID` and `WithCOSEExternalAAD` options.

3. **`encryption/cose.go`** — COSE_Encrypt0 encryption and decryption:
   - `COSEEncrypter` / `COSEDecrypter` interfaces that encrypt/decrypt with explicit AAD and nonce handling.
   - `NewCOSEXChaCha20Poly1305` and `NewCOSEAES256GCM` constructors.
   - `EncryptCOSE0(plaintext, encrypter, opts...)` — produces a COSE_Encrypt0 message with the Enc_structure as AAD.
   - `DecryptCOSE0(coseBytes, decrypter, opts...)` — decrypts a COSE_Encrypt0 message and verifies the algorithm identifier.
   - Optional `WithCOSEEncryptExternalAAD` option.

### Why COSE over raw custom formats

| Concern               | Custom format          | COSE                           |
| --------------------- | ---------------------- | ------------------------------ |
| Interoperability      | Go-only                | Any RFC 9052 implementation    |
| Algorithm IDs         | Custom strings         | IANA-registered integers       |
| Wire format           | Base64 JSON metadata   | Compact CBOR arrays            |
| Standard test vectors | None                   | RFC 9052 appendices, NIST CAVP |
| Future algorithms     | Manual string addition | Use existing IANA registry     |

### Why COSE_Sign1 and COSE_Encrypt0 (not COSE_Sign / COSE_Encrypt)

- COSE_Sign1 covers the single-signer case that matches the existing signing module's scope. Multi-signer scenarios are already handled by `signing/multisig`.
- COSE_Encrypt0 covers the single-recipient/symmetric-key case that matches the existing encryption module. Multi-recipient or key-wrapped scenarios can be added later if needed.

### Why sign the canonical event representation

`SignCOSE1` places the library's existing canonical signing bytes in the COSE payload field. This keeps the security coverage identical to the existing `signing.SignMiddleware` path: ID, type, aggregate, version, schema version, occurredAt, and payload hash are all authenticated. The COSE envelope becomes a portable wrapper around the existing trust boundary.

### Why keep the existing middleware and interfaces unchanged

COSE support is additive. The existing `Signer`/`Verifier` and `Encrypter`/`Decrypter` interfaces and middleware continue to work exactly as before. Consumers opt into COSE by using the new `COSESigner`/`COSEEncrypter` types and `SignCOSE1`/`EncryptCOSE0` functions.

## Consequences

- Consumers can produce and verify RFC 9052 COSE_Sign1 messages for events.
- Consumers can produce and decrypt RFC 9052 COSE_Encrypt0 messages for arbitrary payloads.
- No new external dependencies: the implementation reuses the existing `fxamacker/cbor/v2` dependency in `codec/` and standard library crypto in `signing/` and `encryption/`.
- The COSE layer uses the same canonical CBOR encoding mode as `CBORCodec`, ensuring deterministic `Sig_structure`/`Enc_structure` bytes.
- The existing signing and encryption middleware remain the default; COSE is an opt-in, interoperable escape hatch.
- `signing/go.mod` and `encryption/go.mod` need to include the same replace directives that `event/go.mod` already has for the new transitive workspace dependencies.

## Future work (not in this ADR)

- COSE-aware middleware that attaches COSE_Sign1/COSE_Encrypt0 bytes to event metadata.
- COSE key distribution helpers (`COSE_Key`, `COSE_KeySet`) for consumers who need to transport keys.
- COSE countersignatures for the `signing/multisig` module.
- COSE_Mac0 support for message authentication codes using the same HMAC primitives.
