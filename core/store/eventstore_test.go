package store_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/store"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

type eventTestConfig struct {
	aggType event.AggregateType
	evtType event.Type
	payload func(version event.Version) []byte
}

func (c eventTestConfig) newEvent(
	t *testing.T,
	aggID id.AggregateID,
	version event.Version,
	opts ...event.Option,
) *event.ImmutableEvent {
	t.Helper()
	evt, err := event.NewEvent(c.evtType, aggID, c.aggType, version, c.payload(version), opts...)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	return evt
}

func defaultTestConfig() eventTestConfig {
	return eventTestConfig{
		aggType: "User",
		evtType: "UserCreated",
		payload: func(v event.Version) []byte {
			return []byte(fmt.Sprintf(`{"name":"user-%d"}`, v))
		},
	}
}

func newTestEventStore(t *testing.T) (store.Backend, event.Store) {
	t.Helper()
	backend := memory.NewBackend()
	return backend, store.NewEventStore(backend)
}

func TestEventStore_SaveAndLoad(t *testing.T) {
	t.Parallel()
	cfg := defaultTestConfig()
	_, es := newTestEventStore(t)

	aggID := id.NewAggregateID()
	evt := cfg.newEvent(t, aggID, 1)

	err := es.Save(context.Background(), event.NewAggregateRef(cfg.aggType, aggID), []event.Event{evt}, 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := es.Load(context.Background(), event.NewAggregateRef(cfg.aggType, aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
	if loaded[0].ID() != evt.ID() {
		t.Errorf("ID mismatch: got %v, want %v", loaded[0].ID(), evt.ID())
	}
}

func TestEventStore_ConcurrencyConflict(t *testing.T) {
	t.Parallel()
	cfg := defaultTestConfig()
	_, es := newTestEventStore(t)

	aggID := id.NewAggregateID()
	evt := cfg.newEvent(t, aggID, 1)

	err := es.Save(context.Background(), event.NewAggregateRef(cfg.aggType, aggID), []event.Event{evt}, 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	evt2 := cfg.newEvent(t, aggID, 2)
	err = es.Save(context.Background(), event.NewAggregateRef(cfg.aggType, aggID), []event.Event{evt2}, 0)
	if !errors.Is(err, event.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestEventStore_AppendBatch(t *testing.T) {
	t.Parallel()
	cfg := defaultTestConfig()
	_, es := newTestEventStore(t)

	aggID := id.NewAggregateID()
	evt1 := cfg.newEvent(t, aggID, 1)
	evt2 := cfg.newEvent(t, aggID, 2)

	err := es.AppendBatch(context.Background(), event.NewAggregateRef(cfg.aggType, aggID), []event.Event{evt1, evt2})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := es.Load(context.Background(), event.NewAggregateRef(cfg.aggType, aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}
	testhelpers.AssertEventVersion(t, loaded, 0, 1)
	testhelpers.AssertEventVersion(t, loaded, 1, 2)
}

func TestEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()
	cfg := defaultTestConfig()
	_, es := newTestEventStore(t)

	aggID := id.NewAggregateID()
	evt1 := cfg.newEvent(t, aggID, 1)
	evt2 := cfg.newEvent(t, aggID, 2)
	evt3 := cfg.newEvent(t, aggID, 3)

	err := es.AppendBatch(context.Background(), event.NewAggregateRef(cfg.aggType, aggID), []event.Event{evt1, evt2, evt3})
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := es.LoadFromVersion(context.Background(), event.NewAggregateRef(cfg.aggType, aggID), 1)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events after version 1, got %d", len(loaded))
	}
	testhelpers.AssertEventVersion(t, loaded, 0, 2)
	testhelpers.AssertEventVersion(t, loaded, 1, 3)
}

func TestEventStore_LoadToVersion(t *testing.T) {
	t.Parallel()
	cfg := defaultTestConfig()
	_, es := newTestEventStore(t)

	aggID := id.NewAggregateID()
	events := []event.Event{
		cfg.newEvent(t, aggID, 1),
		cfg.newEvent(t, aggID, 2),
		cfg.newEvent(t, aggID, 3),
	}

	err := es.AppendBatch(context.Background(), event.NewAggregateRef(cfg.aggType, aggID), events)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := es.LoadToVersion(context.Background(), event.NewAggregateRef(cfg.aggType, aggID), 2)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events up to version 2, got %d", len(loaded))
	}
}

func TestEventStore_Load_NotFound(t *testing.T) {
	t.Parallel()
	_, es := newTestEventStore(t)

	aggID := id.NewAggregateID()
	_, err := es.Load(context.Background(), event.NewAggregateRef("User", aggID))
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got %v", err)
	}
}

func TestEventStore_MetadataRoundtrip(t *testing.T) {
	t.Parallel()
	cfg := defaultTestConfig()
	_, es := newTestEventStore(t)

	aggID := id.NewAggregateID()
	correlationID := id.CorrelationID(id.New[id.CorrelationID]())
	evt := cfg.newEvent(t, aggID, 1, event.WithCorrelationID(correlationID))

	err := es.Save(context.Background(), event.NewAggregateRef(cfg.aggType, aggID), []event.Event{evt}, 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := es.Load(context.Background(), event.NewAggregateRef(cfg.aggType, aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded[0].Metadata().CorrelationID != correlationID {
		t.Errorf("CorrelationID: got %v, want %v", loaded[0].Metadata().CorrelationID, correlationID)
	}
}

func TestEventStore_Save_Empty(t *testing.T) {
	t.Parallel()
	_, es := newTestEventStore(t)

	aggID := id.NewAggregateID()
	err := es.Save(context.Background(), event.NewAggregateRef("User", aggID), nil, 0)
	if err != nil {
		t.Fatalf("Save empty: %v", err)
	}
}
