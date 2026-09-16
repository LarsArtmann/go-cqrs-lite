package conformance

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// runReads pins the read surface: filters, pagination, ordering,
// counting, and status rollups.
func (s *suite) runReads(t *testing.T) {
	t.Run("filters and default order", s.pinListFilters)
	t.Run("pagination", s.pinPagination)
	t.Run("parked and priority bounds", s.pinParkedAndBands)
	t.Run("substring query", s.pinQuery)
	t.Run("status counts", s.pinStatusCounts)
	t.Run("reprioritize pending with facts", s.pinUpdatePriority)
}

// filter is the zero filter.
func filter() queue.Filter { return queue.Filter{} }

// pinListFilters pins status/project/type filters and the default
// priority-descending, age-ascending order.
func (s *suite) pinListFilters(t *testing.T) {
	e := s.openEnv(t)

	p1Email := e.enqueue(t, task.New[Payload]{Project: "p1", Type: "email", Priority: 1})
	p1Shell := e.enqueue(t, task.New[Payload]{Project: "p1", Type: "sh", Priority: 9})
	p2Email := e.enqueue(t, task.New[Payload]{Project: "p2", Type: "email", Priority: 5})

	got := listAll(t, e, queue.Filter{})
	if len(got) != 3 {
		t.Fatalf("listed %d, want 3", len(got))
	}

	// Default order: priority DESC, then age ASC.
	if got[0].ID != p1Shell.ID || got[1].ID != p2Email.ID || got[2].ID != p1Email.ID {
		t.Fatalf("default order = %s,%s,%s; want priority-desc %s,%s,%s",
			got[0].ID, got[1].ID, got[2].ID, p1Shell.ID, p2Email.ID, p1Email.ID)
	}

	proj := "p1"
	typ := "email"
	status := task.Pending

	if got := listAll(t, e, queue.Filter{Project: &proj}); len(got) != 2 {
		t.Fatalf("project filter = %d, want 2", len(got))
	}

	if got := listAll(t, e, queue.Filter{Type: &typ}); len(got) != 2 {
		t.Fatalf("type filter = %d, want 2", len(got))
	}

	if got := listAll(t, e, queue.Filter{Status: &status}); len(got) != 3 {
		t.Fatalf("status filter = %d, want 3", len(got))
	}
}

// pinPagination pins Limit/Offset.
func (s *suite) pinPagination(t *testing.T) {
	e := s.openEnv(t)

	for i := range 5 {
		e.enqueue(t, task.New[Payload]{Project: "page", Type: "sh", Priority: i})
	}

	got := listAll(t, e, queue.Filter{Project: ptr("page"), Limit: 2})
	if len(got) != 2 {
		t.Fatalf("limit = %d, want 2", len(got))
	}

	got2 := listAll(t, e, queue.Filter{Project: ptr("page"), Limit: 2, Offset: 2})
	if len(got2) != 2 || got2[0].ID == got[0].ID || got2[0].ID == got[1].ID {
		t.Fatalf("offset window wrong: page1=%v page2=%v", idsOf(got), idsOf(got2))
	}

	n, err := e.store.CountTasks(t.Context(), queue.Filter{Project: ptr("page")})
	if err != nil || n != 5 {
		t.Fatalf("count = %d, %v; want 5", n, err)
	}
}

// pinParkedAndBands pins the Parked view and stored-priority bounds.
func (s *suite) pinParkedAndBands(t *testing.T) {
	e := s.openEnv(t)

	parked := e.enqueue(t, task.New[Payload]{
		Type:      "sh",
		Project:   "bands",
		NotBefore: time.Now().Add(time.Hour),
	})
	e.enqueue(t, task.New[Payload]{Type: "sh", Project: "bands"})

	got := listAll(t, e, queue.Filter{Project: ptr("bands"), Parked: ptr(true)})
	if len(got) != 1 || got[0].ID != parked.ID {
		t.Fatalf("parked filter = %d, want exactly the future-dated task", len(got))
	}

	hot := e.enqueue(t, task.New[Payload]{Type: "sh", Project: "pri", Priority: 140})
	e.enqueue(t, task.New[Payload]{Type: "sh", Project: "pri", Priority: 20})

	got = listAll(t, e, queue.Filter{Project: ptr("pri"), PriorityMin: ptr(100)})
	if len(got) != 1 || got[0].ID != hot.ID {
		t.Fatalf("priority-min filter = %d, want the 140 task", len(got))
	}

	got = listAll(t, e, queue.Filter{Project: ptr("pri"), PriorityMax: ptr(30)})
	if len(got) != 1 {
		t.Fatalf("priority-max filter = %d, want the one <=30 task", len(got))
	}
}

// pinQuery pins the case-insensitive substring search.
func (s *suite) pinQuery(t *testing.T) {
	e := s.openEnv(t)

	e.enqueue(t, task.New[Payload]{Project: "Alpha-Repo", Type: "sh"})
	e.enqueue(t, task.New[Payload]{Project: "beta", Type: "sh"})

	got := listAll(t, e, queue.Filter{Query: "alpha"})
	if len(got) != 1 || got[0].Project != "Alpha-Repo" {
		t.Fatalf("query = %d results, want the case-insensitive alpha match", len(got))
	}

	if got := listAll(t, e, queue.Filter{Query: "no-such-thing"}); len(got) != 0 {
		t.Fatalf("query miss = %d, want 0", len(got))
	}
}

// pinStatusCounts pins the GROUP BY rollup.
func (s *suite) pinStatusCounts(t *testing.T) {
	e := s.openEnv(t)

	e.enqueue(t, task.New[Payload]{Type: "sh"})
	e.enqueue(t, task.New[Payload]{Type: "sh"})
	third := e.enqueue(t, task.New[Payload]{Type: "sh"})

	_ = e.claim(t, "w1")

	if err := e.store.Cancel(t.Context(), third.ID, "not needed"); err != nil {
		t.Fatal(err)
	}

	counts, err := e.store.StatusCounts(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if counts[task.Pending] != 1 || counts[task.Running] != 1 || counts[task.Cancelled] != 1 {
		t.Fatalf("counts = %v, want one each of pending/running/cancelled", counts)
	}
}

// pinUpdatePriority pins the pending-only, fact-backed, idempotent
// priority update.
func (s *suite) pinUpdatePriority(t *testing.T) {
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh", Priority: 10})

	// Same value: no error, no fact.
	if err := e.store.UpdatePendingPriority(t.Context(), subject.ID, 10, "src", "same"); err != nil {
		t.Fatal(err)
	}

	if got := countFacts(t, e, subject.ID, facts.Reprioritized); got != 0 {
		t.Fatalf("same-value update appended %d facts, want 0", got)
	}

	if err := e.store.UpdatePendingPriority(t.Context(), subject.ID, 70, "marker", "P1"); err != nil {
		t.Fatal(err)
	}

	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.Priority != 70 {
		t.Fatalf("priority = %d, want 70", got.Priority)
	}

	f := lastFact(t, e, subject.ID)
	if f.Type != facts.Reprioritized {
		t.Fatalf("fact = %s, want reprioritized", f.Type)
	}

	for _, want := range []string{`"old_priority":10`, `"new_priority":70`, `"source":"marker"`} {
		if !contains(f.Detail, want) {
			t.Fatalf("reprioritize evidence missing %s: %s", want, f.Detail)
		}
	}
}

// listAll runs List and fails on error.
func listAll(t *testing.T, e *env, f queue.Filter) []task.Task[Payload] {
	t.Helper()

	got, err := e.store.List(t.Context(), f)
	if err != nil {
		t.Fatalf("list %+v: %v", f, err)
	}

	return got
}

// idsOf extracts the ID slice.
func idsOf(tasks []task.Task[Payload]) []task.ID {
	out := make([]task.ID, len(tasks))
	for i, subject := range tasks {
		out[i] = subject.ID
	}

	return out
}

// ptr returns a pointer to v — filter-field sugar.
func ptr[T any](v T) *T { return new(v) }
