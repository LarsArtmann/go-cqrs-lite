package event_test

import (
	"context"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func BenchmarkNewEvent(b *testing.B) {
	b.ReportAllocs()
	streamID := id.NewStreamID()

	for b.Loop() {
		_, err := event.NewEvent(
			event.Type("UserCreated"),
			streamID,
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
	b.ReportAllocs()
	streamID := id.NewStreamID()
	corrID := id.NewCorrelationID()

	for b.Loop() {
		_, err := event.NewEvent(
			event.Type("UserCreated"),
			streamID,
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

func BenchmarkNew_TypedPayload(b *testing.B) {
	b.ReportAllocs()
	streamID := id.NewStreamID()

	for b.Loop() {
		_, err := event.New(
			event.Type("UserCreated"),
			streamID,
			"User",
			1,
			map[string]string{"name": "Alice"},
		)
		if err != nil {
			b.Fatalf("New: %v", err)
		}
	}
}

func BenchmarkClassify(b *testing.B) {
	b.ReportAllocs()
	err := errorfamily.NewTransient("db.timeout", "connection lost")

	for b.Loop() {
		_ = errorfamily.Classify(err)
	}
}

func BenchmarkIsRetryable(b *testing.B) {
	b.ReportAllocs()
	err := errorfamily.NewTransient("db.timeout", "connection lost")

	for b.Loop() {
		_ = errorfamily.IsRetryable(err)
	}
}

func BenchmarkBusPublish(b *testing.B) {
	b.ReportAllocs()
	bus := eventtest.NewFakeBus()

	streamID := id.NewStreamID()
	events := make([]event.Event, 10)

	for i := range 10 {
		evt, err := event.NewEvent(
			event.Type("UserCreated"),
			streamID,
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
	b.ReportAllocs()
	streamID := id.NewStreamID()

	evt, err := event.NewEvent(
		event.Type("UserCreated"),
		streamID,
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
