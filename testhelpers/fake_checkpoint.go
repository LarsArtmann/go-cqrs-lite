package testhelpers

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// FakeCheckpointStore implements event.CheckpointStore for testing.
// All operations are no-ops.
type FakeCheckpointStore struct{}

// Load returns a zero EventID (no-op).
func (FakeCheckpointStore) Load(_ context.Context, _ string) (id.EventID, error) {
	return id.EventID{}, nil
}

// Save does nothing (no-op).
func (FakeCheckpointStore) Save(_ context.Context, _ string, _ id.EventID) error {
	return nil
}
