package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestMemoryOutboxStore_AppendAndPoll(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outbox := memory.NewMemoryOutboxStore()
	aggID := id.NewAggregateID()

	evt := testhelpers.NewEvent(t, "TestEvent", aggID, "Test", 1, nil)

	err := outbox.Append(ctx, []event.Event{evt})
	if err != nil {
		t.Fatalf("append to outbox: %v", err)
	}

	entries, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending: %v", err)
	}

	testhelpers.AssertLenFatal(t, "entries", entries, 1)

	testhelpers.AssertLen(t, "entry events", entries[0].Events, 1)

	if entries[0].Events[0].Type() != "TestEvent" {
		t.Errorf("expected event type TestEvent, got %s", entries[0].Events[0].Type())
	}
}

func TestMemoryOutboxStore_AckRemovesFromPending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outbox := memory.NewMemoryOutboxStore()
	aggID := id.NewAggregateID()

	evt := testhelpers.NewEvent(t, "TestEvent", aggID, "Test", 1, nil)

	err := outbox.Append(ctx, []event.Event{evt})
	if err != nil {
		t.Fatalf("append to outbox: %v", err)
	}

	entries, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending: %v", err)
	}

	testhelpers.AssertLenFatal(t, "entries", entries, 1)

	err = outbox.Ack(ctx, []event.OutboxID{entries[0].ID})
	if err != nil {
		t.Fatalf("ack entry: %v", err)
	}

	entries, err = outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending after ack: %v", err)
	}

	testhelpers.AssertLen(t, "pending entries after ack", entries, 0)
}

func TestMemoryOutboxStore_PollPendingLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outbox := memory.NewMemoryOutboxStore()

	for i := range 5 {
		aggID := id.NewAggregateID()
		evt := testhelpers.NewEvent(t, "TestEvent", aggID, "Test", i+1, nil)

		err := outbox.Append(ctx, []event.Event{evt})
		if err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
	}

	entries, err := outbox.PollPending(ctx, 2)
	if err != nil {
		t.Fatalf("poll pending: %v", err)
	}

	testhelpers.AssertLen(t, "entries with limit", entries, 2)
}

func TestMemoryOutboxStore_Ack_RemovesEntryFromSlice(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outbox := memory.NewMemoryOutboxStore()

	aggID1 := id.NewAggregateID()
	evt1 := testhelpers.NewEvent(t, "Event1", aggID1, "Test", 1, nil)

	err := outbox.Append(ctx, []event.Event{evt1})
	if err != nil {
		t.Fatalf("append event 1: %v", err)
	}

	entries1, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending 1: %v", err)
	}

	testhelpers.AssertLenFatal(t, "entries1", entries1, 1)

	err = outbox.Ack(ctx, []event.OutboxID{entries1[0].ID})
	if err != nil {
		t.Fatalf("ack entry 1: %v", err)
	}

	aggID2 := id.NewAggregateID()
	evt2 := testhelpers.NewEvent(t, "Event2", aggID2, "Test", 1, nil)

	err = outbox.Append(ctx, []event.Event{evt2})
	if err != nil {
		t.Fatalf("append event 2: %v", err)
	}

	entries2, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending 2: %v", err)
	}

	testhelpers.AssertLenFatal(t, "entries2", entries2, 1)

	if entries2[0].Events[0].Type() != "Event2" {
		t.Errorf("expected Event2, got %s", entries2[0].Events[0].Type())
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

	evt := testhelpers.NewEvent(t, "TestEvent", aggID, "Test", 1, []byte(`original`))

	err := outbox.Append(ctx, []event.Event{evt})
	if err != nil {
		t.Fatalf("append to outbox: %v", err)
	}

	entries, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending: %v", err)
	}

	testhelpers.AssertLenFatal(t, "entries", entries, 1)

	entries[0].Events[0].Payload()[0] = 'X'

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

func TestMemoryOutboxStore_Ack_PartialAck(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outbox := memory.NewMemoryOutboxStore()

	aggID1 := id.NewAggregateID()
	aggID2 := id.NewAggregateID()

	evt1 := testhelpers.NewEvent(t, "Event1", aggID1, "Test", 1, nil)
	evt2 := testhelpers.NewEvent(t, "Event2", aggID2, "Test", 1, nil)

	err := outbox.Append(ctx, []event.Event{evt1})
	if err != nil {
		t.Fatalf("append event 1: %v", err)
	}

	err = outbox.Append(ctx, []event.Event{evt2})
	if err != nil {
		t.Fatalf("append event 2: %v", err)
	}

	entries, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll pending: %v", err)
	}

	testhelpers.AssertLenFatal(t, "entries", entries, 2)

	err = outbox.Ack(ctx, []event.OutboxID{entries[0].ID})
	if err != nil {
		t.Fatalf("ack first entry: %v", err)
	}

	remaining, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll remaining: %v", err)
	}

	testhelpers.AssertLen(t, "remaining entries", remaining, 1)

	if remaining[0].Events[0].Type() != "Event2" {
		t.Errorf("expected Event2 remaining, got %s", remaining[0].Events[0].Type())
	}
}
