package system_test

import (
	"context"
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_Durability_WiredToEngine verifies an explicit instance tier
// flows through to engine construction (SQLite maps it to PRAGMA
// synchronous) and construction succeeds.
func TestSystem_Durability_WiredToEngine(t *testing.T) {
	t.Parallel()

	sys, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"db": {Driver: "sqlite"}},
		Instances: []system.InstanceConfig{{
			Role:       system.RoleSourceOfTruth,
			Engine:     "db",
			Durability: system.DurabilityStrict,
		}},
	})
	if err != nil {
		t.Fatalf("system.New with strict sqlite: %v", err)
	}

	defer sys.Close()
}

// TestSystem_Durability_AgreeingInstances verifies two instances sharing an
// engine with the SAME explicit tier construct fine (dedicated role instance
// tiers count toward resolution too).
func TestSystem_Durability_AgreeingInstances(t *testing.T) {
	t.Parallel()

	sys, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"db": {Driver: "sqlite"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "db", Durability: system.DurabilityRelaxed},
			{Role: system.RoleCommands, Engine: "db", Durability: system.DurabilityRelaxed},
		},
	})
	if err != nil {
		t.Fatalf("system.New with agreeing tiers: %v", err)
	}

	defer sys.Close()
}

// TestSystem_Durability_Conflict verifies two instances requesting different
// tiers on one engine fail construction: the engine is constructed once, so
// there is no per-instance durability to grant.
func TestSystem_Durability_Conflict(t *testing.T) {
	t.Parallel()

	_, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"db": {Driver: "sqlite"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "db", Durability: system.DurabilityStrict},
			{
				Role:       system.RoleProjections,
				Engines:    []string{"db"},
				Durability: system.DurabilityRelaxed,
			},
		},
	})
	if !errors.Is(err, system.ErrDurabilityConflict) {
		t.Fatalf("conflicting tiers error = %v, want ErrDurabilityConflict", err)
	}
}

// TestSystem_Durability_InvalidTier verifies a bogus tier value fails with
// the instance named.
func TestSystem_Durability_InvalidTier(t *testing.T) {
	t.Parallel()

	_, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"db": {Driver: "sqlite"}},
		Instances: []system.InstanceConfig{{
			Role:       system.RoleSourceOfTruth,
			Engine:     "db",
			Durability: "bogus",
		}},
	})
	if !errors.Is(err, metaengine.ErrUnsupportedDurability) {
		t.Fatalf("bogus tier error = %v, want ErrUnsupportedDurability", err)
	}
}

// TestSystem_Durability_MemoryStrictFails verifies the memory driver rejects
// a strict request loudly (in-process memory cannot promise fsync
// durability) and system surfaces the failure.
func TestSystem_Durability_MemoryStrictFails(t *testing.T) {
	t.Parallel()

	_, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"db": {Driver: "memory"}},
		Instances: []system.InstanceConfig{{
			Role:       system.RoleSourceOfTruth,
			Engine:     "db",
			Durability: system.DurabilityStrict,
		}},
	})
	if !errors.Is(err, metaengine.ErrUnsupportedDurability) {
		t.Fatalf("strict memory error = %v, want ErrUnsupportedDurability", err)
	}
}

// TestSystem_Durability_UnspecifiedKeepsDefaults verifies empty tiers pass
// through as engine defaults — the behavior every pre-tier deployment has.
func TestSystem_Durability_UnspecifiedKeepsDefaults(t *testing.T) {
	t.Parallel()

	sys, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"db": {Driver: "sqlite"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "db"},
			{Role: system.RoleCommands, Engine: "db"},
		},
	})
	if err != nil {
		t.Fatalf("system.New with unspecified tiers: %v", err)
	}

	defer sys.Close()
}
