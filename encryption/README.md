# encryption — Event Payload Encryption

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/encryption/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/encryption/v4)

Authenticated event payload encryption for confidential event storage and transit. Two algorithms behind a single interface.

```bash
go get github.com/larsartmann/go-cqrs-lite/encryption/v4
```

## Why Encrypt Events?

Event stores and message brokers often reside in shared infrastructure. Encrypting event payloads provides:

- **Data-at-rest protection**: Stored events are ciphertext — database compromises leak no sensitive fields
- **Data-in-transit protection**: Events crossing trust boundaries (VPCs, message brokers) are protected
- **Compliance**: Meets GDPR, HIPAA, PCI-DSS encryption-at-rest requirements
- **Defense in depth**: Even if signing is bypassed, encrypted payloads are unreadable without the key

## Algorithm Selection

|                    | XChaCha20-Poly1305          | AES-256-GCM               |
| ------------------ | --------------------------- | ------------------------- |
| **Recommendation** | **Default choice**          | Legacy / stdlib-only      |
| Nonce size         | 24 bytes                    | 12 bytes                  |
| Birthday bound     | ~2^96 per key               | ~2^48 per key             |
| Constant-time      | Always (pure software)      | Only with AES-NI hardware |
| Dependency         | `golang.org/x/crypto`       | Go stdlib                 |
| Used by            | WireGuard, Age, MinIO, NaCl | TLS 1.3, AWS KMS          |

## Quick Start

### XChaCha20-Poly1305 (recommended)

```go
enc, err := encryption.NewXChaCha20Poly1305(key) // key must be 32 bytes
if err != nil { ... }

ct, err := enc.Encrypt([]byte(`{"ssn":"123-45-6789"}`))
if err != nil { ... }

plaintext, err := enc.Decrypt(ct)
// plaintext == original JSON
```

### AES-256-GCM (stdlib-only)

```go
enc, err := encryption.NewAES256GCM(key) // key must be 32 bytes
```

### Auto-Encrypt on Publish

```go
enc, _ := encryption.NewXChaCha20Poly1305(key)

// With algorithm identification + key ID for key rotation:
bus.UsePublish(encryption.EncryptMiddleware(enc, encryption.WithMiddlewareKeyID("key-v2")))

// Every published event's payload is automatically encrypted
// Algorithm ("xchacha20-poly1305") and key ID ("key-v2") are stored in event metadata
```

### Auto-Decrypt on Handle

```go
enc, _ := encryption.NewXChaCha20Poly1305(key)
bus.Use(encryption.DecryptMiddleware(enc))

// Encrypted events are decrypted before reaching handlers
// Unencrypted events pass through (supports mixed streams)
```

### Composable Codec Wrapper

```go
enc, _ := encryption.NewXChaCha20Poly1305(key)
c := encryption.NewCodec(codec.JSONCodec{}, enc)

// Use with event.New — encrypts during Encode, decrypts during Decode
evt, _ := event.New("user.created", aggID, "User", 1, payload, event.WithCodec(c))
```

### Full Pipeline (Sign + Encrypt)

```go
signer, _ := signing.NewHMAC(signingKey)
enc, _ := encryption.NewXChaCha20Poly1305(encryptionKey)

bus.UsePublish(signing.SignMiddleware(signer))
bus.UsePublish(encryption.EncryptMiddleware(enc))
bus.Use(encryption.DecryptMiddleware(enc))
bus.Use(signing.VerifyMiddleware(signer))

// Publish: sign → encrypt → store
// Subscribe: decrypt → verify → handle
```

## Design

- **Two algorithms, one interface**: `Encrypter`, `Decrypter`, `EncrypterDecrypter`
- **Composable, not combinable**: codec/ stays pure (Layer 0, zero deps). Encryption wraps any codec via `NewCodec`
- **Random nonce per encryption**: Prepended to ciphertext
- **Ciphertext in metadata**: Base64-encoded ciphertext stored in event custom metadata (`event.encrypted`)
- **Algorithm identification**: EncryptMiddleware auto-detects algorithm and stores it in `event.encryption.algorithm` metadata
- **Key rotation support**: Optional key ID stored in `event.encryption.key-id` metadata for multi-key scenarios
- **Constant-time comparison**: `Ciphertext.Equal` uses `crypto/subtle.ConstantTimeCompare`
- **Composable middleware**: EncryptMiddleware and DecryptMiddleware integrate with the event bus

## Algorithm Identification

The middleware automatically detects which algorithm produced a given ciphertext:

```go
alg, err := encryption.ExtractAlgorithm(evt)
// alg == encryption.AES256GCM or encryption.XChaCha20Poly1305
```

Both implementations report their algorithm via an `Algorithm()` method. The middleware stores it in event metadata so consumers can identify which algorithm was used without trying both.

## Key Rotation

For multi-key scenarios (e.g., rotating encryption keys), use `WithMiddlewareKeyID`:

```go
// Publisher: encrypt with key-v2
bus.UsePublish(encryption.EncryptMiddleware(encV2, encryption.WithMiddlewareKeyID("key-v2")))

// Consumer: select decrypter based on key ID from event metadata
keyID, _ := encryption.ExtractKeyID(evt)
decrypter := keyLookup[keyID] // consumer provides this map
plaintext, _ := decrypter.Decrypt(ct)
```

The key ID is stored alongside the ciphertext in event metadata. DecryptMiddleware removes all encryption metadata (ciphertext, algorithm, key ID) after decryption.

## Key Management Helpers

The module ships focused helpers for the three common key lifecycles. For
master-key custody prefer your cloud KMS or Vault — these helpers cover
generation, derivation, and rotation selection, not storage of secrets.

### Generation

```go
key, err := encryption.GenerateKey()          // 32 random bytes (crypto/rand)
keyB64, err := encryption.GenerateKeyBase64() // same, base64-encoded for env vars/secrets managers
```

### Derivation (multi-tenant per-tenant keys)

`DeriveKey` derives domain-separated subkeys from one master key via
HKDF-SHA256 — each tenant gets a unique key without storing per-tenant
material:

```go
tenantKey, err := encryption.DeriveKey(masterKey, "tenant:acme-corp", 32)
enc, err := encryption.NewXChaCha20Poly1305(tenantKey)
```

The `info` string is domain separation: the same master key with different
`info` values yields unrelated keys.

### Rotation (key ID → decrypter selection)

`StaticKeyResolver` implements `KeyResolver` for the startup-known key set —
pair it with `ExtractKeyID` on the consumer side:

```go
resolver := encryption.NewStaticKeyResolver(map[encryption.KeyID]encryption.Decrypter{
    "key-v1": decV1, // old key: decrypt-only, for pre-rotation rows
    "key-v2": decV2, // current key
})
```

The resolver copies the map and reports the available key IDs in its
not-found error, so a missing key is diagnosable from logs alone.

### Self-Describing Ciphertext (binary envelope)

`WrapCiphertext`/`UnwrapCiphertext` prefix raw ciphertext with version and
algorithm bytes (`[version:1][algorithm:1][ciphertext:N]`) so a stored blob
decrypts without external metadata:

```go
wrapped, err := encryption.WrapCiphertext(rawCiphertext, encryption.XChaCha20Poly1305)
alg, raw, err := encryption.UnwrapCiphertext(wrapped) // alg == encryption.XChaCha20Poly1305
```

## Envelope Wire Formats (v1 ↔ v2)

`Envelope` + `MarshalEnvelope`/`UnmarshalEnvelope` wrap ciphertext with
versioning metadata as JSON for SQL columns:

```json
{"v":"v2","ct":"3q2-7wBC","alg":"xchacha20-poly1305","kid":"key-v1"}
```

| Field      | Key  | Notes                                              |
| ---------- | ---- | -------------------------------------------------- |
| Version    | `v`  | `"v1"` or `"v2"`; empty on write defaults to `v2`  |
| Ciphertext | `ct` | URL-safe base64 (nonce is prepended inside)        |
| Algorithm  | `alg`| omitted when empty                                 |
| KeyID      | `kid`| omitted when empty                                 |

- **v2 (current, always written)**: the JSON object itself. PostgreSQL and
  MySQL snapshot-state columns are JSON/JSONB and reject the v1 form with
  "invalid input syntax for type json" — v2 stores everywhere, including
  opaque KV engines.
- **v1 (legacy, still readable)**: the same JSON base64url-wrapped as one
  opaque string. `UnmarshalEnvelope` auto-detects the shape (`{` prefix →
  raw JSON; otherwise base64-decode), so readers need no version branch.

The wire format is pinned by reviewed goldens
(`envelope_wire_golden_test.go`) and a v1↔v2 decode-symmetry property test
(`envelope_symmetry_test.go`): whichever generation wrote a row, the current
reader recovers the identical envelope. Any format change must ship as a new
envelope version, never an edit of v2 bytes.

## Performance

Benchmarks on 1KB payloads (AMD Ryzen):

```
AES256GCM_Encrypt              ~700ns/op    3 allocs/op
AES256GCM_Decrypt              ~500ns/op    2 allocs/op
XChaCha20Poly1305_Encrypt      ~800ns/op    3 allocs/op
XChaCha20Poly1305_Decrypt      ~600ns/op    2 allocs/op
```

## Security Considerations

- **Key management**: Master-key custody belongs in your cloud provider's KMS or HashiCorp Vault; see [Key Management Helpers](#key-management-helpers) for the in-module generation/derivation/rotation surfaces.
- **AES-GCM nonce space**: 12-byte random nonces have a birthday bound at ~2^48 per key. Rotate keys well before this limit. XChaCha20's 24-byte nonce eliminates this concern.
- **Key rotation**: Design your system for key rotation. Store the key ID alongside encrypted events.

## Related Modules

- [**signing**](../signing/README.md) — Sign then encrypt for tamper-proof + confidential streams
- [**go-codec**](https://github.com/larsartmann/go-codec) — `encryption.NewCodec` wraps a codec with transparent encryption
- [**event**](../event/README.md) — Encrypts event payloads via `bus.UsePublish` / `bus.Use`
- [**middleware**](../middleware/README.md) — `EncryptMiddleware` / `DecryptMiddleware` re-exported here
