package projection

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// HealthCheck verifies that the runner's downstream dependencies are reachable.
// It pings the checkpoint store and the journal (if configured).
// Returns nil if all dependencies are healthy.
func (r *Runner) HealthCheck(ctx context.Context) error {
	if _, err := r.checkpoint.Load(ctx, "__health__"); err != nil {
		return fmt.Errorf("projection health: checkpoint store: %w", err)
	}

	if r.journal != nil {
		if _, err := r.journal.ReadAll(ctx); err != nil {
			return fmt.Errorf("projection health: journal: %w", err)
		}
	}

	return nil
}

// RegisteredProjections returns the names of all registered projections.
// Useful for health check reporting.
func (r *Runner) RegisteredProjections() []string {
	names := make([]string, len(r.projections))
	for i, p := range r.projections {
		names[i] = p.Name()
	}

	return names
}

// IsRunning returns true if the runner has an active subscription (Run was called).
func (r *Runner) IsRunning() bool {
	if r.projections == nil {
		return false
	}

	for _, p := range r.projections {
		cp, err := r.checkpoint.Load(context.Background(), p.Name())
		if err != nil {
			return false
		}

		if !cp.IsZero() {
			return true
		}
	}

	return false
}

// HealthStatus contains the health check result for a runner.
type HealthStatus struct {
	Healthy     bool
	Projections []ProjectionHealth
}

// ProjectionHealth contains health information for a single projection.
type ProjectionHealth struct {
	Name       string
	Checkpoint string
	Healthy    bool
	Error      string
}

// DetailedHealthCheck performs a health check for each registered projection
// and returns individual results.
func (r *Runner) DetailedHealthCheck(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Healthy:     true,
		Projections: make([]ProjectionHealth, 0, len(r.projections)),
	}

	for _, p := range r.projections {
		ph := ProjectionHealth{
			Name: p.Name(),
		}

		cp, err := r.checkpoint.Load(ctx, p.Name())
		if err != nil {
			ph.Healthy = false
			ph.Error = err.Error()
			status.Healthy = false
		} else {
			ph.Healthy = true
			ph.Checkpoint = cp.String()
		}

		status.Projections = append(status.Projections, ph)
	}

	return status
}

// HealthChecker provides a standardized health check interface.
// Both projection.Runner and saga.Runner implement this interface.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthCheckAll runs health checks on multiple HealthChecker instances
// and returns the first error encountered.
func HealthCheckAll(ctx context.Context, checkers ...HealthChecker) error {
	for _, c := range checkers {
		if err := c.HealthCheck(ctx); err != nil {
			return event.WrapInfrastructure(err, "projection.health_check_all",
				"health check failed")
		}
	}

	return nil
}
