package encryption

import (
	"crypto/cipher"

	errorfamily "github.com/larsartmann/go-error-family"
	"golang.org/x/crypto/chacha20poly1305"
)

const xchacha20NonceSize = 24

type xchacha20 struct {
	aead cipher.AEAD
}

var _ EncrypterDecrypter = (*xchacha20)(nil)

func NewXChaCha20Poly1305(key []byte) (*xchacha20, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, errorfamily.Wrapf(
			ErrInvalidKey, errorfamily.Rejection,
			"encryption.xchacha20_key_wrong_size",
			"XChaCha20 key length %d != required %d",
			len(key), chacha20poly1305.KeySize,
		)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"encryption.xchacha20_init",
			"initialize XChaCha20-Poly1305",
		)
	}

	return &xchacha20{aead: aead}, nil
}

func (e *xchacha20) Algorithm() Algorithm { return XChaCha20Poly1305 }

func (e *xchacha20) Encrypt(plaintext []byte) (Ciphertext, error) {
	return aeadEncrypt(e.aead, plaintext, xchacha20NonceSize, "encryption.xchacha20_nonce_gen")
}

func (e *xchacha20) Decrypt(ciphertext Ciphertext) ([]byte, error) {
	return aeadDecrypt(e.aead, ciphertext, xchacha20NonceSize,
		"encryption.xchacha20_ciphertext_too_short", "encryption.xchacha20_decrypt")
}
