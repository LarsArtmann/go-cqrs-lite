package system

import "github.com/larsartmann/go-cqrs-lite/snapshot/v4"

// RegisterDeciderOption tunes decider registration.
type RegisterDeciderOption func(*registerDeciderConfig)

type registerDeciderConfig struct {
	snapshotStrategy snapshot.SnapshotStrategy
}

// WithSnapshotStrategy sets the snapshot strategy for the decider. When the
// engine implements SnapshotBackend, this enables automatic snapshot creation.
// Without a strategy, the snapshot store is wired for reads but snapshots are
// never written automatically.
func WithSnapshotStrategy(s snapshot.SnapshotStrategy) RegisterDeciderOption {
	return func(c *registerDeciderConfig) { c.snapshotStrategy = s }
}
