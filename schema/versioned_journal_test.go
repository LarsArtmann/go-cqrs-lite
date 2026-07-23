package schema_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestVersionedSeekableJournal_NilJournal(t *testing.T) {
	t.Parallel()

	j, err := schema.NewVersionedSeekableJournal(nil)
	if j != nil {
		t.Fatal("expected nil VersionedSeekableJournal")
	}

	if err == nil {
		t.Fatal("expected error for nil journal")
	}
}

func TestVersionedSeekableJournal_NoUpcasters(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewStreamID()

	evt, _ := event.NewEvent("test.event", aggID, "Test", event.Version(1), []byte("payload"))
	saveTestEvents(t, ctx, store, aggID, evt)

	journal, err := schema.NewVersionedSeekableJournal(store)
	if err != nil {
		t.Fatalf("new versioned journal: %v", err)
	}

	all, err := journal.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("expected 1 event, got %d", len(all))
	}
}

func TestVersionedSeekableJournal_ReadAll_Upcast(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewStreamID()

	evt, _ := event.NewEvent(
		"test.upcast", aggID, "Test", event.Version(1), []byte("v1"),
		event.WithSchemaVersion(1),
	)
	saveTestEvents(t, ctx, store, aggID, evt)

	journal, err := schema.NewVersionedSeekableJournal(store, versionUpcaster{})
	if err != nil {
		t.Fatal(err)
	}

	all, err := journal.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("expected 1 event, got %d", len(all))
	}

	if string(all[0].Payload()) != "v2" {
		t.Errorf("payload = %q, want v2", all[0].Payload())
	}

	if all[0].SchemaVersion() != 2 {
		t.Errorf("schemaVersion = %d, want 2", all[0].SchemaVersion())
	}
}

func TestVersionedSeekableJournal_ReadFrom_upcast(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewStreamID()

	evt1, _ := event.NewEvent(
		"test.upcast", aggID, "Test", event.Version(1), []byte("v1"),
		event.WithSchemaVersion(1),
	)
	evt2, _ := event.NewEvent(
		"test.upcast", aggID, "Test", event.Version(2), []byte("v1"),
		event.WithSchemaVersion(1),
	)
	saveTestEvents(t, ctx, store, aggID, evt1, evt2)

	journal, err := schema.NewVersionedSeekableJournal(store, versionUpcaster{})
	if err != nil {
		t.Fatal(err)
	}

	from, err := journal.ReadFrom(ctx, evt1.ID(), 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(from) != 1 {
		t.Fatalf("expected 1 event after cursor, got %d", len(from))
	}

	if string(from[0].Payload()) != "v2" {
		t.Errorf("payload = %q, want v2", from[0].Payload())
	}
}

func TestVersionedSeekableJournal_ReadAll_upcastError(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewStreamID()

	evt, _ := event.NewEvent(
		"test.upcast", aggID, "Test", event.Version(1), []byte("v1"),
		event.WithSchemaVersion(1),
	)
	saveTestEvents(t, ctx, store, aggID, evt)

	journal, err := schema.NewVersionedSeekableJournal(store, &failingUpcaster{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = journal.ReadAll(ctx)
	if err == nil {
		t.Fatal("expected error from failing upcaster")
	}
}

func TestVersionedSeekableJournal_ReadFrom_upcastError(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewStreamID()

	evt1, _ := event.NewEvent(
		"test.upcast", aggID, "Test", event.Version(1), []byte("v1"),
		event.WithSchemaVersion(1),
	)
	evt2, _ := event.NewEvent(
		"test.upcast", aggID, "Test", event.Version(2), []byte("v1"),
		event.WithSchemaVersion(1),
	)
	saveTestEvents(t, ctx, store, aggID, evt1, evt2)

	journal, err := schema.NewVersionedSeekableJournal(store, &failingUpcaster{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = journal.ReadFrom(ctx, evt1.ID(), 10)
	if err == nil {
		t.Fatal("expected error from failing upcaster")
	}
}

func TestVersionedSeekableJournal_NilUpcasters(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	journal, err := schema.NewVersionedSeekableJournal(store, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if journal == nil {
		t.Fatal("expected non-nil VersionedSeekableJournal")
	}
}

// Compile-time: VersionedSeekableJournal implements event.SeekableJournal.
var _ event.SeekableJournal = (*schema.VersionedSeekableJournal)(nil)
