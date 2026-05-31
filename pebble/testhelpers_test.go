package pebble

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/event/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id"
)

type storeTestConfig = eventtest.StoreTestConfig

func issueStoreConfig() storeTestConfig {
	return eventtest.IssueStoreConfig()
}

func saveCfgEvent(
	t *testing.T,
	store event.Store,
	cfg storeTestConfig,
	aggID id.AggregateID,
	evt *event.ImmutableEvent,
) {
	eventtest.SaveEvent(t, store, cfg, aggID, evt)
}

func testEventStore_SaveAndLoad(t *testing.T, store event.Store, cfg storeTestConfig) {
	eventtest.TestStoreSaveAndLoad(t, store, cfg)
}

func testEventStore_ConcurrencyConflict(t *testing.T, store event.Store, cfg storeTestConfig) {
	eventtest.TestStoreConcurrencyConflict(t, store, cfg)
}

func testEventStore_AppendBatch(t *testing.T, store event.Store, cfg storeTestConfig) {
	eventtest.TestStoreAppendBatch(t, store, cfg)
}

func testEventStore_LoadFromVersion(t *testing.T, store event.Store, cfg storeTestConfig) {
	eventtest.TestStoreLoadFromVersion(t, store, cfg)
}

func testEventStore_MetadataRoundtrip(
	t *testing.T,
	store event.Store,
	cfg storeTestConfig,
	customEnv string,
) {
	eventtest.TestStoreMetadataRoundtrip(t, store, cfg, customEnv)
}
