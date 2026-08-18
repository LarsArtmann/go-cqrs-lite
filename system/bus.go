package system

import (
	"fmt"
	"io"
	"maps"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// createEventBus creates a watermill.EventBus based on the deployment config.
// The Driver field maps to watermill backend selection: "gochannel" (default)
// uses Watermill's in-process GoChannel pub/sub. Future drivers (nats, kafka)
// will use watermill.WithBackend to inject external publisher/subscriber.
//
// Every configured bus entry is validated — an unknown driver anywhere in the
// map fails construction, even if another (valid) entry is iterated first.
// Iteration order is by sorted bus name so the selected bus is deterministic
// across boots.
func createEventBus(deployment DeploymentConfig) (event.Bus, error) {
	// Validate every entry first — an unknown driver anywhere in the map must
	// fail construction even when a valid entry sorts first.
	for _, name := range slices.Sorted(maps.Keys(deployment.Buses)) {
		busCfg := deployment.Buses[name]
		switch busCfg.Driver {
		case "", "gochannel":
		default:
			return nil, fmt.Errorf(
				"%w: %q (bus %q, supported: gochannel)",
				ErrUnknownBusDriver,
				busCfg.Driver,
				name,
			)
		}
	}

	// Deterministic selection: all supported drivers map to the same
	// in-process bus today, so any entry yields the same result. When multiple
	// drivers exist, extend this to pick by sorted bus name so the choice is
	// stable across boots.
	if len(deployment.Buses) > 0 {
		return watermill.NewEventBus(), nil
	}

	// Default: Watermill GoChannel (in-process pub/sub).
	return watermill.NewEventBus(), nil
}

// fanoutBus pairs a fan-out bus with its Publish target name from the
// deployment config. The name is what the operator wrote under Publish and
// what [System.PublisherFor] resolves.
type fanoutBus struct {
	name      string
	publisher event.Publisher
	closer    io.Closer
}

// buildPublisher creates the publisher for the decider repository.
// If the source-of-truth has multiple Publish targets, returns a MultiBus
// wrapping a Watermill bus per named target. Otherwise returns the local
// bus. The second return value lists the fan-out buses created (the caller
// must register them for lifecycle closure).
func buildPublisher(
	deployment DeploymentConfig, localBus event.Publisher,
) (event.Publisher, []fanoutBus) {
	for _, inst := range deployment.Instances {
		if isSourceOfTruth(inst.Role) && len(inst.Publish) > 1 {
			multi := NewMultiBus(localBus)
			fanouts := make([]fanoutBus, 0, len(inst.Publish))

			for _, target := range inst.Publish {
				bus := watermill.NewEventBus()
				multi.AddNamedPublisher(target, bus)
				fanouts = append(fanouts, fanoutBus{
					name:      target,
					publisher: bus,
					closer:    bus,
				})
			}

			return multi, fanouts
		}
	}

	return localBus, nil
}
