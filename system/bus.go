package system

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// createEventBus creates a watermill.EventBus based on the deployment config.
// The Driver field maps to watermill backend selection: "gochannel" (default)
// uses Watermill's in-process GoChannel pub/sub. Future drivers (nats, kafka)
// will use watermill.WithBackend to inject external publisher/subscriber.
func createEventBus(deployment DeploymentConfig) (event.Bus, error) {
	for _, busCfg := range deployment.Buses {
		switch busCfg.Driver {
		case "", "gochannel":
			return watermill.NewEventBus(), nil
		default:
			return nil, fmt.Errorf(
				"%w: %q (supported: gochannel)",
				ErrUnknownBusDriver,
				busCfg.Driver,
			)
		}
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
