package metaengine_test

import (
	"context"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// capability_surface_test.go pins the Doctor graph-non-replication note and the
// ExplainPlan capability-drift banner added 2026-08-16: honest degradation must
// be visible at runtime, not only in the adttest gate.

func TestDoctorNotesGraphNonReplication(t *testing.T) {
	t.Parallel()

	// A replicated engine with graph support (the iroh shape), modeled via
	// fakeEngine so the test does not import irohengine (wrong direction).
	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name:        "fake-replicated",
		Replication: metaengine.ReplicationLeaderless,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:   metaengine.ComplexityO1,
			metaengine.ADTGraph: metaengine.ComplexityODegree,
		},
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	doctor := store.Doctor(context.Background())
	if !strings.Contains(doctor, "--- Capability ---") {
		t.Fatalf("Doctor missing Capability section:\n%s", doctor)
	}

	want := "graph writes are local-only"
	if !strings.Contains(doctor, want) {
		t.Errorf("Doctor Capability section missing %q note for replicated graph engine:\n%s", want, doctor)
	}

	// A non-replicated graph engine must NOT get the note.
	plain := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "fake-local",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:   metaengine.ComplexityO1,
			metaengine.ADTGraph: metaengine.ComplexityODegree,
		},
	}}
	store2, err := metaengine.Plan([]metaengine.Engine{plain}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	if out := store2.Doctor(context.Background()); strings.Contains(out, want) {
		t.Errorf("non-replicated engine must not carry the replication note:\n%s", out)
	}
}

func TestExplainPlanShowsCapabilityDriftBanner(t *testing.T) {
	t.Parallel()

	// A plain memory engine conforms; its plan must NOT show the banner.
	honest, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer honest.Close()

	if plan := honest.ExplainPlan(); strings.Contains(plan, "Capability Warnings") {
		t.Errorf("healthy plan must not show capability warnings:\n%s", plan)
	}

	// A lying engine declares native vector support but implements no
	// VectorBackend — the over-declaration shape the banner must surface.
	lying := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "lying",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:    metaengine.ComplexityO1,
			metaengine.ADTVector: metaengine.ComplexityON,
		},
	}}
	drifted, err := metaengine.Plan([]metaengine.Engine{lying}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer drifted.Close()

	plan := drifted.ExplainPlan()
	if !strings.Contains(plan, "--- Capability Warnings ---") ||
		!strings.Contains(plan, "WARN capability drift") {
		t.Errorf("drifted plan missing capability warning banner:\n%s", plan)
	}
}
