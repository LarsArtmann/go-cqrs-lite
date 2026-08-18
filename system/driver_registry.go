package system

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// createEngineFromDriver looks up the driver in the metaengine registry and
// constructs an engine. It bridges system.EngineConfig (operator-facing,
// parsed from YAML/TOML via koanf) to metaengine.DriverConfig (the registry's
// internal config struct). durability is the per-engine tier resolved from
// the instances bound to this engine (resolveEngineDurability); empty means
// unspecified — engine defaults.
func createEngineFromDriver(
	ctx context.Context, cfg EngineConfig, durability metaengine.DurabilityTier,
) (metaengine.Engine, error) {
	factory, err := metaengine.LookupDriver(cfg.Driver)
	if err != nil {
		return nil, err
	}

	eng, err := factory(ctx, metaengine.DriverConfig{
		DSN:        cfg.DSN,
		Pragmas:    cfg.Pragmas,
		Priority:   cfg.Priority,
		Durability: durability,
	})
	if err != nil {
		return nil, fmt.Errorf("system: driver %q create: %w", cfg.Driver, err)
	}

	return eng, nil
}
