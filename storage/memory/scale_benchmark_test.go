package memory_test

import (
	"context"
	"errors"
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
		streamID := id.NewStreamID()
		evt := benchEvent(b, streamID, 1)
		if err := store.AppendBatch(
			ctx,
			id.NewStreamRef("Bench", streamID),
			[]event.Event{evt},
		); err != nil {
			b.Fatalf("seed AppendBatch: %v", err)
		}
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
		events, err := store.ReadAll(ctx)
		if err != nil {
			b.Fatalf("ReadAll: %v", err)
		}
		if len(events) == 0 {
			b.Fatal("ReadAll returned empty — store not populated")
		}
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
			var firstID id.EventID

			for range n {
				streamID := id.NewStreamID()
				evt := benchEvent(b, streamID, 1)
				if firstID == (id.EventID{}) {
					firstID = evt.ID()
				}
				if err := store.AppendBatch(
					ctx,
					id.NewStreamRef("Bench", streamID),
					[]event.Event{evt},
				); err != nil {
					b.Fatalf("seed AppendBatch: %v", err)
				}
			}

			b.ResetTimer()

			for b.Loop() {
				events, err := store.ReadFrom(ctx, firstID, 100)
				if err != nil {
					b.Fatalf("ReadFrom: %v", err)
				}
				if len(events) == 0 {
					b.Fatal("ReadFrom returned empty — store not populated")
				}
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
	var firstErr error
	var errOnce sync.Once

	b.ResetTimer()

	for b.Loop() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newID := id.NewStreamID()
			newEvt := benchEvent(b, newID, 1)
			if err := store.Save(
				ctx,
				id.NewStreamRef("Bench", newID),
				[]event.Event{newEvt},
				0,
			); err != nil {
				errOnce.Do(func() { firstErr = err })
			}
		}()
	}

	wg.Wait()

	if firstErr != nil {
		b.Fatalf("Save: %v", firstErr)
	}

	// Verify data was written.
	events, err := store.ReadAll(ctx)
	if err != nil {
		b.Fatalf("verify ReadAll: %v", err)
	}
	if len(events) == 0 {
		b.Fatal("verify ReadAll: empty — Save was a no-op")
	}
}

func BenchmarkMemoryStore_ReadWrite_Concurrent(b *testing.B) {
	b.ReportAllocs()

	store := memory.NewMemoryStore()
	b.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	benchPopulateStore(b, store, ctx, 100)

	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	b.ResetTimer()

	for b.Loop() {
		wg.Add(2)

		go func() {
			defer wg.Done()
			events, err := store.ReadAll(ctx)
			if err != nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("ReadAll: %w", err) })
			}
			if len(events) == 0 {
				errOnce.Do(func() { firstErr = errors.New("ReadAll returned empty") })
			}
		}()

		go func() {
			defer wg.Done()
			streamID := id.NewStreamID()
			evt := benchEvent(b, streamID, 1)
			if err := store.Save(
				ctx,
				id.NewStreamRef("Bench", streamID),
				[]event.Event{evt},
				0,
			); err != nil {
				errOnce.Do(func() { firstErr = err })
			}
		}()
	}

	wg.Wait()

	if firstErr != nil {
		b.Fatalf("Save: %v", firstErr)
	}
}
