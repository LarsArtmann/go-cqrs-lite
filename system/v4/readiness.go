package system

import "context"

// ReadyCheck reports whether the system is ready to serve traffic. It runs
// every [System.HealthCheck] and additionally fails outside the Start-to-Close
// window: projection workers and managed timers only run in that window, so
// before Start a read model can be stale and a timer never fires.
//
// It returns nil when the system is ready, otherwise the first problem:
//   - [ErrSystemStopped] once the system was closed;
//   - [ErrSystemNotStarted] while a system that owns a projection host or
//     managed timers has not been started;
//   - [ErrTimersStopped] when managed schedulers were started but are no
//     longer running (their Start context was cancelled);
//   - any [System.HealthCheck] failure.
//
// A system that owns no background work is ready as soon as it is constructed.
func (s *System) ReadyCheck(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stopped {
		return ErrSystemStopped
	}

	if !s.started && (s.projHost != nil || len(s.timers) > 0) {
		return ErrSystemNotStarted
	}

	if len(s.timers) > 0 && s.timersState() == "stopped" {
		return ErrTimersStopped
	}

	return s.healthCheckLocked(ctx)
}
