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

type publisherState int

const (
	publisherIdle publisherState = iota
	publisherRunning
	publisherClosed
)

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

type OutboxPublisherOption func(*OutboxPublisher)

func WithPollInterval(d time.Duration) OutboxPublisherOption {
	return func(p *OutboxPublisher) { p.interval = d }
}

func WithBatchSize(n int) OutboxPublisherOption {
	return func(p *OutboxPublisher) { p.batchSize = n }
}

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

	p := &outboxPublisher{ //nolint:exhaustruct // options fill remaining fields
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

func (p *OutboxPublisher) publishPending(ctx context.Context) {
	err := p.pollPublishAck(ctx)
	if err != nil {
		slog.Warn("outbox publish cycle failed", "error", err)
	}
}

func (p *OutboxPublisher) PublishNow(ctx context.Context) error {
	return p.pollPublishAck(ctx)
}
