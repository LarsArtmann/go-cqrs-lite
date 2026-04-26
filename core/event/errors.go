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
