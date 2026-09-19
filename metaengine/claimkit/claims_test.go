package claimkit_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/claimkit"

	_ "modernc.org/sqlite" // test-only driver registration
)

// host adapts the claimkit runtimes to metaengine.Engine so the adttest
// conformance suites can drive them — exactly the shape a real SQL engine
// takes by embedding (constructor + profile, no hand-written claims).
type host struct {
	db *sql.DB

	*claimkit.Claims

	*claimkit.Dedup
}

func (host) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{Name: "claimkit-sqlite"}
}

func (h host) Close() error { return nil }

func newHost(t *testing.T) host {
	t.Helper()

	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:claimkit_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()

	claims, err := claimkit.New(ctx, db, claiming.DialectSQLite)
	if err != nil {
		t.Fatalf("claimkit.New: %v", err)
	}

	dedup, err := claimkit.NewDedup(ctx, db, claiming.DialectSQLite)
	if err != nil {
		t.Fatalf("claimkit.NewDedup: %v", err)
	}

	return host{db: db, Claims: claims, Dedup: dedup}
}

func TestClaimKit_SQLiteConformance(t *testing.T) {
	t.Parallel()

	adttest.AssertDueClaimer(t, []adttest.Factory{
		{Name: "claimkit-sqlite", Create: func(t *testing.T) metaengine.Engine { return newHost(t) }},
	})
	adttest.AssertDedupStore(t, []adttest.Factory{
		{Name: "claimkit-sqlite", Create: func(t *testing.T) metaengine.Engine { return newHost(t) }},
	})
}

func TestClaimKit_FactSinkSameTransaction(t *testing.T) {
	t.Parallel()

	h := newHost(t)
	ctx := context.Background()
	now := time.UnixMilli(1000)

	mustT(t, h.Claims.ClaimInsert(ctx, "tasks", "t1", now.Add(-time.Minute), []byte("w")))
	mustT(t, h.Claims.ClaimInsert(ctx, "tasks", "t2", now.Add(-time.Minute), []byte("w2")))

	claims, err := h.Claims.ClaimDueFacts(ctx, metaengine.ClaimDueRequest{
		Collection: "tasks", Owner: "w1", Lease: time.Minute, Now: now,
	}, func(cl metaengine.DueClaim) []metaengine.ClaimFact {
		return []metaengine.ClaimFact{{Type: "claimed", Payload: []byte(cl.Key)}}
	})
	mustT(t, err)

	if len(claims) != 2 {
		t.Fatalf("claims = %d, want 2 (both due; default limit 512)", len(claims))
	}

	// Epoch-guarded delete with facts: matching epoch deletes + records;
	// stale epoch records NOTHING (a fact without its state change did not
	// happen either).
	ok, err := h.Claims.ClaimDeleteFacts(ctx, "tasks", "t2", now.Add(-time.Minute),
		metaengine.ClaimFact{Type: "completed"})
	mustT(t, err)

	if !ok {
		t.Fatal("matching-epoch delete must succeed")
	}

	ok, err = h.Claims.ClaimDeleteFacts(ctx, "tasks", "t2", now.Add(-time.Minute),
		metaengine.ClaimFact{Type: "completed"})
	mustT(t, err)

	if ok {
		t.Fatal("stale-epoch delete must be a no-op")
	}

	countFacts := func(key string) int {
		var n int

		if err := h.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM meta_claim_facts WHERE collection = ? AND key = ?",
			"tasks", key).Scan(&n); err != nil {
			t.Fatalf("count facts: %v", err)
		}

		return n
	}

	// ClaimDueFacts claimed t1 or t2 in due-key order: c < ... both due_at
	// equal, key tie-break → t1 first. Default limit 512 → both claimed.
	if n := countFacts("t1"); n != 1 {
		t.Fatalf("t1 facts = %d, want 1 (claimed)", n)
	}

	if n := countFacts("t2"); n != 2 {
		t.Fatalf("t2 facts = %d, want 2 (claimed + completed once; stale delete recorded nothing)", n)
	}
}

func mustT(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
