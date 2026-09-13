package scenario_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/scenario/v4"
)

// equivalenceTaskIDs is the closed key space the generator draws from: a
// small set makes key collisions (last-write-wins fold updates) likely, so
// the property exercises updates and overwrites, not just inserts.
var equivalenceTaskIDs = []string{"t1", "t2", "t3"}

// genEquivalenceEvent draws one task or audit event over the closed key
// space, with random titles/notes/done flags.
func genEquivalenceEvent(rt *rapid.T) event.Event {
	id := rapid.SampledFrom(equivalenceTaskIDs).Draw(rt, "id")

	if rapid.Bool().Draw(rt, "isTask") {
		title := rapid.StringN(1, 8, 16).Draw(rt, "title")
		done := rapid.Bool().Draw(rt, "done")

		return equivalenceEvent(rt, "taskPayload", taskPayload{ID: id, Title: title, Done: done})
	}

	note := rapid.StringN(1, 8, 16).Draw(rt, "note")

	return equivalenceEvent(rt, "auditPayload", auditPayload{ID: "a-" + id, Note: note})
}

// TestProjectionObservationalEquivalence_Property generalizes the
// fixed-sequence regression gate: for EVERY randomized task/audit event
// sequence, projection A's answers are deeply equal whether B is
// interleaved on the same Store or absent. A divergence is a stream-
// isolation bug, never a flaky test (ADR-0136 observational equivalence).
func TestProjectionObservationalEquivalence_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		events := rapid.SliceOfN(rapid.Custom(genEquivalenceEvent), 0, 64).Draw(rt, "events")

		storeA := newEquivalenceStore(rt, false)
		defer storeA.Close()

		storeAB := newEquivalenceStore(rt, true)
		defer storeAB.Close()

		probes := make([]scenario.EquivalenceProbe, len(equivalenceTaskIDs))
		for i, qid := range equivalenceTaskIDs {
			probes[i] = scenario.EquivalenceProbe{
				Baseline:    func() (any, error) { return storeA.Execute(findTask{ID: qid}) },
				Interleaved: func() (any, error) { return storeAB.Execute(findTask{ID: qid}) },
			}
		}

		scenario.AssertObservationalEquivalence(
			rt,
			events,
			projectionadapter.New("tasks-only", storeA, equivalenceDecoder),
			scenario.Interleaved(
				projectionadapter.New("tasks", storeAB, equivalenceDecoder),
				projectionadapter.New("audit", storeAB, equivalenceDecoder),
			),
			probes...,
		)
	})
}
