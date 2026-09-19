package system

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// checkpointCollection is the metaengine Map collection projection
// checkpoints persist in (ADR-0142: the last trivial satellite rides engines).
const checkpointCollection = "system_checkpoints"

// checkpointWire is the stored value shape: a JSON-portable projection of
// event.Checkpoint. The event ID travels as its string form so the record
// round-trips identically on every engine (memory engines keep typed values,
// SQL engines JSON-encode — the wire struct is the stable contract between
// them).
type checkpointWire struct {
	EventID     string    `json:"eventId"`
	ProcessedAt time.Time `json:"processedAt"`
}

// engineCheckpointStore persists projection checkpoints as entries of a
// metaengine Map collection on the deployment-declared engine — checkpoints
// survive restarts wherever the engine does, with zero dedicated
// infrastructure (ADR-0142).
type engineCheckpointStore struct {
	engine metaengine.MapBackend
}

// Save implements [event.CheckpointSink].
func (s *engineCheckpointStore) Save(
	ctx context.Context,
	projection string,
	cp event.Checkpoint,
) error {
	wire := checkpointWire{EventID: cp.EventID.String(), ProcessedAt: cp.ProcessedAt}

	if err := s.engine.MapSet(ctx, checkpointCollection, projection, wire); err != nil {
		return fmt.Errorf("system: save checkpoint %q: %w", projection, err)
	}

	return nil
}

// Load implements [event.CheckpointSource]. A projection with no stored
// checkpoint reads as zero (full replay), mirroring memoryCheckpointStore.
func (s *engineCheckpointStore) Load(
	ctx context.Context,
	projection string,
) (event.Checkpoint, error) {
	raw, found, err := s.engine.MapGet(ctx, checkpointCollection, projection)
	if err != nil {
		return event.Checkpoint{}, fmt.Errorf("system: load checkpoint %q: %w", projection, err)
	}

	if !found || raw == nil {
		return event.Checkpoint{}, nil
	}

	wire, err := reifyCheckpoint(raw)
	if err != nil {
		return event.Checkpoint{}, fmt.Errorf("system: decode checkpoint %q: %w", projection, err)
	}

	eventID, err := id.ParseEventID(wire.EventID)
	if err != nil {
		return event.Checkpoint{}, fmt.Errorf(
			"system: checkpoint %q event id %q: %w",
			projection,
			wire.EventID,
			err,
		)
	}

	return event.Checkpoint{EventID: eventID, ProcessedAt: wire.ProcessedAt}, nil
}

// Close is a no-op: the engine's lifecycle belongs to the System.
func (s *engineCheckpointStore) Close() error { return nil }

// Compile-time assertion: engineCheckpointStore implements event.CheckpointStore.
var _ event.CheckpointStore = (*engineCheckpointStore)(nil)

// reifyCheckpoint decodes a stored checkpoint across engine value shapes —
// the typed wire struct (memory engines), raw JSON bytes (JSONValue), or its
// decoded JSON form (map[string]any from SQL engines) — via JSON round-trip.
func reifyCheckpoint(raw any) (checkpointWire, error) {
	if w, ok := raw.(checkpointWire); ok {
		return w, nil
	}

	if jv, ok := raw.(metaengine.JSONValue); ok {
		var w checkpointWire
		if err := json.Unmarshal(jv, &w); err != nil {
			return checkpointWire{}, fmt.Errorf("unmarshal stored JSON value: %w", err)
		}

		return w, nil
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return checkpointWire{}, fmt.Errorf("marshal stored value: %w", err)
	}

	var w checkpointWire
	if err := json.Unmarshal(b, &w); err != nil {
		return checkpointWire{}, fmt.Errorf("unmarshal stored value: %w", err)
	}

	return w, nil
}

// checkpointEngine resolves the engine deployments designate for checkpoint
// storage: the engine named "checkpoints" when present, else the first
// configured engine — but ONLY when it carries the Map ADT (runtime
// assertion, the SupportsDedup pattern). Returns nil when no engine
// qualifies; the constructor then falls back to in-memory checkpoints (lost
// on restart, forcing full projection replays).
func (s *System) checkpointEngine() metaengine.MapBackend {
	var engine metaengine.Engine

	for _, ne := range s.engines {
		if ne.name == "checkpoints" {
			engine = ne.engine
			break
		}
	}

	if engine == nil && len(s.engines) > 0 {
		engine = s.engines[0].engine
	}

	if engine == nil {
		return nil
	}

	backend, ok := engine.(metaengine.MapBackend)
	if !ok {
		return nil
	}

	return backend
}
