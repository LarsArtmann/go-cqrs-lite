package conformance

import (
	"bytes"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// factTypes lists one task's fact types in Seq order.
func factTypes(t *testing.T, e *env, id task.ID) []facts.FactType {
	t.Helper()

	all := factsFor(t, e, id)
	out := make([]facts.FactType, len(all))
	for i, f := range all {
		out[i] = f.Type
	}

	return out
}

// factsFor reads one task's full fact trail (unbounded).
func factsFor(t *testing.T, e *env, id task.ID) []facts.Fact {
	t.Helper()

	all, err := e.store.FactsForTask(t.Context(), id, 0)
	if err != nil {
		t.Fatalf("facts for %s: %v", id, err)
	}

	return all
}

// lastFact returns a task's most recent fact.
func lastFact(t *testing.T, e *env, id task.ID) facts.Fact {
	t.Helper()

	all := factsFor(t, e, id)
	if len(all) == 0 {
		t.Fatalf("no facts for %s", id)
	}

	return all[len(all)-1]
}

// countFacts counts one task's facts of one type.
func countFacts(t *testing.T, e *env, id task.ID, want facts.FactType) int {
	t.Helper()

	n := 0
	for _, f := range factsFor(t, e, id) {
		if f.Type == want {
			n++
		}
	}

	return n
}

// contains reports whether the detail bytes contain the reason string.
func contains(detail []byte, reason string) bool {
	return bytes.Contains(detail, []byte(reason))
}

// equalFactTypes compares two fact-type slices for exact equality.
func equalFactTypes(got, want []facts.FactType) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}

// seqsOf extracts the Seq slice — failure-message sugar.
func seqsOf(all []facts.Fact) []int64 {
	out := make([]int64, len(all))
	for i, f := range all {
		out[i] = f.Seq
	}

	return out
}
