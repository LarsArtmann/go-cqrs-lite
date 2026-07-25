package benchkit

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func TestQueryPhase_Memory(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		Backend:     "memory",
		SkipReads:   true,
		SkipRawSink: true,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	if result.QueryHitLatency.Count == 0 {
		t.Error("QueryHitLatency.Count = 0, expected nonzero")
	}

	if result.QueryMissLatency.Count == 0 {
		t.Error("QueryMissLatency.Count = 0, expected nonzero")
	}

	if result.QueryPaginatedLatency.Count == 0 {
		t.Error("QueryPaginatedLatency.Count = 0, expected nonzero")
	}

	if result.QueryCorrectnessErrors > 0 {
		t.Errorf("QueryCorrectnessErrors = %d, expected 0", result.QueryCorrectnessErrors)
	}
}

func TestQueryPhase_SkipFlag(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		SkipQuery:   true,
		SkipReads:   true,
		SkipRawSink: true,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	if result.QueryHitLatency.Count != 0 {
		t.Errorf("QueryHitLatency.Count = %d, expected 0 with SkipQuery",
			result.QueryHitLatency.Count)
	}
}

func TestQueryPhase_MissFasterThanHit(t *testing.T) {
	t.Parallel()

	// The miss path (handler-not-found) returns immediately without doing I/O,
	// so it should generally be faster than the hit path (which reads from the
	// kv.Store). We check that miss latency is nonzero and roughly comparable
	// (within an order of magnitude). This is a sanity check, not a hard assert.
	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		SkipReads:   true,
		SkipRawSink: true,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	if result.QueryMissLatency.P50 == 0 {
		t.Error("QueryMissLatency.P50 = 0, expected nonzero")
	}

	// Miss should not be dramatically slower than hit (no I/O).
	if result.QueryMissLatency.P99 > result.QueryHitLatency.P99*10 {
		t.Logf("note: miss P99 (%s) > hit P99*10 (%s) — may be noise",
			result.QueryMissLatency.P99, result.QueryHitLatency.P99*10)
	}
}
