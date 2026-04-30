package event

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// CheckpointStore tracks the last processed event position for projections.
// Each projection maintains its own checkpoint, enabling independent
// recovery and replay.
type CheckpointStore interface {
	// Load returns the last processed event ID for a projection.
	// Returns id.EventID zero value if no checkpoint exists.
	Load(ctx context.Context, projectionName string) (id.EventID, error)

	// Save persists the checkpoint for a projection.
	// The eventID should be the ID of the last successfully processed event.
	Save(ctx context.Context, projectionName string, eventID id.EventID) error
}
