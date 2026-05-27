package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

// testUpcaster increments a counter in the payload.
type testUpcaster struct {
	sourceType    event.Type
	sourceVersion event.SchemaVersion
}

func (u *testUpcaster) SourceType() event.Type             { return u.sourceType }
func (u *testUpcaster) SourceVersion() event.SchemaVersion { return u.sourceVersion }
func (u *testUpcaster) Upcast(evt event.Event) (*event.ImmutableEvent, error) {
	// In a real upcaster, transform payload here
	return evt.(*event.ImmutableEvent), nil
}

func TestVersionedStore_Load_NoUpcasters(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	vs := event.NewVersionedStore(store)

	ctx := context.Background()
	aggID := id.NewAggregateID()

	// Save an event
	evt, _ := event.New("test.event", aggID, "Test", event.Version(1), "payload")
	if err := store.Save(ctx, "Test", aggID, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Load through versioned store
	loaded, err := vs.Load(ctx, "Test", aggID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
}

func TestVersionedStore_NewVersionedStore_NilUpcasters(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	vs := event.NewVersionedStore(store)

	if vs == nil {
		t.Fatal("expected non-nil VersionedStore")
	}
}

// versionUpcaster transforms payload from "v1" to "v2" and bumps schema version.
type versionUpcaster struct{}

func (versionUpcaster) SourceType() event.Type             { return "test.upcast" }
func (versionUpcaster) SourceVersion() event.SchemaVersion { return 1 }
func (versionUpcaster) Upcast(evt event.Event) (*event.ImmutableEvent, error) {
	payload := string(evt.Payload())
	if payload == "v1" {
		payload = "v2"
	}
	return event.NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		[]byte(payload),
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithSchemaVersion(2),
	)
}

func TestVersionedStore_UpcastIntegration(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close() //nolint:errcheck // test helper

	// Save a v1 event
	ctx := context.Background()
	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent(
		"test.upcast", aggID, "Test", 1, []byte("v1"),
		event.WithSchemaVersion(1),
	)
	if err := store.Save(ctx, "Test", aggID, []event.Event{evt}, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Load through VersionedStore with upcaster
	vs := event.NewVersionedStore(store, versionUpcaster{})
	loaded, err := vs.Load(ctx, "Test", aggID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if string(loaded[0].Payload()) != "v2" {
		t.Errorf("payload = %q, want v2", loaded[0].Payload())
	}

	if loaded[0].SchemaVersion() != 2 {
		t.Errorf("schemaVersion = %d, want 2", loaded[0].SchemaVersion())
	}
}

func TestVersionedStore_LoadFromVersion_Upcast(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close() //nolint:errcheck // test helper

	ctx := context.Background()
	aggID := id.NewAggregateID()

	evt1, _ := event.NewEvent(
		"test.upcast",
		aggID,
		"Test",
		1,
		[]byte("v1"),
		event.WithSchemaVersion(1),
	)
	evt2, _ := event.NewEvent(
		"test.upcast",
		aggID,
		"Test",
		2,
		[]byte("skip"),
		event.WithSchemaVersion(2),
	)
	if err := store.Save(ctx, "Test", aggID, []event.Event{evt1, evt2}, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	vs := event.NewVersionedStore(store, versionUpcaster{})
	loaded, err := vs.LoadFromVersion(ctx, "Test", aggID, 1)
	if err != nil {
		t.Fatalf("load from version: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	// Only evt2 is loaded (from version 1, exclusive), and it has schema v2 so no upcast
	if string(loaded[0].Payload()) != "skip" {
		t.Errorf("payload = %q, want skip", loaded[0].Payload())
	}
}
