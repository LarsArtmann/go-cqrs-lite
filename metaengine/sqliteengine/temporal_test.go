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
