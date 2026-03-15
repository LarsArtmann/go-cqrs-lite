package aggregate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/aggregate"
	"github.com/larsartmann/go-cqrs-lite/event"
)

func TestBase(t *testing.T) {
	base := aggregate.NewBase("user-123", event.AggregateType("User"))
	if base.ID() != "user-123" {
		t.Errorf("expected ID user-123, got %s", base.ID())
	}
	if base.Type() != "User" {
		t.Errorf("expected type User, got %s", base.Type())
	}
	if base.Version() != 0 {
		t.Errorf("expected version 0, got %d", base.Version())
	}
}

func TestBaseLoadFromHistory(t *testing.T) {
	base := aggregate.NewBase("user-123", event.AggregateType("User"))

	evt, err := event.NewEvent("UserCreated", "user-123", "User", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = base.LoadFromHistory([]event.Event{evt})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if base.Version() != 1 {
		t.Errorf("expected version 1 after loading history, got %d", base.Version())
	}
}
