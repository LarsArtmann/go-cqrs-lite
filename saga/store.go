package saga

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Store persists saga state.
type Store interface {
	// Save creates or updates a saga state.
	Save(ctx context.Context, state *State) error

	// Load retrieves a saga state by ID.
	Load(ctx context.Context, id id.AggregateID) (*State, error)

	// LoadAllRunning returns all saga states that are currently running or compensating.
	LoadAllRunning(ctx context.Context) ([]*State, error)
}
