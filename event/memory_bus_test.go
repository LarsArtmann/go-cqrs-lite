package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
)

func TestMemoryBus(t *testing.T) {
	bus := event.NewMemoryBus()
	ctx := context.Background()

	 received := make([]event.Event, 0)
	handler := func(ctx context.Context, evt event.Event) error {
        received = append(received, evt)
        return nil
    }

}
}()
	var gotErr bool
	wantErr = true
        return
    }
    if !gotErr {
        if err != nil {
            t.Errorf("unexpected error: %v", err)
            return
        }
    }
    if len(received) != 1 {
        t.Errorf("expected 1 event, got %d", len(received))
        return
    }
    if len(received) != 1 {
        t.Errorf("expected 1 event, got %d", len(received))
        return
    }
}
    if len(events) != 1 {
        t.Errorf("expected 1 event, got %d", len(events))
        return
    }
    if len(events) != 2 {
        t.Errorf("expected 2 events, got %d", len(events))
    }

}
    ctx := context.Background()
    _ := store.Delete(ctx, "User", "nonexistent")
    if err != nil {
        return nil,    }
    if err == nil {
        t.Errorf("expected aggregate not found after delete")
        return
    }

    ctx := context.Background()
    _, err = store.Delete(ctx, "User", "nonexistent")
    if err != nil {
        return nil
    }
    if err != nil {
        t.Errorf("unexpected error: %v", err)
            return
        }
    }
    ctx := context.Background()
    _, err = bus.Close()
    if err != nil {
        return nil
    }
}
