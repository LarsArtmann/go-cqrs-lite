package pebble

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

// CheckpointStore implements event.CheckpointStore backed by Pebble.
//
// Checkpoints (the last processed event position for a projection) are stored
// as CBOR-encoded envelopes under the key pattern
// cqrs_checkpoint:{projectionName}. One checkpoint per projection is retained;
// saving overwrites the prior value.
//
// The store shares the Pebble DB with other stores (event, snapshot) via
// disjoint key prefixes, so a single *pebble.DB can back the full CQRS stack.
type CheckpointStore struct {
	storeBase
}

// CheckpointOption configures a CheckpointStore.
type CheckpointOption func(*CheckpointStore)

// WithCheckpointAsyncWrites disables sync writes for higher throughput at the
// cost of durability guarantees. Use only when a lost checkpoint on crash is
// acceptable (checkpoints can be rebuilt by replaying the journal).
func WithCheckpointAsyncWrites() CheckpointOption {
	return func(s *CheckpointStore) { s.syncWrites = false }
}

// WithCheckpointPrefix overrides the default key prefix ("cqrs_checkpoint:").
// Useful when multiple logical checkpoint stores share one Pebble DB.
func WithCheckpointPrefix(p string) CheckpointOption {
	return func(s *CheckpointStore) { s.prefix = p }
}

// NewCheckpointStore creates a new CheckpointStore using an existing Pebble DB.
// Panics if db is nil.
func NewCheckpointStore(
	database *pebble.DB,
	logger *slog.Logger,
	opts ...CheckpointOption,
) *CheckpointStore {
	if database == nil {
		panic("pebble: NewCheckpointStore called with nil db")
	}

	s := &CheckpointStore{
		storeBase: storeBase{
			db:         database,
			logger:     logger,
			prefix:     "cqrs_checkpoint:",
			syncWrites: true,
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Save persists the checkpoint for a projection. Overwrites any prior value.
func (s *CheckpointStore) Save(
	ctx context.Context,
	projectionName string,
	checkpoint event.Checkpoint,
) error {
	_, span := startProjectionSpan(ctx, "pebble.checkpoint.save", projectionName)
	defer span.End()

	if projectionName == "" {
		return event.NewRejection("pebble.empty_projection_name",
			"projection name must not be empty")
	}

	key := s.checkpointKey(projectionName)

	data, err := serializeCheckpoint(checkpoint)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapCorruption(err, "pebble.serialize_checkpoint",
			"serialize checkpoint for projection "+projectionName)
	}

	err = s.db.Set(key, data, s.writeOptions())
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.WrapInfrastructure(err, "pebble.write_checkpoint",
			"write checkpoint for projection "+projectionName)
	}

	return nil
}

// Load returns the last checkpoint for a projection.
// Returns a zero-value event.Checkpoint when no checkpoint exists, so callers
// can distinguish "first run" (zero) from a stored checkpoint without checking
// errors.
func (s *CheckpointStore) Load(
	ctx context.Context,
	projectionName string,
) (event.Checkpoint, error) {
	_, span := startProjectionSpan(ctx, "pebble.checkpoint.load", projectionName)
	defer span.End()

	if projectionName == "" {
		return event.Checkpoint{
				EventID: id.EventID{},
			}, event.NewRejection("pebble.empty_projection_name",
				"projection name must not be empty")
	}

	key := s.checkpointKey(projectionName)

	val, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return event.Checkpoint{
				EventID: id.EventID{},
			}, nil
		}

		cqrsotel.RecordError(span, err)

		return event.Checkpoint{
				EventID: id.EventID{},
			}, event.WrapInfrastructure(err, "pebble.read_checkpoint",
				"read checkpoint for projection "+projectionName)
	}

	defer func() { _ = closer.Close() }()

	buf := make([]byte, len(val))
	copy(buf, val)

	checkpoint, err := deserializeCheckpoint(buf)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.Checkpoint{
				EventID: id.EventID{},
			}, event.WrapCorruption(err, "pebble.deserialize_checkpoint",
				"deserialize checkpoint for projection "+projectionName)
	}

	return checkpoint, nil
}

// Close is a no-op; the underlying *pebble.DB is owned by the caller.
// Implemented to satisfy io.Closer for event.CheckpointSink/Source.
func (s *CheckpointStore) Close() error { return nil }

func (s *CheckpointStore) checkpointKey(projectionName string) []byte {
	return fmt.Appendf(nil, "%s%s", s.prefix, projectionName)
}

// serializableCheckpoint is the CBOR envelope for stored checkpoints.
type serializableCheckpoint struct {
	EventID     id.EventID `json:"event_id"`
	ProcessedAt int64      `json:"processed_at"`
}

func serializeCheckpoint(checkpoint event.Checkpoint) ([]byte, error) {
	s := serializableCheckpoint{
		EventID:     checkpoint.EventID,
		ProcessedAt: checkpoint.ProcessedAt.UnixNano(),
	}

	return pebbleEncMode.Marshal(s)
}

func deserializeCheckpoint(data []byte) (event.Checkpoint, error) {
	var s serializableCheckpoint

	if isCBOR(data) {
		err := pebbleDecMode.Unmarshal(data, &s)
		if err != nil {
			return event.Checkpoint{}, event.Wrapf(
				err,
				event.Corruption,
				"pebble.checkpoint_cbor",
				"cbor unmarshal checkpoint",
			)
		}
	} else {
		// Legacy JSON fallback for checkpoints written before CBOR migration.
		err := json.Unmarshal(data, &s)
		if err != nil {
			return event.Checkpoint{}, event.Wrapf(
				err,
				event.Corruption,
				"pebble.checkpoint_json",
				"json unmarshal checkpoint",
			)
		}
	}

	return event.Checkpoint{
		EventID:     s.EventID,
		ProcessedAt: time.Unix(0, s.ProcessedAt),
	}, nil
}

var (
	_ event.CheckpointSink   = (*CheckpointStore)(nil)
	_ event.CheckpointSource = (*CheckpointStore)(nil)
	_ event.CheckpointStore  = (*CheckpointStore)(nil)
)
