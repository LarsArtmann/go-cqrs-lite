# encryption — Event Payload Encryption

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/encryption.svg)](https://pkg.go.dev/github.com/LarsArtmann/go-cqrs-lite/encryption)

AES-256-GCM event payload encryption for confidential event storage and transit. Provides authenticated encryption (confidentiality + integrity) using only Go stdlib.

```bash
go get github.com/larsartmann/go-cqrs-lite/encryption
```

## Why Encrypt Events?

Event stores and message brokers often reside in shared infrastructure. Encrypting event payloads provides:

- **Data-at-rest protection**: Stored events are ciphertext — database compromises leak no sensitive fields
- **Data-in-transit protection**: Events crossing trust boundaries (VPCs, message brokers) are protected
- **Compliance**: Meets GDPR, HIPAA, PCI-DSS encryption-at-rest requirements
- **Defense in depth**: Even if signing is bypassed, encrypted payloads are unreadable without the key

## Quick Start

### Encrypt and Decrypt

```go
enc, err := encryption.NewAES256GCM(key) // key must be 32 bytes
if err != nil { ... }

ct, err := enc.Encrypt([]byte(`{"ssn":"123-45-6789"}`))
if err != nil { ... }

plaintext, err := enc.Decrypt(ct)
// plaintext == original JSON
```

### Auto-Encrypt on Publish

```go
enc, _ := encryption.NewAES256GCM(key)
bus.UsePublish(encryption.EncryptMiddleware(enc))

// Every published event's payload is automatically encrypted
```

### Auto-Decrypt on Handle

```go
enc, _ := encryption.NewAES256GCM(key)
bus.Use(encryption.DecryptMiddleware(enc))

// Encrypted events are decrypted before reaching handlers
// Unencrypted events pass through (supports mixed streams)
```

### Full Pipeline (Sign + Encrypt)

```go
signer, _ := signing.NewHMAC(signingKey)
enc, _ := encryption.NewAES256GCM(encryptionKey)

bus.UsePublish(signing.SignMiddleware(signer))
bus.UsePublish(encryption.EncryptMiddleware(enc))
bus.Use(encryption.DecryptMiddleware(enc))
bus.Use(signing.VerifyMiddleware(signer))

// Publish: sign → encrypt → store
// Subscribe: decrypt → verify → handle
```

## Design

- **No external crypto dependencies**: Uses Go stdlib (`crypto/aes`, `crypto/cipher`)
- **AES-256-GCM**: Authenticated encryption with associated data (confidentiality + integrity)
- **Random nonce per encryption**: 12-byte random nonce prepended to ciphertext
- **Ciphertext in metadata**: Base64-encoded ciphertext stored in event custom metadata (`event.encrypted`)
- **Constant-time comparison**: `Ciphertext.Equal` uses `crypto/subtle.ConstantTimeCompare`
- **Composable middleware**: EncryptMiddleware and DecryptMiddleware integrate with the event bus

## Performance

Benchmarks on 1KB payloads (AMD Ryzen):

```
AES256GCM_Encrypt     ~700ns/op   3 allocs/op
AES256GCM_Decrypt     ~500ns/op   2 allocs/op
AES256GCM_RoundTrip  ~1100ns/op   5 allocs/op
```

## Security Considerations

- **Key management**: This module handles encryption, not key management. Use your cloud provider's KMS (AWS KMS, GCP KMS, Azure Key Vault) or HashiCorp Vault for key storage and rotation.
- **Nonce space**: AES-GCM with random 12-byte nonces has a birthday bound of ~2^48 encryptions per key. Rotate keys well before this limit.
- **Key rotation**: Design your system to support key rotation. Store the key ID alongside encrypted events for multi-key support.
