package enginetest

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// counterKeyEvent is the payload used by RunCounterKeyCompositeTest: a
// project-local status transition (pending → running).
type counterKeyEvent struct {
	Project string
	From    string
	At      int64
}

type counterKeyListIn struct{}

type counterKeyRow struct {
	ID     string
	Status string
}

type counterKeyCountIn struct{}

// RunCounterKeyCompositeTest pins the composite-shaped counter-key contract
// (the tq row-683 class): a counter fold that emits Delta keys containing a
// separator ("api|pending") must store them VERBATIM — never truncated to a
// parent-aggregate row ("api|"). Both fold shapes are exercised: the
// seed+transition pair (net "api|pending"=0, "api|running"=1) and a
// from-only delta on a second project (net "solo|pending"=-1).
//
// Engines that do not implement [metaengine.CounterBackend] skip before any
// fold runs: a counter query on a non-CounterBackend engine errors by
// design, so the capability assertion must precede Plan/ApplyRecord. The
// caller is responsible for closing the engine.
func RunCounterKeyCompositeTest(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	cb, ok := eng.(metaengine.CounterBackend)
	if !ok {
		t.Skipf("engine %s does not implement CounterBackend — no counter store to inspect, class structurally out of reach", engineName(eng))
	}

	countsName := ScopedCollection("counterkey-counts-" + engineName(eng))

	tasksQuery := metaengine.Query[counterKeyListIn, counterKeyRow](
		ScopedCollection("counterkey-tasks-" + engineName(eng)),
		metaengine.OnRecordTyped("seed", counterKeyEvent{}, func(_ record.Record, e counterKeyEvent) (int64, counterKeyRow) {
			return e.At, counterKeyRow{ID: e.Project, Status: "pending"}
		}),
		metaengine.OnRecordTyped("T", counterKeyEvent{}, func(_ record.Record, e counterKeyEvent, prev counterKeyRow) counterKeyRow {
			prev.Status = "running"

			return prev
		}),
		metaengine.WithColumnarLayout(),
	)

	countsQuery := metaengine.Query[counterKeyCountIn, map[string]int64](
		countsName,
		metaengine.OnRecordTyped("seed", counterKeyEvent{}, func(_ record.Record, e counterKeyEvent) metaengine.Delta {
			return metaengine.Delta{e.Project + "|pending": 1}
		}),
		metaengine.OnRecordTyped("T", counterKeyEvent{}, func(_ record.Record, e counterKeyEvent) metaengine.Delta {
			return metaengine.Delta{e.Project + "|running": 1, e.Project + "|" + e.From: -1}
		}),
	)

	st, err := metaengine.Plan([]metaengine.Engine{eng}, tasksQuery, countsQuery)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	ctx := context.Background()

	if err := st.ApplyRecord(ctx, record.Record{Type: "seed"}, counterKeyEvent{Project: "api", At: 1}); err != nil {
		t.Fatalf("ApplyRecord seed: %v", err)
	}

	if err := st.ApplyRecord(
		ctx,
		record.Record{Type: "T"},
		counterKeyEvent{Project: "api", From: "pending", At: 2},
	); err != nil {
		t.Fatalf("ApplyRecord transition: %v", err)
	}

	got, err := cb.CounterGet(ctx, countsName)
	if err != nil {
		t.Fatalf("CounterGet: %v", err)
	}

	for _, key := range []string{"api|", "api|pending", "api|running"} {
		want := int64(0)
		switch key {
		case "api|pending":
			want = 0
		case "api|running":
			want = 1
		}
		if key == "api|" && got[key] != 0 {
			t.Fatalf("truncated parent row present: %q = %d (counters: %v)", key, got[key], got)
		}
		if key != "api|" && got[key] != want {
			t.Fatalf("composite key %q = %d, want %d (counters: %v)", key, got[key], want, got)
		}
	}

	fromOnly, err := metaengine.Plan([]metaengine.Engine{eng}, countsQuery)
	if err != nil {
		t.Fatalf("metaengine.Plan (from-only store): %v", err)
	}

	if err := fromOnly.ApplyRecord(
		ctx,
		record.Record{Type: "T"},
		counterKeyEvent{Project: "solo", From: "pending", At: 3},
	); err != nil {
		t.Fatalf("ApplyRecord from-only delta: %v", err)
	}

	got2, err := cb.CounterGet(ctx, countsName)
	if err != nil {
		t.Fatalf("CounterGet (from-only): %v", err)
	}

	if got2["solo|"] != 0 {
		t.Fatalf("truncated parent row present after from-only delta: %v", got2)
	}

	if got2["solo|pending"] != -1 {
		t.Fatalf("from-side composite key %q = %d, want -1 (counters: %v)", "solo|pending", got2["solo|pending"], got2)
	}
}
