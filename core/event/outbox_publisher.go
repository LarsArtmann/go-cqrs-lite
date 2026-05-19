package event

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

const defaultBatchSize = 100

const defaultPollInterval = time.Second

// publisherState represents the lifecycle state of an OutboxPublisher.
// Using an enum instead of multiple bools prevents split-brain conditions
// where state variables could drift out of sync.
type publisherState int

const (
	publisherIdle publisherState = iota
	publisherRunning
	publisherClosed
)

// OutboxPublisher polls an Outbox for pending entries and publishes them
// to a Bus. Intended for reliable eventual publishing in event-sourced systems.
//
// Start the background loop with Start; stop it with Close (implements io.Closer).
type OutboxPublisher struct {
	outbox    Outbox
	publisher Publisher
	interval  time.Duration
	batchSize int

	mu     sync.Mutex
	state  publisherState
	cancel context.CancelFunc
	done   chan struct{}
}

var _ io.Closer = (*OutboxPublisher)(nil)

// OutboxPublisherOption configures an OutboxPublisher.
type OutboxPublisherOption func(*OutboxPublisher)

// WithPollInterval sets the interval between outbox polls.
// Default is 1 second. Must be positive.
func WithPollInterval(d time.Duration) OutboxPublisherOption {
	return func(p *OutboxPublisher) { p.interval = d }
}

// WithBatchSize sets the maximum number of entries fetched per poll.
// Default is 100. Must be positive.
func WithBatchSize(n int) OutboxPublisherOption {
	return func(p *OutboxPublisher) { p.batchSize = n }
}

// NewOutboxPublisher creates a publisher that polls outbox and publishes via the Publisher.
// Returns an error if outbox or publisher is nil.
func NewOutboxPublisher(
	outbox Outbox,
	publisher Publisher,
	opts ...OutboxPublisherOption,
) (*OutboxPublisher, error) {
	if outbox == nil {
		return nil, fmt.Errorf("%w", ErrNilOutbox)
	}

	if publisher == nil {
		return nil, fmt.Errorf("%w", ErrNilBus)
	}

	p := &OutboxPublisher{ //nolint:exhaustruct // options fill remaining fields
		outbox:    outbox,
		publisher: publisher,
		interval:  defaultPollInterval,
		batchSize: defaultBatchSize,
		done:      make(chan struct{}),
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.interval <= 0 {
		p.interval = defaultPollInterval
	}

	if p.batchSize <= 0 {
		p.batchSize = defaultBatchSize
	}

	return p, nil
}

// Start begins the background polling loop.
// Returns ErrAlreadyStarted if already running, or ErrPublisherClosed if closed.
func (p *OutboxPublisher) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.state {
	case publisherClosed:
		return ErrPublisherClosed
	case publisherRunning:
		return ErrAlreadyStarted
	}

	p.state = publisherRunning
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go p.run(ctx)

	return nil
}

// Close stops the background polling loop and waits for it to finish.
// After Close, Start returns ErrPublisherClosed. Close is idempotent.
func (p *OutboxPublisher) Close() error {
	p.mu.Lock()

	if p.state == publisherClosed {
		p.mu.Unlock()

		return nil
	}

	p.state = publisherClosed

	if p.cancel == nil {
		p.mu.Unlock()

		return nil
	}

	cancel := p.cancel
	p.cancel = nil
	p.mu.Unlock()

	cancel()

	<-p.done

	return nil
}

func (p *OutboxPublisher) run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error(
				"outbox publisher recovered from panic",
				"error", r,
				"stack", string(debug.Stack()),
			)
		}

		close(p.done)
	}()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.publishPending(ctx)
		}
	}
}

// pollPublishAck performs a single poll-publish-ack cycle.
// Returns the poll error, publish error, and ack error (in that priority).
func (p *OutboxPublisher) pollPublishAck(ctx context.Context) error {
	entries, err := p.outbox.PollPending(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("poll pending: %w", err)
	}

	if len(entries) == 0 {
		return nil
	}

	var ackIDs []OutboxID

	for _, entry := range entries {
		err = p.publisher.Publish(ctx, entry.Events...)
		if err != nil {
			break
		}

		ackIDs = append(ackIDs, entry.ID)
	}

	publishErr := err

	if len(ackIDs) > 0 {
		ackErr := p.outbox.Ack(ctx, ackIDs)
		if ackErr != nil && publishErr == nil {
			return fmt.Errorf("ack entries: %w", ackErr)
		}
	}

	if publishErr != nil {
		return fmt.Errorf("publish events: %w", publishErr)
	}

	return nil
}

// publishPending performs a single poll-publish-ack cycle.
// Errors are silently swallowed to keep the background loop running.
// For error visibility, use PublishNow which returns errors to the caller.
func (p *OutboxPublisher) publishPending(ctx context.Context) {
	_ = p.pollPublishAck(ctx)
}

// PublishNow performs a single poll-publish-ack cycle immediately.
// Useful for testing or triggering from application code.
func (p *OutboxPublisher) PublishNow(ctx context.Context) error {
	return p.pollPublishAck(ctx)
}
