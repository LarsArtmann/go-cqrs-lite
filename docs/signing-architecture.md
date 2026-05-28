# Signing Module Architecture

**Module:** `github.com/larsartmann/go-cqrs-lite/signing`  
**Coverage:** 94.2%  
**Algorithms:** HMAC-SHA256, Ed25519  
**Last Updated:** 2026-05-28

---

## Overview

The signing module provides cryptographically verifiable event integrity for the go-cqrs-lite library. Every event can carry one or more signatures, enabling tamper detection without requiring a separate audit log or database-level constraints.

**Design goal:** Zero trust in storage. If an attacker modifies event payloads in the database, signature verification fails on load.

---

## Core Concepts

### Canonical Payload

Signatures are computed over a **canonical payload** — a deterministic byte representation of event fields. This ensures that two identical events always produce the same signature, regardless of when or where they are created.

```
canonicalPayload = sha256(
    event.Type() + "|" +
    event.AggregateID().String() + "|" +
    event.AggregateType() + "|" +
    strconv.Itoa(int(event.Version())) + "|" +
    base64(event.Payload())
)
```

**Key properties:**
- Field order is fixed (type → aggregate ID → aggregate type → version → payload)
- Payload is base64-encoded to handle binary data deterministically
- No metadata, timestamps, or event IDs are included — these change between clones

### Signature Types

| Type | Use Case | Performance | Key Management |
|------|----------|-------------|----------------|
| `HMAC-SHA256` | Same-organization microservices | ~750 ns/op | Shared secret |
| `Ed25519` | Cross-organization / device auth | ~14 µs/op sign, ~30 µs/op verify | Public/private key pair |

**HMAC-SHA256** is preferred when all participants trust each other (same-organization). It's 20× faster than Ed25519.

**Ed25519** is required when participants don't share secrets (devices, third-party services, audit authorities). It provides non-repudiation: a verifier cannot forge signatures.

---

## Single-Signature API

```go
// Create a signer (HMAC)
signer, _ := signing.NewHMAC(secretKey)
sig, _ := signer.Sign(evt)

// Verify
verifier, _ := signing.NewHMAC(secretKey)
err := verifier.Verify(evt, sig)

// Attach signature to event for storage/transport
signedEvt, _ := signing.AttachSignature(evt, sig)

// Extract and verify on the other side
extracted, _ := signing.ExtractSignature(signedEvt)
```

---

## Multi-Signature API

Multi-signature enables **multi-party authorization** — multiple actors sign the same event, and a verifier checks that all required signatures are present and valid.

```go
// Each actor creates their own MultiSigner
deviceMulti, _ := signing.NewMultiSigner(
    signing.Actor("device"),
    signing.AlgorithmEd25519,
    deviceSigner,
    signing.WithVerifier(deviceVerifier),
)

// Sign the event (adds actor's signature to metadata)
deviceSigned, _ := deviceMulti.Sign(evt)

// Second actor signs the already-signed event
serverSigned, _ := serverMulti.Sign(deviceSigned)

// Verify all signatures at once
verifiers := signing.VerifierMap(deviceMulti, serverMulti)
err := signing.VerifyAll(serverSigned, verifiers)
```

**Key design decisions:**

1. **Interface segregation:** `VerifierMap` returns `map[Actor]Verifier`, not `[]*MultiSigner`. This keeps `VerifyAll` decoupled from the signer types — any `Verifier` implementation works.

2. **Re-sign replaces:** If the same actor signs an event twice, the old signature is replaced. This prevents duplicate entries.

3. **Metadata storage:** Multi-signatures are stored as JSON in `event.Metadata` under the key `_multisig`. This makes them transportable across buses and persistent in event stores without schema changes.

---

## Middleware Integration

The signing module provides middleware for both the **publish path** (sign before sending) and **consume path** (verify on receipt):

```go
// Publish side: automatically sign all outgoing events
bus.UsePublish(signing.SignMiddleware(signer))

// Subscribe side: verify signatures before handling
bus.Use(signing.VerifyMiddleware(verifier))

// Require signature: reject unsigned events
bus.Use(signing.RequireSignatureMiddleware(verifier))

// Multi-sig variants
bus.UsePublish(signing.MultiSignMiddleware(multiSigner))
bus.Use(signing.MultiVerifyMiddleware(multiSigner))
bus.Use(signing.RequireMultiSigMiddleware(verifierMap))
```

---

## Security Properties

| Threat | Defense |
|--------|---------|
| Tampered payload | Signature verification fails (`ErrInvalidSignature`) |
| Stolen signature replayed on different event | Canonical payload includes aggregate ID + version, so replay fails |
| Missing signature | `RequireSignatureMiddleware` rejects the event |
| Partial multi-sig (only some actors signed) | `VerifyAll` returns error listing missing actors |
| Clock skew in multi-sig timestamps | `SignedAt` is informational; verification uses cryptographic payload, not timestamp |
| Weak HMAC key | `NewHMAC` rejects keys shorter than 32 bytes |
| Invalid Ed25519 key | `NewEd25519` / `NewEd25519Verifier` validate key lengths |

---

## Module Boundaries

The signing module is the **most self-contained module** in the monorepo:

- **Imports from monorepo:** `core/event`, `core/pkg/id` only
- **External deps:** Standard library only (`crypto/hmac`, `crypto/sha256`, `crypto/ed25519`)
- **No imports from:** storage, memory, catalog, middleware, projection, saga, watermill

This means the signing module could be extracted to its own repository with minimal effort.

---

## File Map

| File | Lines | Purpose |
|------|-------|---------|
| `signer.go` | 171 | `Signature` type, `canonicalPayload`, `Signer`/`Verifier` interfaces |
| `hmac.go` | 78 | `hmacSigner` (unexported) — HMAC-SHA256 implementation |
| `ed25519.go` | 105 | `ed25519Signer`/`ed25519Verifier` (unexported) — Ed25519 implementation |
| `multisig.go` | 153 | `MultiSigner`, `Sign`, `Verify`, `verifyActorEntry` |
| `multisig_types.go` | 154 | `Actor`, `SignatureEntry`, `MultiSignature`, options |
| `multisig_extract.go` | 123 | `VerifyAll`, `VerifierMap`, `ExtractMultiSignature`, `attachMultiSignature` |
| `multisig_middleware.go` | 172 | 4 middleware functions with nil guards |
| `middleware.go` | 108 | `SignMiddleware`, `VerifyMiddleware`, `RequireSignatureMiddleware` |
| `event.go` | 85 | `cloneEvent`, `ExtractSignature`, `HasSignature` |
| `errors.go` | 31 | Sentinel errors (`ErrNilEvent`, `ErrInvalidSignature`, etc.) |
| `benchmark_test.go` | ~120 | HMAC + Ed25519 + VerifyAll benchmarks |

---

## Testing Strategy

- **Unit tests:** Every public function has table-driven tests
- **Middleware tests:** Nil guards, panic recovery, tamper detection
- **End-to-end:** Device signs → server verifies → server signs → both verify → tamper detection
- **Benchmarks:** HMAC sign/verify, Ed25519 sign/verify, VerifyAll with 2 actors
- **Coverage target:** >90% (currently 94.2%)
