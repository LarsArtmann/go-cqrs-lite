package stack

import (
	"fmt"
	"log/slog"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
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

	//cqrs-lint:ignore(A017,B025) library code or intentional pattern
	return decider.NewRepository(store, b.Publisher, d, opts...)
}

// readModelCodec validates that the Bundle has a read-model backend and
// returns the codec to use (caller-provided, or the Bundle default when nil).
// Shared by ReadModel and NewMaterialize.
func (b *Bundle) readModelCodec(c codec.Codec) (codec.Codec, error) {
	if b.ReadModels == nil {
		return nil, ErrMissingReadModels
	}

	if c == nil {
		return b.DefaultCodec(), nil
	}

	return c, nil
}

// ReadModel constructs a typed [kv.TypedStore] over the Bundle's read-model
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
// to [kv.NewTypedStore]. If c is nil, [codec.JSONCodec] is used.
func ReadModel[T any, K fmt.Stringer](
	b *Bundle,
	c codec.Codec,
	opts ...kv.TypedOption[T, K],
) (*kv.TypedStore[T, K], error) {
	c, err := b.readModelCodec(c)
	if err != nil {
		return nil, err
	}

	allOpts := append([]kv.TypedOption[T, K]{kv.WithTypedCodec[T, K](c)}, opts...)

	return kv.NewTypedStore(b.ReadModels, allOpts...), nil
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

// EventStore returns the composite event.Store from the Bundle's EventSink
// field. Consumers frequently need the raw store for query handlers, journal
// access, or SSE broker registration without keeping a separate reference.
//
// Returns the store and true when configured via [WithEventStore] (or a preset
// that sets one). Returns nil, false when no store is configured or when
// [WithEventSink] was used with a sink that doesn't also implement EventSource.
func (b *Bundle) EventStore() (event.Store, bool) {
	return b.eventStore()
}

// TypedRepository constructs a [decider.TypedRepository] over the Bundle's
// event store, binding both the State and Command type parameters at compile
// time. The Decide function lives on the TypedDecider, not passed per-call.
//
// Call it as:
//
//	repo, err := stack.TypedRepository[UserState, CreateUser](bundle, typedDecider)
//
// This is the typed-command evolution of [Repository] (ADR-0001, Phase 6).
// The legacy [Repository] remains for consumers who pass DecideFunc per-call.
func TypedRepository[State, Cmd any](
	b *Bundle,
	d decider.TypedDecider[State, Cmd],
	opts ...decider.RepositoryOption[State],
) (*decider.TypedRepository[State, Cmd], error) {
	store, ok := b.eventStore()
	if !ok {
		return nil, ErrMissingEventStore
	}

	return decider.NewTypedRepository(store, b.Publisher, d, opts...)
}

// NewMaterialize constructs a [Materialize] over the Bundle's read-model backend.
// It is the deployer-first projection builder: the consumer defines how events
// create/update/tombstone view records; the deployer chose where the KV store
// lives (SQLite, Pebble, memory).
//
// Call it as:
//
//	mat, err := stack.NewMaterialize[UserView, UserID](bundle, codec.JSONCodec{},
//	    func(evt event.Event) (UserID, error) { ... })
//
// Then wire it as a Watermill handler: mat.HandlerFunc().
func NewMaterialize[V any, K fmt.Stringer](
	b *Bundle,
	c codec.Codec,
	keyFunc func(evt event.Event) (K, error),
) (*Materialize[V, K], error) {
	c, err := b.readModelCodec(c)
	if err != nil {
		return nil, err
	}

	store := kv.NewTypedStore(b.ReadModels, kv.WithTypedCodec[V, K](c))

	return &Materialize[V, K]{
		Store:        store,
		KeyFromEvent: keyFunc,
	}, nil
}

// CatchUpSubscriber constructs a [watermill.CatchUpSubscriber] over the
// Bundle's seekable journal, live subscriber, and checkpoint store. The
// subscriber replays journal events from the last checkpoint, then
// seamlessly switches to live delivery.
//
// Call it as:
//
//	sub, err := stack.CatchUpSubscriber(bundle)
//
// All three prerequisites are required. Returns the relevant ErrMissing*
// error if any is absent. The Bundle's Subscriber must be a *watermill.EventBus
// (the default in all presets).
func (b *Bundle) CatchUpSubscriber() (*cqrswatermill.CatchUpSubscriber, error) {
	if b.SeekableJournal == nil {
		return nil, ErrMissingJournal
	}

	if b.Subscriber == nil {
		return nil, ErrMissingSubscriber
	}

	if b.CheckpointStore == nil {
		return nil, ErrMissingCheckpoint
	}

	liveSub, ok := b.Subscriber.(*cqrswatermill.EventBus)
	if !ok {
		return nil, errorfamily.NewRejection(
			"stack.invalid_subscriber_type",
			"subscriber must be *watermill.EventBus for CatchUpSubscriber",
		)
	}

	return cqrswatermill.NewCatchUpSubscriber(
		b.SeekableJournal, liveSub.MessageSubscriber(), b.CheckpointStore, slog.Default(),
	)
}

// QueryAuditMiddleware creates a [query.AuditMiddleware] from the Bundle's
// query sink. The deployer chooses the audit level; the consumer applies
// the middleware to their dispatcher:
//
//	mw, err := stack.QueryAuditMiddleware(bundle, query.AuditFull)
//	d.Use(mw)
//
// Returns [ErrMissingQueryStore] if the Bundle has no query sink.
func QueryAuditMiddleware(b *Bundle, level query.AuditLevel) (query.Middleware, error) {
	if b.QuerySink == nil {
		return nil, ErrMissingQueryStore
	}

	return query.AuditMiddleware(b.QuerySink, level, slog.Default()), nil
}
