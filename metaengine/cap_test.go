package metaengine_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type capEvent struct {
	ID  string
	Seq int
}

type capItem struct {
	ID  string
	Seq int
}

func cappedQuery(maxRows int, orderBy string) metaengine.QueryDecl[struct{}, capItem] {
	return metaengine.Query[struct{}, capItem]("recent_items",
		metaengine.OnRecord(capEvent{}, func(_ record.Record, e capEvent) (string, capItem) {
			return e.ID, capItem{ID: e.ID, Seq: e.Seq}
		}),
		metaengine.WithCap(maxRows, orderBy),
	)
}

func newCappedStore(t *testing.T, query any) *metaengine.Store {
	t.Helper()

	mem := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = mem.Close() })

	store, err := metaengine.Plan([]metaengine.Engine{mem}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store
}

func seedCapEvents(t *testing.T, store *metaengine.Store, n int) {
	t.Helper()

	ctx := t.Context()

	for i := 1; i <= n; i++ {
		if err := store.Apply(
			ctx,
			"capEvent",
			capEvent{ID: fmt.Sprintf("e%02d", i), Seq: i},
		); err != nil {
			t.Fatalf("Apply %d: %v", i, err)
		}
	}
}

// TestWithCap_EnforceCapKeepsNewestRows pins issue #72: a keep-newest-N policy
// is declared once on the query and enforced by the store, so consumers no
// longer hand-roll a converging check-then-expunge loop.
func TestWithCap_EnforceCapKeepsNewestRows(t *testing.T) {
	t.Parallel()

	store := newCappedStore(t, cappedQuery(3, "Seq"))
	seedCapEvents(t, store, 6)

	removed, err := store.EnforceCap(t.Context(), "recent_items")
	if err != nil {
		t.Fatalf("EnforceCap: %v", err)
	}

	if removed != 3 {
		t.Fatalf("EnforceCap removed %d rows, want 3", removed)
	}

	reader := metaengine.NewReader[capItem](store, "recent_items")

	rows, err := reader.Scan(
		t.Context(),
		metaengine.WithLimit(0),
		metaengine.WithSort("Seq", false),
	)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("after trim: %d rows, want 3", len(rows))
	}

	for i, row := range rows {
		want := i + 4
		if row.Seq != want {
			t.Errorf("row %d Seq = %d, want %d (newest three survive)", i, row.Seq, want)
		}
	}
}

// A collection already within its cap is a no-op, and a second pass after a
// trim removes nothing.
func TestWithCap_EnforceCapIsIdempotent(t *testing.T) {
	t.Parallel()

	store := newCappedStore(t, cappedQuery(5, "Seq"))
	seedCapEvents(t, store, 4)

	removed, err := store.EnforceCap(t.Context(), "recent_items")
	if err != nil {
		t.Fatalf("EnforceCap under cap: %v", err)
	}

	if removed != 0 {
		t.Fatalf("under-cap EnforceCap removed %d rows, want 0", removed)
	}

	seedCapEvents(t, store, 2)

	if _, err := store.EnforceCap(t.Context(), "recent_items"); err != nil {
		t.Fatalf("EnforceCap: %v", err)
	}

	removed, err = store.EnforceCap(t.Context(), "recent_items")
	if err != nil {
		t.Fatalf("second EnforceCap: %v", err)
	}

	if removed != 0 {
		t.Fatalf("second EnforceCap removed %d rows, want 0", removed)
	}
}

// EnforceCaps trims every capped collection and leaves uncapped ones alone.
func TestWithCap_EnforceCapsOnlyTrimsCappedCollections(t *testing.T) {
	t.Parallel()

	store := newCappedStore(t, cappedQuery(2, "Seq"))
	seedCapEvents(t, store, 5)

	removed, err := store.EnforceCaps(t.Context())
	if err != nil {
		t.Fatalf("EnforceCaps: %v", err)
	}

	if removed != 3 {
		t.Fatalf("EnforceCaps removed %d rows, want 3", removed)
	}

	// An uncapped query must report 0 rather than an error.
	uncapped := metaengine.Query[struct{}, capItem]("all_items",
		metaengine.OnRecord(capEvent{}, func(_ record.Record, e capEvent) (string, capItem) {
			return e.ID, capItem{ID: e.ID, Seq: e.Seq}
		}),
	)

	other := newCappedStore(t, uncapped)

	removed, err = other.EnforceCap(t.Context(), "all_items")
	if err != nil {
		t.Fatalf("EnforceCap on uncapped query: %v", err)
	}

	if removed != 0 {
		t.Fatalf("uncapped EnforceCap removed %d rows, want 0", removed)
	}
}

// WithCap without an order-by column is rejected at Plan time, not silently
// ignored.
func TestWithCap_RequiresOrderBy(t *testing.T) {
	t.Parallel()

	mem := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = mem.Close() })

	_, err := metaengine.Plan([]metaengine.Engine{mem}, cappedQuery(3, ""))
	if err == nil {
		t.Fatal("Plan with empty orderBy succeeded, want an error")
	}

	if !strings.Contains(err.Error(), "orderBy") {
		t.Fatalf("Plan error = %v, want it to name the missing orderBy column", err)
	}
}

// WithCap is a Map-only policy.
func TestWithCap_RejectsNonMapADT(t *testing.T) {
	t.Parallel()

	mem := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = mem.Close() })

	counter := metaengine.Query[struct{}, int]("counts",
		metaengine.OnRecord(capEvent{}, func(_ record.Record, _ capEvent) metaengine.Delta {
			return metaengine.Delta{"count": 1}
		}),
		metaengine.WithCap(10, "Seq"),
	)

	_, err := metaengine.Plan([]metaengine.Engine{mem}, counter)
	if !errors.Is(err, metaengine.ErrCapRequiresMap) {
		t.Fatalf("Plan with non-Map cap err = %v, want ErrCapRequiresMap", err)
	}
}
