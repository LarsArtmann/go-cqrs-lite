package projection

import (
	"context"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/projection/internal/stream"
)

// Runner processes events through registered projections using reactive streams.
// Supports replay from store and live subscription via samber/ro.
type Runner struct {
	store      event.Store
	bus        event.Bus
	checkpoint event.CheckpointStore
	registry   *HandlerRegistry
	opts       runnerOptions
}

// NewRunner creates a projection runner.
// store is used for replay, bus for live events, checkpoint tracks position.
// Returns an error if any dependency is nil.
func NewRunner(
	store event.Store,
	bus event.Bus,
	checkpoint event.CheckpointStore,
	opts ...RunnerOption,
) (*Runner, error) {
	if store == nil {
		return nil, errors.Wrap(ErrNilStore, "create runner")
	}

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

	return &Runner{
		store:      store,
		bus:        bus,
		checkpoint: checkpoint,
		registry:   NewHandlerRegistry(),
		opts:       o,
	}, nil
}

// On registers a handler for a specific event type.
// Must be called before Run().
func (r *Runner) On(eventType string, handler event.Handler) error {
	return r.registry.On(eventType, handler)
}

// OnAll registers a handler for all event types.
func (r *Runner) OnAll(handler event.Handler) error {
	return r.registry.OnAll(handler)
}

// Run starts the projection: replays from store, then subscribes to live events.
// Blocks until context is cancelled.
func (r *Runner) Run(ctx context.Context) error {
	if !r.registry.HasHandlers() {
		return fmt.Errorf("projection: no handlers registered")
	}

	err := r.replay(ctx)
	if err != nil {
		return fmt.Errorf("replay: %w", err)
	}

	return r.subscribeLive(ctx)
}

func (r *Runner) replay(ctx context.Context) error {
	types := r.registry.EventTypes()

	if len(types) == 0 && len(r.registry.wildcard) == 0 {
		return nil
	}

	return stream.ProcessAll(ctx, nil, types, r.handleAndCheckpoint)
}

func (r *Runner) handleAndCheckpoint(ctx context.Context, evt event.Event) error {
	err := r.registry.dispatch(ctx, evt)
	if err != nil {
		return err
	}

	return r.checkpoint.Save(ctx, "default", evt.ID())
}

func (r *Runner) subscribeLive(ctx context.Context) error {
	handler := func(ctx context.Context, evt event.Event) error {
		return r.handleAndCheckpoint(ctx, evt)
	}

	err := r.bus.SubscribeAll(handler)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	<-ctx.Done()

	return nil
}

// CurrentCheckpoint returns the last processed event ID.
func (r *Runner) CurrentCheckpoint(ctx context.Context) (id.EventID, error) {
	return r.checkpoint.Load(ctx, "default")
}
