package projection

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// HealthCheck verifies that the runner's downstream dependencies are reachable.
// It pings the checkpoint store and the journal (if configured).
// Returns nil if all dependencies are healthy.
func (r *Runner) HealthCheck(ctx context.Context) error {
	_, err := r.checkpoint.Load(ctx, "__health__")
	if err != nil {
		return event.WrapInfrastructure(err, "projection.health_check",
			"checkpoint store unreachable")
	}

	if seekable, ok := r.journal.(event.SeekableJournal); ok {
		_, err = seekable.ReadFrom(ctx, id.EventID{}, 1)
		if err != nil {
			return event.WrapInfrastructure(err, "projection.health_check",
				"journal unreachable")
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
	return r.running.Load()
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
		projHealth := ProjectionHealth{
			Name: p.Name(),
		}

		checkpoint, err := r.checkpoint.Load(ctx, p.Name())
		if err != nil {
			projHealth.Healthy = false
			projHealth.Error = err.Error()
			status.Healthy = false
		} else {
			projHealth.Healthy = true
			projHealth.Checkpoint = checkpoint.EventID.String()
		}

		status.Projections = append(status.Projections, projHealth)
	}

	return status
}

// HealthChecker provides a standardized health check interface.
// projection.Runner implements this interface.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthCheckAll runs health checks on multiple HealthChecker instances
// and returns the first error encountered.
func HealthCheckAll(ctx context.Context, checkers ...HealthChecker) error {
	for _, c := range checkers {
		err := c.HealthCheck(ctx)
		if err != nil {
			return event.WrapInfrastructure(err, "projection.health_check_all",
				"health check failed")
		}
	}

	return nil
}
