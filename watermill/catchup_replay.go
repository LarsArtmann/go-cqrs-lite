package watermill

import (
	"context"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

func (s *CatchUpSubscriber) replayPhase(ctx context.Context, sub *catchUpSubscription) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "watermill.replay.from_journal",
		cqrsotel.SpanKindInternal,
		cqrsotel.WithAttributes(cqrsotel.AttrString("cqrs.projection.name", sub.topic)),
	)
	defer span.End()

	checkpoint, err := s.checkpoint.Load(ctx, sub.topic)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapInfrastructure(err, "watermill.catchup.load_checkpoint",
			fmt.Sprintf("load checkpoint for %s", sub.topic))
	}

	var after id.EventID

	if !checkpoint.IsZero() {
		after = checkpoint.EventID
	}

	// Replay in fixed-size batches so memory stays bounded regardless of
	// journal size (same pattern as transport/http.SSEBroker). Each batch is
	// fetched, forwarded, and checkpointed before the next is loaded.
	const batchSize = 500

	cursor := after
	totalReplayed := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.closeCh:
			return nil
		default:
		}

		events, err := s.journal.ReadFrom(ctx, cursor, batchSize)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return errorfamily.WrapInfrastructure(err, "watermill.catchup.replay_read",
				"replay read from journal")
		}

		if len(events) == 0 {
			break
		}

		for _, evt := range events {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-s.closeCh:
				return nil
			default:
			}

			msg := eventToMessage(evt)
			// Mark ModeReplay in message metadata. Consumers reconstruct it into
			// the handler context via ProcessingModeMiddleware (the metadata is
			// the only channel that survives process boundaries in Watermill).
			msg.Metadata.Set(metaProcessingMode, string(event.ModeReplay))

			sub.replayIDs.Add(evt.ID().String())

			select {
			case sub.output <- msg:
				// Save checkpoint after forwarding each replay event.
				// Best-effort: log on error, don't block the stream.
				if saveErr := s.saveCheckpoint(ctx, sub.topic, evt.ID()); saveErr != nil {
					s.logger.Warn("catch-up: save checkpoint after replay event",
						"topic", sub.topic, "event_id", evt.ID().String(), "error", saveErr)
				}
			case <-ctx.Done():
				return ctx.Err()
			case <-s.closeCh:
				return nil
			}
		}

		totalReplayed += len(events)

		if len(events) < batchSize {
			break // journal drained
		}

		cursor = events[len(events)-1].ID()
	}

	span.SetAttributes(cqrsotel.AttrInt(cqrsotel.AttrEventCount, totalReplayed))
	span.SetAttributes(cqrsotel.AttrInt("cqrs.watermill.dedup_ring_size", sub.replayIDs.Len()))

	s.logger.Info(
		"catch-up replay",
		"topic", sub.topic,
		"events", totalReplayed,
		"after", after.String(),
	)

	return nil
}
