package pebble

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

type storeTestConfig struct {
	aggType event.AggregateType
	evtType event.Type
	payload func(version event.Version) []byte
}

func (c storeTestConfig) newTestEvent(
	t *testing.T,
	aggID id.AggregateID,
	version event.Version,
	extraOpts ...event.Option,
) *event.ImmutableEvent {
	t.Helper()

	evt, err := event.NewEvent(
		c.evtType,
		aggID,
		c.aggType,
		version,
		c.payload(version),
		extraOpts...,
	)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	return evt
}

func newStoreTestConfig(
	aggType event.AggregateType,
	evtType event.Type,
	jsonField, valuePrefix string,
) storeTestConfig {
	return storeTestConfig{
		aggType: aggType,
		evtType: evtType,
		payload: func(v event.Version) []byte {
			return []byte(fmt.Sprintf(`{"%s":"%s-%d"}`, jsonField, valuePrefix, v))
		},
	}
}

func issueStoreConfig() storeTestConfig {
	return newStoreTestConfig("Issue", "IssueCreated", "title", "test")
}

func saveCfgEvent(
	t *testing.T,
	store event.Store,
	cfg storeTestConfig,
	aggID id.AggregateID,
	evt *event.ImmutableEvent,
) {
	t.Helper()

	err := store.Save(
		context.Background(),
		event.NewAggregateRef(cfg.aggType, aggID),
		[]event.Event{evt},
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func testEventStore_SaveAndLoad(t *testing.T, store event.Store, cfg storeTestConfig) {
	t.Helper()

	aggID := id.NewAggregateID()
	evt := cfg.newTestEvent(t, aggID, 1)

	err := store.Save(
		context.Background(),
		event.NewAggregateRef(cfg.aggType, aggID),
		[]event.Event{evt},
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), event.NewAggregateRef(cfg.aggType, aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if loaded[0].Type() != cfg.evtType {
		t.Errorf("Type = %q, want %q", loaded[0].Type(), cfg.evtType)
	}

	if loaded[0].ID() != evt.ID() {
		t.Errorf("ID = %v, want %v", loaded[0].ID(), evt.ID())
	}
}

func testEventStore_ConcurrencyConflict(t *testing.T, store event.Store, cfg storeTestConfig) {
	t.Helper()

	aggID := id.NewAggregateID()
	evt := cfg.newTestEvent(t, aggID, 1)

	saveCfgEvent(t, store, cfg, aggID, evt)

	evt2 := cfg.newTestEvent(t, aggID, 2)

	err := store.Save(
		context.Background(),
		event.NewAggregateRef(cfg.aggType, aggID),
		[]event.Event{evt2},
		event.Version(0),
	)
	if !errors.Is(err, event.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func testEventStore_AppendBatch(t *testing.T, store event.Store, cfg storeTestConfig) {
	t.Helper()

	aggID := id.NewAggregateID()
	evt1 := cfg.newTestEvent(t, aggID, 1)
	evt2 := cfg.newTestEvent(t, aggID, 2)

	err := store.AppendBatch(
		context.Background(),
		event.NewAggregateRef(cfg.aggType, aggID),
		[]event.Event{evt1, evt2},
	)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.Load(context.Background(), event.NewAggregateRef(cfg.aggType, aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}

	testhelpers.AssertEventVersion(t, loaded, 0, 1)
	testhelpers.AssertEventVersion(t, loaded, 1, 2)
}

func testEventStore_LoadFromVersion(t *testing.T, store event.Store, cfg storeTestConfig) {
	t.Helper()

	aggID := id.NewAggregateID()
	evt1 := cfg.newTestEvent(t, aggID, 1)
	evt2 := cfg.newTestEvent(t, aggID, 2)
	evt3 := cfg.newTestEvent(t, aggID, 3)

	err := store.AppendBatch(
		context.Background(),
		event.NewAggregateRef(cfg.aggType, aggID),
		[]event.Event{evt1, evt2, evt3},
	)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.LoadFromVersion(
		context.Background(),
		event.NewAggregateRef(cfg.aggType, aggID),
		event.Version(1),
	)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events after version 1, got %d", len(loaded))
	}

	testhelpers.AssertEventVersion(t, loaded, 0, 2)
}

func testEventStore_MetadataRoundtrip(
	t *testing.T,
	store event.Store,
	cfg storeTestConfig,
	customEnv string,
) {
	t.Helper()

	aggID := id.NewAggregateID()
	cid := id.NewCorrelationID()
	uid := id.NewUserID()

	evt := cfg.newTestEvent(
		t, aggID, 1,
		event.WithCorrelationID(cid),
		event.WithUserID(uid),
		event.WithCustom("env", customEnv),
	)

	saveCfgEvent(t, store, cfg, aggID, evt)

	loaded, err := store.Load(context.Background(), event.NewAggregateRef(cfg.aggType, aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	meta := loaded[0].Metadata()
	if meta == nil {
		t.Fatal("Metadata is nil")
	}

	if meta.CorrelationID != cid {
		t.Errorf("CorrelationID = %v, want %v", meta.CorrelationID, cid)
	}

	if meta.UserID != uid {
		t.Errorf("UserID = %v, want %v", meta.UserID, uid)
	}

	if meta.Custom["env"] != customEnv {
		t.Errorf("Custom[env] = %q, want %q", meta.Custom["env"], customEnv)
	}
}
