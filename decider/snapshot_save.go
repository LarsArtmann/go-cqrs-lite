package decider

import (
	"context"
	"log/slog"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// saveSnapshotAfterEvents folds new events onto state to get the final state,
// then attempts to save a snapshot. Errors are recorded on the active span and
// swallowed — snapshots are best-effort and must not block the write path.
func (r *Repository[State]) saveSnapshotAfterEvents(
	ctx context.Context,
	ref id.StreamRef,
	newVersion event.Version,
	state State,
	newEvents []event.Event,
) {
	if !r.shouldSnapshot(ref, newVersion) {
		return
	}

	finalState := state

	for _, evt := range newEvents {
		var foldErr error

		finalState, foldErr = r.decider.Apply(finalState, evt)
		if foldErr != nil {
			err := opError(ref, "apply event %s for snapshot: %w", evt.Type(), foldErr)
			cqrsotel.RecordError(cqrsotel.SpanFromContext(ctx), err)
			slog.WarnContext(
				ctx,
				"snapshot apply failed",
				"ref",
				ref,
				"event_type",
				evt.Type(),
				"error",
				foldErr,
			)

			return
		}
	}

	encoded, encErr := r.codec.Encode(finalState)
	if encErr != nil {
		err := opError(ref, "encode snapshot: %w", encErr)
		cqrsotel.RecordError(cqrsotel.SpanFromContext(ctx), err)
		slog.WarnContext(ctx, "snapshot encode failed", "ref", ref, "error", encErr)

		return
	}

	snap, snapErr := snapshot.NewSnapshot(ref, newVersion, encoded, snapshotEncoding(r.codec))
	if snapErr != nil {
		cqrsotel.RecordError(cqrsotel.SpanFromContext(ctx), snapErr)
		slog.WarnContext(ctx, "snapshot construct failed", "ref", ref, "error", snapErr)

		return
	}

	snap.StateShape = r.snapshotStateShape

	saveErr := r.snapshotStore.Save(ctx, snap)
	if saveErr != nil {
		cqrsotel.RecordError(cqrsotel.SpanFromContext(ctx), saveErr)
		slog.WarnContext(
			ctx,
			"snapshot save failed",
			"ref",
			ref,
			"version",
			newVersion,
			"error",
			saveErr,
		)
	}
}

// updateCacheAfterExecute folds the new events onto the pre-command state
// and stores the result in the hot-state cache. On fold error the cache
// entry is invalidated to force a full reload on next access.
func (r *Repository[State]) updateCacheAfterExecute(
	ref id.StreamRef,
	state State,
	newVersion event.Version,
	newEvents []event.Event,
) {
	if len(newEvents) == 0 {
		return
	}

	finalState, err := r.foldEvents(state, newEvents, ref)
	if err != nil {
		r.stateCache.Invalidate(ref)

		return
	}

	r.stateCache.Put(ref, finalState, newVersion)
}

// snapshotEncoding maps the repository codec's encoding name onto the
// compact typed stamp carried on saved snapshots. Unknown codec names stamp
// record.EncodingUnknown rather than guessing.
//
// art-dupl:accept codec->record conversion mirror of snapshot.recordEncodingFrom (dep-isolated, ADR-0111)
func snapshotEncoding(c codec.Codec) record.Encoding {
	parsed, err := record.ParseEncoding(string(c.Encoding()))
	if err != nil {
		return record.EncodingUnknown
	}

	return parsed
}
