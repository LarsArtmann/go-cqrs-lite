package sqliteengine_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Issue #62: a time.Time filter bound raw (driver-default rendering, space
// separator) lexicographically miscompares against the RFC3339 text the view
// JSON stores — Le/Lt leak the whole day, Ge/Gt drop it. Both repro legs
// assert the same-day inclusion the raw bind got wrong; they fail loudly if
// the coercion regresses. Leg 1 is the standard json_extract path (the shape
// the issue reports); leg 2 is the planned-table path (TEXT time columns).

type callItem struct {
	ID     string    `json:"id"`
	At     time.Time `json:"at"`
	Status string    `json:"status"`
}

func callItemQuery() metaengine.QueryDecl[struct{}, callItem] {
	return metaengine.Query[struct{}, callItem](
		"calls",
		metaengine.OnRecord(callItem{}, func(_ record.Record, c callItem) (string, callItem) {
			return c.ID, c
		}),
	)
}

func callItemPlannedQuery() metaengine.QueryDecl[struct{}, callItem] {
	return metaengine.Query[struct{}, callItem](
		"calls_planned",
		metaengine.OnRecord(callItem{}, func(_ record.Record, c callItem) (string, callItem) {
			return c.ID, c
		}),
		metaengine.FilterOnField[callItem]("at", metaengine.FilterGe),
		metaengine.FilterOnField[callItem]("status", metaengine.FilterEq),
	)
}

func setupCallStore(t *testing.T, planned bool) (*metaengine.Store, context.Context) {
	t.Helper()

	ctx := context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	q := callItemQuery()
	if planned {
		q = callItemPlannedQuery()
	}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	day := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	items := []callItem{
		{ID: "morning", At: day.Add(8 * time.Hour), Status: "open"},
		{ID: "evening", At: day.Add(20 * time.Hour), Status: "open"},
		{ID: "nextday", At: day.Add(26 * time.Hour), Status: "open"},
	}
	for _, item := range items {
		if err := store.Apply(ctx, "call", item); err != nil {
			t.Fatalf("Apply %s: %v", item.ID, err)
		}
	}

	return store, ctx
}

func assertSameDayBounds(t *testing.T, reader *metaengine.TypedReader[callItem], collection string) {
	t.Helper()

	ctx := context.Background()

	day := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	dayEnd := day.Add(24 * time.Hour)

	n, err := reader.Count(ctx, metaengine.WithFilter("at", metaengine.FilterGe, day))
	if err != nil {
		t.Fatalf("Count Ge: %v", err)
	}
	if n != 2 {
		t.Errorf("%s Ge(day) = %d, want 2 (same-day rows kept; the raw bind dropped the day)", collection, n)
	}

	n, err = reader.Count(ctx, metaengine.WithFilter("at", metaengine.FilterLt, dayEnd))
	if err != nil {
		t.Fatalf("Count Lt: %v", err)
	}
	if n != 2 {
		t.Errorf("%s Lt(dayEnd) = %d, want 2 (same-day rows kept; the raw bind leaked the day)", collection, n)
	}

	n, err = reader.Count(ctx, metaengine.WithFilter("at", metaengine.FilterGt, day.Add(-time.Second)))
	if err != nil {
		t.Fatalf("Count Gt: %v", err)
	}
	if n != 3 {
		t.Errorf("%s Gt(day-1s) = %d, want 3", collection, n)
	}

	n, err = reader.Count(ctx, metaengine.WithFilter("at", metaengine.FilterEq, day.Add(8*time.Hour)))
	if err != nil {
		t.Fatalf("Count Eq: %v", err)
	}
	if n != 1 {
		t.Errorf("%s Eq(morning) = %d, want 1 (bind must round-trip the stored rendering)", collection, n)
	}
}

// TestStandardFilter_SameDayTimeBounds is the standard-path repro
// (json_extract over the JSON view document, the shape the issue reports).
func TestStandardFilter_SameDayTimeBounds(t *testing.T) {
	t.Parallel()

	store, _ := setupCallStore(t, false)
	reader := metaengine.NewReader[callItem](store, "calls")

	assertSameDayBounds(t, reader, "standard")
}

// TestPlannedFilter_SameDayTimeBounds is the planned-table leg: the layout
// maps time columns to TEXT, so the write side and the filter bind must both
// speak RFC3339.
func TestPlannedFilter_SameDayTimeBounds(t *testing.T) {
	t.Parallel()

	store, _ := setupCallStore(t, true)
	reader := metaengine.NewReader[callItem](store, "calls_planned")

	assertSameDayBounds(t, reader, "planned")
}
