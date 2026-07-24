package decider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

func executeCreateEvent(
	t *testing.T,
	repo *decider.Repository[counterState],
	streamID id.StreamID,
) error {
	t.Helper()

	return repo.Execute(
		context.Background(), streamID, "Counter",
		counterCreatedEventFn(t, streamID),
	)
}

type failingCodec struct {
	err error
}

func (f *failingCodec) Encoding() codec.Encoding     { return codec.EncodingJSON }
func (f *failingCodec) Encode(_ any) ([]byte, error) { return nil, f.err }
func (f *failingCodec) Decode(_ []byte, _ any) error { return nil }

func TestExecute_EnricherAppliesOptions(t *testing.T) {
	t.Parallel()

	enriched := false
	enricher := func(_ context.Context) []event.Option {
		return []event.Option{
			func(_ event.Event) { enriched = true },
		}
	}

	_, repo := newEnricherRepo(t, enricher)
	streamID := id.NewStreamID()

	if err := executeWithAggID(t, repo, streamID, counterCreatedEventFn(t, streamID)); err != nil {
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
	streamID := id.NewStreamID()

	if err := executeWithAggID(t, repo, streamID, counterCreatedEventFn(t, streamID)); err != nil {
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
	streamID := id.NewStreamID()

	if err := executeWithAggID(
		t,
		repo,
		streamID,
		counterCreatedEventFn(t, streamID),
	); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	published := bus.Published
	if len(published) == 0 {
		t.Fatal("expected published events")
	}

	md := published[0].Metadata()
	if md.CorrelationID != correlationID {
		t.Errorf("expected correlation ID %s, got %s", correlationID, md.CorrelationID)
	}
}

func TestExecute_SnapshotCodecEncodeError(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	store := eventtest.NewFakeStore()
	snapshotStore := eventtest.NewFakeSnapshotStore()
	codec := &failingCodec{err: errors.New("encode failed")}

	repo, err := decider.NewRepository(
		store, bus, counterDecider(),
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
		decider.WithSnapshotStrategy[counterState](everyN(1)),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	streamID := id.NewStreamID()

	err = executeCreateEvent(t, repo, streamID)
	if err != nil {
		t.Fatalf("Execute should succeed even if snapshot encode fails: %v", err)
	}
}

func TestLoadFromSnapshot_FoldError(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	store := eventtest.NewFakeStore()
	snapshotStore := eventtest.NewFakeSnapshotStore()
	codec := codec.JSONCodec{}

	d := decider.Decider[counterState]{
		Initial: counterState{Value: 0},
		Apply:   failingApply("apply always fails"),
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapshotStore),
		decider.WithCodec[counterState](codec),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	streamID := id.NewStreamID()

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterCreated", streamID, 1),
	})

	state, snapErr := codec.Encode(counterState{Value: 42})
	if snapErr != nil {
		t.Fatalf("encode: %v", snapErr)
	}

	snap := snapshot.Snapshot{
		StreamID:   streamID,
		StreamType: "Counter",
		Version:    event.Version(1),
		State:      state,
	}

	snapErr = snapshotStore.Save(context.Background(), snap)
	if snapErr != nil {
		t.Fatalf("save snapshot: %v", snapErr)
	}

	snapshotStore.SetSnapshot(&snap)

	evt := makeEvent(t, "CounterIncremented", streamID, 2)
	mustAppendBatch(t, store, "Counter", streamID, []event.Event{evt})

	_, _, loadErr := repo.Load(context.Background(), streamID, "Counter")
	if loadErr == nil {
		t.Fatal("expected apply error from Load after snapshot")
	}
}
