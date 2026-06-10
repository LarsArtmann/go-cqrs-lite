// Package encryption provides event payload encryption and decryption for
// confidential event storage and transit.
//
// It supports AES-256-GCM (authenticated encryption with associated data).
// Consumers can:
//
//  1. Encrypt event payloads before storage/transit using an Encrypter
//  2. Decrypt event payloads on read to restore plaintext using a Decrypter
//  3. Use EncryptMiddleware to auto-encrypt events on the publish path
//  4. Use DecryptMiddleware to auto-decrypt events before handling
//
// Example:
//
//	enc, err := encryption.NewAES256GCM(key)
//	if err != nil { ... }
//
//	ct, err := enc.Encrypt([]byte(`{"ssn":"123-45-6789"}`))
//	if err != nil { ... }
//
//	plaintext, err := enc.Decrypt(ct)
//	// plaintext == original JSON
//
// Design principles:
//   - No external crypto dependencies beyond Go stdlib (crypto/aes, crypto/cipher)
//   - AES-256-GCM provides both confidentiality and integrity (authenticated encryption)
//   - Random 12-byte nonce per encryption, prepended to ciphertext
//   - Nonce + ciphertext + GCM tag stored as a single Ciphertext value
//   - Failures are explicit errors, never panics
package encryption
