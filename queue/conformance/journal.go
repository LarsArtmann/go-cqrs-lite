package conformance

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// runJournal pins the journal-first lineage: seq ordering, the
// tail-bounded FactsForTask read, HeadSeq, orphan observation, and the
// consumer watermark/cursor API.
func (s *suite) runJournal(t *testing.T) {
	t.Run("facts append in seq order with HeadSeq", s.pinSeqOrder)
	t.Run("FactsForTask limit returns the most recent, ascending", s.pinTailBound)
	t.Run("Facts after cursor returns strictly greater", s.pinAfterCursor)
	t.Run("orphan observation is idempotent", s.pinOrphaned)
	t.Run("watermark monotonic upsert", s.pinWatermark)
	t.Run("same-tx fact appends commit or roll back together", s.pinFactTx)
}

// pinSeqOrder pins monotonic seqs and HeadSeq agreement.
func (s *suite) pinSeqOrder(t *testing.T) {
	e := s.openEnv(t)

	if head, err := e.store.HeadSeq(t.Context()); err != nil || head != 0 {
		t.Fatalf("empty HeadSeq = %d, %v; want 0", head, err)
	}

	subject := e.enqueue(t, task.New[Payload]{Type: "sh"})
	c := e.claim(t, "w1")

	if err := e.store.Complete(t.Context(), subject.ID, c.Token, nil); err != nil {
		t.Fatal(err)
	}

	all := factsFor(t, e, subject.ID)
	if len(all) != 3 {
		t.Fatalf("facts = %d, want 3", len(all))
	}

	for i := 1; i < len(all); i++ {
		if all[i].Seq <= all[i-1].Seq {
			t.Fatalf("seqs not ascending: %v", seqsOf(all))
		}
	}

	head, err := e.store.HeadSeq(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if head != all[len(all)-1].Seq {
		t.Fatalf("HeadSeq = %d, want the last fact's %d", head, all[len(all)-1].Seq)
	}

	for _, f := range all {
		if f.Time.IsZero() {
			t.Fatal("fact time not assigned")
		}
	}
}

// pinTailBound pins the cross-store catch from the donor: a limit bounds
// to the MOST RECENT n facts, still ascending — not the first n.
func (s *suite) pinTailBound(t *testing.T) {
	//art-dupl:accept scenario prologue idiom (openEnv + enqueue + claim); test boilerplate
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh", MaxAttempts: 1})
	c := e.claim(t, "w1")
	_ = e.store.Fail(t.Context(), subject.ID, c.Token, "boom", 0, nil) // failed + dead-lettered

	all := factsFor(t, e, subject.ID)
	if len(all) < 3 {
		t.Fatalf("need at least 3 facts, have %d", len(all))
	}

	tail, err := e.store.FactsForTask(t.Context(), subject.ID, 2)
	if err != nil {
		t.Fatal(err)
	}

	want := all[len(all)-2:]
	if len(tail) != 2 || tail[0].Seq != want[0].Seq || tail[1].Seq != want[1].Seq {
		t.Fatalf("tail = %v, want most-recent 2 %v", seqsOf(tail), seqsOf(want))
	}
}

// pinAfterCursor pins the tailer read: strictly greater than after, in
// order, bounded by limit.
func (s *suite) pinAfterCursor(t *testing.T) {
	e := s.openEnv(t)

	for range 4 {
		e.enqueue(t, task.New[Payload]{Type: "sh"})
	}

	all, err := e.store.Facts(t.Context(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(all) != 4 {
		t.Fatalf("facts = %d, want 4", len(all))
	}

	after := all[1].Seq

	rest, err := e.store.Facts(t.Context(), after, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(rest) != 2 || rest[0].Seq <= after {
		t.Fatalf("after-cursor read wrong: %v after %d", seqsOf(rest), after)
	}

	bounded, err := e.store.Facts(t.Context(), 0, 2)
	if err != nil || len(bounded) != 2 || bounded[1].Seq != all[1].Seq {
		t.Fatalf("bounded read = %v, want the first two %v", seqsOf(bounded), seqsOf(all[:2]))
	}
}

// pinOrphaned pins MarkOrphaned: observation only, idempotent, task
// stays Running.
func (s *suite) pinOrphaned(t *testing.T) {
	//art-dupl:accept standard scenario prologue (openEnv + enqueue); independent scenario tests
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh"})

	_, err := e.store.ClaimDue(t.Context(), "gone", 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(60 * time.Millisecond)

	n, err := e.store.MarkOrphaned(t.Context(), time.Now())
	if err != nil || n != 1 {
		t.Fatalf("MarkOrphaned = %d, %v; want 1", n, err)
	}

	again, err := e.store.MarkOrphaned(t.Context(), time.Now())
	if err != nil || again != 0 {
		t.Fatalf("second MarkOrphaned = %d, %v; want 0 (idempotent)", again, err)
	}

	got, _ := e.store.Get(t.Context(), subject.ID)
	if got.Status != task.Running {
		t.Fatalf("orphan observation changed state: %s", got.Status)
	}

	f := lastFact(t, e, subject.ID)
	if f.Type != facts.Orphaned || f.Owner != "gone" {
		t.Fatalf("orphaned fact = %+v, want owner gone", f)
	}
}

// pinWatermark pins the consumer-cursor API: unknown → absent, monotonic
// upsert, and seq 0 as a valid checkpoint.
func (s *suite) pinWatermark(t *testing.T) {
	e := s.openEnv(t)

	read := func(consumer string, wantSeq int64, wantExists bool) {
		t.Helper()

		seq, exists, err := e.store.Watermark(t.Context(), consumer)
		if err != nil || exists != wantExists || seq != wantSeq {
			t.Fatalf(
				"watermark %s = %d, %v, %v; want %d/%v",
				consumer, seq, exists, err, wantSeq, wantExists,
			)
		}
	}

	read("bridge", 0, false)

	if err := e.store.SaveWatermark(t.Context(), "bridge", 0); err != nil {
		t.Fatalf("save seq 0: %v", err)
	}

	read("bridge", 0, true)

	for _, save := range []int64{5, 9, 3} {
		if err := e.store.SaveWatermark(t.Context(), "bridge", save); err != nil {
			t.Fatal(err)
		}
	}

	read("bridge", 9, true)

	// Consumers are independent cursors.
	if err := e.store.SaveWatermark(t.Context(), "sweeper", 2); err != nil {
		t.Fatal(err)
	}

	read("sweeper", 2, true)

	// The operator surface lists every cursor.
	all, err := e.store.Watermarks(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(all) != 2 || all["bridge"] != 9 || all["sweeper"] != 2 {
		t.Fatalf("watermarks = %v, want bridge=9 sweeper=2", all)
	}
}
