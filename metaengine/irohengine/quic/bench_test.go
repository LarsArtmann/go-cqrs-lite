//go:build cgo

package quic_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/quic/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func BenchmarkQuicMapSet(b *testing.B) {
	ctx := context.Background()

	tA, err := quic.New(quic.WithLocalOnly())
	if err != nil {
		b.Fatal(err)
	}
	defer tA.Close()

	tB, err := quic.New(quic.WithLocalOnly())
	if err != nil {
		b.Fatal(err)
	}
	defer tB.Close()

	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	defer nodeA.Close()

	ticket, _ := tA.Ticket()
	if err := tB.Connect(ticket); err != nil {
		b.Fatal(err)
	}

	// Wait for connection
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if tA.PeerCount() >= 1 && tB.PeerCount() >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	b.ResetTimer()
	for b.Loop() {
		key := "bench-key"
		val := "bench-value"
		if err := nodeA.(metaengine.MapBackend).MapSet(ctx, "bench", key, val); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInProcessMapSet(b *testing.B) {
	ctx := context.Background()

	net := irohengine.NewInProcessNetwork()
	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(net.Join("a")),
	)
	defer nodeA.Close()

	b.ResetTimer()
	for b.Loop() {
		key := "bench-key"
		val := "bench-value"
		if err := nodeA.(metaengine.MapBackend).MapSet(ctx, "bench", key, val); err != nil {
			b.Fatal(err)
		}
	}
}
