package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
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

	eventtest.AssertEventType(t, events, 0, "user.created")
	eventtest.AssertEventVersion(t, events, 0, 1)
	eventtest.AssertEventVersion(t, events, 1, 2)
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

func TestNewEvents_Batch(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	events, err := event.NewEvents(
		aggID, "User", 0,
		[]event.Type{"user.created"},
		[]any{map[string]string{"name": "Bob"}},
	)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
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

func TestSchemaVersion_AddSub(t *testing.T) {
	t.Parallel()

	sv := event.SchemaVersion(3)

	if sv.Add(2) != 5 {
		t.Errorf("Add(2) = %d, want 5", sv.Add(2))
	}

	if sv.Sub(1) != 2 {
		t.Errorf("Sub(1) = %d, want 2", sv.Sub(1))
	}
}

func TestSchemaVersion_Add_PanicsOnUnderflow(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for Add that would produce non-positive result")
		}
	}()

	event.SchemaVersion(1).Add(-2)
}

func TestSchemaVersion_Sub_PanicsOnUnderflow(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for Sub that would produce non-positive result")
		}
	}()

	event.SchemaVersion(1).Sub(1)
}

func TestVersion_JSON(t *testing.T) {
	t.Parallel()

	v := event.Version(42)

	b, err := v.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	if string(b) != "42" {
		t.Errorf("MarshalJSON = %s, want 42", b)
	}

	var got event.Version
	if err := got.UnmarshalJSON(b); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if got != v {
		t.Errorf("UnmarshalJSON = %d, want %d", got, v)
	}
}

func TestSchemaVersion_JSON(t *testing.T) {
	t.Parallel()

	sv := event.SchemaVersion(3)

	b, err := sv.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	if string(b) != "3" {
		t.Errorf("MarshalJSON = %s, want 3", b)
	}

	var got event.SchemaVersion
	if err := got.UnmarshalJSON(b); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if got != sv {
		t.Errorf("UnmarshalJSON = %d, want %d", got, sv)
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

func TestIsReplay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	if event.IsReplay(ctx) {
		t.Error("background context should not be replay")
	}

	if event.IsReplay(event.WithReplay(ctx, false)) {
		t.Error("WithReplay(false) should not be replay")
	}

	if !event.IsReplay(event.WithReplay(ctx, true)) {
		t.Error("WithReplay(true) should be replay")
	}
}
