package bbolt

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestBackupRestore_FullLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Phase 1: Open source, write data.
	srcDir := t.TempDir()
	source, err := Open(filepath.Join(srcDir, "source.db"), nil)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	defer deferClose(source)

	store := source.EventStore()
	ref := id.NewStreamRef("User", id.NewStreamID())

	evt1, _ := event.NewEvent("user.created", ref.ID, "User", 1, []byte(`{"name":"alice"}`))
	evt2, _ := event.NewEvent("user.updated", ref.ID, "User", 2, []byte(`{"name":"alice2"}`))
	if err := store.Save(ctx, ref, []event.Event{evt1, evt2}, 0); err != nil {
		t.Fatalf("save events: %v", err)
	}

	// Phase 2: Backup via bbolt tx.WriteTo.
	backupPath := filepath.Join(t.TempDir(), "backup.db")
	backupFile(t, source.DB(), backupPath)

	// Phase 3: Write more data after backup (must NOT appear in restore).
	evt3, _ := event.NewEvent("user.deleted", ref.ID, "User", 3, []byte(`{}`))
	if err := store.Save(ctx, ref, []event.Event{evt3}, 2); err != nil {
		t.Fatalf("save post-backup event: %v", err)
	}

	// Phase 4: Open restored backend, verify only pre-backup data.
	restored, err := Open(backupPath, nil)
	if err != nil {
		t.Fatalf("open restored: %v", err)
	}
	defer deferClose(restored)

	rStore := restored.EventStore()
	events, err := rStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("load from restored: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events in backup, got %d", len(events))
	}
	for i, e := range events {
		if e.Version() != event.Version(i+1) {
			t.Errorf("event %d version = %d, want %d", i, e.Version(), i+1)
		}
	}

	// Phase 5: Verify restored backend accepts new writes.
	newRef := id.NewStreamRef("Order", id.NewStreamID())
	evtNew, _ := event.NewEvent("order.created", newRef.ID, "Order", 1, []byte(`{}`))
	if err := rStore.Save(ctx, newRef, []event.Event{evtNew}, 0); err != nil {
		t.Fatalf("save to restored: %v", err)
	}
	loaded, err := rStore.Load(ctx, newRef)
	if err != nil || len(loaded) != 1 {
		t.Fatalf("restored backend should accept new writes: len=%d err=%v", len(loaded), err)
	}
}

func backupFile(t *testing.T, db *bolt.DB, destPath string) {
	t.Helper()

	f, err := os.Create(destPath)
	if err != nil {
		t.Fatalf("create backup file: %v", err)
	}
	defer f.Close()

	if err := db.View(func(tx *bolt.Tx) error {
		_, err := tx.WriteTo(f)
		return err
	}); err != nil {
		t.Fatalf("backup: %v", err)
	}
}
