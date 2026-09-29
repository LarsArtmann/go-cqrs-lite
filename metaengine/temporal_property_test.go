package metaengine

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"cmp"

	"pgregory.net/rapid"
)

// Temporal properties over the version-chain contract (ADR-0141 / M18):
// out-of-order + same-ts last-writer-wins collapse, retention never pruning
// the newest entry, and tombstone as-of visibility — all with explicit
// timestamps (deterministic: no wall-clock sleeps inside the draws).

type stampedWrite struct {
	ts      time.Time
	value   any // nil = tombstone
	applOrd int
}

// shuffleInPlace permutes s by drawn sort keys (rapid has no Shuffle API).
func shuffleInPlace[T any](rt *rapid.T, s []T, key func(T) int) []T {
	type keyed struct {
		v   T
		pri int
	}

	keyedSlice := make([]keyed, len(s))
	for i, v := range s {
		keyedSlice[i] = keyed{v: v, pri: key(v) + rapid.IntRange(0, 1<<30).Draw(rt, "prio")}
	}

	slices.SortStableFunc(keyedSlice, func(a, b keyed) int { return cmp.Compare(a.pri, b.pri) })

	for i := range s {
		s[i] = keyedSlice[i].v
	}

	return s
}

// drawEventSequence generates m stamped writes over a base timeline with
// duplicate timestamps (the same-ts collapse feed) and returns them in a
// shuffled application order.
func drawEventSequence(rt *rapid.T, base time.Time) []stampedWrite {
	n := rapid.IntRange(4, 20).Draw(rt, "events")

	writes := make([]stampedWrite, n)
	for i := range writes {
		// Coarse offsets create same-ts collisions; the application order
		// decides the winner at equal ts (last write wins).
		off := rapid.IntRange(0, n).Draw(rt, "offset")
		writes[i] = stampedWrite{
			ts:      base.Add(time.Duration(off) * time.Millisecond),
			value:   int64(i),
			applOrd: i,
		}
	}

	applied := slices.Clone(writes)
	shuffleInPlace(rt, applied, func(stampedWrite) int { return 0 })

	// applOrd is the APPLICATION position: at equal timestamps the engine's
	// insertAt lands later inserts after existing ones, so the last-applied
	// write wins (last-writer-wins at same ts).
	for i := range applied {
		applied[i].applOrd = i
	}

	return applied
}

// expectedAsOf resolves the reference model at t: the value of the latest
// write with ts <= t under last-writer-wins at equal ts (higher applOrd
// wins), nil meaning tombstone/not-yet-written.
func expectedAsOf(writes []stampedWrite, t time.Time) (int64, bool) {
	best := -1

	for i, w := range writes {
		if w.ts.After(t) {
			continue
		}

		if best == -1 ||
			w.ts.After(writes[best].ts) ||
			(!w.ts.Before(writes[best].ts) && w.applOrd > writes[best].applOrd) {
			best = i
		}
	}

	if best == -1 || writes[best].value == nil {
		return 0, false
	}

	return writes[best].value.(int64), true
}

// TestProperty_VersionChain_OutOfOrderAndSameTsLWW: writes applied in
// shuffled order with duplicate timestamps resolve exactly like the reference
// model — as-of reads at every event timestamp and between events match, and
// the latest view (MapGet) equals the model at the maximum timestamp.
func TestProperty_VersionChain_OutOfOrderAndSameTsLWW(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		eng := NewMemoryEngineWithVersioning()
		defer eng.Close()

		vw := eng.(VersionedWriter)
		vs := eng.(VersionedStorage)
		mb := eng.(MapBackend)

		ctx := context.Background()
		base := time.Unix(1_700_000_000, 0).UTC()

		applied := drawEventSequence(rt, base)

		for _, w := range applied {
			var err error

			if w.value == nil {
				err = vw.MapDeleteAt(ctx, "props", "k", w.ts)
			} else {
				err = vw.MapSetAt(ctx, "props", "k", w.value, w.ts)
			}

			if err != nil {
				rt.Fatalf("stamped write: %v", err)
			}
		}

		sorted := slices.Clone(applied)
		slices.SortFunc(sorted, func(a, b stampedWrite) int { return a.ts.Compare(b.ts) })

		// Probe at every event timestamp and the midpoint between adjacent
		// distinct timestamps (boundary: exactly-at and strictly-between).
		probes := make([]time.Time, 0, 2*len(sorted)+1)

		for _, w := range sorted {
			probes = append(probes, w.ts)
		}

		for i := 1; i < len(sorted); i++ {
			if sorted[i].ts != sorted[i-1].ts {
				probes = append(probes, sorted[i-1].ts.Add(sorted[i].ts.Sub(sorted[i-1].ts)/2))
			}
		}

		for _, probe := range probes {
			want, wantExists := expectedAsOf(applied, probe)

			got, err := vs.MapGetAsOf(ctx, "props", "k", probe)

			if !wantExists {
				if !errors.Is(err, ErrNotFound) {
					rt.Fatalf("AsOf(%v): want ErrNotFound, got val=%v err=%v", probe, got, err)
				}

				continue
			}

			if err != nil {
				rt.Fatalf("AsOf(%v): unexpected error %v", probe, err)
			}

			if got != want {
				rt.Fatalf(
					"AsOf(%v): got %v, want %d (same-ts LWW or ordering broken)",
					probe,
					got,
					want,
				)
			}
		}

		// Latest view mirrors the model at the maximum timestamp.
		maxTs := sorted[len(sorted)-1].ts
		want, wantExists := expectedAsOf(applied, maxTs)

		got, ok, err := mb.MapGet(ctx, "props", "k")

		if !wantExists {
			if ok || !errors.Is(err, ErrNotFound) {
				rt.Fatalf(
					"MapGet after tombstone-at-max: want ErrNotFound, got val=%v ok=%v err=%v",
					got,
					ok,
					err,
				)
			}

			return
		}

		if err != nil || !ok || got != want {
			rt.Fatalf("MapGet: got (%v, %v, %v), want %d", got, ok, err, want)
		}
	})
}

// TestProperty_Retention_NeverPrunesNewest: under adversarial retention
// (MaxVersions as low as 1, plus MaxAge windows), the just-written version is
// ALWAYS readable as-of its own timestamp immediately after the write — the
// documented "retention never removes the newest entry" invariant
// (memory_versioned.go trimRetentionLocked).
func TestProperty_Retention_NeverPrunesNewest(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		maxVersions := rapid.IntRange(1, 4).Draw(rt, "maxVersions")
		maxAgeMs := rapid.IntRange(0, 3).Draw(rt, "maxAgeMs")

		policy := RetentionPolicy{MaxVersions: maxVersions}
		if maxAgeMs > 0 {
			policy.MaxAge = time.Duration(maxAgeMs) * time.Millisecond
		}

		eng := NewMemoryEngineWithVersioning(WithRetention(policy))
		defer eng.Close()

		vw := eng.(VersionedWriter)
		vs := eng.(VersionedStorage)

		ctx := context.Background()
		base := time.Unix(1_700_000_000, 0).UTC()

		for i := range 12 {
			ts := base.Add(time.Duration(i*2) * time.Millisecond)
			value := int64(1000 + i)

			if i%5 == 4 { // periodic tombstones
				if err := vw.MapDeleteAt(ctx, "props", "k", ts); err != nil {
					rt.Fatalf("MapDeleteAt: %v", err)
				}
			} else if err := vw.MapSetAt(ctx, "props", "k", value, ts); err != nil {
				rt.Fatalf("MapSetAt: %v", err)
			}

			got, err := vs.MapGetAsOf(ctx, "props", "k", ts)
			if i%5 == 4 {
				if !errors.Is(err, ErrNotFound) {
					rt.Fatalf("write %d: tombstone pruned/forgotten: val=%v err=%v", i, got, err)
				}

				continue
			}

			if err != nil || got != value {
				rt.Fatalf(
					"write %d: newest version not readable as-of its own ts: got (%v, %v), want %d "+
						"(retention pruned the newest entry)",
					i,
					got,
					err,
					value,
				)
			}
		}
	})
}

// TestProperty_Tombstone_AsOfVisibility: set → tombstone → rebirth timelines
// (random gaps, shuffled application) resolve at every boundary: before the
// set NotFound, between set and tombstone the value, between tombstone and
// rebirth NotFound, after rebirth the rebirth value.
func TestProperty_Tombstone_AsOfVisibility(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		eng := NewMemoryEngineWithVersioning()
		defer eng.Close()

		vw := eng.(VersionedWriter)
		vs := eng.(VersionedStorage)

		ctx := context.Background()
		base := time.Unix(1_700_000_000, 0).UTC()

		gap1 := time.Duration(rapid.IntRange(1, 50).Draw(rt, "gap1")) * time.Millisecond
		gap2 := time.Duration(rapid.IntRange(1, 50).Draw(rt, "gap2")) * time.Millisecond

		tSet := base
		tDelete := base.Add(gap1)
		tRebirth := tDelete.Add(gap2)

		events := []struct {
			ts    time.Time
			value any
		}{
			{tSet, int64(1)},
			{tDelete, nil},
			{tRebirth, int64(2)},
		}

		applied := slices.Clone(events)
		shuffleInPlace(rt, applied, func(e struct {
			ts    time.Time
			value any
		},
		) int {
			return 0
		})

		for _, e := range applied {
			var err error

			if e.value == nil {
				err = vw.MapDeleteAt(ctx, "props", "k", e.ts)
			} else {
				err = vw.MapSetAt(ctx, "props", "k", e.value, e.ts)
			}

			if err != nil {
				rt.Fatalf("stamped write: %v", err)
			}
		}

		type phase struct {
			probe    time.Time
			wantVal  int64
			wantGone bool
		}

		ns := time.Nanosecond

		phases := []phase{
			{tSet.Add(-ns), 0, true},               // before first write
			{tSet, 1, false},                       // at set
			{midpoint(tSet, tDelete), 1, false},    // between set and tombstone
			{tDelete.Add(-ns), 1, false},           // just before tombstone
			{tDelete, 0, true},                     // at tombstone
			{midpoint(tDelete, tRebirth), 0, true}, // tombstoned window
			{tRebirth.Add(-ns), 0, true},           // just before rebirth
			{tRebirth, 2, false},                   // at rebirth
			{tRebirth.Add(time.Hour), 2, false},    // well after
		}

		for _, ph := range phases {
			got, err := vs.MapGetAsOf(ctx, "props", "k", ph.probe)

			if ph.wantGone {
				if !errors.Is(err, ErrNotFound) {
					rt.Fatalf(
						"AsOf(%v): want tombstone NotFound, got val=%v err=%v",
						ph.probe,
						got,
						err,
					)
				}

				continue
			}

			if err != nil {
				rt.Fatalf("AsOf(%v): unexpected error %v", ph.probe, err)
			}

			if got != ph.wantVal {
				rt.Fatalf("AsOf(%v): got %v, want %d", ph.probe, got, ph.wantVal)
			}
		}
	})
}

// midpoint returns the temporal midpoint of two strictly ordered timestamps.
func midpoint(a, b time.Time) time.Time {
	return a.Add(b.Sub(a) / 2)
}
