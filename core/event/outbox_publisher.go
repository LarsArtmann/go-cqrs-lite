package event

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

const defaultBatchSize = 100

const defaultPollInterval = time.Second

// OutboxPublisher polls an Outbox for pending entries and publishes them
// to a Bus. Intended for reliable eventual publishing in event-sourced systems.
//
// Start the background loop with Start; stop it with Close (implements io.Closer).
type OutboxPublisher struct {
	outbox    Outbox
	bus       Bus
	interval  time.Duration
	batchSize int

	mu     sync.Mutex
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

// NewOutboxPublisher creates a publisher that polls outbox and publishes to bus.
// Returns an error if outbox or bus is nil.
func NewOutboxPublisher(
	outbox Outbox,
	bus Bus,
	opts ...OutboxPublisherOption,
) (*OutboxPublisher, error) {
	if outbox == nil {
		return nil, fmt.Errorf("%w", ErrNilOutbox)
	}

	if bus == nil {
		return nil, fmt.Errorf("%w", ErrNilBus)
	}

	p := &OutboxPublisher{
		outbox:    outbox,
		bus:       bus,
		interval:  defaultPollInterval,
		batchSize: defaultBatchSize,
		mu:        sync.Mutex{},
		cancel:    nil,
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
// Returns an error if already started.
func (p *OutboxPublisher) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		return ErrAlreadyStarted
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go p.run(ctx)

	return nil
}

// Close stops the background polling loop and waits for it to finish.
func (p *OutboxPublisher) Close() error {
	p.mu.Lock()

	if p.cancel == nil {
		p.mu.Unlock()

		return nil
	}

	p.cancel()
	p.mu.Unlock()

	<-p.done

	return nil
}

func (p *OutboxPublisher) run(ctx context.Context) {
	defer close(p.done)

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

// publishPending performs a single poll-publish-ack cycle.
// Errors are silently swallowed to keep the background loop running.
// For error visibility, use PublishNow which returns errors to the caller.
func (p *OutboxPublisher) publishPending(ctx context.Context) {
	entries, err := p.outbox.PollPending(ctx, p.batchSize)
	if err != nil {
		return
	}

	var ackIDs []OutboxID

	for _, entry := range entries {
		err = p.bus.Publish(ctx, entry.Events...)
		if err != nil {
			break
		}

		ackIDs = append(ackIDs, entry.ID)
	}

	if len(ackIDs) > 0 {
		_ = p.outbox.Ack(ctx, ackIDs)
	}
}

// PublishNow performs a single poll-publish-ack cycle immediately.
// Useful for testing or triggering from application code.
func (p *OutboxPublisher) PublishNow(ctx context.Context) error {
	entries, err := p.outbox.PollPending(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("poll pending: %w", err)
	}

	if len(entries) == 0 {
		return nil
	}

	var (
		ackIDs  []OutboxID
		lastErr error
	)

	for _, entry := range entries {
		err = p.bus.Publish(ctx, entry.Events...)
		if err != nil {
			lastErr = err

			break
		}

		ackIDs = append(ackIDs, entry.ID)
	}

	if len(ackIDs) > 0 {
		ackErr := p.outbox.Ack(ctx, ackIDs)
		if ackErr != nil && lastErr == nil {
			return fmt.Errorf("ack entries: %w", ackErr)
		}
	}

	if lastErr != nil {
		return fmt.Errorf("publish events: %w", lastErr)
	}

	return nil
}
