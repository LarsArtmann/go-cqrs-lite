package signing

import (
	"crypto/ed25519"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func BenchmarkCanonicalPayload(b *testing.B) {
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("benchmark.created", aggID, "Benchmark", 1, []byte(`{"key":"value"}`))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = canonicalPayload(evt)
	}
}

func BenchmarkHMAC_Sign(b *testing.B) {
	key := []byte("benchmark-key-thirty-two-bytes!!")
	signer, _ := NewHMAC(key)
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("benchmark.created", aggID, "Benchmark", 1, []byte(`{"key":"value"}`))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = signer.Sign(evt)
	}
}

func BenchmarkHMAC_Verify(b *testing.B) {
	key := []byte("benchmark-key-thirty-two-bytes!!")
	signer, _ := NewHMAC(key)
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("benchmark.created", aggID, "Benchmark", 1, []byte(`{"key":"value"}`))
	sig, _ := signer.Sign(evt)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = signer.Verify(evt, sig)
	}
}

func BenchmarkEd25519_Sign(b *testing.B) {
	_, privKey, _ := ed25519.GenerateKey(nil)
	signer, _ := NewEd25519(privKey)
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("benchmark.created", aggID, "Benchmark", 1, []byte(`{"key":"value"}`))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = signer.Sign(evt)
	}
}

func BenchmarkEd25519_Verify(b *testing.B) {
	pubKey, privKey, _ := ed25519.GenerateKey(nil)
	signer, _ := NewEd25519(privKey)
	verifier, _ := NewEd25519Verifier(pubKey)
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("benchmark.created", aggID, "Benchmark", 1, []byte(`{"key":"value"}`))
	sig, _ := signer.Sign(evt)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = verifier.Verify(evt, sig)
	}
}

func BenchmarkVerifyAll(b *testing.B) {
	pubKey, privKey, _ := ed25519.GenerateKey(nil)
	deviceSigner, _ := NewEd25519(privKey)
	deviceVerifier, _ := NewEd25519Verifier(pubKey)
	serverKey := []byte("server-secret-key-thirty-two-by!")
	serverSigner, _ := NewHMAC(serverKey)

	deviceMulti, _ := NewMultiSigner(
		Actor("device"), AlgorithmEd25519, deviceSigner,
		WithVerifier(deviceVerifier),
	)
	serverMulti, _ := NewMultiSigner(
		Actor("server"), AlgorithmHMACSHA256, serverSigner,
	)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("benchmark.created", aggID, "Benchmark", 1, []byte(`{"key":"value"}`))

	deviceSigned, _ := deviceMulti.Sign(evt)
	serverSigned, _ := serverMulti.Sign(deviceSigned)

	verifiers := map[Actor]Verifier{
		Actor("device"): deviceVerifier,
		Actor("server"): serverSigner,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VerifyAll(serverSigned, verifiers)
	}
}
