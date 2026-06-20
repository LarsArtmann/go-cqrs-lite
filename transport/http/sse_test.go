package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestSSEBroker(t *testing.T) {
	bus := eventtest.NewFakeBus()
	defer bus.Close() //nolint:errcheck // test helper

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ch := broker.AddClient("test-client")

	evt, err := event.NewEvent("TestEvent", id.NewAggregateID(), "Test", 1, []byte("test payload"))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case received := <-ch:
		if received.Type() != "TestEvent" {
			t.Fatalf("expected TestEvent, got %s", received.Type())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	if broker.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", broker.ClientCount())
	}
}

func TestSSEHandler(t *testing.T) {
	bus := eventtest.NewFakeBus()
	defer bus.Close() //nolint:errcheck // test helper

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=test", nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	// Wait for SSE handler to register as a client (deterministic)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && broker.ClientCount() == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if broker.ClientCount() == 0 {
		t.Fatal("SSE handler did not register as client within timeout")
	}

	evt, err := event.NewEvent("TestEvent", id.NewAggregateID(), "Test", 1, []byte("test"))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for handler")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected non-empty body")
	}
}

func TestSSEHandler_MissingClient(t *testing.T) {
	bus := eventtest.NewFakeBus()
	defer bus.Close() //nolint:errcheck // test helper

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()

	SSEHandler(broker).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
