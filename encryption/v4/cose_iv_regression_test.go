package encryption_test

import (
	"testing"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/encryption/v4"
)

// coseBoth is a cipher that both builds and opens test messages.
type coseBoth struct {
	encryption.COSEEncrypter
	decryption encryption.COSEDecrypter
}

func (c coseBoth) Decrypt(ciphertext, nonce, aad []byte) ([]byte, error) {
	return c.decryption.Decrypt(ciphertext, nonce, aad)
}

// The IV sits in the unauthenticated header, so anyone can edit it: a wrong
// size must surface as an error, never as the AEAD's nonce-length panic.
func TestDecryptCOSE0WrongIVSizeReturnsError(t *testing.T) {
	t.Parallel()

	key := generateTestKey(t)

	aes, err := encryption.NewCOSEAES256GCM(key)
	if err != nil {
		t.Fatal(err)
	}

	xchaCha, err := encryption.NewCOSEXChaCha20Poly1305(key)
	if err != nil {
		t.Fatal(err)
	}

	ciphers := map[string]coseBoth{"aes-gcm": {aes, aes}, "xchacha20": {xchaCha, xchaCha}}

	for name, cipher := range ciphers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			coseBytes, err := encryption.EncryptCOSE0([]byte("secret"), cipher)
			if err != nil {
				t.Fatal(err)
			}

			msg, err := codec.UnmarshalCOSEEncrypt0(coseBytes)
			if err != nil {
				t.Fatal(err)
			}

			for _, size := range []int{0, 11, 13, 23, 25} {
				msg.Unprotected[codec.COSEHeaderIV] = make([]byte, size)

				tampered, err := codec.MarshalCOSEEncrypt0(msg)
				if err != nil {
					t.Fatal(err)
				}

				assertDecryptError(t, tampered, cipher, size)
			}
		})
	}
}

func assertDecryptError(t *testing.T, coseBytes []byte, dec encryption.COSEDecrypter, size int) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("iv size %d: DecryptCOSE0 panicked: %v", size, r)
		}
	}()

	if _, err := encryption.DecryptCOSE0(coseBytes, dec); err == nil {
		t.Fatalf("iv size %d: expected error, got nil", size)
	}
}
