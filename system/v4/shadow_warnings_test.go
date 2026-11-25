package system_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

type DupView struct {
	ID string
}

type DupCreated struct {
	ID string
}

type DupOtherView struct {
	ID string
}

// DupAltCreated is a second, different sample struct for the same wire type
// (e.g. redeclared in another package of the same app).
type DupAltCreated struct {
	ID string
}

// TestSystem_DuplicateEvolutionRejected pins that two Evolutions for one
// result type fail construction. The first one used to be dropped silently.
func TestSystem_DuplicateEvolutionRejected(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Events: []event.Type{"dup.created"},
		Evolutions: []system.EvolutionSpec{
			system.OnEvolution(system.Evolve[DupView]("dup_first"), "dup.created", DupCreated{}).Done(),
			system.OnEvolution(system.Evolve[DupView]("dup_second"), "dup.created", DupCreated{}).Done(),
		},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[DupView]("dup_lookup").Done(),
		},
	}

	sys, err := system.New(ctx, domain, tombstoneDeployment())
	if err == nil {
		sys.Close()
		t.Fatal("system.New accepted two Evolutions for the same result type")
	}

	for _, want := range []string{"dup_first", "dup_second", "DupView"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}

// TestSystem_ConflictingSampleStructsWarn pins that one wire event type
// declared with two different sample structs is reported. The auto decoder
// follows the last declaration; the other projection's fold then fails on
// every such event.
func TestSystem_ConflictingSampleStructsWarn(t *testing.T) {
	buf := captureSlog(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Events: []event.Type{"dup.created"},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[DupView]("dup_lookup").On("dup.created", DupCreated{}).Done(),
			system.Lookup[DupOtherView]("dup_other").On("dup.created", DupAltCreated{}).Done(),
		},
	}

	sys, err := system.New(ctx, domain, tombstoneDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	if out := buf.String(); !strings.Contains(out, "two different sample structs") ||
		!strings.Contains(out, "dup.created") {
		t.Errorf("expected a conflicting-sample warning for dup.created, got log: %s", out)
	}
}

// TestSystem_ShadowedInstancesWarn pins that instances New wires but never
// uses are named in a warning: a second source-of-truth/events instance and
// a second projections instance used to be ignored without a trace.
func TestSystem_ShadowedInstancesWarn(t *testing.T) {
	buf := captureSlog(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary":   {Driver: "memory"},
			"secondary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleEvents, Engine: "secondary"},
			{Role: system.RoleProjections, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "secondary"},
		},
	}

	sys, err := system.New(ctx, system.DomainConfig{}, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	out := buf.String()

	for _, want := range []string{"events on secondary", "projections on secondary"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected the shadowed instance %q to be reported, got log: %s", want, out)
		}
	}

	if strings.Contains(out, "primary") {
		t.Errorf("the wired instances must not be reported as shadowed, got log: %s", out)
	}
}
