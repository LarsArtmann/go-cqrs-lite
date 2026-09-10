package metaengine

import (
	"context"
	"fmt"
	"testing"
)

func TestStore_Reset_ClearsMemoryEngineAndReplayState(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)
	ctx := context.Background()

	eventLog := NewEventLog()
	WithEventLog(store, eventLog)

	for i := range 3 {
		if err := store.ApplyIdempotent(ctx, fmt.Sprintf("evt-%d", i), "task_created", testTask{
			ID:     testTaskID(fmt.Sprintf("t%d", i)),
			Title:  fmt.Sprintf("Task %d", i),
			Status: "open",
		}); err != nil {
			t.Fatalf("ApplyIdempotent: %v", err)
		}
	}

	reader := NewReader[testTask](store, "tasks")

	before, err := reader.Count(ctx)
	if err != nil {
		t.Fatalf("Count before reset: %v", err)
	}

	if before != 3 {
		t.Fatalf("expected 3 tasks before reset, got %d", before)
	}

	if eventLog.Len() != 3 {
		t.Fatalf("expected event log to record 3 applied events, got %d", eventLog.Len())
	}

	if store.idempotency.Len() != 3 {
		t.Fatalf("expected idempotency window to track 3 event IDs, got %d", store.idempotency.Len())
	}

	result, err := store.Reset(ctx)
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if result.Partial() {
		t.Fatalf("memory engine implements EngineResetter, reset must not be partial: %s", result)
	}

	if len(result.ClearedEngines) != 1 {
		t.Fatalf("expected exactly 1 cleared engine, got %v", result.ClearedEngines)
	}

	after, err := reader.Count(ctx)
	if err != nil {
		t.Fatalf("Count after reset: %v", err)
	}

	if after != 0 {
		t.Fatalf("expected 0 tasks after reset, got %d", after)
	}

	if eventLog.Len() != 0 {
		t.Fatalf("expected event log cleared, got %d entries", eventLog.Len())
	}

	if store.idempotency.Len() != 0 {
		t.Fatalf("expected idempotency window cleared, got %d entries", store.idempotency.Len())
	}

	// A replay after reset must re-apply the same event ID: the cleared
	// idempotency window no longer suppresses it.
	if err := store.ApplyIdempotent(ctx, "evt-0", "task_created", testTask{
		ID: testTaskID("t0"), Title: "Task 0", Status: "open",
	}); err != nil {
		t.Fatalf("replay ApplyIdempotent: %v", err)
	}

	replayed, err := reader.Count(ctx)
	if err != nil {
		t.Fatalf("Count after replay: %v", err)
	}

	if replayed != 1 {
		t.Fatalf("expected replay to re-apply evt-0 (1 task), got %d", replayed)
	}
}

func TestStore_Reset_ReportsUnclearableEngines(t *testing.T) {
	t.Parallel()

	// plainEngine implements Engine (and supports ADTMap so Plan accepts it)
	// but deliberately does NOT implement EngineResetter.
	plain := &plainEngine{profile: EngineProfile{
		Name: "plain-no-reset",
		Supports: map[ADT]Complexity{
			ADTMap: ComplexityO1,
		},
	}}

	store, err := Plan([]Engine{plain}, testTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	result, err := store.Reset(context.Background())
	if err != nil {
		t.Fatalf("Reset should not error for an unclearable engine, only report it: %v", err)
	}

	if !result.Partial() {
		t.Fatal("expected a partial reset when an engine lacks EngineResetter")
	}

	if len(result.UnclearableEngines) != 1 || result.UnclearableEngines[0] != "plain-no-reset" {
		t.Fatalf("expected the unclearable engine to be named, got %v", result.UnclearableEngines)
	}

	if len(result.ClearedEngines) != 0 {
		t.Fatalf("expected no cleared engines, got %v", result.ClearedEngines)
	}
}
