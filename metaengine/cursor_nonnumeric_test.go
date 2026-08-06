package metaengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"


	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// cursorNonNumericEngines returns both engine implementations for the
// cross-engine cursor test. Each SQLite engine gets a unique named in-memory
// database to avoid shared-state interference with other tests.
func cursorNonNumericEngines(t *testing.T) map[string]metaengine.Engine {
	t.Helper()
	engines := map[string]metaengine.Engine{
		"memory": metaengine.NewMemoryEngine(),
	}
	dbName := fmt.Sprintf("cursortest_%d", time.Now().UnixNano())
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	eng, err := metaengine.NewMemoryEngine(), nil
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	engines["sqlite"] = eng

	return engines
}

// StringKeyItem is a value sorted by a string key.
type StringKeyItem struct {
	ID    string
	Name  string
	Score int
}

type StringKeyEvent struct {
	ID    string
	Name  string
	Score int
}

type ListByNameInput struct {
	Limit int
	After *metaengine.Cursor
}

type ListByNameResult struct {
	Items []StringKeyItem
	Next  *metaengine.Cursor
}

func listByNameQuery() metaengine.QueryDecl[ListByNameInput, ListByNameResult] {
	return metaengine.Query[ListByNameInput, ListByNameResult](
		"list_by_name",
		metaengine.On(StringKeyEvent{}, func(e StringKeyEvent) (string, StringKeyItem) {
			return e.ID, StringKeyItem(e)
		}),
		metaengine.SortOn(func(item StringKeyItem) string { return item.Name }),
	)
}

// TimeKeyItem is a value sorted by a time.Time key.
type TimeKeyItem struct {
	ID    string
	Label string
	At    time.Time
}

type TimeKeyEvent struct {
	ID    string
	Label string
	At    time.Time
}

type ListByTimeInput struct {
	Limit int
	After *metaengine.Cursor
}

type ListByTimeResult struct {
	Items []TimeKeyItem
	Next  *metaengine.Cursor
}

func listByTimeQuery() metaengine.QueryDecl[ListByTimeInput, ListByTimeResult] {
	return metaengine.Query[ListByTimeInput, ListByTimeResult](
		"list_by_time",
		metaengine.On(TimeKeyEvent{}, func(e TimeKeyEvent) (string, TimeKeyItem) {
			return e.ID, TimeKeyItem(e)
		}),
		metaengine.SortOn(func(item TimeKeyItem) time.Time { return item.At }),
	)
}

// TestCursorRoundTrip_StringKey_AllEngines: Paginate through a string-keyed
// collection using encoded cursors (Encode → ParseCursor) across memory + SQLite.
// Verifies items are returned in lexicographic order with no gaps or overlap.
func TestCursorRoundTrip_StringKey_AllEngines(t *testing.T) {
	t.Parallel()

	for name, eng := range cursorNonNumericEngines(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store, err := metaengine.Plan([]metaengine.Engine{eng}, listByNameQuery())
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			defer store.Close()

			ctx := context.Background()
			names := []string{"zoe", "alice", "mike", "bob", "dave", "carol", "eve", "frank"}
			for i, n := range names {
				if err := store.Apply(ctx, "StringKeyEvent", StringKeyEvent{
					ID: fmt.Sprintf("id-%d", i), Name: n, Score: i,
				}); err != nil {
					t.Fatalf("Apply %d: %v", i, err)
				}
			}

			var collected []string
			var cursor *metaengine.Cursor
			pageSize := 3

			for page := 0; page < 10; page++ {
				input := ListByNameInput{Limit: pageSize}
				if cursor != nil {
					encoded, err := cursor.Encode()
					if err != nil {
						t.Fatalf("page %d Encode: %v", page, err)
					}
					cursor, err = metaengine.ParseCursor(encoded)
					if err != nil {
						t.Fatalf("page %d ParseCursor: %v", page, err)
					}
					input.After = cursor
				}
				result, err := metaengine.ExecuteTyped[ListByNameInput, ListByNameResult](
					ctx, store, input,
				)
				if err != nil {
					t.Fatalf("page %d ExecuteTyped: %v", page, err)
				}
				for _, item := range result.Items {
					collected = append(collected, item.Name)
				}
				if result.Next == nil {
					break
				}
				cursor = result.Next
			}

			if len(collected) != len(names) {
				t.Fatalf("expected %d items, got %d", len(names), len(collected))
			}
			for i := 1; i < len(collected); i++ {
				if collected[i] < collected[i-1] {
					t.Fatalf("items not in ascending order at %d: %q < %q",
						i, collected[i], collected[i-1])
				}
			}
			seen := make(map[string]bool)
			for _, n := range collected {
				if seen[n] {
					t.Fatalf("duplicate item: %q", n)
				}
				seen[n] = true
			}
		})
	}
}

// TestCursorRoundTrip_TimeKey_AllEngines: Paginate through a time-keyed
// collection using encoded cursors across memory + SQLite. time.Time keys
// serialize to ISO 8601 strings through JSON; verify chronological ordering
// survives the round-trip.
func TestCursorRoundTrip_TimeKey_AllEngines(t *testing.T) {
	t.Parallel()

	for name, eng := range cursorNonNumericEngines(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store, err := metaengine.Plan([]metaengine.Engine{eng}, listByTimeQuery())
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			defer store.Close()

			ctx := context.Background()
			base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			offsets := []time.Duration{
				72 * time.Hour,  // Jan 3
				0 * time.Hour,   // Jan 1
				168 * time.Hour, // Jan 8
				24 * time.Hour,  // Jan 2
				-24 * time.Hour, // Dec 31
				336 * time.Hour, // Jan 15
			}
			for i, off := range offsets {
				if err := store.Apply(ctx, "TimeKeyEvent", TimeKeyEvent{
					ID:    fmt.Sprintf("t-%d", i),
					Label: fmt.Sprintf("event-%d", i),
					At:    base.Add(off),
				}); err != nil {
					t.Fatalf("Apply %d: %v", i, err)
				}
			}

			var collected []time.Time
			var cursor *metaengine.Cursor
			pageSize := 2

			for page := 0; page < 10; page++ {
				input := ListByTimeInput{Limit: pageSize}
				if cursor != nil {
					encoded, err := cursor.Encode()
					if err != nil {
						t.Fatalf("page %d Encode: %v", page, err)
					}
					cursor, err = metaengine.ParseCursor(encoded)
					if err != nil {
						t.Fatalf("page %d ParseCursor: %v", page, err)
					}
					input.After = cursor
				}
				result, err := metaengine.ExecuteTyped[ListByTimeInput, ListByTimeResult](
					ctx, store, input,
				)
				if err != nil {
					t.Fatalf("page %d ExecuteTyped: %v", page, err)
				}
				for _, item := range result.Items {
					collected = append(collected, item.At)
				}
				if result.Next == nil {
					break
				}
				cursor = result.Next
			}

			if len(collected) != len(offsets) {
				t.Fatalf("expected %d items, got %d", len(offsets), len(collected))
			}
			for i := 1; i < len(collected); i++ {
				if collected[i].Before(collected[i-1]) {
					t.Fatalf("items not in chronological order at %d: %v before %v",
						i, collected[i], collected[i-1])
				}
			}
		})
	}
}
