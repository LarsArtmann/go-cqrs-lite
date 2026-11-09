//go:build integration

package sqlstore_test

// PostgreSQL integration tests for the idempotency sqlstore. Run with:
//
//	nix run .#integration-pg -- -run TestIntegration_PostgresIdempotency
//
// They exercise the dialect path SQLite cannot prove: $N placeholders, BIGINT
// expires_at, row-level serialization of INSERT ... ON CONFLICT DO UPDATE ...
// WHERE under real connection parallelism, and cross-connection visibility
// (the multi-process dedup scenario the SQL store exists for).

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"

	"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
)

func TestIntegration_PostgresIdempotency_CheckAndRecordLifecycle(t *testing.T) {
	t.Parallel()

	_, store := openStore(t, "pgx", pgTestDSN(t), sqlstore.NewPostgresStore)
	ctx := context.Background()

	if err := store.CheckAndRecord(ctx, "pg-lifecycle", time.Minute); err != nil {
		t.Fatalf("first CheckAndRecord: %v", err)
	}

	err := store.CheckAndRecord(ctx, "pg-lifecycle", time.Minute)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("duplicate CheckAndRecord: want ErrDuplicate, got %v", err)
	}

	assertIntegrationSeen(t, store, "pg-lifecycle", true)

	if err := store.Record(ctx, "pg-lifecycle", time.Minute); err != nil {
		t.Fatalf("Record on existing key must be a no-op without error: %v", err)
	}

	if err := store.CheckAndRecord(ctx, "pg-lifecycle-other", time.Minute); err != nil {
		t.Fatalf("different key must be independent: %v", err)
	}
}

func TestIntegration_PostgresIdempotency_AtomicClaimUnderConcurrency(t *testing.T) {
	t.Parallel()

	_, store := openStore(t, "pgx", pgTestDSN(t), sqlstore.NewPostgresStore)
	concurrentClaimExactlyOnce(t, store, "pg-atomic", 50)
}

func TestIntegration_PostgresIdempotency_TTLExpiryReclaimsKey(t *testing.T) {
	t.Parallel()

	_, store := openStore(t, "pgx", pgTestDSN(t), sqlstore.NewPostgresStore)
	ctx := context.Background()
	ttl, wait := ttlTestParams()

	if err := store.CheckAndRecord(ctx, "pg-ttl", ttl); err != nil {
		t.Fatalf("CheckAndRecord with short TTL: %v", err)
	}

	time.Sleep(wait)

	assertIntegrationSeen(t, store, "pg-ttl", false)

	// The conditional upsert (ON CONFLICT DO UPDATE ... WHERE expires_at < $3)
	// must reclaim the expired row on a real server.
	if err := store.CheckAndRecord(ctx, "pg-ttl", time.Minute); err != nil {
		t.Fatalf("CheckAndRecord after expiry must reclaim: %v", err)
	}

	assertIntegrationSeen(t, store, "pg-ttl", true)
}

func TestIntegration_PostgresIdempotency_SweepDeletesExpired(t *testing.T) {
	t.Parallel()

	_, store := openStore(t, "pgx", pgTestDSN(t), sqlstore.NewPostgresStore)
	ctx := context.Background()
	ttl, wait := ttlTestParams()

	for _, key := range []string{"pg-sweep-1", "pg-sweep-2"} {
		if err := store.Record(ctx, key, ttl); err != nil {
			t.Fatalf("Record %s: %v", key, err)
		}
	}

	if err := store.Record(ctx, "pg-sweep-live", time.Hour); err != nil {
		t.Fatalf("Record live key: %v", err)
	}

	time.Sleep(wait)

	deleted, err := store.Sweep(ctx)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	if deleted != 2 {
		t.Fatalf("Sweep: want 2 expired rows deleted, got %d", deleted)
	}

	assertIntegrationSeen(t, store, "pg-sweep-live", true)
	assertIntegrationSeen(t, store, "pg-sweep-1", false)
}

// TestIntegration_PostgresIdempotency_VisibleAcrossConnections is the core
// multi-process story: a key claimed over one connection is seen and rejected
// over a second, independent connection — the reason consumers pick the SQL
// store over the in-process MemoryStore.
func TestIntegration_PostgresIdempotency_VisibleAcrossConnections(t *testing.T) {
	t.Parallel()

	dsn := pgTestDSN(t)
	_, store1 := openStore(t, "pgx", dsn, sqlstore.NewPostgresStore)

	if err := store1.CheckAndRecord(context.Background(), "pg-shared", time.Minute); err != nil {
		t.Fatalf("claim on connection 1: %v", err)
	}

	_, store2 := openStore(t, "pgx", dsn, sqlstore.NewPostgresStore)

	assertIntegrationSeen(t, store2, "pg-shared", true)

	err := store2.CheckAndRecord(context.Background(), "pg-shared", time.Minute)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("claim on connection 2: want ErrDuplicate, got %v", err)
	}
}
