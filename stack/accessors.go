package stack

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/readmodel/v2"
)

// Repository constructs a typed [decider.Repository] over the Bundle's event
// store. The publisher is optional — when the Bundle has no publisher, the
// repository operates in pure event-sourcing mode (persist without publish).
//
// It is a top-level generic function, not a method on [Bundle], because Go
// does not permit generic methods on non-generic types. Call it as:
//
//	repo, err := stack.Repository[State](bundle, decider)
//
// The Bundle must have been configured with [WithEventStore] (or a preset that
// sets one). Returns [ErrMissingEventStore] otherwise.
//
// opts are forwarded to [decider.NewRepository]; use them to enable
// snapshots (decider.WithSnapshotStore) or disable load coalescing.
func Repository[State any](
	b *Bundle,
	d decider.Decider[State],
	opts ...decider.RepositoryOption[State],
) (*decider.Repository[State], error) {
	store, ok := b.eventStore()
	if !ok {
		return nil, ErrMissingEventStore
	}

	return decider.NewRepository[State](store, b.Publisher, d, opts...)
}

// ReadModel constructs a typed [readmodel.Store] over the Bundle's read-model
// backend.
//
// It is a top-level generic function (see [Repository] for why). Call it as:
//
//	store, err := stack.ReadModel[Todo, TodoID](bundle, codec.JSONCodec{})
//
// The Bundle must have been configured with [WithReadModels]. The deployer
// chose the backend (memory, Pebble, SQL); the application wraps it with a
// typed Store parameterised by its value type T and key type K (which must
// implement [fmt.Stringer], as branded IDs do).
//
// opts configure the Store (key prefix, custom key encoding) and are forwarded
// to [readmodel.New]. If c is nil, [codec.JSONCodec] is used.
func ReadModel[T any, K fmt.Stringer](
	b *Bundle,
	c codec.Codec,
	opts ...readmodel.Option[T, K],
) (*readmodel.Store[T, K], error) {
	if b.ReadModels == nil {
		return nil, ErrMissingReadModels
	}

	if c == nil {
		c = codec.JSONCodec{}
	}

	allOpts := append([]readmodel.Option[T, K]{readmodel.WithCodec[T, K](c)}, opts...)

	return readmodel.New[T, K](b.ReadModels, allOpts...), nil
}

// ProjectionRunner constructs a [projection.Runner] over the Bundle's journal,
// subscriber, and checkpoint store. It is a method (not a generic function)
// because it has no type parameters.
//
// All three prerequisites are required: the journal replays history, the
// subscriber handles live events, and the checkpoint store tracks how far the
// runner has caught up. Returns the relevant ErrMissing* error if any is
// absent.
func (b *Bundle) ProjectionRunner(
	opts ...projection.RunnerOption,
) (*projection.Runner, error) {
	if b.Journal == nil {
		return nil, ErrMissingJournal
	}

	if b.Subscriber == nil {
		return nil, ErrMissingSubscriber
	}

	if b.CheckpointStore == nil {
		return nil, ErrMissingCheckpoint
	}

	runner, err := projection.NewRunner(b.Journal, b.Subscriber, b.CheckpointStore, opts...)
	if err != nil {
		return nil, fmt.Errorf("stack: create projection runner: %w", err)
	}

	return runner, nil
}

// eventStore recovers the composite event.Store from the Bundle's segregated
// EventSink field. When the Bundle was configured via [WithEventStore] (the
// common case), EventSink holds a value that implements the full Store
// interface (EventSink + EventSource). If it was configured with
// [WithEventSink] alone using a sink that does not also implement
// EventSource, no composite store is available and this returns false.
func (b *Bundle) eventStore() (event.Store, bool) {
	if b.EventSink == nil {
		return nil, false
	}

	store, ok := b.EventSink.(event.Store)

	return store, ok
}
