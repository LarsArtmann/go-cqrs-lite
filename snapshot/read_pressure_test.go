package snapshot_test

import (
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

func TestNewReadPressure_ReturnsErrorForZero(t *testing.T) {
	t.Parallel()

	_, err := snapshot.NewReadPressure(0)
	if err == nil {
		t.Error("NewReadPressure(0) should return error")
	}
}

func TestNewReadPressure_ReturnsErrorForNegative(t *testing.T) {
	t.Parallel()

	_, err := snapshot.NewReadPressure(-5)
	if err == nil {
		t.Error("NewReadPressure(-5) should return error")
	}
}

func TestNewReadPressure_Success(t *testing.T) {
	t.Parallel()

	rp, err := snapshot.NewReadPressure(10)
	if err != nil {
		t.Fatalf("NewReadPressure(10) err = %v", err)
	}

	if rp == nil {
		t.Fatal("expected non-nil ReadPressure")
	}
}

func TestReadPressure_ShouldSnapshot_NoReads(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 5)
	ref := id.NewAggregateRef("User", id.NewAggregateID())

	got := rp.ShouldSnapshotFor(ref, event.Version(1))
	if got {
		t.Error("ShouldSnapshotFor should return false with zero reads")
	}
}

func TestReadPressure_ShouldSnapshot_BelowThreshold(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 5)
	ref := id.NewAggregateRef("User", id.NewAggregateID())

	for range 4 {
		rp.RecordRead(ref, event.Version(1))
	}

	got := rp.ShouldSnapshotFor(ref, event.Version(1))
	if got {
		t.Error("ShouldSnapshotFor should return false below threshold")
	}
}

func TestReadPressure_ShouldSnapshot_AtThreshold(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 5)
	ref := id.NewAggregateRef("User", id.NewAggregateID())

	for range 5 {
		rp.RecordRead(ref, event.Version(1))
	}

	got := rp.ShouldSnapshotFor(ref, event.Version(1))
	if !got {
		t.Error("ShouldSnapshotFor should return true at threshold")
	}
}

func TestReadPressure_ShouldSnapshot_AboveThreshold(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 3)
	ref := id.NewAggregateRef("User", id.NewAggregateID())

	for range 10 {
		rp.RecordRead(ref, event.Version(1))
	}

	got := rp.ShouldSnapshotFor(ref, event.Version(1))
	if !got {
		t.Error("ShouldSnapshotFor should return true above threshold")
	}
}

func TestReadPressure_ResetsAfterSnapshot(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 3)
	ref := id.NewAggregateRef("User", id.NewAggregateID())

	for range 3 {
		rp.RecordRead(ref, event.Version(1))
	}

	got := rp.ShouldSnapshotFor(ref, event.Version(1))
	if !got {
		t.Fatal("expected snapshot at threshold")
	}

	if rp.ReadCount(ref) != 0 {
		t.Errorf("expected read count 0 after snapshot, got %d", rp.ReadCount(ref))
	}

	got = rp.ShouldSnapshotFor(ref, event.Version(1))
	if got {
		t.Error("ShouldSnapshotFor should return false after reset")
	}
}

func TestReadPressure_PerAggregateIsolation(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 3)
	ref1 := id.NewAggregateRef("User", id.NewAggregateID())
	ref2 := id.NewAggregateRef("User", id.NewAggregateID())

	for range 3 {
		rp.RecordRead(ref1, event.Version(1))
	}

	if rp.ShouldSnapshotFor(ref2, event.Version(1)) {
		t.Error("ref2 should not trigger — reads were for ref1")
	}

	if !rp.ShouldSnapshotFor(ref1, event.Version(1)) {
		t.Error("ref1 should trigger — has 3 reads")
	}
}

func TestReadPressure_ShouldSnapshot_DelegatesToInner(t *testing.T) {
	t.Parallel()

	inner := everyN(t, 10)
	rp := mustReadPressureWithInner(t, 100, inner)

	ref := id.NewAggregateRef("User", id.NewAggregateID())

	// No reads recorded, but inner strategy fires at version 10
	got := rp.ShouldSnapshotFor(ref, event.Version(10))
	if !got {
		t.Error("inner strategy should trigger at version 10")
	}

	got = rp.ShouldSnapshotFor(ref, event.Version(9))
	if got {
		t.Error("inner strategy should not trigger at version 9")
	}
}

func TestReadPressure_ShouldSnapshot_NoInnerReturnsFalse(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 5)

	// Plain ShouldSnapshot (no ref) cannot evaluate read pressure
	got := rp.ShouldSnapshot("User", event.Version(10))
	if got {
		t.Error("ShouldSnapshot without ref should return false when no inner")
	}
}

func TestReadPressure_ShouldSnapshot_WithInnerDelegates(t *testing.T) {
	t.Parallel()

	inner := everyN(t, 10)
	rp := mustReadPressureWithInner(t, 100, inner)

	got := rp.ShouldSnapshot("User", event.Version(10))
	if !got {
		t.Error("ShouldSnapshot should delegate to inner")
	}
}

func TestReadPressure_ReadCount(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 10)
	ref := id.NewAggregateRef("User", id.NewAggregateID())

	if rp.ReadCount(ref) != 0 {
		t.Fatalf("initial read count = %d, want 0", rp.ReadCount(ref))
	}

	rp.RecordRead(ref, event.Version(1))
	rp.RecordRead(ref, event.Version(1))

	if rp.ReadCount(ref) != 2 {
		t.Errorf("read count = %d, want 2", rp.ReadCount(ref))
	}
}

func TestReadPressure_ConcurrentReads(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 1000)
	ref := id.NewAggregateRef("User", id.NewAggregateID())

	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range 10 {
				rp.RecordRead(ref, event.Version(1))
			}
		}()
	}

	wg.Wait()

	if rp.ReadCount(ref) != 1000 {
		t.Errorf("read count = %d, want 1000", rp.ReadCount(ref))
	}
}

func TestReadPressure_ConcurrentReadAndSnapshot(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 10)
	ref := id.NewAggregateRef("User", id.NewAggregateID())

	var wg sync.WaitGroup

	for range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range 10 {
				rp.RecordRead(ref, event.Version(1))
			}
		}()
	}

	for range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_ = rp.ShouldSnapshotFor(ref, event.Version(1))
		}()
	}

	wg.Wait()
}

func TestShouldSnapshotFor_ReadPressure(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 3)
	sink := newFakeStore()
	t.Cleanup(func() { _ = sink.Close() })

	ref := id.NewAggregateRef("User", id.NewAggregateID())

	// No reads → false
	if snapshot.ShouldSnapshotFor(rp, sink, codec.JSONCodec{}, ref, event.Version(1)) {
		t.Error("expected false with zero reads")
	}

	for range 3 {
		rp.RecordRead(ref, event.Version(1))
	}

	// 3 reads → true
	if !snapshot.ShouldSnapshotFor(rp, sink, codec.JSONCodec{}, ref, event.Version(1)) {
		t.Error("expected true at threshold")
	}
}

func TestShouldSnapshotFor_FallsBackToShouldSnapshot(t *testing.T) {
	t.Parallel()

	inner := everyN(t, 10)
	sink := newFakeStore()
	t.Cleanup(func() { _ = sink.Close() })

	ref := id.NewAggregateRef("User", id.NewAggregateID())

	// EveryNEvents does not implement AggregateAwareStrategy,
	// so ShouldSnapshotFor falls back to ShouldSnapshot.
	got := snapshot.ShouldSnapshotFor(inner, sink, codec.JSONCodec{}, ref, event.Version(10))
	if !got {
		t.Error("fallback ShouldSnapshot should return true at version 10")
	}
}

func TestShouldSnapshotFor_NilGuards(t *testing.T) {
	t.Parallel()

	rp := mustReadPressure(t, 1)
	ref := id.NewAggregateRef("User", id.NewAggregateID())
	rp.RecordRead(ref, event.Version(1))
	cdc := codec.JSONCodec{}

	t.Run("nil strategy", func(t *testing.T) {
		t.Parallel()

		sink := newFakeStore()
		t.Cleanup(func() { _ = sink.Close() })

		if snapshot.ShouldSnapshotFor(nil, sink, cdc, ref, event.Version(1)) {
			t.Error("expected false with nil strategy")
		}
	})

	t.Run("nil sink", func(t *testing.T) {
		t.Parallel()

		if snapshot.ShouldSnapshotFor(rp, nil, cdc, ref, event.Version(1)) {
			t.Error("expected false with nil sink")
		}
	})

	t.Run("nil codec", func(t *testing.T) {
		t.Parallel()

		sink := newFakeStore()
		t.Cleanup(func() { _ = sink.Close() })

		if snapshot.ShouldSnapshotFor(rp, sink, nil, ref, event.Version(1)) {
			t.Error("expected false with nil codec")
		}
	})
}

func mustReadPressure(tb testing.TB, threshold int) *snapshot.ReadPressure {
	tb.Helper()

	rp, err := snapshot.NewReadPressure(threshold)
	if err != nil {
		tb.Fatalf("NewReadPressure(%d): %v", threshold, err)
	}

	return rp
}

func mustReadPressureWithInner(
	tb testing.TB,
	threshold int,
	inner snapshot.SnapshotStrategy,
) *snapshot.ReadPressure {
	tb.Helper()

	rp, err := snapshot.NewReadPressure(threshold, snapshot.WithInnerStrategy(inner))
	if err != nil {
		tb.Fatalf("NewReadPressure(%d, inner): %v", threshold, err)
	}

	return rp
}
