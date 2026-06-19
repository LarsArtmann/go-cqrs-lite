package snapshot

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// TypedSnapshot is a snapshot with a typed State field, closing the
// type-safety hole where [Snapshot].State is an untyped []byte.
//
// Every consumer that loads a snapshot today must manually decode State via
// event.DecodePayload or codec.Decode, with no compile-time guarantee that
// the bytes match the expected type. TypedSnapshot makes that decode the
// adapter's responsibility: a [TypedStore] decodes once at the boundary, and
// every downstream consumer holds a State of the right type.
//
// TypedSnapshot mirrors [Snapshot] field-for-field except State, which is
// generic. Convert between them with [TypedStore.TypedToBytes] and
// [TypedStore.BytesToTyped] (or just let the adapter handle it).
type TypedSnapshot[State any] struct {
	AggregateID   id.AggregateID
	AggregateType event.AggregateType
	Version       event.Version
	State         State
	CreatedAt     time.Time
}

// TypedSnapshotSink saves typed snapshots. The typed analogue of
// [SnapshotSink], without the io.Closer embedding (closing the underlying
// store is the adapter's job).
type TypedSnapshotSink[State any] interface {
	Save(ctx context.Context, snapshot TypedSnapshot[State]) error
	Delete(ctx context.Context, ref event.AggregateRef) error
}

// TypedSnapshotSource loads typed snapshots. The typed analogue of
// [SnapshotSource].
type TypedSnapshotSource[State any] interface {
	Load(ctx context.Context, ref event.AggregateRef) (*TypedSnapshot[State], error)
	LoadAtVersion(
		ctx context.Context,
		ref event.AggregateRef,
		version event.Version,
	) (*TypedSnapshot[State], error)
}

// TypedStore is the typed analogue of [SnapshotStore]. It adapts an untyped
// [SnapshotStore] plus a [codec.Codec] into a typed interface over State.
//
// Construct one with [NewTypedStore]:
//
//	ts := snapshot.NewTypedStore[MyState](store, codec.JSONCodec{})
//	_ = ts.Save(ctx, snapshot.TypedSnapshot[MyState]{State: state, ...})
//	got, _ := ts.Load(ctx, ref)
//	// got.State is MyState, not []byte
//
// TypedStore does not own the underlying store's lifecycle; closing the
// untyped SnapshotStore (or the Bundle that wraps it) is the caller's job.
type TypedStore[State any] struct {
	store SnapshotStore
	codec codec.Codec
}

// NewTypedStore creates a typed adapter over store using c for State
// serialization. If c is nil, [codec.JSONCodec] is used.
func NewTypedStore[State any](store SnapshotStore, c codec.Codec) *TypedStore[State] {
	if c == nil {
		c = codec.JSONCodec{}
	}

	return &TypedStore[State]{store: store, codec: c}
}

// Save encodes snapshot.State and delegates to the underlying [SnapshotStore].
func (t *TypedStore[State]) Save(ctx context.Context, snapshot TypedSnapshot[State]) error {
	encoded, err := t.codec.Encode(snapshot.State)
	if err != nil {
		return fmt.Errorf(
			"snapshot: encode state for %s v%d: %w",
			snapshot.AggregateID,
			snapshot.Version,
			err,
		)
	}

	err = t.store.Save(ctx, Snapshot{
		AggregateID:   snapshot.AggregateID,
		AggregateType: snapshot.AggregateType,
		Version:       snapshot.Version,
		State:         encoded,
		CreatedAt:     snapshot.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("snapshot: save %s v%d: %w", snapshot.AggregateID, snapshot.Version, err)
	}

	return nil
}

// Delete removes the snapshot for ref from the underlying store.
func (t *TypedStore[State]) Delete(ctx context.Context, ref event.AggregateRef) error {
	err := t.store.Delete(ctx, ref)
	if err != nil {
		return fmt.Errorf("snapshot: delete %s: %w", ref, err)
	}

	return nil
}

// Load retrieves the snapshot for ref and decodes its State.
func (t *TypedStore[State]) Load(
	ctx context.Context,
	ref event.AggregateRef,
) (*TypedSnapshot[State], error) {
	raw, err := t.store.Load(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("snapshot: load %s: %w", ref, err)
	}

	return t.decode(raw)
}

// LoadAtVersion retrieves the snapshot for ref at the given version and
// decodes its State.
func (t *TypedStore[State]) LoadAtVersion(
	ctx context.Context,
	ref event.AggregateRef,
	version event.Version,
) (*TypedSnapshot[State], error) {
	raw, err := t.store.LoadAtVersion(ctx, ref, version)
	if err != nil {
		return nil, fmt.Errorf("snapshot: load %s v%d: %w", ref, version, err)
	}

	return t.decode(raw)
}

// Store returns the underlying untyped [SnapshotStore].
func (t *TypedStore[State]) Store() SnapshotStore { return t.store }

func (t *TypedStore[State]) decode(raw *Snapshot) (*TypedSnapshot[State], error) {
	var state State

	err := t.codec.Decode(raw.State, &state)
	if err != nil {
		return nil, fmt.Errorf(
			"snapshot: decode state for %s v%d: %w",
			raw.AggregateID,
			raw.Version,
			err,
		)
	}

	return &TypedSnapshot[State]{
		AggregateID:   raw.AggregateID,
		AggregateType: raw.AggregateType,
		Version:       raw.Version,
		State:         state,
		CreatedAt:     raw.CreatedAt,
	}, nil
}
