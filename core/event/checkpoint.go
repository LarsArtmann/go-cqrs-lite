package event

import (
	"context"
	"io"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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

// CheckpointStore tracks the last processed event position for projections.
// Each projection maintains its own checkpoint, enabling independent
// recovery and replay.
// All implementations must support lifecycle management via io.Closer.
type CheckpointStore interface {
	io.Closer

	// Load returns the last checkpoint for a projection.
	// Returns a zero-value Checkpoint if no checkpoint exists.
	Load(ctx context.Context, projectionName string) (Checkpoint, error)

	// Save persists the checkpoint for a projection.
	// ProcessedAt should record when the event was successfully handled.
	Save(ctx context.Context, projectionName string, cp Checkpoint) error
}
