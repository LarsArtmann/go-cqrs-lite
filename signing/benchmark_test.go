package signing

import (
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
