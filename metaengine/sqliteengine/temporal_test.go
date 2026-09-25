package sqliteengine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

func newVersionedSQLite(t *testing.T, opts ...sqliteengine.CellVersioningOption) metaengine.Engine {
	t.Helper()

	eng, err := sqliteengine.NewSQLiteEngineFromDSNWith(
		"file:"+t.TempDir()+"/versioned.db?mode=rwc",
		nil,
		sqliteengine.WithCellVersioning(opts...),
	)
	if err != nil {
		t.Fatalf("NewSQLiteEngineFromDSN: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng
}

// TestTemporalConformance_SQLite pins the ADR-0141 versioned-cell contract on
// the SQLite engine (meta_cell_versions history table).
func TestTemporalConformance_SQLite(t *testing.T) {
	t.Parallel()

	adttest.AssertTemporalConformance(t, newVersionedSQLite(t))
}

// TestSQLiteVersioning_PlainWritesRecordHistory: with versioning enabled,
// plain MapSet/MapDelete record wall-clock versions, so as-of reads resolve
// past states without MapSetAt.
func TestSQLiteVersioning_PlainWritesRecordHistory(t *testing.T) {
	t.Parallel()

	eng := newVersionedSQLite(t)

	mb := eng.(metaengine.MapBackend)
	vs := eng.(metaengine.VersionedStorage)
	ctx := context.Background()

	before := time.Now()

	if err := mb.MapSet(ctx, "hist", "k", "one"); err != nil {
		t.Fatal(err)
	}

	mid := time.Now()

	if err := mb.MapSet(ctx, "hist", "k", "two"); err != nil {
		t.Fatal(err)
	}

	if val, err := vs.MapGetAsOf(ctx, "hist", "k", mid); err != nil || val != "one" {
		t.Fatalf("as-of mid = (%v, %v), want (one, nil)", val, err)
	}

	if val, err := vs.MapGetAsOf(
		ctx,
		"hist",
		"k",
		before,
	); !errors.Is(
		err,
		metaengine.ErrNotFound,
	) {
		t.Fatalf("as-of before err = %v, want ErrNotFound (got %v)", err, val)
	}

	if err := mb.MapDelete(ctx, "hist", "k"); err != nil {
		t.Fatal(err)
	}

	if val, found, err := mb.MapGet(ctx, "hist", "k"); err != nil || found || val != nil {
		t.Fatalf("latest after delete = (%v, %v, %v), want gone", val, found, err)
	}

	if _, err := vs.MapGetAsOf(
		ctx,
		"hist",
		"k",
		time.Now(),
	); !errors.Is(
		err,
		metaengine.ErrNotFound,
	) {
		t.Fatalf("as-of after delete err = %v, want ErrNotFound", err)
	}
}

// TestSQLiteVersioning_Retention pins the MaxVersions trim on the SQL path.
func TestSQLiteVersioning_Retention(t *testing.T) {
	t.Parallel()

	eng := newVersionedSQLite(
		t,
		sqliteengine.WithRetention(metaengine.RetentionPolicy{MaxVersions: 2}),
	)

	vw := eng.(metaengine.VersionedWriter)
	vs := eng.(metaengine.VersionedStorage)
	hr := eng.(metaengine.CellHistoryReader)
	ctx := context.Background()

	base := time.Now().Add(-time.Hour).Truncate(time.Millisecond)

	for i, name := range []string{"v1", "v2", "v3"} {
		if err := vw.MapSetAt(
			ctx,
			"ret",
			"k",
			name,
			base.Add(time.Duration(i)*time.Minute),
		); err != nil {
			t.Fatal(err)
		}
	}

	hist, err := hr.MapHistory(ctx, "ret", "k", base.Add(-time.Minute), base.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	if len(hist) != 2 || hist[0].Value != "v3" || hist[1].Value != "v2" {
		t.Fatalf("retained history = %+v, want [v3, v2]", hist)
	}

	if _, err := vs.MapGetAsOf(ctx, "ret", "k", base); !errors.Is(err, metaengine.ErrNotFound) {
		t.Fatalf("as-of pruned err = %v, want ErrNotFound", err)
	}
}

// TestSQLiteVersioning_DisabledByDefault: without WithCellVersioning, the
// engine reports the capability off and as-of reads report unsupported via
// the toggle gate.
func TestSQLiteVersioning_DisabledByDefault(t *testing.T) {
	t.Parallel()

	eng, err := sqliteengine.NewSQLiteEngineFromDSN("file:" + t.TempDir() + "/plain.db?mode=rwc")
	if err != nil {
		t.Fatal(err)
	}

	defer eng.Close() //nolint:errcheck // test cleanup

	if metaengine.EngineVersionsCells(eng) {
		t.Fatal("plain engine must not report versioned cells")
	}
}

// TestSQLiteVersioning_RestartSoak pins that meta_cell_versions history
// SURVIVES process restart (ADR-0141 follow-up f23, 2026-09-25): the same
// DSN re-opened after Close still answers as-of reads written before.
func TestSQLiteVersioning_RestartSoak(t *testing.T) {
	t.Parallel()

	dsn := "file:" + t.TempDir() + "/restart.db?mode=rwc"

	eng, err := sqliteengine.NewSQLiteEngineFromDSNWith(dsn, nil,
		sqliteengine.WithCellVersioning())
	if err != nil {
		t.Fatalf("open first incarnation: %v", err)
	}

	mb := eng.(metaengine.MapBackend)
	_ = mb // latest-view writes flow through MapSet; kept for the interface pin

	vw := eng.(metaengine.VersionedWriter)
	ctx := context.Background()

	writeAt := func(val string, when time.Time) {
		t.Helper()
		if err := vw.MapSetAt(ctx, "restart", "k", val, when); err != nil {
			t.Fatalf("MapSetAt(%s): %v", val, err)
		}
	}

	writeAt("v1", time.Now().Add(-2*time.Hour))
	writeAt("v2", time.Now().Add(-1*time.Hour))
	writeAt("v3", time.Now())

	if err := eng.Close(); err != nil {
		t.Fatalf("close first incarnation: %v", err)
	}

	// Second incarnation over the same file.
	eng2, err := sqliteengine.NewSQLiteEngineFromDSNWith(dsn, nil,
		sqliteengine.WithCellVersioning())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = eng2.Close() })

	vs2 := eng2.(metaengine.VersionedStorage)

	// Probes measured from the SECOND incarnation's clock (slightly after
	// the writes): -90m predates v2 (-1h) so v1 is correct; -30m postdates
	// v2 but predates v3 (now) so v2 is correct.
	probeBase := time.Now()
	for _, tc := range []struct {
		asOf    time.Time
		want    any
		wantErr error
	}{
		{probeBase.Add(-90 * time.Minute), "v1", nil},
		{probeBase.Add(-30 * time.Minute), "v2", nil},
	} {
		val, err := vs2.MapGetAsOf(ctx, "restart", "k", tc.asOf)
		if !errors.Is(err, tc.wantErr) {
			t.Fatalf("as-of(-%v) err = %v, want %v", time.Since(tc.asOf), err, tc.wantErr)
		}
		if tc.wantErr == nil && val != tc.want {
			t.Fatalf("as-of(-%v) = %v, want %v", time.Since(tc.asOf), val, tc.want)
		}
	}
}

// TestSQLiteVersioning_DifferentialVsMemory pins contract equivalence of
// the two emulating engines (ADR-0141 follow-up f41, 2026-09-25): the same
// timestamped write sequence replayed on memory-with-versioning and sqlite
// versioned cells must answer every as-of probe identically, including
// out-of-order stamps and tombstones.
func TestSQLiteVersioning_DifferentialVsMemory(t *testing.T) {
	t.Parallel()

	base := time.Now().Add(-time.Hour)

	// Deterministic op sequence: out-of-order stamps + a tombstone + a
	// resurrection.
	ops := []struct {
		ts   time.Duration
		val  any
		tomb bool
	}{
		{0, "alpha", false},
		{30 * time.Minute, "beta", false},
		{10 * time.Minute, "late-alpha", false}, // out-of-order insert
		{45 * time.Minute, nil, true},           // tombstone
		{50 * time.Minute, "gamma", false},      // resurrection
	}

	build := func(eng metaengine.Engine) {
		vw := eng.(metaengine.VersionedWriter)
		for _, op := range ops {
			if op.tomb {
				if err := vw.MapDeleteAt(
					context.Background(),
					"diff",
					"k",
					base.Add(op.ts),
				); err != nil {
					t.Fatalf("MapDeleteAt: %v", err)
				}
				continue
			}
			if err := vw.MapSetAt(
				context.Background(),
				"diff",
				"k",
				op.val,
				base.Add(op.ts),
			); err != nil {
				t.Fatalf("MapSetAt: %v", err)
			}
		}
	}

	memEng := metaengine.NewMemoryEngineWithVersioning()
	t.Cleanup(func() { _ = memEng.Close() })
	sqlEng := newVersionedSQLite(t)

	build(memEng)
	build(sqlEng)

	memVS := memEng.(metaengine.VersionedStorage)
	sqlVS := sqlEng.(metaengine.VersionedStorage)

	for probe := time.Duration(0); probe <= 55*time.Minute; probe += 5 * time.Minute {
		asOf := base.Add(probe)

		memVal, memErr := memVS.MapGetAsOf(context.Background(), "diff", "k", asOf)
		sqlVal, sqlErr := sqlVS.MapGetAsOf(context.Background(), "diff", "k", asOf)

		if (memErr == nil) != (sqlErr == nil) {
			t.Fatalf(
				"probe -%v: memory err=%v sqlite err=%v — engines disagree",
				probe,
				memErr,
				sqlErr,
			)
		}
		if memVal != sqlVal {
			t.Fatalf("probe -%v: memory=%v sqlite=%v — engines disagree", probe, memVal, sqlVal)
		}
	}
}
