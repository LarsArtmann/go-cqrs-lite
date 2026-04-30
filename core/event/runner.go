package event

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// ProjectionRunner drives projections by feeding events and tracking checkpoints.
type ProjectionRunner interface {
	// Register adds a projection to the runner.
	Register(projection Projection) error

	// Handle processes an event through all registered projections.
	// This can be used as a Bus.SubscribeAll handler.
	Handle(ctx context.Context, evt Event) error
}

// InMemoryRunner processes events through registered projections with
// checkpoint tracking. Intended for testing and single-process deployments.
//
// Usage:
//
//	runner := event.NewInMemoryRunner(checkpointStore)
//	runner.Register(myProjection)
//	bus.SubscribeAll(runner.Handle)
type InMemoryRunner struct {
	checkpoint CheckpointStore

	mu          sync.RWMutex
	projections []Projection
}

// NewInMemoryRunner creates a runner that tracks checkpoints.
func NewInMemoryRunner(checkpoint CheckpointStore) *InMemoryRunner {
	return &InMemoryRunner{
		checkpoint:  checkpoint,
		mu:          sync.RWMutex{},
		projections: nil,
	}
}

// Register adds a projection to the runner.
func (r *InMemoryRunner) Register(projection Projection) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.projections = append(r.projections, projection)

	return nil
}

// Handle dispatches an event to all registered projections that
// subscribe to its type. After successful processing, the checkpoint
// is updated.
func (r *InMemoryRunner) Handle(ctx context.Context, evt Event) error {
	r.mu.RLock()
	projections := make([]Projection, len(r.projections))
	copy(projections, r.projections)
	r.mu.RUnlock()

	evtType := evt.Type()

	for _, proj := range projections {
		if !subscribesTo(proj, evtType) {
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

// subscribesTo returns true if the projection subscribes to the given event type.
// Returns true if the projection subscribes to all events (nil EventTypes).
func subscribesTo(proj Projection, evtType Type) bool {
	types := proj.EventTypes()
	if len(types) == 0 {
		return true
	}

	return slices.Contains(types, evtType)
}

var _ Handler = (*InMemoryRunner)(nil).Handle
