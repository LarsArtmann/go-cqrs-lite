package main

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// makeSystemMemoryFactory crosses the benchkit bridge: the benchmark runs
// against a system.New deployment (the strategic composition root, ADR-0123)
// via benchkit.FactoryFromSystem instead of a stack preset. Capabilities the
// system does not expose (kv read models) skip with recorded warnings — the
// runner's honesty, not a silent gap.
func makeSystemMemoryFactory() (benchkit.Factory, string, func()) {
	return benchkit.FactoryFromSystem(func(ctx context.Context) (*system.System, error) {
		deployment := system.DeploymentConfig{
			Engines: map[string]system.EngineConfig{
				"primary": {Driver: "memory"},
			},
			Instances: []system.InstanceConfig{
				{Role: system.RoleSourceOfTruth, Engine: "primary"},
				{Role: system.RoleProjections, Engine: "primary"},
			},
		}

		return system.New(ctx, system.DomainConfig{}, deployment)
	}), "", nil
}

func makeMemoryFactory(_ context.Context) (benchkit.Factory, string, func()) {
	return func() (*stack.Bundle, error) { return memory.New() }, "", nil
}
