package snapshot_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

type counterState struct {
	Count int    `json:"count"`
	Label string `json:"label"`
}

func TestTypedStore_SaveLoad_Roundtrip(t *testing.T) {
	t.Parallel()

	store := snapshot.NewTypedStore[counterState](newFakeStore(), codec.JSONCodec{})

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Counter", streamID)

	input := snapshot.TypedSnapshot[counterState]{
		StreamID:   streamID,
		StreamType: "Counter",
		Version:    3,
		State:      counterState{Count: 42, Label: "answer"},
	}

	ctx := context.Background()

	if err := store.Save(ctx, input); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.State.Count != 42 || got.State.Label != "answer" {
		t.Fatalf("roundtrip mismatch: got %+v", got.State)
	}

	if got.Version != 3 {
		t.Fatalf("version = %d, want 3", got.Version)
	}
}

func TestTypedStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := snapshot.NewTypedStore[counterState](newFakeStore(), codec.JSONCodec{})

	ref := id.NewStreamRef("Counter", id.NewStreamID())

	_, err := store.Load(context.Background(), ref)
	if !errors.Is(err, snapshot.ErrSnapshotNotFound) {
		t.Fatalf("got err=%v, want ErrSnapshotNotFound", err)
	}
}

func TestTypedStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	store := snapshot.NewTypedStore[counterState](newFakeStore(), codec.JSONCodec{})

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Counter", streamID)

	ctx := context.Background()

	if err := store.Save(ctx, snapshot.TypedSnapshot[counterState]{
		StreamID:   streamID,
		StreamType: "Counter",
		Version:    5,
		State:      counterState{Count: 7},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.LoadAtVersion(ctx, ref, 5)
	if err != nil {
		t.Fatalf("LoadAtVersion: %v", err)
	}

	if got.State.Count != 7 {
		t.Fatalf("count = %d, want 7", got.State.Count)
	}
}

func TestTypedStore_Delete(t *testing.T) {
	t.Parallel()

	store := snapshot.NewTypedStore[counterState](newFakeStore(), codec.JSONCodec{})

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Counter", streamID)

	ctx := context.Background()

	if err := store.Save(ctx, snapshot.TypedSnapshot[counterState]{
		StreamID:   streamID,
		StreamType: "Counter",
		Version:    1,
		State:      counterState{Count: 1},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := store.Load(ctx, ref); !errors.Is(err, snapshot.ErrSnapshotNotFound) {
		t.Fatalf("after Delete, Load err=%v, want ErrSnapshotNotFound", err)
	}
}

func TestTypedStore_NilCodecDefaultsToCBOR(t *testing.T) {
	t.Parallel()

	store := snapshot.NewTypedStore[counterState](newFakeStore(), nil)

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Counter", streamID)

	ctx := context.Background()

	if err := store.Save(ctx, snapshot.TypedSnapshot[counterState]{
		StreamID:   streamID,
		StreamType: "Counter",
		Version:    1,
		State:      counterState{Count: 99},
	}); err != nil {
		t.Fatalf("Save with default CBOR codec: %v", err)
	}

	got, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.State.Count != 99 {
		t.Fatalf("count = %d, want 99", got.State.Count)
	}
}

func TestTypedStore_LegacyStateDecodesAcrossCodecs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	seed := func(t *testing.T, state []byte) (*fakeStore, id.StreamRef) {
		t.Helper()

		fake := newFakeStore()
		streamID := id.NewStreamID()
		ref := id.NewStreamRef("Counter", streamID)
		if err := fake.Save(ctx, snapshot.Snapshot{
			StreamID:   streamID,
			StreamType: "Counter",
			Version:    1,
			State:      state,
		}); err != nil {
			t.Fatalf("seed Save: %v", err)
		}

		return fake, ref
	}

	rawJSON := []byte(`{"count":11,"label":"legacy-json"}`)

	rawCBOR, err := codec.CBORCodec{}.Encode(counterState{Count: 22, Label: "legacy-cbor"})
	if err != nil {
		t.Fatalf("encode raw CBOR: %v", err)
	}

	// Raw JSON state read by the CBOR-default store.
	cborFake, cborRef := seed(t, rawJSON)
	cborDefault := snapshot.NewTypedStore[counterState](cborFake, nil)
	loaded, err := cborDefault.Load(ctx, cborRef)
	if err != nil {
		t.Fatalf("Load legacy raw JSON: %v", err)
	}
	if loaded.State.Count != 11 || loaded.State.Label != "legacy-json" {
		t.Fatalf("legacy raw JSON under CBOR default: got %+v", loaded.State)
	}

	// Raw CBOR state read by a JSON-configured store.
	jsonFake, jsonRef := seed(t, rawCBOR)
	jsonConfigured := snapshot.NewTypedStore[counterState](jsonFake, codec.JSONCodec{})
	loaded, err = jsonConfigured.Load(ctx, jsonRef)
	if err != nil {
		t.Fatalf("Load legacy raw CBOR: %v", err)
	}
	if loaded.State.Count != 22 || loaded.State.Label != "legacy-cbor" {
		t.Fatalf("legacy raw CBOR under JSON config: got %+v", loaded.State)
	}
}
