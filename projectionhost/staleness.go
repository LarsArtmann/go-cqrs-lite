package projectionhost

import (
	"fmt"
	"strings"
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

// ErrWorkerFailed is returned by [Host.CheckStaleness] and
// [Host.CheckProjectionStaleness] when a worker has exhausted its restart
// budget and STOPPED consuming. It is classified as Infrastructure, not
// Transient: a failed worker never recovers on its own — restarting the host
// (or widening the restart budget) is an operator action, so retry-until-
// fresh loops would spin forever on a Transient classification.
//
// Behavior change (2026-08-30): the failed-worker branch previously returned
// [ErrProjectionStale]; match this sentinel instead.
var ErrWorkerFailed = errorfamily.NewInfrastructure(
	"projectionhost.worker_failed",
	"worker(s) exhausted their restart budget and stopped consuming",
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

	if failed := h.failedWorkers(); failed != "" {
		return errorfamily.Wrapf(
			ErrWorkerFailed, errorfamily.Infrastructure,
			"projectionhost.check_staleness",
			"worker(s) failed, read model may be incomplete: %s", failed,
		)
	}

	lag := h.LagDuration()
	if lag == 0 {
		return nil
	}

	if lag > maxStaleness {
		return errorfamily.Wrapf(
			ErrProjectionStale, errorfamily.Transient,
			"projectionhost.check_staleness",
			"lag %s exceeds %s", lag, maxStaleness,
		)
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
		return errorfamily.NewRejection(
			"projectionhost.unknown_projection",
			fmt.Sprintf("projection %q is not registered", name),
		)
	}

	if h.isWorkerFailed(name) {
		return errorfamily.Wrapf(
			ErrWorkerFailed, errorfamily.Infrastructure,
			"projectionhost.check_projection_staleness",
			"worker for %q failed, read model may be incomplete", name,
		)
	}

	if lag == 0 {
		return nil
	}

	if lag > maxStaleness {
		return errorfamily.Wrapf(
			ErrProjectionStale, errorfamily.Transient,
			"projectionhost.check_projection_staleness",
			"projection %q lag %s exceeds %s", name, lag, maxStaleness,
		)
	}

	return nil
}

// failedWorkers returns the semicolon-separated names of failed workers,
// or "" when none are. A failed worker stops consuming: its read model can
// no longer be assumed fresh even when lag reads zero.
func (h *Host) failedWorkers() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	var failed []string

	for _, w := range h.workers {
		if w.snapshot().Status == WorkerFailed {
			failed = append(failed, w.name)
		}
	}

	return strings.Join(failed, ";")
}

func (h *Host) isWorkerFailed(name string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	w, ok := h.workers[name]
	if !ok {
		return false
	}

	return w.snapshot().Status == WorkerFailed
}
