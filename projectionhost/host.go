package projectionhost

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/projection/v3"
)

// Host manages the lifecycle of multiple projection workers. Each worker reads
// events from a shared journal, applies them to a registered Projection, and
// tracks its checkpoint independently.
type Host struct {
	journal event.SeekableJournal
	cpStore event.CheckpointStore
	opts    hostOptions

	mu      sync.Mutex
	workers map[string]*worker
	started bool
	stopped bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New creates a Host that reads from journal and persists checkpoints to cpStore.
func New(
	journal event.SeekableJournal,
	cpStore event.CheckpointStore,
	opts ...HostOption,
) (*Host, error) {
	if journal == nil {
		return nil, errors.New("projectionhost: journal must not be nil")
	}

	if cpStore == nil {
		return nil, errors.New("projectionhost: checkpoint store must not be nil")
	}

	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	return &Host{
		journal: journal,
		cpStore: cpStore,
		opts:    o,
		workers: make(map[string]*worker),
	}, nil
}

// Register adds a projection to the host. The projection's Name() must be
// unique across all registered projections. Must be called before Start.
func (h *Host) Register(p projection.Projection) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		return errors.New("projectionhost: cannot register after Start")
	}

	name := p.Name()
	if name == "" {
		return errors.New("projectionhost: projection name must not be empty")
	}

	if _, exists := h.workers[name]; exists {
		return fmt.Errorf("projectionhost: projection %q already registered", name)
	}

	h.workers[name] = &worker{
		name:       name,
		projection: p,
		journal:    h.journal,
		cpStore:    h.cpStore,
		opts:       h.opts,
		state: WorkerState{
			Name:   name,
			Status: WorkerIdle,
		},
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	return nil
}

// Start begins processing for all registered projections. It blocks until
// Stop is called or the context is cancelled. Each worker runs in its own
// goroutine with independent crash-restart logic.
func (h *Host) Start(ctx context.Context) error {
	h.mu.Lock()
	if h.started {
		h.mu.Unlock()

		return errors.New("projectionhost: already started")
	}

	h.started = true
	runCtx, cancel := context.WithCancel(ctx)
	h.cancel = cancel

	workers := make([]*worker, 0, len(h.workers))
	for _, w := range h.workers {
		workers = append(workers, w)
	}
	h.mu.Unlock()

	for _, w := range workers {
		h.wg.Add(1)
		go w.run(runCtx, &h.wg)
	}

	return nil
}

// Stop gracefully shuts down all workers, waiting for in-flight events to
// complete. Safe to call multiple times.
func (h *Host) Stop() error {
	h.mu.Lock()
	if !h.started || h.stopped {
		h.mu.Unlock()

		return nil
	}

	h.stopped = true
	h.cancel()

	for _, w := range h.workers {
		close(w.stop)
	}
	h.mu.Unlock()

	done := make(chan struct{})

	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(30 * time.Second):
		return errors.New("projectionhost: graceful shutdown timed out after 30s")
	}
}

// Status returns a snapshot of every worker's current state.
func (h *Host) Status() []WorkerState {
	h.mu.Lock()
	defer h.mu.Unlock()

	result := make([]WorkerState, 0, len(h.workers))
	for _, w := range h.workers {
		result = append(result, w.snapshot())
	}

	return result
}

// RegisterAndWait is a convenience that registers a projection, starts the
// host, and blocks until ctx is cancelled or all workers stop. Useful for
// simple single-projection setups.
func RegisterAndWait(ctx context.Context, h *Host, projections ...projection.Projection) error {
	for _, p := range projections {
		if err := h.Register(p); err != nil {
			return err
		}
	}

	return h.Start(ctx)
}
