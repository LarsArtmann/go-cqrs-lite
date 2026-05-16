package storage

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func newPebbleTestStore(t *testing.T) *CQRSAdapter {
	t.Helper()

	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return NewCQRSAdapter(db, slog.Default())
}

func pebbleTestEvent(t *testing.T, aggID id.AggregateID, version event.Version) *event.Core {
	t.Helper()

	evt, err := event.NewEvent(
		"IssueCreated",
		aggID,
		"Issue",
		version,
		[]byte(fmt.Sprintf(`{"title":"test-%d"}`, version)),
	)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	return evt
}

func TestPebbleEventStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	evt := pebbleTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), "Issue", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Issue", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if loaded[0].Type() != "IssueCreated" {
		t.Errorf("Type = %q, want IssueCreated", loaded[0].Type())
	}

	if loaded[0].ID() != evt.ID() {
		t.Errorf("ID = %v, want %v", loaded[0].ID(), evt.ID())
	}
}

func TestPebbleEventStore_Save_ConcurrencyConflict(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	evt := pebbleTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), "Issue", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save first: %v", err)
	}

	evt2 := pebbleTestEvent(t, aggID, 2)

	err = store.Save(context.Background(), "Issue", aggID, []event.Event{evt2}, event.Version(0))
	if err == nil {
		t.Fatal("expected version mismatch error")
	}
}

func TestPebbleEventStore_AppendBatch(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	evt1 := pebbleTestEvent(t, aggID, 1)
	evt2 := pebbleTestEvent(t, aggID, 2)

	err := store.AppendBatch(context.Background(), "Issue", aggID, []event.Event{evt1, evt2})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Issue", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}

	if loaded[0].Version() != 1 {
		t.Errorf("events[0].Version = %d, want 1", loaded[0].Version())
	}

	if loaded[1].Version() != 2 {
		t.Errorf("events[1].Version = %d, want 2", loaded[1].Version())
	}
}

func TestPebbleEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	evt1 := pebbleTestEvent(t, aggID, 1)
	evt2 := pebbleTestEvent(t, aggID, 2)
	evt3 := pebbleTestEvent(t, aggID, 3)

	err := store.AppendBatch(context.Background(), "Issue", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.LoadFromVersion(context.Background(), "Issue", aggID, event.Version(1))
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events after version 1, got %d", len(loaded))
	}

	if loaded[0].Version() != 2 {
		t.Errorf("events[0].Version = %d, want 2", loaded[0].Version())
	}
}

func TestPebbleEventStore_Delete(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	evt := pebbleTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), "Issue", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	err = store.Delete(context.Background(), "Issue", aggID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Issue", aggID)
	if err != nil {
		t.Fatalf("Load after delete: %v", err)
	}

	if len(loaded) != 0 {
		t.Fatalf("expected 0 events after delete, got %d", len(loaded))
	}
}

func TestPebbleEventStore_MetadataRoundtrip(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()
	cid := id.NewCorrelationID()
	uid := id.NewUserID()

	evtWithMeta, err := event.NewEvent(
		"IssueCreated",
		aggID,
		"Issue",
		1,
		[]byte(`{"title":"test-1"}`),
		event.WithCorrelationID(cid),
		event.WithUserID(uid),
		event.WithCustom("env", "test"),
	)
	if err != nil {
		t.Fatalf("create event with metadata: %v", err)
	}

	err = store.Save(
		context.Background(),
		"Issue",
		aggID,
		[]event.Event{evtWithMeta},
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), "Issue", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	meta := loaded[0].Metadata()
	if meta == nil {
		t.Fatal("Metadata is nil")
	}

	if meta.CorrelationID != cid {
		t.Errorf("CorrelationID = %v, want %v", meta.CorrelationID, cid)
	}

	if meta.UserID != uid {
		t.Errorf("UserID = %v, want %v", meta.UserID, uid)
	}

	if meta.Custom["env"] != "test" {
		t.Errorf("Custom[env] = %q, want %q", meta.Custom["env"], "test")
	}
}

func TestPebbleEventStore_Close(t *testing.T) {
	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	store := NewCQRSAdapter(db, slog.Default())

	err = store.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestPebbleConfig_BackendPebble(t *testing.T) {
	cfg := NewPebbleConfig()
	if cfg.Backend != PebbleBackendPebble {
		t.Errorf("Backend = %q, want %q", cfg.Backend, PebbleBackendPebble)
	}
}

func TestPebbleConfig_WithProvider(t *testing.T) {
	provider := func(logger *slog.Logger) (event.Store, error) {
		return nil, nil
	}

	cfg := NewPebbleConfig(WithPebbleProvider(provider))
	if cfg.Provider == nil {
		t.Fatal("Provider should be set")
	}
}

func TestPebbleConfig_NewEventStore_MemoryBackend(t *testing.T) {
	cfg := NewPebbleConfig(WithPebbleBackend(PebbleBackendMemory))
	store, err := NewPebbleEventStore(cfg, slog.Default())
	if err != nil {
		t.Fatalf("NewEventStore: %v", err)
	}

	if store == nil {
		t.Fatal("store should not be nil")
	}
}

func TestPebbleConfig_NewEventStore_PebbleWithoutProvider(t *testing.T) {
	cfg := NewPebbleConfig()
	_, err := NewPebbleEventStore(cfg, slog.Default())
	if err == nil {
		t.Fatal("expected error for pebble backend without provider")
	}
}

func TestPebbleConfig_NewEventStore_UnknownBackend(t *testing.T) {
	cfg := NewPebbleConfig(WithPebbleBackend("unknown"))
	_, err := NewPebbleEventStore(cfg, slog.Default())
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

func TestPebbleEventStore_Persistence(t *testing.T) {
	dir := t.TempDir()

	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	store := NewCQRSAdapter(db, slog.Default())
	aggID := id.NewAggregateID()

	evt := pebbleTestEvent(t, aggID, 1)

	err = store.Save(context.Background(), "Issue", aggID, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	db2, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("reopen pebble: %v", err)
	}

	t.Cleanup(func() { _ = db2.Close() })

	store2 := NewCQRSAdapter(db2, slog.Default())
	loaded, err := store2.Load(context.Background(), "Issue", aggID)
	if err != nil {
		t.Fatalf("Load after reopen: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event after reopen, got %d", len(loaded))
	}

	if loaded[0].Type() != "IssueCreated" {
		t.Errorf("Type = %q, want IssueCreated", loaded[0].Type())
	}
}

func TestPebbleEventStore_Save_AggregateTypeMismatch(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	evt := pebbleTestEvent(t, aggID, 1)

	err := store.Save(context.Background(), "Project", aggID, []event.Event{evt}, event.Version(0))
	if err == nil {
		t.Fatal("expected error for aggregate type mismatch")
	}
}

func TestPebbleEventStore_Save_AggregateIDMismatch(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()
	otherID := id.NewAggregateID()

	evt := pebbleTestEvent(t, otherID, 1)

	err := store.Save(context.Background(), "Issue", aggID, []event.Event{evt}, event.Version(0))
	if err == nil {
		t.Fatal("expected error for aggregate ID mismatch")
	}
}

func TestPebbleEventStore_Save_VersionMismatch(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	evt := pebbleTestEvent(t, aggID, 5)

	err := store.Save(context.Background(), "Issue", aggID, []event.Event{evt}, event.Version(0))
	if err == nil {
		t.Fatal("expected error for version mismatch")
	}
}

func TestPebbleEventStore_Close_NilDB(t *testing.T) {
	store := NewCQRSAdapter(nil, slog.Default())

	err := store.Close()
	if err != nil {
		t.Fatalf("Close with nil db should return nil, got %v", err)
	}
}

func TestPebbleEventStore_Save_EmptyEvents(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	err := store.Save(context.Background(), "Issue", aggID, nil, event.Version(0))
	if err != nil {
		t.Fatalf("Save with empty events should return nil, got %v", err)
	}
}

func TestPebbleEventStore_AppendBatch_EmptyEvents(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	err := store.AppendBatch(context.Background(), "Issue", aggID, nil)
	if err != nil {
		t.Fatalf("AppendBatch with empty events should return nil, got %v", err)
	}
}

func TestPebbleEventStore_Load_Empty(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	loaded, err := store.Load(context.Background(), "Issue", aggID)
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}

	if len(loaded) != 0 {
		t.Fatalf("expected 0 events for empty aggregate, got %d", len(loaded))
	}
}

func TestPebbleEventStore_Delete_Empty(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	err := store.Delete(context.Background(), "Issue", aggID)
	if err != nil {
		t.Fatalf("Delete empty aggregate should succeed, got %v", err)
	}
}
