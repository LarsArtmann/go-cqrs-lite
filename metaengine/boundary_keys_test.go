package metaengine

import (
	"context"
	"errors"
	"testing"
)

// TestStore_BoundaryKeyTypeValidation verifies that Execute returns
// ErrKeyTypeMismatch when the input struct has no field matching the query's
// declared key type (derived from the fold). This catches the footgun where
// the fold's key extraction type doesn't match any field in the input struct.
func TestStore_BoundaryKeyTypeValidation(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)
	defer store.Close()

	// testFindTask has a testTaskID field matching the fold's key type — valid.
	// It's OK if the key doesn't exist (ErrNotFound), we just verify no
	// ErrKeyTypeMismatch.
	_, err := ExecuteTyped[testFindTask, testTask](
		context.Background(), store, testFindTask{ID: "valid"},
	)
	if errors.Is(err, ErrKeyTypeMismatch) {
		t.Fatalf("valid input should not trigger ErrKeyTypeMismatch: %v", err)
	}
}

// TestStore_BoundaryKeyTypeMismatch creates a query where the fold extracts
// int keys but the input struct has no int field. Execute must return
// ErrKeyTypeMismatch.
func TestStore_BoundaryKeyTypeMismatch(t *testing.T) {
	t.Parallel()

	type event struct {
		ID int
	}

	type input struct {
		Name string // no int field — mismatch with fold's int key
	}

	q := Query[input, bool](
		"mismatch_test",
		On(event{}, func(e event) (int, bool) {
			return e.ID, true
		}),
	)

	eng := NewMemoryEngine()
	store, err := Plan([]Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	// Pre-populate so the key exists.
	_ = store.Apply(context.Background(), "event", event{ID: 42})

	// Execute with the mismatched input — no int field to extract the key from.
	_, err = store.ExecuteCtx(context.Background(), input{Name: "no-int-field"})
	if !errors.Is(err, ErrKeyTypeMismatch) {
		t.Fatalf("expected ErrKeyTypeMismatch, got: %v", err)
	}
}
