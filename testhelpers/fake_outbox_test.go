package testhelpers

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestFakeOutbox_Append(t *testing.T) {
	t.Parallel()

	outbox := NewFakeOutbox()

	evt, _ := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)

	err := outbox.Append(context.Background(), []event.Event{evt})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	if len(outbox.Entries) != 1 {
		t.Fatalf("len(Entries) = %d, want 1", len(outbox.Entries))
	}

	if outbox.Entries[0].ID == "" {
		t.Error("entry ID should not be empty")
	}
}

func TestFakeOutbox_AppendFn(t *testing.T) {
	t.Parallel()

	outbox := NewFakeOutbox()

	called := false

	outbox.AppendFn(func(_ []event.Event) error {
		called = true

		return errors.New("custom error")
	})

	err := outbox.Append(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error from custom AppendFn")
	}

	if !called {
		t.Error("AppendFn callback not called")
	}
}

func TestFakeOutbox_PollPending(t *testing.T) {
	t.Parallel()

	outbox := NewFakeOutbox()

	evt, _ := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	_ = outbox.Append(context.Background(), []event.Event{evt})

	entries, err := outbox.PollPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("PollPending: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
}

func TestFakeOutbox_Ack(t *testing.T) {
	t.Parallel()

	outbox := NewFakeOutbox()

	evt, _ := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	_ = outbox.Append(context.Background(), []event.Event{evt})

	entries, _ := outbox.PollPending(context.Background(), 10)

	err := outbox.Ack(context.Background(), []event.OutboxID{entries[0].ID})
	if err != nil {
		t.Fatalf("Ack: %v", err)
	}

	if len(outbox.Entries) != 0 {
		t.Errorf("len(Entries) after ack = %d, want 0", len(outbox.Entries))
	}
}

func TestFakeOutbox_Close(t *testing.T) {
	t.Parallel()

	outbox := NewFakeOutbox()

	err := outbox.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}
