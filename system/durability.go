package system

import (
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// resolveEngineDurability computes the durability tier each named engine
// should be constructed with, from the tiers requested by the instances
// bound to it.
//
// Rules:
//   - An instance tier of "" (unspecified) never constrains the engine: the
//     instance accepts engine defaults.
//   - All explicit tiers on one engine must agree. An engine is constructed
//     ONCE, so there is no per-instance durability; two instances requesting
//     different tiers on the same engine is a deployment conflict. Failing
//     construction keeps the config honest instead of silently granting one
//     instance's tier to the other.
//   - Invalid tier values fail immediately with the offending instance named.
func resolveEngineDurability(
	deployment DeploymentConfig,
) (map[string]metaengine.DurabilityTier, error) {
	claimed := make(map[string]metaengine.DurabilityTier)

	for _, inst := range deployment.Instances {
		tier := metaengine.DurabilityTier(inst.Durability)

		if err := metaengine.ValidateDurabilityTier(tier); err != nil {
			return nil, fmt.Errorf("system: instance %q durability: %w", inst.Role, err)
		}

		if tier == "" {
			continue
		}

		for _, engineName := range instanceEngineNames(inst) {
			if prior, ok := claimed[engineName]; ok && prior != tier {
				return nil, fmt.Errorf(
					"%w: engine %q requested as %q and %q by different instances — engines are constructed once, so all instances sharing an engine must agree",
					ErrDurabilityConflict,
					engineName,
					prior,
					tier,
				)
			}

			claimed[engineName] = tier
		}
	}

	return claimed, nil
}

// instanceEngineNames returns every engine name an instance references: the
// Engines list plus the singular Engine field. The result never aliases the
// instance's slices.
func instanceEngineNames(inst InstanceConfig) []string {
	names := make([]string, 0, len(inst.Engines)+1)
	names = append(names, inst.Engines...)

	if inst.Engine != "" {
		names = append(names, inst.Engine)
	}

	return names
}
