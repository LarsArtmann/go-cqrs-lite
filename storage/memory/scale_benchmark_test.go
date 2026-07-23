package memory_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func benchPopulateStore(b *testing.B, store *memory.MemoryStore, ctx context.Context, n int) {
	b.Helper()

	for range n {
		aggID := id.NewStreamID()
		evt := benchEvent(b, aggID, 1)
		_ = store.AppendBatch(ctx, id.NewStreamRef("Bench", aggID), []event.Event{evt})
	}
}

func benchStoreWithNEvents(b *testing.B, n int) {
	b.Helper()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	benchPopulateStore(b, store, ctx, n)

	b.ResetTimer()

	for b.Loop() {
		_, _ = store.ReadAll(ctx)
	}
}

func BenchmarkMemoryStore_ReadAll_Scale(b *testing.B) {
	sizes := []int{100, 1_000, 10_000, 100_000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("%dEvents", n), func(b *testing.B) {
			b.ReportAllocs()
			benchStoreWithNEvents(b, n)
		})
	}
}

func BenchmarkMemoryStore_ReadFrom_Scale(b *testing.B) {
	b.ReportAllocs()

	sizes := []int{100, 1_000, 10_000, 100_000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("%dEvents", n), func(b *testing.B) {
			store := memory.NewMemoryStore()
			b.Cleanup(func() { _ = store.Close() })

			ctx := context.Background()
			var lastID id.EventID

			for range n {
				aggID := id.NewStreamID()
				evt := benchEvent(b, aggID, 1)
				lastID = evt.ID()
				_ = store.AppendBatch(
					ctx,
					id.NewStreamRef("Bench", aggID),
					[]event.Event{evt},
				)
			}

			b.ResetTimer()

			for b.Loop() {
				_, _ = store.ReadFrom(ctx, lastID, 100)
			}
		})
	}
}

func BenchmarkMemoryStore_Save_Concurrent(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	var wg sync.WaitGroup

	b.ResetTimer()

	for b.Loop() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newID := id.NewStreamID()
			newEvt := benchEvent(b, newID, 1)
			_ = store.Save(ctx, id.NewStreamRef("Bench", newID), []event.Event{newEvt}, 0)
		}()
	}

	wg.Wait()
}

func BenchmarkMemoryStore_ReadWrite_Concurrent(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	benchPopulateStore(b, store, ctx, 100)

	var wg sync.WaitGroup

	b.ResetTimer()

	for b.Loop() {
		wg.Add(2)

		go func() {
			defer wg.Done()
			_, _ = store.ReadAll(ctx)
		}()

		go func() {
			defer wg.Done()
			aggID := id.NewStreamID()
			evt := benchEvent(b, aggID, 1)
			_ = store.Save(ctx, id.NewStreamRef("Bench", aggID), []event.Event{evt}, 0)
		}()
	}

	wg.Wait()
}
