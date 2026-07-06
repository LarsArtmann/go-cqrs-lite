package projectionhost

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/dedup/v3"
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
		return nil, errorfamily.NewRejection(
			"projectionhost.journal_required",
			"projectionhost: journal must not be nil",
		)
	}

	if cpStore == nil {
		return nil, errorfamily.NewRejection(
			"projectionhost.checkpoint_store_required",
			"projectionhost: checkpoint store must not be nil",
		)
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
		return errorfamily.NewRejection(
			"projectionhost.register_after_start",
			"projectionhost: cannot register after Start",
		)
	}

	name := p.Name()
	if name == "" {
		return errorfamily.NewRejection(
			"projectionhost.empty_projection_name",
			"projectionhost: projection name must not be empty",
		)
	}

	if _, exists := h.workers[name]; exists {
		return errorfamily.NewRejection("projectionhost.duplicate_name",
			fmt.Sprintf("projection %q already registered", name))
	}

	h.workers[name] = &worker{
		name:       name,
		projection: p,
		journal:    h.journal,
		cpStore:    h.cpStore,
		opts:       h.opts,
		logger:     h.opts.logger,
		seenIDs:    dedup.NewRing(dedup.DefaultCapacity),
		typeSet:    buildTypeSet(p.EventTypes()),
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

		return errorfamily.NewRejection(
			"projectionhost.already_started",
			"projectionhost: already started",
		)
	}

	h.started = true
	runCtx, cancel := context.WithCancel(ctx)
	h.cancel = cancel

	workers := make([]*worker, 0, len(h.workers))
	for _, w := range h.workers {
		workers = append(workers, w)
	}
	h.mu.Unlock()

	for i, w := range workers {
		h.wg.Add(1)

		go func(worker *worker, delay int) {
			if delay > 0 {
				select {
				case <-time.After(time.Duration(delay) * time.Millisecond):
				case <-runCtx.Done():
					return
				}
			}

			worker.run(runCtx, &h.wg)
		}(w, i*10) // 10ms stagger between workers to avoid thundering herd on journal
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
		if s := w.snapshot().Status; s == WorkerRunning || s == WorkerLive || s == WorkerBackoff {
			w.setStatus(WorkerDraining)
		}

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
	case <-time.After(h.opts.shutdownTimeout):
		return errorfamily.NewInfrastructure(
			"projectionhost.shutdown_timeout",
			fmt.Sprintf(
				"projectionhost: graceful shutdown timed out after %s",
				h.opts.shutdownTimeout,
			),
		)
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

// LastProcessedAt returns the wall-clock time of the most recently processed
// event across all workers, or the zero time if no worker has processed any
// event yet. Useful for projection-lag gauges: compare against the last event
// capture time to detect if projections are falling behind.
func (h *Host) LastProcessedAt() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()

	var latest time.Time

	for _, w := range h.workers {
		ts := w.lastProcessedAt()
		if ts.After(latest) {
			latest = ts
		}
	}

	return latest
}

// LagDuration returns how long since the most recently processed event across
// all workers. This is a projection-lag indicator: if the value grows over
// time, projections are falling behind. Returns 0 if no event has been
// processed yet. Consumers register this as a Prometheus gauge:
//
//	gauge.Set(float64(host.LagDuration().Milliseconds()))
func (h *Host) LagDuration() time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()

	var latest time.Time

	for _, w := range h.workers {
		ts := w.lastProcessedAt()
		if ts.After(latest) {
			latest = ts
		}
	}

	if latest.IsZero() {
		return 0
	}

	return time.Since(latest)
}

// Resettable is an optional interface a Projection can implement to support
// full state reset. When Host.Reset detects that the projection implements
// Resettable, it calls Reset(ctx) so the projection can clear its read-model
// state (e.g. drop all rows from a SQL table, flush a KV store).
//
// Projections that don't implement Resettable will only have their checkpoint
// dropped — the next Start replays from zero, but any stale read-model state
// remains unless the consumer clears it manually.
type Resettable interface {
	Reset(ctx context.Context) error
}

// Reset drops the checkpoint for the named projection and, if the projection
// implements [Resettable], calls its Reset method to clear read-model state.
// After Reset, the next Start replys all events from the beginning of the
// journal. Use this to rebuild a projection from scratch after fixing a
// handler bug.
//
// Reset returns an error if the projection name is not registered or the host
// is currently running (Stop first). It is safe to call Reset multiple times.
func (h *Host) Reset(ctx context.Context, name string) error {
	h.mu.Lock()

	if h.started && !h.stopped {
		h.mu.Unlock()

		return errorfamily.NewRejection(
			"projectionhost.reset_while_running",
			"projectionhost: cannot reset while host is running — Stop first",
		)
	}

	w, ok := h.workers[name]
	if !ok {
		h.mu.Unlock()

		return errorfamily.NewRejection(
			"projectionhost.unknown_projection",
			fmt.Sprintf("projection %q is not registered", name),
		)
	}

	h.mu.Unlock()

	if r, ok := w.projection.(Resettable); ok {
		if err := r.Reset(ctx); err != nil {
			return errorfamily.WrapInfrastructure(err, "projectionhost.reset_projection",
				fmt.Sprintf("reset projection %q", name))
		}
	}

	if err := h.cpStore.Save(ctx, name, event.Checkpoint{}); err != nil {
		return errorfamily.WrapInfrastructure(err, "projectionhost.reset_checkpoint",
			fmt.Sprintf("clear checkpoint for %q", name))
	}

	w.setCheckpoint("")

	return nil
}

// buildTypeSet converts a slice of event types into an O(1) lookup map.
// Returns nil for empty/nil input, which means "handle all events".
func buildTypeSet(types []event.Type) map[event.Type]struct{} {
	if len(types) == 0 {
		return nil
	}

	m := make(map[event.Type]struct{}, len(types))

	for _, t := range types {
		m[t] = struct{}{}
	}

	return m
}

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
		return ReplayResult{}, errorfamily.NewRejection(
			"projectionhost.no_dead_letter_store",
			"projectionhost: no dead-letter store configured",
		)
	}

	entries, err := dlq.List(ctx, projectionName)
	if err != nil {
		return ReplayResult{}, errorfamily.WrapInfrastructure(
			err,
			"projectionhost.list_dead_letters",
			"list dead letters for replay",
		)
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
