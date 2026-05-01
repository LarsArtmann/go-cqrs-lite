package projection

import (
	"context"
	"fmt"
	"slices"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type Runner struct {
	loader      event.GlobalLoader
	bus         event.Bus
	checkpoint  event.CheckpointStore
	opts        runnerOptions
	projections []event.Projection
}

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

	return &Runner{
		loader:     loader,
		bus:        bus,
		checkpoint: checkpoint,
		opts:       o,
	}, nil
}

func (r *Runner) Register(p event.Projection) error {
	if p == nil {
		return ErrNilHandler
	}

	r.projections = append(r.projections, p)

	return nil
}

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

		_ = r.handleAndCheckpoint(ctx, p, evt)
	}
}

func (r *Runner) CurrentCheckpoint(ctx context.Context, projectionName string) (id.EventID, error) {
	return r.checkpoint.Load(ctx, projectionName)
}

func (r *Runner) Close() error { return nil }

func subscribesTo(p event.Projection, evtType event.Type) bool {
	types := p.EventTypes()

	if len(types) == 0 {
		return true
	}

	return slices.Contains(types, evtType)
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
