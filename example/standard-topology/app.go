package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

var errNoteExists = errors.New("note already exists")

// registerCommands wires the decider and the single command.
func registerCommands(sys *system.System) {
	if err := system.RegisterDecider(sys, streamType, decider.Decider[struct{}]{
		Initial: struct{}{},
		Apply: func(_ struct{}, _ event.Event) (struct{}, error) {
			return struct{}{}, nil
		},
	}); err != nil {
		panic(err)
	}

	if err := system.RegisterCommand[CreateNoteCmd, struct{}](sys, cmdCreateNote, createOp); err != nil {
		panic(err)
	}
}

func createOp(ctx context.Context, cmd CreateNoteCmd) system.Op[struct{}] {
	return system.Execute(ctx, cmd.StreamID(), streamType,
		func(_ struct{}, v event.Version) ([]event.Event, error) {
			return decideCreate(cmd, v)
		})
}

func decideCreate(cmd CreateNoteCmd, v event.Version) ([]event.Event, error) {
	evt, err := event.New(evtNoteCreated, cmd.StreamID(), streamType, v.Increment(),
		NoteCreated{ID: cmd.StreamID().Get(), Text: cmd.Text})
	if err != nil {
		return nil, fmt.Errorf("note.created: %w", err)
	}

	return []event.Event{evt}, nil
}

// registerQueries serves the Lookup read model through the typed query bus.
func registerQueries(sys *system.System) {
	byID := metaengine.NewReader[NoteView](sys.MetaEngine(), notesCollection)

	if err := system.RegisterQuery[GetNote, NoteView](sys, string(qryGetNote),
		func(ctx context.Context, q GetNote) (NoteView, error) {
			view, found, err := byID.Get(ctx, q.ID)
			if err != nil {
				return NoteView{}, err
			}

			if !found {
				return NoteView{}, fmt.Errorf("note %s not found", q.ID)
			}

			return view, nil
		}); err != nil {
		panic(err)
	}
}
