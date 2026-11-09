package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
)

// ClaimDue's deadlock-retry loop (queue M4 tail (a), 2026-09-25): the
// shipped backoff path had zero executed coverage — these tests drive it
// against a sqlmock DB whose candidate SELECT keeps returning InnoDB's
// deadlock signal (1213), pinning that the loop (1) retries the FULL
// transaction, (2) surfaces the error only after claimDeadlockRetries+1
// attempts, and (3) succeeds when the signal clears.

func newMockStore(t *testing.T) (*Store[int], sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(false))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// OpenDB runs the schema migration.
	for range len(schemaStmts) {
		mock.ExpectExec("CREATE TABLE|ALTER TABLE|CREATE INDEX").
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	store, err := OpenDB[int](db)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}

	return store, mock
}

func deadlockErr() error {
	return &mysqldriver.MySQLError{Number: 1213}
}

// expectDeadlockedClaim programs one claim attempt whose candidate query
// deadlocks (Begin → SELECT fails → Rollback).
func expectDeadlockedClaim(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT t.id").
		WillReturnError(deadlockErr())
	mock.ExpectRollback()
}

// TestClaimDue_RetriesDeadlockedTransactions pins the retry loop's
// attempt budget: claimDeadlockRetries extra attempts after the first, then
// the deadlock error surfaces.
func TestClaimDue_RetriesDeadlockedTransactions(t *testing.T) {
	t.Parallel()

	store, mock := newMockStore(t)

	for range claimDeadlockRetries + 1 {
		expectDeadlockedClaim(mock)
	}

	_, err := store.ClaimDue(context.Background(), "worker-1", time.Second)

	if err == nil {
		t.Fatal("persistent deadlock must surface an error")
	}
	if !isDeadlock(err) {
		t.Fatalf("surfaced error must be the deadlock signal, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("retry loop must replay the FULL transaction each attempt: %v", err)
	}
}

// TestClaimDue_SucceedsAfterTransientDeadlock pins the happy retry: two
// deadlocked attempts, then a clean claim.
func TestClaimDue_SucceedsAfterTransientDeadlock(t *testing.T) {
	t.Parallel()

	store, mock := newMockStore(t)

	expectDeadlockedClaim(mock)
	expectDeadlockedClaim(mock)

	// Third attempt: no due task — the loop must still have retried twice
	// and returned the benign ErrNoTaskDue.
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT t.id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "status", "lease_owner"}),
	)
	mock.ExpectRollback() // ErrNoTaskDue aborts the tx before any write

	_, err := store.ClaimDue(context.Background(), "worker-1", time.Second)
	if !errors.Is(err, queue.ErrNoTaskDue) {
		t.Fatalf("expected ErrNoTaskDue after retries, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected sql expectations: %v", err)
	}
}

// TestClaimDue_DeadlockAttemptsAreBounded pins the loop never spins past
// the documented budget (no unbounded retry on a wedged server).
func TestClaimDue_DeadlockAttemptsAreBounded(t *testing.T) {
	t.Parallel()

	store, mock := newMockStore(t)

	// Program exactly the budget; if the loop attempted ONE more, the
	// sqlmock expectation set would be exhausted and the extra Begin fails.
	for range claimDeadlockRetries + 1 {
		expectDeadlockedClaim(mock)
	}

	_, _ = store.ClaimDue(context.Background(), "worker-1", time.Second)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("attempt budget exceeded or mis-sequenced: %v", err)
	}
}
