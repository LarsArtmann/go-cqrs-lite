package querytest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// StoreSuite combines [query.QueryStore] (sink + source) with
// [query.SeekableQueryJournal] (global journal reads). Every backend's query
// store implementation satisfies this interface.
type StoreSuite interface {
	query.QueryStore
	query.SeekableQueryJournal
}

// StoreFactory creates a fresh store for each subtest. Each call must produce
// an independent store with its own backing data.
type StoreFactory func(t *testing.T) StoreSuite

// MustCreateQuery creates a [query.PersistedQuery] for testing. It fails the
// test on error.
func MustCreateQuery(t *testing.T, queryType string) *query.PersistedQuery {
	t.Helper()

	q, err := query.NewPersistedQuery(
		query.Type(queryType),
		[]byte(`{"filter":"active"}`),
	)
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	return q
}

// RunStoreSuite runs the full query store conformance suite. Each subtest
// gets a fresh store via the factory.
func RunStoreSuite(t *testing.T, factory StoreFactory) {
	t.Helper()

	t.Run("SaveAndLoadQueries", func(t *testing.T) {
		t.Parallel()
		testSaveAndLoadQueries(t, factory(t))
	})
	t.Run("DuplicateDetection", func(t *testing.T) {
		t.Parallel()
		testDuplicateDetection(t, factory(t))
	})
	t.Run("ReadAllQueries", func(t *testing.T) {
		t.Parallel()
		testReadAllQueries(t, factory(t))
	})
	t.Run("ReadQueriesFrom", func(t *testing.T) {
		t.Parallel()
		testReadQueriesFrom(t, factory(t))
	})
}

func testSaveAndLoadQueries(t *testing.T, store StoreSuite) {
	t.Helper()

	ctx := context.Background()

	before := time.Now()

	firstQuery := MustCreateQuery(t, "user.list")
	if err := store.SaveQuery(ctx, firstQuery); err != nil {
		t.Fatalf("SaveQuery firstQuery: %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	secondQuery := MustCreateQuery(t, "order.search")
	if err := store.SaveQuery(ctx, secondQuery); err != nil {
		t.Fatalf("SaveQuery secondQuery: %v", err)
	}

	// Load all after `before` → both
	all, err := store.LoadQueries(ctx, before)
	if err != nil {
		t.Fatalf("LoadQueries: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(all))
	}

	// Load after q1's timestamp → only q2
	mid := firstQuery.ReceivedAt()

	filtered, err := store.LoadQueries(ctx, mid)
	if err != nil {
		t.Fatalf("LoadQueries filtered: %v", err)
	}

	if len(filtered) != 1 {
		t.Fatalf("expected 1 query after mid, got %d", len(filtered))
	}

	if filtered[0].ID() != secondQuery.ID() {
		t.Errorf("expected q2, got %s", filtered[0].ID())
	}
}

func testDuplicateDetection(t *testing.T, store StoreSuite) {
	t.Helper()

	ctx := context.Background()

	q := MustCreateQuery(t, "user.get")

	if err := store.SaveQuery(ctx, q); err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}

	err := store.SaveQuery(ctx, q)
	if !errors.Is(err, query.ErrDuplicateQuery) {
		t.Fatalf("expected ErrDuplicateQuery, got %v", err)
	}
}

func testReadAllQueries(t *testing.T, store StoreSuite) {
	t.Helper()

	ctx := context.Background()

	for range 3 {
		q := MustCreateQuery(t, "user.list")
		if err := store.SaveQuery(ctx, q); err != nil {
			t.Fatalf("SaveQuery: %v", err)
		}

		time.Sleep(2 * time.Millisecond)
	}

	all, err := store.ReadAllQueries(ctx)
	if err != nil {
		t.Fatalf("ReadAllQueries: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 queries, got %d", len(all))
	}
}

func testReadQueriesFrom(t *testing.T, store StoreSuite) {
	t.Helper()

	ctx := context.Background()

	queryIDs := make([]id.RequestID, 0, 5)

	for range 5 {
		q := MustCreateQuery(t, "user.list")
		queryIDs = append(queryIDs, q.ID())

		if err := store.SaveQuery(ctx, q); err != nil {
			t.Fatalf("SaveQuery: %v", err)
		}

		time.Sleep(2 * time.Millisecond)
	}

	// Read from beginning
	all, err := store.ReadQueriesFrom(ctx, id.RequestID{}, 0)
	if err != nil {
		t.Fatalf("ReadQueriesFrom all: %v", err)
	}

	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}

	// Read after 2nd query with limit 2
	page, err := store.ReadQueriesFrom(ctx, queryIDs[1], 2)
	if err != nil {
		t.Fatalf("ReadQueriesFrom page: %v", err)
	}

	if len(page) != 2 {
		t.Fatalf("expected 2 in page, got %d", len(page))
	}

	if page[0].ID() != queryIDs[2] {
		t.Errorf("expected first in page to be query 3, got %s", page[0].ID())
	}
}
