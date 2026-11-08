package main

// The entire developer-owned surface of this app: one event, one read model,
// one command, one query. Everything else — where data lives, which engines
// serve which role — is operator config filled by system.Standard.

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

const (
	streamType      = "Note"
	notesCollection = "notes"
	evtNoteCreated  = event.Type("note.created")
	cmdCreateNote   = command.Type("note.create")
	qryGetNote      = query.Type("note.get")
)

// NoteCreated is the single fact this domain mints.
type NoteCreated struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// NoteView is the read model: one Evolution folds NoteCreated into it.
type NoteView struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// CreateNoteCmd implements command.Command via the embedded basic command.
type CreateNoteCmd struct {
	*command.BasicCommand

	Text string
}

// GetNote asks for one note view by ID.
type GetNote struct {
	*query.BasicQuery

	ID string
}

// domain declares Events (the journal universe), the Evolution (the one fold
// declaration per read model), the Lookup projection, and the command and
// query registrations. Identical for every deployment.
func domain() system.DomainConfig {
	notes := system.Evolve[NoteView](notesCollection).
		On(string(evtNoteCreated), NoteCreated{})

	return system.DomainConfig{
		Events:     []event.Type{evtNoteCreated},
		Evolutions: []system.EvolutionSpec{notes.Done()},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[NoteView](notesCollection).Done(),
		},
		Commands: registerCommands,
		Queries:  registerQueries,
	}
}
