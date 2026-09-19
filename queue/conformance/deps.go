package conformance

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// minute is the suite's standard lease length.
const minute = time.Minute

// runDeps pins the DAG dependency contract at the enqueue boundary: deps
// must exist (ErrDanglingDep), chains build only in dependency order, and
// the existence validation is the cycle guard — a cycle's closing edge
// always references a task that does not exist yet, so it is rejected as
// dangling rather than deadlocking the queue at claim time.
func (s *suite) runDeps(t *testing.T) {
	t.Run("dangling dep rejected", s.pinDanglingDepRejected)
	t.Run("dep on cancelled task is enqueueable but never unblocks", s.pinDepOnCancelled)
	t.Run("dead dep keeps waiter gated until rescued", s.pinDeadDepGates)
	t.Run("chain builds and drains in order", s.pinChainDrains)
}

// pinDanglingDepRejected pins the validation: an enqueue naming unknown
// dep IDs fails with ErrDanglingDep and leaves NO trace — no task row, no
// fact. The not-yet-existing ID is exactly the shape a dependency cycle's
// closing edge would need, which is why this rejection doubles as cycle
// prevention.
func (s *suite) pinDanglingDepRejected(t *testing.T) {
	e := s.openEnv(t)

	phantom := task.ID("0000000000000000deadbeefdeadbeef")

	_, err := e.store.Enqueue(t.Context(), task.New[Payload]{
		Type: "sh",
		Deps: []task.ID{phantom},
	})
	mustError(t, "enqueue with phantom dep", err, queue.ErrDanglingDep)

	// The rejection is total: nothing persisted, journal untouched.
	counts, err := e.store.StatusCounts(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if total := statusTotal(counts); total != 0 {
		t.Fatalf("status counts = %v after rejected enqueue, want empty store", counts)
	}

	head, err := e.store.HeadSeq(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if head != 0 {
		t.Fatalf("head seq = %d after rejected enqueue, want 0 (no facts)", head)
	}
}

// pinDepOnCancelled pins that validation is existence-only: depending on
// a CANCELLED task is legal (it exists), but the gate only opens on
// completed — a cancelled dep blocks forever, by design, so operators see
// stranded waiters instead of silently running work whose upstream was
// withdrawn.
func (s *suite) pinDepOnCancelled(t *testing.T) {
	e := s.openEnv(t)

	gone := e.enqueue(t, task.New[Payload]{Type: "sh"})
	if err := e.store.Cancel(t.Context(), gone.ID, "upstream withdrawn"); err != nil {
		t.Fatal(err)
	}

	waiter := e.enqueue(t, task.New[Payload]{Type: "sh", Deps: []task.ID{gone.ID}})
	if waiter.ID == "" {
		t.Fatal("enqueue on cancelled dep: no task returned")
	}

	if _, err := e.store.ClaimDue(t.Context(), "w1", minute); !errors.Is(err, queue.ErrNoTaskDue) {
		t.Fatalf("claim with cancelled dep: error = %v, want ErrNoTaskDue", err)
	}
}

// pinDeadDepGates pins the DLQ interaction: a dead dep also blocks
// forever; rescuing it (back to pending) re-opens the waiter's gate.
func (s *suite) pinDeadDepGates(t *testing.T) {
	e := s.openEnv(t)

	blocker := e.enqueue(t, task.New[Payload]{Type: "sh"})
	waiter := e.enqueue(t, task.New[Payload]{Type: "sh", Deps: []task.ID{blocker.ID}})

	c := e.claim(t, "w1")
	if c.Task.ID != blocker.ID {
		t.Fatalf("claimed %s, want the blocker %s", c.Task.ID, blocker.ID)
	}

	if err := e.store.FailPermanent(t.Context(), blocker.ID, c.Token, "poison", nil); err != nil {
		t.Fatal(err)
	}

	if _, err := e.store.ClaimDue(t.Context(), "w1", minute); !errors.Is(err, queue.ErrNoTaskDue) {
		t.Fatalf("claim with dead dep: error = %v, want ErrNoTaskDue (dead is not completed)", err)
	}

	// Rescue re-opens the gate: the dep is pending again, and once it
	// completes the waiter runs.
	if err := e.store.RescueDead(t.Context(), blocker.ID, 1); err != nil {
		t.Fatal(err)
	}

	c = e.claim(t, "w1")
	if c.Task.ID != blocker.ID {
		t.Fatalf("claimed %s after rescue, want the blocker %s", c.Task.ID, blocker.ID)
	}

	if err := e.store.Complete(t.Context(), blocker.ID, c.Token, nil); err != nil {
		t.Fatal(err)
	}

	c = e.claim(t, "w1")
	if c.Task.ID != waiter.ID {
		t.Fatalf("claimed %s, want the waiter %s after blocker completed", c.Task.ID, waiter.ID)
	}
}

// pinChainDrains pins a multi-hop chain under validation: A → B → C
// builds only in dependency order (the reverse order is dangling), and
// the workers drain it strictly head-first.
func (s *suite) pinChainDrains(t *testing.T) {
	e := s.openEnv(t)

	a := e.enqueue(t, task.New[Payload]{Type: "sh"})

	// The cycle-guard shape, pinned directly: B depending on C before C
	// exists is rejected, so B→C→B can never be assembled.
	_, err := e.store.Enqueue(t.Context(), task.New[Payload]{
		Type: "sh",
		Deps: []task.ID{"0000000000000000c0ffee0000000001"},
	})
	mustError(t, "enqueue dep-on-future", err, queue.ErrDanglingDep)

	b := e.enqueue(t, task.New[Payload]{Type: "sh", Deps: []task.ID{a.ID}})
	cc := e.enqueue(t, task.New[Payload]{Type: "sh", Deps: []task.ID{b.ID}})

	for _, want := range []task.ID{a.ID, b.ID, cc.ID} {
		c := e.claim(t, "w1")
		if c.Task.ID != want {
			t.Fatalf("chain order broken: claimed %s, want %s", c.Task.ID, want)
		}

		if err := e.store.Complete(t.Context(), want, c.Token, nil); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := e.store.ClaimDue(t.Context(), "w1", minute); !errors.Is(err, queue.ErrNoTaskDue) {
		t.Fatalf("drained chain still claimable: error = %v, want ErrNoTaskDue", err)
	}
}

// statusTotal sums a StatusCounts result.
func statusTotal(counts map[task.Status]int) int {
	total := 0
	for _, n := range counts {
		total += n
	}

	return total
}
