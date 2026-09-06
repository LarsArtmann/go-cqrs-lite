package watermill

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func benchEvent(tb testing.TB) event.Event {
	tb.Helper()

	streamID := id.NewStreamID()
	evt, err := event.NewEvent(
		"BenchEvent", streamID, "Bench", 1,
		[]byte(`{"name":"test"}`),
		event.WithCorrelationID(id.NewCorrelationID()),
		event.WithOccurredAt(time.Now()),
	)
	if err != nil {
		tb.Fatal(err)
	}

	return evt
}

func BenchmarkEventToMessage(b *testing.B) {
	b.ReportAllocs()

	evt := benchEvent(b)

	b.ResetTimer()

	for b.Loop() {
		_ = eventToMessage(evt)
	}
}

func BenchmarkMessageToEvent(b *testing.B) {
	b.ReportAllocs()

	evt := benchEvent(b)
	msg := eventToMessage(evt)

	b.ResetTimer()

	for b.Loop() {
		_, err := MessageToEvent("Bench", msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPublisherAdapter_Publish(b *testing.B) {
	b.ReportAllocs()

	bus := eventtest.NewFakeBus()
	adapter := NewPublisherAdapter(bus)
	b.Cleanup(func() { _ = adapter.Close(); _ = bus.Close() })

	msg := eventToMessage(benchEvent(b))

	b.ResetTimer()

	for b.Loop() {
		err := adapter.Publish("Bench", msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildMetadata(b *testing.B) {
	b.ReportAllocs()

	md := message.Metadata{
		"event_id":       id.NewEventID().String(),
		"event_type":     "BenchEvent",
		"aggregate_id":   id.NewStreamID().String(),
		"aggregate_type": "Bench",
		"version":        "1",
		"schema_version": "1",
		"encoding":       "json",
		"occurred_at":    time.Now().Format(time.RFC3339Nano),
		"correlation_id": id.NewCorrelationID().String(),
	}

	b.ResetTimer()

	for b.Loop() {
		_, err := buildMetadata(md)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchCommand(tb testing.TB) *command.BasicCommand {
	tb.Helper()

	cmd, err := command.New(
		"BenchCommand", id.NewStreamID(),
		command.WithCorrelationID(id.NewCorrelationID()),
		command.WithUserID(id.NewUserID()),
		command.WithCustomMetadata("tenant", "acme"),
	)
	if err != nil {
		tb.Fatal(err)
	}

	return cmd
}

func BenchmarkCommandToMessage(b *testing.B) {
	b.ReportAllocs()

	cmd := benchCommand(b)

	b.ResetTimer()

	for b.Loop() {
		_ = CommandToMessage(cmd)
	}
}

func BenchmarkMessageToCommand(b *testing.B) {
	b.ReportAllocs()

	msg := CommandToMessage(benchCommand(b))

	b.ResetTimer()

	for b.Loop() {
		_, err := MessageToCommand("BenchCommand", msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}
