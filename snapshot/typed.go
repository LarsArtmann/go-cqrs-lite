package snapshot

import (
	"context"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
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
	AggregateType id.AggregateType
	Version       event.Version
	State         State
	CreatedAt     time.Time
}

// TypedSnapshotSink saves typed snapshots. The typed analogue of
// [SnapshotSink], without the io.Closer embedding (closing the underlying
// store is the adapter's job).
type TypedSnapshotSink[State any] interface {
	Save(ctx context.Context, snapshot TypedSnapshot[State]) error
	Delete(ctx context.Context, ref id.AggregateRef) error
}

// TypedSnapshotSource loads typed snapshots. The typed analogue of
// [SnapshotSource].
type TypedSnapshotSource[State any] interface {
	Load(ctx context.Context, ref id.AggregateRef) (*TypedSnapshot[State], error)
	LoadAtVersion(
		ctx context.Context,
		ref id.AggregateRef,
		version event.Version,
	) (*TypedSnapshot[State], error)
}

// TypedStore is the typed analogue of [SnapshotStore]. It adapts an untyped
// [SnapshotStore] plus a [codec.Codec] into a typed interface over State.
//
// Construct one with [NewTypedStore]:
//
//	ts := snapshot.NewTypedStore[MyState](store, codec.CBORCodec{})
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
// serialization. If c is nil, [codec.CBORCodec] is used.
// Pre-envelope data (raw JSON) is auto-detected on read.
func NewTypedStore[State any](store SnapshotStore, c codec.Codec) *TypedStore[State] {
	if c == nil {
		c = codec.CBORCodec{}
	}

	return &TypedStore[State]{store: store, codec: c}
}

// Save encodes snapshot.State and delegates to the underlying [SnapshotStore].
func (t *TypedStore[State]) Save(ctx context.Context, snapshot TypedSnapshot[State]) error {
	encoded, err := codec.WrapEncode(snapshot.State, t.codec)
	if err != nil {
		return errorfamily.Wrapf(err, errorfamily.Corruption, "snapshot.encode_state",
			"encode state for %s v%d", snapshot.AggregateID, snapshot.Version)
	}

	err = t.store.Save(ctx, Snapshot{
		AggregateID:   snapshot.AggregateID,
		AggregateType: snapshot.AggregateType,
		Version:       snapshot.Version,
		State:         encoded,
		CreatedAt:     snapshot.CreatedAt,
	})
	if err != nil {
		return errorfamily.Wrapf(err, errorfamily.Infrastructure, "snapshot.save",
			"save %s v%d", snapshot.AggregateID, snapshot.Version)
	}

	return nil
}

// Delete removes the snapshot for ref from the underlying store.
func (t *TypedStore[State]) Delete(ctx context.Context, ref id.AggregateRef) error {
	err := t.store.Delete(ctx, ref)
	if err != nil {
		return errorfamily.Wrapf(err, errorfamily.Infrastructure, "snapshot.delete",
			"delete %s", ref)
	}

	return nil
}

// Load retrieves the snapshot for ref and decodes its State.
func (t *TypedStore[State]) Load(
	ctx context.Context,
	ref id.AggregateRef,
) (*TypedSnapshot[State], error) {
	raw, err := t.store.Load(ctx, ref)
	if err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Infrastructure, "snapshot.load",
			"load %s", ref)
	}

	return t.decode(raw)
}

// LoadAtVersion retrieves the snapshot for ref at the given version and
// decodes its State.
func (t *TypedStore[State]) LoadAtVersion(
	ctx context.Context,
	ref id.AggregateRef,
	version event.Version,
) (*TypedSnapshot[State], error) {
	raw, err := t.store.LoadAtVersion(ctx, ref, version)
	if err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Infrastructure, "snapshot.load_version",
			"load %s v%d", ref, version)
	}

	return t.decode(raw)
}

// Store returns the underlying untyped [SnapshotStore].
func (t *TypedStore[State]) Store() SnapshotStore { return t.store }

func (t *TypedStore[State]) decode(raw *Snapshot) (*TypedSnapshot[State], error) {
	var state State

	c, inner := codec.UnwrapDecode(raw.State, codec.JSONCodec{})

	err := c.Decode(inner, &state)
	if err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Corruption, "snapshot.decode_state",
			"decode state for %s v%d", raw.AggregateID, raw.Version)
	}

	return &TypedSnapshot[State]{
		AggregateID:   raw.AggregateID,
		AggregateType: raw.AggregateType,
		Version:       raw.Version,
		State:         state,
		CreatedAt:     raw.CreatedAt,
	}, nil
}
