package event

import (
	"context"
	"errors"
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

func TestInMemoryRunner_NilCheckpoint(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil CheckpointStore")
		}

		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic value = %v (%T), want string", r, r)
		}

		if msg != "event: nil CheckpointStore" {
			t.Errorf("panic message = %q, want %q", msg, "event: nil CheckpointStore")
		}
	}()

	NewInMemoryRunner(nil)
}

func TestInMemoryRunner_RegisterNilProjection(t *testing.T) {
	t.Parallel()

	runner := NewInMemoryRunner(nopCheckpointStore{})

	err := runner.Register(nil)
	if err == nil {
		t.Fatal("expected error for nil projection")
	}
}

func TestInMemoryRunner_RegisterDuplicateName(t *testing.T) {
	t.Parallel()

	runner := NewInMemoryRunner(nopCheckpointStore{})

	proj := NewProjection("dup", func(_ context.Context, _ Event) error { return nil }, nil)

	err := runner.Register(proj)
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

	runner := NewInMemoryRunner(nopCheckpointStore{})

	proj := NewProjection(
		"test",
		func(_ context.Context, evt Event) error {
			handled = append(handled, string(evt.Type()))

			return nil
		},
		[]Type{"UserCreated"},
	)

	err := runner.Register(proj)
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

	runner := NewInMemoryRunner(nopCheckpointStore{})

	proj := NewProjection(
		"test",
		func(_ context.Context, _ Event) error {
			handled = true

			return nil
		},
		[]Type{"UserCreated"},
	)

	err := runner.Register(proj)
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

	runner := NewInMemoryRunner(nopCheckpointStore{})

	proj := NewProjection(
		"all-events",
		func(_ context.Context, _ Event) error {
			count++

			return nil
		},
		nil,
	)

	err := runner.Register(proj)
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

	runner := NewInMemoryRunner(nopCheckpointStore{})

	proj := NewProjection(
		"failing",
		func(_ context.Context, _ Event) error {
			return errors.New("boom")
		},
		[]Type{"UserCreated"},
	)

	err := runner.Register(proj)
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
