package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func TestMemoryOutboxStore_AppendAndPoll(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outbox := memory.NewMemoryOutboxStore()
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent("TestEvent", aggID, "Test", 1, nil)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	err = outbox.Append(ctx, []event.Event{evt})
	if err != nil {
		t.Fatalf("append to outbox: %v", err)
	}

	entries, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 pending entry, got %d", len(entries))
	}

	if len(entries[0].Events) != 1 {
		t.Errorf("expected 1 event in entry, got %d", len(entries[0].Events))
	}

	if entries[0].Events[0].Type() != "TestEvent" {
		t.Errorf("expected event type TestEvent, got %s", entries[0].Events[0].Type())
	}
}

func TestMemoryOutboxStore_AckRemovesFromPending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outbox := memory.NewMemoryOutboxStore()
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent("TestEvent", aggID, "Test", 1, nil)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	err = outbox.Append(ctx, []event.Event{evt})
	if err != nil {
		t.Fatalf("append to outbox: %v", err)
	}

	entries, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 pending entry, got %d", len(entries))
	}

	err = outbox.Ack(ctx, []event.OutboxID{entries[0].ID})
	if err != nil {
		t.Fatalf("ack entry: %v", err)
	}

	// After ack, no pending entries
	entries, err = outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending after ack: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 pending entries after ack, got %d", len(entries))
	}
}

func TestMemoryOutboxStore_PollPendingLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outbox := memory.NewMemoryOutboxStore()

	// Append 5 events
	for i := range 5 {
		aggID := id.NewAggregateID()

		evt, err := event.NewEvent("TestEvent", aggID, "Test", i+1, nil)
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}

		err = outbox.Append(ctx, []event.Event{evt})
		if err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
	}

	// Poll with limit 2
	entries, err := outbox.PollPending(ctx, 2)
	if err != nil {
		t.Fatalf("poll pending: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries with limit, got %d", len(entries))
	}
}

func TestMemoryOutboxStore_AckEmptyIDs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outbox := memory.NewMemoryOutboxStore()

	err := outbox.Ack(ctx, nil)
	if err != nil {
		t.Errorf("expected no error for empty ack, got %v", err)
	}
}

func TestMemoryOutboxStore_DefensiveCopy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outbox := memory.NewMemoryOutboxStore()
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent("TestEvent", aggID, "Test", 1, []byte(`original`))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	err = outbox.Append(ctx, []event.Event{evt})
	if err != nil {
		t.Fatalf("append to outbox: %v", err)
	}

	entries, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	// Modify the returned event payload
	entries[0].Events[0].Payload()[0] = 'X'

	// Poll again and verify original is unchanged
	entries2, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending again: %v", err)
	}

	if string(entries2[0].Events[0].Payload()) != "original" {
		t.Errorf(
			"defensive copy failed: got %q, want %q",
			entries2[0].Events[0].Payload(),
			"original",
		)
	}
}
