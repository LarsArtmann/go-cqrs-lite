package conformance

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// runTokens pins the ADR-0134 fencing-token contract: every claim mints
// a fresh unguessable token, every finalize is token-checked, and a
// holder whose lease was re-claimed loses to the new token — the
// finalize path is the theft detector.
func (s *suite) runTokens(t *testing.T) {
	t.Run("token minted per claim", s.pinTokenMintedPerClaim)
	t.Run("theft: lapsed holder loses to the new token", s.pinTheftDetected)
}

// pinTokenMintedPerClaim pins that claims carry non-empty tokens and a
// reclaim mints a fresh one (never reused).
func (s *suite) pinTokenMintedPerClaim(t *testing.T) {
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh"})

	first := e.claim(t, "w1")
	if first.Token == "" {
		t.Fatal("first claim carries no token")
	}

	time.Sleep(60 * time.Millisecond)

	// Not a reclaim yet — but expire the lease and reclaim to observe a
	// second mint.
	_, err := e.store.ClaimDue(t.Context(), "w2", time.Minute)
	if !errors.Is(err, queue.ErrNoTaskDue) {
		t.Fatalf("pre-expiry claim: error = %v, want ErrNoTaskDue", err)
	}

	short := e.enqueue(t, task.New[Payload]{Type: "sh"})

	c, err := e.store.ClaimDue(t.Context(), "w3", 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	if c.Task.ID != short.ID || c.Token == "" {
		t.Fatalf("short claim = %s token=%q, want the fresh task with a token", c.Task.ID, c.Token)
	}

	time.Sleep(60 * time.Millisecond)

	reclaimed := e.claim(t, "w4")
	if reclaimed.Task.ID != short.ID {
		t.Fatalf("reclaim claimed %s, want %s", reclaimed.Task.ID, short.ID)
	}

	if reclaimed.Token == "" || reclaimed.Token == c.Token {
		t.Fatalf("reclaim reused the token: %q vs %q", reclaimed.Token, c.Token)
	}

	// The lapsed holder cannot heartbeat its old claim back to life.
	mustError(
		t,
		"lapsed holder heartbeat",
		e.store.Heartbeat(t.Context(), short.ID, c.Token, time.Minute),
		queue.ErrLeaseNotHeld,
	)

	_ = subject
}

// pinTheftDetected pins the core ADR-0134 story: worker 1's lease lapses,
// worker 2 re-claims (fresh token), worker 1's Complete fails with
// ErrLeaseNotHeld while worker 2 succeeds — and the journal attributes
// the completion to worker 2.
func (s *suite) pinTheftDetected(t *testing.T) {
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh"})

	stolen, err := e.store.ClaimDue(t.Context(), "w1", 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(60 * time.Millisecond)

	heir := e.claim(t, "w2")
	if heir.Task.ID != subject.ID {
		t.Fatalf("reclaim claimed %s, want %s", heir.Task.ID, subject.ID)
	}

	// The lapsed holder's finalize is the theft detection point.
	mustError(
		t,
		"lapsed holder complete",
		e.store.Complete(t.Context(), subject.ID, stolen.Token, nil),
		queue.ErrLeaseNotHeld,
	)

	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.Status != task.Running || got.LeaseOwner != "w2" {
		t.Fatalf("theft attempt disturbed the claim: %+v", got)
	}

	if err := e.store.Complete(t.Context(), subject.ID, heir.Token, nil); err != nil {
		t.Fatalf("heir complete: %v", err)
	}

	var completed *facts.Fact

	for _, f := range factsFor(t, e, subject.ID) {
		if f.Type == facts.Completed {
			cf := f
			completed = &cf
		}
	}

	if completed == nil {
		t.Fatal("no completed fact after heir finalize")
	}

	if completed.Owner != "w2" {
		t.Fatalf("completed fact owner = %q, want the reclaiming worker w2", completed.Owner)
	}
}
