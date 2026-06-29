package projectionhost

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/projection/v3"
)

// Host manages the lifecycle of multiple projection workers. Each worker reads
// events from a shared journal, applies them to a registered Projection, and
// tracks its checkpoint independently.
//
// Workers are batch-drainers: they catch up from the last checkpoint and exit
// when no more events remain. They auto-restart on crash with backoff. This
// host is NOT a live stream consumer — for continuous tailing, pair it with
// watermill.CatchUpSubscriber (see example/projectionhost).
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
		logger:     h.opts.logger,
		state: WorkerState{
			Name:   name,
			Status: WorkerIdle,
		},
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	return nil
}

// Start launches a goroutine per registered projection and returns immediately.
// Each worker drains the journal from its last checkpoint until caught up
// (ReadFrom returns zero events), then exits. Workers auto-restart on crash
// with exponential backoff up to maxRestarts.
//
// For continuous live tailing, pair this host with watermill.CatchUpSubscriber
// or poll periodically by re-calling Start on a fresh Host. This host is a
// batch-drainer with crash-restart semantics, not a live stream consumer.
//
// Returns an error if already started.
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

// ReplayResult reports the outcome of a pure ReplayDeadLetters run.
// ReplayDeadLetters does NOT mutate the DeadLetterStore — callers decide
// whether to Purge the successfully replayed entries.
type ReplayResult struct {
	// Replayed are the entries whose handler now succeeds.
	Replayed []DeadLetterEntry
	// StillFailing are the entries whose handler still returns an error.
	StillFailing []ReplayFailure
}

// ReplayFailure pairs a dead-letter entry with the error its handler returned
// during the replay attempt.
type ReplayFailure struct {
	Entry DeadLetterEntry
	Err   error
}

// ReplayDeadLetters re-feeds dead-letter entries to the matching registered
// projection WITHOUT mutating the store. Pure retry: the caller decides whether
// to call Purge afterwards. Use this after fixing the handler bug that
// originally poisoned the events.
//
// An empty projectionName replays across all projections. The Host must have a
// DeadLetterStore configured (WithDeadLetterStore); otherwise ReplayDeadLetters
// returns an error. The Host need not be running.
//
// Entries whose projection is not registered, or whose Event field is nil, are
// skipped silently (counted in neither Replayed nor StillFailing).
func (h *Host) ReplayDeadLetters(ctx context.Context, projectionName string) (ReplayResult, error) {
	h.mu.Lock()
	dlq := h.opts.dlq
	workers := maps.Clone(h.workers)
	h.mu.Unlock()

	if dlq == nil {
		return ReplayResult{}, errors.New("projectionhost: no dead-letter store configured")
	}

	entries, err := dlq.List(ctx, projectionName)
	if err != nil {
		return ReplayResult{}, fmt.Errorf("projectionhost: list dead letters: %w", err)
	}

	result := ReplayResult{}

	for _, entry := range entries {
		w, ok := workers[entry.ProjectionName]
		if !ok {
			continue
		}

		if entry.Event == nil {
			continue
		}

		if err := w.projection.Handle(ctx, entry.Event); err != nil {
			h.opts.logger.Warn("dead-letter replay still failing",
				"projection", entry.ProjectionName,
				"event_id", entry.EventID, "error", err)
			result.StillFailing = append(result.StillFailing, ReplayFailure{Entry: entry, Err: err})

			continue
		}

		result.Replayed = append(result.Replayed, entry)
	}

	return result, nil
}
