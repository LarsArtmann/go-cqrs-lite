package conformance

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// pinCooperativeCancel pins the request-flag/finalize pair and its
// idempotency.
func (s *suite) pinCooperativeCancel(t *testing.T) {
	e := s.openEnv(t)

	tk := e.enqueue(t, task.New[Payload]{Type: "sh"})
	_ = e.claim(t, "w1")

	if err := e.store.CancelRunning(t.Context(), tk.ID, "stop it"); err != nil {
		t.Fatalf("cancel-running: %v", err)
	}

	if err := e.store.CancelRunning(t.Context(), tk.ID, "again"); err != nil {
		t.Fatalf("idempotent re-request: %v", err)
	}

	if got := countFacts(t, e, tk.ID, facts.CancelRequested); got != 1 {
		t.Fatalf("cancel-requested facts = %d, want 1 (idempotent)", got)
	}

	requested, err := e.store.CancelRequested(t.Context(), tk.ID)
	if err != nil || !requested {
		t.Fatalf("CancelRequested = %v, %v", requested, err)
	}

	// Only running tasks can carry a request.
	pending := e.enqueue(t, task.New[Payload]{Type: "sh"})
	mustError(
		t,
		"cancel-running pending",
		e.store.CancelRunning(t.Context(), pending.ID, "x"),
		queue.ErrInvalidTransition,
	)

	if err := e.store.CancelOwned(t.Context(), tk.ID, "w1"); err != nil {
		t.Fatalf("cancel-owned: %v", err)
	}

	got, _ := e.store.Get(t.Context(), tk.ID)
	if got.Status != task.Cancelled {
		t.Fatalf("status = %s, want cancelled", got.Status)
	}

	last := lastFact(t, e, tk.ID)
	if last.Type != facts.Cancelled || !contains(last.Detail, "stop it") {
		t.Fatalf(
			"final fact = %s %s, want cooperative cancelled with reason",
			last.Type,
			last.Detail,
		)
	}

	mustError(
		t,
		"cancel-owned after finalize",
		e.store.CancelOwned(t.Context(), tk.ID, "w1"),
		queue.ErrLeaseNotHeld,
	)
}

// pinReclaimFinalizesCancel pins the donor's crash path: a worker whose
// task got a cancel request never finalizes, the reclaim does it — the
// task is never re-executed after its cancel was requested.
func (s *suite) pinReclaimFinalizesCancel(t *testing.T) {
	e := s.openEnv(t)

	tk := e.enqueue(t, task.New[Payload]{Type: "sh"})

	_, err := e.store.ClaimDue(t.Context(), "dead-w", 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	if err := e.store.CancelRunning(t.Context(), tk.ID, "stale"); err != nil {
		t.Fatal(err)
	}

	time.Sleep(60 * time.Millisecond)

	// The reclaim finalizes the cancel instead of re-executing: this
	// claim must never hand THIS task out running again.
	if c, err := e.store.ClaimDue(
		t.Context(),
		"w2",
		time.Minute,
	); err == nil &&
		c.Task.ID == tk.ID {
		t.Fatal("reclaimed a cancel-requested task for execution")
	}

	got, _ := e.store.Get(t.Context(), tk.ID)
	if got.Status != task.Cancelled {
		t.Fatalf("status = %s, want cancelled (reclaim finalized it)", got.Status)
	}

	if last := lastFact(t, e, tk.ID); last.Type != facts.Cancelled {
		t.Fatalf("fact = %s, want cancelled", last.Type)
	}
}
