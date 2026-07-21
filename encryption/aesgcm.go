package encryption

import (
	"crypto/aes"
	"crypto/cipher"

	errorfamily "github.com/larsartmann/go-error-family"
)

const KeySize = 32

const aesGCMNonceSize = 12

type aes256gcm struct {
	aead cipher.AEAD
}

var _ EncrypterDecrypter = (*aes256gcm)(nil)

// NewAES256GCM creates an AES-256-GCM encrypter.
//
// AES-256-GCM uses 12-byte (96-bit) random nonces. Due to the birthday bound
// on the nonce space, a single key must encrypt at most ~2³² (~4 billion)
// messages before the probability of a nonce collision — which is catastrophic
// for GCM — becomes non-negligible. For high-volume use cases, prefer
// [NewXChaCha20Poly1305] (24-byte nonce, safe well beyond 2⁹⁶ messages).
func NewAES256GCM(key []byte) (*aes256gcm, error) {
	if len(key) != KeySize {
		return nil, errorfamily.Wrapf(
			ErrInvalidKey, errorfamily.Rejection,
			"encryption.aes_key_wrong_size",
			"AES-256 key length %d != required %d",
			len(key), KeySize,
		)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.aes_init",
			"initialize AES cipher",
		)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.gcm_init",
			"initialize GCM mode",
		)
	}

	return &aes256gcm{aead: aead}, nil
}

func (e *aes256gcm) Algorithm() Algorithm { return AES256GCM }

func (e *aes256gcm) Encrypt(plaintext []byte) (Ciphertext, error) {
	return aeadEncrypt(e.aead, plaintext, aesGCMNonceSize, "encryption.nonce_gen")
}

func (e *aes256gcm) Decrypt(ciphertext Ciphertext) ([]byte, error) {
	return aeadDecrypt(e.aead, ciphertext, aesGCMNonceSize,
		"encryption.ciphertext_too_short", "encryption.decrypt")
}
