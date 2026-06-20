package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/larsartmann/go-cqrs-lite/storage/v2"
)

// PgxListener is a storage.NotificationListener backed by a dedicated
// pgxpool connection. It is the canonical LISTEN-side implementation to
// pair with [storage.PostgresBus] when wiring the [Bundle] preset.
//
// One dedicated *pgx.Conn is acquired from the pool for the lifetime of the
// listener: the LISTEN handshake and the subsequent WaitForNotification loop
// both run on it. The publishing side (SELECT pg_notify) goes through the
// regular *sql.DB pool, so the listener does not contend with publishers.
//
// Auto-reconnect: if the connection drops (network blip, server restart),
// the listener automatically re-acquires a connection and re-issues LISTEN
// with exponential backoff (default: 10 attempts, 1s→30s). Disable with
// [WithoutReconnect]. Tune with [WithReconnect] and [WithReconnectBackoff].
//
// Lifecycle:
//
//	listener, _ := postgres.NewPgxListener(ctx, pool)
//	bus, _ := storage.NewPostgresBus(db, store, listener) // bus calls Listen
//	defer bus.Close() // Close drains the listener and releases the conn
type PgxListener struct {
	pool          *pgxpool.Pool
	ownsPool      bool // true when this listener created the pool (and should close it)
	conn          *pgxpool.Conn
	cancelFn      context.CancelFunc // cancels the listen-specific child context
	notifications chan string
	logger        *slog.Logger
	channel       string          // stored for reconnect (re-LISTEN after conn loss)
	reconnectCfg  reconnectConfig // auto-reconnect settings

	closeOnce sync.Once
	done      chan struct{} // closed when the receive loop has exited
	closed    atomic.Bool
}

// Concurrency invariant for conn, cancelFn, and channel:
//
// These fields are touched by exactly two goroutines — the background
// receiveLoop and the caller's Close. receiveLoop reads/writes conn inside
// receiveOnce and reconnect; Close reads/writes conn when releasing it. The two
// CANNOT overlap because Close waits on <-l.done (closed by receiveLoop's
// deferred doneClose) before touching conn. Per the Go memory model, the
// close(done) → <-done edge establishes happens-before, so Close always observes
// receiveLoop's final conn write and never races it. This is channel-based
// synchronization by design — a mutex here would needlessly serialize the
// notification hot path. See TestPgxListener_ConnAccessRaceFree.

// Compile-time check that PgxListener satisfies the bus's listener contract.
var _ storage.NotificationListener = (*PgxListener)(nil)

// ErrListenerAlreadyListening is returned when Listen is called more than once.
var ErrListenerAlreadyListening = errors.New("pgx_listener: already listening")

// ErrListenerClosed is returned when Listen is called after Close.
var ErrListenerClosed = errors.New("pgx_listener: closed")

// errEmptyChannelName is the sentinel for empty channel-name input.
var errEmptyChannelName = errors.New("pgx_listener: empty channel name")

// errInvalidChannelName is the base error for invalid channel-name input.
// The invalid value is included via fmt.Errorf wrapping.
var errInvalidChannelName = errors.New(
	"pgx_listener: invalid channel name (must be [A-Za-z_][A-Za-z0-9_]*)",
)

// NewPgxListener wraps an existing pgxpool for LISTEN/NOTIFY. The caller
// retains ownership of the pool; closing the listener only releases the
// dedicated connection it acquired, not the pool itself.
func NewPgxListener(pool *pgxpool.Pool, opts ...PgxListenerOption) *PgxListener {
	l := &PgxListener{ //nolint:exhaustruct // fields set lazily on Listen/Close
		pool:          pool,
		notifications: make(chan string, defaultListenerQueue),
		logger:        slog.Default(),
		done:          make(chan struct{}),
		reconnectCfg:  defaultReconnectConfig(),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// NewPgxListenerFromDSN creates a dedicated single-connection pool for
// LISTEN/NOTIFY. The pool (and connection) are owned by the listener and
// closed on Close. Use this when the listener should not share a pool with
// the publishing side.
func NewPgxListenerFromDSN(
	ctx context.Context,
	dsn string,
	opts ...PgxListenerOption,
) (*PgxListener, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx_listener: parse dsn: %w", err)
	}

	// A listener needs only one connection. Pool-concurrency of 1 guarantees
	// that the dedicated LISTEN conn does not get recycled into general use.
	cfg.MaxConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgx_listener: create pool: %w", err)
	}

	l := NewPgxListener(pool, opts...)
	l.ownsPool = true

	return l, nil
}

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

// Listen acquires a dedicated connection from the pool, issues LISTEN on the
// given channel, and starts a background receive loop. The loop delivers
// NOTIFY payloads to Notifications() until Close is called or the connection
// is lost.
//
// Channel names must be valid Postgres identifiers (alphanumeric + underscore,
// not starting with a digit). This matches the default "cqrs_events" and any
// reasonable consumer-defined name.
func (l *PgxListener) Listen(ctx context.Context, channel string) error {
	if l.closed.Load() {
		return ErrListenerClosed
	}

	if l.conn != nil {
		return ErrListenerAlreadyListening
	}

	err := validateChannelName(channel)
	if err != nil {
		return err
	}

	// Derive a child context so Close() can cancel the receive loop
	// independently of the caller's context. This is critical: pgxpool's
	// Conn.Release() does NOT interrupt in-flight WaitForNotification, so
	// without our own cancel we would deadlock in Close() waiting on
	// <-l.done while receiveLoop is stuck on WaitForNotification.
	listenCtx, cancel := context.WithCancel(ctx)

	conn, err := l.pool.Acquire(listenCtx)
	if err != nil {
		cancel()

		return fmt.Errorf("pgx_listener: acquire conn: %w", err)
	}

	// Postgres LISTEN takes an unquoted identifier (lowercased) or a
	// double-quoted identifier (case preserved). Quoting keeps arbitrary
	// validated names round-tripping with pg_notify's string arg.
	_, err = conn.Exec(listenCtx, fmt.Sprintf(`LISTEN "%s"`, channel))
	if err != nil {
		cancel()
		conn.Release()

		return fmt.Errorf("pgx_listener: LISTEN %q: %w", channel, err)
	}

	l.conn = conn
	l.cancelFn = cancel
	l.channel = channel

	go l.receiveLoop(listenCtx)

	return nil
}

// receiveLoop is the outer reconnect loop. It delegates one notification
// cycle to receiveOnce, and on connection error attempts to reconnect with
// exponential backoff. The loop exits (closing the notifications channel)
// only when: the context is cancelled (Close), or all reconnect attempts
// are exhausted. This is how consumers detect end-of-stream.
// doneClose closes the notifications channel exactly once. Used as the
// receiveLoop deferred exit.
func (l *PgxListener) doneClose() {
	close(l.done)
	close(l.notifications)
}

// Notifications returns the channel of NOTIFY payloads. The channel is closed
// when the listener stops (Close called, context cancelled, or connection lost).
func (l *PgxListener) Notifications() <-chan string {
	return l.notifications
}

// Close cancels the listen context (which unblocks WaitForNotification in
// the receive loop), waits for the loop to exit, then releases the dedicated
// connection and — if the listener owns the pool — closes the pool.
// Safe to call multiple times (sync.Once guard).
//
// Graceful drain: after Close returns, the notifications channel is closed
// (consumers detect end-of-stream). Any payload already buffered in the
// channel remains readable until drained. No new NOTIFY payloads will arrive.
func (l *PgxListener) Close() error {
	l.closeOnce.Do(func() {
		l.closed.Store(true)

		// Cancel the listen context FIRST. This unblocks receiveLoop's
		// WaitForNotification, which otherwise blocks on the network fd
		// indefinitely (pgxpool Conn.Release does NOT interrupt it).
		if l.cancelFn != nil {
			l.cancelFn()
		}

		// Wait for receiveLoop to exit before touching conn or pool.
		// Only wait if Listen was actually called (cancelFn set).
		// This prevents pool.Close() from deadlocking on an unreleased conn.
		if l.cancelFn != nil && l.done != nil {
			<-l.done
		}

		if l.conn != nil {
			l.conn.Release()
			l.conn = nil
		}

		if l.ownsPool && l.pool != nil {
			l.pool.Close()
		}
	})

	return nil
}

// validateChannelName rejects names that are not safe Postgres identifiers.
// We compose LISTEN via fmt.Sprintf (Postgres does not parameterize LISTEN),
// so allow-listing is the SQL-injection defence.
func validateChannelName(channel string) error {
	if channel == "" {
		return errEmptyChannelName
	}

	for i, r := range channel {
		ok := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(i > 0 && r >= '0' && r <= '9')
		if !ok {
			return fmt.Errorf("%w: %q", errInvalidChannelName, channel)
		}
	}

	return nil
}
