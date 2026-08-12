package stack

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// TombstonePolicy controls which records appear in [Materialize.List] results.
type TombstonePolicy int

const (
	// IncludeTombstoned returns all records, including tombstoned ones.
	IncludeTombstoned TombstonePolicy = iota
	// ExcludeTombstoned filters out tombstoned records (default behavior).
	ExcludeTombstoned
	// OnlyTombstoned returns only tombstoned records.
	OnlyTombstoned
)

// Materialize turns a stream of events into a materialized view stored in a
// [kv.ViewStore]. It is the deployer's read-model builder: given event
// handlers (OnCreate, OnUpdate, OnTombstone, OnRebirth), it maintains a typed
// projection that consumers query via [Materialize.View] and [Materialize.List].
//
// The Store field accepts any [kv.ViewStore] implementation:
//
//   - [kv.TypedStore] — KV-backed (memory, Pebble, or SQLKVStore blob table).
//   - storage.SQLViewStore — SQL-backed with real, queryable columns.
//
// The Materialize API is tombstone-aware (ADR-0006, ADR-0030). There are no
// hard deletes — a tombstoned record stays in the store and is filtered by the
// query policy.
//
// Usage (KV-backed):
//
//	mat := stack.Materialize[UserView, UserID]{
//		Store:       kvStore,
//		KeyFromEvent: func(evt event.Event) (UserID, error) { ... },
//		OnCreate:    func(ctx, evt) (*UserView, error) { ... },
//		OnUpdate:    func(ctx, evt, existing *UserView) (*UserView, error) { ... },
//	}
//	router.AddHandler("users", topic, catchUpSub, "users_view", pub, mat.HandlerFunc())
//
// Usage (SQL-backed, real columns):
//
//	store, _ := storage.NewSQLiteViewStore[UserView, UserID](db, mapper)
//	mat := stack.Materialize[UserView, UserID]{Store: store, ...}
type Materialize[V any, K fmt.Stringer] struct {
	// Store is the typed view store that holds the materialized view.
	// Use [kv.TypedStore] for KV backends or storage.SQLViewStore for
	// SQL-backed views with queryable columns.
	Store kv.ViewStore[V, K]

	// KeyFromEvent extracts the read-model key from an event.
	// Required: every event type this projection handles must produce a valid key.
	KeyFromEvent func(evt event.Event) (K, error)

	// OnCreate handles an event that creates a new view record.
	// If the record already exists, it is treated as an update instead.
	OnCreate func(ctx context.Context, evt event.Event) (*V, error)

	// OnUpdate handles an event that modifies an existing view record.
	// If the record does not exist, the event is skipped.
	OnUpdate func(ctx context.Context, evt event.Event, existing *V) (*V, error)

	// OnTombstone handles a tombstone event (soft-delete).
	// The record is NOT deleted from the store — it is marked tombstoned
	// and excluded from default query results.
	OnTombstone func(ctx context.Context, evt event.Event, existing *V) (*V, error)

	// OnRebirth handles a rebirth event (undo tombstone).
	// The record is restored to active status.
	OnRebirth func(ctx context.Context, evt event.Event, existing *V) (*V, error)

	// ProjectionName optionally sets the name returned by [Name] for
	// diagnostics and runner registration. Defaults to "materialize".
	ProjectionName string
}

// Name implements [projection.Projection]. Returns ProjectionName or
// "materialize" if unset.
func (m *Materialize[V, K]) Name() string {
	if m.ProjectionName != "" {
		return m.ProjectionName
	}

	return "materialize"
}

// Handle implements [projection.Projection]. Delegates to handleEvent.
// Materialize handles ALL event types (returns nil from EventTypes) — it
// relies on KeyFromEvent + the On* callbacks to decide what to do.
func (m *Materialize[V, K]) Handle(ctx context.Context, evt event.Event) error {
	return m.handleEvent(ctx, evt)
}

// EventTypes implements [projection.Projection]. Returns nil — Materialize
// handles all event types by design (the On* callbacks filter by key existence).
func (m *Materialize[V, K]) EventTypes() []event.Type { return nil }

// HandlerFunc returns a [message.NoPublishHandlerFunc] that decodes Watermill messages
// back to cqrs events and dispatches them to the appropriate On* handler.
// Uses [watermill.MessageToEvent] for decoding — the single source of truth
// for the Watermill ↔ cqrs protocol.
func (m *Materialize[V, K]) HandlerFunc() message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		ctx := msg.Context()

		// Decode using the canonical watermill protocol.
		topic := msg.Metadata.Get("event_type")

		evt, err := cqrswatermill.MessageToEvent(topic, msg)
		if err != nil {
			return errorfamily.WrapCorruption(err, "stack.materialize.decode",
				"decode message")
		}

		if err := m.handleEvent(ctx, evt); err != nil {
			return errorfamily.Wrap(err, errorfamily.Classify(err),
				"stack.materialize.handle_event",
				"handle event "+evt.ID().String())
		}

		return nil
	}
}

func (m *Materialize[V, K]) handleEvent(ctx context.Context, evt event.Event) error {
	key, err := m.KeyFromEvent(evt)
	if err != nil {
		return errorfamily.WrapRejection(err, "stack.materialize.extract_key",
			"extract key from event")
	}

	md := evt.Metadata()

	// Check for tombstone/rebirth marks.
	if md.Tombstone != nil {
		existing, getErr := m.Store.Get(ctx, key)
		if getErr != nil && !errors.Is(getErr, kv.ErrNotFound) {
			return errorfamily.Wrap(getErr, errorfamily.Classify(getErr),
				"stack.materialize.load_tombstone", "load existing for tombstone")
		}

		if errors.Is(getErr, kv.ErrNotFound) {
			existing = nil
		}

		switch md.Tombstone.Status {
		case event.TombstoneTombstoned:
			if m.OnTombstone != nil {
				updated, err := m.OnTombstone(ctx, evt, existing)
				if err != nil {
					return err
				}

				return m.Store.Set(ctx, key, updated)
			}
		case event.TombstoneActive:
			if m.OnRebirth != nil {
				updated, err := m.OnRebirth(ctx, evt, existing)
				if err != nil {
					return err
				}

				return m.Store.Set(ctx, key, updated)
			}
		case event.TombstoneUndetermined:
			// Can't determine status — skip projection. A subsequent event
			// with a definitive status will resolve the stream state.
		}

		return nil
	}

	// Regular event: try OnUpdate first, fall back to OnCreate.
	existing, err := m.Store.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, kv.ErrNotFound) {
			return errorfamily.Wrap(err, errorfamily.Classify(err),
				"stack.materialize.load_existing", "load existing record")
		}

		// Not found → create.
		if m.OnCreate != nil {
			val, createErr := m.OnCreate(ctx, evt)
			if createErr != nil {
				return createErr
			}

			return m.Store.Set(ctx, key, val)
		}

		return nil
	}

	// Found → update.
	if m.OnUpdate != nil {
		updated, err := m.OnUpdate(ctx, evt, existing)
		if err != nil {
			return err
		}

		return m.Store.Set(ctx, key, updated)
	}

	return nil
}

// View retrieves a single record by key.
func (m *Materialize[V, K]) View(ctx context.Context, key K) (*V, error) {
	return m.Store.Get(ctx, key)
}

// List returns all records matching the given tombstone policy.
// Records are returned in lexicographic key order.
//
// When the backing store implements [kv.TombstoneQuerier] (e.g. SQL-backed
// stores with a configured tombstone column), the filter is pushed to SQL —
// only matching records are loaded. Otherwise, all records are loaded and
// filtered in Go.
func (m *Materialize[V, K]) List(ctx context.Context, policy TombstonePolicy) ([]*V, error) {
	if tq, ok := m.Store.(kv.TombstoneQuerier[V]); ok {
		results, err := tq.QueryByTombstone(
			ctx,
			policy == ExcludeTombstoned,
			policy == OnlyTombstoned,
		)
		if err != nil {
			return nil, err
		}

		// Safety net: stores without a tombstone column return all records.
		// FilterTombstoned is a no-op when the store already filtered.
		return FilterTombstoned(results, policy), nil
	}

	all, err := m.Store.Scan(ctx, nil)
	if err != nil {
		return nil, err
	}

	return FilterTombstoned(all, policy), nil
}

// FilterTombstoned filters a slice of records according to the given tombstone policy.
// Records whose value type implements `IsTombstoned() bool` are checked; all others
// are treated as active.
func FilterTombstoned[V any](all []*V, policy TombstonePolicy) []*V {
	if policy == IncludeTombstoned {
		return all
	}

	filtered := make([]*V, 0, len(all))

	for _, v := range all {
		isTombstoned := isMaterializedTombstoned(v)

		switch {
		case policy == ExcludeTombstoned && !isTombstoned:
			filtered = append(filtered, v)
		case policy == OnlyTombstoned && isTombstoned:
			filtered = append(filtered, v)
		}
	}

	return filtered
}

// isMaterializedTombstoned checks if a record has a TombstoneMark field.
// It uses reflection-free interface check — the value type V should have a
// boolean IsTombstoned() method or a TombstoneMark field for this to work.
// If V does not expose tombstone state, this returns false (always visible).
func isMaterializedTombstoned[V any](v *V) bool {
	type tombstoner interface {
		IsTombstoned() bool
	}

	if t, ok := any(v).(tombstoner); ok {
		return t.IsTombstoned()
	}

	return false
}
