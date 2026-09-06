package watermill

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
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

// BenchmarkCatchUp_ReplayThroughput measures the end-to-end catch-up replay
// pipeline: journal read + event→message conversion + channel hop + consumer
// ack + checkpoint save, per event. Delivery is serialized per subscription
// (forward, then wait for the Ack before the next message) — this is the
// at-least-once checkpoint contract, not an oversight, so the numbers below
// are the ceiling for one subscription against an in-memory journal and
// checkpoint store. Real deployments add broker + network latency on top.
//
// 2026-09-06 baseline: 3.6-6.2 µs/event (~160-280K events/s) at ambient
// load avg 37-59 on 32 cores (an idle machine will land lower). Treat the
// gate as order-of-magnitude: if a future change degrades this by 10x,
// ack-window pipelining (N in-flight unacked messages) is the designed
// remedy — it trades strictly-serialized checkpointing for throughput and
// was deliberately NOT added while the ceiling stayed in this range.
func BenchmarkCatchUp_ReplayThroughput(b *testing.B) {
	b.ReportAllocs()

	const total = 1000

	store := eventtest.NewFakeStore()

	streamID := id.NewStreamID()

	events := make([]event.Event, 0, total)
	for range total {
		evt, _ := event.NewEvent(
			"bench.catchup", streamID, "Bench", event.Version(1),
			[]byte(`{}`),
		)
		events = append(events, evt)
	}

	_ = store.AppendBatch(context.Background(),
		id.NewStreamRef("Bench", streamID), events)

	b.ResetTimer()

	for b.Loop() {
		bus := eventtest.NewFakeBus()

		catchUp, err := NewCatchUpSubscriber(
			store, NewSubscriberAdapter(bus), memory.NewMemoryCheckpointStore(), nil)
		if err != nil {
			b.Fatal(err)
		}

		ch, err := catchUp.Subscribe(context.Background(), "bench.catchup")
		if err != nil {
			b.Fatal(err)
		}

		received := 0
		for received < total {
			msg := <-ch
			if msg == nil {
				b.Fatalf("subscription closed after %d of %d events", received, total)
			}

			msg.Ack()
			received++
		}

		if err := catchUp.Close(); err != nil {
			b.Fatal(err)
		}
	}

	perIteration := b.Elapsed().Nanoseconds()
	b.ReportMetric(float64(perIteration)/float64(b.N*total), "ns/event")
}
