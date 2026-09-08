package metaengine

import (
	"context"
	"log"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// recordContextEvent is the event type used by the Record-context tests.
type recordContextEvent struct {
	TaskID string
}

// recordContextView is what the OnRecord fold projects.
type recordContextView struct {
	TaskID   string
	StreamID string
	Version  int64
}

func recordContextQuery() QueryDecl[recordContextEvent, map[string]recordContextView] {
	return Query[recordContextEvent, map[string]recordContextView](
		"record_context_tasks",
		OnRecord(
			recordContextEvent{},
			func(rec record.Record, e recordContextEvent) (string, recordContextView) {
				return e.TaskID, recordContextView{
					TaskID:   e.TaskID,
					StreamID: string(rec.StreamID),
					Version:  rec.Version,
				}
			},
		),
	)
}

func newRecordContextStore(t *testing.T) *Store {
	t.Helper()

	store, err := Plan([]Engine{NewMemoryEngine()}, recordContextQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	return store
}

// TestApply_SyntheticRecordAdvisory proves that Apply feeds OnRecord folds a
// Type-only Record and that the hazard is surfaced: the Doctor reports the
// synthesized applies instead of staying silent.
func TestApply_SyntheticRecordAdvisory(t *testing.T) {
	t.Parallel()

	store := newRecordContextStore(t)
	ctx := context.Background()

	if err := store.Apply(ctx, "recordContextEvent", recordContextEvent{TaskID: "t1"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	doctor := store.Doctor(ctx)
	if !strings.Contains(doctor, "--- Record context ---") {
		t.Fatalf("Doctor missing Record context section:\n%s", doctor)
	}

	if !strings.Contains(doctor, "recordContextEvent") {
		t.Fatalf("Doctor does not list the record-aware event type:\n%s", doctor)
	}

	if !strings.Contains(doctor, "synthesized Type-only Record") {
		t.Fatalf("Doctor does not report the synthesized applies:\n%s", doctor)
	}
}

// TestApplyRecord_FullContext pins the contract: ApplyRecord feeds the fold
// the real StreamID/Version, and the Doctor reports full context.
func TestApplyRecord_FullContext(t *testing.T) {
	t.Parallel()

	store := newRecordContextStore(t)
	ctx := context.Background()

	rec := record.Record{
		Type:     "recordContextEvent",
		StreamID: "Task/01JTEST",
		Version:  3,
	}

	if err := store.ApplyRecord(ctx, rec, recordContextEvent{TaskID: "t1"}); err != nil {
		t.Fatalf("ApplyRecord: %v", err)
	}

	doctor := store.Doctor(ctx)
	if !strings.Contains(doctor, "all applies carried full Record context") {
		t.Fatalf("Doctor should report full context only:\n%s", doctor)
	}

	mb := store.engines[0].(MapBackend)
	raw, ok, err := mb.MapGet(ctx, "record_context_tasks", "t1")
	if err != nil || !ok {
		t.Fatalf("MapGet: ok=%v err=%v", ok, err)
	}

	view, err := reify[recordContextView](raw)
	if err != nil {
		t.Fatalf("reify: %v", err)
	}

	if view.StreamID != "Task/01JTEST" || view.Version != 3 {
		t.Fatalf("fold saw partial context: %+v", view)
	}
}

// plainRecordEvent is an On-only event type (no Record parameter).
type plainRecordEvent struct{ ID string }

// TestApply_NoRecordAwareFolds_NoAdvisory proves plain On folds never trip
// the advisory: synthesized Records are harmless when the handler ignores
// the Record.
func TestApply_NoRecordAwareFolds_NoAdvisory(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngine()}, Query[plainRecordEvent, map[string]string](
		"plain_tasks",
		On(plainRecordEvent{}, func(e plainRecordEvent) (string, string) {
			return e.ID, e.ID
		}),
	))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()

	if err := store.Apply(ctx, "plainRecordEvent", plainRecordEvent{ID: "p1"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if doctor := store.Doctor(ctx); strings.Contains(doctor, "--- Record context ---") {
		t.Fatalf("Doctor should have no Record context section for On-only folds:\n%s", doctor)
	}
}

// TestIsSyntheticRecord pins the detection predicate.
func TestIsSyntheticRecord(t *testing.T) {
	t.Parallel()

	if !isSyntheticRecord(record.Record{Type: "e"}) {
		t.Fatal("Type-only record must be synthetic")
	}

	if isSyntheticRecord(record.Record{Type: "e", StreamID: "T/1", Version: 1}) {
		t.Fatal("stream-scoped record must not be synthetic")
	}
}

// TestApplyBatch_HonorsRecord pins the contract that ApplyBatch forwards
// EventInput.Record to OnRecord folds instead of synthesizing a Type-only
// Record (the pre-fix behavior silently dropped the field).
func TestApplyBatch_HonorsRecord(t *testing.T) {
	t.Parallel()

	store := newRecordContextStore(t)
	ctx := context.Background()

	events := []EventInput{
		{
			Type:    "recordContextEvent",
			Payload: recordContextEvent{TaskID: "t1"},
			Record: record.Record{
				Type:     "recordContextEvent",
				StreamID: "Task/01JTEST",
				Version:  3,
			},
		},
		{
			Type:    "recordContextEvent",
			Payload: recordContextEvent{TaskID: "t2"},
			Record:  record.Record{StreamID: "Task/01JOTHER", Version: 1},
		},
	}

	if err := store.ApplyBatch(ctx, events); err != nil {
		t.Fatalf("ApplyBatch: %v", err)
	}

	mb := store.engines[0].(MapBackend)

	for key, wantStream := range map[string]string{"t1": "Task/01JTEST", "t2": "Task/01JOTHER"} {
		raw, ok, err := mb.MapGet(ctx, "record_context_tasks", key)
		if err != nil || !ok {
			t.Fatalf("MapGet(%s): ok=%v err=%v", key, ok, err)
		}

		view, err := reify[recordContextView](raw)
		if err != nil {
			t.Fatalf("reify(%s): %v", key, err)
		}

		if view.StreamID != wantStream {
			t.Fatalf("fold for %s saw StreamID %q, want %q", key, view.StreamID, wantStream)
		}
	}

	if doctor := store.Doctor(
		ctx,
	); !strings.Contains(
		doctor,
		"all applies carried full Record context",
	) {
		t.Fatalf("Doctor should report full context only:\n%s", doctor)
	}
}

// TestApplyBatch_SyntheticRecord_Counted proves a batch event without a
// Record gets the same synthesized Type-only Record as Apply, and the
// synthetic-apply advisory still counts it.
func TestApplyBatch_SyntheticRecord_Counted(t *testing.T) {
	t.Parallel()

	store := newRecordContextStore(t)
	ctx := context.Background()

	events := []EventInput{
		{Type: "recordContextEvent", Payload: recordContextEvent{TaskID: "t1"}},
		{
			Type:    "recordContextEvent",
			Payload: recordContextEvent{TaskID: "t2"},
			Record:  record.Record{Type: "recordContextEvent"},
		},
	}

	if err := store.ApplyBatch(ctx, events); err != nil {
		t.Fatalf("ApplyBatch: %v", err)
	}

	doctor := store.Doctor(ctx)
	if !strings.Contains(doctor, "2 apply event(s) arrived with a synthesized Type-only Record") {
		t.Fatalf("Doctor should count both synthesized batch applies:\n%s", doctor)
	}

	mb := store.engines[0].(MapBackend)
	raw, ok, err := mb.MapGet(ctx, "record_context_tasks", "t1")
	if err != nil || !ok {
		t.Fatalf("MapGet: ok=%v err=%v", ok, err)
	}

	view, err := reify[recordContextView](raw)
	if err != nil {
		t.Fatalf("reify: %v", err)
	}

	if view.StreamID != "" || view.Version != 0 {
		t.Fatalf("synthetic batch apply must see empty context, got %+v", view)
	}
}

// TestRegisterQuery_InvalidatesRecordAwareCache proves the record-aware
// event-type memo is recomputed when a query with OnRecord folds is
// registered at runtime: applies AFTER registration must count as synthetic,
// even though an earlier apply already populated the cache.
func TestRegisterQuery_InvalidatesRecordAwareCache(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngine()}, Query[plainRecordEvent, map[string]string](
		"plain_tasks",
		On(plainRecordEvent{}, func(e plainRecordEvent) (string, string) {
			return e.ID, e.ID
		}),
	))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()

	if err := store.Apply(ctx, "plainRecordEvent", plainRecordEvent{ID: "p1"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if err := store.RegisterQuery(recordContextQuery()); err != nil {
		t.Fatalf("RegisterQuery: %v", err)
	}

	if err := store.Apply(ctx, "recordContextEvent", recordContextEvent{TaskID: "t1"}); err != nil {
		t.Fatalf("Apply after RegisterQuery: %v", err)
	}

	doctor := store.Doctor(ctx)
	if !strings.Contains(doctor, "1 apply event(s) arrived with a synthesized Type-only Record") {
		t.Fatalf("advisory missed the post-registration synthetic apply:\n%s", doctor)
	}
}

// TestSyntheticRecordAdvisory_LoggerPath pins the Hooks.Logger advisory: the
// first synthetic apply logs once, subsequent applies stay silent.
func TestSyntheticRecordAdvisory_LoggerPath(t *testing.T) {
	t.Parallel()

	store := newRecordContextStore(t)
	ctx := context.Background()

	var buf strings.Builder

	WithHooks(store, Hooks{Logger: log.New(&buf, "", 0)})

	for i := range 3 {
		if err := store.Apply(
			ctx,
			"recordContextEvent",
			recordContextEvent{TaskID: "t1"},
		); err != nil {
			t.Fatalf("Apply %d: %v", i, err)
		}
	}

	logged := buf.String()
	if !strings.Contains(logged, `event "recordContextEvent" applied via Store.Apply`) {
		t.Fatalf("advisory not logged:\n%s", logged)
	}

	if n := strings.Count(logged, "recordContextEvent"); n != 1 {
		t.Fatalf("one-time advisory logged %d times, want 1:\n%s", n, logged)
	}
}
