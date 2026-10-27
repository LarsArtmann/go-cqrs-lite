package metaengine

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Context-cancellation matrix (testing suite T24).
//
// The matrix exists because ctx handling is the one contract every layer
// assumes and nobody asserts: a `context.Canceled` that turns into a panic, a
// nil error, or a half-applied write is invisible until an HTTP handler times
// out under production traffic.
//
// States: already-canceled, past-deadline, canceled-inside-the-callback.
// Every combination must (a) not panic, (b) leave the store usable, and for the
// READ path (c) surface the context error itself.
//
// KNOWN GAP, pinned here rather than hidden: the WRITE path on the memory engine
// does not observe cancellation — `Apply`, `ApplyIdempotent` and
// `InTransaction` return nil and perform the write. Reads do observe it. That
// asymmetry is recorded in TODO_LIST (T24 remaining) because changing write
// semantics is a design decision (a write that has not started should abort; one
// already committed must not), not a test fix.

func cancellationMatrix(t *testing.T) map[string]context.Context {
	t.Helper()

	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	expired, expiredCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	t.Cleanup(expiredCancel)

	return map[string]context.Context{
		"canceled":        canceled,
		"deadline-passed": expired,
	}
}

func TestContextCancellation_ReadPathSurfacesTheError(t *testing.T) {
	t.Parallel()

	store, _ := roleTestStore(t)

	// Seed with a live context so the only reason to fail is cancellation.
	if err := store.Apply(context.Background(), "role_item_created", roleItemCreated{ID: "a", Name: "A"}); err != nil {
		t.Fatalf("seed Apply: %v", err)
	}

	for name, ctx := range cancellationMatrix(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := store.ExecuteQueryByName(ctx, "role_items", roleFindItem{ID: "a"})
			if err == nil {
				t.Fatalf("query with a %s context returned no error", name)
			}
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("query error = %v, want it to wrap context.Canceled or context.DeadlineExceeded", err)
			}
		})
	}
}

func TestContextCancellation_WritePathNeverPanicsAndStaysUsable(t *testing.T) {
	t.Parallel()

	for name, ctx := range cancellationMatrix(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store, _ := roleTestStore(t)

			// Each of these must return without panicking. The error is
			// deliberately not asserted: see the KNOWN GAP note above.
			_ = store.Apply(ctx, "role_item_created", roleItemCreated{ID: "c", Name: "C"})
			_ = store.ApplyIdempotent(ctx, "idem-cancel-1", "role_item_created", roleItemCreated{ID: "d", Name: "D"})
			_ = store.InTransaction(ctx, func(context.Context) error { return nil })
			_ = store.ApplyBatch(ctx, nil)

			// The store must still serve a normal request afterwards: a
			// canceled call that poisons shared state would be far worse than
			// one that returns nil.
			if err := store.Apply(context.Background(), "role_item_created", roleItemCreated{ID: "after", Name: "After"}); err != nil {
				t.Fatalf("Apply after a %s call failed: %v", name, err)
			}

			if _, err := store.ExecuteQueryByName(context.Background(), "role_items", roleFindItem{ID: "after"}); err != nil {
				t.Fatalf("query after a %s call failed: %v", name, err)
			}
		})
	}
}

// TestContextCancellation_CancelInsideTransaction pins the callback contract:
// when the context is canceled DURING the transaction body, the body sees the
// cancellation, the call does not panic, and the store survives.
func TestContextCancellation_CancelInsideTransaction(t *testing.T) {
	t.Parallel()

	store, _ := roleTestStore(t)

	ctx, cancel := context.WithCancel(context.Background())

	var bodySawCancellation bool

	err := store.InTransaction(ctx, func(txCtx context.Context) error {
		cancel()
		// Let the cancellation propagate before the body inspects the ctx.
		select {
		case <-txCtx.Done():
			bodySawCancellation = true
		case <-time.After(2 * time.Second):
			t.Error("transaction context did not observe cancellation within 2s")
		}

		return txCtx.Err()
	})

	if !bodySawCancellation {
		t.Fatal("the callback never observed cancellation on its own context")
	}

	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("InTransaction error = %v, want nil or context.Canceled", err)
	}

	if applyErr := store.Apply(context.Background(), "role_item_created", roleItemCreated{ID: "post-tx", Name: "P"}); applyErr != nil {
		t.Fatalf("Apply after a canceled transaction failed: %v", applyErr)
	}
}
