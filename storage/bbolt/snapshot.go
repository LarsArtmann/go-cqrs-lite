package bbolt

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// SnapshotStore implements snapshot.SnapshotStore backed by bbolt.
type SnapshotStore struct {
	storeBase
}

func NewSnapshotStore(database *bolt.DB, logger *slog.Logger) (*SnapshotStore, error) {
	if database == nil {
		return nil, ErrNilDatabase
	}

	return &SnapshotStore{storeBase: storeBase{db: database, logger: logger}}, nil
}

// Save persists a snapshot to the bbolt store.
//
// cqrs-lint:ignore(A023) library code or intentional pattern
func (s *SnapshotStore) Save(ctx context.Context, snap snapshot.Snapshot) error { //nolint:godoclint
	_, span := startStreamSpan(ctx, "bbolt.snapshot.save",
		id.NewStreamRef(snap.StreamType, snap.StreamID))
	defer span.End()

	key := snapshotKey(snap.StreamType, snap.StreamID)

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketSnapshots))
		if bucket == nil {
			return errorfamily.NewInfrastructure("bbolt.bucket_missing",
				"snapshots bucket not found")
		}

		existing := bucket.Get(key)
		if existing != nil {
			var prev serializableSnapshot
			if err := unmarshalCBOROrJSON(slices.Clone(existing), &prev,
				"bbolt.snapshot_existing",
				"unmarshal existing snapshot"); err != nil {
				return err
			}

			if prev.Version > snap.Version.Int() {
				return nil
			}
		}

		data, err := marshalCBOR(serializableSnapshot{
			StreamType: string(snap.StreamType),
			StreamID:   snap.StreamID,
			Version:    snap.Version.Int(),
			State:      snap.State,
			CreatedAt:  snap.CreatedAt.UnixNano(),
		})
		if err != nil {
			return errorfamily.WrapCorruption(err, "bbolt.serialize_snapshot",
				"serialize snapshot for "+string(snap.StreamType))
		}

		return wrapBucketErr(bucket.Put(key, data),
			"bbolt.write_snapshot", "write snapshot")
	})
	if err != nil {
		cqrsotel.RecordError(span, err)
	}

	return err
}

func (s *SnapshotStore) Load(
	ctx context.Context,
	ref id.StreamRef,
) (*snapshot.Snapshot, error) {
	_, span := startStreamSpan(ctx, "bbolt.snapshot.load", ref)
	defer span.End()

	return s.loadSnapshot(span, ref, 0, false)
}

func (s *SnapshotStore) LoadAtVersion(
	ctx context.Context,
	ref id.StreamRef,
	version event.Version,
) (*snapshot.Snapshot, error) {
	_, span := startStreamSpan(ctx, "bbolt.snapshot.load_at_version", ref,
		cqrsotel.AttrInt(cqrsotel.AttrStreamVersion, version.Int()))
	defer span.End()

	return s.loadSnapshot(span, ref, version, true)
}

func (s *SnapshotStore) loadSnapshot(
	_ cqrsotel.Span,
	ref id.StreamRef,
	maxVersion event.Version,
	enforceMax bool,
) (*snapshot.Snapshot, error) {
	key := snapshotKey(ref.Type, ref.ID)

	var result *snapshot.Snapshot

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketSnapshots))
		if bucket == nil {
			return snapshot.ErrSnapshotNotFound
		}

		val := bucket.Get(key)
		if val == nil {
			return snapshot.ErrSnapshotNotFound
		}

		var ss serializableSnapshot
		if err := unmarshalCBOROrJSON(slices.Clone(val), &ss,
			"bbolt.deserialize_snapshot", "unmarshal snapshot"); err != nil {
			return err
		}

		if enforceMax && ss.Version > maxVersion.Int() {
			return snapshot.ErrSnapshotNotFound
		}

		result = &snapshot.Snapshot{
			StreamType: id.StreamType(ss.StreamType),
			StreamID:   ss.StreamID,
			Version:    event.Version(ss.Version),
			State:      slices.Clone(ss.State),
			CreatedAt:  time.Unix(0, ss.CreatedAt).UTC(),
		}

		return nil
	})

	return result, wrapBucketErr(err, "bbolt.snapshot_load", "load snapshot")
}

func (s *SnapshotStore) Delete(ctx context.Context, ref id.StreamRef) error {
	_, span := startStreamSpan(ctx, "bbolt.snapshot.delete", ref)
	defer span.End()

	key := snapshotKey(ref.Type, ref.ID)

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketSnapshots))
		if bucket == nil {
			return nil
		}

		return wrapBucketErr(bucket.Delete(key),
			"bbolt.delete_snapshot", "delete snapshot")
	})
	if err != nil {
		cqrsotel.RecordError(span, err)
	}

	return err
}

func (s *SnapshotStore) Close() error { return nil }

func snapshotKey(streamType id.StreamType, streamID id.StreamID) []byte {
	return fmt.Appendf(nil, "%s:%s", streamType, streamID)
}

type serializableSnapshot struct {
	StreamType string      `json:"stream_type"`
	StreamID   id.StreamID `json:"stream_id"`
	Version    int         `json:"version"`
	State      []byte      `json:"state"`
	CreatedAt  int64       `json:"created_at"`
}
