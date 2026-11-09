package metaengine

import (
	"context"
	"strings"
	"testing"
)

func stampTestPlan() LayoutPlan {
	return LayoutPlan{
		Collection: "user_views",
		Table:      "meta_planned_users",
		Columns: []PlannedColumn{
			{Name: "status", Type: "TEXT"},
			{Name: "rank", Type: "INTEGER"},
		},
		Indexes: []PlannedIndex{
			{Name: "idx_planned_users_status", Columns: []string{"status"}},
		},
	}
}

func TestLayoutPlanFingerprintStableAndSensitive(t *testing.T) {
	t.Parallel()

	base := stampTestPlan()
	same := stampTestPlan()

	if base.Fingerprint() != same.Fingerprint() {
		t.Fatal("identical plans fingerprinted differently")
	}

	// Column order must not matter (sorted serialization).
	reordered := stampTestPlan()
	reordered.Columns = []PlannedColumn{
		{Name: "rank", Type: "INTEGER"},
		{Name: "status", Type: "TEXT"},
	}

	if reordered.Fingerprint() != base.Fingerprint() {
		t.Fatal("column order changed the fingerprint (serialization not sorted)")
	}

	drifted := []LayoutPlan{
		func() LayoutPlan {
			p := stampTestPlan()
			p.Table = "meta_planned_users_v2"

			return p
		}(),
		func() LayoutPlan {
			p := stampTestPlan()
			p.Columns = append(p.Columns, PlannedColumn{Name: "email", Type: "TEXT"})

			return p
		}(),
		func() LayoutPlan {
			p := stampTestPlan()
			p.Columns = []PlannedColumn{{Name: "status", Type: "INTEGER"}}

			return p
		}(),
		func() LayoutPlan {
			p := stampTestPlan()
			p.Indexes = nil

			return p
		}(),
	}

	for i, driftedPlan := range drifted {
		if driftedPlan.Fingerprint() == base.Fingerprint() {
			t.Errorf("case %d: layout change did not change the fingerprint", i)
		}
	}
}

func TestLayoutStampRecordAndDiff(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	engine := NewMemoryEngine()
	t.Cleanup(func() { _ = engine.Close() })
	backend, ok := engine.(MapBackend)
	if !ok {
		t.Fatal("memory engine must implement MapBackend")
	}

	plan := stampTestPlan()

	// Absent = accept: no stamp recorded yet, no diff.
	diffs, err := LayoutStampDiffs(ctx, backend, []LayoutPlan{plan})
	if err != nil {
		t.Fatalf("diffs on absent stamps: %v", err)
	}

	if len(diffs) != 0 {
		t.Fatalf("absent stamp must not drift, got %v", diffs)
	}

	if err := RecordLayoutStamps(ctx, backend, []LayoutPlan{plan}); err != nil {
		t.Fatalf("record: %v", err)
	}

	// Recorded == declared: still no diff.
	diffs, err = LayoutStampDiffs(ctx, backend, []LayoutPlan{plan})
	if err != nil {
		t.Fatalf("diffs on matching stamp: %v", err)
	}

	if len(diffs) != 0 {
		t.Fatalf("matching stamp must not drift, got %v", diffs)
	}

	// Declared plan changed → exactly one diff with both fingerprints.
	changed := stampTestPlan()
	changed.Columns = append(changed.Columns, PlannedColumn{Name: "email", Type: "TEXT"})

	diffs, err = LayoutStampDiffs(ctx, backend, []LayoutPlan{changed})
	if err != nil {
		t.Fatalf("diffs on drifted stamp: %v", err)
	}

	if len(diffs) != 1 {
		t.Fatalf("expected one drift, got %v", diffs)
	}

	if diffs[0].Collection != plan.Collection ||
		diffs[0].Recorded != plan.Fingerprint() ||
		diffs[0].Declared != changed.Fingerprint() {
		t.Fatalf("diff carried wrong details: %+v", diffs[0])
	}
}

func TestLayoutStampsClearedByReset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	engine := NewMemoryEngine()
	t.Cleanup(func() { _ = engine.Close() })
	backend, ok := engine.(MapBackend)
	if !ok {
		t.Fatal("memory engine must implement MapBackend")
	}

	plan := stampTestPlan()

	if err := RecordLayoutStamps(ctx, backend, []LayoutPlan{plan}); err != nil {
		t.Fatalf("record: %v", err)
	}

	// Engine resets clear materialized collections — the stamp store rides
	// ordinary map persistence, so it clears too (ADR-0143; the journal is
	// the survivor). After a reset the stamps are simply absent again.
	resetter, ok := engine.(EngineResetter)
	if !ok {
		t.Fatal("memory engine must implement EngineResetter")
	}

	if err := resetter.ResetEngine(ctx); err != nil {
		t.Fatalf("reset: %v", err)
	}

	diffs, err := LayoutStampDiffs(ctx, backend, []LayoutPlan{plan})
	if err != nil {
		t.Fatalf("diffs after reset: %v", err)
	}

	if len(diffs) != 0 {
		t.Fatalf("cleared stamp must read as absent (accepted), got %v", diffs)
	}
}

func TestReplayCompleteMarker(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	engine := NewMemoryEngine()
	t.Cleanup(func() { _ = engine.Close() })
	backend, ok := engine.(MapBackend)
	if !ok {
		t.Fatal("memory engine must implement MapBackend")
	}

	if _, ok, err := ReplayCompletedAt(ctx, backend, "user_views"); err != nil || ok {
		t.Fatalf("unmarked collection must report no marker (ok=%v err=%v)", ok, err)
	}

	if err := MarkReplayComplete(ctx, backend, "user_views"); err != nil {
		t.Fatalf("mark: %v", err)
	}

	completedAt, ok, err := ReplayCompletedAt(ctx, backend, "user_views")
	if err != nil || !ok {
		t.Fatalf("marked collection must report its marker (ok=%v err=%v)", ok, err)
	}

	if completedAt == "" {
		t.Fatal("marker carried no timestamp")
	}

	section := LayoutStampsDoctorSection(ctx, backend, []LayoutPlan{stampTestPlan()})
	if section == "" {
		t.Fatal("doctor section must render for a declared plan")
	}

	if !strings.Contains(section, "user_views") || !strings.Contains(section, "replayed at") {
		t.Fatalf("doctor section missing expected rows:\n%s", section)
	}
}
