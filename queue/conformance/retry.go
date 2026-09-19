package conformance

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// runRetry pins the failure story: the attempt ladder with backoff, the
// dead-letter queue, permanent failures, requeue without attempt burn,
// and the failure-evidence forensics.
func (s *suite) runRetry(t *testing.T) {
	t.Run("fail with attempts remaining requeues with backoff", s.pinBackoffLadder)
	t.Run("fail to dead-letter after budget", s.pinDeadLetter)
	t.Run("permanent failure dead-letters immediately", s.pinPermanent)
	t.Run("failure evidence rides the fact verbatim", s.pinEvidence)
	t.Run("requeue burns no attempt", s.pinRequeue)
	t.Run("rescue restores a fresh budget", s.pinRescue)
	t.Run("dismiss withdraws with reason and by", s.pinDismiss)
}

// pinBackoffLadder pins attempt counting and the NotBefore backoff.
func (s *suite) pinBackoffLadder(t *testing.T) {
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh", MaxAttempts: 3})
	c := e.claim(t, "w1")

	// Zero backoff: the task re-enters the ready set immediately.
	if err := e.store.Fail(t.Context(), subject.ID, c.Token, "boom", 0, nil); err != nil {
		t.Fatalf("fail: %v", err)
	}

	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.Status != task.Pending || got.Attempts != 1 {
		t.Fatalf("after fail 1: status=%s attempts=%d, want pending/1", got.Status, got.Attempts)
	}

	if got.LastError != "boom" {
		t.Fatalf("LastError = %q, want the error text", got.LastError)
	}

	f := lastFact(t, e, subject.ID)
	if f.Type != facts.Failed || f.Attempt != 1 || f.Error != "boom" {
		t.Fatalf("failed fact = %+v, want attempt 1 carrying the error", f)
	}

	// The re-claim sees the counted attempt.
	c = e.claim(t, "w1")
	if c.Task.ID != subject.ID || c.Task.Attempts != 1 {
		t.Fatalf("re-claim = %s attempts=%d, want the same task at 1", c.Task.ID, c.Task.Attempts)
	}

	// A real backoff parks the task until NotBefore.
	if err := e.store.Fail(t.Context(), subject.ID, c.Token, "boom2", time.Hour, nil); err != nil {
		t.Fatal(err)
	}

	got, _ = e.store.Get(t.Context(), subject.ID)
	if got.Status != task.Pending || got.Attempts != 2 {
		t.Fatalf("after fail 2: status=%s attempts=%d, want pending/2", got.Status, got.Attempts)
	}

	if !got.NotBefore.After(time.Now()) {
		t.Fatalf("NotBefore = %v, want now+backoff (parked)", got.NotBefore)
	}
}

// pinDeadLetter pins the exhausted ladder: the third failure of three
// dead-letters with the class-stamped fact pair.
func (s *suite) pinDeadLetter(t *testing.T) {
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh", MaxAttempts: 2})

	for attempt := 1; attempt <= 2; attempt++ {
		c := e.claim(t, "w1")
		if c.Task.ID != subject.ID {
			t.Fatalf("attempt %d claimed %s, want %s", attempt, c.Task.ID, subject.ID)
		}

		if err := e.store.Fail(t.Context(), subject.ID, c.Token, "boom", 0, nil); err != nil {
			t.Fatal(err)
		}
	}

	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.Status != task.Dead || got.Attempts != 2 {
		t.Fatalf("after budget: status=%s attempts=%d, want dead/2", got.Status, got.Attempts)
	}

	if got.LeaseOwner != "" || got.LeaseExpires != nil {
		t.Fatalf("dead task still holds a lease: %+v", got)
	}

	types := factTypes(t, e, subject.ID)

	want := []facts.FactType{
		facts.Enqueued,
		facts.Claimed,
		facts.Failed,
		facts.Claimed,
		facts.Failed,
		facts.DeadLettered,
	}
	if !equalFactTypes(types, want) {
		t.Fatalf("fact trail = %v, want %v", types, want)
	}

	last := lastFact(t, e, subject.ID)
	if last.Type != facts.DeadLettered || !bytes.Contains(last.Detail, []byte("exhausted")) {
		t.Fatalf("dead-lettered fact = %s %s, want class exhausted", last.Type, last.Detail)
	}

	// Dead is fenced: no claim until rescued or dismissed.
	_, err := e.store.ClaimDue(t.Context(), "w1", time.Minute)
	if err == nil {
		t.Fatal("dead task was claimable")
	}
}

// pinPermanent pins FailPermanent: dead immediately, budget irrelevant,
// class "permanent".
func (s *suite) pinPermanent(t *testing.T) {
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh", MaxAttempts: 9})
	c := e.claim(t, "w1")

	if err := e.store.FailPermanent(t.Context(), subject.ID, c.Token, "bad payload", nil); err != nil {
		t.Fatalf("fail-permanent: %v", err)
	}

	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.Status != task.Dead {
		t.Fatalf("status = %s, want dead", got.Status)
	}

	if got.Attempts != 1 {
		t.Fatalf("attempts = %d, want the single counted attempt", got.Attempts)
	}

	last := lastFact(t, e, subject.ID)
	if last.Type != facts.DeadLettered || !bytes.Contains(last.Detail, []byte("permanent")) {
		t.Fatalf("fact = %s %s, want class permanent", last.Type, last.Detail)
	}
}

// pinEvidence pins the forensics contract: evidence bytes ride the
// Failed fact's Detail verbatim — big enough to be a real tail.
func (s *suite) pinEvidence(t *testing.T) {
	//art-dupl:accept standard scenario prologue (openEnv + enqueue); independent scenario tests
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh"})
	c := e.claim(t, "w1")

	evidence := []byte(
		`{"stage":"verify","exit_code":1,"tail":"` + strings.Repeat("x", 2048) + `"}`,
	)
	if err := e.store.Fail(
		t.Context(),
		subject.ID,
		c.Token,
		"verify failed",
		0,
		evidence,
	); err != nil {
		t.Fatal(err)
	}

	for _, f := range factsFor(t, e, subject.ID) {
		if f.Type != facts.Failed {
			continue
		}

		if !bytes.Equal(f.Detail, evidence) {
			t.Fatalf(
				"evidence not verbatim on the fact: got %d bytes, want %d",
				len(f.Detail),
				len(evidence),
			)
		}

		return
	}

	t.Fatal("no failed fact found")
}

// pinRequeue pins the preflight path: back to pending, no attempt
// burned, RequeueEvidence on the fact.
func (s *suite) pinRequeue(t *testing.T) {
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh", MaxAttempts: 3})
	c := e.claim(t, "w1")

	if err := e.store.Requeue(
		t.Context(),
		subject.ID,
		c.Token,
		"env not ready",
		50*time.Millisecond,
	); err != nil {
		t.Fatalf("requeue: %v", err)
	}

	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.Status != task.Pending || got.Attempts != 0 {
		t.Fatalf(
			"after requeue: status=%s attempts=%d, want pending/0 (no burn)",
			got.Status,
			got.Attempts,
		)
	}

	if got.NotBefore.Before(time.Now()) {
		t.Fatalf("NotBefore = %v, want now+delay", got.NotBefore)
	}

	f := lastFact(t, e, subject.ID)
	if f.Type != facts.Requeued {
		t.Fatalf("fact = %s, want requeued", f.Type)
	}

	if !bytes.Contains(f.Detail, []byte("env not ready")) ||
		!bytes.Contains(f.Detail, []byte("retry_in_ms")) {
		t.Fatalf("requeue evidence missing: %s", f.Detail)
	}

	// Claimable again once the delay lapses; the re-claim is
	// lease-checked like every finalize.
	time.Sleep(100 * time.Millisecond)

	c = e.claim(t, "w1")
	mustError(
		t,
		"requeue stale token",
		e.store.Requeue(t.Context(), c.Task.ID, "forged-token", "x", 0),
		queue.ErrLeaseNotHeld,
	)
}

// pinRescue pins RescueDead: fresh budget, claimable again, Enqueued
// fact with the rescue marker.
func (s *suite) pinRescue(t *testing.T) {
	e := s.openEnv(t)

	dead := deadTask(t, e)

	if err := e.store.RescueDead(t.Context(), dead.ID, 2); err != nil {
		t.Fatalf("rescue: %v", err)
	}

	got, _ := e.store.Get(t.Context(), dead.ID)
	if got.Status != task.Pending || got.Attempts != 0 || got.MaxAttempts != 2 {
		t.Fatalf(
			"after rescue: status=%s attempts=%d max=%d, want pending/0/2",
			got.Status,
			got.Attempts,
			got.MaxAttempts,
		)
	}

	if got.LastError != "" {
		t.Fatalf("LastError = %q, want cleared", got.LastError)
	}

	last := lastFact(t, e, dead.ID)
	if last.Type != facts.Enqueued || !bytes.Contains(last.Detail, []byte("rescue")) {
		t.Fatalf("fact = %s %s, want enqueued with rescue marker", last.Type, last.Detail)
	}

	c := e.claim(t, "w1")
	if c.Task.ID != dead.ID {
		t.Fatalf("rescued task not claimable, got %s", c.Task.ID)
	}
}

// deadTask drives one task to Dead and returns its pre-death record.
func deadTask(t *testing.T, e *env) task.Task[Payload] {
	t.Helper()

	subject := e.enqueue(t, task.New[Payload]{Type: "sh", MaxAttempts: 1})

	c := e.claim(t, "w1")
	if c.Task.ID != subject.ID {
		t.Fatalf("claimed %s, want %s", c.Task.ID, subject.ID)
	}

	if err := e.store.Fail(t.Context(), subject.ID, c.Token, "boom", 0, nil); err != nil {
		t.Fatal(err)
	}

	return subject
}

// pinDismiss pins DismissDead: Dead → Cancelled with reason and by on
// the fact.
func (s *suite) pinDismiss(t *testing.T) {
	e := s.openEnv(t)

	dead := deadTask(t, e)

	if err := e.store.DismissDead(t.Context(), dead.ID, "unfixable", "operator"); err != nil {
		t.Fatalf("dismiss: %v", err)
	}

	got, _ := e.store.Get(t.Context(), dead.ID)
	if got.Status != task.Cancelled {
		t.Fatalf("status = %s, want cancelled", got.Status)
	}

	last := lastFact(t, e, dead.ID)
	if last.Type != facts.Cancelled {
		t.Fatalf("fact = %s, want cancelled", last.Type)
	}

	if !bytes.Contains(last.Detail, []byte("unfixable")) ||
		!bytes.Contains(last.Detail, []byte("operator")) {
		t.Fatalf("dismiss detail = %s, want reason and by", last.Detail)
	}
}
