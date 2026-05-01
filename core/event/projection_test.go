package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
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
