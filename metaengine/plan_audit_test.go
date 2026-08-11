package metaengine_test

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// newAuditTestStore builds a minimal Store with one fake engine + one query,
// suitable for exercising the plan audit trail without external dependencies.
func newAuditTestStore(t *testing.T) *metaengine.Store {
	t.Helper()

	eng := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "audit-engine",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store
}

// TestPlanAudit_RecordsManualReplan verifies that a manual Replan appends a
// history entry tagged "manual" with the correct version.
func TestPlanAudit_RecordsManualReplan(t *testing.T) {
	t.Parallel()

	store := newAuditTestStore(t)
	ctx := context.Background()

	if err := store.Replan(ctx); err != nil {
		t.Fatalf("Replan: %v", err)
	}

	hist := store.PlanHistory()
	if len(hist) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(hist))
	}

	entry := hist[0]
	if entry.Trigger != "manual" {
		t.Errorf("trigger = %q, want %q", entry.Trigger, "manual")
	}

	if entry.Version != store.Plan().Version {
		t.Errorf("entry.Version = %d, want plan version %d", entry.Version, store.Plan().Version)
	}
}

// TestPlanAudit_RecordsPriorityChange verifies that SetPriority appends an
// entry tagged "priority-change" with a snapshot of the active priority.
func TestPlanAudit_RecordsPriorityChange(t *testing.T) {
	t.Parallel()

	store := newAuditTestStore(t)
	ctx := context.Background()

	pc := &metaengine.PriorityConfig{Global: metaengine.PriorityReadSpeed}
	if err := store.SetPriority(ctx, pc); err != nil {
		t.Fatalf("SetPriority: %v", err)
	}

	hist := store.PlanHistory()
	if len(hist) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(hist))
	}

	entry := hist[0]
	if entry.Trigger != "priority-change" {
		t.Errorf("trigger = %q, want %q", entry.Trigger, "priority-change")
	}

	if entry.Priority == nil {
		t.Fatal("priority snapshot should not be nil after SetPriority")
	}

	if got := entry.Priority.Resolve("", ""); got != metaengine.PriorityReadSpeed {
		t.Errorf("priority snapshot resolves to %s, want ReadSpeed", got)
	}
}

// TestPlanAudit_RecordsEngineAdded verifies that AddEngine appends an entry
// tagged "engine-added".
func TestPlanAudit_RecordsEngineAdded(t *testing.T) {
	t.Parallel()

	store := newAuditTestStore(t)
	ctx := context.Background()

	extra := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "extra-engine",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}

	if err := store.AddEngine(ctx, extra); err != nil {
		t.Fatalf("AddEngine: %v", err)
	}

	hist := store.PlanHistory()
	if len(hist) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(hist))
	}

	if hist[0].Trigger != "engine-added" {
		t.Errorf("trigger = %q, want %q", hist[0].Trigger, "engine-added")
	}
}

// TestPlanAudit_MultipleTransitions verifies version ordering and that the
// audit trail records transitions in chronological order.
func TestPlanAudit_MultipleTransitions(t *testing.T) {
	t.Parallel()

	store := newAuditTestStore(t)
	ctx := context.Background()

	_ = store.Replan(ctx)
	_ = store.SetPriority(ctx, &metaengine.PriorityConfig{Global: metaengine.PriorityWriteSpeed})
	_ = store.Replan(ctx)

	hist := store.PlanHistory()
	if len(hist) != 3 {
		t.Fatalf("expected 3 audit entries, got %d", len(hist))
	}

	wantTriggers := []string{"manual", "priority-change", "manual"}
	wantVersions := []int{2, 3, 4} // initial Plan sets v1, then 3 replans

	for i, e := range hist {
		if e.Trigger != wantTriggers[i] {
			t.Errorf("entry %d trigger = %q, want %q", i, e.Trigger, wantTriggers[i])
		}

		if e.Version != wantVersions[i] {
			t.Errorf("entry %d version = %d, want %d", i, e.Version, wantVersions[i])
		}
	}

	if !hist[2].At.After(hist[0].At) || !hist[2].At.Equal(hist[2].At) {
		t.Error("timestamps should be non-decreasing")
	}
}

// TestPlanAudit_DoctorReport verifies the Doctor report surfaces the audit
// trail in the Routing section.
func TestPlanAudit_DoctorReport(t *testing.T) {
	t.Parallel()

	store := newAuditTestStore(t)
	ctx := context.Background()

	_ = store.SetPriority(ctx, &metaengine.PriorityConfig{Global: metaengine.PriorityReadSpeed})

	report := store.Doctor(ctx)

	if !strings.Contains(report, "audit:") {
		t.Errorf("Doctor report should contain 'audit:' line after replan:\n%s", report)
	}

	if !strings.Contains(report, "priority-change") {
		t.Errorf("Doctor report should name the trigger in the audit trail:\n%s", report)
	}
}

// TestPlanAudit_PrioritySnapshotIsImmutable verifies that mutating the
// PriorityConfig passed to SetPriority after the call does NOT retroactively
// change the recorded audit entry.
func TestPlanAudit_PrioritySnapshotIsImmutable(t *testing.T) {
	t.Parallel()

	store := newAuditTestStore(t)
	ctx := context.Background()

	pc := &metaengine.PriorityConfig{
		Global:    metaengine.PriorityReadSpeed,
		PerEngine: map[string]metaengine.Priority{"audit-engine": metaengine.PriorityBalanced},
	}

	_ = store.SetPriority(ctx, pc)

	// Mutate the original AFTER the call.
	pc.Global = metaengine.PriorityWriteSpeed
	pc.PerEngine["audit-engine"] = metaengine.PriorityStorageSpace

	hist := store.PlanHistory()
	if len(hist) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(hist))
	}

	snap := hist[0].Priority
	if snap.Global != metaengine.PriorityReadSpeed {
		t.Errorf("snapshot Global = %s, want ReadSpeed (should be immune to post-call mutation)", snap.Global)
	}

	if got := snap.PerEngine["audit-engine"]; got != metaengine.PriorityBalanced {
		t.Errorf("snapshot PerEngine = %s, want Balanced (should be immune to post-call mutation)", got)
	}
}
