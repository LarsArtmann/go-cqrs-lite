package saga

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Store persists saga instances.
type Store interface {
	// Save creates or updates a saga instance.
	Save(ctx context.Context, instance *Instance) error

	// Load retrieves a saga instance by ID.
	Load(ctx context.Context, id id.AggregateID) (*Instance, error)

	// LoadAllRunning returns all saga instances that are currently running or compensating.
	LoadAllRunning(ctx context.Context) ([]*Instance, error)
}
