package system

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// buildEventBus creates the event bus based on the deployment config.
// If the deployment configures a bus with a known driver, the driver factory
// is used. Otherwise a Watermill GoChannel bus is created (D9).
func buildEventBus(deployment DeploymentConfig) (event.Bus, error) {
	// If a bus is explicitly configured, use the driver registry.
	for _, busCfg := range deployment.Buses {
		if busCfg.Driver == "" {
			continue
		}

		// "gochannel" is the built-in default: Watermill GoChannel.
		if busCfg.Driver == "gochannel" {
			return watermill.NewEventBus(), nil
		}

		factory, err := lookupBusDriver(busCfg.Driver)
		if err != nil {
			return nil, fmt.Errorf("system: bus driver %q: %w", busCfg.Driver, err)
		}

		bus, err := factory(busCfg)
		if err != nil {
			return nil, fmt.Errorf("system: bus driver %q create: %w", busCfg.Driver, err)
		}

		eb, ok := bus.(event.Bus)
		if !ok {
			return nil, fmt.Errorf(
				"system: bus driver %q returned %T: %w",
				busCfg.Driver, bus, ErrBusDriverNotEventBus,
			)
		}

		return eb, nil
	}

	// Default: Watermill GoChannel (in-process pub/sub).
	return watermill.NewEventBus(), nil
}

// buildPublisher creates the publisher for the decider repository.
// If the source-of-truth has multiple Publish targets, returns a MultiBus
// wrapping a Watermill bus for each target. Otherwise returns the local bus.
func buildPublisher(deployment DeploymentConfig, localBus event.Publisher) event.Publisher {
	for _, inst := range deployment.Instances {
		if isSourceOfTruth(inst.Role) && len(inst.Publish) > 1 {
			buses := make([]event.Publisher, len(inst.Publish))

			for i := range inst.Publish {
				buses[i] = watermill.NewEventBus()
			}

			// Include the local bus so local subscribers still receive events.
			buses = append([]event.Publisher{localBus}, buses...)

			return NewMultiBus(buses...)
		}
	}

	return localBus
}
