package metaengine

import (
	"reflect"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestReifyReflect_MapToTypedStruct(t *testing.T) {
	t.Parallel()

	type orderView struct {
		ID     string
		Status string
		Total  int
	}

	target := reflect.TypeFor[orderView]()

	// Simulate what SQL engines return: map[string]any from JSON decode.
	sqlPrev := map[string]any{
		"ID":     "order-42",
		"Status": "shipped",
		"Total":  float64(99),
	}

	got := reifyReflect(sqlPrev, target)
	if !got.IsValid() {
		t.Fatal("reifyReflect returned invalid Value for map input")
	}

	result := got.Interface().(orderView)
	if result.ID != "order-42" {
		t.Errorf("ID = %q, want %q", result.ID, "order-42")
	}
	if result.Status != "shipped" {
		t.Errorf("Status = %q, want %q", result.Status, "shipped")
	}
}

func TestReifyReflect_TypedValueFastPath(t *testing.T) {
	t.Parallel()

	type view struct{ Name string }
	target := reflect.TypeFor[view]()

	// Memory engines return typed values directly — fast path, no JSON round-trip.
	original := view{Name: "Alice"}
	got := reifyReflect(original, target)

	result := got.Interface().(view)
	if result.Name != "Alice" {
		t.Errorf("Name = %q, want %q", result.Name, "Alice")
	}
}

func TestReifyReflect_NilFallsBackToZero(t *testing.T) {
	t.Parallel()

	type view struct{ Name string }
	target := reflect.TypeFor[view]()

	got := reifyReflect(nil, target)
	if !got.IsValid() {
		t.Fatal("reifyReflect(nil) returned invalid Value")
	}

	result := got.Interface().(view)
	if result.Name != "" {
		t.Errorf("expected zero value, got Name=%q", result.Name)
	}
}

// TestOnRecord_UpdateFold_ReifyMapPrev is the direct regression test for the
// fixed bug in record_fold.go:115. Before the fix, the OnRecord update path
// passed prev directly to reflect.ValueOf(prev), which panics when prev is
// map[string]any (as returned by SQL engines). The fix calls reifyReflect to
// rebuild the typed value via JSON round-trip.
func TestOnRecord_UpdateFold_ReifyMapPrev(t *testing.T) {
	t.Parallel()

	type evt struct {
		ID     string
		Status string
	}
	type view struct {
		ID     string
		Status string
	}

	fold := OnRecordTyped("test_update", evt{}, func(
		_ record.Record,
		e evt,
		prev view,
	) view {
		prev.Status = e.Status
		return prev
	})

	uf, ok := fold.(*updateFold)
	if !ok {
		t.Fatalf("expected *updateFold, got %T", fold)
	}

	// Simulate SQL engine returning map[string]any for prev.
	sqlPrev := map[string]any{
		"ID":     "order-42",
		"Status": "pending",
	}

	eventPayload := evt{ID: "order-42", Status: "shipped"}

	// This must not panic — before the fix it did.
	result := uf.invoke(eventPayload, sqlPrev)

	got, ok := result.(view)
	if !ok {
		t.Fatalf("expected view, got %T", result)
	}
	if got.ID != "order-42" {
		t.Errorf("ID = %q, want %q", got.ID, "order-42")
	}
	if got.Status != "shipped" {
		t.Errorf("Status = %q, want %q", got.Status, "shipped")
	}
}

// TestOnRecord_UpdateFold_NilPrev uses zero value.
func TestOnRecord_UpdateFold_NilPrev(t *testing.T) {
	t.Parallel()

	type evt struct{ ID string }
	type view struct{ ID, Status string }

	fold := OnRecordTyped("test_nil_prev", evt{}, func(
		_ record.Record,
		e evt,
		prev view,
	) view {
		prev.ID = e.ID
		prev.Status = "created"
		return prev
	})

	uf, ok := fold.(*updateFold)
	if !ok {
		t.Fatalf("expected *updateFold, got %T", fold)
	}

	result := uf.invoke(evt{ID: "new-1"}, nil)

	got, ok := result.(view)
	if !ok {
		t.Fatalf("expected view, got %T", result)
	}
	if got.ID != "new-1" {
		t.Errorf("ID = %q, want %q", got.ID, "new-1")
	}
	if got.Status != "created" {
		t.Errorf("Status = %q, want %q", got.Status, "created")
	}
}
