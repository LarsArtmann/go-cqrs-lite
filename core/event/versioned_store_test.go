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

func (u *testUpcaster) SourceType() event.Type                { return u.sourceType }
func (u *testUpcaster) SourceVersion() event.SchemaVersion    { return u.sourceVersion }
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
