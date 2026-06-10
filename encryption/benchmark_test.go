package encryption_test

import (
	"crypto/rand"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/encryption/v2"
)

func BenchmarkAES256GCM_Encrypt(b *testing.B) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := encryption.NewAES256GCM(key)
	if err != nil {
		b.Fatal(err)
	}

	payload := make([]byte, 1024)
	_, _ = rand.Read(payload)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := enc.Encrypt(payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAES256GCM_Decrypt(b *testing.B) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := encryption.NewAES256GCM(key)
	if err != nil {
		b.Fatal(err)
	}

	payload := make([]byte, 1024)
	_, _ = rand.Read(payload)

	ct, err := enc.Encrypt(payload)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := enc.Decrypt(ct)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAES256GCM_RoundTrip(b *testing.B) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := encryption.NewAES256GCM(key)
	if err != nil {
		b.Fatal(err)
	}

	payload := make([]byte, 1024)
	_, _ = rand.Read(payload)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ct, err := enc.Encrypt(payload)
		if err != nil {
			b.Fatal(err)
		}

		_, err = enc.Decrypt(ct)
		if err != nil {
			b.Fatal(err)
		}
	}
}
