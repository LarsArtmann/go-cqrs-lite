package schema_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/schema"
)

type testUpcaster struct {
	sourceType    event.Type
	sourceVersion event.SchemaVersion
}

func (u *testUpcaster) SourceType() event.Type             { return u.sourceType }
func (u *testUpcaster) SourceVersion() event.SchemaVersion { return u.sourceVersion }
func (u *testUpcaster) Upcast(evt event.Event) (*event.ImmutableEvent, error) {
	immutable, ok := evt.(*event.ImmutableEvent)
	if !ok {
		return nil, fmt.Errorf("unexpected event type %T", evt)
	}

	return immutable, nil
}

func TestVersionedStore_Load_NoUpcasters(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	versioned := schema.NewVersionedStore(store)

	ctx := context.Background()
	aggID := id.NewAggregateID()

	evt, _ := event.New("test.event", aggID, "Test", event.Version(1), "payload")
	if err := store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("Test"), aggID),
		[]event.Event{evt},
		event.Version(0),
	); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := versioned.Load(ctx, event.NewAggregateRef(event.AggregateType("Test"), aggID))
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
	versioned := schema.NewVersionedStore(store)
	if versioned == nil {
		t.Fatal("expected non-nil VersionedStore")
	}
}

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

	ctx := context.Background()
	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent(
		"test.upcast", aggID, "Test", 1, []byte("v1"),
		event.WithSchemaVersion(1),
	)
	if err := store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("Test"), aggID),
		[]event.Event{evt},
		0,
	); err != nil {
		t.Fatalf("save: %v", err)
	}

	versioned := schema.NewVersionedStore(store, versionUpcaster{})
	loaded, err := versioned.Load(ctx, event.NewAggregateRef(event.AggregateType("Test"), aggID))
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
	if err := store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("Test"), aggID),
		[]event.Event{evt1, evt2},
		0,
	); err != nil {
		t.Fatalf("save: %v", err)
	}

	versioned := schema.NewVersionedStore(store, versionUpcaster{})
	loaded, err := versioned.LoadFromVersion(
		ctx,
		event.NewAggregateRef(event.AggregateType("Test"), aggID),
		1,
	)
	if err != nil {
		t.Fatalf("load from version: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if string(loaded[0].Payload()) != "skip" {
		t.Errorf("payload = %q, want skip", loaded[0].Payload())
	}
}
