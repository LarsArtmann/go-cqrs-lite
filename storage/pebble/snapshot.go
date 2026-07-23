package pebble

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// SnapshotStore implements snapshot.SnapshotStore backed by Pebble.
//
// One snapshot per aggregate is retained: saving a newer version overwrites the
// prior snapshot, while saving an older version is silently ignored (matching
// the memory implementation). Snapshots are stored as CBOR-encoded envelopes
// under the key pattern cqrs_snapshot:{streamType}:{streamID}.
//
// The store shares the Pebble DB with other stores (event, checkpoint) via
// disjoint key prefixes, so a single *pebble.DB can back the full CQRS stack.
type SnapshotStore struct {
	storeBase
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
// Returns ErrNilDatabase if db is nil.
func NewSnapshotStore(
	database *pebble.DB,
	logger *slog.Logger,
	opts ...SnapshotOption,
) (*SnapshotStore, error) {
	if database == nil {
		return nil, ErrNilDatabase
	}

	s := &SnapshotStore{
		storeBase: storeBase{
			db:         database,
			logger:     logger,
			prefix:     "cqrs_snapshot:",
			syncWrites: true,
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Save stores the snapshot, overwriting any existing snapshot for the same
// aggregate. Snapshots older than the currently stored version are silently
// ignored to prevent state regressions.
func (s *SnapshotStore) Save(
	ctx context.Context,
	snap snapshot.Snapshot,
) error {
	_, span := startAggregateSpan(ctx, "pebble.snapshot.save",
		id.NewAggregateRef(snap.StreamType, snap.StreamID),
		cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, snap.Version.Int()))
	defer span.End()

	key := s.snapshotKey(snap.StreamType, snap.StreamID)

	existing, found, err := s.loadRaw(key)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return err
	}

	if found && existing.Version > snap.Version.Int() {
		if s.logger != nil {
			s.logger.Debug(
				"ignoring older snapshot",
				slog.String("aggregate_type", string(snap.StreamType)),
				slog.String("aggregate_id", snap.StreamID.String()),
				slog.Int("existing_version", existing.Version),
				slog.Int("rejected_version", snap.Version.Int()),
			)
		}

		return nil
	}

	data, err := serializeSnapshot(snap)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapCorruption(err, "pebble.serialize_snapshot",
			fmt.Sprintf("serialize snapshot for %s %s", snap.StreamType, snap.StreamID))
	}

	err = s.db.Set(key, data, s.writeOptions())
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapInfrastructure(err, "pebble.write_snapshot",
			fmt.Sprintf("write snapshot for %s %s", snap.StreamType, snap.StreamID))
	}

	return nil
}

// Load returns the latest snapshot for the aggregate.
// startSnapshotSpan opens an aggregate span named spanName and returns it
// alongside the computed snapshot key. Shared by Load and Delete so the
// span+key boilerplate stays in one place.
func (s *SnapshotStore) startSnapshotSpan(
	ctx context.Context,
	spanName string,
	ref id.StreamRef,
) (cqrsotel.Span, []byte) {
	_, span := startAggregateSpan(ctx, spanName, ref)

	return span, s.snapshotKey(ref.Type, ref.ID)
}

// Load returns snapshot.ErrSnapshotNotFound when no snapshot exists.
func (s *SnapshotStore) Load(
	ctx context.Context,
	ref id.StreamRef,
) (*snapshot.Snapshot, error) {
	span, key := s.startSnapshotSpan(ctx, "pebble.snapshot.load", ref)
	defer span.End()

	raw, found, err := s.loadRaw(key)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	if !found {
		return nil, snapshot.ErrSnapshotNotFound
	}

	span.SetAttributes(cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, raw.Version))

	return raw.toSnapshot(ref), nil
}

// LoadAtVersion returns the snapshot for the aggregate whose version is less
// than or equal to the requested version. Returns snapshot.ErrSnapshotNotFound
// when no such snapshot exists.
func (s *SnapshotStore) LoadAtVersion(
	ctx context.Context,
	ref id.StreamRef,
	version event.Version,
) (*snapshot.Snapshot, error) {
	_, span := startAggregateSpan(ctx, "pebble.snapshot.load_at_version", ref,
		cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, version.Int()))
	defer span.End()

	key := s.snapshotKey(ref.Type, ref.ID)

	raw, found, err := s.loadRaw(key)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	if !found || raw.Version > version.Int() {
		return nil, snapshot.ErrSnapshotNotFound
	}

	return raw.toSnapshot(ref), nil
}

// Delete removes the snapshot for the given aggregate. A no-op if no snapshot
// exists.
func (s *SnapshotStore) Delete(
	ctx context.Context,
	ref id.StreamRef,
) error {
	span, key := s.startSnapshotSpan(ctx, "pebble.snapshot.delete", ref)
	defer span.End()

	err := s.db.Delete(key, s.writeOptions())
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapInfrastructure(err, "pebble.delete_snapshot",
			fmt.Sprintf("delete snapshot for %s %s", ref.Type, ref.ID))
	}

	return nil
}

// Close is a no-op; the underlying *pebble.DB is owned by the caller.
// Implemented to satisfy io.Closer for snapshot.SnapshotSink/Source.
func (s *SnapshotStore) Close() error { return nil }

func (s *SnapshotStore) snapshotKey(
	aggType id.StreamType,
	aggID id.StreamID,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s", s.prefix, aggType, aggID)
}

// loadRaw reads and deserializes a snapshot by key.
// Returns (nil, false, nil) when the key does not exist (not an error).
func (s *SnapshotStore) loadRaw(key []byte) (*serializableSnapshot, bool, error) {
	val, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, false, nil
		}

		return nil, false, errorfamily.WrapInfrastructure(err, "pebble.read_snapshot",
			"read snapshot at key "+string(key))
	}

	defer func() { _ = closer.Close() }()

	// Copy because Pebble buffers are only valid until closer.Close().
	buf := make([]byte, len(val))
	copy(buf, val)

	stored, err := deserializeSnapshot(buf)
	if err != nil {
		return nil, false, errorfamily.WrapCorruption(err, "pebble.deserialize_snapshot",
			"deserialize snapshot at key "+string(key))
	}

	return stored, true, nil
}

// serializableSnapshot is the CBOR envelope for stored snapshots.
// Timestamps use UnixNano for deterministic, locale-independent ordering.
type serializableSnapshot struct {
	StreamID   id.StreamID `json:"aggregate_id"`
	StreamType string      `json:"aggregate_type"`
	Version    int         `json:"version"`
	State      []byte      `json:"state"`
	CreatedAt  int64       `json:"created_at"`
}

func (s *serializableSnapshot) toSnapshot(ref id.StreamRef) *snapshot.Snapshot {
	return &snapshot.Snapshot{
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    event.Version(s.Version),
		State:      s.State,
		CreatedAt:  time.Unix(0, s.CreatedAt).UTC(),
	}
}

func serializeSnapshot(snap snapshot.Snapshot) ([]byte, error) {
	s := serializableSnapshot{
		StreamID:   snap.StreamID,
		StreamType: string(snap.StreamType),
		Version:    snap.Version.Int(),
		State:      snap.State,
		CreatedAt:  snap.CreatedAt.UnixNano(),
	}

	return marshalCBOR(s)
}

func deserializeSnapshot(data []byte) (*serializableSnapshot, error) {
	var s serializableSnapshot

	if isCBOR(data) {
		err := unmarshalCBOR(data, &s)
		if err != nil {
			return nil, errorfamily.Wrapf(
				err,
				errorfamily.Corruption,
				"pebble.snapshot_cbor",
				"cbor unmarshal snapshot",
			)
		}
	} else {
		// Legacy JSON fallback for snapshots written before CBOR migration.
		err := json.Unmarshal(data, &s, json.MatchCaseInsensitiveNames(true))
		if err != nil {
			return nil, errorfamily.Wrapf(
				err,
				errorfamily.Corruption,
				"pebble.snapshot_json",
				"json unmarshal snapshot",
			)
		}
	}

	return &s, nil
}

var (
	_ snapshot.SnapshotSink   = (*SnapshotStore)(nil)
	_ snapshot.SnapshotSource = (*SnapshotStore)(nil)
	_ snapshot.SnapshotStore  = (*SnapshotStore)(nil)
)
