package decider_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestExecuteWithResult_Created(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Fold:    func(s counterState, _ event.Event) (counterState, error) { return s, nil },
	}

	repo, _ := decider.NewRepository[counterState](store, bus, d)

	result, err := repo.ExecuteWithResult(
		context.Background(), aggID, "Test",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			evt, _ := event.NewEvent("test.created", aggID, "Test", v.Add(1), []byte(`{}`))
			return []event.Event{evt}, nil
		},
	)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if !result.Created {
		t.Error("expected Created = true")
	}

	if result.Updated {
		t.Error("expected Updated = false")
	}

	if result.NoOp {
		t.Error("expected NoOp = false")
	}

	if result.Version.Int() != 1 {
		t.Errorf("version = %d, want 1", result.Version.Int())
	}

	if len(result.Events) != 1 {
		t.Errorf("events = %d, want 1", len(result.Events))
	}
}

func TestExecuteWithResult_Updated(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Fold:    func(s counterState, _ event.Event) (counterState, error) { return s, nil },
	}

	repo, _ := decider.NewRepository[counterState](store, bus, d)

	evt, _ := event.NewEvent("test.init", aggID, "Test", event.Version(1), []byte(`{}`))
	_ = store.Save(context.Background(), "Test", aggID, []event.Event{evt}, event.Version(0))

	result, err := repo.ExecuteWithResult(
		context.Background(), aggID, "Test",
		func(_ counterState, v event.Version) ([]event.Event, error) {
			evt2, _ := event.NewEvent("test.updated", aggID, "Test", v.Add(1), []byte(`{}`))
			return []event.Event{evt2}, nil
		},
	)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if result.Created {
		t.Error("expected Created = false")
	}

	if !result.Updated {
		t.Error("expected Updated = true")
	}

	if result.Version.Int() != 2 {
		t.Errorf("version = %d, want 2", result.Version.Int())
	}
}

func TestExecuteWithResult_NoOp(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	bus := testhelpers.NewFakeBus()
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Fold:    func(s counterState, _ event.Event) (counterState, error) { return s, nil },
	}

	repo, _ := decider.NewRepository[counterState](store, bus, d)

	result, err := repo.ExecuteWithResult(
		context.Background(), aggID, "Test",
		func(_ counterState, _ event.Version) ([]event.Event, error) {
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if !result.NoOp {
		t.Error("expected NoOp = true")
	}

	if result.Created || result.Updated {
		t.Error("expected no creation or update")
	}
}
