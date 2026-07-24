package decider

import (
	"context"
	"errors"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// ErrWaitTimeout is returned by [Repository.WaitForVersion] when the target
// version does not become visible within the configured timeout.
var ErrWaitTimeout = errorfamily.NewTransient(
	"decider.wait_timeout",
	"timed out waiting for version to become visible",
)

// WaitOption configures [Repository.WaitForVersion].
type WaitOption func(*waitConfig)

type waitConfig struct {
	timeout      time.Duration
	pollInterval time.Duration
}

// WithWaitTimeout sets the maximum time [Repository.WaitForVersion] will poll
// before returning [ErrWaitTimeout]. Default: 2s.
func WithWaitTimeout(d time.Duration) WaitOption {
	return func(c *waitConfig) { c.timeout = d }
}

// WithPollInterval sets the interval between polls in [Repository.WaitForVersion].
// Default: 10ms.
func WithPollInterval(d time.Duration) WaitOption {
	return func(c *waitConfig) { c.pollInterval = d }
}

// WaitForVersion polls the event store until the target version is visible,
// then returns the events at or after that version. This implements the
// read-your-writes consistency pattern for distributed setups where the
// event store has read replicas or the write was performed by a different
// process.
//
// In a single-process setup with a synchronous store (MemoryStore, SQLite),
// the version is immediately visible after [Repository.Execute] returns, so
// this function returns without polling.
//
// Returns [ErrWaitTimeout] if the version does not become visible within the
// timeout (default 2s). Use [WithWaitTimeout] to override.
func (r *Repository[State]) WaitForVersion(
	ctx context.Context,
	streamID id.StreamID,
	streamType id.StreamType,
	version event.Version,
	opts ...WaitOption,
) ([]event.Event, error) {
	cfg := waitConfig{
		timeout:      2 * time.Second,
		pollInterval: 10 * time.Millisecond,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	ref := id.NewStreamRef(streamType, streamID)

	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "decider.wait_for_version",
		cqrsotel.SpanKindInternal,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString(cqrsotel.AttrStreamType, string(streamType)),
			cqrsotel.AttrString(cqrsotel.AttrStreamID, streamID.String()),
			cqrsotel.AttrInt(cqrsotel.AttrStreamVersion, version.Int()),
		),
	)
	defer span.End()

	pollCtx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	ticker := time.NewTicker(cfg.pollInterval)
	defer ticker.Stop()

	// Check version parameter is valid.
	if version <= 0 {
		return nil, errorfamily.NewRejection(
			"decider.wait_invalid_version",
			"version must be >= 1",
		)
	}

	for {
		events, err := r.store.LoadFromVersion(pollCtx, ref, version-1)
		if err != nil {
			if errors.Is(err, event.ErrStreamNotFound) {
				// Stream doesn't exist yet — keep polling.
			} else {
				cqrsotel.RecordError(span, err)

				return nil, opError(ref, "%w: %w", ErrLoadFailed, err)
			}
		}

		if len(events) > 0 {
			return events, nil
		}

		select {
		case <-pollCtx.Done():
			cqrsotel.RecordError(span, ErrWaitTimeout)

			return nil, ErrWaitTimeout
		case <-ticker.C:
			// Continue polling.
		}
	}
}
