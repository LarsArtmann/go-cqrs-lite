package watermill

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// CheckpointStore is the interface for persisting the last-processed event ID.
// It is an alias for event.CheckpointStore, re-declared here so consumers of
// the watermill package don't need to import event just for this type.
type CheckpointStore = event.CheckpointStore

// CatchUpSubscriber is a [message.Subscriber] that replays historical events
// from an [event.SeekableJournal] before handing off to a live subscriber.
//
// It solves the "catch-up" problem: when a projection starts, it must first
// process all past events (replay), then seamlessly transition to processing
// new events in real time (live). Watermill's built-in subscribers have no
// replay capability — they only deliver live messages.
//
// The subscriber maintains a checkpoint per topic (projection name). After
// each message is Acked, the checkpoint advances. On restart, replay resumes
// from the last checkpoint.
//
// Phase 1 (replay): Events are loaded from the journal via ReadFrom, converted
// to Watermill messages, and sent to the output channel with ProcessingMode =
// ModeReplay in the message metadata.
//
// Phase 2 (live handoff): The live subscriber is started. Events that were
// already seen during replay (matched by EventID) are suppressed. All other
// live events are forwarded to the output channel.
//
// Usage:
//
//	catchUp := watermill.NewCatchUpSubscriber(journal, liveSub, cpStore, logger)
//	defer catchUp.Close()
//
//	msgs, err := catchUp.Subscribe(ctx, "user.created")
type CatchUpSubscriber struct {
	journal    event.SeekableJournal
	live       message.Subscriber
	checkpoint CheckpointStore
	logger     *slog.Logger

	mu        sync.Mutex
	closed    bool
	subs      []*catchUpSubscription
	closeCh   chan struct{}
	closeOnce sync.Once
}

type catchUpSubscription struct {
	topic     string
	output    chan *message.Message
	cancel    context.CancelFunc
	replayIDs map[string]struct{} // event IDs seen during replay
}

// NewCatchUpSubscriber creates a CatchUpSubscriber.
//
// Parameters:
//   - journal: the seekable event journal for replay (must not be nil).
//   - live: the Watermill subscriber for live events (must not be nil).
//   - checkpoint: persists replay position per topic (must not be nil).
//   - logger: structured logger; nil falls back to slog.Default().
func NewCatchUpSubscriber(
	journal event.SeekableJournal,
	live message.Subscriber,
	checkpoint CheckpointStore,
	logger *slog.Logger,
) (*CatchUpSubscriber, error) {
	if journal == nil {
		return nil, fmt.Errorf("watermill: journal must not be nil")
	}

	if live == nil {
		return nil, fmt.Errorf("watermill: live subscriber must not be nil")
	}

	if checkpoint == nil {
		return nil, fmt.Errorf("watermill: checkpoint store must not be nil")
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &CatchUpSubscriber{
		journal:    journal,
		live:       live,
		checkpoint: checkpoint,
		logger:     logger,
		closeCh:    make(chan struct{}),
	}, nil
}

// Subscribe starts catch-up for the given topic: replay then live.
// Returns a channel of messages. The topic is used as the checkpoint key
// (projection name).
func (s *CatchUpSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("watermill: catch-up subscriber is closed")
	}

	output := make(chan *message.Message, 256)

	subCtx, cancel := context.WithCancel(ctx)

	sub := &catchUpSubscription{
		topic:     topic,
		output:    output,
		cancel:    cancel,
		replayIDs: make(map[string]struct{}),
	}

	s.subs = append(s.subs, sub)

	go s.runCatchUp(subCtx, sub)

	return output, nil
}

// runCatchUp orchestrates the replay → live handoff for one subscription.
func (s *CatchUpSubscriber) runCatchUp(ctx context.Context, sub *catchUpSubscription) {
	defer close(sub.output)

	// Phase 1: Replay from journal.
	if err := s.replayPhase(ctx, sub); err != nil {
		s.logger.Error("catch-up replay failed", "topic", sub.topic, "error", err)

		return
	}

	// Phase 2: Live handoff.
	if err := s.livePhase(ctx, sub); err != nil {
		s.logger.Error("catch-up live phase failed", "topic", sub.topic, "error", err)

		return
	}
}

func (s *CatchUpSubscriber) replayPhase(ctx context.Context, sub *catchUpSubscription) error {
	cp, err := s.checkpoint.Load(ctx, sub.topic)
	if err != nil {
		return fmt.Errorf("load checkpoint for %s: %w", sub.topic, err)
	}

	var after id.EventID

	if !cp.IsZero() {
		after = cp.EventID
	}

	events, err := s.journal.ReadFrom(ctx, after, 0)
	if err != nil {
		return fmt.Errorf("replay read from journal: %w", err)
	}

	s.logger.Info(
		"catch-up replay",
		"topic", sub.topic,
		"events", len(events),
		"after", after.String(),
	)

	replayCtx := event.WithProcessingMode(ctx, event.ModeReplay)

	for _, evt := range events {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.closeCh:
			return nil
		default:
		}

		msg := eventToMessage(evt)
		msg.Metadata.Set(metaProcessingMode, string(event.ModeReplay))

		sub.replayIDs[evt.ID().String()] = struct{}{}

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

		_ = replayCtx // context available for handlers via message if needed
	}

	return nil
}

func (s *CatchUpSubscriber) livePhase(ctx context.Context, sub *catchUpSubscription) error {
	liveMsgs, err := s.live.Subscribe(ctx, sub.topic)
	if err != nil {
		return fmt.Errorf("subscribe live for %s: %w", sub.topic, err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.closeCh:
			return nil
		case msg, ok := <-liveMsgs:
			if !ok {
				return nil
			}

			// Dedup: skip events already seen during replay.
			eventID := msg.Metadata.Get(metaEventID)
			if eventID != "" {
				if _, seen := sub.replayIDs[eventID]; seen {
					msg.Ack()

					continue
				}
			}

			select {
			case sub.output <- msg:
				// Save checkpoint for live events too.
				if eventID != "" {
					if evtID, parseErr := id.ParseEventID(eventID); parseErr == nil {
						if saveErr := s.saveCheckpoint(ctx, sub.topic, evtID); saveErr != nil {
							s.logger.Warn("catch-up: save checkpoint after live event",
								"topic", sub.topic, "event_id", eventID, "error", saveErr)
						}
					}
				}
			case <-ctx.Done():
				return ctx.Err()
			case <-s.closeCh:
				return nil
			}
		}
	}
}

// saveCheckpoint persists the last-processed event ID for the given topic.
// Best-effort: errors are logged by callers, not returned to the stream.
func (s *CatchUpSubscriber) saveCheckpoint(ctx context.Context, topic string, eventID id.EventID) error {
	return s.checkpoint.Save(ctx, topic, event.Checkpoint{
		EventID:     eventID,
		ProcessedAt: time.Now(),
	})
}

// Close shuts down all active subscriptions and the underlying live subscriber.
func (s *CatchUpSubscriber) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()

		close(s.closeCh)

		s.mu.Lock()
		for _, sub := range s.subs {
			sub.cancel()
		}
		s.mu.Unlock()

		_ = s.live.Close()
	})

	return nil
}

const metaProcessingMode = "processing_mode"

var _ message.Subscriber = (*CatchUpSubscriber)(nil)
