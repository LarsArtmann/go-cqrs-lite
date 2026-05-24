package testhelpers

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestFakeBus_Publish(t *testing.T) {
	t.Parallel()

	bus := NewFakeBus()

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("test.created", aggID, "Test", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if len(bus.Published) != 1 {
		t.Fatalf("len(Published) = %d, want 1", len(bus.Published))
	}

	if bus.Published[0].ID() != evt.ID() {
		t.Error("published event ID mismatch")
	}
}

func TestFakeBus_PublishError(t *testing.T) {
	t.Parallel()

	bus := NewFakeBus()
	bus.PublishErr = errors.New("boom")

	err := bus.Publish(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	if bus.PublishErr.Error() != "boom" {
		t.Errorf("error = %q, want boom", err)
	}
}

func TestFakeBus_SubscribeAll(t *testing.T) {
	t.Parallel()

	bus := NewFakeBus()

	err := bus.SubscribeAll(func(_ context.Context, _ event.Event) error { return nil })
	if err != nil {
		t.Fatalf("SubscribeAll: %v", err)
	}
}

func TestFakeBus_SubscribeAllFn(t *testing.T) {
	t.Parallel()

	bus := NewFakeBus()

	called := false

	bus.SubscribeAllFn(func(_ event.Handler) error {
		called = true

		return nil
	})

	err := bus.SubscribeAll(func(_ context.Context, _ event.Event) error { return nil })
	if err != nil {
		t.Fatalf("SubscribeAll: %v", err)
	}

	if !called {
		t.Error("SubscribeAllFn callback not called")
	}
}

func TestFakeBus_Subscribe(t *testing.T) {
	t.Parallel()

	bus := NewFakeBus()

	err := bus.Subscribe("test.created", func(_ context.Context, _ event.Event) error { return nil })
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
}

func TestFakeBus_Use(t *testing.T) {
	t.Parallel()

	bus := NewFakeBus()

	err := bus.Use(func(_ event.Handler) event.Handler { return nil })
	if err != nil {
		t.Fatalf("Use: %v", err)
	}
}

func TestFakeBus_Close(t *testing.T) {
	t.Parallel()

	bus := NewFakeBus()

	err := bus.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}
