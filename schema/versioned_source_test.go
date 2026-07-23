package schema_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

type testUpcaster struct {
	sourceType    event.Type
	sourceVersion event.SchemaVersion
}

func (u *testUpcaster) SourceType() event.Type             { return u.sourceType }
func (u *testUpcaster) SourceVersion() event.SchemaVersion { return u.sourceVersion }
func (u *testUpcaster) Upcast(evt event.Event) (event.Event, error) {
	return evt, nil
}

func TestVersionedStore_Load_NoUpcasters(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	versioned, err := schema.NewVersionedStore(store)
	if err != nil {
		t.Fatalf("new versioned store: %v", err)
	}

	ctx := context.Background()
	aggID := id.NewStreamID()

	evt, _ := event.New("test.event", aggID, "Test", event.Version(1), "payload")
	if err := store.Save(
		ctx,
		id.NewStreamRef(id.StreamType("Test"), aggID),
		[]event.Event{evt},
		event.Version(0),
	); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := versioned.Load(ctx, id.NewStreamRef(id.StreamType("Test"), aggID))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
}

func TestVersionedStore_NewVersionedStore_NilStore(t *testing.T) {
	t.Parallel()

	versioned, err := schema.NewVersionedStore(nil)
	if versioned != nil {
		t.Fatal("expected nil VersionedStore")
	}

	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestVersionedStore_NewVersionedStore_NilUpcasters(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	versioned, err := schema.NewVersionedStore(store)
	if err != nil {
		t.Fatal(err)
	}
	if versioned == nil {
		t.Fatal("expected non-nil VersionedStore")
	}
}

type versionUpcaster struct{}

func (versionUpcaster) SourceType() event.Type             { return "test.upcast" }
func (versionUpcaster) SourceVersion() event.SchemaVersion { return 1 }
func (versionUpcaster) Upcast(evt event.Event) (event.Event, error) {
	payload := string(evt.Payload())
	if payload == "v1" {
		payload = "v2"
	}

	return event.NewEvent(
		evt.Type(),
		evt.StreamID(),
		evt.StreamType(),
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
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewStreamID()
	evt, _ := event.NewEvent(
		"test.upcast", aggID, "Test", 1, []byte("v1"),
		event.WithSchemaVersion(1),
	)
	saveTestEvents(t, ctx, store, aggID, evt)

	versioned, err := schema.NewVersionedStore(store, versionUpcaster{})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := versioned.Load(ctx, id.NewStreamRef(id.StreamType("Test"), aggID))
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
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewStreamID()

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
	saveTestEvents(t, ctx, store, aggID, evt1, evt2)

	versioned, err := schema.NewVersionedStore(store, versionUpcaster{})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := versioned.LoadFromVersion(
		ctx,
		id.NewStreamRef(id.StreamType("Test"), aggID),
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

func TestVersionedStore_LoadToVersion_Upcast(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewStreamID()

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
		[]byte("v1"),
		event.WithSchemaVersion(1),
	)
	evt3, _ := event.NewEvent(
		"test.upcast",
		aggID,
		"Test",
		3,
		[]byte("skip"),
		event.WithSchemaVersion(2),
	)
	saveTestEvents(t, ctx, store, aggID, evt1, evt2, evt3)

	versioned, err := schema.NewVersionedStore(store, versionUpcaster{})
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := versioned.LoadToVersion(
		ctx,
		id.NewStreamRef(id.StreamType("Test"), aggID),
		2,
	)
	if err != nil {
		t.Fatalf("load to version: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}

	if string(loaded[0].Payload()) != "v2" {
		t.Errorf("payload[0] = %q, want v2", loaded[0].Payload())
	}

	if string(loaded[1].Payload()) != "v2" {
		t.Errorf("payload[1] = %q, want v2", loaded[1].Payload())
	}
}

func TestVersionedStore_LoadToTimestamp_Upcast(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewStreamID()

	ts := time.Now()

	evt1, _ := event.NewEvent(
		"test.upcast",
		aggID,
		"Test",
		1,
		[]byte("v1"),
		event.WithSchemaVersion(1),
		event.WithOccurredAt(ts),
	)
	evt2, _ := event.NewEvent(
		"test.upcast",
		aggID,
		"Test",
		2,
		[]byte("skip"),
		event.WithSchemaVersion(2),
		event.WithOccurredAt(ts.Add(time.Second)),
	)
	saveTestEvents(t, ctx, store, aggID, evt1, evt2)

	versioned, err := schema.NewVersionedStore(store, versionUpcaster{})
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := versioned.LoadToTimestamp(
		ctx,
		id.NewStreamRef(id.StreamType("Test"), aggID),
		ts.Add(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("load to timestamp: %v", err)
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

func TestVersionedStore_LoadToVersion_UpcastError(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewStreamID()

	evt, _ := event.NewEvent(
		"test.upcast",
		aggID,
		"Test",
		1,
		[]byte("v1"),
		event.WithSchemaVersion(1),
	)
	saveTestEvents(t, ctx, store, aggID, evt)

	failingUpcaster := &failingUpcaster{}
	versioned, err := schema.NewVersionedStore(store, failingUpcaster)
	if err != nil {
		t.Fatal(err)
	}

	_, err = versioned.LoadToVersion(
		ctx,
		id.NewStreamRef(id.StreamType("Test"), aggID),
		1,
	)
	if err == nil {
		t.Fatal("expected error from failing upcaster")
	}
}

func TestVersionedStore_LoadToTimestamp_UpcastError(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	aggID := id.NewStreamID()

	evt, _ := event.NewEvent(
		"test.upcast",
		aggID,
		"Test",
		1,
		[]byte("v1"),
		event.WithSchemaVersion(1),
		event.WithOccurredAt(time.Now()),
	)
	saveTestEvents(t, ctx, store, aggID, evt)

	failingUpcaster := &failingUpcaster{}
	versioned, err := schema.NewVersionedStore(store, failingUpcaster)
	if err != nil {
		t.Fatal(err)
	}

	_, err = versioned.LoadToTimestamp(
		ctx,
		id.NewStreamRef(id.StreamType("Test"), aggID),
		time.Now().Add(time.Hour),
	)
	if err == nil {
		t.Fatal("expected error from failing upcaster")
	}
}

type failingUpcaster struct{}

func (*failingUpcaster) SourceType() event.Type             { return "test.upcast" }
func (*failingUpcaster) SourceVersion() event.SchemaVersion { return 1 }
func (*failingUpcaster) Upcast(_ event.Event) (event.Event, error) {
	return nil, errors.New("upcast intentionally failed")
}

func saveTestEvents(
	t *testing.T,
	ctx context.Context,
	store *memory.MemoryStore,
	aggID id.StreamID,
	events ...event.Event,
) {
	t.Helper()

	if err := store.Save(
		ctx,
		id.NewStreamRef(id.StreamType("Test"), aggID),
		events,
		0,
	); err != nil {
		t.Fatalf("save: %v", err)
	}
}
