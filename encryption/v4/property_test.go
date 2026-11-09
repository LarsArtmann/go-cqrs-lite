package encryption

import (
	"bytes"
	"crypto/rand"
	"testing"

	"pgregory.net/rapid"
)

func TestEncryptDecryptIsInvolutory_AES256GCM(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := NewAES256GCM(key)
	if err != nil {
		t.Fatalf("NewAES256GCM: %v", err)
	}

	propEncryptDecryptInvolutory(t, enc)
}

func propEncryptDecryptInvolutory(t *testing.T, enc EncrypterDecrypter) {
	t.Helper()

	rapid.Check(t, func(t *rapid.T) {
		size := rapid.IntRange(0, 8192).Draw(t, "size")
		plaintext := rapid.SliceOfN(rapid.Byte(), size, size).Draw(t, "plaintext")

		ct, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}

		if len(plaintext) == 0 {
			if !ct.IsZero() {
				t.Fatal("empty plaintext should produce zero ciphertext")
			}

			return
		}

		decrypted, err := enc.Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}

		if !bytes.Equal(decrypted, plaintext) {
			t.Fatalf("involution failed: decrypted != plaintext")
		}
	})
}

func TestEncryptDecryptIsInvolutory_XChaCha20(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := NewXChaCha20Poly1305(key)
	if err != nil {
		t.Fatalf("NewXChaCha20Poly1305: %v", err)
	}

	propEncryptDecryptInvolutory(t, enc)
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := NewAES256GCM(key)

	rapid.Check(t, func(t *rapid.T) {
		plaintext := rapid.SliceOfN(rapid.Byte(), 1, 1024).Draw(t, "plaintext")

		ct1, _ := enc.Encrypt(plaintext)
		ct2, _ := enc.Encrypt(plaintext)

		if ct1.Equal(ct2) {
			t.Fatal(
				"two encryptions of same plaintext should produce different ciphertexts (random nonce)",
			)
		}
	})
}

func TestCiphertextIsolation(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := NewXChaCha20Poly1305(key)

	rapid.Check(t, func(t *rapid.T) {
		plaintext := rapid.SliceOfN(rapid.Byte(), 1, 1024).Draw(t, "plaintext")

		ct, _ := enc.Encrypt(plaintext)

		raw := ct.Bytes()
		if len(raw) > 0 {
			raw[0] ^= 0xff
			after := ct.Bytes()
			if after[0] == raw[0] {
				t.Fatal("mutating Bytes() result affected internal ciphertext state")
			}
		}
	})
}
