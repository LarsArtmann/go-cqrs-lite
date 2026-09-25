package bigtableengine_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/bigtable/bttest"
	bigtableengine "github.com/larsartmann/go-cqrs-lite/metaengine/bigtableengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// newFakeEngine starts an in-process bttest fake and returns an engine
// bound to it. The fake is a rough BigTable approximation, but it implements
// the cell-version semantics this engine depends on.
func newFakeEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	srv, err := bttest.NewServer("localhost:0")
	if err != nil {
		t.Fatalf("bttest.NewServer: %v", err)
	}

	t.Cleanup(srv.Close)

	conn, err := grpc.NewClient(srv.Addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}

	t.Cleanup(func() { _ = conn.Close() })

	ctx := context.Background()

	admin, err := bigtable.NewAdminClient(ctx, "proj", "inst", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("NewAdminClient: %v", err)
	}

	client, err := bigtable.NewClient(ctx, "proj", "inst", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	eng, err := bigtableengine.NewWithClients(
		admin,
		client,
		fmt.Sprintf("t_%d", time.Now().UnixNano()),
	)
	if err != nil {
		t.Fatalf("NewWithClients: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng
}

// TestTemporalConformance_Bigtable pins the ADR-0141 versioned-cell contract
// on the native BigTable implementation (bttest fake).
func TestTemporalConformance_Bigtable(t *testing.T) {
	t.Parallel()

	adttest.AssertTemporalConformance(t, newFakeEngine(t))
}

// TestBigtable_MillisecondGranularity: BigTable truncates cell timestamps to
// milliseconds — two writes to one cell inside the same millisecond collapse
// last-writer-wins (documented ADR-0141 §1), and as-of bounds resolve at
// millisecond resolution.
func TestBigtable_MillisecondGranularity(t *testing.T) {
	t.Parallel()

	eng := newFakeEngine(t)

	vw := eng.(metaengine.VersionedWriter)
	vs := eng.(metaengine.VersionedStorage)
	ctx := context.Background()

	ms := time.Now().Truncate(time.Millisecond)

	if err := vw.MapSetAt(ctx, "g", "k", "first", ms); err != nil {
		t.Fatal(err)
	}

	if err := vw.MapSetAt(ctx, "g", "k", "second", ms.Add(time.Microsecond)); err != nil {
		t.Fatal(err)
	}

	if val, err := vs.MapGetAsOf(
		ctx,
		"g",
		"k",
		ms.Add(time.Millisecond),
	); err != nil ||
		val != "second" {
		t.Fatalf("same-ms as-of = (%v, %v), want (second, nil) — last write wins", val, err)
	}

	hr := eng.(metaengine.CellHistoryReader)

	hist, err := hr.MapHistory(ctx, "g", "k", ms.Add(-time.Second), ms.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	if len(hist) != 1 {
		t.Fatalf("same-ms history length = %d, want 1 (collapsed)", len(hist))
	}
}

// TestBigtable_Counters: ReadModifyWrite increments and prefix-scan CounterGet.
func TestBigtable_Counters(t *testing.T) {
	t.Parallel()

	eng := newFakeEngine(t)

	cb := eng.(metaengine.CounterBackend)
	ctx := context.Background()

	if err := cb.CounterIncrement(
		ctx,
		"counts",
		metaengine.Delta{"open": 2, "done": 1},
	); err != nil {
		t.Fatal(err)
	}

	if err := cb.CounterIncrement(ctx, "counts", metaengine.Delta{"open": 3}); err != nil {
		t.Fatal(err)
	}

	got, err := cb.CounterGet(ctx, "counts")
	if err != nil {
		t.Fatal(err)
	}

	if got["open"] != 5 || got["done"] != 1 {
		t.Fatalf("counters = %v, want open=5 done=1", got)
	}
}

// TestBigtable_ResetWipesAllRows: ResetEngine deletes every row (map + counter).
func TestBigtable_ResetWipesAllRows(t *testing.T) {
	t.Parallel()

	eng := newFakeEngine(t)

	mb := eng.(metaengine.MapBackend)
	cb := eng.(metaengine.CounterBackend)
	ctx := context.Background()

	if err := mb.MapSet(ctx, "r", "k", "v"); err != nil {
		t.Fatal(err)
	}

	if err := cb.CounterIncrement(ctx, "rc", metaengine.Delta{"n": 1}); err != nil {
		t.Fatal(err)
	}

	if err := eng.(interface {
		ResetEngine(ctx context.Context) error
	}).ResetEngine(ctx); err != nil {
		t.Fatal(err)
	}

	if _, found, err := mb.MapGet(ctx, "r", "k"); err != nil || found {
		t.Fatalf("map after reset = (%v, %v), want gone", found, err)
	}

	counts, err := cb.CounterGet(ctx, "rc")
	if err != nil || len(counts) != 0 {
		t.Fatalf("counters after reset = (%v, %v), want empty", counts, err)
	}
}

// TestBigtable_DriverRegistration: the DSN driver factory parses
// "project/instance/table" and rejects malformed DSNs loudly.
func TestBigtable_DriverRegistration(t *testing.T) {
	t.Parallel()

	factory, err := metaengine.LookupDriver("bigtable")
	if err != nil {
		t.Fatalf("bigtable driver not registered: %v", err)
	}

	if _, err := factory(
		context.Background(),
		metaengine.DriverConfig{DSN: "not-a-dsn"},
	); err == nil {
		t.Fatal("malformed DSN must fail loudly")
	}

	if _, err := factory(context.Background(), metaengine.DriverConfig{
		DSN:        "p/i/t",
		Durability: "strict",
	}); err == nil {
		t.Fatal("durability tier must be rejected (engine applies its own)")
	}
}

// TestBigtable_TombstoneThenRebirth: deleting and re-writing a key keeps the
// full version history — the tombstone is a version, not an erase.
func TestBigtable_TombstoneThenRebirth(t *testing.T) {
	t.Parallel()

	eng := newFakeEngine(t)

	vw := eng.(metaengine.VersionedWriter)
	vs := eng.(metaengine.VersionedStorage)
	ctx := context.Background()

	base := time.Now().Add(-time.Hour).Truncate(time.Millisecond)

	if err := vw.MapSetAt(ctx, "tb", "k", "alive", base); err != nil {
		t.Fatal(err)
	}

	if err := vw.MapDeleteAt(ctx, "tb", "k", base.Add(10*time.Millisecond)); err != nil {
		t.Fatal(err)
	}

	if err := vw.MapSetAt(ctx, "tb", "k", "reborn", base.Add(20*time.Millisecond)); err != nil {
		t.Fatal(err)
	}

	if _, err := vs.MapGetAsOf(
		ctx,
		"tb",
		"k",
		base.Add(15*time.Millisecond),
	); !errors.Is(
		err,
		metaengine.ErrNotFound,
	) {
		t.Fatalf("as-of during tombstone err = %v, want ErrNotFound", err)
	}

	if val, err := vs.MapGetAsOf(
		ctx,
		"tb",
		"k",
		base.Add(25*time.Millisecond),
	); err != nil ||
		val != "reborn" {
		t.Fatalf("as-of reborn = (%v, %v), want (reborn, nil)", val, err)
	}
}

// TestBigtable_RestartSafety pins ADR-0141 follow-up f24 (2026-09-25): two
// engines over ONE bttest server must not bleed state across instances, and
// a second engine over the same table re-reads the first engine's writes
// cleanly (the restart-shape contract: no cross-instance contamination, no
// stale caching).
func TestBigtable_RestartSafety(t *testing.T) {
	t.Parallel()

	srv, err := bttest.NewServer("localhost:0")
	if err != nil {
		t.Fatalf("bttest.NewServer: %v", err)
	}
	t.Cleanup(srv.Close)

	conn, err := grpc.NewClient(srv.Addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ctx := context.Background()

	newEngine := func(table string) metaengine.Engine {
		t.Helper()

		admin, err := bigtable.NewAdminClient(ctx, "proj", "inst", option.WithGRPCConn(conn))
		if err != nil {
			t.Fatalf("NewAdminClient: %v", err)
		}

		client, err := bigtable.NewClient(ctx, "proj", "inst", option.WithGRPCConn(conn))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}

		eng, err := bigtableengine.NewWithClients(admin, client, table)
		if err != nil {
			t.Fatalf("NewWithClients: %v", err)
		}
		t.Cleanup(func() { _ = eng.Close() })

		return eng
	}

	sharedTable := fmt.Sprintf("restart_%d", time.Now().UnixNano())

	first := newEngine(sharedTable)
	second := newEngine(fmt.Sprintf("other_%d", time.Now().UnixNano()))

	mb1 := first.(metaengine.MapBackend)
	mb2 := second.(metaengine.MapBackend)

	// Distinct tables: the second engine must NOT see the first's writes.
	if err := mb1.MapSet(ctx, "isolation", "only-first", "sentinel"); err != nil {
		t.Fatal(err)
	}

	if _, found, err := mb2.MapGet(ctx, "isolation", "only-first"); err != nil || found {
		t.Fatalf("cross-instance bleed: found=%v err=%v — separate tables must be isolated", found, err)
	}

	// Same table re-opened (the restart shape): a third engine over the
	// FIRST's table re-reads the sentinel cleanly.
	third := newEngine(sharedTable)
	mb3 := third.(metaengine.MapBackend)

	val, found, err := mb3.MapGet(ctx, "isolation", "only-first")
	if err != nil || !found || val != "sentinel" {
		t.Fatalf("re-open over same table: val=%v found=%v err=%v — want sentinel", val, found, err)
	}
}
