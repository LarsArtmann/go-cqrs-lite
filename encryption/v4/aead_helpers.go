package encryption

import (
	"crypto/cipher"
	"crypto/rand"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"
)

// aeadEncrypt produces nonce||sealed using aead with the given nonce size.
// Shared by AES-256-GCM and XChaCha20-Poly1305. nonceErrCode is the stable
// errorfamily code for nonce-generation failures (e.g. "encryption.nonce_gen"
// or "encryption.xchacha20_nonce_gen").
func aeadEncrypt(
	aead cipher.AEAD,
	plaintext []byte,
	nonceSize int,
	nonceErrCode string,
) (Ciphertext, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}

	nonce := make([]byte, nonceSize)

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errorfamily.WrapInfrastructure(err, nonceErrCode, "generate nonce")
	}

	sealed := aead.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, 0, nonceSize+len(sealed))
	result = append(result, nonce...)
	result = append(result, sealed...)

	return Ciphertext(result), nil
}

// aeadDecrypt splits ciphertext into nonce||data and opens it via aead.
// Shared by AES-256-GCM and XChaCha20-Poly1305. shortErrCode is the stable
// code for "ciphertext too short" rejection (e.g. "encryption.ciphertext_too_short"
// or "encryption.xchacha20_ciphertext_too_short"); decryptErrCode is the stable
// code for the AEAD .Open failure (e.g. "encryption.decrypt").
func aeadDecrypt(
	aead cipher.AEAD,
	ciphertext Ciphertext,
	nonceSize int,
	shortErrCode, decryptErrCode string,
) ([]byte, error) {
	if ciphertext.IsZero() {
		return nil, nil
	}

	if len(ciphertext) < nonceSize+aead.Overhead() {
		return nil, errorfamily.Wrapf(
			ErrDecryptionFailed, errorfamily.Rejection,
			shortErrCode,
			"ciphertext length %d < minimum %d",
			len(ciphertext), nonceSize+aead.Overhead(),
		)
	}

	nonce := ciphertext[:nonceSize]
	data := ciphertext[nonceSize:]

	plaintext, err := aead.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, decryptErrCode, "decrypt ciphertext")
	}

	return plaintext, nil
}
