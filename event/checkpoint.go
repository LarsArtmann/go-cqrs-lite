package event

import (
	"context"
	"io"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// Checkpoint records the last processed event position for a projection.
type Checkpoint struct {
	EventID     id.EventID
	ProcessedAt time.Time
}

// IsZero reports whether this checkpoint represents no prior progress.
func (c Checkpoint) IsZero() bool {
	return c.EventID.IsZero()
}

// String returns the checkpoint's event ID as a string.
func (c Checkpoint) String() string {
	return c.EventID.String()
}

// CheckpointSink is the write side of checkpoint persistence.
type CheckpointSink interface {
	io.Closer

	// Save persists the checkpoint for a projection.
	// ProcessedAt should record when the event was successfully handled.
	Save(ctx context.Context, projectionName string, cp Checkpoint) error
}

// CheckpointSource is the read side of checkpoint persistence.
type CheckpointSource interface {
	io.Closer

	// Load returns the last checkpoint for a projection.
	// Returns a zero-value Checkpoint if no checkpoint exists.
	Load(ctx context.Context, projectionName string) (Checkpoint, error)
}

// CheckpointStore is the composite of CheckpointSink + CheckpointSource.
// All existing implementations satisfy CheckpointStore.
type CheckpointStore interface {
	CheckpointSink
	CheckpointSource
}
