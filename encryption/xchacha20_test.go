package encryption_test

import (
	"crypto/rand"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/encryption/v2"
)

func TestXChaCha20_RoundTrip(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	enc, err := encryption.NewXChaCha20Poly1305(key)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte(`{"name":"Alice","email":"alice@example.com"}`)

	ct, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestXChaCha20_DifferentCiphertexts(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	enc, err := encryption.NewXChaCha20Poly1305(key)
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte("same payload")

	ct1, _ := enc.Encrypt(payload)
	ct2, _ := enc.Encrypt(payload)

	if ct1.Equal(ct2) {
		t.Error("two encryptions of same plaintext should produce different ciphertexts")
	}
}

func TestXChaCha20_WrongKeyFails(t *testing.T) {
	t.Parallel()

	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	_, _ = rand.Read(key1)
	_, _ = rand.Read(key2)

	enc1, _ := encryption.NewXChaCha20Poly1305(key1)
	enc2, _ := encryption.NewXChaCha20Poly1305(key2)

	ct, _ := enc1.Encrypt([]byte("secret"))

	_, err := enc2.Decrypt(ct)
	if err == nil {
		t.Error("expected error decrypting with wrong key")
	}
}

func TestXChaCha20_InvalidKeySize(t *testing.T) {
	t.Parallel()

	_, err := encryption.NewXChaCha20Poly1305([]byte("short"))
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestXChaCha20_EmptyPayload(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := encryption.NewXChaCha20Poly1305(key)

	ct, err := enc.Encrypt([]byte{})
	if err != nil {
		t.Fatal(err)
	}

	if !ct.IsZero() {
		t.Error("empty plaintext should produce zero ciphertext")
	}

	plain, err := enc.Decrypt(nil)
	if err != nil {
		t.Fatal(err)
	}

	if plain != nil {
		t.Error("empty ciphertext should produce nil plaintext")
	}
}

func TestXChaCha20_TruncatedCiphertext(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := encryption.NewXChaCha20Poly1305(key)

	_, err := enc.Decrypt(encryption.Ciphertext([]byte{1, 2, 3}))
	if err == nil {
		t.Error("expected error for truncated ciphertext")
	}
}
