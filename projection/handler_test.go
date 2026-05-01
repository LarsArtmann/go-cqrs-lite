package projection

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

func TestHandlerRegistry_On(t *testing.T) {
	t.Parallel()

	r := NewHandlerRegistry()

	err := r.On("UserCreated", func(_ context.Context, _ event.Event) error { return nil })
	if err != nil {
		t.Fatalf("On: %v", err)
	}

	types := r.EventTypes()
	if len(types) != 1 || types[0] != "UserCreated" {
		t.Errorf("EventTypes = %v, want [UserCreated]", types)
	}

	handlers := r.Lookup("UserCreated")
	if len(handlers) != 1 {
		t.Errorf("Lookup(UserCreated) = %d handlers, want 1", len(handlers))
	}
}

func TestHandlerRegistry_On_NilHandler(t *testing.T) {
	t.Parallel()

	r := NewHandlerRegistry()

	err := r.On("UserCreated", nil)
	if err == nil {
		t.Fatal("expected error for nil handler")
	}
}

func TestHandlerRegistry_OnAll(t *testing.T) {
	t.Parallel()

	r := NewHandlerRegistry()

	err := r.OnAll(func(_ context.Context, _ event.Event) error { return nil })
	if err != nil {
		t.Fatalf("OnAll: %v", err)
	}

	handlers := r.Lookup("anything")
	if len(handlers) != 1 {
		t.Errorf("Lookup(anything) = %d handlers, want 1 (wildcard)", len(handlers))
	}
}

func TestHandlerRegistry_Lookup_CombinesSpecificAndWildcard(t *testing.T) {
	t.Parallel()

	r := NewHandlerRegistry()

	_ = r.On("UserCreated", func(_ context.Context, _ event.Event) error { return nil })
	_ = r.OnAll(func(_ context.Context, _ event.Event) error { return nil })

	handlers := r.Lookup("UserCreated")
	if len(handlers) != 2 {
		t.Errorf("Lookup(UserCreated) = %d handlers, want 2 (specific + wildcard)", len(handlers))
	}

	handlers = r.Lookup("OtherEvent")
	if len(handlers) != 1 {
		t.Errorf("Lookup(OtherEvent) = %d handlers, want 1 (wildcard only)", len(handlers))
	}
}

func TestHandlerRegistry_HasHandlers(t *testing.T) {
	t.Parallel()

	r := NewHandlerRegistry()
	if r.HasHandlers() {
		t.Error("empty registry should not have handlers")
	}

	_ = r.On("Test", func(_ context.Context, _ event.Event) error { return nil })
	if !r.HasHandlers() {
		t.Error("registry with On handler should have handlers")
	}
}
