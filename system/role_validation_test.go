package system_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_NewRejectsUnknownInstanceRole pins M10 F32: a typo'd role used
// to match no wiring at all — the instance was silently ignored (no store
// bound, no error) and the system came up misconfigured. Construction now
// fails with ErrUnknownInstanceRole and the message lists the valid roles.
func TestSystem_NewRejectsUnknownInstanceRole(t *testing.T) {
	t.Parallel()

	_, err := system.New(
		context.Background(),
		system.DomainConfig{},
		system.DeploymentConfig{
			Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
			Instances: []system.InstanceConfig{
				{Role: system.InstanceRole("soure-of-truth"), Engine: "primary"},
			},
		},
	)

	if !errors.Is(err, system.ErrUnknownInstanceRole) {
		t.Fatalf("New error = %v, want ErrUnknownInstanceRole", err)
	}

	for _, want := range []string{
		"soure-of-truth",
		"source-of-truth", "events", "commands", "queries", "snapshots", "projections",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q must mention %q", err, want)
		}
	}
}

// TestSystem_NewImplicitMemorySOT_Advisory pins M10 F33: when no
// source-of-truth/events instance is configured, the implicit in-memory
// fallback must leave an ADVISORY on the ScreamReport instead of passing
// silently — a production config that reaches the fallback by typo loses
// every event on restart.
func TestSystem_NewImplicitMemorySOT_Advisory(t *testing.T) {
	t.Parallel()

	sys, err := system.New(
		context.Background(),
		system.DomainConfig{},
		system.DeploymentConfig{}, // no engines, no instances
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = sys.Close() }()

	report := sys.ScreamReport()
	if report == nil {
		t.Fatal("ScreamReport must exist")
	}

	var found bool

	for _, d := range report.Diagnostics {
		if d.Rule == "sot.implicit_memory" {
			found = true

			if d.Tier != system.TierAdvisory {
				t.Fatalf("implicit-memory SOT tier = %s, want ADVISORY", d.Tier)
			}

			if !strings.Contains(d.Detail, "in-memory") {
				t.Fatalf("detail must explain the in-memory fallback: %q", d.Detail)
			}
		}
	}

	if !found {
		t.Fatalf("no sot.implicit_memory diagnostic in report: %+v", report.Diagnostics)
	}
}

// TestSystem_NewExplicitSOT_NoImplicitMemoryAdvisory guards the inverse: an
// explicitly configured source-of-truth engine must NOT trip the advisory.
func TestSystem_NewExplicitSOT_NoImplicitMemoryAdvisory(t *testing.T) {
	t.Parallel()

	sys, err := system.New(
		context.Background(),
		system.DomainConfig{},
		system.DeploymentConfig{
			Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
			Instances: []system.InstanceConfig{
				{Role: system.RoleSourceOfTruth, Engine: "primary"},
			},
		},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = sys.Close() }()

	for _, d := range sys.ScreamReport().Diagnostics {
		if d.Rule == "sot.implicit_memory" {
			t.Fatalf("explicit source-of-truth must not trip the implicit-memory advisory: %+v", d)
		}
	}
}

// TestSystem_NewUnknownEngineListsContext pins M10 F34: ErrUnknownEngine
// messages carry the configured engine names and the registered drivers, so a
// name mismatch is diagnosable from the error alone.
func TestSystem_NewUnknownEngineListsContext(t *testing.T) {
	t.Parallel()

	_, err := system.New(
		context.Background(),
		system.DomainConfig{},
		system.DeploymentConfig{
			Engines: map[string]system.EngineConfig{
				"primary": {Driver: "memory"},
				"spare":   {Driver: "memory"},
			},
			Instances: []system.InstanceConfig{
				{Role: system.RoleSourceOfTruth, Engine: "primar"},
			},
		},
	)

	if !errors.Is(err, system.ErrUnknownEngine) {
		t.Fatalf("New error = %v, want ErrUnknownEngine", err)
	}

	for _, want := range []string{"primar", "primary", "spare", "memory"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q must mention %q", err, want)
		}
	}
}
