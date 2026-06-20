package storage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/v2"
)

func newTestQueryStore(t *testing.T) *storage.SQLQueryStore {
	t.Helper()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	err = storage.SQLiteInitSchema(context.Background(), db)
	if err != nil {
		t.Fatalf("init schema: %v", err)
	}

	store, err := storage.NewSQLiteQueryStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteQueryStore: %v", err)
	}

	return store
}

func testQuery(t *testing.T, typ query.Type) *query.PersistedQuery {
	t.Helper()

	q, err := query.NewPersistedQuery(typ, []byte(`{"q":"alice"}`))
	if err != nil {
		t.Fatalf("create test query: %v", err)
	}

	return q
}

func TestSQLQueryStore_SaveAndLoadQueries(t *testing.T) {
	t.Parallel()

	store := newTestQueryStore(t)
	ctx := context.Background()

	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	old, err := query.NewPersistedQuery("user.search", nil, query.WithQueryReceivedAt(t1))
	if err != nil {
		t.Fatalf("create old: %v", err)
	}

	recent, err := query.NewPersistedQuery("user.count", nil, query.WithQueryReceivedAt(t2))
	if err != nil {
		t.Fatalf("create recent: %v", err)
	}

	if err := store.SaveQuery(ctx, old); err != nil {
		t.Fatalf("SaveQuery old: %v", err)
	}

	if err := store.SaveQuery(ctx, recent); err != nil {
		t.Fatalf("SaveQuery recent: %v", err)
	}

	result, err := store.LoadQueries(ctx, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("LoadQueries: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 query after cutoff, got %d", len(result))
	}

	if result[0].Type() != "user.count" {
		t.Errorf("expected user.count, got %s", result[0].Type())
	}
}

func TestSQLQueryStore_DuplicateQuery(t *testing.T) {
	t.Parallel()

	store := newTestQueryStore(t)
	ctx := context.Background()

	q := testQuery(t, "user.search")

	if err := store.SaveQuery(ctx, q); err != nil {
		t.Fatalf("SaveQuery first: %v", err)
	}

	err := store.SaveQuery(ctx, q)
	if err == nil {
		t.Fatal("expected error for duplicate query")
	}

	if !errors.Is(err, query.ErrDuplicateQuery) {
		t.Errorf("expected ErrDuplicateQuery, got: %v", err)
	}
}

func TestSQLQueryStore_ReadAllQueries(t *testing.T) {
	t.Parallel()

	store := newTestQueryStore(t)
	ctx := context.Background()

	q1 := testQuery(t, "user.search")
	q2 := testQuery(t, "user.count")

	if err := store.SaveQuery(ctx, q1); err != nil {
		t.Fatalf("SaveQuery q1: %v", err)
	}

	if err := store.SaveQuery(ctx, q2); err != nil {
		t.Fatalf("SaveQuery q2: %v", err)
	}

	all, err := store.ReadAllQueries(ctx)
	if err != nil {
		t.Fatalf("ReadAllQueries: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(all))
	}
}

func TestSQLQueryStore_ReadAllQueries_Empty(t *testing.T) {
	t.Parallel()

	store := newTestQueryStore(t)
	ctx := context.Background()

	all, err := store.ReadAllQueries(ctx)
	if err != nil {
		t.Fatalf("ReadAllQueries empty: %v", err)
	}

	if len(all) != 0 {
		t.Fatalf("expected 0 queries from empty store, got %d", len(all))
	}
}

func TestSQLQueryStore_ReadQueriesFrom_ZeroID(t *testing.T) {
	t.Parallel()

	store := newTestQueryStore(t)
	ctx := context.Background()

	for range 3 {
		q := testQuery(t, "user.search")
		if err := store.SaveQuery(ctx, q); err != nil {
			t.Fatalf("SaveQuery: %v", err)
		}
	}

	result, err := store.ReadQueriesFrom(ctx, id.RequestID{}, 2)
	if err != nil {
		t.Fatalf("ReadQueriesFrom zero ID: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 queries from start, got %d", len(result))
	}
}

func TestSQLQueryStore_ReadQueriesFrom_AfterID(t *testing.T) {
	t.Parallel()

	store := newTestQueryStore(t)
	ctx := context.Background()

	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	q1, _ := query.NewPersistedQuery("q1", nil, query.WithQueryReceivedAt(t1))
	q2, _ := query.NewPersistedQuery("q2", nil, query.WithQueryReceivedAt(t2))
	q3, _ := query.NewPersistedQuery("q3", nil, query.WithQueryReceivedAt(t3))

	for _, q := range []*query.PersistedQuery{q1, q2, q3} {
		if err := store.SaveQuery(ctx, q); err != nil {
			t.Fatalf("SaveQuery: %v", err)
		}
	}

	result, err := store.ReadQueriesFrom(ctx, q1.ID(), 10)
	if err != nil {
		t.Fatalf("ReadQueriesFrom q1: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 queries after q1, got %d", len(result))
	}
}

func TestSQLQueryStore_MetadataRoundtrip(t *testing.T) {
	t.Parallel()

	store := newTestQueryStore(t)
	ctx := context.Background()

	meta := query.NewMetadata()
	meta.CorrelationID = id.NewCorrelationID()
	meta.UserID = id.NewUserID()
	query.EnsureCustom(&meta)
	meta.Custom["source"] = "test"

	q, err := query.NewPersistedQuery("user.search", []byte(`{}`), query.WithQueryMetadata(meta))
	if err != nil {
		t.Fatalf("create query: %v", err)
	}

	if err := store.SaveQuery(ctx, q); err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}

	all, err := store.ReadAllQueries(ctx)
	if err != nil {
		t.Fatalf("ReadAllQueries: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("expected 1 query, got %d", len(all))
	}

	got := all[0].Metadata()
	if got.CorrelationID != meta.CorrelationID {
		t.Errorf("CorrelationID mismatch: got %v, want %v", got.CorrelationID, meta.CorrelationID)
	}

	if got.UserID != meta.UserID {
		t.Errorf("UserID mismatch: got %v, want %v", got.UserID, meta.UserID)
	}

	if got.Custom["source"] != "test" {
		t.Errorf("Custom[source] = %q, want %q", got.Custom["source"], "test")
	}
}

func TestSQLQueryStore_Close(t *testing.T) {
	t.Parallel()

	store := newTestQueryStore(t)
	ctx := context.Background()

	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	q := testQuery(t, "user.search")
	err := store.SaveQuery(ctx, q)
	if err == nil {
		t.Fatal("expected error on SaveQuery after close")
	}

	if !errors.Is(err, query.ErrStoreClosed) {
		t.Errorf("expected ErrStoreClosed, got: %v", err)
	}
}

func TestSQLBackend_QueryStoreFacade(t *testing.T) {
	t.Parallel()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	err = storage.SQLiteInitSchema(context.Background(), db)
	if err != nil {
		t.Fatalf("init schema: %v", err)
	}

	backend, err := storage.NewSQLiteBackend(db)
	if err != nil {
		t.Fatalf("NewSQLiteBackend: %v", err)
	}

	qs1, err := backend.QueryStore()
	if err != nil {
		t.Fatalf("QueryStore first: %v", err)
	}

	qs2, err := backend.QueryStore()
	if err != nil {
		t.Fatalf("QueryStore second: %v", err)
	}

	if qs1 != qs2 {
		t.Error("QueryStore() should return the same instance on repeated calls")
	}

	cs1, err := backend.CommandStore()
	if err != nil {
		t.Fatalf("CommandStore: %v", err)
	}

	if cs1 == nil {
		t.Error("CommandStore() should return non-nil")
	}
}
