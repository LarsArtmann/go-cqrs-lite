package metaengine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Fixtures for the compound-cursor issuance protocol (T07): ScanPage must
// return the engine-issued SortKeyCursor so tie-heavy datasets paginate
// exactly once, and ParseCursor must normalize the encoded compound shape
// back into a SortKeyCursor on the string round-trip.

type tieItem struct {
	ID       string
	Status   string
	Priority int
}

type tieListInput struct {
	Status string
}

func tieScanQuery() metaengine.QueryDecl[tieListInput, tieItem] {
	return metaengine.Query[tieListInput, tieItem](
		"tie_scan",
		metaengine.OnRecord(
			tieItem{},
			func(_ record.Record, e tieItem) (string, tieItem) { return e.ID, e },
		),
		metaengine.FilterOnField[tieItem]("Status", metaengine.FilterEq),
		metaengine.SortOnField[tieItem]("Priority", true),
	)
}

// setupTieScan seeds n items whose Priority is i%3 — 3-item tie blocks that
// any limit not divisible by 3 straddles across page boundaries.
func setupTieScan(t *testing.T, n int) *metaengine.TypedReader[tieItem] {
	t.Helper()

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, tieScanQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	for i := range n {
		item := tieItem{
			ID:       fmt.Sprintf("item-%03d", i),
			Status:   "open",
			Priority: i % 3,
		}

		if err := store.Apply(ctx, "tieItem", item); err != nil {
			t.Fatalf("seed Apply[%d]: %v", i, err)
		}
	}

	return metaengine.NewReader[tieItem](store, "tie_scan")
}

// walkPages drives ScanPage to exhaustion, passing each cursor back in the
// given mode (raw Value or encoded string), and returns every visited ID.
func walkPages(
	t *testing.T,
	reader *metaengine.TypedReader[tieItem],
	limit int,
	asString bool,
) []string {
	t.Helper()

	ctx := context.Background()

	var visited []string

	next := ""

	for range 50 { //nolint:intrange // fixed safety bound, not iteration count
		opts := []metaengine.ScanOption{
			metaengine.WithSort("Priority", false),
			metaengine.WithLimit(limit),
		}

		if next != "" {
			opts = append(opts, metaengine.WithCursorString(next))
		} else if asString && len(visited) > 0 {
			break
		}

		items, cursor, err := reader.ScanPage(ctx, opts...)
		if err != nil {
			t.Fatalf("ScanPage: %v", err)
		}

		for _, item := range items {
			visited = append(visited, item.ID)
		}

		if cursor == nil {
			break
		}

		encoded, err := cursor.Encode()
		if err != nil {
			t.Fatalf("cursor Encode: %v", err)
		}

		if asString {
			next = encoded
		} else {
			opts = append(opts, metaengine.WithCursor(cursor.Value))
			next = encoded
		}
	}

	return visited
}
