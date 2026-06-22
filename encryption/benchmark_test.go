package encryption_test

import (
	"crypto/rand"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/encryption/v3"
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

func BenchmarkXChaCha20Poly1305_Encrypt(b *testing.B) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := encryption.NewXChaCha20Poly1305(key)
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

func BenchmarkXChaCha20Poly1305_Decrypt(b *testing.B) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := encryption.NewXChaCha20Poly1305(key)
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

func BenchmarkCodecWrapper_XChaCha20_Encode(b *testing.B) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := encryption.NewXChaCha20Poly1305(key)
	codec := encryption.NewCodec(codec.JSONCodec{}, enc)

	type payload struct {
		Name string `json:"name"`
	}

	data := payload{Name: "benchmark-test"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodecWrapper_XChaCha20_Decode(b *testing.B) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := encryption.NewXChaCha20Poly1305(key)
	codec := encryption.NewCodec(codec.JSONCodec{}, enc)

	type payload struct {
		Name string `json:"name"`
	}

	encoded, err := codec.Encode(payload{Name: "benchmark-test"})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var decoded payload
		if err := codec.Decode(encoded, &decoded); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodecWrapper_XChaCha20_RoundTrip(b *testing.B) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := encryption.NewXChaCha20Poly1305(key)
	codec := encryption.NewCodec(codec.JSONCodec{}, enc)

	type payload struct {
		Name string `json:"name"`
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		encoded, err := codec.Encode(payload{Name: "benchmark-test"})
		if err != nil {
			b.Fatal(err)
		}

		var decoded payload
		if err := codec.Decode(encoded, &decoded); err != nil {
			b.Fatal(err)
		}
	}
}
