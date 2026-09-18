package adttest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// AssertTemporalConformance pins the versioned-cell contract (ADR-0141 §1) on
// one engine instance: as-of resolution, timestamped tombstones, out-of-order
// writes, same-timestamp last-writer-wins, history ranges, and latest-view
// consistency. Run it from every engine module whose engine claims temporal
// capabilities — divergence here is a split-brain class (same pattern as
// AssertVectorDimensionGuard).
//
// Timestamps are spaced 10ms apart: engines may truncate cell timestamps to
// coarser granularity (BigTable is millisecond-only), so the contract is
// asserted at engine-safe distances. Engine-specific truncation behavior is
// pinned in-engine, not here.
func AssertTemporalConformance(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	if !metaengine.EngineVersionsCells(eng) {
		t.Fatal("engine does not record cell versions (VersionedWriter + toggle)")
	}

	vs, ok := eng.(metaengine.VersionedStorage)
	if !ok {
		t.Fatal("engine has VersionedWriter but not VersionedStorage")
	}

	vw, ok := eng.(metaengine.VersionedWriter)
	if !ok {
		t.Fatal("engine has VersionedStorage but not VersionedWriter")
	}

	ctx := context.Background()
	col := fmt.Sprintf("temporal_conf_%d", time.Now().UnixNano())

	base := time.Now().Truncate(time.Millisecond)
	t1, t2, t3, t4 := base, base.Add(10*time.Millisecond),
		base.Add(20*time.Millisecond), base.Add(30*time.Millisecond)

	assertAsOfResolution(t, ctx, vs, vw, col, t1, t2)
	assertTombstoneSemantics(t, ctx, vs, vw, col, t1, t2, t3)
	assertOutOfOrderWrites(t, ctx, vs, vw, col, t2, t3)
	assertSameTimestampLastWriteWins(t, ctx, vs, vw, col, t4)
	assertHistoryRange(t, ctx, vs, vw, col, t1, t4)
	assertLatestViewConsistency(t, ctx, vs, vw, col, t1, t2)
	assertVersionedUpdater(t, ctx, eng, col, t4)
}

func assertAsOfResolution(
	t *testing.T,
	ctx context.Context,
	vs metaengine.VersionedStorage,
	vw metaengine.VersionedWriter,
	col string, t1, t2 time.Time,
) {
	t.Helper()

	if err := vw.MapSetAt(ctx, col, "k", "v1", t1); err != nil {
		t.Fatalf("MapSetAt v1: %v", err)
	}

	if err := vw.MapSetAt(ctx, col, "k", "v2", t2); err != nil {
		t.Fatalf("MapSetAt v2: %v", err)
	}

	assertAsOf(t, ctx, vs, col, "k", t1, "v1")
	assertAsOf(t, ctx, vs, col, "k", t1.Add(5*time.Millisecond), "v1")
	assertAsOf(t, ctx, vs, col, "k", t2, "v2")

	if _, err := vs.MapGetAsOf(ctx, col, "k", t1.Add(-time.Millisecond)); !errors.Is(err, metaengine.ErrNotFound) {
		t.Fatalf("as-of before creation err = %v, want ErrNotFound", err)
	}

	if ok, _ := vs.MapExistsAsOf(ctx, col, "k", t1.Add(-time.Millisecond)); ok {
		t.Fatal("exists-as-of before creation = true, want false")
	}
}

func assertTombstoneSemantics(
	t *testing.T,
	ctx context.Context,
	vs metaengine.VersionedStorage,
	vw metaengine.VersionedWriter,
	col string, t1, t2, t3 time.Time,
) {
	t.Helper()

	if err := vw.MapDeleteAt(ctx, col, "k", t3); err != nil {
		t.Fatalf("MapDeleteAt: %v", err)
	}

	if _, err := vs.MapGetAsOf(ctx, col, "k", t3); !errors.Is(err, metaengine.ErrNotFound) {
		t.Fatalf("as-of after tombstone err = %v, want ErrNotFound", err)
	}

	if ok, _ := vs.MapExistsAsOf(ctx, col, "k", t3); ok {
		t.Fatal("exists-as-of after tombstone = true, want false")
	}

	assertAsOf(t, ctx, vs, col, "k", t2, "v2")
}

func assertOutOfOrderWrites(
	t *testing.T,
	ctx context.Context,
	vs metaengine.VersionedStorage,
	vw metaengine.VersionedWriter,
	col string, t2, t3 time.Time,
) {
	t.Helper()

	if err := vw.MapSetAt(ctx, col, "ooo", "newer", t3); err != nil {
		t.Fatalf("MapSetAt newer: %v", err)
	}

	if err := vw.MapSetAt(ctx, col, "ooo", "older", t2); err != nil {
		t.Fatalf("MapSetAt older (late arrival): %v", err)
	}

	assertAsOf(t, ctx, vs, col, "ooo", t2, "older")
	assertAsOf(t, ctx, vs, col, "ooo", t3, "newer")
}

func assertSameTimestampLastWriteWins(
	t *testing.T,
	ctx context.Context,
	vs metaengine.VersionedStorage,
	vw metaengine.VersionedWriter,
	col string, t4 time.Time,
) {
	t.Helper()

	if err := vw.MapSetAt(ctx, col, "lww", "first", t4); err != nil {
		t.Fatalf("MapSetAt first: %v", err)
	}

	if err := vw.MapSetAt(ctx, col, "lww", "second", t4); err != nil {
		t.Fatalf("MapSetAt second (same ts): %v", err)
	}

	assertAsOf(t, ctx, vs, col, "lww", t4, "second")
}

func assertHistoryRange(
	t *testing.T,
	ctx context.Context,
	vs metaengine.VersionedStorage,
	vw metaengine.VersionedWriter,
	col string, t1, t4 time.Time,
) {
	t.Helper()

	hr, ok := eng2History(vs)
	if !ok {
		return // CellHistoryReader optional in v4.x
	}

	hist, err := hr.MapHistory(ctx, col, "k", t1.Add(-time.Second), t4.Add(time.Second))
	if err != nil {
		t.Fatalf("MapHistory: %v", err)
	}

	if len(hist) < 3 {
		t.Fatalf("history length = %d, want >= 3 (tombstone + 2 versions)", len(hist))
	}

	if hist[0].Value != nil {
		t.Fatalf("newest history entry = %+v, want tombstone (nil value)", hist[0])
	}

	if !hist[0].Timestamp.After(hist[1].Timestamp) {
		t.Fatal("history not newest-first")
	}
}

func assertLatestViewConsistency(
	t *testing.T,
	ctx context.Context,
	_ metaengine.VersionedStorage,
	vw metaengine.VersionedWriter,
	col string, _, _ time.Time,
) {
	t.Helper()

	mb, ok := vw.(metaengine.MapBackend)
	if !ok {
		return // latest reads via MapBackend optional for pure-temporal engines
	}

	val, found, err := mb.MapGet(ctx, col, "fresh")
	if err != nil {
		t.Fatalf("MapGet fresh: %v", err)
	}

	if found || val != nil {
		t.Fatalf("MapGet fresh before write = (%v, %v), want not-found", val, found)
	}

	now := time.Now().Truncate(time.Millisecond)
	if err := vw.MapSetAt(ctx, col, "fresh", "latest", now); err != nil {
		t.Fatalf("MapSetAt fresh: %v", err)
	}

	val, found, err = mb.MapGet(ctx, col, "fresh")
	if err != nil || !found || val != "latest" {
		t.Fatalf("MapGet fresh after write = (%v, %v, %v), want latest", val, found, err)
	}}

func assertVersionedUpdater(
	t *testing.T,
	ctx context.Context,
	eng metaengine.Engine,
	col string, t4 time.Time,
) {
	t.Helper()

	vu, ok := eng.(metaengine.VersionedUpdater)
	if !ok {
		return // atomic timestamped RMW optional
	}

	vs := eng.(metaengine.VersionedStorage)
	ts := t4.Add(40 * time.Millisecond)

	if err := vu.MapUpdateAt(ctx, col, "rmw", func(prev any) any {
		if prev != nil {
			t.Fatalf("prev = %v, want nil on first update", prev)
		}

		return "one"
	}, ts); err != nil {
		t.Fatalf("MapUpdateAt one: %v", err)
	}

	if err := vu.MapUpdateAt(ctx, col, "rmw", func(prev any) any {
		if prev != "one" {
			t.Fatalf("prev = %v, want \"one\"", prev)
		}

		return "two"
	}, ts.Add(10*time.Millisecond)); err != nil {
		t.Fatalf("MapUpdateAt two: %v", err)
	}

	assertAsOf(t, ctx, vs, col, "rmw", ts, "one")
}

func assertAsOf(
	t *testing.T,
	ctx context.Context,
	vs metaengine.VersionedStorage,
	col, key string,
	at time.Time,
	want any,
) {
	t.Helper()

	val, err := vs.MapGetAsOf(ctx, col, key, at)
	if err != nil {
		t.Fatalf("MapGetAsOf(%s@%v): %v", key, at, err)
	}

	if val != want {
		t.Fatalf("MapGetAsOf(%s@%v) = %v, want %v", key, at, val, want)
	}
}

func eng2History(vs metaengine.VersionedStorage) (metaengine.CellHistoryReader, bool) {
	hr, ok := vs.(metaengine.CellHistoryReader)

	return hr, ok
}
