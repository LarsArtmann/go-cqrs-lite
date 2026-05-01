package projection

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestHandlerRegistry_On(t *testing.T) {
	t.Parallel()

	r := NewHandlerRegistry()

	err := r.On("UserCreated", testhelpers.NoopEventHandler())
	if err != nil {
		t.Fatalf("On: %v", err)
	}

	types := r.EventTypes()
	if len(types) != 1 || types[0] != "UserCreated" {
		t.Errorf("EventTypes = %v, want [UserCreated]", types)
	}

	handlers := r.Lookup("UserCreated")
	testhelpers.AssertLen(t, "handlers", handlers, 1)
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

	err := r.OnAll(testhelpers.NoopEventHandler())
	if err != nil {
		t.Fatalf("OnAll: %v", err)
	}

	handlers := r.Lookup("anything")
	testhelpers.AssertLen(t, "handlers", handlers, 1)
}

func TestHandlerRegistry_Lookup_CombinesSpecificAndWildcard(t *testing.T) {
	t.Parallel()

	r := NewHandlerRegistry()

	_ = r.On("UserCreated", testhelpers.NoopEventHandler())
	_ = r.OnAll(testhelpers.NoopEventHandler())

	handlers := r.Lookup("UserCreated")
	testhelpers.AssertLen(t, "handlers", handlers, 2)

	handlers = r.Lookup("OtherEvent")
	testhelpers.AssertLen(t, "handlers", handlers, 1)
}

func TestHandlerRegistry_HasHandlers(t *testing.T) {
	t.Parallel()

	r := NewHandlerRegistry()
	if r.HasHandlers() {
		t.Error("empty registry should not have handlers")
	}

	_ = r.On("Test", testhelpers.NoopEventHandler())
	if !r.HasHandlers() {
		t.Error("registry with On handler should have handlers")
	}
}
