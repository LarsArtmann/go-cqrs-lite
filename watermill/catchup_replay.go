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

			select {
			case sub.output <- msg:
				// Checkpoint advances only on Ack: a crash or Nack between
				// handoff and processing replays this event on restart
				// (at-least-once).
				if !s.awaitAck(ctx, msg, sub.topic, "replay") {
					return errorfamily.NewOrchestration("watermill.catchup.replay_nacked",
						"consumer nacked replay event; stopping catch-up for "+sub.topic)
				}

				sub.replayWatermark = evt.ID().String()
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
	span.SetAttributes(cqrsotel.AttrString("cqrs.watermill.replay_watermark", sub.replayWatermark))

	s.logger.Info(
		"catch-up replay",
		"topic", sub.topic,
		"events", totalReplayed,
		"after", after.String(),
	)

	return nil
}
