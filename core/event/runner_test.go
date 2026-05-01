package event

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type nopCheckpointStore struct{}

func (nopCheckpointStore) Load(_ context.Context, _ string) (id.EventID, error) {
	return id.EventID{}, nil
}

func (nopCheckpointStore) Save(_ context.Context, _ string, _ id.EventID) error {
	return nil
}

func (nopCheckpointStore) Close() error { return nil }

func TestInMemoryRunner_NilCheckpoint(t *testing.T) {
	t.Parallel()

	_, err := NewInMemoryRunner(nil)
	if err == nil {
		t.Fatal("expected error for nil CheckpointStore")
	}

	if !errors.Is(err, ErrNilCheckpointStore) {
		t.Errorf("error = %v, want ErrNilCheckpointStore", err)
	}
}

func TestInMemoryRunner_RegisterNilProjection(t *testing.T) {
	t.Parallel()

	runner, err := NewInMemoryRunner(nopCheckpointStore{})
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	err = runner.Register(nil)
	if err == nil {
		t.Fatal("expected error for nil projection")
	}
}

func TestInMemoryRunner_RegisterDuplicateName(t *testing.T) {
	t.Parallel()

	runner, err := NewInMemoryRunner(nopCheckpointStore{})
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	proj := NewProjection("dup", func(_ context.Context, _ Event) error { return nil }, nil)

	err = runner.Register(proj)
	if err != nil {
		t.Fatalf("first Register: %v", err)
	}

	err = runner.Register(proj)
	if err == nil {
		t.Fatal("expected error for duplicate projection name")
	}
}

func TestInMemoryRunner_Handle_DispatchesToMatching(t *testing.T) {
	t.Parallel()

	var handled []string

	runner, err := NewInMemoryRunner(nopCheckpointStore{})
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	proj := NewProjection(
		"test",
		func(_ context.Context, evt Event) error {
			handled = append(handled, string(evt.Type()))

			return nil
		},
		[]Type{"UserCreated"},
	)

	err = runner.Register(proj)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	evt, err := NewEvent("UserCreated", id.NewAggregateID(), "User", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = runner.Handle(context.Background(), evt)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if len(handled) != 1 || handled[0] != "UserCreated" {
		t.Errorf("handled = %v, want [UserCreated]", handled)
	}
}

func TestInMemoryRunner_Handle_SkipsNonMatching(t *testing.T) {
	t.Parallel()

	handled := false

	runner, err := NewInMemoryRunner(nopCheckpointStore{})
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	proj := NewProjection(
		"test",
		func(_ context.Context, _ Event) error {
			handled = true

			return nil
		},
		[]Type{"UserCreated"},
	)

	err = runner.Register(proj)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	evt, err := NewEvent("UserDeleted", id.NewAggregateID(), "User", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = runner.Handle(context.Background(), evt)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if handled {
		t.Error("projection should not have been called for non-matching event type")
	}
}

func TestInMemoryRunner_Handle_SubscribesToAll(t *testing.T) {
	t.Parallel()

	count := 0

	runner, err := NewInMemoryRunner(nopCheckpointStore{})
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	proj := NewProjection(
		"all-events",
		func(_ context.Context, _ Event) error {
			count++

			return nil
		},
		nil,
	)

	err = runner.Register(proj)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	evt, err := NewEvent("Anything", id.NewAggregateID(), "User", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = runner.Handle(context.Background(), evt)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestInMemoryRunner_Handle_ProjectionError(t *testing.T) {
	t.Parallel()

	runner, err := NewInMemoryRunner(nopCheckpointStore{})
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	proj := NewProjection(
		"failing",
		func(_ context.Context, _ Event) error {
			return errors.New("boom")
		},
		[]Type{"UserCreated"},
	)

	err = runner.Register(proj)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	evt, err := NewEvent("UserCreated", id.NewAggregateID(), "User", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = runner.Handle(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error from failing projection")
	}
}

func TestInMemoryRunner_HandleParallel_DispatchesConcurrently(t *testing.T) {
	t.Parallel()

	var mtx sync.Mutex

	handled := map[string]bool{}

	runner, err := NewInMemoryRunner(nopCheckpointStore{})
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	for _, name := range []string{"proj-a", "proj-b", "proj-c"} {
		p := NewProjection(name, func(_ context.Context, _ Event) error {
			mtx.Lock()
			handled[name] = true
			mtx.Unlock()

			return nil
		}, nil)

		err = runner.Register(p)
		if err != nil {
			t.Fatalf("Register %s: %v", name, err)
		}
	}

	evt, err := NewEvent("TestEvent", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = runner.HandleParallel(context.Background(), evt)
	if err != nil {
		t.Fatalf("HandleParallel: %v", err)
	}

	mtx.Lock()
	defer mtx.Unlock()

	if len(handled) != 3 {
		t.Fatalf("handled %d projections, want 3", len(handled))
	}
}

func TestInMemoryRunner_HandleParallel_SkipsNonMatching(t *testing.T) {
	t.Parallel()

	called := false

	runner, err := NewInMemoryRunner(nopCheckpointStore{})
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	proj := NewProjection("filtered", func(_ context.Context, _ Event) error {
		called = true

		return nil
	}, []Type{"UserCreated"})

	err = runner.Register(proj)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	evt, err := NewEvent("OtherEvent", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = runner.HandleParallel(context.Background(), evt)
	if err != nil {
		t.Fatalf("HandleParallel: %v", err)
	}

	if called {
		t.Error("projection should not have been called for non-matching type")
	}
}

func TestInMemoryRunner_HandleParallel_ProjectionError(t *testing.T) {
	t.Parallel()

	runner, err := NewInMemoryRunner(nopCheckpointStore{})
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	proj := NewProjection("failing", func(_ context.Context, _ Event) error {
		return errors.New("boom")
	}, nil)

	err = runner.Register(proj)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	evt, err := NewEvent("TestEvent", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = runner.HandleParallel(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error from failing projection")
	}
}

func TestInMemoryRunner_HandleParallel_NoProjections(t *testing.T) {
	t.Parallel()

	runner, err := NewInMemoryRunner(nopCheckpointStore{})
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	evt, err := NewEvent("TestEvent", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = runner.HandleParallel(context.Background(), evt)
	if err != nil {
		t.Fatalf("HandleParallel with no projections: %v", err)
	}
}

func TestInMemoryRunner_HandleParallel_PartialFailure_StillRunsOthers(t *testing.T) {
	t.Parallel()

	var mtx sync.Mutex

	handled := map[string]bool{}

	runner, err := NewInMemoryRunner(nopCheckpointStore{})
	if err != nil {
		t.Fatalf("NewInMemoryRunner: %v", err)
	}

	err = runner.Register(NewProjection("ok-1", func(_ context.Context, _ Event) error {
		mtx.Lock()
		handled["ok-1"] = true
		mtx.Unlock()

		return nil
	}, nil))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	err = runner.Register(NewProjection("failing", func(_ context.Context, _ Event) error {
		mtx.Lock()
		handled["failing"] = true
		mtx.Unlock()

		return errors.New("fail")
	}, nil))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	err = runner.Register(NewProjection("ok-2", func(_ context.Context, _ Event) error {
		mtx.Lock()
		handled["ok-2"] = true
		mtx.Unlock()

		return nil
	}, nil))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	evt, err := NewEvent("TestEvent", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = runner.HandleParallel(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error from failing projection")
	}

	mtx.Lock()
	defer mtx.Unlock()

	if len(handled) != 3 {
		t.Fatalf("handled %d projections, want 3 (all run despite failure)", len(handled))
	}
}
