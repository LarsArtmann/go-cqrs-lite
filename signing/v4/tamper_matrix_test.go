package signing_test

import (
	"crypto/ed25519"
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/signing/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4/internal/testutil"
)

// Crypto negative corpus (testing suite T24): a modified payload must be
// REJECTED deterministically, by every verifier, through a stable sentinel —
// and never by panicking.
//
// The sentinel is what consumers branch on (`errors.Is(err,
// signing.ErrInvalidSignature)`), so it is pinned here for the whole verifier
// family rather than per implementation.
//
// FAMILY TENSION, deliberately recorded instead of quietly "fixed":
// docs/error-taxonomy.md defines Corruption as "data integrity violation,
// checksum mismatch", which is exactly what a tampered event is — yet the
// shipped family is Rejection (signing/errors.go). Consumers that route by
// family therefore treat a tamper as ordinary invalid input, not as an
// integrity incident. Reclassifying a public error family changes consumer
// behaviour (errorfamily.Classify branches), so it is an owner decision rowed
// in TODO_LIST (T24); this test pins TODAY's family so the mismatch cannot
// drift unnoticed before that decision lands.
func TestTamperMatrix_RejectedWithStableSentinel(t *testing.T) {
	t.Parallel()

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	edSigner, _ := signing.NewEd25519(privKey)
	edVerifier, _ := signing.NewEd25519Verifier(pubKey)

	hmacKey := make([]byte, signing.MinimumKeyLength)
	hmacSigner, err := signing.NewHMAC(hmacKey)
	if err != nil {
		t.Fatalf("NewHMAC: %v", err)
	}

	verifiers := map[string]struct {
		signer   signing.Signer
		verifier signing.Verifier
	}{
		"ed25519": {signer: edSigner, verifier: edVerifier},
		// The HMAC signer is its own verifier (shared secret).
		"hmac": {signer: hmacSigner, verifier: hmacSigner},
	}

	for name, pair := range verifiers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			evt := testutil.MakeTestEvent(t)

			sig, err := pair.signer.Sign(evt)
			if err != nil {
				t.Fatalf("sign: %v", err)
			}

			if err := pair.verifier.Verify(evt, sig); err != nil {
				t.Fatalf("verify of the untampered event failed: %v", err)
			}

			tampered := testutil.TamperEvent(t, evt)

			err = pair.verifier.Verify(tampered, sig)
			if err == nil {
				t.Fatal("a tampered event verified successfully")
			}

			if !errors.Is(err, signing.ErrInvalidSignature) {
				t.Fatalf("tamper error = %v, want it to wrap signing.ErrInvalidSignature", err)
			}

			if family := errorfamily.Classify(err); family != errorfamily.Rejection {
				t.Fatalf(
					"tamper error family = %v, want Rejection (TODAY's pinned behaviour — see the "+
						"FAMILY TENSION note; a deliberate reclassification to Corruption must update this test)",
					family,
				)
			}
		})
	}
}

// TestTamperMatrix_WrongSignatureRejected pins the other half of the negative
// corpus: a well-formed signature from a DIFFERENT signer must not verify.
func TestTamperMatrix_WrongSignatureRejected(t *testing.T) {
	t.Parallel()

	pubKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	_, otherPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}

	otherSigner, _ := signing.NewEd25519(otherPriv)
	verifier, _ := signing.NewEd25519Verifier(pubKey)

	evt := testutil.MakeTestEvent(t)

	sig, err := otherSigner.Sign(evt)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	err = verifier.Verify(evt, sig)
	if err == nil {
		t.Fatal("a signature from another key verified")
	}

	if !errors.Is(err, signing.ErrInvalidSignature) {
		t.Fatalf("wrong-key error = %v, want signing.ErrInvalidSignature", err)
	}
}
