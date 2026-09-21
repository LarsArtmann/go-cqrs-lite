package system_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Tombstone auto-fold domain (evolution inheritance, zero fold closures) ──
//
// Pins the ADR-0114 × ADR-0116 contract: ONE Evolution declares
// Created/Deleted convention samples; the Lookup projection declares NO
// samples and inherits every fold by result type. The `*Deleted` event must
// remove the read-model row (type-driven Remove, never the deprecated
// metadata tombstone path), and a later `*Created` for the same key must
// rebirth it.

type EvoTombCreated struct {
	ID    string
	Title string
}

type EvoTombDeleted struct {
	ID string
}

type EvoTombView struct {
	ID    string
	Title string
}

type evoTombState struct {
	Title string
	Exists bool
}

func applyEvoTomb(state evoTombState, evt event.Event) (evoTombState, error) {
	switch evt.Type() {
	case "evotomb.created":
		var p EvoTombCreated

		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return state, err
		}

		state.Title = p.Title
		state.Exists = true
	case "evotomb.deleted":
		state.Exists = false
	}

	return state, nil
}

func newTombstoneDomain() system.DomainConfig {
	tasks := system.OnEvolution(
		system.Evolve[EvoTombView]("evo_tomb_views"),
		"evotomb.created", EvoTombCreated{},
	)
	evolution := system.OnEvolution(tasks, "evotomb.deleted", EvoTombDeleted{})

	return system.DomainConfig{
		Evolutions: []system.EvolutionSpec{evolution.Done()},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[EvoTombView]("evo_tomb_views").Done(),
		},
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "EvoTomb", decider.Decider[evoTombState]{
				Initial: evoTombState{},
				Apply:   applyEvoTomb,
			})

			system.RegisterCommand[*command.BasicCommand, evoTombState](sys, "evotomb.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[evoTombState] {
					return system.Execute(ctx, cmd.StreamID(), "EvoTomb",
						func(state evoTombState, ver event.Version) ([]event.Event, error) {
							var p EvoTombCreated

							if err := json.Unmarshal(cmd.Payload(), &p); err != nil {
								return nil, err
							}

							if state.Exists {
								return nil, errors.New("already exists")
							}

							return []event.Event{mustEvent(event.New(
								"evotomb.created", cmd.StreamID(), "EvoTomb", ver+1, p,
							))}, nil
						})
				})

			system.RegisterCommand[*command.BasicCommand, evoTombState](sys, "evotomb.delete",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[evoTombState] {
					return system.Execute(ctx, cmd.StreamID(), "EvoTomb",
						func(state evoTombState, ver event.Version) ([]event.Event, error) {
							if !state.Exists {
								return nil, errors.New("not found")
							}

							return []event.Event{mustEvent(event.New(
								"evotomb.deleted", cmd.StreamID(), "EvoTomb", ver+1,
								EvoTombDeleted{ID: cmd.StreamID().String()},
							))}, nil
						})
				})
		},
	}
}

func tombstoneDeployment() system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}
}

func dispatchTomb(t *testing.T, ctx context.Context, sys *system.System,
	cmdType command.Type, streamID id.StreamID, payload any,
) {
	t.Helper()

	basic, err := command.New(cmdType, streamID, payload)
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, basic); err != nil {
		t.Fatalf("dispatch %s: %v", cmdType, err)
	}
}

// awaitTombView polls the inherited-fold lookup until view resolves (nil) or
// metaengine.ErrNotFound surfaces (errGone).
func awaitTombView(ctx context.Context, sys *system.System, key string,
) (view *EvoTombView, errGone bool) {
	deadline := loadScaledDeadline(5 * time.Second)

	for time.Now().Before(deadline) {
		got, qerr := metaengine.ExecuteTyped[system.LookupInput[string], EvoTombView](
			ctx, sys.MetaEngine(), system.LookupInput[string]{ID: key},
		)
		if qerr == nil {
			return &got, false
		}

		if errors.Is(qerr, metaengine.ErrNotFound) {
			return nil, true
		}

		time.Sleep(25 * time.Millisecond)
	}

	return nil, false
}

func startTombstoneSystem(t *testing.T, ctx context.Context) *system.System {
	t.Helper()

	sys, err := system.New(ctx, newTombstoneDomain(), tombstoneDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	t.Cleanup(func() { sys.Close() })

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	return sys
}

// TestSystem_EvolutionInheritance_TombstoneRemovesRow pins the tombstone
// auto-fold: through the INHERITED evolution folds (the projection declares
// no samples of its own), the *Deleted convention event removes the read-model
// row — the read becomes metaengine.ErrNotFound, not a stale ghost row.
func TestSystem_EvolutionInheritance_TombstoneRemovesRow(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sys := startTombstoneSystem(t, ctx)

	streamID := id.NewStreamID()
	dispatchTomb(t, ctx, sys, "evotomb.create", streamID,
		EvoTombCreated{ID: streamID.String(), Title: "tombstone me"})

	view, gone := awaitTombView(ctx, sys, streamID.String())
	if gone || view == nil {
		t.Fatalf("created view never appeared (gone=%v)", gone)
	}

	if view.Title != "tombstone me" {
		t.Fatalf("inherited insert fold: title %q, want %q", view.Title, "tombstone me")
	}

	dispatchTomb(t, ctx, sys, "evotomb.delete", streamID, nil)

	deadline := loadScaledDeadline(5 * time.Second)

	for time.Now().Before(deadline) {
		_, gone := awaitTombView(ctx, sys, streamID.String())
		if gone {
			return
		}

		time.Sleep(25 * time.Millisecond)
	}

	t.Fatal("tombstone auto-fold did not remove the inherited read-model row")
}

// TestSystem_EvolutionInheritance_RebirthAfterTombstone pins rebirth: after a
// tombstone, a fresh *Created for the same key re-materializes the row with
// the new payload — remove-then-insert composes through the inherited folds.
func TestSystem_EvolutionInheritance_RebirthAfterTombstone(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sys := startTombstoneSystem(t, ctx)

	streamID := id.NewStreamID()
	dispatchTomb(t, ctx, sys, "evotomb.create", streamID,
		EvoTombCreated{ID: streamID.String(), Title: "first life"})
	dispatchTomb(t, ctx, sys, "evotomb.delete", streamID, nil)

	deadline := loadScaledDeadline(5 * time.Second)

	for time.Now().Before(deadline) {
		if _, gone := awaitTombView(ctx, sys, streamID.String()); gone {
			break
		}

		time.Sleep(25 * time.Millisecond)
	}

	dispatchTomb(t, ctx, sys, "evotomb.create", streamID,
		EvoTombCreated{ID: streamID.String(), Title: "second life"})

	for time.Now().Before(deadline) {
		view, gone := awaitTombView(ctx, sys, streamID.String())
		if gone || view == nil {
			time.Sleep(25 * time.Millisecond)

			continue
		}

		if view.Title != "second life" {
			t.Fatalf("rebirth fold kept stale payload: title %q, want %q",
				view.Title, "second life")
		}

		return
	}

	t.Fatal("rebirth after tombstone never re-materialized the row")
}
