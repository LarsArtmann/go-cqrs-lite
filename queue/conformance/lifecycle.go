package conformance

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// runLifecycle pins the task lifecycle: the state machine, the
// enqueue→claim→complete roundtrip, heartbeats, and the cooperative
// cancel family.
func (s *suite) runLifecycle(t *testing.T) {
	t.Run("transition matrix is the contract's", s.pinTransitionMatrix)
	t.Run("enqueue assigns identity and defaults", s.pinEnqueueDefaults)
	t.Run("enqueue to claim to complete roundtrip", s.pinRoundtrip)
	t.Run("empty type is refused", s.pinEmptyType)
	t.Run("unknown id is not found", s.pinNotFound)
	t.Run("lease guards finalize calls", s.pinLeaseGuards)
	t.Run("heartbeat extends the lease", s.pinHeartbeat)
	t.Run("cancel withdraws a pending task with reason", s.pinCancel)
	t.Run("status guards refuse wrong-source transitions", s.pinStatusGuards)
	t.Run("cooperative cancel family", s.pinCooperativeCancel)
	t.Run("reclaim finalizes a requested cancel", s.pinReclaimFinalizesCancel)
}

// pinTransitionMatrix pins the pure state machine table itself.
func (s *suite) pinTransitionMatrix(t *testing.T) {
	legal := map[task.Status][]task.Status{
		task.Pending:   {task.Running, task.Cancelled},
		task.Running:   {task.Pending, task.Completed, task.Dead, task.Cancelled},
		task.Completed: {},
		task.Dead:      {task.Pending, task.Cancelled},
		task.Cancelled: {},
	}

	for _, from := range task.AllStatuses() {
		for _, target := range task.AllStatuses() {
			want := slices.Contains(legal[from], target)

			if got := task.CanTransitionTo(from, target); got != want {
				t.Fatalf("CanTransitionTo(%s, %s) = %v, want %v", from, target, got, want)
			}
		}
	}

	for _, status := range task.AllStatuses() {
		if want := status == task.Completed || status == task.Dead ||
			status == task.Cancelled; task.Terminal(
			status,
		) != want {
			t.Fatalf("Terminal(%s) mismatch", status)
		}

		if !status.Valid() {
			t.Fatalf("AllStatuses contains invalid %s", status)
		}
	}

	if task.Status("bogus").Valid() {
		t.Fatal("unknown status reports Valid")
	}
}

// pinEnqueueDefaults pins ID/defaults assignment and field round-trip.
func (s *suite) pinEnqueueDefaults(t *testing.T) {
	e := s.openEnv(t)

	want := task.New[Payload]{
		Project:     "proj",
		Type:        "email",
		Payload:     Payload{Cmd: "send"},
		Priority:    7,
		MaxAttempts: 5,
	}

	got := e.enqueue(t, want)
	if got.ID == "" {
		t.Fatal("store assigned no ID")
	}

	if got.Status != task.Pending || got.Attempts != 0 || got.MaxAttempts != 5 {
		t.Fatalf(
			"defaults wrong: status=%s attempts=%d max=%d",
			got.Status,
			got.Attempts,
			got.MaxAttempts,
		)
	}

	if got.Payload.Cmd != "send" || got.Project != "proj" || got.Priority != 7 {
		t.Fatalf("field roundtrip mismatch: %+v", got)
	}

	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatal("timestamps not assigned")
	}

	// Zero MaxAttempts normalizes to the contract default.
	norm := e.enqueue(t, task.New[Payload]{Type: "t"})
	if norm.MaxAttempts != task.DefaultMaxAttempts {
		t.Fatalf("MaxAttempts = %d, want default %d", norm.MaxAttempts, task.DefaultMaxAttempts)
	}
}

// pinRoundtrip pins the happy path: pending → claim → running → complete,
// with the facts landing in the same transactions.
func (s *suite) pinRoundtrip(t *testing.T) {
	//art-dupl:accept standard scenario prologue (openEnv + enqueue); independent scenario tests
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh"})
	c := e.claim(t, "w1")

	if c.Task.ID != subject.ID || c.Task.Status != task.Running || c.Task.LeaseOwner != "w1" {
		t.Fatalf("claim state wrong: %+v", c.Task)
	}

	if c.LeaseUntil.IsZero() || c.Task.LeaseExpires == nil ||
		!c.Task.LeaseExpires.Equal(c.LeaseUntil) {
		t.Fatal("claim lease deadline not surfaced")
	}

	if c.Token == "" {
		t.Fatal("claim carries no token (ADR-0134 fencing)")
	}

	_, err := e.store.ClaimDue(t.Context(), "w2", time.Minute)
	if !errors.Is(err, queue.ErrNoTaskDue) {
		t.Fatalf("second claim: error = %v, want ErrNoTaskDue", err)
	}

	if err := e.store.Complete(
		t.Context(),
		subject.ID,
		c.Token,
		[]byte(`{"ok":true}`),
	); err != nil {
		t.Fatalf("complete: %v", err)
	}

	got, err := e.store.Get(t.Context(), subject.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got.Status != task.Completed || got.CompletedAt == nil || got.LeaseOwner != "" {
		t.Fatalf("completed state wrong: %+v", got)
	}

	want := []facts.FactType{facts.Enqueued, facts.Claimed, facts.Completed}
	if got := factTypes(t, e, subject.ID); !equalFactTypes(got, want) {
		t.Fatalf("fact trail = %v, want %v", got, want)
	}
}

// pinEmptyType pins the ErrEmptyType refusal.
func (s *suite) pinEmptyType(t *testing.T) {
	e := s.openEnv(t)

	_, err := e.store.Enqueue(t.Context(), task.New[Payload]{})
	mustError(t, "enqueue empty type", err, queue.ErrEmptyType)
}

// pinNotFound pins ErrNotFound on unknown IDs.
func (s *suite) pinNotFound(t *testing.T) {
	e := s.openEnv(t)

	id := task.NewID()

	_, err := e.store.Get(t.Context(), id)
	mustError(t, "get unknown", err, queue.ErrNotFound)

	if err := e.store.Complete(t.Context(), id, "no-such-token", nil); err == nil {
		t.Fatal("complete unknown: expected error, got nil")
	}
}

// pinLeaseGuards pins that finalize calls are token-checked: wrong
// tokens and expired leases get ErrLeaseNotHeld.
func (s *suite) pinLeaseGuards(t *testing.T) {
	//art-dupl:accept standard scenario prologue (openEnv + enqueue); independent scenario tests
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh"})

	// Claim subject once so it is Running; the forged-token calls below
	// must not change that state.
	_ = e.claim(t, "w1")

	mustError(
		t,
		"complete wrong token",
		e.store.Complete(t.Context(), subject.ID, "forged-token", nil),
		queue.ErrLeaseNotHeld,
	)
	mustError(
		t,
		"fail wrong token",
		e.store.Fail(t.Context(), subject.ID, "forged-token", "x", 0, nil),
		queue.ErrLeaseNotHeld,
	)
	mustError(
		t,
		"heartbeat wrong token",
		e.store.Heartbeat(t.Context(), subject.ID, "forged-token", time.Minute),
		queue.ErrLeaseNotHeld,
	)

	// The task survived every forged finalize.
	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.Status != task.Running {
		t.Fatalf("status = %s after forged finalizes, want running", got.Status)
	}

	// Expired lease: a second task claimed short completes after its
	// deadline is refused.
	_ = e.enqueue(t, task.New[Payload]{Type: "sh"})

	c, err := e.store.ClaimDue(t.Context(), "expire-w", 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(60 * time.Millisecond)

	mustError(
		t,
		"complete expired",
		e.store.Complete(t.Context(), c.Task.ID, c.Token, nil),
		queue.ErrLeaseNotHeld,
	)
}

// pinHeartbeat pins lease extension.
func (s *suite) pinHeartbeat(t *testing.T) {
	//art-dupl:accept standard scenario prologue (openEnv + enqueue); independent scenario tests
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh"})
	c := e.claim(t, "w1")

	// Leases are stamped ms-truncated; a heartbeat in the SAME millisecond
	// with an equal duration rewrites an identical deadline. Sleep past the
	// boundary so strict extension is deterministic.
	time.Sleep(2 * time.Millisecond)

	if err := e.store.Heartbeat(t.Context(), subject.ID, c.Token, time.Minute); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.LeaseExpires == nil || !got.LeaseExpires.After(c.LeaseUntil) {
		t.Fatalf("lease not extended: %v <= %v", got.LeaseExpires, c.LeaseUntil)
	}
}

// pinCancel pins Cancel on pending with the reason riding the fact.
func (s *suite) pinCancel(t *testing.T) {
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh"})
	if err := e.store.Cancel(t.Context(), subject.ID, "superseded"); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.Status != task.Cancelled {
		t.Fatalf("status = %s, want cancelled", got.Status)
	}

	last := lastFact(t, e, subject.ID)
	if last.Type != facts.Cancelled {
		t.Fatalf("fact = %s, want cancelled", last.Type)
	}

	if !contains(last.Detail, "superseded") {
		t.Fatalf("cancel reason not on fact detail: %s", last.Detail)
	}

	// Cancel is terminal: repeating it is an invalid transition now.
	mustError(
		t,
		"cancel cancelled",
		e.store.Cancel(t.Context(), subject.ID, "again"),
		queue.ErrInvalidTransition,
	)
}

// pinStatusGuards pins ErrInvalidTransition on wrong-source mutations.
func (s *suite) pinStatusGuards(t *testing.T) {
	//art-dupl:accept standard scenario prologue (openEnv + enqueue); independent scenario tests
	e := s.openEnv(t)

	running := e.enqueue(t, task.New[Payload]{Type: "sh"})
	_ = e.claim(t, "w1")

	mustError(
		t,
		"cancel running",
		e.store.Cancel(t.Context(), running.ID, "x"),
		queue.ErrInvalidTransition,
	)
	mustError(
		t,
		"repri running",
		e.store.UpdatePendingPriority(t.Context(), running.ID, 9, "s", "r"),
		queue.ErrInvalidTransition,
	)

	done := e.enqueue(t, task.New[Payload]{Type: "sh"})
	c := e.claim(t, "w1")

	if err := e.store.Complete(t.Context(), done.ID, c.Token, nil); err != nil {
		t.Fatal(err)
	}

	mustError(
		t,
		"cancel completed",
		e.store.Cancel(t.Context(), done.ID, "x"),
		queue.ErrInvalidTransition,
	)
	mustError(
		t,
		"rescue completed",
		e.store.RescueDead(t.Context(), done.ID, 3),
		queue.ErrInvalidTransition,
	)
	mustError(
		t,
		"dismiss completed",
		e.store.DismissDead(t.Context(), done.ID, "r", "op"),
		queue.ErrInvalidTransition,
	)
}
