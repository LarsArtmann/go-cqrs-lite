package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

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

// go-cqrs-lite in ~140 lines — the simplest useful example.
//
// Pipeline: Command → Decider → Event Store → Projection Host → metaengine
// read model. The consumer declares only events, state, decide logic, and
// folds. The deployer picks engines in ONE place (in-memory here); swap
// memory → sqlite/postgres/pebble by changing a single EngineConfig line —
// the consumer code doesn't change.

const (
	evtIncremented = event.Type("counter.incremented")
	cmdIncrement   = command.Type("counter.increment")
	streamType     = "Counter"

	primaryEngine            = "primary"
	counterCollection        = "counter_views"
	projectionSettleDeadline = 5 * time.Second
	projectionPollInterval   = 20 * time.Millisecond
)

var errProjectionConvergence = errors.New("projection did not converge")

type IncrementedPayload struct {
	Amount int `json:"amount"`
}

type CounterState struct{ Value int }

func applyCounter(s CounterState, evt event.Event) (CounterState, error) {
	if evt.Type() != evtIncremented {
		return s, nil
	}

	p, err := event.DecodePayloadAuto[IncrementedPayload](evt)
	if err != nil {
		return s, err
	}

	s.Value += p.Amount

	return s, nil
}

// increment returns a DecideFunc that the Repository executes against
// the current (replayed) state. Version enables optimistic concurrency.
func increment(aggID id.StreamID, amount int) decider.DecideFunc[CounterState] {
	return func(_ CounterState, v event.Version) ([]event.Event, error) {
		evt, err := event.New(evtIncremented, aggID, streamType,
			v.Increment(), IncrementedPayload{Amount: amount})
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

// IncrementCmd is the only write-side API callers need; the dispatcher
// routes it to the decider registered for streamType.
type IncrementCmd struct {
	*command.BasicCommand

	Amount int
}

// CounterView is the read model: a metaengine Map query folded from events.
type CounterView struct {
	Value int `json:"value"`
}

// counterProjection declares the read model as folds over the event types.
// The first fold creates a view on the stream's first event; the second
// updates existing views. metaengine picks the right one per event.
func counterProjection() ([]system.ProjectionDeclaration, *projectionadapter.TypeDecoder) {
	query := metaengine.Query[counterLookup, CounterView](counterCollection,
		metaengine.OnRecordTyped(string(evtIncremented),
			projectionadapter.EventWithID[IncrementedPayload]{},
			func(_ record.Record, e projectionadapter.EventWithID[IncrementedPayload]) (string, CounterView) {
				return e.ID, CounterView{Value: e.Payload.Amount}
			},
		),
		metaengine.OnRecordTyped(string(evtIncremented),
			projectionadapter.EventWithID[IncrementedPayload]{},
			func(_ record.Record, e projectionadapter.EventWithID[IncrementedPayload], prev CounterView) CounterView {
				prev.Value += e.Payload.Amount

				return prev
			},
		),
	)

	decoder := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(evtIncremented, IncrementedPayload{}),
	)

	return []system.ProjectionDeclaration{system.RawQuery(query)}, decoder
}

// counterLookup is the (unused) query-input type; reads go through
// metaengine.NewReader.Get keyed by the fold-returned stream ID.
type counterLookup struct{}

// buildSystem wires the composition root. dsn == "" selects the in-memory
// engine; any other value selects SQLite at that path. This is the ONE
// function an operator changes to deploy differently. Drivers self-register
// via init(), so each backend the deployment might use needs one blank
// import (the sqlite import above).
func buildSystem(ctx context.Context, dsn string) (*system.System, error) {
	projection, typeDecoder := counterProjection()

	engine := system.EngineConfig{Driver: "memory"}
	if dsn != "" {
		engine = system.EngineConfig{Driver: "sqlite", DSN: dsn}
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{primaryEngine: engine},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: primaryEngine},
			{Role: system.RoleProjections, Engine: primaryEngine},
		},
	}

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			if err := system.RegisterDecider(sys, streamType, decider.Decider[CounterState]{
				Initial: CounterState{},
				Apply:   applyCounter,
			}); err != nil {
				panic(err)
			}

			err := system.RegisterCommand[IncrementCmd, CounterState](sys, cmdIncrement,
				func(cmdCtx context.Context, cmd IncrementCmd) system.Op[CounterState] {
					return system.Execute(cmdCtx, cmd.StreamID(), streamType,
						increment(cmd.StreamID(), cmd.Amount))
				})
			if err != nil {
				panic(err)
			}
		},
		Projections:           projection,
		ProjectionTypeDecoder: typeDecoder,
	}

	return system.New(ctx, domain, deployment)
}

// runPipeline starts the projection host (it replays the journal, then
// follows live), dispatches the increments, and reads the projected view,
// polling until the projection catches up.
func runPipeline(ctx context.Context, sys *system.System, counterID id.StreamID) (CounterView, error) {
	hostCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() { _ = sys.ProjectionHost().Start(hostCtx) }()

	for _, amt := range []int{5, 3, 2} {
		bc, err := command.New(cmdIncrement, counterID)
		if err != nil {
			return CounterView{}, err
		}

		if err := sys.CommandDispatcher().Dispatch(ctx, IncrementCmd{BasicCommand: bc, Amount: amt}); err != nil {
			return CounterView{}, err
		}
	}

	reader := metaengine.NewReader[CounterView](sys.MetaEngine(), counterCollection)

	deadline := time.Now().Add(projectionSettleDeadline)

	for {
		view, found, err := reader.Get(ctx, counterID.String())
		if err != nil {
			return CounterView{}, err
		}

		if found && view.Value == 10 {
			return view, nil
		}

		if time.Now().After(deadline) {
			return CounterView{}, fmt.Errorf("%w: got %+v (found=%t)", errProjectionConvergence, view, found)
		}

		time.Sleep(projectionPollInterval)
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	sys, err := buildSystem(ctx, "")
	if err != nil {
		return err
	}
	defer func() { _ = sys.Close() }()

	counterID := id.NewStreamID()

	view, err := runPipeline(ctx, sys, counterID)
	if err != nil {
		return err
	}

	fmt.Printf("Counter %s: value=%d (expected 10)\n", counterID, view.Value)

	// Swap in-memory → persistent by changing ONE EngineConfig line:
	//   system.EngineConfig{Driver: "sqlite", DSN: "counter.db"}
	// The domain code above doesn't change.

	return nil
}
