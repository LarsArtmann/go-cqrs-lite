package event

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestNewEvents(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	types := []Type{"user.created", "user.activated"}
	payloads := []any{
		struct{ Name string }{Name: "Alice"},
		struct{ At string }{At: "now"},
	}

	events, err := newEvents(aggID, "User", 0, types, payloads)
	if err != nil {
		t.Fatalf("newEvents() error = %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Type() != "user.created" {
		t.Errorf("event[0] type = %q, want %q", events[0].Type(), "user.created")
	}

	if events[0].Version() != 1 {
		t.Errorf("event[0] version = %d, want 1", events[0].Version())
	}

	if events[1].Version() != 2 {
		t.Errorf("event[1] version = %d, want 2", events[1].Version())
	}
}

func TestNewEvents_MismatchedSlices(t *testing.T) {
	t.Parallel()

	_, err := newEvents(id.NewAggregateID(), "User", 0,
		[]Type{"a"}, []any{"x", "y"})

	if err == nil {
		t.Fatal("expected error for mismatched slices")
	}
}

func TestNewEvents_WithExpectedVersion(t *testing.T) {
	t.Parallel()

	events, err := newEvents(id.NewAggregateID(), "User", 5,
		[]Type{"user.updated"}, []any{map[string]string{"k": "v"}})
	if err != nil {
		t.Fatalf("newEvents() error = %v", err)
	}

	if events[0].Version() != 6 {
		t.Errorf("version = %d, want 6", events[0].Version())
	}
}

func TestMustNewEvents(t *testing.T) {
	t.Parallel()

	t.Run("valid inputs", func(t *testing.T) {
		t.Parallel()

		events := mustNewEvents(id.NewAggregateID(), "User", 0,
			[]Type{"x"}, []any{map[string]string{}})

		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}
	})

	t.Run("invalid inputs panic", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		mustNewEvents(id.AggregateID{}, "", 0,
			[]Type{"x"}, []any{"y"})
	})
}

func TestDecodePayloads(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}

	p := payload{Name: "Bob"}
	events, err := newEvents(id.NewAggregateID(), "User", 0,
		[]Type{"user.created"}, []any{p})
	if err != nil {
		t.Fatalf("newEvents() error = %v", err)
	}

	results, err := decodePayloads[payload](events, JSONCodec{})
	if err != nil {
		t.Fatalf("decodePayloads() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Name != "Bob" {
		t.Errorf("name = %q, want %q", results[0].Name, "Bob")
	}
}

func TestDecodePayloads_InvalidJSON(t *testing.T) {
	t.Parallel()

	evt, _ := NewEvent("bad", id.NewAggregateID(), "User", 1, []byte("not json"))

	_, err := decodePayloads[struct{}]([]Event{evt}, JSONCodec{})

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestNewEvents_EncodeError(t *testing.T) {
	t.Parallel()

	_, err := newEvents(id.NewAggregateID(), "User", 0,
		[]Type{"x"}, []any{func() {}})

	if err == nil {
		t.Fatal("expected error for unmarshallable payload")
	}
}

func TestNewEvents_Validation(t *testing.T) {
	t.Parallel()

	_, err := newEvents(id.AggregateID{}, "User", 0,
		[]Type{"x"}, []any{"y"})

	if err == nil {
		t.Fatal("expected error for zero aggregate ID")
	}

	if !errors.Is(err, ErrNilAggregateID) {
		t.Errorf("error = %v, want ErrNilAggregateID", err)
	}
}
