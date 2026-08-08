//go:build integration

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestSQLiteWAL_ConcurrentReadWrite verifies that SQLite WAL mode allows
// multiple concurrent readers to coexist with a writer without "database is
// locked" errors. This is the core concurrency benefit of WAL over the default
// rollback journal mode.
func TestSQLiteWAL_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	dbPath := t.TempDir() + "/wal_concurrent.db"
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	// WAL allows multiple concurrent readers + one writer.
	db.SetMaxOpenConns(10)

	ctx := context.Background()

	_, err = db.ExecContext(ctx, `CREATE TABLE kv (key TEXT PRIMARY KEY, val INTEGER)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Verify WAL mode is active.
	var mode string

	err = db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}

	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want %q", mode, "wal")
	}

	const writers = 5
	const readers = 10
	const opsPerG = 50

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	var writeCount, readCount, errorCount int64

	for w := range writers {
		go func(w int) {
			defer wg.Done()

			for i := range opsPerG {
				key := fmt.Sprintf("key-%03d", w*opsPerG+i)
				_, err := db.ExecContext(ctx,
					"INSERT INTO kv (key, val) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET val = excluded.val",
					key, i,
				)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					t.Errorf("writer %d op %d: %v", w, i, err)

					return
				}

				atomic.AddInt64(&writeCount, 1)
			}
		}(w)
	}

	for r := range readers {
		go func(r int) {
			defer wg.Done()

			for i := range opsPerG {
				var count int

				err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM kv").Scan(&count)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					t.Errorf("reader %d op %d: %v", r, i, err)

					return
				}

				atomic.AddInt64(&readCount, 1)
			}
		}(r)
	}

	done := make(chan struct{})

	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for concurrent operations")
	}

	expectedWrites := int64(writers * opsPerG)
	if writeCount != expectedWrites {
		t.Errorf("writeCount = %d, want %d", writeCount, expectedWrites)
	}

	expectedReads := int64(readers * opsPerG)
	if readCount != expectedReads {
		t.Errorf("readCount = %d, want %d", readCount, expectedReads)
	}

	if errorCount != 0 {
		t.Errorf("errorCount = %d, want 0", errorCount)
	}

	var finalCount int

	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM kv").Scan(&finalCount)
	if err != nil {
		t.Fatalf("final count: %v", err)
	}

	if finalCount != int(expectedWrites) {
		t.Errorf("finalCount = %d, want %d", finalCount, expectedWrites)
	}
}

// TestSQLiteWAL_SnapshotIsolation verifies that a read transaction started
// before a write sees a consistent snapshot (WAL snapshot isolation), not the
// new data.
func TestSQLiteWAL_SnapshotIsolation(t *testing.T) {
	t.Parallel()

	dbPath := t.TempDir() + "/wal_snapshot.db"
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(2)

	ctx := context.Background()

	_, err = db.ExecContext(ctx, `CREATE TABLE counters (id INTEGER PRIMARY KEY, n INTEGER)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = db.ExecContext(ctx, "INSERT INTO counters (id, n) VALUES (1, 100)")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Start a read transaction (snapshot).
	readTx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatalf("begin read tx: %v", err)
	}

	defer func() { _ = readTx.Rollback() }()

	// Read initial value in the snapshot.
	var beforeWrite int

	err = readTx.QueryRowContext(ctx, "SELECT n FROM counters WHERE id = 1").Scan(&beforeWrite)
	if err != nil {
		t.Fatalf("snapshot read: %v", err)
	}

	// Concurrently update the value outside the snapshot.
	_, err = db.ExecContext(ctx, "UPDATE counters SET n = 999 WHERE id = 1")
	if err != nil {
		t.Fatalf("concurrent write: %v", err)
	}

	// Read again within the same snapshot — should see old value (100), not 999.
	var snapshotRead int

	err = readTx.QueryRowContext(ctx, "SELECT n FROM counters WHERE id = 1").Scan(&snapshotRead)
	if err != nil {
		t.Fatalf("snapshot read after write: %v", err)
	}

	if snapshotRead != beforeWrite {
		t.Errorf("snapshot isolation violated: read %d in snapshot, expected %d (WAL should isolate)",
			snapshotRead, beforeWrite)
	}

	// After committing the read tx, a new read should see the updated value.
	_ = readTx.Rollback()

	var afterWrite int

	err = db.QueryRowContext(ctx, "SELECT n FROM counters WHERE id = 1").Scan(&afterWrite)
	if err != nil {
		t.Fatalf("post-write read: %v", err)
	}

	if afterWrite != 999 {
		t.Errorf("post-write read = %d, want 999", afterWrite)
	}
}

// TestSQLiteWAL_BusyTimeoutPreventsLockError verifies that with a proper
// busy_timeout, concurrent writers retry instead of getting an immediate
// "database is locked" error.
func TestSQLiteWAL_BusyTimeoutPreventsLockError(t *testing.T) {
	t.Parallel()

	dbPath := t.TempDir() + "/wal_busy.db"
	// Short busy_timeout (1s) to keep the test fast.
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(1000)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(4)

	ctx := context.Background()

	_, err = db.ExecContext(ctx, `CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	const goroutines = 4
	const insertsPerG = 25

	var wg sync.WaitGroup
	wg.Add(goroutines)

	var successCount, errorCount int64

	for g := range goroutines {
		go func(g int) {
			defer wg.Done()

			for i := range insertsPerG {
				_, err := db.ExecContext(ctx,
					"INSERT INTO items (name) VALUES (?)",
					fmt.Sprintf("g%d-i%d", g, i),
				)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					t.Errorf("insert g%d-i%d: %v", g, i, err)

					return
				}

				atomic.AddInt64(&successCount, 1)
			}
		}(g)
	}

	done := make(chan struct{})

	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for concurrent inserts")
	}

	expected := int64(goroutines * insertsPerG)
	if successCount != expected {
		t.Errorf("successCount = %d, want %d", successCount, expected)
	}

	if errorCount != 0 {
		t.Errorf("errorCount = %d, want 0 (busy_timeout should retry)", errorCount)
	}
}
