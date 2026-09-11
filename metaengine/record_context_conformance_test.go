package metaengine

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// conformWaitDeadline bounds polling for asynchronous shadow replication in
// the conformance sweep (same budget as the replicator tests).
const conformWaitDeadline = 5 * time.Second

// TestFoldDispatch_RecordContextConformance is the entry-point conformance
// sweep for the record-context contract: EVERY fold-dispatch path that can
// reach an OnRecord fold is walked once, asserting (a) which record the fold
// observed — full context or the synthesized Type-only form — and (b) whether
// the synthetic-apply advisory counted it. This kills the spot-test
// whack-a-mole the Demote mirror-leg gap (and the encoded-apply gap after it)
// exposed: a new dispatch path that breaks the contract fails this table by
// omission or assertion, not by someone remembering to spot-test it.
//
// Contract reference (EventInput, Store.Apply, Doctor "--- Record context ---"):
// applies carrying a Record feed it through as-is; applies without one are
// counted by the advisory; replays (Backfill/Verify/Demote/replication) honor
// the recorded Record and never count the advisory.
func TestFoldDispatch_RecordContextConformance(t *testing.T) {
	t.Parallel()

	const (
		wantFullStream  = "Task/CONF"
		wantFullVersion = int64(7)
	)

	fullRecord := record.Record{Type: "recordContextEvent", StreamID: wantFullStream, Version: wantFullVersion}

	syntheticView := recordContextView{TaskID: "t1"}
	fullView := recordContextView{TaskID: "t1", StreamID: wantFullStream, Version: wantFullVersion}

	cases := []struct {
		name     string
		want     recordContextView
		advisory uint64
		run      func(t *testing.T) (*Store, recordContextView)
	}{
		{
			name: "Apply", want: syntheticView, advisory: 1,
			run: func(t *testing.T) (*Store, recordContextView) {
				store := newRecordContextStore(t)
				applyOK(t, store.Apply(context.Background(), "recordContextEvent", recordContextEvent{TaskID: "t1"}))
				return store, conformView(t, store.engines[0])
			},
		},
		{
			name: "ApplyIdempotent", want: syntheticView, advisory: 1,
			run: func(t *testing.T) (*Store, recordContextView) {
				store := newRecordContextStore(t)
				applyOK(t, store.ApplyIdempotent(
					context.Background(), "evt-1", "recordContextEvent", recordContextEvent{TaskID: "t1"},
				))
				return store, conformView(t, store.engines[0])
			},
		},
		{
			name: "ApplyBatch synthetic", want: syntheticView, advisory: 1,
			run: func(t *testing.T) (*Store, recordContextView) {
				store := newRecordContextStore(t)
				applyOK(t, store.ApplyBatch(context.Background(), []EventInput{
					{Type: "recordContextEvent", Payload: recordContextEvent{TaskID: "t1"}},
				}))
				return store, conformView(t, store.engines[0])
			},
		},
		{
			name: "ApplyBatch record", want: fullView, advisory: 0,
			run: func(t *testing.T) (*Store, recordContextView) {
				store := newRecordContextStore(t)
				applyOK(t, store.ApplyBatch(context.Background(), []EventInput{
					{Type: "recordContextEvent", Payload: recordContextEvent{TaskID: "t1"}, Record: fullRecord},
				}))
				return store, conformView(t, store.engines[0])
			},
		},
		{
			name: "ApplyRecord", want: fullView, advisory: 0,
			run: func(t *testing.T) (*Store, recordContextView) {
				store := newRecordContextStore(t)
				applyOK(t, store.ApplyRecord(context.Background(), fullRecord, recordContextEvent{TaskID: "t1"}))
				return store, conformView(t, store.engines[0])
			},
		},
		{
			name: "ApplyEncoded", want: syntheticView, advisory: 1,
			run: func(t *testing.T) (*Store, recordContextView) {
				store := newRecordContextStore(t)
				WithEventLog(store, NewEventLog())
				applyOK(t, store.ApplyEncoded(context.Background(), "recordContextEvent", []byte(`{"TaskID":"t1"}`)))
				view := conformView(t, store.engines[0])

				// Encoded applies are EventLog citizens: a replay from the log
				// rebuilds the row by decoding the raw payload per fold.
				mb := store.engines[0].(MapBackend)
				if err := mb.MapDelete(context.Background(), "record_context_tasks", "t1"); err != nil {
					t.Fatalf("MapDelete: %v", err)
				}

				applyOK(t, store.Backfill(context.Background()))
				if got := conformView(t, store.engines[0]); got != view {
					t.Fatalf("Backfill did not rebuild the encoded apply identically: %+v vs %+v", got, view)
				}

				return store, view
			},
		},
		{
			name: "ApplyEncodedRecord", want: fullView, advisory: 0,
			run: func(t *testing.T) (*Store, recordContextView) {
				store := newRecordContextStore(t)
				applyOK(t, store.ApplyEncodedRecord(context.Background(), fullRecord, []byte(`{"TaskID":"t1"}`)))
				return store, conformView(t, store.engines[0])
			},
		},
		{
			name: "Backfill primary replay", want: fullView, advisory: 0,
			run: func(t *testing.T) (*Store, recordContextView) {
				store := newRecordContextStore(t)
				WithEventLog(store, NewEventLog())
				applyOK(t, store.ApplyRecord(context.Background(), fullRecord, recordContextEvent{TaskID: "t1"}))

				if err := store.engines[0].(MapBackend).
					MapDelete(context.Background(), "record_context_tasks", "t1"); err != nil {
					t.Fatalf("MapDelete: %v", err)
				}

				applyOK(t, store.Backfill(context.Background()))
				return store, conformView(t, store.engines[0])
			},
		},
		{
			name: "Backfill shadow replay (replayShadows)", want: fullView, advisory: 0,
			run: func(t *testing.T) (*Store, recordContextView) {
				store := newRecordContextStore(t)
				WithEventLog(store, NewEventLog())
				applyOK(t, store.ApplyRecord(context.Background(), fullRecord, recordContextEvent{TaskID: "t1"}))

				shadow := renamed("conf_shadow")
				applyOK(t, store.AddEngine(context.Background(), shadow, WithEngineRole(RoleBackup)))
				applyOK(t, store.Backfill(context.Background()))

				return store, conformView(t, shadow)
			},
		},
		{
			name: "Verify fresh-store replay", want: fullView, advisory: 0,
			run: func(t *testing.T) (*Store, recordContextView) {
				store := newRecordContextStore(t)
				WithEventLog(store, NewEventLog())
				applyOK(t, store.ApplyRecord(context.Background(), fullRecord, recordContextEvent{TaskID: "t1"}))

				fresh := NewMemoryEngine()
				applyOK(t, store.Verify(context.Background(), []Engine{fresh}))

				return store, conformView(t, fresh)
			},
		},
		{
			name: "DemoteEngine mirror leg (replayToShadow)", want: fullView, advisory: 0,
			run: func(t *testing.T) (*Store, recordContextView) {
				items := costShaped("conf_items", map[ReadPattern]float64{ReadAggregate: 1_000_000})
				counts := costShaped("conf_counts", map[ReadPattern]float64{ReadPointLookup: 1_000_000})

				store, err := Plan([]Engine{items, counts}, recordContextQuery(), roleCountQuery())
				if err != nil {
					t.Fatal(err)
				}

				WithEventLog(store, NewEventLog())
				t.Cleanup(func() { _ = store.Close() })

				applyOK(t, store.ApplyRecord(context.Background(), fullRecord, recordContextEvent{TaskID: "t1"}))
				applyOK(t, store.DemoteEngine(context.Background(), "conf_counts", WithDemoteForce()))

				return store, conformView(t, counts)
			},
		},
		{
			name: "DemoteEngine re-routed leg (applyReplay)", want: fullView, advisory: 0,
			run: func(t *testing.T) (*Store, recordContextView) {
				// Same routing split as the mirror-leg case — the record query
				// (PointLookup-priced map) lives on conf_items — but demote the
				// engine that SERVES it: the query re-routes to conf_counts and
				// catches up via applyReplay.
				items := costShaped("conf_items", map[ReadPattern]float64{ReadAggregate: 1_000_000})
				counts := costShaped("conf_counts", map[ReadPattern]float64{ReadPointLookup: 1_000_000})

				store, err := Plan([]Engine{items, counts}, recordContextQuery(), roleCountQuery())
				if err != nil {
					t.Fatal(err)
				}

				WithEventLog(store, NewEventLog())
				t.Cleanup(func() { _ = store.Close() })

				if qa := queryAssignment(store, "record_context_tasks"); qa != "conf_items" {
					t.Fatalf("record query routed to %q, want conf_items", qa)
				}

				applyOK(t, store.ApplyRecord(context.Background(), fullRecord, recordContextEvent{TaskID: "t1"}))
				applyOK(t, store.DemoteEngine(context.Background(), "conf_items"))

				return store, conformView(t, counts)
			},
		},
		{
			name: "Replicator live mirror", want: fullView, advisory: 0,
			run: func(t *testing.T) (*Store, recordContextView) {
				store := newRecordContextStore(t)

				shadow := renamed("conf_live_mirror")
				applyOK(t, store.AddEngine(context.Background(), shadow, WithEngineRole(RoleBackup)))
				applyOK(t, store.ApplyRecord(context.Background(), fullRecord, recordContextEvent{TaskID: "t1"}))

				if !waitFor(t, conformWaitDeadline, func() bool {
					_, found, err := shadow.(MapBackend).MapGet(
						context.Background(), "record_context_tasks", "t1",
					)
					return err == nil && found
				}) {
					t.Fatal("live mirror never received the apply")
				}

				return store, conformView(t, shadow)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store, got := tc.run(t)

			if got != tc.want {
				t.Fatalf("fold observed %+v, want %+v", got, tc.want)
			}

			if applies := store.syntheticRecordApplies.Load(); applies != tc.advisory {
				t.Fatalf("synthetic-apply advisory = %d, want %d", applies, tc.advisory)
			}
		})
	}
}

func applyOK(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

// conformView reads the record_context_tasks row the conformance sweep's fold
// projected onto eng and reifies it to the observable view.
func conformView(t *testing.T, eng Engine) recordContextView {
	t.Helper()

	mb, ok := eng.(MapBackend)
	if !ok {
		t.Fatalf("%T is not a MapBackend", eng)
	}

	raw, found, err := mb.MapGet(context.Background(), "record_context_tasks", "t1")
	if err != nil || !found {
		t.Fatalf("conformView: found=%v err=%v", found, err)
	}

	view, err := reify[recordContextView](raw)
	if err != nil {
		t.Fatalf("conformView reify: %v", err)
	}

	return view
}

func queryAssignment(store *Store, query string) string {
	for _, qa := range store.Plan().Queries {
		if qa.QueryName == query {
			return qa.EngineName
		}
	}

	return ""
}
