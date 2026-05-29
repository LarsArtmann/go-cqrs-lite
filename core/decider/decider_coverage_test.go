package decider_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

type nonImmutableEvent struct{}

func (nonImmutableEvent) ID() id.EventID                     { return id.NewEventID() }
func (nonImmutableEvent) Type() event.Type                   { return "Test" }
func (nonImmutableEvent) AggregateID() id.AggregateID        { return id.NewAggregateID() }
func (nonImmutableEvent) AggregateType() event.AggregateType { return "Test" }
func (nonImmutableEvent) Version() event.Version             { return 1 }
func (nonImmutableEvent) SchemaVersion() event.SchemaVersion {
	sv, _ := event.ParseSchemaVersion(1)

	return sv
}
func (nonImmutableEvent) Payload() []byte           { return nil }
func (nonImmutableEvent) Metadata() *event.Metadata { return nil }
func (nonImmutableEvent) OccurredAt() time.Time     { return time.Now() }

func TestExecute_EnricherAppliesOptions(t *testing.T) {
	t.Parallel()

	enriched := false
	enricher := func(_ context.Context) []event.Option {
		return []event.Option{
			func(_ *event.ImmutableEvent) { enriched = true },
		}
	}

	_, repo := newEnricherRepo(t, enricher)
	aggID := id.NewAggregateID()

	evtFn := func(_ counterState, ver event.Version) ([]event.Event, error) {
		return []event.Event{makeEvent(t, "CounterCreated", aggID, ver+1)}, nil
	}
	if err := executeWithAggID(t, repo, aggID, evtFn); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !enriched {
		t.Error("expected enricher to be applied to events")
	}
}

func TestExecute_EnricherReturnsEmptyOpts(t *testing.T) {
	t.Parallel()

	enricher := func(_ context.Context) []event.Option { return nil }

	_, repo := newEnricherRepo(t, enricher)
	aggID := id.NewAggregateID()

	evtFn := func(_ counterState, ver event.Version) ([]event.Event, error) {
		return []event.Event{makeEvent(t, "CounterCreated", aggID, ver+1)}, nil
	}
	if err := executeWithAggID(t, repo, aggID, evtFn); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestExecute_EnricherSkipsNonImmutableEvents(t *testing.T) {
	t.Parallel()

	enricher := func(_ context.Context) []event.Option {
		return []event.Option{
			func(_ *event.ImmutableEvent) {
				t.Error("enricher should not be called for non-ImmutableEvent")
			},
		}
	}

	_, repo := newEnricherRepo(t, enricher)
	aggID := id.NewAggregateID()

	if err := executeWithAggID(t, repo, aggID, func(_ counterState, _ event.Version) ([]event.Event, error) {
		return []event.Event{nonImmutableEvent{}}, nil
	}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestExecute_EnricherSetsCorrelationID(t *testing.T) {
	t.Parallel()

	correlationID := id.NewCorrelationID()
	enricher := func(_ context.Context) []event.Option {
		return []event.Option{
			event.WithCorrelationID(correlationID),
		}
	}

	bus, repo := newEnricherRepo(t, enricher)
	aggID := id.NewAggregateID()

	if err := executeWithAggID(t, repo, aggID, func(_ counterState, ver event.Version) ([]event.Event, error) {
		return []event.Event{makeEvent(t, "CounterCreated", aggID, ver+1)}, nil
	}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	published := bus.Published
	if len(published) == 0 {
		t.Fatal("expected published events")
	}

	md := published[0].Metadata()
	if md == nil {
		t.Fatal("expected metadata on enriched event")
	}

	if md.CorrelationID != correlationID {
		t.Errorf("expected correlation ID %s, got %s", correlationID, md.CorrelationID)
	}
}

func TestExecute_TransactionalStore_SaveWithOutboxError(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus()
	txStore := &fakeTransactionalStore{
		FakeStore:         testhelpers.NewFakeStore(),
		saveWithOutboxErr: errors.New("tx failed"),
	}

	repo, err := decider.NewRepository(
		txStore, bus, counterDecider(),
		decider.WithOutbox[counterState](testhelpers.NewFakeOutbox()),
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

	repo, err := decider.NewRepository(
		store, bus, counterDecider(),
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
