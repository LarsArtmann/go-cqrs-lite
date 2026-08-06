package bbolt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
)

func TestContract_SaveAndLoad(t *testing.T) {
	backend := newTestBackend(t)
	cfg := eventtest.IssueStoreConfig()
	eventtest.TestStoreSaveAndLoad(t, backend.EventStore(), cfg)
}

func TestContract_ConcurrencyConflict(t *testing.T) {
	backend := newTestBackend(t)
	cfg := eventtest.IssueStoreConfig()
	eventtest.TestStoreConcurrencyConflict(t, backend.EventStore(), cfg)
}

func TestContract_AppendBatch(t *testing.T) {
	backend := newTestBackend(t)
	cfg := eventtest.IssueStoreConfig()
	eventtest.TestStoreAppendBatch(t, backend.EventStore(), cfg)
}

func TestContract_LoadFromVersion(t *testing.T) {
	backend := newTestBackend(t)
	cfg := eventtest.IssueStoreConfig()
	eventtest.TestStoreLoadFromVersion(t, backend.EventStore(), cfg)
}

func TestContract_MetadataRoundtrip(t *testing.T) {
	backend := newTestBackend(t)
	cfg := eventtest.IssueStoreConfig()
	eventtest.TestStoreMetadataRoundtrip(t, backend.EventStore(), cfg, "")
}

func TestContract_InterfaceCompliance(t *testing.T) {
	backend := newTestBackend(t)

	var _ event.Store = backend.EventStore()
	var _ event.EventSink = backend.EventStore()
	var _ event.EventSource = backend.EventStore()
}
