package event

import "github.com/cockroachdb/errors"

// ErrVersionConflict is returned when there is a version conflict.
var ErrVersionConflict = errors.New("version conflict")

// ErrAggregateNotFound is returned when an aggregate is not found.
var ErrAggregateNotFound = errors.New("aggregate not found")

// ErrStoreClosed is returned when the event store is closed.
var ErrStoreClosed = errors.New("event store is closed")

// ErrBusClosed is returned when the event bus is closed.
var ErrBusClosed = errors.New("event bus is closed")

// ErrSnapshotNotFound is returned when a snapshot is not found.
var ErrSnapshotNotFound = errors.New("snapshot not found")

// ErrSnapshotStoreClosed is returned when the snapshot store is closed.
var ErrSnapshotStoreClosed = errors.New("snapshot store is closed")

// ErrNilProjection is returned when a nil projection is registered.
var ErrNilProjection = errors.New("event: nil projection")

// ErrNilCheckpointStore is returned when a nil checkpoint store is passed to NewInMemoryRunner.
var ErrNilCheckpointStore = errors.New("event: nil checkpoint store")

// ErrDuplicateProjection is returned when a projection with the same name is already registered.
var ErrDuplicateProjection = errors.New("event: duplicate projection")

// ErrNilOutbox is returned when a nil outbox is passed to NewOutboxPublisher.
var ErrNilOutbox = errors.New("event: nil outbox")

// ErrNilBus is returned when a nil bus is passed to NewOutboxPublisher.
var ErrNilBus = errors.New("event: nil bus")

// ErrAlreadyStarted is returned when OutboxPublisher.Start is called more than once.
var ErrAlreadyStarted = errors.New("event: outbox publisher already started")
