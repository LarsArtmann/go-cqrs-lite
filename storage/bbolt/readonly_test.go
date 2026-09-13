package bbolt

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Regression for issue #22: OpenWith with ReadOnly used to fail during
// construction because newBackend unconditionally created buckets via a
// write transaction, which bbolt rejects on a read-only-opened database.
// Read-only opens must succeed, serve reads, and reject writes at call time.
func TestOpenWithReadOnlyServesReadsAndRejectsWrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "readonly.db")

	writeBackend, err := Open(path, nil)
	if err != nil {
		t.Fatalf("open writable backend: %v", err)
	}

	ctx := context.Background()
	ref := id.NewStreamRef("User", id.NewStreamID())
	evt, err := event.NewEvent("user.created", ref.ID, "User", 1, []byte(`{"name":"alice"}`))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := writeBackend.EventStore().Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := writeBackend.Close(); err != nil {
		t.Fatalf("close writable backend: %v", err)
	}

	readBackend, err := OpenWith(path, &bolt.Options{
		ReadOnly: true,
		Timeout:  5 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("open read-only backend (issue #22): %v", err)
	}
	t.Cleanup(func() { _ = readBackend.Close() })

	loaded, err := readBackend.EventStore().Load(ctx, ref)
	if err != nil {
		t.Fatalf("load from read-only backend: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if loaded[0].Type() != "user.created" {
		t.Fatalf("expected type user.created, got %s", loaded[0].Type())
	}

	newEvt, err := event.NewEvent("user.updated", ref.ID, "User", 2, []byte(`{"name":"bob"}`))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := readBackend.EventStore().Save(ctx, ref, []event.Event{newEvt}, 1); err == nil {
		t.Fatal("expected write on read-only backend to fail, got nil error")
	}
}
