package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// OutboxPoller is a background worker that polls the outbox and dispatches
// events to a publisher. It runs on a configurable interval and processes batches
// of pending outbox entries.
//
// Usage:
//
//	poller := storage.NewOutboxPoller(outbox, bus, storage.WithPollInterval(5*time.Second))
//	go poller.Start(ctx)
//	defer poller.Stop()
type OutboxPoller struct {
	outbox    event.Outbox
	publisher event.Publisher
	interval  time.Duration
	batchSize int
	logger    *slog.Logger
	cancel    context.CancelFunc
}

// OutboxPollerOption configures an OutboxPoller.
type OutboxPollerOption func(*OutboxPoller)

// WithPollInterval sets how often the poller checks for new outbox entries.
// Default: 5 seconds.
func WithPollInterval(d time.Duration) OutboxPollerOption {
	return func(p *OutboxPoller) { p.interval = d }
}

// WithBatchSize sets the maximum number of outbox entries to process per poll.
// Default: 100.
func WithBatchSize(n int) OutboxPollerOption {
	return func(p *OutboxPoller) { p.batchSize = n }
}

// WithPollerLogger sets the logger for the poller.
func WithPollerLogger(l *slog.Logger) OutboxPollerOption {
	return func(p *OutboxPoller) { p.logger = l }
}

// NewOutboxPoller creates a background outbox relay worker.
func NewOutboxPoller(
	outbox event.Outbox,
	publisher event.Publisher,
	opts ...OutboxPollerOption,
) *OutboxPoller {
	p := &OutboxPoller{
		outbox:    outbox,
		publisher: publisher,
		interval:  5 * time.Second,
		batchSize: 100,
		logger:    slog.Default(),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Start begins polling the outbox in a background goroutine.
// The poller runs until the context is cancelled or Stop is called.
func (p *OutboxPoller) Start(ctx context.Context) {
	ctx, p.cancel = context.WithCancel(ctx)

	go p.loop(ctx)
}

// Stop signals the poller to stop and waits for the current iteration to complete.
func (p *OutboxPoller) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
}

func (p *OutboxPoller) loop(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.poll(ctx); err != nil {
				p.logger.Error("outbox poll failed", "error", err)
			}
		}
	}
}

func (p *OutboxPoller) poll(ctx context.Context) error {
	entries, err := p.outbox.PollPending(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("poll pending: %w", err)
	}

	var ackIDs []event.OutboxID

	for _, entry := range entries {
		if err := p.publishEntry(ctx, entry); err != nil {
			p.logger.Error("failed to publish outbox entry", "id", entry.ID, "error", err)
			continue
		}

		ackIDs = append(ackIDs, entry.ID)
	}

	if len(ackIDs) > 0 {
		if err := p.outbox.Ack(ctx, ackIDs); err != nil {
			return fmt.Errorf("ack %d entries: %w", len(ackIDs), err)
		}

		p.logger.Info(
			"outbox relay processed",
			"entries",
			len(ackIDs),
			"events",
			p.countEvents(entries),
		)
	}

	return nil
}

func (p *OutboxPoller) publishEntry(ctx context.Context, entry event.OutboxEntry) error {
	for _, evt := range entry.Events {
		if err := p.publisher.Publish(ctx, evt); err != nil {
			return fmt.Errorf("publish event %s: %w", evt.ID(), err)
		}
	}

	return nil
}

func (p *OutboxPoller) countEvents(entries []event.OutboxEntry) int {
	var count int

	for _, e := range entries {
		count += len(e.Events)
	}

	return count
}
