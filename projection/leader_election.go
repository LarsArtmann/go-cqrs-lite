package projection

import "context"

// LeaderElection is the contract for distributed leader election, enabling
// multi-instance projection coordination per ADR-0018. Only the leader instance
// runs the projection Runner; followers stand by and take over on leader failure.
//
// The library does NOT provide an implementation — consumers wire this to
// their coordination infrastructure (Kubernetes leases, Redis, etcd, Raft).
// The interface is intentionally minimal so any consensus system can adapt to it.
//
// Usage:
//
//	type RedisLeaderElection struct { ... }
//	func (r *RedisLeaderElection) IsLeader(ctx context.Context) bool { ... }
//	func (r *RedisLeaderElection) WaitForLeadership(ctx context.Context) error { ... }
//	func (r *RedisLeaderElection) Resign(ctx context.Context) error { ... }
//
//	le := &RedisLeaderElection{...}
//	runner.RunWithLeaderElection(ctx, projection, le)
type LeaderElection interface {
	// IsLeader returns true if this instance currently holds leadership.
	// Must be safe for concurrent access — the Runner calls this frequently.
	IsLeader(ctx context.Context) bool

	// WaitForLeadership blocks until this instance becomes leader or ctx is
	// cancelled. Called once at Runner startup before replay begins.
	WaitForLeadership(ctx context.Context) error

	// Resign releases leadership voluntarily. Called during graceful shutdown.
	// No-op if this instance was not leader.
	Resign(ctx context.Context) error
}

// AlwaysLeader is a LeaderElection implementation that always returns true.
// Use for single-instance deployments where distributed coordination is not needed.
// This is the default — existing Runner behavior is equivalent to using AlwaysLeader.
type AlwaysLeader struct{}

// IsLeader always returns true.
func (AlwaysLeader) IsLeader(context.Context) bool { return true }

// WaitForLeadership returns immediately.
func (AlwaysLeader) WaitForLeadership(context.Context) error { return nil }

// Resign is a no-op.
func (AlwaysLeader) Resign(context.Context) error { return nil }
