package scheduling

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"
)

// Scheduler polls a TimerStore for due timers and dispatches them.
// The type parameter P is the timer payload, forwarded to DispatchFunc.
type Scheduler[P any] struct {
	store    TimerStore[P]
	dispatch DispatchFunc[P]
	opts     schedulerOptions
	logger   *slog.Logger
}

const (
	defaultPollInterval = 1 * time.Second
	defaultMaxRetries   = 3
	defaultRetryDelay   = 100 * time.Millisecond
	jitterHalfDivisor   = 2
)

type schedulerOptions struct {
	pollInterval time.Duration
	maxRetries   int
	retryDelay   time.Duration
	logger       *slog.Logger
}

// Option configures a Scheduler.
type Option func(*schedulerOptions)

func defaultOptions() schedulerOptions {
	return schedulerOptions{ //nolint:exhaustruct // logger set by New() if nil
		pollInterval: defaultPollInterval,
		maxRetries:   defaultMaxRetries,
		retryDelay:   defaultRetryDelay,
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

// WithRetryDelay sets the base delay between dispatch retry attempts.
// The actual delay is randomized with full jitter (0 to retryDelay*2^attempt)
// to avoid thundering-herd retries when many timers fire at once. Default: 100ms.
func WithRetryDelay(d time.Duration) Option {
	return func(o *schedulerOptions) { o.retryDelay = d }
}

// WithLogger sets a structured logger. Default: slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(o *schedulerOptions) {
		if l != nil {
			o.logger = l
		}
	}
}

// New creates a Scheduler that polls store and dispatches via dispatch.
func New[P any](store TimerStore[P], dispatch DispatchFunc[P], opts ...Option) *Scheduler[P] {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	logger := o.logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler[P]{
		store:    store,
		dispatch: dispatch,
		opts:     o,
		logger:   logger,
	}
}

// Start begins polling for due timers. Blocks until ctx is cancelled.
func (s *Scheduler[P]) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.opts.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("scheduler stopped: %w", ctx.Err())
		case <-ticker.C:
			if err := s.tick(ctx); err != nil {
				s.logger.Warn("scheduler tick failed", "error", err)
			}
		}
	}
}

func (s *Scheduler[P]) tick(ctx context.Context) error {
	now := time.Now()

	due, err := s.store.Due(ctx, now)
	if err != nil {
		return fmt.Errorf("failed to query due timers: %w", err)
	}

	for _, timer := range due {
		if err := s.dispatchWithRetry(ctx, timer); err != nil {
			s.logger.Error(
				"timer dispatch failed after retries; timer remains due for next poll",
				"timer_id", timer.ID,
				"error", err,
			)

			continue
		}

		if err := s.store.MarkFired(ctx, timer.ID); err != nil {
			s.logger.Error(
				"failed to mark timer as fired; timer may re-fire on next poll",
				"timer_id", timer.ID,
				"error", err,
			)
		}
	}

	return nil
}

func (s *Scheduler[P]) dispatchWithRetry(ctx context.Context, timer Timer[P]) error {
	var lastErr error

	for attempt := range s.opts.maxRetries {
		select {
		case <-ctx.Done():
			return fmt.Errorf("scheduler stopped: %w", ctx.Err())
		default:
		}

		err := s.dispatch(ctx, timer)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't sleep after the final attempt.
		if attempt < s.opts.maxRetries-1 {
			// Equal jitter: guaranteed minimum of cap/2, plus random up to cap/2.
			// Better than full jitter for per-message retries where a guaranteed
			// minimum delay gives the downstream a real window to recover.
			exp := s.opts.retryDelay * time.Duration(1<<uint(attempt))
			half := int64(exp) / jitterHalfDivisor
			delay := time.Duration(half + rand.Int64N(half+1)) //nolint:gosec // non-crypto jitter

			select {
			case <-ctx.Done():
				return fmt.Errorf("scheduler stopped during retry backoff: %w", ctx.Err())
			case <-time.After(delay):
			}
		}
	}

	return errors.Join(lastErr)
}
