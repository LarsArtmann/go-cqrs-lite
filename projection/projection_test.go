package projection

import (
	"context"
	"errors"
	"slices"
	"testing"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestNewProjection_NameAndEventTypes(t *testing.T) {
	t.Parallel()

	types := []cqrsevent.Type{"USER_CREATED", "USER_UPDATED"}
	proj := NewProjection(
		"users",
		func(context.Context, cqrsevent.Event) error { return nil },
		types,
	)

	if proj.Name() != "users" {
		t.Fatalf("name = %q, want %q", proj.Name(), "users")
	}

	got := proj.EventTypes()
	if len(got) != 2 || string(got[0]) != "USER_CREATED" || string(got[1]) != "USER_UPDATED" {
		t.Fatalf("types = %v", got)
	}
}

func TestNewProjection_EventTypesReturnsClone(t *testing.T) {
	t.Parallel()

	original := []cqrsevent.Type{"A", "B"}
	proj := NewProjection(
		"p",
		func(context.Context, cqrsevent.Event) error { return nil },
		original,
	)

	mutated := proj.EventTypes()
	mutated[0] = "HACKED"

	if string(original[0]) == "HACKED" {
		t.Fatal("EventTypes() returned the backing slice, not a clone")
	}

	again := proj.EventTypes()
	if again[0] == "HACKED" {
		t.Fatal("second EventTypes() call returned mutated data")
	}
}

func TestNewProjection_HandleDispatchesToHandler(t *testing.T) {
	t.Parallel()

	called := false
	proj := NewProjection("p", func(_ context.Context, evt cqrsevent.Event) error {
		called = true
		if string(evt.Type()) != "TEST" {
			t.Fatalf("event type = %q", evt.Type())
		}

		return nil
	}, nil)

	evt := testEvent(t, "TEST")
	if err := proj.Handle(context.Background(), evt); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if !called {
		t.Fatal("handler was not called")
	}
}

func TestNewProjection_HandlePropagatesError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	proj := NewProjection("p", func(context.Context, cqrsevent.Event) error { return wantErr }, nil)

	err := proj.Handle(context.Background(), testEvent(t, "X"))
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func TestNewProjection_EmptyEventTypes(t *testing.T) {
	t.Parallel()

	proj := NewProjection("p", func(context.Context, cqrsevent.Event) error { return nil }, nil)
	got := proj.EventTypes()

	if len(got) != 0 {
		t.Fatalf("empty types = %v, want empty", got)
	}
}

func testEvent(t *testing.T, eventType string) cqrsevent.Event {
	t.Helper()

	streamID, err := id.ParseStreamID("test-agg")
	if err != nil {
		t.Fatalf("parse agg id: %v", err)
	}

	evt, err := cqrsevent.NewEvent(
		cqrsevent.Type(eventType),
		streamID,
		"Test",
		cqrsevent.Version(1),
		[]byte("{}"),
	)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	return evt
}

// Compile-time: projectionFunc satisfies Projection (already asserted in
// production code, but keeping here ensures the test package sees it too).
var _ Projection = (*projectionFunc)(nil)

// Ensure slices is used (import guard — we use slices.Clone semantics).
var _ = slices.Clone[[]int]
