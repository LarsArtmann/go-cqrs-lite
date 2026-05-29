// Package aggregate provides backward-compatible type aliases for the decider package.
//
// Deprecated: Use github.com/larsartmann/go-cqrs-lite/core/decider instead.
// This package exists solely as a migration shim and will be removed in v2.0.0.
//
// Migration guide:
//
//	aggregate.Repository[State]  → decider.Repository[State]
//	aggregate.Decider[State]     → decider.Decider[State]
//	aggregate.NewRepository      → decider.NewRepository
//	aggregate.DecideFunc[State]  → decider.DecideFunc[State]
package aggregate

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Decider is a type alias for decider.Decider.
//
// Deprecated: Use decider.Decider[State] directly.
type Decider[State any] = decider.Decider[State]

// Repository is a type alias for decider.Repository.
//
// Deprecated: Use decider.Repository[State] directly.
type Repository[State any] = decider.Repository[State]

// DecideFunc is a type alias for decider.DecideFunc.
//
// Deprecated: Use decider.DecideFunc[State] directly.
type DecideFunc[State any] = decider.DecideFunc[State]

// NewRepository wraps decider.NewRepository.
//
// Deprecated: Use decider.NewRepository directly.
func NewRepository[State any](
	store event.Store,
	publisher event.Publisher,
	d Decider[State],
	opts ...decider.RepositoryOption[State],
) (*Repository[State], error) {
	return decider.NewRepository(store, publisher, d, opts...)
}

// Execute delegates to decider.Repository.Execute.
//
// Deprecated: Use decider.Repository.Execute directly.
func Execute[State any](
	repo *Repository[State],
	ctx context.Context,
	aggID id.AggregateID,
	aggType event.AggregateType,
	decide DecideFunc[State],
) error {
	return repo.Execute(ctx, aggID, aggType, decide)
}
