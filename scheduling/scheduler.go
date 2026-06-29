package scheduling

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// Scheduler polls a TimerStore for due timers and dispatches them.
type Scheduler struct {
	store    TimerStore
	dispatch DispatchFunc
	opts     schedulerOptions
	logger   *slog.Logger
}

type schedulerOptions struct {
	pollInterval time.Duration
	maxRetries   int
}

// Option configures a Scheduler.
type Option func(*schedulerOptions)

func defaultOptions() schedulerOptions {
	return schedulerOptions{
		pollInterval: 1 * time.Second,
		maxRetries:   3,
	}
}

// WithPollInterval sets how often the scheduler checks for due timers.
// Default: 1 second.
func WithPollInterval(d time.Duration) Option {
	return func(o *schedulerOptions) { o.pollInterval = d }
}

// WithMaxRetries sets the max retry count for failed dispatches.
// Default: 3.
func WithMaxRetries(n int) Option {
	return func(o *schedulerOptions) { o.maxRetries = n }
}

// WithLogger sets a structured logger. Default: slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(_ *schedulerOptions) {}
}

// New creates a Scheduler that polls store and dispatches via dispatch.
func New(store TimerStore, dispatch DispatchFunc, opts ...Option) *Scheduler {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	return &Scheduler{
		store:    store,
		dispatch: dispatch,
		opts:     o,
		logger:   slog.Default(),
	}
}

// Start begins polling for due timers. Blocks until ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.opts.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.tick(ctx); err != nil {
				s.logger.Warn("scheduler tick failed", "error", err)
			}
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) error {
	now := time.Now()

	due, err := s.store.Due(ctx, now)
	if err != nil {
		return err
	}

	for _, timer := range due {
		if err := s.dispatchWithRetry(ctx, timer); err != nil {
			s.logger.Error(
				"timer dispatch failed after retries",
				"timer_id", timer.ID,
				"error", err,
			)
		}

		_ = s.store.MarkFired(ctx, timer.ID)
	}

	return nil
}

func (s *Scheduler) dispatchWithRetry(ctx context.Context, timer Timer) error {
	var lastErr error

	for range s.opts.maxRetries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := s.dispatch(ctx, timer)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return errors.Join(lastErr)
}
