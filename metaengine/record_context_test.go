package metaengine

import (
	"context"
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
		OnRecord(recordContextEvent{}, func(rec record.Record, e recordContextEvent) (string, recordContextView) {
			return e.TaskID, recordContextView{
				TaskID:   e.TaskID,
				StreamID: string(rec.StreamID),
				Version:  rec.Version,
			}
		}),
	)
}

func newRecordContextStore(t *testing.T) *Store {
	t.Helper()

	store, err := Plan(NewMemoryEngine(), recordContextQuery())
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

// TestApply_NoRecordAwareFolds_NoAdvisory proves plain On folds never trip
// the advisory: synthesized Records are harmless when the handler ignores
// the Record.
func TestApply_NoRecordAwareFolds_NoAdvisory(t *testing.T) {
	t.Parallel()

	store, err := Plan(NewMemoryEngine(), Query[plainEvent, map[string]string](
		"plain_tasks",
		On(plainEvent{}, func(e plainEvent) (string, string) {
			return e.ID, e.ID
		}),
	))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()

	if err := store.Apply(ctx, "plainEvent", plainEvent{ID: "p1"}); err != nil {
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
