package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
)

func TestMemoryBusP_publish(t *testing.T) {
	bus := event.NewMemoryBus()
	ctx := context.Background()

	received := make([]event.Event, 0)

	handler := func(ctx context.Context, evt event.Event) error {
	 received = append(received, evt)
        return nil
    }

}

}

