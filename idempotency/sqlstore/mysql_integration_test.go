//go:build integration

package sqlstore_test

// MySQL/MariaDB integration tests for the idempotency sqlstore. The MySQL
// conditional claim is the most dialect-sensitive code in the module
// (ON DUPLICATE KEY UPDATE + IF() + VALUES(), where a no-op update must report
// 0 affected rows so it maps to ErrDuplicate) — a syntax check cannot prove
// those semantics. Set MYSQL_TEST_DSN, otherwise these skip:
//
//	MYSQL_TEST_DSN="cqrs:cqrs@tcp(127.0.0.1:3306)/cqrs_test?parseTime=true" \
//	  go test -tags "integration goexperiment.jsonv2" ./...
//
// The nix runners export it: nix run .#integration-mysql-nspawn (or -vm).

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	_ "github.com/go-sql-driver/mysql"

	"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
)

// mysqlDSN returns MYSQL_TEST_DSN or skips the test.
func mysqlDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL integration test")
	}

	return dsn
}

// mysqlOpen drops the shared table for per-test isolation (the nix runners
// export a single DSN for the whole package) and recreates it via
// NewMySQLStore. Tests using it must not run in parallel with each other.
func mysqlOpen(t *testing.T) *sqlstore.Store {
	t.Helper()

	dsn := mysqlDSN(t)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping mysql: %v", err)
	}

	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS idempotency_keys"); err != nil {
		t.Fatalf("drop idempotency_keys: %v", err)
	}

	store, err := sqlstore.NewMySQLStore(ctx, db)
	if err != nil {
		t.Fatalf("create mysql store: %v", err)
	}

	return store
}

func TestIntegration_MySQLIdempotency_CheckAndRecordLifecycle(t *testing.T) {
	store := mysqlOpen(t)
	ctx := context.Background()

	if err := store.CheckAndRecord(ctx, "my-lifecycle", time.Minute); err != nil {
		t.Fatalf("first CheckAndRecord: %v", err)
	}

	err := store.CheckAndRecord(ctx, "my-lifecycle", time.Minute)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("duplicate CheckAndRecord: want ErrDuplicate, got %v", err)
	}

	assertIntegrationSeen(t, store, "my-lifecycle", true)

	if err := store.Record(ctx, "my-lifecycle", time.Minute); err != nil {
		t.Fatalf("Record on existing key must be a no-op without error: %v", err)
	}
}

func TestIntegration_MySQLIdempotency_AtomicClaimUnderConcurrency(t *testing.T) {
	store := mysqlOpen(t)
	concurrentClaimExactlyOnce(t, store, "my-atomic", 30)
}

func TestIntegration_MySQLIdempotency_TTLExpiryReclaimsKey(t *testing.T) {
	store := mysqlOpen(t)
	ctx := context.Background()
	ttl, wait := ttlTestParams()

	if err := store.CheckAndRecord(ctx, "my-ttl", ttl); err != nil {
		t.Fatalf("CheckAndRecord with short TTL: %v", err)
	}

	time.Sleep(wait)

	assertIntegrationSeen(t, store, "my-ttl", false)

	// Exercises the IF(...VALUES(expires_at)...) conditional update: the
	// expired row must be reclaimed (affected rows > 0), while an in-date row
	// must report 0 affected rows → ErrDuplicate (asserted by the lifecycle
	// test above).
	if err := store.CheckAndRecord(ctx, "my-ttl", time.Minute); err != nil {
		t.Fatalf("CheckAndRecord after expiry must reclaim: %v", err)
	}

	assertIntegrationSeen(t, store, "my-ttl", true)
}

func TestIntegration_MySQLIdempotency_VisibleAcrossConnections(t *testing.T) {
	dsn := mysqlDSN(t)
	store1 := mysqlOpen(t)

	if err := store1.CheckAndRecord(context.Background(), "my-shared", time.Minute); err != nil {
		t.Fatalf("claim on connection 1: %v", err)
	}

	// A second, independent connection to the SAME table (no drop/recreate).
	_, store2 := openStore(t, "mysql", dsn, sqlstore.NewMySQLStore)

	assertIntegrationSeen(t, store2, "my-shared", true)

	err := store2.CheckAndRecord(context.Background(), "my-shared", time.Minute)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("claim on connection 2: want ErrDuplicate, got %v", err)
	}
}
