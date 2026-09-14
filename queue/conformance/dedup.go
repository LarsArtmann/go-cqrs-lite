package conformance

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// runDedup pins idempotent enqueue: a DedupKey makes Enqueue convergent —
// the FIRST task under a key wins forever, in any status, and keys
// without a match never collide.
func (s *suite) runDedup(t *testing.T) {
	t.Run("same key returns the stored task unchanged", s.pinDedupReturnsStored)
	t.Run("terminal key still suppresses", s.pinDedupTerminal)
	t.Run("no key enqueues freely", s.pinNoKey)
}

// pinDedupReturnsStored pins the convergence: same key, different
// content — the stored task comes back, no new row, no new fact.
func (s *suite) pinDedupReturnsStored(t *testing.T) {
	e := s.openEnv(t)

	first := e.enqueue(t, task.New[Payload]{
		Type:     "sh",
		Payload:  Payload{Cmd: "first"},
		DedupKey: "harvest:repo:1",
	})

	again := e.enqueue(t, task.New[Payload]{
		Type:     "sh",
		Payload:  Payload{Cmd: "SECOND"},
		DedupKey: "harvest:repo:1",
	})

	if again.ID != first.ID {
		t.Fatalf("dedup returned new task %s, want stored %s", again.ID, first.ID)
	}

	if again.Payload.Cmd != "first" {
		t.Fatalf("dedup mutated content: payload = %q, want the stored first", again.Payload.Cmd)
	}

	if got := countFacts(t, e, first.ID, facts.Enqueued); got != 1 {
		t.Fatalf("enqueued facts = %d, want 1 (no duplicate fact)", got)
	}
}

// pinDedupTerminal pins the forever-part: a cancelled (or dead) task's
// key still suppresses — suppression is keyed on existence, not status.
func (s *suite) pinDedupTerminal(t *testing.T) {
	e := s.openEnv(t)

	first := e.enqueue(t, task.New[Payload]{Type: "sh", DedupKey: "k1"})
	if err := e.store.Cancel(t.Context(), first.ID, "done elsewhere"); err != nil {
		t.Fatal(err)
	}

	again := e.enqueue(t, task.New[Payload]{Type: "sh", DedupKey: "k1"})
	if again.ID != first.ID || again.Status != task.Cancelled {
		t.Fatalf("terminal key re-enqueued: got %+v, want the stored cancelled %s", again, first.ID)
	}

	dead := e.enqueue(t, task.New[Payload]{Type: "sh", MaxAttempts: 1, DedupKey: "k2"})
	c := e.claim(t, "w1")
	if c.Task.ID != dead.ID {
		t.Fatalf("claimed %s, want %s", c.Task.ID, dead.ID)
	}

	if err := e.store.Fail(t.Context(), dead.ID, "w1", "boom", 0, nil); err != nil {
		t.Fatal(err)
	}

	deadAgain := e.enqueue(t, task.New[Payload]{Type: "sh", DedupKey: "k2"})
	if deadAgain.ID != dead.ID || deadAgain.Status != task.Dead {
		t.Fatalf("dead key re-enqueued: got %+v, want the stored dead %s", deadAgain, dead.ID)
	}
}

// pinNoKey pins that tasks WITHOUT a key never collide: two identical
// templates produce two tasks.
func (s *suite) pinNoKey(t *testing.T) {
	e := s.openEnv(t)

	tk := task.New[Payload]{Type: "sh", Payload: Payload{Cmd: "same"}}
	one := e.enqueue(t, tk)
	two := e.enqueue(t, tk)

	if one.ID == two.ID {
		t.Fatal("keyless double enqueue collapsed into one task")
	}

	n, err := e.store.CountTasks(t.Context(), filter())
	if err != nil || n != 2 {
		t.Fatalf("count = %d, %v; want 2 tasks", n, err)
	}
}
