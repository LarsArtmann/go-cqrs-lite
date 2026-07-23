package snapshot_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
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

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Counter", aggID)

	input := snapshot.TypedSnapshot[counterState]{
		StreamID:   aggID,
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

	ref := id.NewAggregateRef("Counter", id.NewAggregateID())

	_, err := store.Load(context.Background(), ref)
	if !errors.Is(err, snapshot.ErrSnapshotNotFound) {
		t.Fatalf("got err=%v, want ErrSnapshotNotFound", err)
	}
}

func TestTypedStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	store := snapshot.NewTypedStore[counterState](newFakeStore(), codec.JSONCodec{})

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Counter", aggID)

	ctx := context.Background()

	if err := store.Save(ctx, snapshot.TypedSnapshot[counterState]{
		StreamID:   aggID,
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

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Counter", aggID)

	ctx := context.Background()

	if err := store.Save(ctx, snapshot.TypedSnapshot[counterState]{
		StreamID:   aggID,
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

func TestTypedStore_NilCodecDefaultsToJSON(t *testing.T) {
	t.Parallel()

	store := snapshot.NewTypedStore[counterState](newFakeStore(), nil)

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Counter", aggID)

	ctx := context.Background()

	if err := store.Save(ctx, snapshot.TypedSnapshot[counterState]{
		StreamID:   aggID,
		StreamType: "Counter",
		Version:    1,
		State:      counterState{Count: 99},
	}); err != nil {
		t.Fatalf("Save with default JSON codec: %v", err)
	}

	got, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.State.Count != 99 {
		t.Fatalf("count = %d, want 99", got.State.Count)
	}
}
