package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
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

func BenchmarkPublishChanges(b *testing.B) {
	bus := testhelpers.NewFakeBus()

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
		err := event.PublishChanges(ctx, bus, nil, events)
		if err != nil {
			b.Fatalf("PublishChanges: %v", err)
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

	codec := event.JSONCodec{}

	for b.Loop() {
		_, err = event.DecodePayload[map[string]string](evt, codec)
		if err != nil {
			b.Fatalf("DecodePayload: %v", err)
		}
	}
}
