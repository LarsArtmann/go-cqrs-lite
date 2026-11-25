package metaengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestScanPageHonorsDefaultLimit pins that ScanPage judges "full page" by
// the limit Scan actually applied: the store default when no WithLimit is set.
func TestScanPageHonorsDefaultLimit(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		defaultLimit int
		rows         int
	}{
		{defaultLimit: 2, rows: 5},
		{defaultLimit: 150, rows: 120},
	} {
		store := storeWithTasks(t, tc.rows, metaengine.WithDefaultLimit(tc.defaultLimit))
		reader := metaengine.NewReader[FindTaskResult](store, "find_task")

		seen := map[TaskID]int{}

		var cursor *metaengine.Cursor

		for range tc.rows + 2 {
			opts := []metaengine.ScanOption{metaengine.WithSort("Title", false)}
			if cursor != nil {
				opts = append(opts, metaengine.WithCursorString(encodeCursor(t, cursor)))
			}

			page, next, err := reader.ScanPage(context.Background(), opts...)
			if err != nil {
				t.Fatalf("ScanPage: %v", err)
			}

			for _, row := range page {
				seen[row.ID]++
			}

			cursor = next
			if cursor == nil {
				break
			}
		}

		if len(seen) != tc.rows {
			t.Errorf("WithDefaultLimit(%d): walk reached %d distinct rows of %d",
				tc.defaultLimit, len(seen), tc.rows)
		}

		for id, n := range seen {
			if n != 1 {
				t.Errorf("WithDefaultLimit(%d): row %s served %d times", tc.defaultLimit, id, n)
			}
		}
	}
}

// --- fixtures ---

func storeWithTasks(t *testing.T, n int, planOpts ...any) *metaengine.Store {
	t.Helper()

	mem := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = mem.Close() })

	args := append([]any{}, planOpts...)
	args = append(args, findTaskQuery())

	store, err := metaengine.Plan([]metaengine.Engine{mem}, args...)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	mb := mem.(metaengine.MapBackend)

	for i := range n {
		id := TaskID(string(rune('a' + i)))

		err := mb.MapSet(context.Background(), "find_task", id, FindTaskResult{
			ID: id, Title: "t" + string(id), Status: []string{"open", "closed"}[i%2],
		})
		if err != nil {
			t.Fatalf("MapSet: %v", err)
		}
	}

	return store
}

func seededTaskReader(t *testing.T) (*metaengine.Store, *metaengine.TypedReader[FindTaskResult]) {
	t.Helper()

	store := storeWithTasks(t, 5)

	return store, metaengine.NewReader[FindTaskResult](store, "find_task")
}

func encodeCursor(t *testing.T, c *metaengine.Cursor) string {
	t.Helper()

	enc, err := c.Encode()
	if err != nil {
		t.Fatalf("Cursor.Encode: %v", err)
	}

	return enc
}
