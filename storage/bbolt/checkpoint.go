package bbolt

import (
	"context"
	"log/slog"
	"slices"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// CheckpointStore implements event.CheckpointStore backed by bbolt.
// One checkpoint per projection is stored in the checkpoints bucket.
type CheckpointStore struct {
	storeBase
}

func NewCheckpointStore(database *bolt.DB, logger *slog.Logger) (*CheckpointStore, error) {
	if database == nil {
		return nil, ErrNilDatabase
	}

	return &CheckpointStore{storeBase: storeBase{db: database, logger: logger}}, nil
}

func (s *CheckpointStore) Save(
	_ context.Context,
	projectionName string,
	checkpoint event.Checkpoint,
) error {
	if projectionName == "" {
		return errorfamily.NewRejection("bbolt.empty_projection_name",
			"projection name must not be empty")
	}

	data, err := marshalCBOR(serializableCheckpoint{
		EventID:     checkpoint.EventID,
		ProcessedAt: checkpoint.ProcessedAt.UnixNano(),
	})
	if err != nil {
		return errorfamily.WrapCorruption(err, "bbolt.serialize_checkpoint",
			"serialize checkpoint for projection "+projectionName)
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketCheckpoints))
		if bucket == nil {
			return errorfamily.NewInfrastructure("bbolt.bucket_missing",
				"checkpoints bucket not found")
		}

		return wrapBucketErr(bucket.Put([]byte(projectionName), data),
			"bbolt.write_checkpoint", "write checkpoint for "+projectionName)
	})
}

func (s *CheckpointStore) Load(
	_ context.Context,
	projectionName string,
) (event.Checkpoint, error) {
	if projectionName == "" {
		return event.Checkpoint{EventID: id.EventID{}},
			errorfamily.NewRejection("bbolt.empty_projection_name",
				"projection name must not be empty")
	}

	var result event.Checkpoint

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketCheckpoints))
		if bucket == nil {
			result = event.Checkpoint{EventID: id.EventID{}}
			return nil
		}

		val := bucket.Get([]byte(projectionName))
		if val == nil {
			result = event.Checkpoint{EventID: id.EventID{}}
			return nil
		}

		var sc serializableCheckpoint
		if err := unmarshalCBOROrJSON(slices.Clone(val), &sc,
			"bbolt.deserialize_checkpoint",
			"deserialize checkpoint for "+projectionName); err != nil {
			return err
		}

		result = event.Checkpoint{
			EventID:     sc.EventID,
			ProcessedAt: time.Unix(0, sc.ProcessedAt).UTC(),
		}

		return nil
	})

	return result, wrapBucketErr(err, "bbolt.checkpoint_load", "load checkpoint")

func (s *CheckpointStore) Close() error { return nil }

type serializableCheckpoint struct {
	EventID     id.EventID `json:"event_id"`
	ProcessedAt int64      `json:"processed_at"`
}

func wrapBucketErr(err error, code, msg string) error {
	if err == nil {
		return nil
	}

	return errorfamily.WrapInfrastructure(err, code, msg)
}
