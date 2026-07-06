package storage

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

// listenLoop processes incoming notifications from the listener.
// It re-fetches the full event from the store and dispatches to local handlers.
func (b *PostgresBus) listenLoop(ctx context.Context) {
	defer b.wg.Done()

	notifications := b.listener.Notifications()

	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-notifications:
			if !ok {
				return
			}

			b.handleNotification(ctx, payload)
		}
	}
}

func (b *PostgresBus) handleNotification(ctx context.Context, payload string) {
	var np notifyPayload
	if err := json.Unmarshal([]byte(payload), &np); err != nil {
		b.opts.logger.ErrorContext(ctx, "failed to unmarshal notify payload", "error", err)
		return
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, sqlpkg.Tracer(), "pg_bus.handle_notification",
		cqrsotel.SpanKindConsumer,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString("cqrs.event.type", string(np.EventType)),
			cqrsotel.AttrString("cqrs.event.id", np.EventID.String()),
			cqrsotel.AttrString("cqrs.aggregate.type", string(np.AggregateType)),
		),
	)
	defer span.End()

	evt, err := b.refetchEvent(ctx, np)
	if err != nil {
		cqrsotel.RecordError(span, err)
		b.opts.logger.ErrorContext(ctx, "failed to re-fetch event from store",
			"event_id", np.EventID, "type", np.EventType, "error", err)
		return
	}

	if err := b.dispatchLocal(ctx, evt); err != nil {
		cqrsotel.RecordError(span, err)
		b.opts.logger.ErrorContext(ctx, "failed to dispatch re-fetched event",
			"event_id", np.EventID, "type", np.EventType, "error", err)
	}
}

// refetchEvent loads the full event from the store, retrying to handle
// the visibility gap where a NOTIFY arrives before the producing transaction
// is committed and visible to this connection.
//
// If the store implements EventByIDLoader (SQLEventStore does), uses the
// efficient indexed LoadByEventID path. Otherwise falls back to LoadFromVersion
// with a version scan.
func (b *PostgresBus) refetchEvent(ctx context.Context, np notifyPayload) (event.Event, error) {
	// Fast path: indexed lookup by event ID (O(1) query).
	if byIDLoader, ok := b.store.(EventByIDLoader); ok {
		return b.refetchByID(ctx, byIDLoader, np.EventID)
	}

	// Fallback: version scan (O(N) where N = events since version).
	return b.refetchByVersion(ctx, np)
}

func (b *PostgresBus) refetchByID(
	ctx context.Context,
	loader EventByIDLoader,
	eventID id.EventID,
) (event.Event, error) {
	var lastErr error

	for range b.opts.refetchAttempts {
		evt, loadErr := loader.LoadByEventID(ctx, eventID)
		if loadErr == nil {
			return evt, nil
		}

		if !errors.Is(loadErr, event.ErrEventNotFound) {
			return nil, errorfamily.WrapInfrastructure(loadErr, "pg_bus.refetch_by_id",
				"load event by ID during refetch")
		}

		lastErr = loadErr

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(b.opts.refetchDelay):
		}
	}

	return nil, errorfamily.WrapInfrastructure(
		lastErr,
		"storage.pg_bus_refetch_by_id",
		"re-fetch event "+eventID.String()+" after "+strconv.Itoa(
			b.opts.refetchAttempts,
		)+" attempts",
	)
}

func (b *PostgresBus) refetchByVersion(ctx context.Context, np notifyPayload) (event.Event, error) {
	ref := event.NewAggregateRef(np.AggregateType, np.AggregateID)

	var lastErr error

	for range b.opts.refetchAttempts {
		loadVersion := np.Version
		if loadVersion > 0 {
			loadVersion--
		}

		events, loadErr := b.store.LoadFromVersion(ctx, ref, loadVersion)
		if loadErr == nil {
			for _, evt := range events {
				if evt.Version() == np.Version {
					return evt, nil
				}
			}
		}

		lastErr = loadErr

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(b.opts.refetchDelay):
		}
	}

	if lastErr != nil {
		return nil, errorfamily.WrapInfrastructure(
			lastErr,
			"storage.pg_bus_refetch",
			"re-fetch event "+np.EventID.String()+" after "+strconv.Itoa(
				b.opts.refetchAttempts,
			)+" attempts",
		)
	}

	return nil, errorfamily.WrapInfrastructure(
		errEventNotFoundAfterRetries,
		"storage.pg_bus_refetch_not_found",
		"event "+np.EventID.String()+" not found after re-fetch attempts",
	)
}

// Close stops the listener goroutine and releases the notification listener.
// Safe to call multiple times.
func (b *PostgresBus) Close() error {
	if !b.closed.CompareAndSwap(false, true) {
		return nil
	}

	if b.cancelFn != nil {
		b.cancelFn()
	}

	b.wg.Wait()

	if b.listener != nil {
		err := b.listener.Close()
		if err != nil {
			return errorfamily.WrapInfrastructure(err, "storage.pg_bus_close_listener",
				"close notification listener")
		}
	}

	return nil
}
