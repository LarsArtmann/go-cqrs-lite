package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// The example pipeline runs end-to-end against a real SQLite engine —
// proving the "swap one EngineConfig line" deployment story: the domain
// code (events, decider, folds) is identical to the in-memory main().
func TestGettingStarted_CounterValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "counter.db")

	sys, err := buildSystem(ctx, dbPath)
	if err != nil {
		t.Fatalf("buildSystem: %v", err)
	}

	defer func() { _ = sys.Close() }()

	counterID := id.NewStreamID()

	view, err := runPipeline(ctx, sys, counterID)
	if err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	if view.Value != 10 {
		t.Errorf("counter value: got %d, want 10", view.Value)
	}
}
