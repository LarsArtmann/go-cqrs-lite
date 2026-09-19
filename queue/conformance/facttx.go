package conformance

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// pinFactTx pins the same-tx fact-append capability (queue.FactTx): a
// failing fn rolls every sink append back; a succeeding fn lands all of
// them with ascending seqs, visible through the ordinary journal reads.
func (s *suite) pinFactTx(t *testing.T) {
	e := s.openEnv(t)

	subject := e.enqueue(t, task.New[Payload]{Type: "sh"})

	before, err := e.store.HeadSeq(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	ftx, ok := e.store.(queue.FactTx)
	if !ok {
		t.Fatal("store does not implement queue.FactTx (mandatory for engines)")
	}

	// Rollback: fn appends two facts, then fails — neither may survive.
	boom := errors.New("observation abandoned")
	err = ftx.WithFacts(t.Context(), func(sink queue.FactSink) error {
		for i, typ := range []string{"note.one", "note.two"} {
			if err := sink.Append(t.Context(), facts.Fact{
				TaskID: subject.ID.String(),
				Type:   facts.FactType(typ),
				Detail: []byte{byte('0' + i)},
			}); err != nil {
				t.Fatal(err)
			}
		}

		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("WithFacts error = %v, want the fn error propagated", err)
	}

	afterRollback, err := e.store.HeadSeq(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if afterRollback != before {
		t.Fatalf("rolled-back appends survived: head %d -> %d", before, afterRollback)
	}

	// Commit: both land, in order, readable via Facts.
	err = ftx.WithFacts(t.Context(), func(sink queue.FactSink) error {
		for _, typ := range []string{"note.three", "note.four"} {
			if err := sink.Append(t.Context(), facts.Fact{
				TaskID: subject.ID.String(),
				Type:   facts.FactType(typ),
			}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	tail, err := e.store.Facts(t.Context(), before, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(tail) != 2 ||
		tail[0].Type != facts.FactType("note.three") ||
		tail[1].Type != facts.FactType("note.four") ||
		tail[0].Seq >= tail[1].Seq {
		t.Fatalf("committed appends wrong: %v", seqsOf(tail))
	}

	for _, f := range tail {
		if f.Time.IsZero() {
			t.Fatal("sink appends must be time-stamped by the journal")
		}
	}
}
