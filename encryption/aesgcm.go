package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const KeySize = 32

const aesGCMNonceSize = 12

type aes256gcm struct {
	aead cipher.AEAD
}

var _ EncrypterDecrypter = (*aes256gcm)(nil)

func NewAES256GCM(key []byte) (*aes256gcm, error) {
	if len(key) != KeySize {
		return nil, event.Wrapf(
			ErrInvalidKey, event.Rejection,
			"encryption.aes_key_wrong_size",
			"AES-256 key length %d != required %d",
			len(key), KeySize,
		)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "encryption.aes_init", "initialize AES cipher")
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "encryption.gcm_init", "initialize GCM mode")
	}

	return &aes256gcm{aead: aead}, nil
}

func (e *aes256gcm) Algorithm() Algorithm { return AES256GCM }

func (e *aes256gcm) Encrypt(plaintext []byte) (Ciphertext, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}

	nonce := make([]byte, aesGCMNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil { //nolint:noinlineerr // error used immediately in if-statement
		return nil, event.WrapInfrastructure(err, "encryption.nonce_gen", "generate nonce")
	}

	sealed := e.aead.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, 0, aesGCMNonceSize+len(sealed))
	result = append(result, nonce...)
	result = append(result, sealed...)

	return Ciphertext(result), nil
}

func (e *aes256gcm) Decrypt(ciphertext Ciphertext) ([]byte, error) {
	if ciphertext.IsZero() {
		return nil, nil
	}

	if len(ciphertext) < aesGCMNonceSize+e.aead.Overhead() {
		return nil, event.Wrapf(
			ErrDecryptionFailed, event.Rejection,
			"encryption.ciphertext_too_short",
			"ciphertext length %d < minimum %d",
			len(ciphertext), aesGCMNonceSize+e.aead.Overhead(),
		)
	}

	nonce := ciphertext[:aesGCMNonceSize]
	data := ciphertext[aesGCMNonceSize:]

	plaintext, err := e.aead.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "encryption.decrypt", "decrypt ciphertext")
	}

	return plaintext, nil
}
