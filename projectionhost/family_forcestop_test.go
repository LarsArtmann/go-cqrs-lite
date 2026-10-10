package projectionhost_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// A Rejection-classified handler error is non-retryable: the worker must
// attempt the event exactly once and DLQ it, instead of burning the full
// dlqThreshold retry budget on an error that can never succeed.
func TestHost_NonRetryableHandlerError_SkipsRetries(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("task.created"))

	const threshold = 3

	proj := &failingProjection{
		name: "rejecting",
		err:  errorfamily.NewRejection("test/rejected", "malformed payload"),
	}

	dlq := projectionhost.NewMemoryDeadLetterStore()
	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithBatchSize(10),
		projectionhost.WithDeadLetterStore(dlq, threshold),
		projectionhost.WithMaxRestarts(-1),
	)
	host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = host.Start(ctx) }()

	requireEventually(t, 2*time.Second, func() bool {
		entries, _ := dlq.List(context.Background(), "")

		return len(entries) == 1
	})

	if got := proj.handles.Load(); got != 1 {
		t.Fatalf("Rejection error handled %d times, want exactly 1 (no retries)", got)
	}

	_ = host.ForceStop(context.Background())
}

// ForceStop returns even when a graceful Stop would time out on stuck
// workers, and re-arms the stopped latch so shutdown can be retried.
func TestHost_ForceStop_BoundsWorkerExit(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("blocked.created"))

	proj := &blockingProjection{name: "blocked", release: make(chan struct{})}
	host, _ := projectionhost.New(
		journal,
		cpStore,
		projectionhost.WithShutdownTimeout(50*time.Millisecond),
	)
	host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = host.Start(ctx) }()

	requireEventually(t, 2*time.Second, func() bool {
		return proj.started.Load()
	})

	if err := host.Stop(); err == nil {
		t.Fatal("expected graceful Stop to time out while the handler blocks")
	}

	cancel()
	close(proj.release)

	forceCtx, forceCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer forceCancel()

	if err := host.ForceStop(forceCtx); err != nil {
		t.Fatalf("ForceStop after releasing the handler: %v", err)
	}
}

type failingProjection struct {
	name    string
	err     error
	handles atomic.Int64
}

func (p *failingProjection) Name() string             { return p.name }
func (p *failingProjection) EventTypes() []event.Type { return nil }

func (p *failingProjection) Handle(_ context.Context, _ event.Event) error {
	p.handles.Add(1)

	return p.err
}

// blockingProjection blocks inside Handle until release is closed.
type blockingProjection struct {
	name    string
	release chan struct{}
	started atomic.Bool
}

func (p *blockingProjection) Name() string             { return p.name }
func (p *blockingProjection) EventTypes() []event.Type { return nil }

func (p *blockingProjection) Handle(_ context.Context, _ event.Event) error {
	p.started.Store(true)

	<-p.release

	return errors.New("unblocked after force stop")
}
