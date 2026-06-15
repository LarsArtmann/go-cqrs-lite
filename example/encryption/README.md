# example/encryption — Event Encryption Patterns

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/example/encryption.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/example/encryption)

Runnable example demonstrating three event-encryption patterns using the [`encryption`](../../encryption/README.md) module: bus-level middleware, store-level decorator, and key rotation.

## Run

```bash
cd example/encryption && go run .
```

## What It Demonstrates

### 1. Bus-Level Encryption

Encrypt events on publish, decrypt on receive — transparent to handlers:

```go
bus.UsePublish(encryption.EncryptMiddleware(enc, encryption.WithMiddlewareKeyID("key-v1")))
bus.Use(encryption.DecryptMiddleware(enc))
```

### 2. Store-Level Encryption

Wrap any event store with transparent encryption via `NewEncryptedStore`:

```go
encryptedStore := encryption.NewEncryptedStore(store, enc)
// encryptedStore implements Store, Journal, SeekableJournal, BackwardsSource
```

### 3. Key Rotation

Rotate keys without downtime using `StaticKeyResolver`:

```go
resolver := encryption.NewStaticKeyResolver(map[encryption.KeyID]encryption.Decrypter{
    "key-v1": oldDecrypter,
    "key-v2": newDecrypter,
})
```

## Dependencies

| Dependency                                  | Purpose                                 |
| ------------------------------------------- | --------------------------------------- |
| [encryption/v2](../../encryption/README.md) | AEAD ciphers, middleware, codec wrapper |
| [event/v2](../../event/README.md)           | Event model and bus                     |
| [memory/v2](../../memory/README.md)         | In-memory store and bus for the demo    |

## Related Modules

- [encryption/v2](../../encryption/README.md) — The library module this example demonstrates
- [example/todo](../todo/README.md) — Full application example
