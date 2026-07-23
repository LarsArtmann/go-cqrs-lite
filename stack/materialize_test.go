package stack_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func buildTestMessage(evt event.Event, eventType string) *message.Message {
	msg := message.NewMessage(evt.ID().String(), evt.Payload())
	msg.Metadata.Set("event_type", eventType)
	msg.Metadata.Set("event_id", evt.ID().String())
	msg.Metadata.Set("aggregate_id", evt.StreamID().String())
	msg.Metadata.Set("aggregate_type", string(evt.StreamType()))
	msg.Metadata.Set("version", "1")
	msg.Metadata.Set("schema_version", "1")

	return msg
}

type stringKey string

func (s stringKey) String() string { return string(s) }

type userView struct {
	Name    string
	Email   string
	Deleted bool
}

func (u *userView) IsTombstoned() bool { return u.Deleted }

func TestMaterialize_OnCreate(t *testing.T) {
	t.Parallel()

	memStore := kv.NewMemStore()
	defer memStore.Close()

	ts := kv.NewTypedStore[userView, stringKey](memStore)

	mat := stack.Materialize[userView, stringKey]{
		Store:        ts,
		KeyFromEvent: func(evt event.Event) (stringKey, error) { return stringKey(evt.StreamID().String()), nil },
		OnCreate: func(_ context.Context, evt event.Event) (*userView, error) {
			return &userView{Name: "from-event", Email: "test@example.com"}, nil
		},
	}

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent(event.Type("user.created"), aggID, "User", event.Version(1), nil)

	// Simulate the handler processing an event.
	msg := buildTestMessage(evt, "user.created")
	handler := mat.HandlerFunc()
	if err := handler(msg); err != nil {
		t.Fatalf("HandlerFunc: %v", err)
	}

	view, err := mat.View(context.Background(), stringKey(aggID.String()))
	if err != nil {
		t.Fatalf("View: %v", err)
	}

	if view.Name != "from-event" {
		t.Fatalf("expected Name 'from-event', got %s", view.Name)
	}
}

func TestMaterialize_TombstonePolicy(t *testing.T) {
	t.Parallel()

	memStore := kv.NewMemStore()
	defer memStore.Close()

	ts := kv.NewTypedStore[userView, stringKey](memStore)

	ctx := context.Background()
	_ = ts.Set(ctx, stringKey("active"), &userView{Name: "Alice", Deleted: false})
	_ = ts.Set(ctx, stringKey("deleted"), &userView{Name: "Bob", Deleted: true})

	// Test ExcludeTombstoned (default).
	results, err := ts.Scan(ctx, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	excluded := stack.FilterTombstoned(results, stack.ExcludeTombstoned)
	if len(excluded) != 1 {
		t.Fatalf("expected 1 active record, got %d", len(excluded))
	}

	onlyTombstoned := stack.FilterTombstoned(results, stack.OnlyTombstoned)
	if len(onlyTombstoned) != 1 {
		t.Fatalf("expected 1 tombstoned record, got %d", len(onlyTombstoned))
	}

	all := stack.FilterTombstoned(results, stack.IncludeTombstoned)
	if len(all) != 2 {
		t.Fatalf("expected 2 total records, got %d", len(all))
	}
}

type failingStore struct{ kv.Store }

var errFailingStore = errors.New("simulated database failure")

func (f *failingStore) Get(_ context.Context, key []byte) ([]byte, error) {
	return nil, errFailingStore
}

func (f *failingStore) Has(_ context.Context, key []byte) (bool, error) {
	return false, errFailingStore
}

func (f *failingStore) NewIterator(_ context.Context, prefix []byte) (kv.Iterator, error) {
	return nil, errFailingStore
}

func TestMaterialize_StoreGetErrorPropagates(t *testing.T) {
	t.Parallel()

	memStore := kv.NewMemStore()
	defer memStore.Close()

	ts := kv.NewTypedStore[userView, stringKey](&failingStore{Store: memStore})

	mat := stack.Materialize[userView, stringKey]{
		Store:        ts,
		KeyFromEvent: func(evt event.Event) (stringKey, error) { return stringKey(evt.StreamID().String()), nil },
		OnCreate: func(_ context.Context, _ event.Event) (*userView, error) {
			t.Fatal("OnCreate should NOT be called when store returns a real error")

			return nil, nil
		},
	}

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent(event.Type("user.created"), aggID, "User", event.Version(1), nil)

	msg := buildTestMessage(evt, "user.created")
	handler := mat.HandlerFunc()

	err := handler(msg)
	if err == nil {
		t.Fatal("expected error from store failure, got nil")
	}

	if !errors.Is(err, errFailingStore) {
		t.Fatalf("expected error to wrap errFailingStore, got: %v", err)
	}
}
