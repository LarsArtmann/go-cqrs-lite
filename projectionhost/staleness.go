package projectionhost

import (
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrProjectionStale is returned by [Host.CheckStaleness] and
// [Host.CheckProjectionStaleness] when the projection lag exceeds the
// configured threshold. It is classified as Transient: a stale projection
// will catch up once the worker drains its backlog.
var ErrProjectionStale = errorfamily.NewTransient(
	"projectionhost.stale",
	"projection lag exceeds staleness threshold",
)

// CheckStaleness returns nil if the maximum projection lag across all workers
// is within maxStaleness, or [ErrProjectionStale] if it exceeds the threshold.
//
// Use this as a read-time guard before serving data from a read model:
//
//	if err := host.CheckStaleness(5 * time.Second); err != nil {
//	    // Projection is too stale — return 503 or stale data with a warning.
//	}
//
// A maxStaleness of 0 or less disables the check (always returns nil).
// If no events have been processed yet (lag == 0), the projection is
// considered fresh — it has not had a chance to fall behind.
func (h *Host) CheckStaleness(maxStaleness time.Duration) error {
	if maxStaleness <= 0 {
		return nil
	}

	lag := h.LagDuration()
	if lag == 0 {
		return nil
	}

	if lag > maxStaleness {
		return fmt.Errorf("%w: lag %s exceeds %s", ErrProjectionStale, lag, maxStaleness)
	}

	return nil
}

// CheckProjectionStaleness returns nil if the named projection's lag is within
// maxStaleness, or [ErrProjectionStale] if it exceeds the threshold.
//
// Returns an error if the projection name is not registered.
//
//	if err := host.CheckProjectionStaleness("users", 5*time.Second); err != nil {
//	    // The "users" projection is too stale.
//	}
func (h *Host) CheckProjectionStaleness(name string, maxStaleness time.Duration) error {
	if maxStaleness <= 0 {
		return nil
	}

	lags := h.LagPerProjection()

	lag, ok := lags[name]
	if !ok {
		return fmt.Errorf(
			"%w: unknown projection %q",
			ErrProjectionNotRegistered, name,
		)
	}

	if lag == 0 {
		return nil
	}

	if lag > maxStaleness {
		return fmt.Errorf(
			"%w: projection %q lag %s exceeds %s",
			ErrProjectionStale, name, lag, maxStaleness,
		)
	}

	return nil
}
