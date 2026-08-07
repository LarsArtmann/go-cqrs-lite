package http_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrshttp "github.com/larsartmann/go-cqrs-lite/transport/http/v4"
)

// sse_fanout_bench_test.go — measures SSE broker fan-out latency.
// Tests how fast events propagate to N connected SSE clients.

// BenchmarkSSE_Fanout measures the time to publish an event and have it
// delivered to N connected clients via the SSE broker.
func BenchmarkSSE_Fanout(b *testing.B) {
	for _, clientCount := range []int{1, 10, 50, 100} {
		b.Run(formatClientCount(clientCount), func(b *testing.B) {
			bus := eventtest.NewFakeBus()
			defer func() { _ = bus.Close() }()

			broker, err := cqrshttp.NewSSEBroker(bus)
			if err != nil {
				b.Fatal(err)
			}
			defer broker.Close()

			// Connect N clients.
			var clients []chan event.Event
			for range clientCount {
				clientID := cqrshttp.SSEClientID(id.NewEventID().String())
				ch := broker.AddClient(clientID)
				clients = append(clients, ch)
			}

			ctx := context.Background()
			streamID := id.NewStreamID()

			evt, err := event.New(
				"bench.event", streamID, "Bench", event.Version(1),
				map[string]any{"data": "test"},
			)
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()

			for b.Loop() {
				// Publish the event through the bus.
				if err := bus.Publish(ctx, evt); err != nil {
					b.Fatal(err)
				}

				// Drain all client channels.
				for _, ch := range clients {
				readLoop:
					for {
						select {
						case <-ch:
							break readLoop
						default:
							break readLoop
						}
					}
				}
			}

			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
		})
	}
}

// BenchmarkSSE_AddRemoveClient measures client connection/disconnection overhead.
func BenchmarkSSE_AddRemoveClient(b *testing.B) {
	bus := eventtest.NewFakeBus()
	defer func() { _ = bus.Close() }()

	broker, err := cqrshttp.NewSSEBroker(bus)
	if err != nil {
		b.Fatal(err)
	}
	defer broker.Close()

	b.ResetTimer()

	for b.Loop() {
		clientID := cqrshttp.SSEClientID(id.NewEventID().String())
		broker.AddClient(clientID)
		broker.RemoveClient(clientID)
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "cycles/sec")
}

func formatClientCount(n int) string {
	if n == 1 {
		return "clients=1"
	}

	return formatInt("clients=", n)
}

func formatInt(prefix string, n int) string {
	if n >= 1000 {
		return prefix + formatThousand(n)
	}

	return prefix + itoa(n)
}

func formatThousand(n int) string {
	k := n / 1000
	rest := n % 1000
	if rest == 0 {
		return itoa(k) + "k"
	}

	return itoa(k) + "k" + itoa(rest)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}

	return string(buf[pos:])
}
