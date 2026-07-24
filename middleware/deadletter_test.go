package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestDeadLetter_CalledOnExhaustion(t *testing.T) {
	t.Parallel()

	store := NewMemoryDeadLetterStore()

	config := retryConfigFast()
	config.OnDeadLetter = store.Handle

	mw := EventRetry(config)

	handler := mw(func(_ context.Context, _ event.Event) error {
		return errors.New("permanent failure")
	})

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = handler(context.Background(), evt)

	if store.Count() != 1 {
		t.Fatalf("expected 1 dead-letter entry, got %d", store.Count())
	}

	entries := store.Entries()
	entry := entries[0]

	if entry.Kind != "event" {
		t.Errorf("expected Kind 'event', got %q", entry.Kind)
	}

	if entry.Attempts != config.MaxAttempts {
		t.Errorf("expected Attempts=%d, got %d", config.MaxAttempts, entry.Attempts)
	}

	if entry.Error == nil {
		t.Error("expected non-nil Error")
	}

	if entry.FailedAt.IsZero() {
		t.Error("expected non-zero FailedAt")
	}
}

func TestDeadLetter_NotCalledOnSuccess(t *testing.T) {
	t.Parallel()

	store := NewMemoryDeadLetterStore()

	config := retryConfigFast()
	config.OnDeadLetter = store.Handle

	mw := EventRetry(config)

	handler := mw(func(_ context.Context, _ event.Event) error {
		return nil
	})

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.Count() != 0 {
		t.Fatalf("expected 0 dead-letter entries, got %d", store.Count())
	}
}

func TestDeadLetter_NotCalledOnNonRetryable(t *testing.T) {
	t.Parallel()

	store := NewMemoryDeadLetterStore()

	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.IsRetryable = func(_ error) bool { return false }
	config.OnDeadLetter = store.Handle

	mw := EventRetry(config)

	handler := mw(func(_ context.Context, _ event.Event) error {
		return errors.New("non-retryable failure")
	})

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = handler(context.Background(), evt)

	if store.Count() != 0 {
		t.Fatalf("non-retryable errors should not be dead-lettered, got %d", store.Count())
	}
}

func TestDeadLetter_CapturesStreamID(t *testing.T) {
	t.Parallel()

	store := NewMemoryDeadLetterStore()
	streamID := id.NewStreamID()

	config := retryConfigFast()
	config.OnDeadLetter = store.Handle

	mw := CommandRetry(config)

	cmd := &testCommand{streamID: streamID}
	handler := mw(func(_ context.Context, _ command.Command) error {
		return errors.New("fail")
	})

	_ = handler(context.Background(), cmd)

	entries := store.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].StreamID != streamID {
		t.Errorf("expected StreamID %v, got %v", streamID, entries[0].StreamID)
	}

	if entries[0].Kind != "command" {
		t.Errorf("expected Kind 'command', got %q", entries[0].Kind)
	}
}

func TestMemoryDeadLetterStore_Clear(t *testing.T) {
	t.Parallel()

	store := NewMemoryDeadLetterStore()

	store.Handle(context.Background(), DeadLetterEntry{Kind: "event", Type: "test"})
	store.Handle(context.Background(), DeadLetterEntry{Kind: "event", Type: "test2"})

	if store.Count() != 2 {
		t.Fatalf("expected 2 entries, got %d", store.Count())
	}

	store.Clear()

	if store.Count() != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", store.Count())
	}
}

func TestMemoryDeadLetterStore_Entries_IsCopy(t *testing.T) {
	t.Parallel()

	store := NewMemoryDeadLetterStore()

	store.Handle(context.Background(), DeadLetterEntry{Kind: "event", Type: "test"})

	entries := store.Entries()
	entries[0] = DeadLetterEntry{Kind: "mutated"}

	again := store.Entries()
	if again[0].Kind != "event" {
		t.Error("modifying returned slice should not affect store")
	}
}
