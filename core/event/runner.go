package event

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// InMemoryRunner processes events through registered projections with
// checkpoint tracking. Intended for testing and single-process deployments.
//
// Usage:
//
//	runner, err := event.NewInMemoryRunner(checkpointStore)
//	if err != nil {
//	    // handle error
//	}
//	runner.Register(myProjection)
//	bus.SubscribeAll(runner.Handle)
type InMemoryRunner struct {
	checkpoint CheckpointStore

	mu          sync.RWMutex
	projections []Projection
}

// NewInMemoryRunner creates a runner that tracks checkpoints.
// Returns an error if checkpoint is nil.
func NewInMemoryRunner(checkpoint CheckpointStore) (*InMemoryRunner, error) {
	if checkpoint == nil {
		return nil, fmt.Errorf("%w", ErrNilCheckpointStore)
	}

	return &InMemoryRunner{
		checkpoint:  checkpoint,
		mu:          sync.RWMutex{},
		projections: nil,
	}, nil
}

// Register adds a projection to the runner.
// Returns an error if the projection is nil or a projection with
// the same name is already registered.
func (r *InMemoryRunner) Register(projection Projection) error {
	if projection == nil {
		return fmt.Errorf("%w", ErrNilProjection)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.projections {
		if existing.Name() == projection.Name() {
			return fmt.Errorf("%w: %q", ErrDuplicateProjection, projection.Name())
		}
	}

	r.projections = append(r.projections, projection)

	return nil
}

// Handle dispatches an event to all registered projections that
// subscribe to its type. After successful processing, the checkpoint
// is updated.
//
// Fail-fast: if any projection returns an error, Handle returns immediately.
// Subsequent projections for this event are not processed, and their
// checkpoints are not saved. The caller is responsible for retry logic.
func (r *InMemoryRunner) Handle(ctx context.Context, evt Event) error {
	r.mu.RLock()
	projections := make([]Projection, len(r.projections))
	copy(projections, r.projections)
	r.mu.RUnlock()

	evtType := evt.Type()

	for _, proj := range projections {
		if !SubscribesTo(proj, evtType) {
			continue
		}

		err := proj.Handle(ctx, evt)
		if err != nil {
			return fmt.Errorf("projection %q handle event %s: %w", proj.Name(), evtType, err)
		}

		err = r.checkpoint.Save(ctx, proj.Name(), evt.ID())
		if err != nil {
			return fmt.Errorf("checkpoint save for projection %q: %w", proj.Name(), err)
		}
	}

	return nil
}

// HandleParallel dispatches an event to all matching projections concurrently.
// Each projection runs in its own goroutine. Returns the first error encountered.
// Checkpoints are saved only for projections that succeed.
// Respects context cancellation: if the context is canceled, remaining
// goroutines are waited for but their results are ignored.
func (r *InMemoryRunner) HandleParallel(ctx context.Context, evt Event) error {
	projections := r.matchingProjections(evt.Type())
	if len(projections) == 0 {
		return nil
	}

	results := r.dispatchProjections(ctx, projections, evt)

	return r.collectResults(ctx, results, projections, evt)
}

func (r *InMemoryRunner) matchingProjections(evtType Type) []Projection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matched := make([]Projection, 0, len(r.projections))
	for _, p := range r.projections {
		if SubscribesTo(p, evtType) {
			matched = append(matched, p)
		}
	}

	return matched
}

type parallelResult struct {
	proj Projection
	err  error
}

func (r *InMemoryRunner) dispatchProjections(
	ctx context.Context,
	projections []Projection,
	evt Event,
) chan parallelResult {
	results := make(chan parallelResult, len(projections))

	for _, proj := range projections {
		go func(p Projection) {
			var res parallelResult

			defer func() {
				if r := recover(); r != nil {
					res = parallelResult{
						proj: p,
						err:  fmt.Errorf("%w: %q: %v", ErrProjectionPanicked, p.Name(), r),
					}
				}

				results <- res
			}()

			res = parallelResult{proj: p, err: p.Handle(ctx, evt)}
		}(proj)
	}

	return results
}

func (r *InMemoryRunner) collectResults(
	ctx context.Context,
	results chan parallelResult,
	projections []Projection,
	evt Event,
) error {
	var firstErr error

	for range projections {
		var res parallelResult

		select {
		case <-ctx.Done():
			if firstErr == nil {
				firstErr = fmt.Errorf("handle parallel canceled: %w", ctx.Err())
			}

			return firstErr
		case res = <-results:
		}

		if res.err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"projection %q handle event %s: %w",
					res.proj.Name(),
					evt.Type(),
					res.err,
				)
			}

			continue
		}

		err := r.checkpoint.Save(ctx, res.proj.Name(), evt.ID())
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("checkpoint save for projection %q: %w", res.proj.Name(), err)
		}
	}

	return firstErr
}

// SubscribesTo returns true if the projection subscribes to the given event type.
// Returns true if the projection subscribes to all events (nil or empty EventTypes).
func SubscribesTo(proj Projection, evtType Type) bool {
	types := proj.EventTypes()
	if len(types) == 0 {
		return true
	}

	return slices.Contains(types, evtType)
}

var _ Handler = (*InMemoryRunner)(nil).Handle
