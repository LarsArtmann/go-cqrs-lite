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
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
)

// SnapshotStore implements snapshot.SnapshotStore backed by Pebble.
//
// One snapshot per aggregate is retained: saving a newer version overwrites the
// prior snapshot, while saving an older version is silently ignored (matching
// the memory implementation). Snapshots are stored as CBOR-encoded envelopes
// under the key pattern cqrs_snapshot:{aggregateType}:{aggregateID}.
//
// The store shares the Pebble DB with other stores (event, checkpoint) via
// disjoint key prefixes, so a single *pebble.DB can back the full CQRS stack.
type SnapshotStore struct {
	db         *pebble.DB
	logger     *slog.Logger
	prefix     string
	syncWrites bool
}

// SnapshotOption configures a SnapshotStore.
type SnapshotOption func(*SnapshotStore)

// WithSnapshotAsyncWrites disables sync writes for higher throughput at the
// cost of durability guarantees. Use only when a lost snapshot on crash is
// acceptable (snapshots are an optimization, not a source of truth).
func WithSnapshotAsyncWrites() SnapshotOption {
	return func(s *SnapshotStore) { s.syncWrites = false }
}

// WithSnapshotPrefix overrides the default key prefix ("cqrs_snapshot:").
// Useful when multiple logical snapshot stores share one Pebble DB.
func WithSnapshotPrefix(p string) SnapshotOption {
	return func(s *SnapshotStore) { s.prefix = p }
}

// NewSnapshotStore creates a new SnapshotStore using an existing Pebble DB.
// Panics if db is nil.
func NewSnapshotStore(
	database *pebble.DB,
	logger *slog.Logger,
	opts ...SnapshotOption,
) *SnapshotStore {
	if database == nil {
		panic("pebble: NewSnapshotStore called with nil db")
	}

	s := &SnapshotStore{ //nolint:exhaustruct // zero values are valid defaults
		db:         database,
		logger:     logger,
		prefix:     "cqrs_snapshot:",
		syncWrites: true,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Save stores the snapshot, overwriting any existing snapshot for the same
// aggregate. Snapshots older than the currently stored version are silently
// ignored to prevent state regressions.
func (s *SnapshotStore) Save(
	_ context.Context,
	snap snapshot.Snapshot,
) error {
	key := s.snapshotKey(snap.AggregateType, snap.AggregateID)

	existing, err := s.loadRaw(key)
	if err != nil {
		return err
	}

	if existing != nil && existing.Version > snap.Version.Int() {
		if s.logger != nil {
			s.logger.Debug(
				"ignoring older snapshot",
				slog.String("aggregate_type", string(snap.AggregateType)),
				slog.String("aggregate_id", snap.AggregateID.String()),
				slog.Int("existing_version", existing.Version),
				slog.Int("rejected_version", snap.Version.Int()),
			)
		}

		return nil
	}

	data, err := serializeSnapshot(snap)
	if err != nil {
		return event.WrapCorruption(err, "pebble.serialize_snapshot",
			fmt.Sprintf("serialize snapshot for %s %s", snap.AggregateType, snap.AggregateID))
	}

	if err := s.db.Set(key, data, s.writeOptions()); err != nil {
		return event.WrapInfrastructure(err, "pebble.write_snapshot",
			fmt.Sprintf("write snapshot for %s %s", snap.AggregateType, snap.AggregateID))
	}

	return nil
}

// Load returns the latest snapshot for the aggregate.
// Returns snapshot.ErrSnapshotNotFound when no snapshot exists.
func (s *SnapshotStore) Load(
	_ context.Context,
	ref event.AggregateRef,
) (*snapshot.Snapshot, error) {
	key := s.snapshotKey(ref.Type, ref.ID)

	raw, err := s.loadRaw(key)
	if err != nil {
		return nil, err
	}

	if raw == nil {
		return nil, snapshot.ErrSnapshotNotFound
	}

	return raw.toSnapshot(ref), nil
}

// LoadAtVersion returns the snapshot for the aggregate whose version is less
// than or equal to the requested version. Returns snapshot.ErrSnapshotNotFound
// when no such snapshot exists.
func (s *SnapshotStore) LoadAtVersion(
	_ context.Context,
	ref event.AggregateRef,
	version event.Version,
) (*snapshot.Snapshot, error) {
	key := s.snapshotKey(ref.Type, ref.ID)

	raw, err := s.loadRaw(key)
	if err != nil {
		return nil, err
	}

	if raw == nil || raw.Version > version.Int() {
		return nil, snapshot.ErrSnapshotNotFound
	}

	return raw.toSnapshot(ref), nil
}

// Delete removes the snapshot for the given aggregate. A no-op if no snapshot
// exists.
func (s *SnapshotStore) Delete(
	_ context.Context,
	ref event.AggregateRef,
) error {
	key := s.snapshotKey(ref.Type, ref.ID)

	if err := s.db.Delete(key, s.writeOptions()); err != nil {
		return event.WrapInfrastructure(err, "pebble.delete_snapshot",
			fmt.Sprintf("delete snapshot for %s %s", ref.Type, ref.ID))
	}

	return nil
}

// Close is a no-op; the underlying *pebble.DB is owned by the caller.
// Implemented to satisfy io.Closer for snapshot.SnapshotSink/Source.
func (s *SnapshotStore) Close() error { return nil }

func (s *SnapshotStore) snapshotKey(
	aggType event.AggregateType,
	aggID id.AggregateID,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s", s.prefix, aggType, aggID)
}

func (s *SnapshotStore) writeOptions() *pebble.WriteOptions {
	if s.syncWrites {
		return pebble.Sync
	}

	return nil
}

func (s *SnapshotStore) loadRaw(key []byte) (*storedSnapshot, error) {
	val, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, nil
		}

		return nil, event.WrapInfrastructure(err, "pebble.read_snapshot",
			"read snapshot at key "+string(key))
	}

	defer func() { _ = closer.Close() }()

	// Copy because Pebble buffers are only valid until closer.Close().
	buf := make([]byte, len(val))
	copy(buf, val)

	stored, err := deserializeSnapshot(buf)
	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.deserialize_snapshot",
			"deserialize snapshot at key "+string(key))
	}

	return stored, nil
}

// serializableSnapshot is the CBOR envelope for stored snapshots.
// Timestamps use UnixNano for deterministic, locale-independent ordering.
type serializableSnapshot struct {
	AggregateID   id.AggregateID `json:"aggregate_id"`   //nolint:tagliatelle // on-disk format uses snake_case
	AggregateType string         `json:"aggregate_type"` //nolint:tagliatelle // on-disk format uses snake_case
	Version       int            `json:"version"`
	State         []byte         `json:"state"`
	CreatedAt     int64          `json:"created_at"` //nolint:tagliatelle // on-disk format uses snake_case
}

// storedSnapshot is a deserialized snapshot awaiting conversion.
type storedSnapshot serializableSnapshot

func (s *storedSnapshot) toSnapshot(ref event.AggregateRef) *snapshot.Snapshot {
	return &snapshot.Snapshot{
		AggregateID:   ref.ID,
		AggregateType: ref.Type,
		Version:       event.Version(s.Version),
		State:         s.State,
		CreatedAt:     time.Unix(0, s.CreatedAt),
	}
}

func serializeSnapshot(snap snapshot.Snapshot) ([]byte, error) {
	s := serializableSnapshot{
		AggregateID:   snap.AggregateID,
		AggregateType: string(snap.AggregateType),
		Version:       snap.Version.Int(),
		State:         snap.State,
		CreatedAt:     snap.CreatedAt.UnixNano(),
	}

	return pebbleEncMode.Marshal(s) //nolint:wrapcheck // storage serialization, not domain error
}

func deserializeSnapshot(data []byte) (*storedSnapshot, error) {
	var s serializableSnapshot

	if isCBOR(data) {
		if err := pebbleDecMode.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("cbor unmarshal snapshot: %w", err)
		}
	} else {
		// Legacy JSON fallback for snapshots written before CBOR migration.
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("json unmarshal snapshot: %w", err)
		}
	}

	stored := storedSnapshot(s)

	return &stored, nil
}

var (
	_ snapshot.SnapshotSink   = (*SnapshotStore)(nil)
	_ snapshot.SnapshotSource = (*SnapshotStore)(nil)
	_ snapshot.SnapshotStore  = (*SnapshotStore)(nil)
)
