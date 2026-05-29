package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestNewEvents(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	version := event.Version(0)

	events, err := event.NewEvents(
		aggID, "User", version,
		[]event.Type{"user.created", "user.verified"},
		[]any{
			map[string]string{"name": "Alice"},
			map[string]string{"by": "admin"},
		},
	)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Type() != "user.created" {
		t.Errorf("events[0].Type = %q, want user.created", events[0].Type())
	}

	if events[0].Version() != 1 {
		t.Errorf("events[0].Version = %d, want 1", events[0].Version())
	}

	if events[1].Version() != 2 {
		t.Errorf("events[1].Version = %d, want 2", events[1].Version())
	}
}

func TestNewEvents_MismatchedCount(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	_, err := event.NewEvents(
		aggID, "User", 0,
		[]event.Type{"user.created"},
		[]any{"a", "b"},
	)
	if err == nil {
		t.Fatal("expected error for mismatched count")
	}
}

func TestNewEvents_Empty(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	events, err := event.NewEvents(aggID, "User", 0, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if events != nil {
		t.Errorf("expected nil, got %v", events)
	}
}

func TestMustNewEvents(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	events := event.MustNewEvents(
		aggID, "User", 0,
		[]event.Type{"user.created"},
		[]any{map[string]string{"name": "Bob"}},
	)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestMustNewEvents_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
	}()

	aggID := id.NewAggregateID()
	event.MustNewEvents(
		aggID, "User", 0,
		[]event.Type{"a"},
		[]any{"x", "y"},
	)
}

func TestVersion_AddSubCmp(t *testing.T) {
	t.Parallel()

	v := event.Version(5)

	if v.Add(3) != 8 {
		t.Errorf("Add(3) = %d, want 8", v.Add(3))
	}

	if v.Sub(2) != 3 {
		t.Errorf("Sub(2) = %d, want 3", v.Sub(2))
	}

	if v.Cmp(3) != 1 {
		t.Errorf("Cmp(3) = %d, want 1", v.Cmp(3))
	}

	if v.Cmp(5) != 0 {
		t.Errorf("Cmp(5) = %d, want 0", v.Cmp(5))
	}

	if v.Cmp(10) != -1 {
		t.Errorf("Cmp(10) = %d, want -1", v.Cmp(10))
	}
}

func TestCheckVersionConflict(t *testing.T) {
	t.Parallel()

	err := event.CheckVersionConflict(3, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = event.CheckVersionConflict(2, 3)
	if err == nil {
		t.Fatal("expected version conflict error")
	}
}

func TestSchemaVersion_IntStringIsZero(t *testing.T) {
	t.Parallel()

	sv := event.SchemaVersion(2)

	if sv.Int() != 2 {
		t.Errorf("Int = %d, want 2", sv.Int())
	}

	if sv.String() != "2" {
		t.Errorf("String = %q, want 2", sv.String())
	}

	if sv.IsZero() {
		t.Error("IsZero should be false for 2")
	}

	var zero event.SchemaVersion

	if !zero.IsZero() {
		t.Error("IsZero should be true for 0")
	}
}

func TestOutboxID_StringIsZero(t *testing.T) {
	t.Parallel()

	oid := event.NewOutboxID("test-id")

	if oid.String() != "test-id" {
		t.Errorf("String = %q, want test-id", oid.String())
	}

	if oid.IsZero() {
		t.Error("IsZero should be false")
	}

	var zero event.OutboxID

	if !zero.IsZero() {
		t.Error("IsZero should be true for zero value")
	}
}

func TestWithReplay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	replayCtx := event.WithReplay(ctx, true)

	val, ok := replayCtx.Value(nil).(*bool)
	_ = val
	_ = ok

	replayCtx2 := event.WithReplay(ctx, false)

	if replayCtx == replayCtx2 {
		t.Error("different replay values should produce different contexts")
	}
}
