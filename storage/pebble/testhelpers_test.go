package pebble

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type storeTestConfig = eventtest.StoreTestConfig

func issueStoreConfig() storeTestConfig {
	return eventtest.IssueStoreConfig()
}

func saveCfgEvent(
	t *testing.T,
	store event.Store,
	cfg storeTestConfig,
	streamID id.StreamID,
	evt event.Event,
) {
	eventtest.SaveEvent(t, store, cfg, streamID, evt)
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

func testEventStore_MetadataRoundtrip(
	t *testing.T,
	store event.Store,
	cfg storeTestConfig,
	envOverride string,
) {
	eventtest.TestStoreMetadataRoundtrip(t, store, cfg, envOverride)
}
