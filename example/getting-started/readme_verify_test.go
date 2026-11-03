package main

// Temporary compile-verification of the README.md Quick Start snippets.
// Assembles the fenced Go blocks into one function and type-checks them.

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // register the "sqlite" driver
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

type readmeIncremented struct {
	Amount int `json:"amount"`
}

type readmeCounterState struct{ Value int }

func readmeApplyCounter(s readmeCounterState, evt event.Event) (readmeCounterState, error) {
	if evt.Type() != "counter.incremented" {
		return s, nil
	}
	p, err := event.DecodePayloadAuto[readmeIncremented](evt)
	if err != nil {
		return s, err
	}
	s.Value += p.Amount
	return s, nil
}

type readmeIncrementCmd struct {
	*command.BasicCommand

	Amount int
}

type readmeCounterView struct {
	Value int `json:"value"`
}

type readmeCounterLookup struct{}

func readmeQuickStart(ctx context.Context, counterID id.StreamID) error {
	counterViews := metaengine.Query[readmeCounterLookup, readmeCounterView]("counter_views",
		metaengine.OnRecordTyped("counter.incremented",
			projectionadapter.EventWithID[readmeIncremented]{},
			func(_ record.Record, e projectionadapter.EventWithID[readmeIncremented]) (string, readmeCounterView) {
				return e.ID, readmeCounterView{Value: e.Payload.Amount}
			}),
		metaengine.OnRecordTyped("counter.incremented",
			projectionadapter.EventWithID[readmeIncremented]{},
			func(_ record.Record, e projectionadapter.EventWithID[readmeIncremented], prev readmeCounterView) readmeCounterView {
				prev.Value += e.Payload.Amount
				return prev
			}),
	)

	sys, err := system.New(ctx,
		system.DomainConfig{
			Commands: func(sys *system.System) {
				_ = system.RegisterDecider(sys, "Counter", decider.Decider[readmeCounterState]{
					Initial: readmeCounterState{},
					Apply:   readmeApplyCounter,
				})
				_ = system.RegisterCommand[readmeIncrementCmd, readmeCounterState](sys, "counter.increment",
					func(cmdCtx context.Context, cmd readmeIncrementCmd) system.Op[readmeCounterState] {
						return system.Execute(cmdCtx, cmd.StreamID(), "Counter",
							func(_ readmeCounterState, v event.Version) ([]event.Event, error) {
								evt, err := event.New("counter.incremented", cmd.StreamID(), "Counter",
									v.Increment(), readmeIncremented{Amount: cmd.Amount})
								if err != nil {
									return nil, err
								}
								return []event.Event{evt}, nil
							})
					})
			},
			Projections: []system.ProjectionDeclaration{system.RawQuery(counterViews)},
		},
		system.DeploymentConfig{
			Engines: map[string]system.EngineConfig{
				"primary": {Driver: "memory"},
			},
			Instances: []system.InstanceConfig{
				{Role: system.RoleSourceOfTruth, Engine: "primary"},
				{Role: system.RoleProjections, Engine: "primary"},
			},
		},
	)
	if err != nil {
		return err
	}
	defer metaengine.DeferClose(sys)

	basic, err := command.New("counter.increment", counterID)
	if err != nil {
		return err
	}
	if err := sys.CommandDispatcher().Dispatch(ctx,
		readmeIncrementCmd{BasicCommand: basic, Amount: 5}); err != nil {
		return err
	}

	reader := metaengine.NewReader[readmeCounterView](sys.MetaEngine(), "counter_views")
	_, _, err = reader.Get(ctx, counterID.Get())
	return err
}

var _ = readmeQuickStart
