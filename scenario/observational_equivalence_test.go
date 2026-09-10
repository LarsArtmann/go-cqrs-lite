package scenario_test

import (
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/scenario/v4"
)

// ── Observational equivalence (Cordis mapping §4, ADR-0136 context) ──
//
// The theorem under test: projection A's observable query results are
// IDENTICAL whether A runs alone or with projection B interleaved on the
// same shared Store. If interleaving B ever disturbed A's collection, this
// test fails — the regression gate for stream isolation + immutable events.

type taskPayload struct {
	ID    string
	Title string
	Done  bool
}

type taskView struct {
	ID    string
	Title string
	Done  bool
}

type findTask struct {
	ID string
}

type auditPayload struct {
	ID   string
	Note string
}

type findAudit struct {
	ID string
}

type auditView struct {
	ID   string
	Note string
}

// equivalenceDecoder routes wire event types to their payload structs.
func equivalenceDecoder(eventType string, payload []byte) (any, error) {
	switch eventType {
	case "taskPayload":
		var e taskPayload

		return e, json.Unmarshal(payload, &e)
	case "auditPayload":
		var a auditPayload

		return a, json.Unmarshal(payload, &a)
	}

	return nil, nil
}

// newEquivalenceStore plans a Store with the task collection, optionally
// also carrying the audit collection (projection B's footprint). Cleanup is
// the caller's: t.Cleanup for tests, defer for property bodies.
func newEquivalenceStore(t scenario.ScenarioReporter, withAudit bool) *metaengine.Store {
	t.Helper()

	taskQuery := metaengine.Query[findTask, taskView](
		"find-task",
		metaengine.OnRecord(taskPayload{}, func(_ record.Record, e taskPayload) (string, taskView) {
			return e.ID, taskView(e)
		}),
	)

	decls := []any{taskQuery}

	if withAudit {
		auditQuery := metaengine.Query[findAudit, auditView](
			"find-audit",
			metaengine.OnRecord(
				auditPayload{},
				func(_ record.Record, a auditPayload) (string, auditView) {
					return a.ID, auditView(a)
				},
			),
		)
		decls = append(decls, auditQuery)
	}

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, decls...)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	return store
}

func equivalenceEvent(t scenario.ScenarioReporter, eventType string, payload any) event.Event {
	t.Helper()

	streamID := id.NewStreamID()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s payload: %v", eventType, err)
	}

	evt, err := event.NewEvent(
		event.Type(eventType), streamID, "Equivalence", event.Version(1),
		payloadBytes,
	)
	if err != nil {
		t.Fatalf("event.NewEvent: %v", err)
	}

	return evt
}

func fixedEquivalenceEvents(t scenario.ScenarioReporter) []event.Event {
	t.Helper()

	return []event.Event{
		equivalenceEvent(t, "taskPayload", taskPayload{ID: "t1", Title: "first"}),
		equivalenceEvent(t, "auditPayload", auditPayload{ID: "a1", Note: "seen t1"}),
		equivalenceEvent(t, "taskPayload", taskPayload{ID: "t2", Title: "second"}),
		equivalenceEvent(t, "auditPayload", auditPayload{ID: "a2", Note: "seen t2"}),
		equivalenceEvent(
			t,
			"taskPayload",
			taskPayload{ID: "t1", Title: "first (renamed)", Done: true},
		),
	}
}

// TestProjectionObservationalEquivalence is the regression gate for the
// observational-equivalence property: the interleaved presence of
// projection B (audit) must not change projection A's (task) answers.
// AssertObservationalEquivalence owns the Given phase and the comparison.
func TestProjectionObservationalEquivalence(t *testing.T) {
	t.Parallel()

	storeA := newEquivalenceStore(t, false)
	t.Cleanup(func() { _ = storeA.Close() })

	storeAB := newEquivalenceStore(t, true)
	t.Cleanup(func() { _ = storeAB.Close() })

	scenario.AssertObservationalEquivalence(
		t,
		fixedEquivalenceEvents(t),
		projectionadapter.New("tasks-only", storeA, equivalenceDecoder),
		scenario.Interleaved(
			projectionadapter.New("tasks", storeAB, equivalenceDecoder),
			projectionadapter.New("audit", storeAB, equivalenceDecoder),
		),
		scenario.EquivalenceProbe{
			Baseline:    func() (any, error) { return storeA.Execute(findTask{ID: "t1"}) },
			Interleaved: func() (any, error) { return storeAB.Execute(findTask{ID: "t1"}) },
		},
	)
}
