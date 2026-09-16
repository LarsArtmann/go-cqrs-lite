package conformance

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// runClaims pins the claim semantics: exclusivity under concurrency, the
// crash-reclaim story, and the ordering rules (priority, delay, deps,
// aging) — the heart of what makes the queue a queue.
func (s *suite) runClaims(t *testing.T) {
	t.Run("concurrent claimers never overlap", s.pinFencing)
	t.Run("expiry reclaim re-opens a crashed claim", s.pinExpiryReclaim)
	t.Run("priority orders claims", s.pinPriorityOrder)
	t.Run("not-before gates claiming", s.pinDelayGating)
	t.Run("dependency gating", s.pinDepGating)
	t.Run("aging reorders within the cap", s.pinAging)
	t.Run("stored priority never mutates", s.pinStoredPriorityStable)
}

// pinFencing stress-tests claim exclusivity: N goroutines drain M tasks;
// every task is claimed by exactly one worker.
func (s *suite) pinFencing(t *testing.T) {
	e := s.openEnv(t)

	const tasks = 24

	ids := make(map[task.ID]bool, tasks)
	for range tasks {
		subject := e.enqueue(t, task.New[Payload]{Type: "sh"})
		ids[subject.ID] = true
	}

	const workers = 8

	claimed := make(chan task.ID, tasks)

	var wg sync.WaitGroup

	for w := range workers {
		wg.Go(func() {
			for {
				c, err := e.store.ClaimDue(t.Context(), claimer(w), time.Minute)
				if errors.Is(err, queue.ErrNoTaskDue) {
					return
				}

				if err != nil {
					t.Errorf("worker %d claim: %v", w, err)

					return
				}

				claimed <- c.Task.ID
			}
		})
	}

	wg.Wait()
	close(claimed)

	seen := make(map[task.ID]int, tasks)
	for id := range claimed {
		seen[id]++
	}

	if len(seen) != tasks {
		t.Fatalf("distinct claimed = %d, want %d", len(seen), tasks)
	}

	for id, n := range seen {
		if n != 1 {
			t.Fatalf("task %s claimed %d times — fencing broken", id, n)
		}
	}
}

// claimer names worker w deterministically.
func claimer(w int) string {
	return "w" + string(rune('0'+w))
}

// pinExpiryReclaim pins the crash story: a lease that lapses re-opens
// the task, and the journal explains the owner change with a Released
// fact for the previous holder.
func (s *suite) pinExpiryReclaim(t *testing.T) {
	//art-dupl:accept standard scenario prologue (openEnv + enqueue); independent scenario tests
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh"})

	_, err := e.store.ClaimDue(t.Context(), "crashed", 40*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	// Before expiry, the lease fences everyone.
	if _, err := e.store.ClaimDue(
		t.Context(),
		"w2",
		time.Minute,
	); !errors.Is(
		err,
		queue.ErrNoTaskDue,
	) {
		t.Fatalf("pre-expiry claim: error = %v, want ErrNoTaskDue", err)
	}

	time.Sleep(80 * time.Millisecond)

	c := e.claim(t, "w2")
	if c.Task.ID != subject.ID {
		t.Fatalf("reclaim claimed %s, want %s", c.Task.ID, subject.ID)
	}

	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.Status != task.Running || got.LeaseOwner != "w2" || got.Attempts != 0 {
		t.Fatalf("reclaimed state wrong: %+v", got)
	}

	types := factTypes(t, e, subject.ID)

	want := []facts.FactType{facts.Enqueued, facts.Claimed, facts.Released, facts.Claimed}
	if !equalFactTypes(types, want) {
		t.Fatalf("fact trail = %v, want %v (released then re-claimed)", types, want)
	}

	for _, f := range factsFor(t, e, subject.ID) {
		if f.Type == facts.Released && f.Owner != "crashed" {
			t.Fatalf("released fact owner = %q, want the previous holder", f.Owner)
		}
	}
}

// pinPriorityOrder pins that the highest stored priority is claimed
// first, breaking ties by age.
func (s *suite) pinPriorityOrder(t *testing.T) {
	e := s.openEnv(t)

	low := e.enqueue(t, task.New[Payload]{Type: "sh", Priority: -5})
	high := e.enqueue(t, task.New[Payload]{Type: "sh", Priority: 5})

	c := e.claim(t, "w1")
	if c.Task.ID != high.ID {
		t.Fatalf("claimed %s, want the priority-5 task %s", c.Task.ID, high.ID)
	}

	if err := e.store.Complete(t.Context(), high.ID, "w1", nil); err != nil {
		t.Fatal(err)
	}

	c = e.claim(t, "w1")
	if c.Task.ID != low.ID {
		t.Fatalf("claimed %s, want the priority--5 task %s", c.Task.ID, low.ID)
	}
}

// pinDelayGating pins NotBefore: a future-dated task is not claimable
// until its time arrives.
func (s *suite) pinDelayGating(t *testing.T) {
	e := s.openEnv(t)

	future := e.enqueue(t, task.New[Payload]{
		Type:      "sh",
		NotBefore: time.Now().Add(time.Minute),
	})
	ready := e.enqueue(t, task.New[Payload]{Type: "sh"})

	c := e.claim(t, "w1")
	if c.Task.ID != ready.ID {
		t.Fatalf("claimed %s, want the ready task (future must stay gated)", c.Task.ID)
	}

	// Backdate the delay into the past via a short one and real time.
	parked := e.enqueue(
		t,
		task.New[Payload]{Type: "sh", NotBefore: time.Now().Add(50 * time.Millisecond)},
	)
	time.Sleep(120 * time.Millisecond)

	c = e.claim(t, "w1")
	if c.Task.ID != parked.ID {
		t.Fatalf("claimed %s, want the now-due parked task %s", c.Task.ID, parked.ID)
	}

	if _, err := e.store.Get(t.Context(), future.ID); err != nil {
		t.Fatal(err)
	}
}

// pinDepGating pins DAG semantics: a waiter is not claimable until every
// dependency completed.
func (s *suite) pinDepGating(t *testing.T) {
	e := s.openEnv(t)

	blocker := e.enqueue(t, task.New[Payload]{Type: "sh"})
	waiter := e.enqueue(t, task.New[Payload]{Type: "sh", Deps: []task.ID{blocker.ID}})

	c := e.claim(t, "w1")
	if c.Task.ID != blocker.ID {
		t.Fatalf("claimed dependent %s before its blocker — DAG gating broken", c.Task.ID)
	}

	if err := e.store.Complete(t.Context(), blocker.ID, "w1", nil); err != nil {
		t.Fatal(err)
	}

	c = e.claim(t, "w1")
	if c.Task.ID != waiter.ID {
		t.Fatalf("after blocker completed, claimed %s, want the waiter %s", c.Task.ID, waiter.ID)
	}
}

// pinAging pins the aging rule with the harness's backdate hook: age
// earns priority at PriorityAgingDaysPerPoint, capped at
// PriorityAgingMaxBonus — the cap keeps a decisively stronger fresh task
// ahead of any age.
func (s *suite) pinAging(t *testing.T) {
	e := s.openEnv(t)

	older := e.enqueue(t, task.New[Payload]{Type: "sh", Priority: 55})
	newer := e.enqueue(t, task.New[Payload]{Type: "sh", Priority: 60})

	// 45 days of age: +15 uncapped, clamped to 10 → effective 65 beats 60.
	s.h.Backdate(t, e.store, older.ID, 45*24*time.Hour)

	c := e.claim(t, "w1")
	if c.Task.ID != older.ID {
		t.Fatalf(
			"aging did not flip order: claimed %s, want older %s over newer %s",
			c.Task.ID,
			older.ID,
			newer.ID,
		)
	}

	if err := e.store.Complete(t.Context(), older.ID, "w1", nil); err != nil {
		t.Fatal(err)
	}

	// Leave nothing behind: newer stays pending — cancel it (completing
	// an unclaimed task is a lease error, not cleanup).
	if err := e.store.Cancel(t.Context(), newer.ID, "aging cleanup"); err != nil {
		t.Fatal(err)
	}

	// The cap: 300 days would be +100 uncapped; the bonus holds at 10
	// (50+10 < 65) and the stronger fresh task still wins.
	ancient := e.enqueue(t, task.New[Payload]{Type: "sh", Priority: 50})
	stronger := e.enqueue(t, task.New[Payload]{Type: "sh", Priority: 65})

	s.h.Backdate(t, e.store, ancient.ID, 300*24*time.Hour)

	c = e.claim(t, "w1")
	if c.Task.ID != stronger.ID {
		t.Fatalf("aging bonus not capped: claimed %s, want %s", c.Task.ID, stronger.ID)
	}
}

// pinStoredPriorityStable pins that aging and claims never mutate the
// stored priority — aging is scheduling, not state.
func (s *suite) pinStoredPriorityStable(t *testing.T) {
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh", Priority: 42})
	s.h.Backdate(t, e.store, subject.ID, 60*24*time.Hour)

	c := e.claim(t, "w1")
	if c.Task.Priority != 42 {
		t.Fatalf("claimed priority = %d, want stored 42 (aging must not mutate)", c.Task.Priority)
	}

	if err := e.store.Heartbeat(t.Context(), subject.ID, "w1", time.Minute); err != nil {
		t.Fatal(err)
	}

	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.Priority != 42 {
		t.Fatalf("stored priority = %d after heartbeat, want 42", got.Priority)
	}
}
