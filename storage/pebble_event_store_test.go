package storage

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func newPebbleTestStore(t *testing.T) *PebbleEventStore {
	t.Helper()

	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return NewPebbleStore(db, slog.Default())
}

func TestPebbleEventStore_SaveAndLoad(t *testing.T) {
	t.Parallel()
	testEventStore_SaveAndLoad(t, newPebbleTestStore(t), issueStoreConfig())
}

func TestPebbleEventStore_Save_ConcurrencyConflict(t *testing.T) {
	t.Parallel()
	testEventStore_ConcurrencyConflict(t, newPebbleTestStore(t), issueStoreConfig())
}

func TestPebbleEventStore_AppendBatch(t *testing.T) {
	t.Parallel()
	testEventStore_AppendBatch(t, newPebbleTestStore(t), issueStoreConfig())
}

func TestPebbleEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()
	testEventStore_LoadFromVersion(t, newPebbleTestStore(t), issueStoreConfig())
}

func TestPebbleEventStore_Delete(t *testing.T) {
	t.Parallel()
	testEventStore_Delete(t, newPebbleTestStore(t), issueStoreConfig())
}

func TestPebbleEventStore_MetadataRoundtrip(t *testing.T) {
	t.Parallel()
	testEventStore_MetadataRoundtrip(t, newPebbleTestStore(t), issueStoreConfig(), "test")
}

func TestPebbleEventStore_Close(t *testing.T) {
	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	store := NewPebbleStore(db, slog.Default())

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

func TestPebbleConfig_NewEventStore_MemoryBackend_RequiresProvider(t *testing.T) {
	cfg := NewPebbleConfig(WithPebbleBackend(PebbleBackendMemory))
	_, err := NewPebbleEventStore(cfg, slog.Default())
	if err == nil {
		t.Fatal("expected error for memory backend without provider")
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

	store := NewPebbleStore(db, slog.Default())
	aggID := id.NewAggregateID()

	evt := issueStoreConfig().newTestEvent(t, aggID, 1)

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

	store2 := NewPebbleStore(db2, slog.Default())
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

func TestPebbleEventStore_Save_Mismatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		saveAggType  event.AggregateType
		saveAggID    id.AggregateID
		eventAggID   id.AggregateID
		eventVersion event.Version
	}{
		{
			name:         "aggregate_type",
			saveAggType:  "Project",
			saveAggID:    id.NewAggregateID(),
			eventAggID:   id.NewAggregateID(),
			eventVersion: 1,
		},
		{
			name:         "aggregate_id",
			saveAggType:  "Issue",
			saveAggID:    id.NewAggregateID(),
			eventAggID:   id.NewAggregateID(),
			eventVersion: 1,
		},
		{
			name:         "version",
			saveAggType:  "Issue",
			saveAggID:    id.NewAggregateID(),
			eventAggID:   id.NewAggregateID(),
			eventVersion: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := newPebbleTestStore(t)
			evt := issueStoreConfig().newTestEvent(t, tt.eventAggID, tt.eventVersion)

			err := store.Save(
				context.Background(),
				tt.saveAggType,
				tt.saveAggID,
				[]event.Event{evt},
				event.Version(0),
			)
			if err == nil {
				t.Fatalf("expected error for %s mismatch", tt.name)
			}
		})
	}
}

func TestPebbleEventStore_Close_NilDB(t *testing.T) {
	store := NewPebbleStore(nil, slog.Default())

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

func TestPebbleEventStore_LoadToVersion(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt1 := issueStoreConfig().newTestEvent(t, aggID, 1)
	evt2 := issueStoreConfig().newTestEvent(t, aggID, 2)
	evt3 := issueStoreConfig().newTestEvent(t, aggID, 3)

	err := store.AppendBatch(ctx, "Issue", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToVersion(ctx, "Issue", aggID, 2)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestPebbleEventStore_LoadToVersion_NotFound(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)

	_, err := store.LoadToVersion(context.Background(), "Issue", id.NewAggregateID(), 5)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestPebbleEventStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	ctx := context.Background()

	aggID := id.NewAggregateID()
	now := time.Now()

	evt1, _ := event.NewEvent(
		"Created",
		aggID,
		"Issue",
		1,
		nil,
		event.WithOccurredAt(now.Add(-2*time.Hour)),
	)
	evt2, _ := event.NewEvent(
		"Updated",
		aggID,
		"Issue",
		2,
		nil,
		event.WithOccurredAt(now.Add(-1*time.Hour)),
	)
	evt3, _ := event.NewEvent("Deleted", aggID, "Issue", 3, nil, event.WithOccurredAt(now))

	err := store.AppendBatch(ctx, "Issue", aggID, []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	events, err := store.LoadToTimestamp(ctx, "Issue", aggID, now.Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestPebbleEventStore_LoadToTimestamp_NotFound(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)

	_, err := store.LoadToTimestamp(context.Background(), "Issue", id.NewAggregateID(), time.Now())
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestPebbleEventStore_ConcurrentSave_VersionConflict(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	cfg := issueStoreConfig()
	aggID := id.NewAggregateID()

	evt1 := cfg.newTestEvent(t, aggID, 1)
	saveCfgEvent(t, store, cfg, aggID, evt1)

	const goroutines = 10
	errCh := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			evt := cfg.newTestEvent(t, aggID, 2)
			errCh <- store.Save(
				context.Background(),
				cfg.aggType,
				aggID,
				[]event.Event{evt},
				event.Version(1),
			)
		}()
	}

	var successes, conflicts int

	for i := 0; i < goroutines; i++ {
		err := <-errCh
		if err == nil {
			successes++
		} else if errors.Is(err, event.ErrVersionConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if successes != 1 {
		t.Fatalf("expected exactly 1 successful save, got %d", successes)
	}

	if conflicts != goroutines-1 {
		t.Fatalf("expected %d conflicts, got %d", goroutines-1, conflicts)
	}

	loaded, err := store.Load(context.Background(), cfg.aggType, aggID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events after concurrent save, got %d", len(loaded))
	}
}
