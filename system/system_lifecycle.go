package system

import (
	"context"
	"errors"
	"fmt"
)

// Construction and runtime lifecycle exits: the fail-path teardown and the
// projection-host start.

// fail tears down everything New created so far and returns err with any
// teardown failures joined after it. It is the single exit for New's error
// paths once the first engine exists: returning the bare error there would
// drop the System on the floor and leak engine file handles, locks, and
// background goroutines. Ordering mirrors [System.Close] (projection host,
// then engines, then registered closers); teardown errors never mask the
// construction error.
func (sys *System) fail(err error) error {
	var errs []error
	if err != nil {
		errs = append(errs, err)
	}

	if sys.projHost != nil {
		if stopErr := sys.projHost.Stop(); stopErr != nil {
			errs = append(errs, fmt.Errorf("system: fail-cleanup projection host: %w", stopErr))
		}
	}

	for _, eng := range sys.orderedEngines() {
		if closeErr := eng.Close(); closeErr != nil {
			errs = append(errs, fmt.Errorf("system: fail-cleanup engine: %w", closeErr))
		}
	}

	for _, nc := range sys.closers {
		if closeErr := nc.closer.Close(); closeErr != nil {
			errs = append(errs, fmt.Errorf("system: fail-cleanup %s: %w", nc.name, closeErr))
		}
	}

	return errors.Join(errs...)
}

// Start begins projection processing (if configured).
func (s *System) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return ErrAlreadyStarted
	}

	s.started = true

	if s.projHost != nil {
		if err := s.projHost.Start(ctx); err != nil {
			return fmt.Errorf("system: start projection host: %w", err)
		}
	}

	s.startTimersLocked(ctx)

	return nil
}
