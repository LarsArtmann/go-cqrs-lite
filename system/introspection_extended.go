package system

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// EngineHealth describes the health status of a single engine.
type EngineHealth struct {
	Name  string
	Error error // nil if healthy
}

// EngineNames returns the names of all engines in the system, in creation
// order. Useful for diagnostics and logging.
func (s *System) EngineNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, len(s.engines))
	for i, ne := range s.engines {
		names[i] = ne.name
	}

	return names
}

// ShutdownOrder returns the resolved close order as engine names. This is
// the same order used by [System.Close]. Useful for debugging shutdown hangs
// and verifying shutdown dependency edges.
func (s *System) ShutdownOrder() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ordered := s.orderedEngines()
	names := make([]string, len(ordered))
	for i, eng := range ordered {
		names[i] = eng.Profile().Name
	}

	return names
}

// HealthCheckDetailed returns the health status of every engine that
// implements [metaengine.HealthChecker], plus projection host worker status.
// Unlike [System.HealthCheck] (which returns the first error only), this
// method reports the status of ALL engines, making it suitable for detailed
// dashboards and debugging.
func (s *System) HealthCheckDetailed(ctx context.Context) []EngineHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []EngineHealth

	for _, ne := range s.engines {
		if hc, ok := ne.engine.(metaengine.HealthChecker); ok {
			err := hc.HealthCheck(ctx)
			result = append(result, EngineHealth{
				Name:  ne.name,
				Error: err,
			})
		}
	}

	if s.projHost != nil {
		for _, w := range s.projHost.Status() {
			if w.Status == projectionhost.WorkerFailed {
				result = append(result, EngineHealth{
					Name:  "projection:" + w.Name,
					Error: fmt.Errorf("projection %q: %s", w.Name, w.LastError),
				})
			}
		}
	}

	return result
}

// LagPerProjection returns per-projection lag keyed by projection name.
// Returns nil if no projection host is configured.
func (s *System) LagPerProjection() map[string]time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.projHost == nil {
		return nil
	}

	return s.projHost.LagPerProjection()
}

// LagDuration returns the maximum lag across all projection workers.
// Returns 0 if no projection host is configured or no events have been
// processed.
func (s *System) LagDuration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.projHost == nil {
		return 0
	}

	return s.projHost.LagDuration()
}

// WorkerStatus returns the status of all projection workers.
// Returns nil if no projection host is configured.
func (s *System) WorkerStatus() []projectionhost.WorkerState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.projHost == nil {
		return nil
	}

	return s.projHost.Status()
}
