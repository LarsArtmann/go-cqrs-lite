package projectionhost_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// replay_bench_test.go — benchmarks for projectionhost replay speed, DLQ
// throughput, and multi-projection parallelism.

// BenchmarkProjectionHost_Replay measures projection catch-up throughput.
// Seeds N events into a journal, starts the host, waits for all events to be
// processed, reports events/sec.
func BenchmarkProjectionHost_Replay(b *testing.B) {
	for _, n := range []int{100, 1_000, 10_000} {
		b.Run(fmt.Sprintf("events=%d", n), func(b *testing.B) {
			b.ResetTimer()

			for range b.N {
				b.StopTimer()
				// Seed journal with n events.
				journal := &memoryJournal{}
				for range n {
					journal.append(makeEvent("bench.created"))
				}

				cpStore := newMemoryCheckpointStore()
				proj := &countingProjection{
					name:       "bench-proj",
					eventTypes: []event.Type{"bench.created"},
				}

				host, err := projectionhost.New(journal, cpStore,
					projectionhost.WithBatchSize(100),
				)
				if err != nil {
					b.Fatal(err)
				}
				if err := host.Register(proj); err != nil {
					b.Fatal(err)
				}

				ctx, cancel := context.WithCancel(context.Background())
				b.StartTimer()

				go func() { _ = host.Start(ctx) }()

				// Wait for all events to be processed.
				deadline := time.Now().Add(30 * time.Second)
				for time.Now().Before(deadline) {
					if proj.count.Load() >= int64(n) {
						break
					}
					time.Sleep(time.Millisecond)
				}

				b.StopTimer()
				cancel()
				_ = host.Stop()

				if proj.count.Load() < int64(n) {
					b.Fatalf("only processed %d/%d events", proj.count.Load(), n)
				}
				b.StartTimer()
			}

			b.ReportMetric(float64(n)*float64(b.N)/b.Elapsed().Seconds(), "events/sec")
		})
	}
}

// BenchmarkProjectionHost_DirectHandle measures synchronous projection.Handle
// throughput — the raw processing cost per event without host polling overhead.
func BenchmarkProjectionHost_DirectHandle(b *testing.B) {
	proj := &countingProjection{
		name:       "direct-proj",
		eventTypes: []event.Type{"bench.created"},
	}
	ctx := context.Background()
	evt := makeEvent("bench.created")

	b.ResetTimer()

	for b.Loop() {
		if err := proj.Handle(ctx, evt); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

// BenchmarkProjectionHost_DLQ measures dead-letter store throughput.
// Inserts, lists, and deletes entries at scale.
func BenchmarkProjectionHost_DLQ(b *testing.B) {
	store := projectionhost.NewMemoryDeadLetterStore()
	ctx := context.Background()

	// Pre-insert entries.
	n := 1_000
	for range n {
		evt := makeEvent("dlq.event")
		_ = store.Store(ctx, projectionhost.DeadLetterEntry{
			ProjectionName: "bench-proj",
			EventID:        evt.ID().String(),
			EventType:      "dlq.event",
			StreamID:       evt.StreamID().String(),
			Event:          evt,
			Error:          "test error",
			FailedAt:       time.Now(),
		})
	}

	b.ResetTimer()

	for b.Loop() {
		// List all entries.
		entries, err := store.List(ctx, "bench-proj")
		if err != nil {
			b.Fatal(err)
		}
		if len(entries) != n {
			b.Fatalf("expected %d entries, got %d", n, len(entries))
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "lists/sec")
}

// BenchmarkProjectionHost_MultiProjection measures combined throughput when
// 3 projections process the same event stream in parallel.
func BenchmarkProjectionHost_MultiProjection(b *testing.B) {
	n := 1_000

	b.ResetTimer()

	for range b.N {
		b.StopTimer()
		journal := &memoryJournal{}
		for range n {
			journal.append(makeEvent("bench.created"))
		}

		cpStore := newMemoryCheckpointStore()

		projs := []*countingProjection{
			{name: "proj-1", eventTypes: []event.Type{"bench.created"}},
			{name: "proj-2", eventTypes: []event.Type{"bench.created"}},
			{name: "proj-3", eventTypes: []event.Type{"bench.created"}},
		}

		host, err := projectionhost.New(journal, cpStore,
			projectionhost.WithBatchSize(100),
		)
		if err != nil {
			b.Fatal(err)
		}
		for _, p := range projs {
			if err := host.Register(p); err != nil {
				b.Fatal(err)
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		b.StartTimer()

		go func() { _ = host.Start(ctx) }()

		// Wait for all 3 projections to process all events.
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			allDone := true
			for _, p := range projs {
				if p.count.Load() < int64(n) {
					allDone = false

					break
				}
			}
			if allDone {
				break
			}
			time.Sleep(time.Millisecond)
		}

		b.StopTimer()
		cancel()
		_ = host.Stop()

		for _, p := range projs {
			if p.count.Load() < int64(n) {
				b.Fatalf("%s only processed %d/%d", p.name, p.count.Load(), n)
			}
		}
		b.StartTimer()
	}

	// Report combined events/sec (3 projections × n events per iteration).
	b.ReportMetric(float64(n*3)*float64(b.N)/b.Elapsed().Seconds(), "total-projections/sec")
}

// suppress unused import.
var _ = atomic.Int64{}
