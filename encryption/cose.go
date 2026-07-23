package encryption

import (
	"crypto/rand"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

// COSEEncrypter encrypts plaintext with authenticated additional data and reports
// the COSE algorithm identifier. It is used by EncryptCOSE0 to build RFC 9052
// COSE_Encrypt0 messages.
type COSEEncrypter interface {
	Encrypt(plaintext []byte, aad []byte) (ciphertext []byte, nonce []byte, err error)
	COSEAlgorithm() int64
}

// COSEDecrypter decrypts ciphertext with authenticated additional data and
// reports the COSE algorithm identifier. It is used by DecryptCOSE0 to validate
// RFC 9052 COSE_Encrypt0 messages.
type COSEDecrypter interface {
	Decrypt(ciphertext []byte, nonce []byte, aad []byte) ([]byte, error)
	COSEAlgorithm() int64
}

// COSEEncryptOption configures the COSE_Encrypt0 encryption/decryption process.
type COSEEncryptOption func(*coseEncryptConfig)

type coseEncryptConfig struct {
	externalAAD []byte
}

// WithCOSEEncryptExternalAAD provides additional authenticated data that is
// included in the Enc_structure but not carried inside the COSE_Encrypt0 message.
// Both encryptor and decryptor must supply the same external AAD.
func WithCOSEEncryptExternalAAD(aad []byte) COSEEncryptOption {
	return func(c *coseEncryptConfig) { c.externalAAD = aad }
}

// coseXChaCha20 implements COSEEncrypter and COSEDecrypter using XChaCha20-Poly1305.
// COSE algorithm identifier 24.
type coseXChaCha20 struct {
	x *xchacha20
}

var (
	_ COSEEncrypter = (*coseXChaCha20)(nil)
	_ COSEDecrypter = (*coseXChaCha20)(nil)
)

// NewCOSEXChaCha20Poly1305 creates a COSE-aware XChaCha20-Poly1305 encrypter/decrypter.
func NewCOSEXChaCha20Poly1305(key []byte) (*coseXChaCha20, error) {
	x, err := NewXChaCha20Poly1305(key)
	if err != nil {
		return nil, err
	}

	return &coseXChaCha20{x: x}, nil
}

// Encrypt returns the ciphertext and nonce for the given plaintext and AAD.
func (e *coseXChaCha20) Encrypt(plaintext, aad []byte) ([]byte, []byte, error) {
	nonce := make([]byte, xchacha20NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.cose_xchacha20_nonce_gen",
			"generate nonce",
		)
	}

	ciphertext := e.x.aead.Seal(nil, nonce, plaintext, aad)

	return ciphertext, nonce, nil
}

// Decrypt returns the plaintext for the given ciphertext, nonce, and AAD.
func (e *coseXChaCha20) Decrypt(ciphertext, nonce, aad []byte) ([]byte, error) {
	plaintext, err := e.x.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.cose_xchacha20_decrypt",
			"decrypt XChaCha20-Poly1305",
		)
	}

	return plaintext, nil
}

// COSEAlgorithm returns the COSE algorithm identifier for XChaCha20-Poly1305 (24).
func (e *coseXChaCha20) COSEAlgorithm() int64 { return codec.COSEAlgChaCha20Poly1305 }

// coseAESGCM implements COSEEncrypter and COSEDecrypter using AES-256-GCM.
// COSE algorithm identifier 3.
type coseAESGCM struct {
	e *aes256gcm
}

var (
	_ COSEEncrypter = (*coseAESGCM)(nil)
	_ COSEDecrypter = (*coseAESGCM)(nil)
)

// NewCOSEAES256GCM creates a COSE-aware AES-256-GCM encrypter/decrypter.
func NewCOSEAES256GCM(key []byte) (*coseAESGCM, error) {
	e, err := NewAES256GCM(key)
	if err != nil {
		return nil, err
	}

	return &coseAESGCM{e: e}, nil
}

// Encrypt returns the ciphertext and nonce for the given plaintext and AAD.
func (e *coseAESGCM) Encrypt(plaintext, aad []byte) ([]byte, []byte, error) {
	nonce := make([]byte, aesGCMNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.cose_aes_nonce_gen",
			"generate nonce",
		)
	}

	ciphertext := e.e.aead.Seal(nil, nonce, plaintext, aad)

	return ciphertext, nonce, nil
}

// Decrypt returns the plaintext for the given ciphertext, nonce, and AAD.
func (e *coseAESGCM) Decrypt(ciphertext, nonce, aad []byte) ([]byte, error) {
	plaintext, err := e.e.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.cose_aes_decrypt",
			"decrypt AES-256-GCM",
		)
	}

	return plaintext, nil
}

// COSEAlgorithm returns the COSE algorithm identifier for AES-256-GCM (3).
func (e *coseAESGCM) COSEAlgorithm() int64 { return codec.COSEAlgAES256GCM }

// EncryptCOSE0 creates a COSE_Encrypt0 message for the given plaintext.
//
// The protected header contains the algorithm identifier (alg). The unprotected
// header contains the initialization vector (IV). The ciphertext is produced by
// the encrypter with the Enc_structure as AAD.
func EncryptCOSE0(plaintext []byte, enc COSEEncrypter, opts ...COSEEncryptOption) ([]byte, error) {
	if enc == nil {
		return nil, ErrNilEncrypter
	}

	cfg := coseEncryptConfig{} //nolint:exhaustruct // zero values are ready

	protected, err := codec.PrepareCOSESetup(&cfg, opts, enc.COSEAlgorithm())
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.cose_setup",
			"prepare COSE setup",
		)
	}

	externalAAD := cfg.externalAAD
	if externalAAD == nil {
		externalAAD = []byte{}
	}

	aad, err := codec.EncStructure0(protected, externalAAD)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.cose_build_enc_structure",
			"build COSE Enc_structure",
		)
	}

	ciphertext, nonce, err := enc.Encrypt(plaintext, aad)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.cose_encrypt",
			"encrypt COSE_Encrypt0 content",
		)
	}

	msg := codec.COSEEncrypt0{
		Protected: protected,
		Unprotected: map[int64]any{
			codec.COSEHeaderIV: nonce,
		},
		Ciphertext: ciphertext,
	}

	coseBytes, err := codec.MarshalCOSEEncrypt0(msg)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.cose_marshal",
			"marshal COSE_Encrypt0",
		)
	}

	return coseBytes, nil
}

// DecryptCOSE0 decrypts a COSE_Encrypt0 message.
//
// It verifies that the algorithm identifier in the protected header matches the
// decrypter, extracts the IV from the unprotected header, reconstructs the
// Enc_structure as AAD, and decrypts the ciphertext.
func DecryptCOSE0(coseBytes []byte, dec COSEDecrypter, opts ...COSEEncryptOption) ([]byte, error) {
	if dec == nil {
		return nil, ErrNilDecrypter
	}

	cfg := coseEncryptConfig{} //nolint:exhaustruct // zero values are ready
	for _, o := range opts {
		o(&cfg)
	}

	msg, err := codec.UnmarshalCOSEEncrypt0(coseBytes)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.cose_unmarshal",
			"unmarshal COSE_Encrypt0",
		)
	}

	protected, err := codec.UnmarshalCOSEProtectedHeader(msg.Protected)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.cose_unmarshal_protected",
			"unmarshal COSE protected header",
		)
	}

	alg, err := codec.NormalizeCOSEAlgorithm(protected[codec.COSEHeaderAlg])
	if err != nil {
		return nil, errorfamily.Wrapf(
			ErrUnknownAlgorithm, errorfamily.Rejection,
			"encryption.cose_invalid_algorithm",
			"COSE protected header has invalid algorithm: %v",
			protected[codec.COSEHeaderAlg],
		)
	}

	if alg != dec.COSEAlgorithm() {
		return nil, errorfamily.Wrapf(
			ErrCOSEAlgorithmMismatch, errorfamily.Rejection,
			"encryption.cose_algorithm_mismatch",
			"COSE algorithm %d does not match decrypter algorithm %d",
			alg, dec.COSEAlgorithm(),
		)
	}

	nonce, err := extractCOSEIV(msg.Unprotected)
	if err != nil {
		return nil, err
	}

	externalAAD := cfg.externalAAD
	if externalAAD == nil {
		externalAAD = []byte{}
	}

	aad, err := codec.EncStructure0(msg.Protected, externalAAD)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.cose_rebuild_enc_structure",
			"rebuild COSE Enc_structure",
		)
	}

	plaintext, err := dec.Decrypt(msg.Ciphertext, nonce, aad)
	if err != nil {
		return nil, errorfamily.Wrapf(
			err, errorfamily.Corruption,
			"encryption.cose_decrypt",
			"decrypt COSE_Encrypt0",
		)
	}

	return plaintext, nil
}

// extractCOSEIV extracts the initialization vector from the unprotected header.
func extractCOSEIV(unprotected map[int64]any) ([]byte, error) {
	if unprotected == nil {
		return nil, errorfamily.NewRejection(
			"encryption.cose_missing_iv",
			"COSE unprotected header is missing IV",
		)
	}

	ivValue, ok := unprotected[codec.COSEHeaderIV]
	if !ok {
		return nil, errorfamily.NewRejection(
			"encryption.cose_missing_iv",
			"COSE unprotected header is missing IV",
		)
	}

	ivBytes, ok := ivValue.([]byte)
	if !ok {
		return nil, errorfamily.NewRejection(
			"encryption.cose_invalid_iv",
			"COSE IV is not a byte string",
		)
	}

	return ivBytes, nil
}
