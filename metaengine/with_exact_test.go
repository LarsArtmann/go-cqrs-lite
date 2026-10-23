package metaengine_test

import (
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestWithExact_MakesTruncationLoud pins issue #71: a bounded scan used to be
// unable to say whether it returned the whole result or only the first N rows.
// WithExact turns truncation into ErrScanTruncated; an exact fit and an
// unbounded scan stay silent.
func TestWithExact_MakesTruncationLoud(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	mem := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = mem.Close() })

	store, err := metaengine.Plan([]metaengine.Engine{mem}, findTaskQuery())
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

	reader := metaengine.NewReader[FindTaskResult](store, "find_task")

	rows, err := reader.Scan(ctx, metaengine.WithLimit(3), metaengine.WithExact())
	if !errors.Is(err, metaengine.ErrScanTruncated) {
		t.Fatalf("truncated exact scan err = %v, want ErrScanTruncated", err)
	}

	if rows != nil {
		t.Fatalf("truncated exact scan returned %d rows, want nil", len(rows))
	}

	rows, err = reader.Scan(ctx, metaengine.WithLimit(5), metaengine.WithExact())
	if err != nil {
		t.Fatalf("exact-fit scan err = %v, want nil", err)
	}

	if len(rows) != 5 {
		t.Fatalf("exact-fit scan returned %d rows, want 5", len(rows))
	}

	rows, err = reader.Scan(ctx, metaengine.WithLimit(0), metaengine.WithExact())
	if err != nil {
		t.Fatalf("unbounded exact scan err = %v, want nil", err)
	}

	if len(rows) != 5 {
		t.Fatalf("unbounded exact scan returned %d rows, want 5", len(rows))
	}
}
