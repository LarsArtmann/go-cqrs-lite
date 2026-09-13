package sqlstore_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"

	"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
)

// TestSQLiteStore_NonPositiveTTLRejected verifies that Record and CheckAndRecord
// reject a zero or negative TTL with ErrInvalidTTL, and that validation happens
// BEFORE any write: the table must stay empty (no row for the rejected key).
// A non-positive TTL would record an expiry already in the past, so the key
// would silently never be seen — rejecting it surfaces the caller bug instead.
// Closes the audit gap in docs/status/2026-08-07_22-38 (expiryFromTTL had zero
// dedicated tests).
func TestSQLiteStore_NonPositiveTTLRejected(t *testing.T) {
	t.Parallel()

	store, db := newTestStore(t)
	ctx := context.Background()

	invalidTTLs := []struct {
		name string
		ttl  time.Duration
	}{
		{"zero", 0},
		{"negative_nanosecond", -time.Nanosecond},
		{"negative_hour", -time.Hour},
	}

	for _, tc := range invalidTTLs {
		if err := store.Record(
			ctx,
			"ttl-rejected/"+tc.name,
			tc.ttl,
		); !errors.Is(
			err,
			idempotency.ErrInvalidTTL,
		) {
			t.Errorf("Record(%s): want ErrInvalidTTL, got %v", tc.name, err)
		}

		key := "ttl-rejected-checkandrecord/" + tc.name
		if err := store.CheckAndRecord(
			ctx,
			key,
			tc.ttl,
		); !errors.Is(
			err,
			idempotency.ErrInvalidTTL,
		) {
			t.Errorf("CheckAndRecord(%s): want ErrInvalidTTL, got %v", tc.name, err)
		}
	}

	assertIdempotencyTableEmpty(t, db)
	assertKeyUnseen(t, store, "ttl-rejected/zero")
}

// assertIdempotencyTableEmpty fails the test when any row was written despite
// TTL validation rejecting the calls.
func assertIdempotencyTableEmpty(t *testing.T, db *sql.DB) {
	t.Helper()

	var rows int
	if err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM idempotency_keys").Scan(&rows); err != nil {
		t.Fatalf("count idempotency_keys rows: %v", err)
	}

	if rows != 0 {
		t.Fatalf("rejected TTLs must not write rows, found %d", rows)
	}
}

// assertKeyUnseen fails the test when a key rejected for an invalid TTL is
// nonetheless reported as seen.
func assertKeyUnseen(t *testing.T, store *sqlstore.Store, key string) {
	t.Helper()

	seen, err := store.Seen(context.Background(), key)
	if err != nil {
		t.Fatalf("Seen(%q): %v", key, err)
	}

	if seen {
		t.Fatalf("key %q rejected for invalid TTL must not be Seen", key)
	}
}
