package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"

	// Registers the "sqlite" metaengine driver (database/sql pattern).
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"

	// Registers the "postgres" metaengine driver — the VM test runs this
	// binary against a NixOS-provisioned postgres over peer auth.
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
)

func main() {
	configPath := flag.String("config", os.Getenv("CQRS_DEPLOYMENT_CONFIG"),
		"deployment config (.yaml/.json); empty = built-in sqlite default")
	flag.Parse()

	if err := run(context.Background(), *configPath); err != nil {
		log.Fatal(err)
	}
}

// run is the whole operator surface: load (or default) the deployment, fill
// the standard topology, compose, dispatch one command, read it back, and
// show where the planner routed every collection.
func run(ctx context.Context, configPath string) error {
	deployment, err := loadDeployment(configPath)
	if err != nil {
		return fmt.Errorf("load deployment: %w", err)
	}

	sys, err := system.New(
		ctx,
		domain(),
		system.Standard(deployment),
	) // art-dupl:accept canonical system boot, deliberately mirrors example/goal-shaped-app
	if err != nil {
		return fmt.Errorf("compose system: %w", err)
	}
	defer metaengine.DeferClose(sys)

	view, err := runStory(ctx, sys)
	if err != nil {
		return err
	}

	fmt.Printf("\nNoteView: %+v\n\n", view)
	fmt.Println(sys.MetaEngine().ExplainPlan())

	return nil
}

// loadDeployment prefers an explicit config file (the NixOS module exports
// CQRS_DEPLOYMENT_CONFIG pointing at its rendered deployment.json); the
// built-in default declares engines ONLY — system.Standard fills the
// three-instance topology (ADR-0165).
func loadDeployment(configPath string) (system.DeploymentConfig, error) {
	if configPath != "" {
		return system.LoadConfig(configPath)
	}

	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "sqlite", DSN: "file:standard-topology.db"},
		},
	}, nil
}

// runStory starts the system, creates one note, and polls the Lookup
// projection until the read model folded it.
func runStory(ctx context.Context, sys *system.System) (NoteView, error) {
	if err := sys.Start(ctx); err != nil {
		return NoteView{}, fmt.Errorf("start system: %w", err)
	}

	basic, err := command.New(cmdCreateNote, id.NewStreamID())
	if err != nil {
		return NoteView{}, fmt.Errorf("command.New: %w", err)
	}

	cmd := CreateNoteCmd{BasicCommand: basic, Text: "standard topology works"}
	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		return NoteView{}, fmt.Errorf("dispatch note.create: %w", err)
	}

	return awaitNote(ctx, sys, cmd.StreamID().Get())
}

// awaitNote polls note.get until the projection host folded the event — any
// lookup miss just means the fold has not landed yet.
func awaitNote(ctx context.Context, sys *system.System, noteID string) (NoteView, error) {
	basic, err := query.New(qryGetNote)
	if err != nil {
		return NoteView{}, fmt.Errorf("query.New: %w", err)
	}

	q := GetNote{BasicQuery: basic, ID: noteID}

	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if view, qerr := system.DispatchQuery[GetNote, NoteView](ctx, sys, q); qerr == nil {
			return view, nil
		}
	}

	return NoteView{}, fmt.Errorf("note %s never appeared in the read model", noteID)
}
