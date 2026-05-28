# signing — Event Signing and Verification

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/signing.svg)](https://pkg.go.dev/github.com/LarsArtmann/go-cqrs-lite/signing)

Cryptographic event signing for tamper-proof event streams. Supports both shared-secret (HMAC-SHA256) and public-key (Ed25519) signing strategies.

```bash
go get github.com/larsartmann/go-cqrs-lite/signing
```

## Why Sign Events?

In distributed systems, events may pass through untrusted intermediaries (message brokers, edge caches, client devices). Signing events provides:

- **Tamper detection**: Any mutation of event content is detected on verification
- **Origin authenticity**: Ed25519 signatures prove which keypair produced an event
- **Audit integrity**: Stored events can be verified years later

## Quick Start

### HMAC-SHA256 (Shared Secret)

Best for same-organization microservices where all participants trust each other:

```go
signer, err := signing.NewHMAC([]byte("my-secret-key-thirty-two-bytes!!"))
if err != nil { ... }

sig, err := signer.Sign(event)
if err != nil { ... }

err = signer.Verify(event, sig)
// err is non-nil if the event was tampered with
```

### Ed25519 (Public Key)

Best for asymmetric scenarios (e.g., client devices sign, server verifies):

```go
// Generate a keypair
pubKey, privKey, err := signing.GenerateEd25519KeyPair()

// Client: sign with private key
signer, _ := signing.NewEd25519(privKey)
sig, _ := signer.Sign(event)

// Server: verify with public key
verifier, _ := signing.NewEd25519Verifier(pubKey)
err = verifier.Verify(event, sig)
```

### Auto-Sign on the Event Bus

```go
signer, _ := signing.NewHMAC(key)
bus.UsePublish(signing.SignMiddleware(signer))

// Every published event is automatically signed
```

### Auto-Verify on Receive

```go
verifier, _ := signing.NewEd25519Verifier(pubKey)
bus.Use(signing.VerifyMiddleware(verifier))

// Unsigned events pass through; signed events are verified
```

## Design

- **No external crypto dependencies**: Uses Go stdlib (`crypto/hmac`, `crypto/ed25519`)
- **Deterministic canonicalization**: Events are serialized to a deterministic byte format before signing, covering ID, type, aggregate, version, payload hash, and occurredAt
- **Signature in metadata**: Signatures attach as URL-safe base64 in event custom metadata (`event.signature`)
- **Composable middleware**: SignMiddleware and VerifyMiddleware integrate with the event bus
