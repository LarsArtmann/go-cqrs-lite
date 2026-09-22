package system

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestCheckpointEngineResolution pins the deployment-declared checkpoint
// engine: the engine named "checkpoints" wins, else the first engine, else
// nil — and only engines carrying the Map ADT qualify.
func TestCheckpointEngineResolution(t *testing.T) {
	t.Parallel()

	primary := metaengine.NewMemoryEngine()
	dedicated := metaengine.NewMemoryEngine()
	dedicatedBackend := dedicated.(metaengine.MapBackend)
	primaryBackend := primary.(metaengine.MapBackend)

	named := &System{engines: []namedEngine{
		{name: "primary", engine: primary},
		{name: "checkpoints", engine: dedicated},
	}}
	if got := named.checkpointEngine(); got == nil || got != dedicatedBackend {
		t.Fatal("the engine named 'checkpoints' must win checkpoint storage")
	}

	fallback := &System{engines: []namedEngine{{name: "primary", engine: primary}}}
	if got := fallback.checkpointEngine(); got == nil || got != primaryBackend {
		t.Fatal("checkpoint storage must fall back to the first engine")
	}

	if got := (&System{}).checkpointEngine(); got != nil {
		t.Fatal("no engines must resolve to a nil checkpoint engine")
	}
}

// TestEngineCheckpointStoreRoundTrip covers the memory-engine value shape
