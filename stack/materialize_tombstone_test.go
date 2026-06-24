package stack_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
)

// mockTombstoneStore implements kv.ViewStore AND kv.TombstoneQuerier.
// It tracks whether QueryByTombstone was called, proving List uses the fast path.
type mockTombstoneStore struct {
	records         []*userView
	queryCalled     bool
	excludeTombFlag bool
	onlyTombFlag    bool
}

func (m *mockTombstoneStore) Get(_ context.Context, _ stringKey) (*userView, error) {
	return nil, kv.ErrNotFound
}

func (m *mockTombstoneStore) Set(_ context.Context, _ stringKey, _ *userView) error {
	return nil
}

func (m *mockTombstoneStore) Delete(_ context.Context, _ stringKey) error { return nil }

func (m *mockTombstoneStore) Scan(_ context.Context, _ []byte) ([]*userView, error) {
	return m.records, nil
}

func (m *mockTombstoneStore) QueryByTombstone(
	_ context.Context,
	excludeTombstoned, onlyTombstoned bool,
) ([]*userView, error) {
	m.queryCalled = true
	m.excludeTombFlag = excludeTombstoned
	m.onlyTombFlag = onlyTombstoned

	return m.records, nil
}

func TestMaterialize_ListUsesTombstoneQuerier(t *testing.T) {
	t.Parallel()

	store := &mockTombstoneStore{
		records: []*userView{
			{Name: "Alice", Deleted: false},
			{Name: "Bob", Deleted: true},
		},
	}

	mat := stack.Materialize[userView, stringKey]{
		Store:        store,
		KeyFromEvent: func(evt event.Event) (stringKey, error) { return "", nil },
	}

	ctx := context.Background()

	// ExcludeTombstoned → should call QueryByTombstone(true, false).
	results, err := mat.List(ctx, stack.ExcludeTombstoned)
	if err != nil {
		t.Fatalf("List ExcludeTombstoned: %v", err)
	}

	if !store.queryCalled {
		t.Fatal("List should use TombstoneQuerier fast path")
	}

	if !store.excludeTombFlag || store.onlyTombFlag {
		t.Fatalf("flags: exclude=%v only=%v; want exclude=true only=false",
			store.excludeTombFlag, store.onlyTombFlag)
	}

	// The safety-net FilterTombstoned should have removed Bob.
	if len(results) != 1 || results[0].Name != "Alice" {
		t.Fatalf("results: got %d records, first=%s; want 1, Alice",
			len(results), safeUserName(results))
	}
}

func TestMaterialize_ListOnlyTombstoned(t *testing.T) {
	t.Parallel()

	store := &mockTombstoneStore{
		records: []*userView{
			{Name: "Alice", Deleted: false},
			{Name: "Bob", Deleted: true},
			{Name: "Charlie", Deleted: true},
		},
	}

	mat := stack.Materialize[userView, stringKey]{
		Store:        store,
		KeyFromEvent: func(evt event.Event) (stringKey, error) { return "", nil },
	}

	results, err := mat.List(context.Background(), stack.OnlyTombstoned)
	if err != nil {
		t.Fatalf("List OnlyTombstoned: %v", err)
	}

	if !store.queryCalled {
		t.Fatal("List should use TombstoneQuerier fast path")
	}

	if store.excludeTombFlag || !store.onlyTombFlag {
		t.Fatalf("flags: exclude=%v only=%v; want exclude=false only=true",
			store.excludeTombFlag, store.onlyTombFlag)
	}

	// Safety net: only tombstoned records survive.
	if len(results) != 2 {
		t.Fatalf("results: got %d, want 2", len(results))
	}
}

func TestMaterialize_ListFallsBackToScan(t *testing.T) {
	t.Parallel()

	// kvStore-backed store does NOT implement TombstoneQuerier.
	memStore := kv.NewMemStore()
	t.Cleanup(func() { _ = memStore.Close() })

	ts := kv.NewTypedStore[userView, stringKey](memStore)

	ctx := context.Background()
	_ = ts.Set(ctx, stringKey("active"), &userView{Name: "Alice", Deleted: false})
	_ = ts.Set(ctx, stringKey("deleted"), &userView{Name: "Bob", Deleted: true})

	mat := stack.Materialize[userView, stringKey]{
		Store:        ts,
		KeyFromEvent: func(evt event.Event) (stringKey, error) { return "", nil },
	}

	// List with ExcludeTombstoned should fall back to Scan + Go filter.
	results, err := mat.List(ctx, stack.ExcludeTombstoned)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(results) != 1 || results[0].Name != "Alice" {
		t.Fatalf("results: got %d, first=%s; want 1, Alice",
			len(results), safeUserName(results))
	}
}

func TestMaterialize_SQLViewStoreCompatibleWithHandler(t *testing.T) {
	t.Parallel()

	// This test proves that a store implementing kv.ViewStore + TombstoneQuerier
	// works correctly with the Materialize event handler (handleEvent path).
	// It simulates what storage.SQLViewStore does in production.
	store := &mockTombstoneStore{records: []*userView{}}

	mat := stack.Materialize[userView, stringKey]{
		Store:        store,
		KeyFromEvent: func(evt event.Event) (stringKey, error) { return stringKey(evt.AggregateID().String()), nil },
		OnCreate: func(_ context.Context, _ event.Event) (*userView, error) {
			return &userView{Name: "from-event", Email: "test@example.com"}, nil
		},
	}

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent(event.Type("user.created"), aggID, "User", event.Version(1), nil)

	msg := buildTestMessage(evt, "user.created")
	handler := mat.HandlerFunc()

	if err := handler(msg); err != nil {
		t.Fatalf("HandlerFunc: %v", err)
	}

	// View should return ErrNotFound (mock store Get always returns not found).
	_, err := mat.View(context.Background(), stringKey(aggID.String()))
	if err == nil || !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("View: err = %v, want ErrNotFound (mock)", err)
	}
}

func safeUserName(views []*userView) string {
	if len(views) == 0 {
		return "(empty)"
	}

	return views[0].Name
}
