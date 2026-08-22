package decider_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestRefForms_MatchPairForms pins the v4 lockstep contract: the deprecated
// (streamID, streamType) pair forms and the *Ref forms address the exact
// same stream and produce the exact same outcomes. The pair forms only
// forward (removed in v5); this test is the guard that the forwarding is
// lossless.
func TestRefForms_MatchPairForms(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	ctx := t.Context()
	sid := id.NewStreamID()
	ref := id.NewStreamRef("Counter", sid)

	if err := repo.ExecuteRef(ctx, ref, func(_ counterState, v event.Version) ([]event.Event, error) {
		return []event.Event{makeCounterEvent("CounterIncremented", sid, v+1)}, nil
	}); err != nil {
		t.Fatalf("ExecuteRef: %v", err)
	}

	state, version, err := repo.Load(ctx, sid, "Counter")
	if err != nil {
		t.Fatalf("Load (pair): %v", err)
	}

	stateRef, versionRef, err := repo.LoadRef(ctx, ref)
	if err != nil {
		t.Fatalf("LoadRef: %v", err)
	}

	if state != stateRef || version != versionRef {
		t.Fatalf("pair/ref mismatch: (%v,%v) vs (%v,%v)", state, version, stateRef, versionRef)
	}

	if version != 1 {
		t.Fatalf("version = %d, want 1", version)
	}
}

// TestRefForms_LoadAtVersionAndTime pins the time-travel ref forms against
// the deprecated pair forms.
func TestRefForms_LoadAtVersionAndTime(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	ctx := t.Context()
	sid := id.NewStreamID()
	ref := id.NewStreamRef("Counter", sid)

	for range 3 {
		err := repo.ExecuteRef(ctx, ref, func(_ counterState, v event.Version) ([]event.Event, error) {
			return []event.Event{makeCounterEvent("CounterIncremented", sid, v+1)}, nil
		})
		if err != nil {
			t.Fatalf("ExecuteRef: %v", err)
		}
	}

	atVPair, verPair, err := repo.LoadAtVersion(ctx, sid, "Counter", 2)
	if err != nil {
		t.Fatalf("LoadAtVersion (pair): %v", err)
	}

	atVRef, verRef, err := repo.LoadAtVersionRef(ctx, ref, 2)
	if err != nil {
		t.Fatalf("LoadAtVersionRef: %v", err)
	}

	if atVPair != atVRef || verPair != verRef {
		t.Fatalf("LoadAtVersion pair/ref mismatch: (%v,%v) vs (%v,%v)", atVPair, verPair, atVRef, verRef)
	}

	future := time.Now().Add(time.Hour)

	atTPair, _, err := repo.LoadAtTime(ctx, sid, "Counter", future)
	if err != nil {
		t.Fatalf("LoadAtTime (pair): %v", err)
	}

	atTRef, _, err := repo.LoadAtTimeRef(ctx, ref, future)
	if err != nil {
		t.Fatalf("LoadAtTimeRef: %v", err)
	}

	if atTPair != atTRef {
		t.Fatalf("LoadAtTime pair/ref mismatch: %v vs %v", atTPair, atTRef)
	}
}

// TestRefForms_TypedRepository pins the typed wrapper's ref forms against
// its deprecated pair forms.
func TestRefForms_TypedRepository(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()

	typed := decider.TypedDecider[counterState, incrementCmd]{
		Initial: counterState{Value: 0},
		Decide: func(state counterState, _ incrementCmd) ([]event.Event, error) {
			return []event.Event{makeCounterEvent("CounterIncremented", typedTestStreamID,
				event.Version(state.Value+1))}, nil
		},
		Apply: applyCounter,
	}

	repo, err := decider.NewTypedRepository[counterState, incrementCmd](store, bus, typed)
	if err != nil {
		t.Fatalf("NewTypedRepository: %v", err)
	}

	ref := id.NewStreamRef("Counter", typedTestStreamID)
	ctx := t.Context()

	if err := repo.ExecuteCommandRef(ctx, ref, incrementCmd{}); err != nil {
		t.Fatalf("ExecuteCommandRef: %v", err)
	}

	state, version, err := repo.LoadRef(ctx, ref)
	if err != nil {
		t.Fatalf("LoadRef: %v", err)
	}

	if version != 1 {
		t.Fatalf("typed version = %d, want 1", version)
	}

	pairState, pairVersion, err := repo.Load(ctx, typedTestStreamID, "Counter")
	if err != nil {
		t.Fatalf("Load (pair): %v", err)
	}

	if pairState != state || pairVersion != version {
		t.Fatal("typed pair/ref mismatch")
	}
}

var typedTestStreamID = id.NewStreamID()
