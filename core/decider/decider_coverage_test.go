package decider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

type fakeTransactionalStore struct {
	*testhelpers.FakeStore

	saveWithOutboxErr error
}

func (f *fakeTransactionalStore) SaveWithOutbox(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ []event.Event,
	_ event.Version,
) error {
	return f.saveWithOutboxErr
}

type failingCodec struct {
	err error
}

func (f *failingCodec) Encode(_ any) ([]byte, error) { return nil, f.err }
func (f *failingCodec) Decode(_ []byte, _ any) error { return nil }

func TestExecute_TransactionalStore_SaveWithOutboxError(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus()
	outbox := testhelpers.NewFakeOutbox()
	store := &fakeTransactionalStore{
		FakeStore:         testhelpers.NewFakeStore(),
		saveWithOutboxErr: errors.New("tx failed"),
	}

	d := counterDecider()

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithOutbox[counterState](outbox),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = repo.Execute(
		context.Background(), aggID, "Counter",
		func(_ counterState, ver event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, ver+1)}, nil
		},
	)
	if err == nil {
		t.Fatal("expected error from SaveWithOutbox")
	}
}

func TestExecute_SnapshotCodecEncodeError(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus()
	store := testhelpers.NewFakeStore()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	codec := &failingCodec{err: errors.New("encode failed")}

	d := counterDecider()

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
		decider.WithSnapshotStrategy[counterState](event.MustEveryNEvents(1)),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	err = repo.Execute(
		context.Background(), aggID, "Counter",
		func(_ counterState, ver event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, ver+1)}, nil
		},
	)
	if err != nil {
		t.Fatalf("Execute should succeed even if snapshot encode fails: %v", err)
	}
}

func TestLoadFromSnapshot_FoldError(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus()
	store := testhelpers.NewFakeStore()
	snapshotStore := testhelpers.NewFakeSnapshotStore()
	codec := event.JSONCodec{}

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold: func(_ counterState, _ event.Event) (counterState, error) {
			return counterState{}, errors.New("fold always fails")
		},
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewAggregateID()

	state, snapErr := codec.Encode(counterState{Value: 42})
	if snapErr != nil {
		t.Fatalf("encode: %v", snapErr)
	}

	snap := event.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Counter",
		Version:       event.Version(1),
		State:         state,
	}

	snapErr = snapshotStore.Save(context.Background(), snap)
	if snapErr != nil {
		t.Fatalf("save snapshot: %v", snapErr)
	}

	snapshotStore.SetSnapshot(&snap)

	evt := makeEvent(t, "CounterIncremented", aggID, 2)
	mustAppendBatch(t, store, "Counter", aggID, []event.Event{evt})

	_, _, loadErr := repo.Load(context.Background(), aggID, "Counter")
	if loadErr == nil {
		t.Fatal("expected fold error from Load after snapshot")
	}
}
