package decider_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/decider"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/event/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/snapshot"
)

type counterState struct {
	Value int
}

func mustAppendBatch(
	t *testing.T,
	store event.Store,
	aggType event.AggregateType,
	aggID id.AggregateID,
	events []event.Event,
) {
	t.Helper()

	err := store.AppendBatch(t.Context(), event.NewAggregateRef(aggType, aggID), events)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}
}

func foldCounter(state counterState, evt event.Event) (counterState, error) {
	switch evt.Type() {
	case "CounterCreated":
		return counterState{Value: 1}, nil
	case "CounterIncremented":
		return counterState{Value: state.Value + 1}, nil
	}

	return state, nil
}

func newTestRepo(
	t *testing.T,
) (*decider.Repository[counterState], *eventtest.FakeStore, *eventtest.FakeBus) {
	t.Helper()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Fold:    foldCounter,
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	return repo, store, bus
}

func newBenchRepo(
	b *testing.B,
) (*decider.Repository[counterState], context.Context) {
	b.Helper()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	d := counterDecider()
	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		b.Fatalf("NewRepository: %v", err)
	}

	return repo, context.Background()
}

func newCounterSnapshotRepo(
	t *testing.T,
	store event.Store,
	bus *eventtest.FakeBus,
	snapshotStore *eventtest.FakeSnapshotStore,
	c codec.Codec,
	opts ...decider.RepositoryOption[counterState],
) *decider.Repository[counterState] {
	t.Helper()

	repo, err := decider.NewRepository(
		store, bus, counterDecider(),
		append([]decider.RepositoryOption[counterState]{
			decider.WithSnapshotStore[counterState](snapshotStore),
			decider.WithCodec[counterState](c),
		}, opts...)...,
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	return repo
}

func newSnapshotSetup(
	t *testing.T,
) (*eventtest.FakeStore, *eventtest.FakeBus, *eventtest.FakeSnapshotStore, codec.Codec) {
	t.Helper()

	return eventtest.NewFakeStore(), eventtest.NewFakeBus(),
		eventtest.NewFakeSnapshotStore(), codec.JSONCodec{}
}

func requireLoadState(
	t *testing.T,
	repo *decider.Repository[counterState],
	aggID id.AggregateID,
	expectValue int,
	expectVersion event.Version,
) {
	t.Helper()

	state, version, err := repo.Load(t.Context(), aggID, "Counter")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if state.Value != expectValue {
		t.Fatalf("expected Value=%d, got %d", expectValue, state.Value)
	}

	if version != expectVersion {
		t.Fatalf("expected version %d, got %d", expectVersion, version)
	}
}

func counterDecider() decider.Decider[counterState] {
	return decider.Decider[counterState]{Initial: counterState{Value: 0}, Fold: foldCounter}
}

func failingFold(msg string) func(counterState, event.Event) (counterState, error) {
	return func(_ counterState, _ event.Event) (counterState, error) {
		return counterState{}, errors.New(msg)
	}
}

func failingDecider() decider.Decider[counterState] {
	return decider.Decider[counterState]{
		Initial: counterState{},
		Fold:    failingFold("corrupted payload"),
	}
}

func newFailingRepo(
	t *testing.T,
) (*decider.Repository[counterState], *eventtest.FakeStore) {
	t.Helper()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	repo, err := decider.NewRepository(store, bus, failingDecider())
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	return repo, store
}

func makeSnapshot(
	t *testing.T,
	c codec.Codec,
	aggID id.AggregateID,
	value int,
	version event.Version,
) snapshot.Snapshot {
	t.Helper()

	snapState, _ := c.Encode(counterState{Value: value})

	return snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Counter",
		Version:       version,
		State:         snapState,
		CreatedAt:     time.Now(),
	}
}

func saveSnapshot(
	t *testing.T,
	snapshotStore *eventtest.FakeSnapshotStore,
	c codec.Codec,
	aggID id.AggregateID,
	value int,
	version event.Version,
) {
	t.Helper()

	_ = snapshotStore.Save(t.Context(), makeSnapshot(t, c, aggID, value, version))
}

func setSnapshot(
	t *testing.T,
	snapshotStore *eventtest.FakeSnapshotStore,
	c codec.Codec,
	aggID id.AggregateID,
	value int,
	version event.Version,
) {
	t.Helper()

	snap := makeSnapshot(t, c, aggID, value, version)
	snapshotStore.SetSnapshot(&snap)
}

func newEnricherRepo(
	t *testing.T,
	enricher func(context.Context) []event.Option,
	opts ...decider.RepositoryOption[counterState],
) (*eventtest.FakeBus, *decider.Repository[counterState]) {
	t.Helper()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	d := counterDecider()
	allOpts := append([]decider.RepositoryOption[counterState]{
		decider.WithEnricher[counterState](enricher),
	}, opts...)
	repo, err := decider.NewRepository(store, bus, d, allOpts...)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	return bus, repo
}

func executeWithAggID(
	t *testing.T,
	repo *decider.Repository[counterState],
	aggID id.AggregateID,
	fn func(counterState, event.Version) ([]event.Event, error),
) error {
	t.Helper()

	return repo.Execute(context.Background(), aggID, "Counter", fn)
}

func counterCreatedEventFn(
	t *testing.T,
	aggID id.AggregateID,
) func(counterState, event.Version) ([]event.Event, error) {
	t.Helper()

	return func(_ counterState, ver event.Version) ([]event.Event, error) {
		return []event.Event{makeEvent(t, "CounterCreated", aggID, ver+1)}, nil
	}
}

func makeEvent(
	t *testing.T,
	eventType string,
	aggID id.AggregateID,
	version event.Version,
) *event.ImmutableEvent {
	t.Helper()

	evt, err := event.NewEvent(event.Type(eventType), aggID, "Counter", version, []byte("{}"))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func executeCounter(
	t *testing.T,
	repo *decider.Repository[counterState],
	aggID id.AggregateID,
	expectedValue int,
	expectedVersion event.Version,
	eventType string,
	eventVersion event.Version,
) {
	t.Helper()

	err := repo.Execute(
		t.Context(), aggID, "Counter",
		func(state counterState, version event.Version) ([]event.Event, error) {
			if state.Value != expectedValue {
				t.Fatalf("expected Value=%d, got %d", expectedValue, state.Value)
			}

			if version != expectedVersion {
				t.Fatalf("expected version %d, got %d", expectedVersion, version)
			}

			return []event.Event{makeEvent(t, eventType, aggID, eventVersion)}, nil
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func executeAndIncrement(
	t *testing.T,
	repo *decider.Repository[counterState],
	aggID id.AggregateID,
	eventType string,
) error {
	t.Helper()

	return repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, eventType, aggID, v.Increment())}, nil
		},
	)
}

func executeCreate(
	t *testing.T,
	repo *decider.Repository[counterState],
	aggID id.AggregateID,
) error {
	t.Helper()

	return repo.Execute(
		t.Context(), aggID, "Counter",
		func(_ counterState, _ event.Version) ([]event.Event, error) {
			return []event.Event{makeEvent(t, "CounterCreated", aggID, 1)}, nil
		},
	)
}

type errStore struct {
	event.Store

	loadToVersionErr   error
	loadToTimestampErr error
}

func (e *errStore) LoadToVersion(
	_ context.Context,
	_ event.AggregateRef,
	_ event.Version,
) ([]event.Event, error) {
	return nil, e.loadToVersionErr
}

func (e *errStore) LoadToTimestamp(
	_ context.Context,
	_ event.AggregateRef,
	_ time.Time,
) ([]event.Event, error) {
	return nil, e.loadToTimestampErr
}

type ctxCheckStore struct{ event.Store }

func (c *ctxCheckStore) checkCtx(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}

func (c *ctxCheckStore) Save(
	ctx context.Context,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if err := c.checkCtx(ctx); err != nil {
		return err
	}

	return c.Store.Save(ctx, ref, events, expectedVersion)
}

func (c *ctxCheckStore) Load(
	ctx context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	if err := c.checkCtx(ctx); err != nil {
		return nil, err
	}

	return c.Store.Load(ctx, ref)
}
