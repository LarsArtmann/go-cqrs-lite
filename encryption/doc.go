// Package encryption provides event payload encryption and decryption for
// confidential event storage and transit.
//
// It supports two authenticated encryption algorithms:
//
//   - XChaCha20-Poly1305 (recommended): 24-byte nonce, constant-time in pure software,
//     effectively infinite nonce space. Import golang.org/x/crypto/chacha20poly1305.
//   - AES-256-GCM: stdlib-only, hardware-accelerated via AES-NI. 12-byte nonce.
//
// Both implement the Encrypter, Decrypter, and EncrypterDecrypter interfaces.
// Consumers can:
//
//  1. Encrypt event payloads before storage/transit using an Encrypter
//  2. Decrypt event payloads on read to restore plaintext using a Decrypter
//  3. Use EncryptMiddleware to auto-encrypt events on the publish path
//  4. Use DecryptMiddleware to auto-decrypt events before handling
//  5. Use NewCodec to compose encryption with any codec.Codec (e.g., JSON)
//
// # Algorithm Selection
//
// Prefer XChaCha20-Poly1305 for new projects. It provides constant-time guarantees
// regardless of hardware, and its 24-byte nonce eliminates key rotation pressure
// (birthday bound at ~2^96 vs ~2^48 for AES-GCM).
//
// Use AES-256-GCM when stdlib-only is required or when AES-NI hardware acceleration
// is guaranteed.
//
// # Algorithm Identification
//
// EncryptMiddleware automatically detects the algorithm from the encrypter and
// stores it in event metadata (event.encryption.algorithm). Extract it with
// ExtractAlgorithm. Both implementations (AES-256-GCM, XChaCha20-Poly1305) report
// their algorithm. DecryptMiddleware cleans up algorithm metadata on decryption.
//
// # Key ID Support
//
// For key rotation scenarios, pass WithMiddlewareKeyID to EncryptMiddleware.
// The key ID is stored in event metadata (event.encryption.key-id) and can be
// retrieved with ExtractKeyID. This enables consumers to select the correct
// decryption key based on the key ID embedded in each event.
//
//	bus.UsePublish(encryption.EncryptMiddleware(enc, encryption.WithMiddlewareKeyID("key-v2")))
//
// # Composable Codec Wrapper
//
// encryption.NewCodec wraps any codec.Codec to add transparent encrypt/decrypt:
//
//	enc, _ := encryption.NewXChaCha20Poly1305(key)
//	c := encryption.NewCodec(codec.JSONCodec{}, enc)
//	evt, _ := event.NewEvent("user.created", aggID, "User", 1, payload, event.WithCodec(c))
//
// # Example (direct encrypt/decrypt)
//
//	enc, _ := encryption.NewXChaCha20Poly1305(key)
//	ct, _ := enc.Encrypt([]byte(`{"ssn":"123-45-6789"}`))
//	plaintext, _ := enc.Decrypt(ct)
//
// # Example (middleware)
//
//	bus.UsePublish(encryption.EncryptMiddleware(enc))
//	bus.Use(encryption.DecryptMiddleware(enc))
//
// # Key Derivation (HKDF)
//
// For multi-tenant systems, DeriveKey uses HKDF-SHA256 (RFC 5869) to derive
// per-tenant encryption keys from a single master key. The info parameter
// provides domain separation:
//
//	masterKey := []byte("32-byte-master-key-from-kms")
//	tenantKey, _ := encryption.DeriveKey(masterKey, "tenant:acme-corp", 32)
//	enc, _ := encryption.NewXChaCha20Poly1305(tenantKey)
//
// DeriveKey is deterministic: the same masterKey + info always produce the same
// derived key. Different info values produce independent keys.
//
// Design principles:
//   - Two algorithms behind the same Encrypter/Decrypter interface
//   - No external crypto dependencies beyond Go stdlib + golang.org/x/crypto
//   - AES-256-GCM provides hardware acceleration; XChaCha20 provides constant-time safety
//   - Random nonce per encryption, prepended to ciphertext
//   - Algorithm and key ID metadata for key rotation and audit
//   - Failures are explicit errors, never panics
package encryption
