package indexing_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v3/indexing"
)

func TestCheckpointScheduler_StartStop(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	sched := indexing.NewCheckpointScheduler(database, 100*time.Millisecond)

	sched.Start(context.Background())
	time.Sleep(150 * time.Millisecond) // allow at least one checkpoint tick
	sched.Stop()

	// No panic = success. Verify idempotency.
	sched.Stop()
}

func TestCheckpointScheduler_Disabled(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	sched := indexing.NewCheckpointScheduler(database, 0)

	// Should be a no-op since interval is 0.
	sched.Start(context.Background())
	sched.Stop()
}

func TestCheckpointScheduler_MultipleStarts(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	sched := indexing.NewCheckpointScheduler(database, 100*time.Millisecond)

	// Multiple Start calls should be idempotent.
	sched.Start(context.Background())
	sched.Start(context.Background())
	sched.Start(context.Background())

	time.Sleep(150 * time.Millisecond)
	sched.Stop()
}
