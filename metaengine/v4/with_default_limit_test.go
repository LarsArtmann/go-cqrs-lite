package metaengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestWithDefaultLimit_CeilingsUnlimitedScans pins the operator ceiling
// (G-T14 v4 half, ruling 2026-09-21: option C): WithDefaultLimit(n) re-pins
// the store-wide scan bound for scans that carry no explicit WithLimit; an
// explicit WithLimit always wins; stores without the option keep the
// built-in 100.
func TestWithDefaultLimit_CeilingsUnlimitedScans(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	newStore := func(opts ...any) *metaengine.Store {
		t.Helper()

		mem := metaengine.NewMemoryEngine()
		t.Cleanup(func() { _ = mem.Close() })

		args := make([]any, 0, len(opts)+1)
		args = append(args, opts...)
		args = append(args, findTaskQuery())

		store, err := metaengine.Plan([]metaengine.Engine{mem}, args...)
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}

		t.Cleanup(func() { _ = store.Close() })

		mb, ok := mem.(metaengine.MapBackend)
		if !ok {
			t.Fatal("memory engine must implement MapBackend")
		}

		for i := range 5 {
			id := TaskID(string(rune('a' + i)))

			if err := mb.MapSet(ctx, "find_task", id,
				TaskCreated{ID: id, Title: "t", Status: "open"},
			); err != nil {
				t.Fatalf("MapSet: %v", err)
			}
		}

		return store
	}

	ceilinged := newStore(metaengine.WithDefaultLimit(2))
	reader := metaengine.NewReader[FindTaskResult](ceilinged, "find_task")

	rows, err := reader.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("ceilinged scan = %d rows, want 2 (operator ceiling)", len(rows))
	}

	rows, err = reader.Scan(ctx, metaengine.WithLimit(4))
	if err != nil {
		t.Fatalf("Scan(WithLimit): %v", err)
	}

	if len(rows) != 4 {
		t.Fatalf("explicit WithLimit(4) = %d rows, want 4 (explicit wins)", len(rows))
	}

	plain := newStore()
	plainReader := metaengine.NewReader[FindTaskResult](plain, "find_task")

	rows, err = plainReader.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(rows) != 5 {
		t.Fatalf("unceilinged scan = %d rows, want 5 (built-in 100 untouched)", len(rows))
	}
}
