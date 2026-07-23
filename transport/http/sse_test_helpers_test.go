package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// newFakeStoreWithEvents creates a FakeStore, saves events built from payloads
// under a fresh aggregate, and returns the store and events. Events use
// sequential versions starting at 1. Extracted to deduplicate the event-setup
// boilerplate shared across SSE handler tests.
func newFakeStoreWithEvents(
	t *testing.T,
	eventType event.Type,
	aggType id.StreamType,
	payloads ...[]byte,
) (
	*eventtest.FakeStore, []event.Event,
) {
	t.Helper()
	store := eventtest.NewFakeStore()
	aggID := id.NewStreamID()
	ref := id.NewStreamRef(aggType, aggID)
	events := make([]event.Event, len(payloads))
	for i, p := range payloads {
		evt, err := event.NewEvent(eventType, aggID, aggType, event.Version(i+1), p)
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}

		events[i] = evt
	}

	if err := store.Save(context.Background(), ref, events, event.Version(0)); err != nil {
		t.Fatalf("save events: %v", err)
	}

	return store, events
}

// startSSE launches SSEHandler(broker) in a goroutine on a fresh cancellable
// request and returns the recorder plus a stop function. The stop function
// cancels the context and waits for the handler to finish. Callers typically
// sleep (and optionally publish to the bus) between startSSE and stop.
//
// Extracted to deduplicate the ctx/req/recorder/goroutine scaffolding repeated
// across every SSE handler test.
func startSSE(broker *SSEBroker, client, lastEventID string) (
	*httptest.ResponseRecorder, func(),
) {
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client="+client, nil)
	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}

	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	return rec, func() {
		cancel()
		<-done
	}
}
