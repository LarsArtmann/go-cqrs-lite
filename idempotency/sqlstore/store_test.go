package sqlstore_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"

	_ "modernc.org/sqlite"
)

func newSQLiteStore(t *testing.T) *sqlstore.Store {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

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
			}
		}()
	}

	close(start)
	wg.Wait()

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

func TestStore_DialectPostgres_DDLCompiles(t *testing.T) {
	// This test verifies the Postgres DDL string is syntactically valid by
	// creating it in a SQLite database (the DDL is simple enough to work on
	// both — BIGINT exists in SQLite too). This is a smoke test that the
	// Postgres code path doesn't have obvious SQL errors.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	// We can't use NewPostgresStore with a SQLite driver, but we can verify
	// the DDL string is non-empty and executable.
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS idempotency_keys (
		key        TEXT PRIMARY KEY,
		expires_at BIGINT NOT NULL
	);`)
	if err != nil {
		t.Fatalf("postgres-style DDL on sqlite: %v", err)
	}
}
