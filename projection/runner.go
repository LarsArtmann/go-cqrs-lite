package projection

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Runner orchestrates projection replay from an event store and live subscription via an event bus.
// Each registered projection tracks its own checkpoint independently.
type Runner struct {
	loader      event.GlobalLoader
	bus         event.Bus
	checkpoint  event.CheckpointStore
	opts        runnerOptions
	logger      *slog.Logger
	projections []event.Projection
}

var _ io.Closer = (*Runner)(nil)

// NewRunner creates a projection Runner. Pass a nil loader to skip replay (live-only mode).
// Returns an error if bus or checkpoint is nil.
func NewRunner(
	loader event.GlobalLoader,
	bus event.Bus,
	checkpoint event.CheckpointStore,
	opts ...RunnerOption,
) (*Runner, error) {
	if bus == nil {
		return nil, errors.Wrap(ErrNilBus, "create runner")
	}

	if checkpoint == nil {
		return nil, errors.Wrap(ErrNilCheckpoint, "create runner")
	}

	o := runnerOptions{}

	for _, opt := range opts {
		opt(&o)
	}

	logger := o.logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Runner{
		loader:     loader,
		bus:        bus,
		checkpoint: checkpoint,
		opts:       o,
		logger:     logger,
	}, nil
}

// Register adds a projection to the runner. Must be called before Run.
// Returns ErrNilHandler if the projection is nil.
func (r *Runner) Register(p event.Projection) error {
	if p == nil {
		return ErrNilHandler
	}

	r.projections = append(r.projections, p)

	return nil
}

// Run replays historical events from the loader (if non-nil), then subscribes to live events.
// Blocks until the context is cancelled. Returns ErrNoProjections if no projections are registered.
func (r *Runner) Run(ctx context.Context) error {
	if len(r.projections) == 0 {
		return ErrNoProjections
	}

	if r.loader != nil {
		err := r.replay(ctx)
		if err != nil {
			return fmt.Errorf("replay: %w", err)
		}
	}

	return r.subscribeLive(ctx)
}

func (r *Runner) replay(ctx context.Context) error {
	allEvents, err := r.loader.LoadAll(ctx)
	if err != nil {
		return fmt.Errorf("load all events: %w", err)
	}

	if len(allEvents) == 0 {
		return nil
	}

	for _, p := range r.projections {
		checkpoint, cpErr := r.checkpoint.Load(ctx, p.Name())
		if cpErr != nil {
			return fmt.Errorf("load checkpoint for %q: %w", p.Name(), cpErr)
		}

		filtered := filterEvents(allEvents, p.EventTypes(), checkpoint)

		for _, evt := range filtered {
			hErr := r.handleAndCheckpoint(ctx, p, evt)
			if hErr != nil {
				return fmt.Errorf("replay projection %q event %s: %w", p.Name(), evt.ID(), hErr)
			}
		}
	}

	return nil
}

func (r *Runner) handleAndCheckpoint(
	ctx context.Context,
	p event.Projection,
	evt event.Event,
) error {
	err := p.Handle(ctx, evt)
	if err != nil {
		return err
	}

	return r.checkpoint.Save(ctx, p.Name(), evt.ID())
}

func (r *Runner) subscribeLive(ctx context.Context) error {
	handler := func(ctx context.Context, evt event.Event) error {
		r.dispatchToProjections(ctx, evt)

		return nil
	}

	err := r.bus.SubscribeAll(handler)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	<-ctx.Done()

	return nil
}

func (r *Runner) dispatchToProjections(ctx context.Context, evt event.Event) {
	for _, p := range r.projections {
		if !subscribesTo(p, evt.Type()) {
			continue
		}

		err := r.handleWithRetry(ctx, p, evt)
		if err != nil {
			r.logger.ErrorContext(ctx, "projection handler failed",
				"projection", p.Name(),
				"event_id", evt.ID(),
				"event_type", evt.Type(),
				"error", err,
			)
		}
	}
}

func (r *Runner) handleWithRetry(ctx context.Context, p event.Projection, evt event.Event) error {
	err := r.handleAndCheckpoint(ctx, p, evt)
	if err == nil {
		return nil
	}

	if r.opts.retryCount <= 0 || !event.IsRetryable(err) {
		return err
	}

	for attempt := 1; attempt <= r.opts.retryCount; attempt++ {
		delay := r.opts.retryDelay * time.Duration(1<<(attempt-1))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		err = r.handleAndCheckpoint(ctx, p, evt)
		if err == nil {
			return nil
		}
	}

	return err
}

// CurrentCheckpoint returns the last processed event ID for the given projection.
func (r *Runner) CurrentCheckpoint(ctx context.Context, projectionName string) (id.EventID, error) {
	return r.checkpoint.Load(ctx, projectionName)
}

// Close releases resources held by the runner. Currently a no-op.
func (r *Runner) Close() error { return nil }

func subscribesTo(p event.Projection, evtType event.Type) bool {
	return event.SubscribesTo(p, evtType)
}

func filterEvents(
	all []event.Event,
	types []event.Type,
	checkpoint id.EventID,
) []event.Event {
	result := make([]event.Event, 0, len(all))

	pastCheckpoint := checkpoint.IsZero()

	for _, evt := range all {
		if !pastCheckpoint {
			if evt.ID() == checkpoint {
				pastCheckpoint = true
			}

			continue
		}

		if len(types) > 0 && !slices.Contains(types, evt.Type()) {
			continue
		}

		result = append(result, evt)
	}

	return result
}
