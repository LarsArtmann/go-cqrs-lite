package storage_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/storage"
)

func TestSQLEventStore_LoadStream(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	func() { _ = storage.SQLiteInitSchema(ctx, db) }()

	store, err := storage.NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	aggID := id.NewAggregateID()

	wantEvents := []event.Event{
		mustEvent(t, "order.placed", aggID, 1),
		mustEvent(t, "order.paid", aggID, 2),
		mustEvent(t, "order.shipped", aggID, 3),
	}

	if err := store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("Order"), aggID),
		wantEvents,
		0,
	); err != nil {
		t.Fatalf("save: %v", err)
	}

	stream, err := store.LoadStream(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID))
	if err != nil {
		t.Fatalf("load stream: %v", err)
	}

	defer func() { _ = stream.Close() }()

	var got []string

	for {
		evt, ok := stream.Next()
		if !ok {
			break
		}

		got = append(got, string(evt.Type()))
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("got %d events, want 3", len(got))
	}

	want := []string{"order.placed", "order.paid", "order.shipped"}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("event[%d] type = %q, want %q", i, got[i], w)
		}
	}
}

func TestSQLEventStore_LoadStream_NotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	func() { _ = storage.SQLiteInitSchema(ctx, db) }()

	store, err := storage.NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	stream, err := store.LoadStream(ctx, event.NewAggregateRef("Order", id.NewAggregateID()))
	if err != nil {
		t.Fatalf("load stream: %v", err)
	}

	defer func() { _ = stream.Close() }()

	_, ok := stream.Next()
	if ok {
		t.Error("expected no events for non-existent aggregate")
	}
}

func mustEvent(tb testing.TB, typ string, aggID id.AggregateID, ver int) event.Event {
	tb.Helper()

	evt, err := event.NewEvent(
		event.Type(typ),
		aggID,
		"Test",
		event.Version(ver),
		[]byte(`{}`),
	)
	if err != nil {
		tb.Fatalf("new event: %v", err)
	}

	return evt
}

func TestSQLEventStore_LoadStream_ScanError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	func() { _ = storage.SQLiteInitSchema(ctx, db) }()

	store, err := storage.NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	aggID := id.NewAggregateID()

	_, err = db.ExecContext(
		ctx,
		`INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, payload_encoding, metadata, occurred_at)
		 VALUES ('invalid-id', 'test', 'Order', ?, 1, 1, '{}', 'json', '{}', '2024-01-01T00:00:00Z')`,
		aggID.String(),
	)
	if err != nil {
		t.Fatalf("insert bad row: %v", err)
	}

	stream, err := store.LoadStream(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID))
	if err != nil {
		t.Fatalf("load stream: %v", err)
	}
	defer func() { _ = stream.Close() }()

	_, ok := stream.Next()
	if ok {
		t.Fatal("expected Next to fail due to scan error")
	}

	if err := stream.Err(); err == nil {
		t.Fatal("expected stream.Err to return scan error")
	}
}
