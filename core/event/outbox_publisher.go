package event

import (
	"context"
	"io"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

const defaultBatchSize = 100

const defaultPollInterval = time.Second

type publisherState int

const (
	publisherIdle publisherState = iota
	publisherRunning
	publisherClosed
)

// OutboxPublisher polls an outbox for pending events and publishes them via a Publisher.
// Start begins background polling; Close stops it gracefully.
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

// WithPollInterval sets the interval between outbox polls. Defaults to 1s.
func WithPollInterval(d time.Duration) OutboxPublisherOption {
	return func(p *OutboxPublisher) { p.interval = d }
}

// WithBatchSize sets the number of entries to poll per batch. Defaults to 100.
func WithBatchSize(n int) OutboxPublisherOption {
	return func(p *OutboxPublisher) { p.batchSize = n }
}

// NewOutboxPublisher creates a polling outbox publisher.
// Returns an error if outbox or publisher is nil.
func NewOutboxPublisher(
	outbox Outbox,
	publisher Publisher,
	opts ...OutboxPublisherOption,
) (*OutboxPublisher, error) {
	if outbox == nil {
		return nil, ErrNilOutbox
	}

	if publisher == nil {
		return nil, ErrNilBus
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

// Start begins background polling of the outbox. Returns ErrPublisherClosed or
// ErrAlreadyStarted if called in an invalid state.
func (p *OutboxPublisher) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.state {
	case publisherClosed:
		return ErrPublisherClosed
	case publisherRunning:
		return ErrAlreadyStarted
	case publisherIdle:
	}

	p.state = publisherRunning
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go p.run(ctx)

	return nil
}

// Close stops the background poller and waits for it to finish.
// Safe to call multiple times.
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

func (p *OutboxPublisher) pollPublishAck(ctx context.Context) error {
	entries, err := p.outbox.PollPending(ctx, p.batchSize)
	if err != nil {
		return WrapInfrastructure(
			err,
			"event.outbox_poll_failed",
			"poll pending outbox entries",
		)
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
			return WrapInfrastructure(
				ackErr,
				"event.outbox_ack_failed",
				"ack outbox entries",
			)
		}
	}

	if publishErr != nil {
		return WrapInfrastructure(
			publishErr,
			"event.publish_failed",
			"publish events from outbox",
		)
	}

	return nil
}

func (p *OutboxPublisher) publishPending(ctx context.Context) {
	err := p.pollPublishAck(ctx)
	if err != nil {
		slog.Warn("outbox publish cycle failed", "error", err)
	}
}

func (p *OutboxPublisher) PublishNow(ctx context.Context) error {
	return p.pollPublishAck(ctx)
}
