package projectionhost_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// capturingMetricsRecorder is a test MetricsRecorder that counts every call.
type capturingMetricsRecorder struct {
	mu            sync.Mutex
	processed     atomic.Int64
	errored       atomic.Int64
	deadLettered  atomic.Int64
	restarted     atomic.Int64
	failed        atomic.Int64
	checkpointAdv atomic.Int64
}

func (c *capturingMetricsRecorder) EventProcessed(_, _ string, _ time.Duration) {
	c.processed.Add(1)
}

func (c *capturingMetricsRecorder) EventErrored(_, _ string) {
	c.errored.Add(1)
}

func (c *capturingMetricsRecorder) EventDeadLettered(_, _ string) {
	c.mu.Lock()
	c.deadLettered.Add(1)
	c.mu.Unlock()
}

func (c *capturingMetricsRecorder) WorkerRestarted(_ string) {
	c.restarted.Add(1)
}

func (c *capturingMetricsRecorder) WorkerFailed(_ string) {
	c.failed.Add(1)
}

func (c *capturingMetricsRecorder) CheckpointAdvanced(_ string, _ time.Duration) {
	c.checkpointAdv.Add(1)
}

func TestHost_Metrics_RecordsProcessedEvents(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	streamID := id.NewStreamID()
	for range 3 {
		evt, _ := event.New("item.added", streamID, "Item", event.Version(1), []byte("p"))
		journal.append(evt)
	}

	recorder := &capturingMetricsRecorder{}

	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithBatchSize(10),
		projectionhost.WithMetrics(recorder),
	)
	_ = host.Register(&countingProjection{name: "metrics-test"})

	ctx, cancel := context.WithCancel(context.Background())
	go host.Start(ctx)

	requireEventually(t, 3*time.Second, func() bool {
		return recorder.processed.Load() == 3
	})
	cancel()
	_ = host.Stop()

	if recorder.processed.Load() != 3 {
		t.Fatalf("expected 3 EventProcessed calls, got %d", recorder.processed.Load())
	}
}
