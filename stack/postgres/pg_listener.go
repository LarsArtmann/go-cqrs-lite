package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

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
// Lifecycle:
//
//	listener, _ := postgres.NewPgxListener(ctx, pool)
//	bus, _ := storage.NewPostgresBus(db, store, listener) // bus calls Listen
//	defer bus.Close() // Close drains the listener and releases the conn
type PgxListener struct {
	pool          *pgxpool.Pool
	ownsPool      bool // true when this listener created the pool (and should close it)
	conn          *pgxpool.Conn
	notifications chan string
	logger        *slog.Logger

	closeOnce sync.Once
	done      chan struct{} // closed when the receive loop has exited
	closed    atomic.Bool
}

// Compile-time check that PgxListener satisfies the bus's listener contract.
var _ storage.NotificationListener = (*PgxListener)(nil)

// ErrListenerAlreadyListening is returned when Listen is called more than once.
var ErrListenerAlreadyListening = errors.New("pgx_listener: already listening")

// ErrListenerClosed is returned when Listen is called after Close.
var ErrListenerClosed = errors.New("pgx_listener: closed")

// NewPgxListener wraps an existing pgxpool for LISTEN/NOTIFY. The caller
// retains ownership of the pool; closing the listener only releases the
// dedicated connection it acquired, not the pool itself.
func NewPgxListener(pool *pgxpool.Pool, opts ...PgxListenerOption) *PgxListener {
	l := &PgxListener{
		pool:          pool,
		notifications: make(chan string, defaultListenerQueue),
		logger:        slog.Default(),
		done:          make(chan struct{}),
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

	if err := validateChannelName(channel); err != nil {
		return err
	}

	conn, err := l.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("pgx_listener: acquire conn: %w", err)
	}

	// Postgres LISTEN takes an unquoted identifier (lowercased) or a
	// double-quoted identifier (case preserved). Quoting keeps arbitrary
	// validated names round-tripping with pg_notify's string arg.
	_, err = conn.Exec(ctx, fmt.Sprintf(`LISTEN "%s"`, channel))
	if err != nil {
		conn.Release()
		return fmt.Errorf("pgx_listener: LISTEN %q: %w", channel, err)
	}

	l.conn = conn

	go l.receiveLoop(ctx)

	return nil
}

// receiveLoop blocks on WaitForNotification and pushes payloads until ctx
// is cancelled, the connection errors, or Close drains it. The loop exits
// by closing the notifications channel — that's how consumers detect end-of-stream.
func (l *PgxListener) receiveLoop(ctx context.Context) {
	defer l.doneClose()

	for {
		// WaitForNotification returns on NOTIFY, ctx cancel, or connection loss.
		notification, err := l.conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// Context cancellation — expected on Close. Exit quietly.
				return
			}
			l.logger.ErrorContext(
				ctx,
				"pgx_listener: WaitForNotification failed; exiting receive loop",
				"error",
				err,
			)
			return
		}

		select {
		case l.notifications <- notification.Payload:
		case <-ctx.Done():
			return
		}
	}
}

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

// Close releases the dedicated connection (and the pool, if owned). Safe to
// call multiple times.
func (l *PgxListener) Close() error {
	var firstErr error

	l.closeOnce.Do(func() {
		l.closed.Store(true)

		if l.conn != nil {
			// Closing the connection cancels any in-flight WaitForNotification,
			// unblocking the receive loop.
			l.conn.Release()

			<-l.done // wait for receiveLoop to exit cleanly
			l.conn = nil
		}

		if l.ownsPool && l.pool != nil {
			l.pool.Close()
		}
	})

	return firstErr
}

// validateChannelName rejects names that are not safe Postgres identifiers.
// We compose LISTEN via fmt.Sprintf (Postgres does not parameterize LISTEN),
// so allow-listing is the SQL-injection defence.
func validateChannelName(channel string) error {
	if channel == "" {
		return errors.New("pgx_listener: empty channel name")
	}

	for i, r := range channel {
		ok := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(i > 0 && r >= '0' && r <= '9')
		if !ok {
			return fmt.Errorf(
				"pgx_listener: invalid channel name %q (must be [A-Za-z_][A-Za-z0-9_]*)",
				channel,
			)
		}
	}

	return nil
}
