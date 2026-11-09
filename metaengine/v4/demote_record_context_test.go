package metaengine

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// TestDemoteEngine_RecordContextReplay pins the catch-up record-context
// contract: both DemoteEngine replays hand the recorded EventInput.Record to
// record-aware folds instead of a synthesized Type-only record — the mirror
// catch-up onto the demoted engine (replayToShadow) AND the re-routed
// catch-up onto the survivor (applyReplay).
func TestDemoteEngine_RecordContextReplay(t *testing.T) {
	t.Parallel()

	items := costShaped("items", map[ReadPattern]float64{ReadAggregate: 1_000_000})
	counts := costShaped("counts", map[ReadPattern]float64{ReadPointLookup: 1_000_000})

	store, err := Plan([]Engine{items, counts}, recordContextQuery(), roleCountQuery())
	if err != nil {
		t.Fatal(err)
	}

	WithEventLog(store, NewEventLog())
	t.Cleanup(func() { _ = store.Close() })

	for _, qa := range store.Plan().Queries {
		switch qa.QueryName {
		case "record_context_tasks":
			if qa.EngineName != "items" {
				t.Fatalf("record_context_tasks routed to %q, want items", qa.EngineName)
			}
		case "role_counts":
			if qa.EngineName != "counts" {
				t.Fatalf("role_counts routed to %q, want counts", qa.EngineName)
			}
		}
	}

	ctx := context.Background()

	if err := store.ApplyRecord(ctx, record.Record{
		Type:     "recordContextEvent",
		StreamID: "Task/01JCTX",
		Version:  7,
	}, recordContextEvent{TaskID: "t1"}); err != nil {
		t.Fatalf("ApplyRecord: %v", err)
	}

	for range 5 {
		if err := store.Apply(
			ctx,
			"roleItemCreated",
			roleItemCreated{ID: "x", Name: "n"},
		); err != nil {
			t.Fatalf("Apply counter event: %v", err)
		}
	}

	// Demote the counter engine. The record-aware map query is never served
	// by counts, so its history catches up on the demoted mirror via
	// replayToShadow; the counter query re-routes to items and catches up
	// via applyReplay. Force is required because the Delta fold is
	// non-idempotent (the receiving projection is empty — same contract as
	// TestDemoteEngine_NonIdempotentGuard).
	if err := store.DemoteEngine(ctx, "counts", WithDemoteForce()); err != nil {
		t.Fatalf("demote counts: %v", err)
	}

	mb, ok := counts.(MapBackend)
	if !ok {
		t.Fatal("counts engine is not a MapBackend")
	}

	raw, found, err := mb.MapGet(ctx, "record_context_tasks", "t1")
	if err != nil || !found {
		t.Fatalf("mirror MapGet: found=%v err=%v", found, err)
	}

	view, err := reify[recordContextView](raw)
	if err != nil {
		t.Fatalf("reify: %v", err)
	}

	if view.StreamID != "Task/01JCTX" || view.Version != 7 {
		t.Fatalf("mirror catch-up fed partial record context: %+v", view)
	}

	cb, ok := items.(CounterBackend)
	if !ok {
		t.Fatal("items engine is not a CounterBackend")
	}

	if m, _ := cb.CounterGet(ctx, "role_counts"); m["created"] != 5 {
		t.Fatalf("re-routed counter catch-up wrong: %#v", m)
	}
}
