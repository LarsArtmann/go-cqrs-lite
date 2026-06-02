package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func BenchmarkNewEvent(b *testing.B) {
	aggID := id.NewAggregateID()

	for b.Loop() {
		_, err := event.NewEvent(
			event.Type("UserCreated"),
			aggID,
			"User",
			1,
			nil,
		)
		if err != nil {
			b.Fatalf("NewEvent: %v", err)
		}
	}
}

func BenchmarkNewEvent_WithOptions(b *testing.B) {
	aggID := id.NewAggregateID()
	corrID := id.NewCorrelationID()

	for b.Loop() {
		_, err := event.NewEvent(
			event.Type("UserCreated"),
			aggID,
			"User",
			1,
			nil,
			event.WithCorrelationID(corrID),
		)
		if err != nil {
			b.Fatalf("NewEvent: %v", err)
		}
	}
}

func BenchmarkClassify(b *testing.B) {
	err := event.NewTransient("db.timeout", "connection lost")

	for b.Loop() {
		_ = event.Classify(err)
	}
}

func BenchmarkIsRetryable(b *testing.B) {
	err := event.NewTransient("db.timeout", "connection lost")

	for b.Loop() {
		_ = event.IsRetryable(err)
	}
}

func BenchmarkBusPublish(b *testing.B) {
	bus := eventtest.NewFakeBus()

	aggID := id.NewAggregateID()
	events := make([]event.Event, 10)

	for i := range 10 {
		evt, err := event.NewEvent(
			event.Type("UserCreated"),
			aggID,
			"User",
			event.Version(i+1),
			nil,
		)
		if err != nil {
			b.Fatalf("NewEvent: %v", err)
		}

		events[i] = evt
	}

	ctx := context.Background()

	for b.Loop() {
		err := bus.Publish(ctx, events...)
		if err != nil {
			b.Fatalf("Publish: %v", err)
		}
	}
}

func BenchmarkDecodePayload(b *testing.B) {
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent(
		event.Type("UserCreated"),
		aggID,
		"User",
		1,
		[]byte(`{"name":"Alice"}`),
	)
	if err != nil {
		b.Fatalf("NewEvent: %v", err)
	}

	c := codec.JSONCodec{}

	for b.Loop() {
		_, err = event.DecodePayload[map[string]string](evt, c)
		if err != nil {
			b.Fatalf("DecodePayload: %v", err)
		}
	}
}
