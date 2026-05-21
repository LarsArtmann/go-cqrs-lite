package projection

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Runner orchestrates projection replay from an event store and live subscription via an event bus.
// Each registered projection tracks its own checkpoint independently.
type Runner struct {
	loader      event.GlobalLoader
	subscriber  event.Subscriber
	checkpoint  event.CheckpointStore
	opts        runnerOptions
	logger      *slog.Logger
	projections []event.Projection
}

var _ io.Closer = (*Runner)(nil)

// NewRunner creates a projection Runner. Pass a nil loader to skip replay (live-only mode).
// Returns an error if subscriber or checkpoint is nil.
func NewRunner(
	loader event.GlobalLoader,
	subscriber event.Subscriber,
	checkpoint event.CheckpointStore,
	opts ...RunnerOption,
) (*Runner, error) {
	if subscriber == nil {
		return nil, fmt.Errorf("create runner: %w", ErrNilBus)
	}

	if checkpoint == nil {
		return nil, fmt.Errorf("create runner: %w", ErrNilCheckpoint)
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
		subscriber: subscriber,
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

	for _, existing := range r.projections {
		if existing.Name() == p.Name() {
			return fmt.Errorf("%w: %q", ErrDuplicateProjection, p.Name())
		}
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
	positional, hasPositional := r.loader.(event.PositionalLoader)

	for _, p := range r.projections {
		checkpoint, cpErr := r.checkpoint.Load(ctx, p.Name())
		if cpErr != nil {
			return fmt.Errorf("load checkpoint for %q: %w", p.Name(), cpErr)
		}

		var events []event.Event

		if hasPositional && !checkpoint.IsZero() {
			loaded, lErr := positional.LoadAllFromPosition(ctx, checkpoint, 0)
			if lErr != nil {
				return fmt.Errorf("load events from position for %q: %w", p.Name(), lErr)
			}

			events = filterByTypes(loaded, p.EventTypes())
		} else {
			allEvents, lErr := r.loader.LoadAll(ctx)
			if lErr != nil {
				return fmt.Errorf("load all events: %w", lErr)
			}

			events = filterEvents(allEvents, p.EventTypes(), checkpoint)
		}

		for _, evt := range events {
			replayCtx := event.WithReplay(ctx, true)
			hErr := r.handleAndCheckpoint(replayCtx, p, evt)
			if hErr != nil {
				return fmt.Errorf("replay projection %q event %s: %w", p.Name(), evt.ID(), hErr)
			}
		}
	}

	return nil
}

func filterByTypes(events []event.Event, types []event.Type) []event.Event {
	if len(types) == 0 {
		return events
	}

	result := make([]event.Event, 0, len(events))

	for _, evt := range events {
		if slices.Contains(types, evt.Type()) {
			result = append(result, evt)
		}
	}

	return result
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
