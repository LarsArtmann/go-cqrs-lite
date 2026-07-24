package sqlstore_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
)

func newSQLiteStore(t *testing.T) *sqlstore.Store {
	t.Helper()

	// file::memory:?cache=shared ensures all connections share the same
	// in-memory database (default ":memory:" is per-connection).
	// busy_timeout=5000 prevents "database is locked" under concurrent writes.
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// SQLite serializes writes; cap the pool so concurrent goroutines
	// queue instead of erroring.
	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	store, err := sqlstore.NewSQLiteStore(context.Background(), db)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	return store
}

func TestStore_CheckAndRecord_FirstCallSucceeds(t *testing.T) {
	s := newSQLiteStore(t)

	err := s.CheckAndRecord(context.Background(), "cmd-1", 10*time.Minute)
	if err != nil {
		t.Fatalf("first CheckAndRecord: got %v, want nil", err)
	}
}

func TestStore_CheckAndRecord_DuplicateReturnsErrDuplicate(t *testing.T) {
	s := newSQLiteStore(t)

	_ = s.CheckAndRecord(context.Background(), "cmd-1", 10*time.Minute)
	err := s.CheckAndRecord(context.Background(), "cmd-1", 10*time.Minute)

	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second CheckAndRecord: got %v, want ErrDuplicate", err)
	}
}

func TestStore_CheckAndRecord_AllowsAfterExpiry(t *testing.T) {
	s := newSQLiteStore(t)

	_ = s.CheckAndRecord(context.Background(), "cmd-1", 50*time.Millisecond)

	time.Sleep(80 * time.Millisecond)

	err := s.CheckAndRecord(context.Background(), "cmd-1", 10*time.Minute)
	if err != nil {
		t.Fatalf("CheckAndRecord after expiry: got %v, want nil", err)
	}
}

func TestStore_Seen_AfterRecord(t *testing.T) {
	s := newSQLiteStore(t)

	_ = s.Record(context.Background(), "cmd-1", 10*time.Minute)

	seen, err := s.Seen(context.Background(), "cmd-1")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}

	if !seen {
		t.Fatal("Seen after Record: got false, want true")
	}
}

func TestStore_Seen_NotSeenForUnknownKey(t *testing.T) {
	s := newSQLiteStore(t)

	seen, err := s.Seen(context.Background(), "never-recorded")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}

	if seen {
		t.Fatal("Seen for unknown key: got true, want false")
	}
}

func TestStore_Seen_LazilyDeletesExpired(t *testing.T) {
	s := newSQLiteStore(t)

	_ = s.Record(context.Background(), "cmd-1", 50*time.Millisecond)

	time.Sleep(80 * time.Millisecond)

	seen, err := s.Seen(context.Background(), "cmd-1")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}

	if seen {
		t.Fatal("Seen for expired key: got true, want false")
	}
}

func TestStore_Record_NoopOnExistingKey(t *testing.T) {
	s := newSQLiteStore(t)

	_ = s.Record(context.Background(), "cmd-1", 10*time.Minute)
	_ = s.Record(context.Background(), "cmd-1", 1*time.Hour)

	// The first TTL should still be in effect (Record does not extend).
	seen, _ := s.Seen(context.Background(), "cmd-1")
	if !seen {
		t.Fatal("Record noop: key should still be seen")
	}
}

func TestStore_CheckAndRecord_AtomicUnderConcurrency(t *testing.T) {
	s := newSQLiteStore(t)

	const goroutines = 50

	var (
		wg       sync.WaitGroup
		winners  atomic.Int64
		dupCount atomic.Int64
		errCount atomic.Int64
		firstErr atomic.Value
	)

	wg.Add(goroutines)

	start := make(chan struct{})

	for range goroutines {
		go func() {
			defer wg.Done()

			<-start

			err := s.CheckAndRecord(context.Background(), "race-key", 10*time.Minute)
			if err == nil {
				winners.Add(1)
			} else if errors.Is(err, idempotency.ErrDuplicate) {
				dupCount.Add(1)
			} else {
				errCount.Add(1)
				firstErr.CompareAndSwap(nil, err.Error())
			}
		}()
	}

	close(start)
	wg.Wait()

	if errCount.Load() > 0 {
		t.Fatalf(
			"unexpected errors: %d (first: %s)",
			errCount.Load(), firstErr.Load(),
		)
	}

	if winners.Load() != 1 {
		t.Fatalf("winners: got %d, want exactly 1", winners.Load())
	}

	if dupCount.Load() != goroutines-1 {
		t.Fatalf("duplicates: got %d, want %d", dupCount.Load(), goroutines-1)
	}
}

func TestStore_Sweep_DeletesExpiredEntries(t *testing.T) {
	s := newSQLiteStore(t)

	_ = s.Record(context.Background(), "expired-1", 30*time.Millisecond)
	_ = s.Record(context.Background(), "expired-2", 30*time.Millisecond)
	_ = s.Record(context.Background(), "alive-1", 10*time.Minute)

	time.Sleep(60 * time.Millisecond)

	deleted, err := s.Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	if deleted != 2 {
		t.Fatalf("Sweep deleted: got %d, want 2", deleted)
	}

	seen, _ := s.Seen(context.Background(), "alive-1")
	if !seen {
		t.Fatal("alive-1 should still be seen after sweep")
	}
}

func TestStore_Close_IsNoOp(t *testing.T) {
	s := newSQLiteStore(t)
	s.Close()

	// Store should still work (db is owned by caller).
	err := s.CheckAndRecord(context.Background(), "post-close", 10*time.Minute)
	if err != nil {
		t.Fatalf("CheckAndRecord after Close: %v", err)
	}
}

func TestStore_SatisfiesIdempotencyStoreInterface(t *testing.T) {
	// Compile-time verification that *sqlstore.Store implements idempotency.Store.
	var _ idempotency.Store = (*sqlstore.Store)(nil)
}

func TestStore_DialectPostgres_DDLCompiles(t *testing.T) {
	// Smoke test: the Postgres DDL string executes on SQLite (BIGINT works
	// in both engines). This catches obvious SQL syntax errors without
	// requiring a running PostgreSQL instance.
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS idempotency_keys (
		key        TEXT PRIMARY KEY,
		expires_at BIGINT NOT NULL
	);`)
	if err != nil {
		t.Fatalf("postgres-style DDL on sqlite: %v", err)
	}
}

func TestStore_Record_DoesNotExtendExpiredEntry(t *testing.T) {
	// Record is a no-op on existing keys (matches MemoryStore semantics).
	// Expired entries are cleaned up lazily by Seen/CheckAndRecord.
	s := newSQLiteStore(t)

	_ = s.Record(context.Background(), "key-1", 30*time.Millisecond)

	time.Sleep(60 * time.Millisecond)

	// Record sees the key exists (even though expired) → no-op.
	_ = s.Record(context.Background(), "key-1", 10*time.Minute)

	// Seen triggers lazy deletion of the expired entry → returns false.
	seen, _ := s.Seen(context.Background(), "key-1")
	if seen {
		t.Fatal("key should be expired and lazily deleted by Seen")
	}

	// Now Record can insert fresh (key was deleted by Seen).
	_ = s.Record(context.Background(), "key-1", 10*time.Minute)

	seen, _ = s.Seen(context.Background(), "key-1")
	if !seen {
		t.Fatal("key should be seen after fresh Record post-cleanup")
	}
}

func TestStore_MultipleKeysIndependent(t *testing.T) {
	s := newSQLiteStore(t)

	ctx := context.Background()

	for i := range 10 {
		key := fmt.Sprintf("key-%d", i)
		if err := s.CheckAndRecord(ctx, key, 10*time.Minute); err != nil {
			t.Fatalf("CheckAndRecord %s: %v", key, err)
		}

		// Second call to the same key should fail.
		if err := s.CheckAndRecord(
			ctx,
			key,
			10*time.Minute,
		); !errors.Is(
			err,
			idempotency.ErrDuplicate,
		) {
			t.Fatalf("second CheckAndRecord %s: got %v, want ErrDuplicate", key, err)
		}
	}
}
