package irohengine

import (
	"context"
	"sync"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// blockingMapLocal is a memory engine whose FIRST MapSet parks until released,
// standing in for a goroutine scheduled out between the LWW guard and the apply.
type blockingMapLocal struct {
	metaengine.Engine

	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (l *blockingMapLocal) MapSet(ctx context.Context, col string, key, value any) error {
	l.once.Do(func() {
		close(l.entered)
		<-l.release
	})

	return l.Engine.(metaengine.MapBackend).MapSet(ctx, col, key, value)
}

func (l *blockingMapLocal) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	return l.Engine.(metaengine.MapBackend).MapGet(ctx, col, key)
}

func (l *blockingMapLocal) MapDelete(ctx context.Context, col string, key any) error {
	return l.Engine.(metaengine.MapBackend).MapDelete(ctx, col, key)
}

var _ metaengine.MapBackend = (*blockingMapLocal)(nil)

// TestApplyRemoteLWW_OlderOpCannotOvertakeNewer pins the LWW guard as ONE
// critical section: while an older op is mid-apply, a newer op for the same
// key must not overtake it and be overwritten by the older value.
func TestApplyRemoteLWW_OlderOpCannotOvertakeNewer(t *testing.T) {
	t.Parallel()

	local := &blockingMapLocal{
		Engine:  metaengine.NewMemoryEngine(),
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	e := &replicatedEngine{
		local: local, cfg: &config{clock: realClock{}}, timestamps: make(map[string]time.Time),
	}

	base := time.Unix(1_700_000_000, 0)
	older := WriteOp{
		ID: "op-old", Collection: "tasks", Kind: OpMapSet, Key: "k", Value: "old",
		Timestamp: base,
	}
	newer := WriteOp{
		ID: "op-new", Collection: "tasks", Kind: OpMapSet, Key: "k", Value: "new",
		Timestamp: base.Add(time.Second),
	}

	done := make(chan struct{}, 2)

	go func() {
		e.applyRemote(older)
		done <- struct{}{}
	}()

	<-local.entered

	go func() {
		e.applyRemote(newer)
		done <- struct{}{}
	}()

	// Unserialized, the newer op completes while the older one is parked.
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
	}

	close(local.release)
	<-done
	<-done

	got, ok, err := local.Engine.(metaengine.MapBackend).MapGet(context.Background(), "tasks", "k")
	if err != nil || !ok {
		t.Fatalf("MapGet = (%v, %v, %v)", got, ok, err)
	}

	if got != "new" {
		t.Fatalf("value = %v, want \"new\": an older op overtook a newer one", got)
	}
}
