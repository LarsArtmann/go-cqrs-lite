package projection

import (
	"context"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

const defaultLeadershipCheckInterval = 5 * time.Second

// distributedOptions configures a DistributedRunner.
type distributedOptions struct {
	checkInterval time.Duration
	logger        *slog.Logger
}

// DistributedOption configures a DistributedRunner.
type DistributedOption func(*distributedOptions)

// WithLeadershipCheckInterval sets how often the DistributedRunner verifies
// it still holds leadership during live processing. Defaults to 5 seconds.
// A shorter interval detects leadership loss faster but adds more overhead.
func WithLeadershipCheckInterval(d time.Duration) DistributedOption {
	return func(o *distributedOptions) { o.checkInterval = d }
}

// WithDistributedLogger sets the structured logger for the DistributedRunner.
// Defaults to slog.Default() if not set.
func WithDistributedLogger(logger *slog.Logger) DistributedOption {
	return func(o *distributedOptions) { o.logger = logger }
}

// DistributedRunner wraps a projection Runner with leader-election gating.
// Only the elected leader instance runs replay and live subscriptions; followers
// stand by. When leadership is lost during live processing, the runner stops
// gracefully (allowing another instance to take over).
//
// Consumers provide a LeaderElection implementation backed by their coordination
// infrastructure (Kubernetes leases, Redis, etcd, Raft). The library provides
// AlwaysLeader as a no-op default for single-instance deployments.
//
// Usage:
//
//	runner, _ := projection.NewRunner(journal, subscriber, checkpoint)
//	runner.Register(myProjection)
//	dr, _ := projection.NewDistributedRunner(runner, &RedisLeaderElection{...})
//	dr.Run(ctx) // blocks until ctx cancelled or leadership lost
type DistributedRunner struct {
	runner   *Runner
	election LeaderElection
	opts     distributedOptions
	logger   *slog.Logger
}

// NewDistributedRunner creates a DistributedRunner that wraps the given Runner
// with leader-election gating. Returns an error if runner or election is nil.
func NewDistributedRunner(
	runner *Runner,
	election LeaderElection,
	opts ...DistributedOption,
) (*DistributedRunner, error) {
	if runner == nil {
		return nil, event.WrapInfrastructure(ErrNilRunner, "projection.create_distributed_runner",
			"create distributed runner: nil runner")
	}

	if election == nil {
		return nil, event.WrapInfrastructure(
			ErrNilLeaderElection,
			"projection.create_distributed_runner",
			"create distributed runner: nil leader election",
		)
	}

	o := distributedOptions{
		checkInterval: defaultLeadershipCheckInterval,
	}

	for _, opt := range opts {
		opt(&o)
	}

	logger := o.logger
	if logger == nil {
		logger = slog.Default()
	}

	return &DistributedRunner{
		runner:   runner,
		election: election,
		opts:     o,
		logger:   logger,
	}, nil
}

// resignTimeout is the maximum time to wait when resigning leadership during shutdown.
const resignTimeout = 10 * time.Second

// Run waits for leadership, then runs the projection (replay + live).
// During live processing, it periodically checks IsLeader. If leadership is
// lost, it cancels the live subscription and returns ErrLeadershipLost.
//
// On exit (whether normal or due to leadership loss), it calls Resign to
// release leadership voluntarily.
//
// Blocks until ctx is cancelled, the runner's Close is called, or leadership
// is lost.
func (dr *DistributedRunner) Run(ctx context.Context) error {
	ctx, span := cqrsotel.StartSpan(ctx, tracer(), "projection.distributed_run",
		cqrsotel.SpanKindClient)
	defer span.End()

	err := dr.election.WaitForLeadership(ctx)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.Wrapf(err, event.Infrastructure, "projection.wait_for_leadership",
			"failed to acquire leadership")
	}

	dr.logger.InfoContext(ctx, "acquired leadership, starting projection runner")

	resignCtx, resignCancel := context.WithTimeout(ctx, resignTimeout)
	defer resignCancel()
	defer dr.resignLeadership(resignCtx)

	leaderCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	lostCh := make(chan error, 1)

	go dr.monitorLeadership(leaderCtx, cancel, lostCh)

	runErr := dr.runner.Run(leaderCtx)

	select {
	case <-lostCh:
		cqrsotel.RecordError(span, ErrLeadershipLost)

		return ErrLeadershipLost
	default:
	}

	if runErr != nil {
		cqrsotel.RecordError(span, runErr)
	}

	return runErr
}

// resignLeadership attempts to release leadership during shutdown.
func (dr *DistributedRunner) resignLeadership(ctx context.Context) {
	err := dr.election.Resign(ctx)
	if err != nil {
		dr.logger.WarnContext(ctx, "failed to resign leadership", "error", err)
	}
}

// monitorLeadership periodically checks IsLeader. If leadership is lost,
// it cancels the context and sends ErrLeadershipLost to lostCh.
func (dr *DistributedRunner) monitorLeadership(
	ctx context.Context,
	cancel context.CancelFunc,
	lostCh chan<- error,
) {
	ticker := time.NewTicker(dr.opts.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !dr.election.IsLeader(ctx) {
				dr.logger.InfoContext(ctx, "leadership lost, stopping projection runner")

				lostCh <- ErrLeadershipLost

				cancel()

				return
			}
		}
	}
}

// Runner returns the underlying Runner, allowing callers to call Register,
// Close, and other methods directly.
func (dr *DistributedRunner) Runner() *Runner {
	return dr.runner
}
