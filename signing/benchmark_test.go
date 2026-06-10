package signing

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func parseAggID(s string) id.AggregateID {
	v, err := id.ParseAggregateID(s)
	if err != nil {
		panic(err)
	}
	return v
}

func BenchmarkCanonicalPayload(b *testing.B) {
	b.ReportAllocs()

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("benchmark.created", aggID, "Benchmark", 1, []byte(`{"key":"value"}`))

	b.ResetTimer()

	for b.Loop() {
		_ = canonicalPayload(evt)
	}
}

func BenchmarkHMAC_Sign(b *testing.B) {
	b.ReportAllocs()

	key := []byte("benchmark-key-thirty-two-bytes!!")
	signer, _ := NewHMAC(key)
	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("benchmark.created", aggID, "Benchmark", 1, []byte(`{"key":"value"}`))

	b.ResetTimer()

	for b.Loop() {
		_, _ = signer.Sign(evt)
	}
}

func BenchmarkHMAC_Verify(b *testing.B) {
	b.ReportAllocs()

	key := []byte("benchmark-key-thirty-two-bytes!!")
	signer, _ := NewHMAC(key)
	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("benchmark.created", aggID, "Benchmark", 1, []byte(`{"key":"value"}`))
	sig, _ := signer.Sign(evt)

	b.ResetTimer()

	for b.Loop() {
		_ = signer.Verify(evt, sig)
	}
}

func BenchmarkEd25519_Sign(b *testing.B) {
	b.ReportAllocs()

	_, privKey, _ := GenerateEd25519KeyPair()
	signer, _ := NewEd25519(privKey)
	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("benchmark.created", aggID, "Benchmark", 1, []byte(`{"key":"value"}`))

	b.ResetTimer()

	for b.Loop() {
		_, _ = signer.Sign(evt)
	}
}

func BenchmarkEd25519_Verify(b *testing.B) {
	b.ReportAllocs()

	pubKey, privKey, _ := GenerateEd25519KeyPair()
	signer, _ := NewEd25519(privKey)
	verifier, _ := NewEd25519Verifier(pubKey)
	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	evt, _ := event.NewEvent("benchmark.created", aggID, "Benchmark", 1, []byte(`{"key":"value"}`))
	sig, _ := signer.Sign(evt)

	b.ResetTimer()

	for b.Loop() {
		_ = verifier.Verify(evt, sig)
	}
}
