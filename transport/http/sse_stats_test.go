package http

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestSSEBroker_Stats(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	if s := broker.Stats(); len(s) != 0 {
		t.Errorf("expected 0 clients initially, got %d", len(s))
	}

	ch1 := broker.AddClient("client-1")
	_ = broker.AddClient("client-2")

	stats := broker.Stats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(stats))
	}

	// Publish an event — it should buffer in both channels.
	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("test.event", aggID, "Test", 1, []byte(`{}`))
	_ = bus.Publish(context.Background(), evt)

	time.Sleep(50 * time.Millisecond)

	stats = broker.Stats()
	for _, s := range stats {
		if s.BufferedEvents != 1 {
			t.Errorf("client %s: expected 1 buffered event, got %d", s.ID, s.BufferedEvents)
		}
	}

	// Drain one channel.
	<-ch1

	stats = broker.Stats()
	for _, s := range stats {
		if s.ID == "client-1" && s.BufferedEvents != 0 {
			t.Errorf("client-1: expected 0 buffered after drain, got %d", s.BufferedEvents)
		}
	}
}

func TestSSEBroker_CloseWithGrace(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatal(err)
	}

	ch := broker.AddClient("grace-test")

	// Buffer an event.
	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("test.event", aggID, "Test", 1, []byte(`{}`))
	_ = bus.Publish(context.Background(), evt)

	time.Sleep(50 * time.Millisecond)

	// CloseWithGrace should let us drain the buffered event.
	start := time.Now()
	broker.CloseWithGrace(100 * time.Millisecond)
	elapsed := time.Since(start)

	// Should have waited at least ~100ms.
	if elapsed < 90*time.Millisecond {
		t.Errorf("expected ~100ms grace, close returned in %v", elapsed)
	}

	// Drain the buffered event first, then check channel is closed.
	<-ch
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after CloseWithGrace")
	}
}
