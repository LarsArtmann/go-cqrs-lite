package system_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// gateDomain builds a minimal domain whose evolution consumes "evo.created"
// via a Lookup projection, parameterized by the declared event universe.
func gateDomain(events []event.Type, disable bool) system.DomainConfig {
	return system.DomainConfig{
		Evolutions: []system.EvolutionSpec{
			system.OnEvolution(
				system.Evolve[EvoView]("gate_tasks"),
				"evo.created", EvoCreated{},
			).Done(),
		},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[EvoView]("gate_lookup").Done(),
		},
		Events:                     events,
		DisableCoeffectValidation: disable,
	}
}

func gateDeployment() system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}
}

// TestSystem_CoeffectGate_DanglingSubscriptionErrors verifies the hard error:
// an evolution/projection consuming a type outside the declared universe
// fails composition instead of silently never updating.
func TestSystem_CoeffectGate_DanglingSubscriptionErrors(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := system.New(ctx,
		gateDomain([]event.Type{"something.else"}, false),
		gateDeployment(),
	)

	if !errors.Is(err, system.ErrDanglingEventSubscription) {
		t.Fatalf("expected ErrDanglingEventSubscription, got %v", err)
	}

	if sys != nil {
		t.Fatal("system should be nil on gate failure")
	}

	if want := "evo.created"; !strings.Contains(err.Error(), want) {
		t.Fatalf("error should name the dangling type %q: %v", want, err)
	}
}

// TestSystem_CoeffectGate_UnconsumedEventAdvisory verifies the soft side:
// a declared type nothing consumes still constructs, surfacing an advisory
// diagnostic (dead events are legitimate for audit-only journals).
func TestSystem_CoeffectGate_UnconsumedEventAdvisory(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := system.New(ctx,
		gateDomain([]event.Type{"evo.created", "evo.audit-only"}, false),
		gateDeployment(),
	)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	report := sys.ScreamReport()

	var found bool

	for _, d := range report.Diagnostics {
		if d.Rule == "coeffect.unconsumed_event" && strings.Contains(d.Detail, "evo.audit-only") {
			found = true
		}
	}

	if !found {
		t.Fatalf("expected unconsumed-event advisory for evo.audit-only, got %+v",
			report.Diagnostics)
	}
}

// TestSystem_CoeffectGate_DisabledSkips verifies the escape hatch: the gate
// is silent when explicitly disabled, even with a dangling subscription.
func TestSystem_CoeffectGate_DisabledSkips(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := system.New(ctx,
		gateDomain([]event.Type{"something.else"}, true),
		gateDeployment(),
	)
	if err != nil {
		t.Fatalf("system.New with disabled gate: %v", err)
	}

	defer sys.Close()
}

// TestSystem_CoeffectGate_RecordTypeAliasInterop pins the ADR-0111 alias:
// a declared universe built from record.Type values matches event.Type
// consumption — the gate must not regress the alias lockstep.
func TestSystem_CoeffectGate_RecordTypeAliasInterop(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// record.Type IS event.Type (alias): declaring through the record
	// vocabulary satisfies the event-typed gate. This line only compiles
	// while the alias holds — the same lockstep trick as the core tests.
	declared := []event.Type{record.Type("evo.created")}

	sys, err := system.New(ctx, gateDomain(declared, false), gateDeployment())
	if err != nil {
		t.Fatalf("system.New with record.Type declarations: %v", err)
	}

	defer sys.Close()
}
