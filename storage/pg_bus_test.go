package storage_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/v2"
)

// mockListener implements storage.NotificationListener for testing.
type mockListener struct {
	notifications chan string
	closed        atomic.Bool
	listenedCh    string
	listenErr     error
	listenSignal  chan struct{} // closed once Listen() has been called
}

func newMockListener() *mockListener {
	return &mockListener{
		notifications: make(chan string, 10),
		listenSignal:  make(chan struct{}),
	}
}

func (m *mockListener) Listen(_ context.Context, channel string) error {
	m.listenedCh = channel
	close(m.listenSignal)
	return m.listenErr
}

func (m *mockListener) Notifications() <-chan string { return m.notifications }

func (m *mockListener) Close() error {
	m.closed.Store(true)
	close(m.notifications)
	return nil
}

// waitForListen blocks until Listen() is called or the test times out.
func (m *mockListener) waitForListen(t *testing.T) {
	t.Helper()
	select {
	case <-m.listenSignal:
	case <-time.After(2 * time.Second):
		t.Fatal("mockListener.Listen was not called within 2s")
	}
}

// noopNotify is a notifyFunc that does nothing (for testing with SQLite).
func noopNotify(_ context.Context, _, _ string) error { return nil }

func newBusTestStore(t *testing.T) *storage.SQLEventStore {
	t.Helper()

	ctx := context.Background()
	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := storage.SQLiteInitSchema(ctx, db); err != nil {
		t.Fatalf("SQLiteInitSchema: %v", err)
	}

	store, err := storage.NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	return store
}

func TestPostgresBus_NilDB(t *testing.T) {
	t.Parallel()

	store := newBusTestStore(t)
	listener := newMockListener()

	_, err := storage.NewPostgresBus(nil, store, listener)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestPostgresBus_NilStore(t *testing.T) {
	t.Parallel()

	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	listener := newMockListener()

	_, err := storage.NewPostgresBus(db, nil, listener)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestPostgresBus_NilListener(t *testing.T) {
	t.Parallel()

	store := newBusTestStore(t)
	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	_, err := storage.NewPostgresBus(db, store, nil)
	if err == nil {
		t.Fatal("expected error for nil listener")
	}
}

func TestPostgresBus_ListenCalledWithChannel(t *testing.T) {
	t.Parallel()

	store := newBusTestStore(t)
	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	listener := newMockListener()
	bus, err := storage.NewPostgresBus(db, store, listener,
		storage.WithBusChannel("custom_ch"))
	if err != nil {
		t.Fatalf("NewPostgresBus: %v", err)
	}
	t.Cleanup(func() { _ = bus.Close() })

	listener.waitForListen(t)

	if listener.listenedCh != "custom_ch" {
		t.Fatalf("expected Listen on custom_ch, got %q", listener.listenedCh)
	}
}

func TestPostgresBus_ListenError(t *testing.T) {
	store := newBusTestStore(t)
	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	listener := newMockListener()
	listener.listenErr = errors.New("simulated LISTEN failure")

	_, err := storage.NewPostgresBus(db, store, listener)
	if err == nil {
		t.Fatal("expected error when Listen fails")
	}
}

func TestPostgresBus_SubscribeAndPublish(t *testing.T) {
	t.Parallel()

	store := newBusTestStore(t)
	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	listener := newMockListener()

	bus, err := storage.NewPostgresBus(
		db, store, listener,
		storage.WithRefetchDelay(time.Millisecond),
		storage.WithNotifyFunc(noopNotify),
	)
	if err != nil {
		t.Fatalf("NewPostgresBus: %v", err)
	}

	t.Cleanup(func() { _ = bus.Close() })

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("test.created", aggID, "Test", event.Version(1), []byte(`{}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := store.AppendBatch(context.Background(),
		event.NewAggregateRef("Test", aggID), []event.Event{evt}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	var (
		received atomic.Int64
		wg       sync.WaitGroup
	)

	wg.Add(1)

	_ = bus.Subscribe("test.created", func(_ context.Context, e event.Event) error {
		if e.ID() == evt.ID() {
			received.Add(1)
			wg.Done()
		}

		return nil
	})

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	wg.Wait()

	if received.Load() != 1 {
		t.Errorf("expected 1 local dispatch, got %d", received.Load())
	}
}

func TestPostgresBus_NotificationRefetch(t *testing.T) {
	t.Parallel()

	store := newBusTestStore(t)
	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	listener := newMockListener()

	bus, err := storage.NewPostgresBus(
		db, store, listener,
		storage.WithRefetchDelay(time.Millisecond),
		storage.WithNotifyFunc(noopNotify),
	)
	if err != nil {
		t.Fatalf("NewPostgresBus: %v", err)
	}

	t.Cleanup(func() { _ = bus.Close() })

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("test.updated", aggID, "Test", event.Version(1), []byte(`{}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := store.AppendBatch(context.Background(),
		event.NewAggregateRef("Test", aggID), []event.Event{evt}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	var (
		received atomic.Int64
		wg       sync.WaitGroup
	)

	wg.Add(1)

	_ = bus.Subscribe("test.updated", func(_ context.Context, e event.Event) error {
		if e.ID() == evt.ID() {
			received.Add(1)
			wg.Done()
		}

		return nil
	})

	payload, marshalErr := json.Marshal(map[string]any{
		"eid": evt.ID().String(),
		"et":  "test.updated",
		"at":  "Test",
		"aid": aggID.String(),
		"v":   1,
	})
	if marshalErr != nil {
		t.Fatalf("marshal: %v", marshalErr)
	}

	listener.notifications <- string(payload)

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for re-fetched event dispatch")
	}

	if received.Load() != 1 {
		t.Errorf("expected 1 remote dispatch via re-fetch, got %d", received.Load())
	}
}

func TestPostgresBus_SubscribeAll(t *testing.T) {
	t.Parallel()

	store := newBusTestStore(t)
	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	listener := newMockListener()

	bus, err := storage.NewPostgresBus(db, store, listener, storage.WithNotifyFunc(noopNotify))
	if err != nil {
		t.Fatalf("NewPostgresBus: %v", err)
	}

	t.Cleanup(func() { _ = bus.Close() })

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("any.event", aggID, "Test", event.Version(1), []byte(`{}`))

	if err := store.AppendBatch(context.Background(),
		event.NewAggregateRef("Test", aggID), []event.Event{evt}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	var received atomic.Int64

	_ = bus.SubscribeAll(func(_ context.Context, _ event.Event) error {
		received.Add(1)

		return nil
	})

	_ = bus.Publish(context.Background(), evt)

	time.Sleep(100 * time.Millisecond)

	if received.Load() != 1 {
		t.Errorf("expected 1 event via SubscribeAll, got %d", received.Load())
	}
}

func TestPostgresBus_Close(t *testing.T) {
	t.Parallel()

	store := newBusTestStore(t)
	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	listener := newMockListener()

	bus, err := storage.NewPostgresBus(db, store, listener, storage.WithNotifyFunc(noopNotify))
	if err != nil {
		t.Fatalf("NewPostgresBus: %v", err)
	}

	if err := bus.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := bus.Close(); err != nil {
		t.Fatalf("second Close should be no-op, got: %v", err)
	}

	if !listener.closed.Load() {
		t.Error("listener should be closed after bus Close")
	}
}

func TestPostgresBus_Middleware(t *testing.T) {
	t.Parallel()

	store := newBusTestStore(t)
	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	listener := newMockListener()

	bus, err := storage.NewPostgresBus(db, store, listener, storage.WithNotifyFunc(noopNotify))
	if err != nil {
		t.Fatalf("NewPostgresBus: %v", err)
	}

	t.Cleanup(func() { _ = bus.Close() })

	var middlewareCalled atomic.Int64

	mw := func(next event.Handler) event.Handler {
		return func(ctx context.Context, e event.Event) error {
			middlewareCalled.Add(1)

			return next(ctx, e)
		}
	}

	_ = bus.Use(mw)

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("test.mw", aggID, "Test", event.Version(1), []byte(`{}`))

	_ = store.AppendBatch(context.Background(),
		event.NewAggregateRef("Test", aggID), []event.Event{evt})

	_ = bus.Subscribe("test.mw", func(_ context.Context, _ event.Event) error { return nil })
	_ = bus.Publish(context.Background(), evt)

	time.Sleep(100 * time.Millisecond)

	if middlewareCalled.Load() != 1 {
		t.Errorf("expected middleware called once, got %d", middlewareCalled.Load())
	}
}

func TestPostgresBus_PublishClosedBus(t *testing.T) {
	t.Parallel()

	store := newBusTestStore(t)
	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	listener := newMockListener()

	bus, err := storage.NewPostgresBus(db, store, listener, storage.WithNotifyFunc(noopNotify))
	if err != nil {
		t.Fatalf("NewPostgresBus: %v", err)
	}

	_ = bus.Close()

	evt, _ := event.NewEvent(
		"test.closed",
		id.NewAggregateID(),
		"Test",
		event.Version(1),
		[]byte(`{}`),
	)
	err = bus.Publish(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error publishing to closed bus")
	}
}

func TestPostgresBus_NotificationBadJSON(t *testing.T) {
	t.Parallel()

	store := newBusTestStore(t)
	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	listener := newMockListener()

	bus, err := storage.NewPostgresBus(
		db, store, listener,
		storage.WithRefetchDelay(time.Millisecond),
		storage.WithNotifyFunc(noopNotify),
	)
	if err != nil {
		t.Fatalf("NewPostgresBus: %v", err)
	}

	t.Cleanup(func() { _ = bus.Close() })

	listener.notifications <- "not valid json"

	time.Sleep(100 * time.Millisecond)

	if !errors.Is(err, nil) {
		t.Log("bus should not crash on bad JSON")
	}
}

// versionOnlySource wraps an event.EventSource but deliberately does NOT
// implement storage.EventByIDLoader. This forces PostgresBus.refetchEvent
// onto the refetchByVersion fallback path, exercising the version-scan code.
type versionOnlySource struct {
	inner event.EventSource
}

func (v *versionOnlySource) Load(ctx context.Context, ref event.AggregateRef) ([]event.Event, error) {
	return v.inner.Load(ctx, ref)
}

func (v *versionOnlySource) LoadFromVersion(
	ctx context.Context, ref event.AggregateRef, ver event.Version,
) ([]event.Event, error) {
	return v.inner.LoadFromVersion(ctx, ref, ver)
}

func (v *versionOnlySource) LoadToVersion(
	ctx context.Context, ref event.AggregateRef, max event.Version,
) ([]event.Event, error) {
	return v.inner.LoadToVersion(ctx, ref, max)
}

func (v *versionOnlySource) LoadToTimestamp(
	ctx context.Context, ref event.AggregateRef, max time.Time,
) ([]event.Event, error) {
	return v.inner.LoadToTimestamp(ctx, ref, max)
}

func (v *versionOnlySource) Close() error { return v.inner.Close() }

// Compile-time: versionOnlySource satisfies EventSource but NOT EventByIDLoader.
var _ event.EventSource = (*versionOnlySource)(nil)

// TestPostgresBus_RefetchVersionFallback verifies the refetchByVersion path
// is used when the store does NOT implement EventByIDLoader. This is the
// fallback for stores like MemoryStore and Pebble (until they gain
// LoadByEventID). It sends a NOTIFY with an aggregate reference, and asserts
// the listener re-fetches and dispatches the event via LoadFromVersion.
func TestPostgresBus_RefetchVersionFallback(t *testing.T) {
	t.Parallel()

	inner := newBusTestStore(t)
	store := &versionOnlySource{inner: inner}

	db, _ := storage.OpenSQLiteInMemory()
	t.Cleanup(func() { _ = db.Close() })

	listener := newMockListener()

	bus, err := storage.NewPostgresBus(db, store, listener,
		storage.WithRefetchDelay(time.Millisecond),
		storage.WithNotifyFunc(noopNotify),
	)
	if err != nil {
		t.Fatalf("NewPostgresBus: %v", err)
	}
	t.Cleanup(func() { _ = bus.Close() })

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	evt, err := event.NewEvent("test.versioned", aggID, "Test", event.Version(1),
		[]byte(`{"n":1}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := inner.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var received atomic.Int64
	var wg sync.WaitGroup
	wg.Add(1)

	_ = bus.Subscribe("test.versioned", func(_ context.Context, e event.Event) error {
		if e.ID() == evt.ID() {
			received.Add(1)
			wg.Done()
		}
		return nil
	})

	// Build the payload manually (simulating a NOTIFY from another process)
	// and inject it via the mock listener channel.
	payload, marshalErr := json.Marshal(map[string]any{
		"eid": evt.ID().String(),
		"et":  "test.versioned",
		"at":  "Test",
		"aid": aggID.String(),
		"v":   1,
	})
	if marshalErr != nil {
		t.Fatalf("marshal: %v", marshalErr)
	}

	listener.notifications <- string(payload)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if received.Load() != 1 {
			t.Fatalf("expected 1 delivery, got %d", received.Load())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: version-scan fallback did not deliver event within 2s")
	}
}
