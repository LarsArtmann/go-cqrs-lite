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

func newStoreWithEngine(t *testing.T, eng metaengine.Engine) *metaengine.Store {
	t.Helper()

	store, err := metaengine.Plan([]metaengine.Engine{eng})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	return store
}

// replicatedGraphEngine pretends to be a replicated engine with graph support
// (the iroh shape) without importing irohengine (wrong dependency direction).
type replicatedGraphEngine struct {
	metaengine.Engine
}

func (replicatedGraphEngine) Profile() metaengine.EngineProfile {
	p := metaengine.NewMemoryEngine().Profile()
	p.Name = "fake-replicated"
	p.Replication = metaengine.ReplicationLeaderless
	p.Supports[metaengine.ADTGraph] = metaengine.ComplexityODegree
	return p
}

func TestDoctorNotesGraphNonReplication(t *testing.T) {
	t.Parallel()

	store := newStoreWithEngine(t, replicatedGraphEngine{metaengine.NewMemoryEngine()})

	doctor := store.Doctor(context.Background())
	if !strings.Contains(doctor, "--- Capability ---") {
		t.Fatalf("Doctor missing Capability section:\n%s", doctor)
	}

	want := "graph writes are local-only"
	if !strings.Contains(doctor, want) {
		t.Errorf("Doctor Capability section missing %q note for replicated graph engine:\n%s", want, doctor)
	}
}

// lyingProfile declares native vector support on an engine that implements no
// VectorBackend — the over-declaration shape the EXPLAIN banner must surface.
type lyingProfile struct {
	metaengine.Engine
}

func (lyingProfile) Profile() metaengine.EngineProfile {
	p := metaengine.NewMemoryEngine().Profile()
	p.Name = "lying"
	p.Supports[metaengine.ADTVector] = metaengine.ComplexityON
	return p
}

func TestExplainPlanShowsCapabilityDriftBanner(t *testing.T) {
	t.Parallel()

	// A plain memory engine conforms; its plan must NOT show the banner.
	if plan := newStoreWithEngine(t, metaengine.NewMemoryEngine()).ExplainPlan(); strings.Contains(plan, "Capability Warnings") {
		t.Errorf("healthy plan must not show capability warnings:\n%s", plan)
	}

	plan := newStoreWithEngine(t, lyingProfile{metaengine.NewMemoryEngine()}).ExplainPlan()
	if !strings.Contains(plan, "--- Capability Warnings ---") ||
		!strings.Contains(plan, "WARN capability drift") {
		t.Errorf("drifted plan missing capability warning banner:\n%s", plan)
	}
}
