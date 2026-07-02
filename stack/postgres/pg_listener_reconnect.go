package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// receiveLoop processes incoming LISTEN/NOTIFY messages. On connection error
// it attempts to reconnect (if enabled). Exits when the context is cancelled
// or reconnect attempts are exhausted. This is how consumers detect end-of-stream.
func (l *PgxListener) receiveLoop(ctx context.Context) {
	defer l.doneClose()

	for {
		err := l.receiveOnce(ctx)

		if ctx.Err() != nil {
			return // Close called — exit quietly.
		}

		if err == nil {
			continue
		}

		// Connection error — attempt reconnect.
		if !l.reconnect(ctx, err) {
			return // reconnect disabled or all attempts exhausted.
		}
	}
}

// receiveOnce blocks on a single WaitForNotification cycle and pushes the
// payload to the notifications channel. Returns nil on success, or the
// error from WaitForNotification / ctx cancellation.
func (l *PgxListener) receiveOnce(ctx context.Context) error {
	notification, err := l.conn.Conn().WaitForNotification(ctx)
	if err != nil {
		return event.WrapInfrastructure(err, "postgres.wait_notification",
			"wait for notification")
	}

	select {
	case l.notifications <- notification.Payload:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("pgx_listener: cancelled during send: %w", ctx.Err())
	}
}

// reconnect attempts to re-acquire a connection and re-issue LISTEN after a
// connection error. Returns true if reconnected successfully, false if
// reconnect is disabled, the context was cancelled (Close), or all attempts
// were exhausted.
func (l *PgxListener) reconnect(ctx context.Context, lastErr error) bool {
	if !l.reconnectCfg.enabled {
		l.logger.ErrorContext(
			ctx,
			"pgx_listener: connection lost; auto-reconnect disabled",
			"error", lastErr,
		)

		return false
	}

	for attempt := range l.reconnectCfg.maxAttempts {
		delay := l.backoffDuration(attempt)

		l.logger.WarnContext(
			ctx,
			"pgx_listener: reconnecting",
			"attempt", attempt+1,
			"max", l.reconnectCfg.maxAttempts,
			"delay", delay,
			"last_error", lastErr,
		)

		select {
		case <-ctx.Done():
			return false
		case <-time.After(delay):
		}

		if l.conn != nil {
			l.conn.Release()
			l.conn = nil
		}

		conn, err := l.pool.Acquire(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return false
			}

			lastErr = err

			continue
		}

		_, err = conn.Exec(ctx, fmt.Sprintf(`LISTEN "%s"`, l.channel))
		if err != nil {
			conn.Release()

			if ctx.Err() != nil {
				return false
			}

			lastErr = err

			continue
		}

		l.conn = conn
		l.logger.InfoContext(
			ctx,
			"pgx_listener: reconnected",
			"attempt", attempt+1,
		)

		return true
	}

	l.logger.ErrorContext(
		ctx,
		"pgx_listener: reconnect failed after all attempts",
		"attempts", l.reconnectCfg.maxAttempts,
		"last_error", lastErr,
	)

	return false
}

// backoffDuration computes the exponential backoff delay for a given attempt
// index (0-based). The delay doubles each attempt, capped at maxBackoff.
func (l *PgxListener) backoffDuration(attempt int) time.Duration {
	shift := min(attempt, backoffShiftCap)
	delay := l.reconnectCfg.initialBackoff * time.Duration(1<<uint(shift))

	return min(delay, l.reconnectCfg.maxBackoff)
}
