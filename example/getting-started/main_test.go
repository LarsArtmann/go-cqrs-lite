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

	// The at-least-once canary (see README): delivery is at-least-once, so a
	// correct projection must dedup the drain/live overlap. The two wrong
	// directions name different seams.
	switch {
	case view.Value > 10:
		t.Errorf(
			"counter value: got %d, want 10 — events applied more than once: the projection double-applied the drain/live overlap (delivery is at-least-once BY DESIGN; the fold layer must be idempotent or the checkpoint store must dedup — this is a library bug, not flakiness)",
			view.Value,
		)
	case view.Value < 10:
		t.Errorf(
			"counter value: got %d, want 10 — events lost between journal and projection: a replay/checkpoint gap (pin drift or a missed journal segment), not a fold bug (folds are pure and total)",
			view.Value,
		)
	}
}
