package system_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_WiringDeterministic pins that constructing two Systems from the
// SAME domain + deployment (with map-iterated engine declarations) yields
// identical wiring (Feedback #6 (c) / M17.4): the Explain() rendering — which
// covers engine creation order, instance roles, and routing — must be
// byte-identical. Map iteration order is randomized per process; any wiring
// decision keyed on map iteration leaks into Explain and fails here.
func TestSystem_WiringDeterministic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	build := func() (string, []string) {
		deployment := system.DeploymentConfig{
			Engines: map[string]system.EngineConfig{
				"zeta":    {Driver: "memory"},
				"alpha":   {Driver: "memory"},
				"midway":  {Driver: "memory"},
				"primary": {Driver: "memory"},
			},
			Instances: []system.InstanceConfig{
				{Role: system.RoleSourceOfTruth, Engine: "primary"},
				{Role: system.RoleProjections, Engines: []string{"zeta", "alpha", "midway"}},
			},
		}

		sys, err := system.New(ctx, system.DomainConfig{}, deployment)
		if err != nil {
			t.Fatalf("system.New: %v", err)
		}

		defer sys.Close()

		return sys.Explain(ctx), sys.EngineNames()
	}

	explainA, namesA := build()
	explainB, namesB := build()

	if explainA != explainB {
		t.Fatalf("Explain() differs between two identical constructs:\n--- A ---\n%s\n--- B ---\n%s",
			explainA, explainB)
	}

	for i := range namesA {
		if namesA[i] != namesB[i] {
			t.Fatalf("EngineNames[%d] = %q vs %q (wiring not deterministic)", i, namesA[i], namesB[i])
		}
	}
}
