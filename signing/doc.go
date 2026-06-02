// Package signing provides event signing and verification for tamper-proof event streams.
//
// It supports HMAC-SHA256 (shared secret) and Ed25519 (public key) signing strategies.
// Consumers can:
//
//  1. Sign events before storage/transit using a Signer
//  2. Verify event signatures on read to detect tampering using a Verifier
//  3. Use PublishMiddleware to auto-sign events on the bus
//  4. Use VerifyMiddleware to auto-verify events before handling
//
// For multi-party signing (multiple actors signing the same event), see the
// [multisig] sub-package:
//
//	import "github.com/larsartmann/go-cqrs-lite/signing/v2/multisig"
//
// Example (single-signature):
//
//	signer, err := signing.NewHMAC("my-secret-key")
//	if err != nil { ... }
//
//	sig, err := signer.Sign(event)
//	if err != nil { ... }
//
//	err = signer.Verify(event, sig)
//	// err is non-nil if the event was tampered with
//
// Design principles:
//   - No external crypto dependencies beyond Go stdlib (crypto/hmac, crypto/ed25519)
//   - Signature bytes are URL-safe base64 encoded by default
//   - Sign and Verify are deterministic (same event + key = same signature)
//   - Failures are explicit errors, never panics
package signing
