package system_test

import (
	"reflect"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestStandard_FillsCanonicalTopology(t *testing.T) {
	got := system.Standard(system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"proj":    {Driver: "sqlite"},
			"primary": {Driver: "sqlite", DSN: "file:app.db"},
		},
	})

	want := []system.InstanceConfig{
		{
			Role: system.RoleSourceOfTruth,
			Collections: []string{
				"events", "commands", "queries", "snapshots", "checkpoints",
			},
			Engine: "primary",
		},
		{
			Role:        system.RoleQueries,
			Collections: []string{"queries"},
			Engine:      "primary",
		},
		{
			Role:        system.RoleProjections,
			Collections: []string{"projections"},
			Engines:     []string{"primary", "proj"},
		},
	}

	if !reflect.DeepEqual(got.Instances, want) {
		t.Fatalf("Standard() instances = %+v, want %+v", got.Instances, want)
	}
}

func TestStandard_PrefersPrimaryNameOverSortedFirst(t *testing.T) {
	got := system.Standard(system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"aaa":    {Driver: "memory"},
			"zzz":    {Driver: "memory"},
			"middle": {Driver: "memory"},
		},
	})

	if got.Instances[0].Engine != "aaa" {
		t.Fatalf("WAL engine = %q, want alphabetically-first %q", got.Instances[0].Engine, "aaa")
	}

	if !reflect.DeepEqual(got.Instances[2].Engines, []string{"aaa", "middle", "zzz"}) {
		t.Fatalf("projection pool = %v, want sorted [aaa middle zzz]", got.Instances[2].Engines)
	}
}

func TestStandard_IdempotentPassthrough(t *testing.T) {
	existing := []system.InstanceConfig{
		{Role: system.RoleEvents, Engine: "one"},
	}

	cfg := system.DeploymentConfig{
		Engines:   map[string]system.EngineConfig{"one": {Driver: "sqlite"}},
		Instances: existing,
	}

	if got := system.Standard(cfg); !reflect.DeepEqual(got, cfg) {
		t.Fatalf("Standard() with instances = %+v, want unchanged %+v", got, cfg)
	}
}

func TestStandard_NoEnginesUnchanged(t *testing.T) {
	cfg := system.DeploymentConfig{}

	if got := system.Standard(cfg); !reflect.DeepEqual(got, cfg) {
		t.Fatalf("Standard() with no engines = %+v, want unchanged", got)
	}
}

// TestStandard_BootsWithNew proves the filled topology actually constructs:
// sqlite WAL + dedicated query audit + projection pool all wire without
// errors, and no implicit-memory advisory fires.
func TestStandard_BootsWithNew(t *testing.T) {
	t.Parallel()

	deploy := system.Standard(system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "sqlite", DSN: t.TempDir() + "/std.db"},
		},
	})

	sys, err := system.New(t.Context(), system.DomainConfig{}, deploy)
	if err != nil {
		t.Fatalf("New(Standard(cfg)) failed: %v", err)
	}
	defer func() { _ = sys.Close() }()

	for _, diag := range sys.ScreamReport().Diagnostics {
		if diag.Rule == "sot.implicit_memory" {
			t.Fatalf("implicit-memory advisory fired on a Standard deployment: %s", diag.Detail)
		}
	}
}
