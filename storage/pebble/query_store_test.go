package pebble_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
)

func newQueryStore(t *testing.T) *cqrspebble.QueryStore {
	t.Helper()

	dir := t.TempDir()

	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	store, err := cqrspebble.NewQueryStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func mustCreateQuery(t *testing.T, queryType string) *query.PersistedQuery {
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

func TestQueryStore_SaveAndLoadQueries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newQueryStore(t)

	before := time.Now()

	firstQuery := mustCreateQuery(t, "user.list")
	if err := store.SaveQuery(ctx, firstQuery); err != nil {
		t.Fatalf("SaveQuery firstQuery: %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	secondQuery := mustCreateQuery(t, "order.search")
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

func TestQueryStore_DuplicateDetection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newQueryStore(t)

	q := mustCreateQuery(t, "user.get")

	if err := store.SaveQuery(ctx, q); err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}

	err := store.SaveQuery(ctx, q)
	if !errors.Is(err, query.ErrDuplicateQuery) {
		t.Fatalf("expected ErrDuplicateQuery, got %v", err)
	}
}

func TestQueryStore_ReadAllQueries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newQueryStore(t)

	for range 3 {
		q := mustCreateQuery(t, "user.list")
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

func TestQueryStore_ReadQueriesFrom(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newQueryStore(t)

	queryIDs := make([]id.RequestID, 0, 5)

	for range 5 {
		q := mustCreateQuery(t, "user.list")
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
