package system

import (
	"log/slog"
	"slices"
	"strings"
)

// warnShadowedInstances reports the instances New wires but never uses. The
// System holds one event store and one projection store, so a second
// source-of-truth/events instance replaces the first one's event store and
// only the first projections instance is planned; the publish buses of every
// later source-of-truth instance are ignored as well. Construction is not
// refused: a deployment may deliberately split roles this way.
func warnShadowedInstances(deployment DeploymentConfig) {
	var (
		seenSoT, seenProjections bool
		shadowed                 []string
	)

	for _, inst := range deployment.Instances {
		switch {
		case isSourceOfTruth(inst.Role) && !seenSoT:
			seenSoT = true
		case inst.Role == RoleProjections && !seenProjections:
			seenProjections = true
		case isSourceOfTruth(inst.Role) || inst.Role == RoleProjections:
			shadowed = append(shadowed,
				string(inst.Role)+" on "+strings.Join(instanceEngines(inst), ", "))
		}
	}

	if len(shadowed) == 0 {
		return
	}

	slog.Warn(
		"system: instance declarations shadowed: only the first projections instance is planned "+
			"and the last source-of-truth/events instance provides the event store; "+
			"these instances are opened, closed on shutdown and otherwise unused",
		"shadowed", strings.Join(slices.Sorted(slices.Values(shadowed)), "; "),
	)
}

func instanceEngines(inst InstanceConfig) []string {
	if len(inst.Engines) > 0 {
		return inst.Engines
	}

	if inst.Engine != "" {
		return []string{inst.Engine}
	}

	return []string{"<no engine>"}
}
