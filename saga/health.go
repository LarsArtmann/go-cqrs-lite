package saga

import (
	"context"
	"fmt"
)

// HealthCheck verifies that the runner's downstream dependencies are reachable.
// It pings the saga store to confirm connectivity.
// Returns nil if the store is healthy.
func (r *Runner) HealthCheck(ctx context.Context) error {
	if r.store == nil {
		return fmt.Errorf("saga health: store is nil")
	}

	if healthChecker, ok := r.store.(interface {
		HealthCheck(context.Context) error
	}); ok {
		return healthChecker.HealthCheck(ctx)
	}

	return nil
}

// RegisteredSagas returns the types of all registered saga definitions.
func (r *Runner) RegisteredSagas() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.registry))
	for t := range r.registry {
		types = append(types, t)
	}

	return types
}
