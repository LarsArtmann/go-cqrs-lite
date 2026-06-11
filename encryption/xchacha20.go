package encryption

import (
	"crypto/cipher"
	"crypto/rand"
	"io"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const xchacha20NonceSize = 24

type xchacha20 struct {
	aead cipher.AEAD
}

var _ EncrypterDecrypter = (*xchacha20)(nil)

func NewXChaCha20Poly1305(key []byte) (*xchacha20, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, event.Wrapf(
			ErrInvalidKey, event.Rejection,
			"encryption.xchacha20_key_wrong_size",
			"XChaCha20 key length %d != required %d",
			len(key), chacha20poly1305.KeySize,
		)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"encryption.xchacha20_init",
			"initialize XChaCha20-Poly1305",
		)
	}

	return &xchacha20{aead: aead}, nil
}

func (e *xchacha20) Algorithm() Algorithm { return XChaCha20Poly1305 }

func (e *xchacha20) Encrypt(plaintext []byte) (Ciphertext, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}

	nonce := make([]byte, xchacha20NonceSize)
	if _, err := io.ReadFull(
		rand.Reader,
		nonce,
	); err != nil { //nolint:noinlineerr // error used immediately in if-statement
		return nil, event.WrapInfrastructure(
			err,
			"encryption.xchacha20_nonce_gen",
			"generate nonce",
		)
	}

	sealed := e.aead.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, 0, xchacha20NonceSize+len(sealed))
	result = append(result, nonce...)
	result = append(result, sealed...)

	return Ciphertext(result), nil
}

func (e *xchacha20) Decrypt(ciphertext Ciphertext) ([]byte, error) {
	if ciphertext.IsZero() {
		return nil, nil
	}

	if len(ciphertext) < xchacha20NonceSize+e.aead.Overhead() {
		return nil, event.Wrapf(
			ErrDecryptionFailed, event.Rejection,
			"encryption.xchacha20_ciphertext_too_short",
			"ciphertext length %d < minimum %d",
			len(ciphertext), xchacha20NonceSize+e.aead.Overhead(),
		)
	}

	nonce := ciphertext[:xchacha20NonceSize]
	data := ciphertext[xchacha20NonceSize:]

	plaintext, err := e.aead.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"encryption.xchacha20_decrypt",
			"decrypt ciphertext",
		)
	}

	return plaintext, nil
}
