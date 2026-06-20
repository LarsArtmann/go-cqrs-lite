package projection

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

func (r *Runner) replay(ctx context.Context) error {
	seekable, hasSeekable := r.journal.(event.SeekableJournal)

	for _, entry := range r.projections {
		ctx, span := cqrsotel.StartSpan(
			ctx, tracer(), "projection.replay",
			cqrsotel.SpanKindClient,
			cqrsotel.WithAttributes(projectionAttrs(entry.projection.Name())...),
		)

		events, err := r.loadReplayEvents(
			ctx,
			seekable,
			hasSeekable,
			entry.projection,
			entry.eventTypes,
		)
		if err != nil {
			cqrsotel.RecordError(span, err)
			span.End()

			return err
		}

		span.SetAttributes(cqrsotel.AttrInt(cqrsotel.AttrEventCount, len(events)))

		for _, evt := range events {
			r.replayIDs[evt.ID()] = struct{}{}

			replayCtx := event.WithProcessingMode(ctx, event.ModeReplay)

			hErr := r.handleAndCheckpoint(replayCtx, entry.projection, evt)
			if hErr != nil {
				cqrsotel.RecordError(span, hErr)
				span.End()

				return event.WrapCorruption(hErr, "projection.replay_event",
					"replay "+entry.projection.Name()+" event "+evt.ID().String())
			}
		}

		span.End()
	}

	return nil
}

func (r *Runner) loadReplayEvents(
	ctx context.Context,
	seekable event.SeekableJournal,
	hasSeekable bool,
	p event.Projection,
	eventTypes []event.Type,
) ([]event.Event, error) {
	checkpoint, cpErr := r.checkpoint.Load(ctx, p.Name())
	if cpErr != nil {
		return nil, event.WrapInfrastructure(cpErr, "projection.load_checkpoint",
			"load checkpoint for "+p.Name())
	}

	if hasSeekable && !checkpoint.IsZero() {
		loaded, lErr := seekable.ReadFrom(ctx, checkpoint.EventID, 0)
		if lErr != nil {
			return nil, event.WrapInfrastructure(lErr, "projection.load_events",
				"load events from position for "+p.Name())
		}

		return filterByEventTypes(loaded, eventTypes), nil
	}

	allEvents, lErr := r.journal.ReadAll(ctx)
	if lErr != nil {
		return nil, event.WrapInfrastructure(lErr, "projection.load_events",
			"load all events")
	}

	return filterFromCheckpoint(allEvents, eventTypes, checkpoint), nil
}

func (r *Runner) handleAndCheckpoint(
	ctx context.Context,
	p event.Projection,
	evt event.Event,
) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "projection.handle",
		cqrsotel.SpanKindConsumer,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString(cqrsotel.AttrEventType, string(evt.Type())),
			cqrsotel.AttrString(cqrsotel.AttrProjectionName, p.Name()),
		),
	)
	defer span.End()

	err := p.Handle(ctx, evt)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.Wrap(err, event.Classify(err), "projection.handle_event",
			"projection "+p.Name()+" handle event "+string(evt.Type()))
	}

	cqrsotel.AddSpanEvent(
		span, "checkpoint_saved",
		cqrsotel.AttrString("projection", p.Name()),
		cqrsotel.AttrString("event_id", evt.ID().String()),
	)

	return r.checkpoint.Save(
		ctx,
		p.Name(),
		event.Checkpoint{EventID: evt.ID(), ProcessedAt: time.Now()},
	)
}
