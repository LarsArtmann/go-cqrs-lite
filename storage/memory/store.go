package memory

import (
	"context"
	"fmt"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// MemoryStore is an in-memory implementation of event.Store and event.Journal.
// It is safe for concurrent use. Designed for testing and single-process deployments.
//
// It embeds the generic [LogStore] core; this file supplies only the
// event-specific policies (version conflicts, stream-not-found errors).
type MemoryStore struct {
	*LogStore[event.Event, id.EventID]
}

var (
	_ event.Store           = (*MemoryStore)(nil)
	_ event.Journal         = (*MemoryStore)(nil)
	_ event.SeekableJournal = (*MemoryStore)(nil)
	_ event.BackwardsSource = (*MemoryStore)(nil)
	_ event.MultiSink       = (*MemoryStore)(nil)
	_ io.Closer             = (*MemoryStore)(nil)
)

// NewMemoryStore creates a new in-memory event store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		LogStore: NewLogStore(LogStoreConfig[event.Event, id.EventID]{
			GetID:     func(evt event.Event) id.EventID { return evt.ID() },
			IsZeroID:  func(evtID id.EventID) bool { return evtID.IsZero() },
			ClosedErr: event.ErrStoreClosed,
			NewDupErr: nil,
			NewNotFound: func(op, streamKey string) error {
				return errorfamily.WrapRejection(event.ErrStreamNotFound,
					"memory.aggregate_not_found",
					fmt.Sprintf("memory %s stream %s not found", op, streamKey))
			},
			TrackStreams: true,
		}),
	}
}

// Save appends events to a stream with optimistic concurrency check.
// Returns ErrVersionConflict if the expected version does not match the current stream length.
func (s *MemoryStore) Save(
	_ context.Context,
	ref id.StreamRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	return s.WithWrite("memory.save_failed", "memory store save", func() error {
		key := ref.StreamKey()

		err := event.CheckVersionConflict(s.StreamLenLocked(key), expectedVersion)
		if err != nil {
			return errorfamily.WrapInfrastructure(err, "memory.save_failed", "memory store save")
		}

		s.AppendLocked(key, events)

		return nil
	})
}

// AppendBatch appends events without a version check. Useful for testing idempotent writes.
func (s *MemoryStore) AppendBatch(
	_ context.Context,
	ref id.StreamRef,
	events []event.Event,
) error {
	return s.WithWrite("memory.append_batch_failed", "memory store append batch", func() error {
		s.AppendLocked(ref.StreamKey(), events)

		return nil
	})
}

// SaveMultiBatch appends events for multiple streams under a single lock.
// All entries are persisted atomically — either all succeed or none.
func (s *MemoryStore) SaveMultiBatch(
	_ context.Context,
	entries []event.MultiBatchEntry,
) error {
	return s.WithWrite(
		"memory.save_multi_batch_failed",
		"memory store save multi batch",
		func() error {
			for _, entry := range entries {
				s.AppendLocked(entry.Ref.StreamKey(), entry.Events)
			}

			return nil
		},
	)
}
