package stack

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
)

// HealthChecker is implemented by resources that can report their health.
// Bundle.HealthCheck calls HealthCheck on every registered resource that
// implements this interface.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthCheck verifies that the Bundle's underlying resources are reachable.
// It checks every registered closer that implements [HealthChecker], plus the
// underlying database handle (via PingContext for SQL backends).
//
// Returns nil if all resources are healthy. If any resource is unhealthy, the
// first error is returned (subsequent resources are still checked but errors
// are joined).
//
// Use this for Kubernetes liveness/readiness probes or any health endpoint
// that needs to verify infrastructure connectivity.
func (b *Bundle) HealthCheck(ctx context.Context) error {
	var errs []error

	// Check the database handle if present.
	if b.db != nil {
		if db, ok := b.db.(*sql.DB); ok {
			if err := db.PingContext(ctx); err != nil {
				errs = append(errs, fmt.Errorf("database ping: %w", err))
			}
		}
	}

	// Check every registered resource that implements HealthChecker.
	seen := make(map[HealthChecker]struct{}, len(b.closers))

	for _, c := range b.closers {
		checker, ok := c.(HealthChecker)
		if !ok {
			continue
		}

		if _, dup := seen[checker]; dup {
			continue
		}

		seen[checker] = struct{}{}

		if err := checker.HealthCheck(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errorfamily.WrapInfrastructure(
			errs[0],
			"stack.bundle.health_check",
			fmt.Sprintf("%d resource(s) unhealthy", len(errs)),
		)
	}

	return nil
}
