package claimkit_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite" // test-only driver registration

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/claimkit"
)

// T18a (ADR-0142): claimkit vs the direct-SQL path a pre-substrate consumer
// wrote themselves — the same statements composed by hand over database/sql
// (the scheduling/sqlstore pattern for claims; a hand-rolled upsert for
// dedup). The runtime IS the claiming/ SQL core, so the abstraction tax
// should be noise-level; these benches keep that promise measurable.
//
// Shapes are steady-state: a fixed working set re-claimed through advancing
// clocks (never a growing scan), fresh-key inserts, and live-window dedup
// hits. The composite timer round-trip (schedule -> due+fact -> fire) is the
// consumer-pattern number the scheduling/engine facade actually pays.

const (
	benchCollection = "bench"
	benchPayload    = "0123456789012345678901234567890123456789012345678901234567890123" // 64 B
	claimWorkload   = 8                                                                  // rows in the steady claim set
)

func newBenchHost(b *testing.B) host {
	b.Helper()

	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:claimkitbench_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		b.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)

	b.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()

	claims, err := claimkit.New(ctx, db, claiming.DialectSQLite)
	if err != nil {
		b.Fatalf("claimkit.New: %v", err)
	}

	dedup, err := claimkit.NewDedup(ctx, db, claiming.DialectSQLite)
	if err != nil {
		b.Fatalf("claimkit.NewDedup: %v", err)
	}

	return host{db: db, Claims: claims, Dedup: dedup}
}

// benchClaimSpec is the spec a hand-rolled consumer passes to claiming/
// directly — identical to claimkit's internal one for sqlite.
func benchClaimSpec() claiming.Spec {
	return claiming.Spec{
		Table:        "meta_due_claims",
		IDColumn:     "key",
		DueColumn:    "due_at",
		LeaseColumn:  "lease_until",
		OwnerColumn:  "owner",
		FilterColumn: "collection",
		Returning:    []string{"key", "due_at", "lease_until", "payload"},
		OrderBy:      "due_at ASC, key ASC",
	}
}

func seedSteadyClaims(b *testing.B, h host, base time.Time) {
	b.Helper()

	ctx := context.Background()

	for i := range claimWorkload {
		err := h.ClaimInsert(ctx, benchCollection, fmt.Sprintf("row-%02d", i),
			base.Add(-time.Hour), []byte(benchPayload))
		if err != nil {
			b.Fatalf("seed claim: %v", err)
		}
	}
}

func BenchmarkClaimDue_ClaimKit(b *testing.B) {
	h := newBenchHost(b)

	base := time.Now()
	seedSteadyClaims(b, h, base)

	ctx := context.Background()
	clock := base

	for b.Loop() {
		clock = clock.Add(10 * time.Second) // leases (1s) always lapsed by next claim

		got, err := h.ClaimDue(ctx, metaengine.ClaimDueRequest{
			Collection: benchCollection,
			Owner:      "bench",
			Now:        clock,
			Lease:      time.Second,
		})
		if err != nil {
			b.Fatal(err)
		}

		if len(got) != claimWorkload {
			b.Fatalf("claimed %d rows, want %d", len(got), claimWorkload)
		}
	}
}

func BenchmarkClaimDue_DirectSQL(b *testing.B) {
	h := newBenchHost(b)

	base := time.Now()
	seedSteadyClaims(b, h, base)

	spec := benchClaimSpec()
	ctx := context.Background()
	clock := base

	for b.Loop() {
		clock = clock.Add(10 * time.Second)

		const sqliteTime = "2006-01-02T15:04:05.000000000Z07:00"

		query, args := claiming.ClaimStmt(claiming.DialectSQLite, spec, claiming.ClaimParams{
			Now:        clock.Format(sqliteTime),
			LeaseUntil: clock.Add(time.Second).Format(sqliteTime),
			Owner:      "bench",
			Filter:     benchCollection,
			Limit:      metaengine.DefaultClaimLimit,
		})

		rows, err := h.db.QueryContext(ctx, query, args...)
		if err != nil {
			b.Fatal(err)
		}

		n := 0

		for rows.Next() {
			var key, due, lease string

			var payload []byte

			if err := rows.Scan(&key, &due, &lease, &payload); err != nil {
				b.Fatal(err)
			}

			n++
		}

		if err := rows.Err(); err != nil {
			b.Fatal(err)
		}

		_ = rows.Close()

		if n != claimWorkload {
			b.Fatalf("claimed %d rows, want %d", n, claimWorkload)
		}
	}
}

// BenchmarkTimerRoundTrip_ClaimKit is the consumer pattern end to end:
// schedule (idempotent insert), claim with a journaled fact, fire
// (epoch-guarded delete with its fact) — the scheduling/engine facade's
// per-timer cost in one number.
func BenchmarkTimerRoundTrip_ClaimKit(b *testing.B) {
	h := newBenchHost(b)

	ctx := context.Background()
	due := time.Now().Add(-time.Minute)
	var n int

	for b.Loop() {
		key := fmt.Sprintf("timer-%08d", n)
		n++

		if err := h.ClaimInsert(ctx, benchCollection, key, due, []byte(benchPayload)); err != nil {
			b.Fatal(err)
		}

		claimed, err := h.ClaimDueFacts(ctx, metaengine.ClaimDueRequest{
			Collection: benchCollection,
			Owner:      "bench",
			Now:        time.Now(),
			Lease:      time.Second,
			Limit:      1,
		}, func(metaengine.DueClaim) []metaengine.ClaimFact {
			return []metaengine.ClaimFact{{Type: "fired", Payload: []byte(`{"gen":1}`)}}
		})
		if err != nil {
			b.Fatal(err)
		}

		if len(claimed) != 1 {
			b.Fatalf("claimed %d, want 1", len(claimed))
		}

		if _, err := h.ClaimDeleteFacts(ctx, benchCollection, key, due,
			metaengine.ClaimFact{Type: "completed"}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDedupCheckAndRecord_FreshKeys_ClaimKit(b *testing.B) {
	h := newBenchHost(b)

	ctx := context.Background()
	now := time.Now()
	var n int

	for b.Loop() {
		seen, err := h.DedupCheckAndRecord(ctx, benchCollection,
			fmt.Sprintf("cmd-%08d", n), time.Hour, now)
		n++

		if err != nil {
			b.Fatal(err)
		}

		if seen {
			b.Fatal("fresh key reported as seen")
		}
	}
}

func BenchmarkDedupCheckAndRecord_FreshKeys_DirectSQL(b *testing.B) {
	h := newBenchHost(b)

	ctx := context.Background()
	now := time.Now()
	var n int

	for b.Loop() {
		key := fmt.Sprintf("cmd-%08d", n)
		n++

		var returned string

		err := h.db.QueryRowContext(ctx,
			`INSERT INTO meta_dedup (collection, key, expires_at) VALUES (?1, ?2, ?3)
ON CONFLICT (collection, key) DO UPDATE SET expires_at = excluded.expires_at
WHERE meta_dedup.expires_at <= ?4
RETURNING key`,
			benchCollection, key,
			now.Add(time.Hour).Format("2006-01-02T15:04:05.000000000Z07:00"),
			now.Format("2006-01-02T15:04:05.000000000Z07:00")).Scan(&returned)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDedupCheckAndRecord_LiveWindowHit(b *testing.B) {
	h := newBenchHost(b)

	ctx := context.Background()
	now := time.Now()

	if _, err := h.DedupCheckAndRecord(ctx, benchCollection, "hot", 24*time.Hour, now); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		seen, err := h.DedupCheckAndRecord(ctx, benchCollection, "hot", 24*time.Hour, now)
		if err != nil {
			b.Fatal(err)
		}

		if !seen {
			b.Fatal("live window not reported as seen")
		}
	}
}
