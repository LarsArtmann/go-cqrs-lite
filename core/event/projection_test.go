package event_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestProjectionFunc(t *testing.T) {
	t.Parallel()

	var handled []string

	proj := event.NewProjection(
		"test-projection",
		func(_ context.Context, evt event.Event) error {
			handled = append(handled, string(evt.Type()))

			return nil
		},
		[]event.Type{"UserCreated", "UserUpdated"},
	)

	if proj.Name() != "test-projection" {
		t.Errorf("Name = %q, want test-projection", proj.Name())
	}

	if len(proj.EventTypes()) != 2 {
		t.Errorf("EventTypes count = %d, want 2", len(proj.EventTypes()))
	}
}

func TestProjectionFunc_Handle(t *testing.T) {
	t.Parallel()

	var handled string

	proj := event.NewProjection(
		"test",
		func(_ context.Context, evt event.Event) error {
			handled = string(evt.Type())

			return nil
		},
		nil,
	)

	evt, err := event.NewEvent("UserCreated", id.NewAggregateID(), "User", 1, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = proj.Handle(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if handled != "UserCreated" {
		t.Errorf("handled = %q, want UserCreated", handled)
	}
}

func TestProjectionFunc_HandleError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("projection failed")

	proj := event.NewProjection("test", func(_ context.Context, _ event.Event) error {
		return wantErr
	}, nil)

	evt, err := event.NewEvent("UserCreated", id.NewAggregateID(), "User", 1, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = proj.Handle(context.Background(), evt)
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}
