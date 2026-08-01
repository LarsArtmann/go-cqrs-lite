package benchkit

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

func metaEngineFactory(t *testing.T) Factory {
	t.Helper()

	return func() (*stack.Bundle, error) {
		eng := metaengine.NewMemoryEngine()

		store, err := metaengine.Plan(
			[]metaengine.Engine{eng},
			metaengine.Query[meBenchCounterInput, map[string]int64](
				"bench_counter",
				metaengine.On(
					meBenchIncrementEvent{},
					func(e meBenchIncrementEvent) metaengine.Delta {
						return metaengine.Delta{e.Status: +1}
					},
				),
			),
		)
		if err != nil {
			return nil, err
		}

		return stack.New(
			stack.WithEventStore(memory.NewMemoryStore()),
			stack.WithBus(cqrswatermill.NewEventBus()),
			stack.WithReadModels(kv.NewMemStore()),
			stack.WithMetaEngine(store),
		)
	}
}

func TestMetaEnginePhase_Memory(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:         ProfileDev,
		PayloadSize:     128,
		Backend:         "memory",
		SkipReads:       true,
		SkipRawSink:     true,
		SkipReadModels:  true,
		SkipProjections: true,
	}, metaEngineFactory(t))

	if result.MetaEngineApplyLatency.Count == 0 {
		t.Error("MetaEngineApplyLatency.Count = 0, expected nonzero")
	}

	if result.MetaEngineQueryLatency.Count == 0 {
		t.Error("MetaEngineQueryLatency.Count = 0, expected nonzero")
	}

	if result.MetaEngineApplyThroughput <= 0 {
		t.Error("MetaEngineApplyThroughput <= 0, expected positive")
	}

	// Map ADT workload metrics
	if result.MetaEngineScanLatency.Count == 0 {
		t.Error("MetaEngineScanLatency.Count = 0, expected nonzero (scan should run)")
	}

	if result.MetaEnginePointReadLatency.Count == 0 {
		t.Error("MetaEnginePointReadLatency.Count = 0, expected nonzero (point read should run)")
	}

	if result.MetaEngineApplyConcurrent <= 0 {
		t.Error("MetaEngineApplyConcurrent <= 0, expected positive")
	}

	if result.MetaEngineScanResults <= 0 {
		t.Error("MetaEngineScanResults <= 0, scan should return active items")
	}
}

func TestMetaEnginePhase_SkipFlag(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:         ProfileDev,
		PayloadSize:     128,
		SkipMetaEngine:  true,
		SkipReads:       true,
		SkipRawSink:     true,
		SkipReadModels:  true,
		SkipProjections: true,
	}, metaEngineFactory(t))

	if result.MetaEngineApplyLatency.Count != 0 {
		t.Errorf("MetaEngineApplyLatency.Count = %d, expected 0 with SkipMetaEngine",
			result.MetaEngineApplyLatency.Count)
	}
}

func TestMetaEnginePhase_NoMetaEngine(t *testing.T) {
	t.Parallel()

	// A plain memory bundle without metaengine — phase should gracefully skip.
	result := mustRun(t, Config{
		Profile:         ProfileDev,
		PayloadSize:     128,
		SkipReads:       true,
		SkipRawSink:     true,
		SkipReadModels:  true,
		SkipProjections: true,
	}, func() (*stack.Bundle, error) {
		return stack.New(
			stack.WithEventStore(memory.NewMemoryStore()),
			stack.WithBus(cqrswatermill.NewEventBus()),
			stack.WithReadModels(kv.NewMemStore()),
		)
	})

	if result.MetaEngineApplyLatency.Count != 0 {
		t.Errorf("MetaEngineApplyLatency.Count = %d, expected 0 without metaengine",
			result.MetaEngineApplyLatency.Count)
	}
}
