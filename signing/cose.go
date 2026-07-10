package signing

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
)

// COSESigner signs raw byte strings and reports the COSE algorithm identifier.
// It is used by SignCOSE1 to build RFC 9052 COSE_Sign1 messages.
type COSESigner interface {
	Sign(data []byte) (Signature, error)
	COSEAlgorithm() int64
}

// COSEVerifier verifies raw byte string signatures and reports the COSE algorithm
// identifier. It is used by VerifyCOSE1 to validate RFC 9052 COSE_Sign1 messages.
type COSEVerifier interface {
	Verify(data []byte, sig Signature) error
	COSEAlgorithm() int64
}

// COSESignOption configures the COSE_Sign1 signing process.
type COSESignOption func(*coseSignConfig)

type coseSignConfig struct {
	externalAAD []byte
	kid         []byte
}

// WithCOSEExternalAAD provides additional authenticated data that is included in
// the Sig_structure but not carried inside the COSE_Sign1 message. Both signer
// and verifier must supply the same external AAD.
func WithCOSEExternalAAD(aad []byte) COSESignOption {
	return func(c *coseSignConfig) { c.externalAAD = aad }
}

// WithCOSEKeyID stores the key identifier in the unprotected header of the
// COSE_Sign1 message. This is a hint for the verifier to select the right key.
func WithCOSEKeyID(kid []byte) COSESignOption {
	return func(c *coseSignConfig) { c.kid = kid }
}

// coseHMAC implements COSESigner and COSEVerifier using HMAC-SHA256.
// COSE algorithm identifier 5.
type coseHMAC struct {
	key []byte
}

var (
	_ COSESigner   = (*coseHMAC)(nil)
	_ COSEVerifier = (*coseHMAC)(nil)
)

// NewCOSEHMAC creates a COSE-aware HMAC-SHA256 signer/verifier.
// It shares the same key-length requirements as NewHMAC.
func NewCOSEHMAC(key []byte) (*coseHMAC, error) {
	if len(key) < MinimumKeyLength {
		return nil, errorfamily.Wrapf(
			ErrInvalidKey, errorfamily.Rejection,
			"signing.cose_hmac_key_too_short",
			"HMAC key length %d < minimum %d",
			len(key),
			MinimumKeyLength,
		)
	}

	return &coseHMAC{key: slices.Clone(key)}, nil
}

// Sign computes an HMAC-SHA256 signature over data.
func (s *coseHMAC) Sign(data []byte) (Signature, error) {
	mac := hmac.New(sha256.New, s.key)
	mac.Write(data)

	return Signature(mac.Sum(nil)), nil
}

// Verify checks an HMAC-SHA256 signature over data.
func (s *coseHMAC) Verify(data []byte, sig Signature) error {
	if sig.IsZero() {
		return ErrNilSignature
	}

	expected, err := s.Sign(data)
	if err != nil {
		return err
	}

	if !expected.Equal(sig) {
		return ErrInvalidSignature
	}

	return nil
}

// COSEAlgorithm returns the COSE algorithm identifier for HMAC-SHA256 (5).
func (s *coseHMAC) COSEAlgorithm() int64 { return codec.COSEAlgHMACSHA256 }

// coseEd25519Signer implements COSESigner using Ed25519.
// COSE algorithm identifier -19.
type coseEd25519Signer struct {
	privateKey ed25519.PrivateKey
}

var _ COSESigner = (*coseEd25519Signer)(nil)

// NewCOSEEd25519Signer creates a COSE-aware Ed25519 signer.
func NewCOSEEd25519Signer(privateKey ed25519.PrivateKey) (*coseEd25519Signer, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errorfamily.Wrapf(
			ErrInvalidKey, errorfamily.Rejection,
			"signing.cose_ed25519_invalid_private_key",
			"expected Ed25519 private key of %d bytes, got %d",
			ed25519.PrivateKeySize,
			len(privateKey),
		)
	}

	return &coseEd25519Signer{privateKey: slices.Clone(privateKey)}, nil
}

// Sign computes an Ed25519 signature over data.
func (s *coseEd25519Signer) Sign(data []byte) (Signature, error) {
	return Signature(ed25519.Sign(s.privateKey, data)), nil
}

// COSEAlgorithm returns the COSE algorithm identifier for Ed25519 (-19).
func (s *coseEd25519Signer) COSEAlgorithm() int64 { return codec.COSEAlgEd25519 }

// coseEd25519Verifier implements COSEVerifier using Ed25519.
type coseEd25519Verifier struct {
	publicKey ed25519.PublicKey
}

var _ COSEVerifier = (*coseEd25519Verifier)(nil)

// NewCOSEEd25519Verifier creates a COSE-aware Ed25519 verifier.
func NewCOSEEd25519Verifier(publicKey ed25519.PublicKey) (*coseEd25519Verifier, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, errorfamily.Wrapf(
			ErrInvalidKey, errorfamily.Rejection,
			"signing.cose_ed25519_invalid_public_key",
			"expected Ed25519 public key of %d bytes, got %d",
			ed25519.PublicKeySize,
			len(publicKey),
		)
	}

	return &coseEd25519Verifier{publicKey: slices.Clone(publicKey)}, nil
}

// Verify checks an Ed25519 signature over data.
func (v *coseEd25519Verifier) Verify(data []byte, sig Signature) error {
	if sig.IsZero() {
		return ErrNilSignature
	}

	if !ed25519.Verify(v.publicKey, data, sig) {
		return ErrInvalidSignature
	}

	return nil
}

// COSEAlgorithm returns the COSE algorithm identifier for Ed25519 (-19).
func (v *coseEd25519Verifier) COSEAlgorithm() int64 { return codec.COSEAlgEd25519 }

// SignCOSE1 and VerifyCOSE1 live in cose_sign1.go.
