package postgres

import (
	"log/slog"
	"time"
)

// PgxListenerOption configures a PgxListener.
type PgxListenerOption func(*PgxListener)

// WithPgxListenerLogger sets the logger used for connection errors.
func WithPgxListenerLogger(logger *slog.Logger) PgxListenerOption {
	return func(l *PgxListener) { l.logger = logger }
}

// WithPgxListenerQueue sets the buffer size of the Notifications channel.
// Default is [defaultListenerQueue]. A larger buffer helps ride bursts of
// NOTIFY traffic without blocking the receive loop.
func WithPgxListenerQueue(size int) PgxListenerOption {
	return func(l *PgxListener) {
		if size > 0 {
			l.notifications = make(chan string, size)
		}
	}
}

// defaultListenerQueue matches Postgres's typical NOTIFY burst tolerance.
const defaultListenerQueue = 256

// Reconnect defaults — overridable via WithReconnect / WithReconnectBackoff.
const (
	defaultReconnectMaxAttempts = 10
	defaultReconnectInitBackoff = 1 * time.Second
	defaultReconnectMaxBackoff  = 30 * time.Second
	backoffShiftCap             = 10 // cap shift at 10 (1024×) to avoid Duration overflow
)

// reconnectConfig controls automatic reconnection when the LISTEN connection
// drops. A dropped connection silently kills event delivery without it.
type reconnectConfig struct {
	enabled        bool
	maxAttempts    int
	initialBackoff time.Duration
	maxBackoff     time.Duration
}

func defaultReconnectConfig() reconnectConfig {
	return reconnectConfig{
		enabled:        true,
		maxAttempts:    defaultReconnectMaxAttempts,
		initialBackoff: defaultReconnectInitBackoff,
		maxBackoff:     defaultReconnectMaxBackoff,
	}
}

// WithReconnect sets the maximum number of reconnect attempts after a
// connection loss. Default is 10 attempts with exponential backoff
// (1s → 2s → 4s → … → 30s cap). Set to 0 to disable auto-reconnect.
func WithReconnect(maxAttempts int) PgxListenerOption {
	return func(l *PgxListener) {
		if maxAttempts <= 0 {
			l.reconnectCfg.enabled = false
		} else {
			l.reconnectCfg.enabled = true
			l.reconnectCfg.maxAttempts = maxAttempts
		}
	}
}

// WithReconnectBackoff configures the exponential backoff schedule for
// auto-reconnect. initial is the delay before the first retry; maxDelay is the
// cap on subsequent delays. Defaults: 1s initial, 30s max.
func WithReconnectBackoff(initial, maxDelay time.Duration) PgxListenerOption {
	return func(l *PgxListener) {
		if initial > 0 {
			l.reconnectCfg.initialBackoff = initial
		}

		if maxDelay > 0 {
			l.reconnectCfg.maxBackoff = maxDelay
		}
	}
}

// WithoutReconnect disables auto-reconnect entirely. The listener will
// stop receiving events when the connection drops and must be manually
// restarted.
func WithoutReconnect() PgxListenerOption {
	return func(l *PgxListener) { l.reconnectCfg.enabled = false }
}
