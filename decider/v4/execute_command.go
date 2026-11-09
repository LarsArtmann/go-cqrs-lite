package decider

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// CausedCommand is the minimal capability [ExecuteCommandRef] needs:
// anything with a minted command identity can stamp causation onto the
// events its decision emits. *command.BasicCommand,
// *command.PersistedCommand, and consumer-defined command structs satisfy it
// structurally — decider does NOT import command/ (ADR-0111 g:
// capability interfaces over core-interface growth).
type CausedCommand interface {
	ID() id.CommandID
}

// CommandDecideFunc is [DecideFunc] that can see the command driving the
// decision. Use it when the decision needs the command's payload or identity
// beyond what the captured closure would provide.
type CommandDecideFunc[State, C any] func(
	state State,
	currentVersion event.Version,
	cmd C,
) ([]event.Event, error)

// commandTyper is the optional capability that enriches the stamped
// Causation with the command's type. *command.BasicCommand and
// *command.PersistedCommand satisfy it (command.Type = record.Type is an
// alias); commands without a Type method still get the ID stamped.
type commandTyper interface {
	Type() record.Type
}

// ExecuteCommandRef is the command-aware form of [Repository.ExecuteRef]: it
// loads the stream, folds it into state, calls decide WITH the command, and
// stamps every emitted event with the command's causation before persisting
// and publishing.
//
// It is a package-level function rather than a Repository method because
// generic methods (a type parameter per method) require Go 1.27 — the same
// reason the repository options are package-level generic functions.
//
// Stamping (per event, skipped when the event already carries a typed
// Causation — a decision that sets its own causation keeps it):
//
//   - Metadata.Causation = {CommandType, CommandID} — the typed form
//     [event.AsRecord] resolves into Record.CausationID and
//     Cause{Kind: CauseCommand}, so downstream records know both the causer's
//     identity AND its kind.
//   - Custom "command.id" (and "command.type" when the command exposes a
//     Type method) — the v2 backward-compat keys.
//
// The command ID must be non-zero for stamping; a zero-ID command executes
// unstamped (same behavior as ExecuteRef).
//
// Everything else matches [Repository.ExecuteRef]: OTel span, flight-recorder
// capture, enricher application, snapshot, cache, and publish semantics. A
// configured [WithEnricher] runs AFTER the causation stamp and may overwrite
// it — prefer [event.WithCommandCausality] on the context with the same
// command identity, or drop the enricher for streams executed through this
// function.
func ExecuteCommandRef[State any, C CausedCommand](
	ctx context.Context,
	repo *Repository[State],
	ref id.StreamRef,
	cmd C,
	decide CommandDecideFunc[State, C],
) error {
	cmdID := cmd.ID()
	cmdType := commandTypeOf(cmd)

	return repo.ExecuteRef(ctx, ref, func(
		state State,
		currentVersion event.Version,
	) ([]event.Event, error) {
		newEvents, err := decide(state, currentVersion, cmd)
		if err != nil {
			return nil, err
		}

		stampCommandCausation(newEvents, cmdType, cmdID)

		return newEvents, nil
	})
}

// commandTypeOf returns the command's type when it exposes the optional
// capability, "" otherwise.
func commandTypeOf[C CausedCommand](cmd C) string {
	if typer, ok := any(cmd).(commandTyper); ok {
		return string(typer.Type())
	}

	return ""
}

// stampCommandCausation applies the causation options onto events that do
// not already carry a typed Causation. Zero-ID commands stamp nothing: a
// causation without a causer identity is noise, and the persisted result
// must be identical to a plain ExecuteRef run.
func stampCommandCausation(events []event.Event, cmdType string, cmdID id.CommandID) {
	if cmdID.IsZero() || len(events) == 0 {
		return
	}

	for _, evt := range events {
		if evt.Metadata().Causation != nil {
			continue
		}

		for _, opt := range causationOptions(cmdType, cmdID) {
			opt(evt)
		}
	}
}

// causationOptions builds the option set that mirrors
// [event.CommandCausalityEnricher]: the typed Causation plus the v2
// backward-compat custom keys. The command.type key is only added when a
// type is known — an empty custom entry is noise, not information.
func causationOptions(cmdType string, cmdID id.CommandID) []event.Option {
	const maxCausationOpts = 3

	opts := make([]event.Option, 0, maxCausationOpts)
	opts = append(opts,
		event.WithCausation(cmdType, cmdID),
		event.WithCustom(event.MetadataKeyCommandID, cmdID.String()),
	)

	if cmdType != "" {
		opts = append(opts, event.WithCustom(event.MetadataKeyCommandType, cmdType))
	}

	return opts
}
