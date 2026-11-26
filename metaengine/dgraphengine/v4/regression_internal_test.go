package dgraphengine

import (
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Two new counters whose keys differ only in a character blank-node labels
// cannot carry must stay two nodes.
func TestCounterIncrementMutation_DistinctLabelsForCollidingKeys(t *testing.T) {
	t.Parallel()

	deltas := metaengine.Delta{"a-b": int64(1), "a_b": int64(1), "a/b": int64(7), "c": int64(1)}
	muts := counterIncrementMutation("cs", deltas, map[string]counterRow{})

	if len(muts) != len(deltas) {
		t.Fatalf("want one mutation object per delta, got %d", len(muts))
	}

	uids := map[string]bool{}
	keys := map[string]any{}

	for _, m := range muts {
		uid, _ := m["uid"].(string)
		if !strings.HasPrefix(uid, "_:new_") {
			t.Fatalf("new counter must use a blank node, got uid %q", uid)
		}

		if uids[uid] {
			t.Fatalf("blank-node label %q used twice", uid)
		}

		uids[uid] = true
		keys[m["cqrs.counter_key"].(string)] = m["cqrs.counter_value"]
	}

	if len(keys) != len(deltas) {
		t.Fatalf("want %d distinct keys, got %v", len(deltas), keys)
	}
}

// Existing counters are updated in place, and a batch mixing known and new
// keys never reuses a label.
func TestCounterIncrementMutation_UpdatesExistingInPlace(t *testing.T) {
	t.Parallel()

	deltas := metaengine.Delta{"old": int64(2), "new": int64(5)}
	muts := counterIncrementMutation("cs", deltas, map[string]counterRow{
		"old": {uid: "0x1", value: 10},
	})

	for _, m := range muts {
		if key := m["cqrs.counter_key"]; key != nil {
			if m["uid"] != "_:new_0" {
				t.Fatalf("new key %v must get the first free label, got %v", key, m["uid"])
			}

			continue
		}

		if m["uid"] != "0x1" || m["cqrs.counter_value"] != int64(12) {
			t.Fatalf("existing counter must be updated on its uid: %v", m)
		}
	}
}

// A dropped edge predicate must not stay marked as applied, otherwise
// ensureEdgeSchema skips recreating it.
func TestForgetEdgeSchema_DropsOnlyTheGivenPredicate(t *testing.T) {
	t.Parallel()

	e := &dgraphEngine{appliedSchemas: map[string]bool{
		"cqrs.edge.a": true, "cqrs.edge.b": true, "vector": true,
	}}

	e.forgetEdgeSchema("cqrs.edge.a")

	remaining := map[string]bool{}
	for k := range e.appliedSchemas {
		remaining[k] = true
	}

	want := map[string]bool{"cqrs.edge.b": true, "vector": true}
	if len(remaining) != len(want) {
		t.Fatalf("want %v, got %v", want, remaining)
	}

	for k := range want {
		if !remaining[k] {
			t.Fatalf("want %v, got %v", want, remaining)
		}
	}
}
