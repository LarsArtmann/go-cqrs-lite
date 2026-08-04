package metaengine

import (
	"path/filepath"
	"testing"
)

func TestPlanDiff_IdenticalPlans(t *testing.T) {
	t.Parallel()

	plan := &SerializablePlan{
		Engines: []string{"memory"},
		Queries: []SerializableQuery{
			{Name: "users", ADT: ADTMap, Engine: "memory"},
		},
	}

	diff := PlanDiff(plan, plan)
	if !diff.IsEmpty() {
		t.Fatalf("expected empty diff for identical plans, got %+v", diff)
	}
}

func TestPlanDiff_EngineAdded(t *testing.T) {
	t.Parallel()

	prev := &SerializablePlan{Engines: []string{"memory"}}
	curr := &SerializablePlan{Engines: []string{"memory", "sqlite"}}

	diff := PlanDiff(prev, curr)
	if len(diff.EnginesAdded) != 1 || diff.EnginesAdded[0] != "sqlite" {
		t.Fatalf("expected sqlite added, got %v", diff.EnginesAdded)
	}
}

func TestPlanDiff_EngineRemoved(t *testing.T) {
	t.Parallel()

	prev := &SerializablePlan{Engines: []string{"memory", "sqlite"}}
	curr := &SerializablePlan{Engines: []string{"memory"}}

	diff := PlanDiff(prev, curr)
	if len(diff.EnginesRemoved) != 1 || diff.EnginesRemoved[0] != "sqlite" {
		t.Fatalf("expected sqlite removed, got %v", diff.EnginesRemoved)
	}
}

func TestPlanDiff_QueryChanged(t *testing.T) {
	t.Parallel()

	prev := &SerializablePlan{
		Queries: []SerializableQuery{
			{Name: "users", ADT: ADTMap, Engine: "memory"},
		},
	}
	curr := &SerializablePlan{
		Queries: []SerializableQuery{
			{Name: "users", ADT: ADTMap, Engine: "sqlite"},
		},
	}

	diff := PlanDiff(prev, curr)
	if len(diff.QueriesChanged) != 1 {
		t.Fatalf("expected 1 changed query, got %d", len(diff.QueriesChanged))
	}

	c := diff.QueriesChanged[0]
	if c.Name != "users" || c.OldEngine != "memory" || c.NewEngine != "sqlite" {
		t.Fatalf("unexpected change: %+v", c)
	}
}

func TestPlanDiff_QueryAddedAndRemoved(t *testing.T) {
	t.Parallel()

	prev := &SerializablePlan{
		Queries: []SerializableQuery{
			{Name: "old_query", ADT: ADTMap, Engine: "memory"},
		},
	}
	curr := &SerializablePlan{
		Queries: []SerializableQuery{
			{Name: "new_query", ADT: ADTCounter, Engine: "memory"},
		},
	}

	diff := PlanDiff(prev, curr)
	if len(diff.QueriesAdded) != 1 || diff.QueriesAdded[0] != "new_query" {
		t.Fatalf("expected new_query added, got %v", diff.QueriesAdded)
	}

	if len(diff.QueriesRemoved) != 1 || diff.QueriesRemoved[0] != "old_query" {
		t.Fatalf("expected old_query removed, got %v", diff.QueriesRemoved)
	}
}

func TestPlanFingerprint_Stable(t *testing.T) {
	t.Parallel()

	plan := &SerializablePlan{
		Engines: []string{"memory"},
		Queries: []SerializableQuery{
			{Name: "users", ADT: ADTMap, Engine: "memory"},
		},
	}

	fp1, err := PlanFingerprint(plan)
	if err != nil {
		t.Fatalf("PlanFingerprint: %v", err)
	}

	fp2, err := PlanFingerprint(plan)
	if err != nil {
		t.Fatalf("PlanFingerprint second: %v", err)
	}

	if fp1 != fp2 {
		t.Fatalf("fingerprint not stable: %s != %s", fp1, fp2)
	}

	if len(fp1) != 64 {
		t.Fatalf("expected 64-char hex hash, got %d chars", len(fp1))
	}
}

func TestPlanFingerprint_DifferentPlans(t *testing.T) {
	t.Parallel()

	plan1 := &SerializablePlan{Engines: []string{"memory"}}
	plan2 := &SerializablePlan{Engines: []string{"sqlite"}}

	fp1, _ := PlanFingerprint(plan1)
	fp2, _ := PlanFingerprint(plan2)

	if fp1 == fp2 {
		t.Fatal("different plans should have different fingerprints")
	}
}

func TestManifest_Roundtrip(t *testing.T) {
	t.Parallel()

	plan := &SerializablePlan{
		Engines: []string{"memory"},
		Queries: []SerializableQuery{
			{Name: "users", ADT: ADTMap, Engine: "memory"},
		},
	}

	manifest, err := NewManifest(plan)
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}

	if manifest.Fingerprint == "" {
		t.Fatal("expected non-empty fingerprint")
	}

	if manifest.Version != 1 {
		t.Fatalf("expected version 1, got %d", manifest.Version)
	}

	// Verify fingerprint matches.
	ok, err := manifest.VerifyFingerprint()
	if err != nil {
		t.Fatalf("VerifyFingerprint: %v", err)
	}

	if !ok {
		t.Fatal("fingerprint verification failed for unchanged plan")
	}

	// Save and load.
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := SaveManifest(path, manifest); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	if loaded.Fingerprint != manifest.Fingerprint {
		t.Fatalf("fingerprint mismatch after roundtrip: %s != %s",
			loaded.Fingerprint, manifest.Fingerprint)
	}

	if len(loaded.Plan.Queries) != 1 {
		t.Fatalf("expected 1 query after roundtrip, got %d", len(loaded.Plan.Queries))
	}
}
