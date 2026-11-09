package system

import (
	"maps"
	"slices"
)

// Standard fills the canonical three-tier deployment topology (ADR-0165) into
// a DeploymentConfig that declares engines but no instances:
//
//  1. a source-of-truth instance on the WAL engine — events, commands,
//     queries, snapshots, checkpoints;
//  2. a dedicated queries instance (the query audit log) on the same engine,
//     so the audit log is a named concern that can later move to its own
//     engine by changing one name;
//  3. a projections instance pooling EVERY declared engine — the cost-based
//     planner routes each query across the pool.
//
// The WAL engine is the engine named "primary" when declared, else the
// alphabetically first engine name — deterministic across boots, like every
// engine iteration in this package.
//
// Standard is a data filler, not a builder: a config that already declares
// Instances passes through unchanged, and a config with no Engines passes
// through unchanged (system.New then falls back to the in-memory engine with
// its sot.implicit_memory advisory). Engines, buses, durability tiers, and
// priorities remain operator-owned. Apply it before construction:
//
//	sys, err := system.New(ctx, domain, system.Standard(deploy))
func Standard(d DeploymentConfig) DeploymentConfig {
	if len(d.Instances) > 0 || len(d.Engines) == 0 {
		return d
	}

	filled := d
	filled.Instances = []InstanceConfig{
		{
			Role:        RoleSourceOfTruth,
			Collections: []string{"events", "commands", "queries", "snapshots", "checkpoints"},
			Engine:      walEngineName(d.Engines),
		},
		{
			Role:        RoleQueries,
			Collections: []string{"queries"},
			Engine:      walEngineName(d.Engines),
		},
		{
			Role:        RoleProjections,
			Collections: []string{"projections"},
			Engines:     slices.Sorted(maps.Keys(d.Engines)),
		},
	}

	return filled
}

// walEngineName picks the engine backing the journal: the conventional
// "primary" when declared, else the alphabetically first engine name.
func walEngineName(engines map[string]EngineConfig) string {
	if _, ok := engines["primary"]; ok {
		return "primary"
	}

	return slices.Sorted(maps.Keys(engines))[0]
}
