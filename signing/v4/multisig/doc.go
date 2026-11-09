// Package multisig provides multi-party event signing and verification.
//
// When events travel through multiple actors (e.g., end-user device → server → gateway),
// each actor adds its own signature. The event accumulates a [MultiSignature] collection
// that any participant can verify.
//
// This sub-package builds on the root signing package's [signing.Signer], [signing.Verifier],
// and [signing.Signature] types. Import as:
//
//	import "github.com/larsartmann/go-cqrs-lite/signing/v4/multisig"
//
// Each actor creates a [MultiSigner] with their own cryptographic key:
//
//	deviceMulti, _ := multisig.NewMultiSigner(
//	    multisig.Actor("device"),
//	    multisig.AlgorithmEd25519,
//	    ed25519Signer,
//	    multisig.WithVerifier(ed25519Verifier),
//	)
//	serverMulti, _ := multisig.NewMultiSigner(
//	    multisig.Actor("server"),
//	    multisig.AlgorithmHMACSHA256,
//	    hmacSigner,
//	)
//
// Sign events in sequence, then verify all signatures at once:
//
//	verifiers := multisig.VerifierMap(deviceMulti, serverMulti)
//	err := multisig.VerifyAll(signed, verifiers)
//
// For automatic signing and verification via event bus middleware, see
// [MultiSignMiddleware], [MultiVerifyMiddleware], and [RequireMultiSigMiddleware].
package multisig
