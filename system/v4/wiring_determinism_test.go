package system_test

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_WiringDeterministic pins that constructing two Systems from the
// SAME domain + deployment (with map-iterated engine declarations) yields
// identical wiring (Feedback #6 (c) / M17.4): the Explain() rendering — which
// covers engine creation order, instance roles, and routing — must be
// byte-identical. Map iteration order is randomized per process; any wiring
// decision keyed on map iteration leaks into Explain and fails here.
//
// Explain() also renders two lines the construct does NOT determine: the
// Drivers line mirrors the process-global driver registry (parallel tests
// register leakprobe-* drivers mid-run — the 2026-10-09 verify flake, where
// construct A snapshotted before a registration construct B saw), and the
// Time line carries the wall clock (a second boundary between builds would
// flake the byte-compare). Both lines are stripped; engine lists, roles,
// routing, and EngineNames keep the determinism pin fully armed.
func TestSystem_WiringDeterministic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	stripNonWiring := func(explain string) string {
		lines := strings.Split(explain, "\n")
		kept := make([]string, 0, len(lines))
		for _, line := range lines {
			if strings.HasPrefix(line, "  Drivers: ") || strings.HasPrefix(line, "  Time: ") {
				continue
			}
			kept = append(kept, line)
		}
		return strings.Join(kept, "\n")
	}

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

	if a, b := stripNonWiring(explainA), stripNonWiring(explainB); a != b {
		t.Fatalf(
			"Explain() differs between two identical constructs:\n--- A ---\n%s\n--- B ---\n%s",
			a,
			b,
		)
	}

	for i := range namesA {
		if namesA[i] != namesB[i] {
			t.Fatalf(
				"EngineNames[%d] = %q vs %q (wiring not deterministic)",
				i,
				namesA[i],
				namesB[i],
			)
		}
	}
}
