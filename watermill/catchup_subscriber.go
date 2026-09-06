package watermill

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
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
// The subscriber maintains a checkpoint per topic (projection name). The
// checkpoint advances only after a message is ACKED by the consumer — never
// on handoff — so a crash or Nack between delivery and processing replays the
// event on restart (at-least-once). On restart, replay resumes from the last
// checkpoint.
//
// Delivery is serialized per subscription: each message is forwarded, then
// the subscriber waits for its Ack (or Nack) before delivering the next one.
// A Nack stops the stream for that subscription (the checkpoint stays behind
// the nacked event); the projection restarts from the checkpoint.
//
// Phase 1 (replay): Events are loaded from the journal via ReadFrom, converted
// to Watermill messages, and sent to the output channel with ProcessingMode =
// ModeReplay in the message metadata.
//
// Phase 2 (live handoff): The live subscriber is started BEFORE replay (to
// close the TOCTOU gap). Live events whose ID is at or below the last
// replayed event ID (a monotonic-ULID watermark) were already covered by the
// replay and are suppressed; all other live events are forwarded.
//
// # Watermark ordering assumption
//
// The watermark compares event-ID strings and assumes ULIDs are monotonic
// across the WHOLE journal — including events minted in other processes.
// Within one process ULIDs are monotonic; across processes, clock skew can
// mint an event whose ID sorts below the watermark even though it was
// published after replay drained the journal. Such an event is suppressed
// live (it looks "already covered") and stays missing from this subscription
// until the next restart, where replay resumes from the checkpoint — which
// sits behind the suppressed event — and delivers it. The staleness window
// is therefore bounded by producer clock skew (milliseconds under
// NTP-disciplined clocks) and is self-healing on restart. Deployments with
// large, unbounded clock skew should not rely on the live-suppression
// shortcut: restarting the projection re-syncs it deterministically.
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
	topic  string
	output chan *message.Message
	cancel context.CancelFunc
	// replayWatermark is the event ID of the LAST event forwarded during
	// replay. Monotonic ULIDs make string comparison equivalent to time
	// order, so any live event at or below the watermark was already covered
	// by the replay and is suppressed — unlike a bounded ring, the watermark
	// covers arbitrarily long replays.
	replayWatermark string
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
		return nil, errorfamily.NewRejection("watermill.create_catchup_subscriber",
			"journal must not be nil")
	}

	if live == nil {
		return nil, errorfamily.NewRejection("watermill.create_catchup_subscriber",
			"live subscriber must not be nil")
	}

	if checkpoint == nil {
		return nil, errorfamily.NewRejection("watermill.create_catchup_subscriber",
			"checkpoint store must not be nil")
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
func (s *CatchUpSubscriber) Subscribe(
	ctx context.Context,
	topic string,
) (<-chan *message.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, errorfamily.NewInfrastructure("watermill.catchup_subscriber_closed",
			"catch-up subscriber is closed")
	}

	output := make(chan *message.Message, 256)

	subCtx, cancel := context.WithCancel(ctx)

	sub := &catchUpSubscription{
		topic:  topic,
		output: output,
		cancel: cancel,
	}

	s.subs = append(s.subs, sub)

	go s.runCatchUp(subCtx, sub)

	return output, nil
}

// runCatchUp orchestrates the catch-up for one subscription: subscribe
// live first (to close the TOCTOU race window), then replay historical events
// from the journal, then drain live messages (deduplicating against IDs seen
// during replay).
func (s *CatchUpSubscriber) runCatchUp(ctx context.Context, sub *catchUpSubscription) {
	defer close(sub.output)

	// Subscribe to live BEFORE replay. Events published during replay are
	// buffered by the live subscriber and deduplicated against replayIDs
	// after replay completes. Without this ordering, events published in the
	// gap between replay draining and live subscribing would be lost.
	//cqrs-lint:ignore(C027) library code or intentional pattern
	liveMsgs, err := s.live.Subscribe(ctx, sub.topic)
	if err != nil {
		s.logger.Error("catch-up: subscribe live failed",
			"topic", sub.topic, "error", err)

		return
	}

	// Phase 1: Replay from journal.
	if err := s.replayPhase(ctx, sub); err != nil {
		s.logger.Error("catch-up replay failed", "topic", sub.topic, "error", err)

		return
	}

	// Phase 2: Drain live messages, dedup against replay.
	s.drainLive(ctx, sub, liveMsgs)
}

// drainLive forwards messages from the live subscriber channel to the
// output channel, suppressing events at or below the replay watermark (they
// were already delivered during replay) and advancing the checkpoint only
// after the consumer Acks.
func (s *CatchUpSubscriber) drainLive(
	ctx context.Context,
	sub *catchUpSubscription,
	liveMsgs <-chan *message.Message,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.closeCh:
			return
		case msg, ok := <-liveMsgs:
			if !ok {
				return
			}

			// Suppress live events already covered by the replay: their ID is
			// at or below the last replayed ID (monotonic ULID ordering).
			eventID := msg.Metadata.Get(metaEventID)
			if eventID != "" && sub.replayWatermark != "" && eventID <= sub.replayWatermark {
				msg.Ack()

				continue
			}

			select {
			case sub.output <- msg:
				if !s.awaitAck(ctx, msg, sub.topic, "live") {
					return
				}
			case <-ctx.Done():
				return
			case <-s.closeCh:
				return
			}
		}
	}
}

// awaitAck blocks until the consumer Acks or Nacks msg. On Ack it advances
// the checkpoint (best-effort); on Nack it reports false so the caller stops
// the stream — the checkpoint stays behind the nacked event, so a restart
// re-delivers it (at-least-once).
func (s *CatchUpSubscriber) awaitAck(
	ctx context.Context,
	msg *message.Message,
	topic, phase string,
) bool {
	select {
	case <-msg.Acked():
		eventID := msg.Metadata.Get(metaEventID)
		if eventID != "" {
			if evtID, parseErr := id.ParseEventID(eventID); parseErr == nil {
				if saveErr := s.saveCheckpoint(ctx, topic, evtID); saveErr != nil {
					s.logger.Warn("catch-up: save checkpoint after ack",
						"topic", topic, "phase", phase, "event_id", eventID, "error", saveErr)
				}
			}
		}

		return true
	case <-msg.Nacked():
		s.logger.Warn("catch-up: consumer nacked message; stopping subscription",
			"topic", topic, "phase", phase,
			"event_id", msg.Metadata.Get(metaEventID))

		return false
	case <-ctx.Done():
		return false
	case <-s.closeCh:
		return false
	}
}

// saveCheckpoint persists the last-processed event ID for the given topic.
// Best-effort: errors are logged by callers, not returned to the stream.
func (s *CatchUpSubscriber) saveCheckpoint(
	ctx context.Context,
	topic string,
	eventID id.EventID,
) error {
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

		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = s.live.Close()
	})

	return nil
}

const metaProcessingMode = "processing_mode"

var _ message.Subscriber = (*CatchUpSubscriber)(nil)
