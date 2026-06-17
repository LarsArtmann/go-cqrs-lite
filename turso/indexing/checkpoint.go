package indexing

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// CheckpointScheduler runs PRAGMA checkpoint on a periodic interval.
// Useful for sync databases where you want to bound WAL growth and
// reduce sync duration.
type CheckpointScheduler struct {
	db       *sql.DB
	interval time.Duration
	mu       sync.Mutex
	stop     chan struct{}
	done     chan struct{}
	stopped  bool
}

// NewCheckpointScheduler creates a scheduler. interval of 0 disables it.
func NewCheckpointScheduler(db *sql.DB, interval time.Duration) *CheckpointScheduler {
	return &CheckpointScheduler{db: db, interval: interval}
}

// Start launches the background goroutine. Returns immediately.
// Safe to call multiple times; only the first call starts the loop.
func (s *CheckpointScheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stop != nil {
		return // already started
	}

	if s.interval <= 0 {
		return // disabled
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	s.stop = stop
	s.done = done

	go s.run(ctx, stop, done)
}

func (s *CheckpointScheduler) run(ctx context.Context, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.runOnce(ctx)
		}
	}
}

func (s *CheckpointScheduler) runOnce(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return event.WrapInfrastructure(err, "indexing.checkpoint",
			fmt.Sprintf("PRAGMA wal_checkpoint (interval=%s)", s.interval))
	}

	return nil
}

// Stop halts the background scheduler. Safe to call multiple times.
func (s *CheckpointScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	s.stopped = true

	if s.stop != nil {
		close(s.stop)
		s.stop = nil
	}

	done := s.done
	s.done = nil
	s.mu.Unlock()

	if done != nil {
		<-done
	}
}
