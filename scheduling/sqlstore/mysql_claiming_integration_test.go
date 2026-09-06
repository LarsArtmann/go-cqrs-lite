//go:build integration

package sqlstore_test

// MySQL/MariaDB claiming integration tests. The claim path relies on
// FOR UPDATE SKIP LOCKED (MySQL 8.0+ / MariaDB 10.6+) plus a two-statement
// claim transaction — semantics a syntax check cannot prove. Set
// MYSQL_TEST_DSN, otherwise these skip:
//
//	MYSQL_TEST_DSN="cqrs:cqrs@tcp(127.0.0.1:33061)/cqrs_test?parseTime=true" \
//	  go test -tags "integration goexperiment.jsonv2" ./...
//
// The nix runners export it: nix run .#integration-mysql-nspawn (or -vm).
// SKIP LOCKED behavior was additionally verified live on MariaDB 11.4
// (2026-09-06): a transaction holding row locks does not block a concurrent
// SKIP LOCKED claim of the remaining rows.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// mysqlClaimDSN returns MYSQL_TEST_DSN or skips the test.
func mysqlClaimDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL claiming integration test")
	}

	return dsn
}

// mysqlClaimOpen opens the MySQL connection and drops any leftover timers
// table for per-test isolation. Tests using it must not run in parallel
// with each other (single shared database across the suite).
func mysqlClaimOpen(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("mysql", mysqlClaimDSN(t))
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`DROP TABLE IF EXISTS timers`); err != nil {
		t.Fatalf("drop timers table: %v", err)
	}

	return db
}

// TestClaimingMySQL_TwoClaimersNoDoubleFire mirrors the Postgres core
// guarantee against live MariaDB SKIP LOCKED: 20 concurrent claim rounds
// across two claimers partition the timers — every timer is claimed exactly
// once while its lease is fresh.
func TestClaimingMySQL_TwoClaimersNoDoubleFire(t *testing.T) {
	db := mysqlClaimOpen(t)

	ctx := context.Background()

	store, err := sqlstore.NewClaimingMySQLStore[struct{}](ctx, db, time.Minute)
	if err != nil {
		t.Fatalf("NewClaimingMySQLStore: %v", err)
	}

	now := time.Now().UTC()

	const total = 20

	for i := range total {
		timer := scheduling.Timer[struct{}]{
			ID:     scheduling.MustParseTimerID(fmt.Sprintf("mysql-claim-%02d", i)),
			FireAt: now.Add(-time.Second),
		}

		if err := store.Schedule(ctx, timer); err != nil {
			t.Fatalf("Schedule %d: %v", i, err)
		}
	}

	// Two claimers poll concurrently. SKIP LOCKED (and the row locks the
	// claim UPDATE takes) must partition the timers between them.
	var mu sync.Mutex

	claimed := make([][]string, 2)

	var wg sync.WaitGroup

	for worker := range 2 {
		wg.Add(1)

		go func(worker int) {
			defer wg.Done()

			for range 5 {
				timers, err := store.Due(ctx, now)
				if err != nil {
					t.Errorf("claimer %d Due: %v", worker, err)

					return
				}

				mu.Lock()
				for _, timer := range timers {
					claimed[worker] = append(claimed[worker], timer.ID.String())
				}
				mu.Unlock()
			}
		}(worker)
	}

	wg.Wait()

	seen := make(map[string]int, total)

	for _, ids := range claimed {
		for _, id := range ids {
			seen[id]++
		}
	}

	if len(seen) != total {
		t.Fatalf("%d distinct timers claimed, want %d", len(seen), total)
	}

	for id, count := range seen {
		if count != 1 {
			t.Fatalf("timer %s claimed %d times — double fire", id, count)
		}
	}
}

// TestClaimingMySQL_LeaseExpiryReclaims pins the crash-recovery half on
// live MariaDB: a timer claimed by a crashed dispatcher becomes claimable
// again once the lease expires.
func TestClaimingMySQL_LeaseExpiryReclaims(t *testing.T) {
	db := mysqlClaimOpen(t)

	ctx := context.Background()

	store, err := sqlstore.NewClaimingMySQLStore[struct{}](ctx, db, time.Minute)
	if err != nil {
		t.Fatalf("NewClaimingMySQLStore: %v", err)
	}

	now := time.Now().UTC()
	timer := scheduling.Timer[struct{}]{
		ID:     scheduling.MustParseTimerID("mysql-lease-reclaim"),
		FireAt: now.Add(-time.Second),
	}

	if err := store.Schedule(ctx, timer); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	if _, err := store.Due(ctx, now); err != nil {
		t.Fatalf("Due (crashed claimer): %v", err)
	}

	inLease, err := store.Due(ctx, now.Add(time.Second))
	if err != nil {
		t.Fatalf("Due (within lease): %v", err)
	}

	if len(inLease) != 0 {
		t.Fatalf("timer reclaimed while lease fresh, want fenced")
	}

	after, err := store.Due(ctx, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("Due (after lease): %v", err)
	}

	if len(after) != 1 || after[0].ID != timer.ID {
		t.Fatalf("timer not reclaimed after lease expiry: got %v", after)
	}
}
