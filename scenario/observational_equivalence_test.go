package scenario_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
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
// also carrying the audit collection (projection B's footprint).
func newEquivalenceStore(t *testing.T, withAudit bool) *metaengine.Store {
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
			metaengine.OnRecord(auditPayload{}, func(_ record.Record, a auditPayload) (string, auditView) {
				return a.ID, auditView(a)
			}),
		)
		decls = append(decls, auditQuery)
	}

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, decls...)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store
}

func equivalenceEvent(t *testing.T, eventType string, payload any) event.Event {
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

// interleavedProjection runs B's Handle after A's for every event — the
// "A+B interleaved" composition under test.
type interleavedProjection struct {
	a, b projection.Projection
}

func (p *interleavedProjection) Name() string { return p.a.Name() + "+" + p.b.Name() }

func (p *interleavedProjection) EventTypes() []event.Type {
	seen := make(map[event.Type]struct{})
	types := append([]event.Type(nil), p.a.EventTypes()...)

	for _, t := range p.b.EventTypes() {
		if _, ok := seen[t]; ok {
			continue
		}

		seen[t] = struct{}{}
		types = append(types, t)
	}

	return types
}

func (p *interleavedProjection) Handle(ctx context.Context, evt event.Event) error {
	if err := p.a.Handle(ctx, evt); err != nil {
		return err
	}

	return p.b.Handle(ctx, evt)
}

// TestProjectionObservationalEquivalence is the regression gate for the
// observational-equivalence property: the interleaved presence of
// projection B (audit) must not change projection A's (task) answers.
func TestProjectionObservationalEquivalence(t *testing.T) {
	t.Parallel()

	events := []event.Event{
		equivalenceEvent(t, "taskPayload", taskPayload{ID: "t1", Title: "first"}),
		equivalenceEvent(t, "auditPayload", auditPayload{ID: "a1", Note: "seen t1"}),
		equivalenceEvent(t, "taskPayload", taskPayload{ID: "t2", Title: "second"}),
		equivalenceEvent(t, "auditPayload", auditPayload{ID: "a2", Note: "seen t2"}),
		equivalenceEvent(t, "taskPayload", taskPayload{ID: "t1", Title: "first (renamed)", Done: true}),
	}

	// Baseline: A alone on its own Store.
	storeA := newEquivalenceStore(t, false)

	scenario.GivenProjection(t,
		projectionadapter.New("tasks-only", storeA, equivalenceDecoder),
		events...,
	).ThenNoError()

	baseline, err := storeA.Execute(findTask{ID: "t1"})
	if err != nil {
		t.Fatalf("baseline execute: %v", err)
	}

	// Interleaved: A and B share one Store; B runs between A's folds.
	storeAB := newEquivalenceStore(t, true)

	interleaved := &interleavedProjection{
		a: projectionadapter.New("tasks", storeAB, equivalenceDecoder),
		b: projectionadapter.New("audit", storeAB, equivalenceDecoder),
	}

	scenario.GivenProjection(t, interleaved, events...).
		ThenQueryResult(
			func() (any, error) { return storeAB.Execute(findTask{ID: "t1"}) },
			baseline,
		)
}
