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

## Multi-Party Signing

When events travel through multiple actors (e.g., end-user device → server), each actor adds its own signature. The event accumulates a `MultiSignature` collection that anyone can verify.

### Device → Server Chain

```go
// Device signs with Ed25519 (private key stays on device)
deviceSigner, _ := signing.NewEd25519(devicePrivKey)
deviceVerifier, _ := signing.NewEd25519Verifier(devicePubKey)
deviceMulti := signing.NewMultiSigner("device", signing.AlgorithmEd25519, deviceSigner,
    signing.WithVerifier(deviceVerifier))

// Server signs with HMAC (shared secret within org)
serverSigner, _ := signing.NewHMAC(serverKey)
serverMulti := signing.NewMultiSigner("server", signing.AlgorithmHMACSHA256, serverSigner)

// Step 1: Device signs
signed, _ := deviceMulti.Sign(event)

// Step 2: Server verifies device, then adds its own signature
deviceMulti.Verify(signed) // verifies device's sig
signed, _ = serverMulti.Sign(signed) // appends server's sig

// Final event carries both signatures
multiSig, _ := signing.ExtractMultiSignature(signed)
multiSig.HasActor("device") // true
multiSig.HasActor("server") // true
```

### Verify All Actors

```go
verifiers := map[string]signing.Verifier{
    "device": deviceVerifier,
    "server": serverSigner,
}
err := signing.VerifyAll(signed, verifiers)
```

### Require All Actors via Middleware

```go
// Reject events missing any required actor's signature
bus.Use(signing.RequireMultiSigMiddleware("device", "server"))
```

### Middleware for Each Actor in the Pipeline

```go
// On the device's publish path
bus.UsePublish(signing.MultiSignMiddleware(deviceMulti))

// On the server's publish path
bus.UsePublish(signing.MultiSignMiddleware(serverMulti))
```
