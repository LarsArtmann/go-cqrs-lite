package benchkit

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	storagemem "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestJourneyPhase_Memory(t *testing.T) {
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

	if result.JourneySamples == 0 {
		t.Fatal("JourneySamples = 0, expected nonzero")
	}

	if result.JourneyLatency.Count == 0 {
		t.Error("JourneyLatency.Count = 0, expected nonzero")
	}

	if result.JourneyProjectionLatency.Count == 0 {
		t.Error("JourneyProjectionLatency.Count = 0, expected nonzero")
	}

	if result.JourneyQueryLatency.Count == 0 {
		t.Error("JourneyQueryLatency.Count = 0, expected nonzero")
	}

	if result.QueryCorrectnessErrors > 0 {
		t.Errorf("QueryCorrectnessErrors = %d, expected 0", result.QueryCorrectnessErrors)
	}
}

func TestJourneyPhase_SkippedWithoutReadModels(t *testing.T) {
	t.Parallel()

	// Bundle without ReadModels — journey should skip gracefully.
	bundle, err := stack.New(
		stack.WithEventStore(storagemem.NewMemoryStore()),
	)
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := Run(ctx, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
		SkipReads:   true,
		SkipRawSink: true,
	}, func() (*stack.Bundle, error) {
		return bundle, nil
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.JourneySamples != 0 {
		t.Errorf("JourneySamples = %d, expected 0 without ReadModels", result.JourneySamples)
	}
}

func TestJourneyPhase_SkipFlag(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:      ProfileDev,
		PayloadSize:  128,
		SkipJourney:  true,
		SkipReads:    true,
		SkipRawSink:  true,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	if result.JourneySamples != 0 {
		t.Errorf("JourneySamples = %d, expected 0 with SkipJourney", result.JourneySamples)
	}
}

func TestJourneyReport(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 128,
		SkipReads:   true,
		SkipRawSink: true,
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})

	var buf bytes.Buffer
	PrintReport(&buf, result)

	output := buf.String()
	if !strings.Contains(output, "Journey") {
		t.Errorf("report missing Journey section;\noutput: %s", output)
	}

	if !strings.Contains(output, "Round trip:") {
		t.Error("report missing 'Round trip:' line")
	}

	if !strings.Contains(output, "Query Dispatch:") {
		t.Error("report missing 'Query Dispatch:' section")
	}
}
