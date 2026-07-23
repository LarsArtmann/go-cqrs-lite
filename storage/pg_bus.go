package storage

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// DefaultPostgresBusChannel is the default LISTEN/NOTIFY channel name.
const DefaultPostgresBusChannel = "cqrs_events"

// defaultRefetchAttempts is how many times the listener retries re-fetching
// an event from the store before giving up (handles the case where a NOTIFY
// arrives before the producing transaction is visible to the listener).
const defaultRefetchAttempts = 5

// defaultRefetchDelay is the delay between re-fetch attempts.
const defaultRefetchDelay = 50 * time.Millisecond

// notifyPayload is the lightweight JSON sent via NOTIFY.
// It carries only references — never the event payload itself — to stay
// well under Postgres's 8KB NOTIFY payload limit. All fields are branded
// domain types: JSON (de)serialization is handled by each type's
// MarshalText/UnmarshalText (ULID for IDs, plain string for Type/StreamType,
// custom MarshalJSON for Version). This eliminates the string-roundtrip
// (String() → parse) the previous version did on the receive side.
type notifyPayload struct {
	EventID    id.EventID    `json:"eid"`
	EventType  event.Type    `json:"et"`
	StreamType id.StreamType `json:"at"`
	StreamID   id.StreamID   `json:"aid"`
	Version    event.Version `json:"v"`
}

// NotificationListener abstracts the driver-specific LISTEN mechanism.
// Consumers implement this for their Postgres driver:
//
//	// pgxpool-based example
//	type PgxListener struct { pool *pgxpool.Pool }
//	func (p *PgxListener) Listen(ctx context.Context, ch string) error { ... }
//	func (p *PgxListener) Notifications() <-chan string { ... }
//	func (p *PgxListener) Close() error { ... }
//
// The bus calls Listen itself (with the configured channel) before starting
// its receive goroutine, so the consumer never has to remember the call.
type NotificationListener interface {
	// Listen subscribes to NOTIFY on the given channel. Must be called once
	// before Notifications() starts delivering payloads. The listener may use
	// ctx only for the LISTEN handshake; the receive loop is owned by the
	// listener and cancelled via Close.
	Listen(ctx context.Context, channel string) error

	// Notifications returns a channel that receives NOTIFY payload strings.
	// The channel is closed when the listener stops.
	Notifications() <-chan string

	// Close stops listening and releases the connection.
	Close() error
}

// postgresBusOptions configures a PostgresBus.
type postgresBusOptions struct {
	channel         string
	refetchAttempts int
	refetchDelay    time.Duration
	logger          *slog.Logger
	notifyFn        notifyFunc
}

// PostgresBusOption configures a PostgresBus.
type PostgresBusOption func(*postgresBusOptions)

// WithBusChannel sets the LISTEN/NOTIFY channel name. Defaults to "cqrs_events".
func WithBusChannel(channel string) PostgresBusOption {
	return func(o *postgresBusOptions) { o.channel = channel }
}

// WithRefetchAttempts sets how many times the listener retries fetching an event
// from the store before giving up. Defaults to 5. Handles the visibility gap
// where a NOTIFY arrives before the producing transaction is committed.
func WithRefetchAttempts(n int) PostgresBusOption {
	return func(o *postgresBusOptions) { o.refetchAttempts = n }
}

// WithRefetchDelay sets the delay between re-fetch retry attempts.
func WithRefetchDelay(d time.Duration) PostgresBusOption {
	return func(o *postgresBusOptions) { o.refetchDelay = d }
}

// WithNotifyFunc overrides the NOTIFY mechanism. Useful for testing or
// custom Postgres driver configurations.
func WithNotifyFunc(fn notifyFunc) PostgresBusOption {
	return func(o *postgresBusOptions) { o.notifyFn = fn }
}

// defaultNotifyFunc returns the standard pg_notify implementation.
func defaultNotifyFunc(db *sql.DB) notifyFunc {
	return func(ctx context.Context, channel, payload string) error {
		_, err := db.ExecContext(ctx, "SELECT pg_notify($1, $2)", channel, payload)
		return err
	}
}

// PostgresBus implements event.Bus using Postgres LISTEN/NOTIFY for
// cross-process event propagation. Multiple processes sharing one database
// can publish and receive events.
//
// Publish sends a lightweight NOTIFY with the event reference (not the full
// payload, respecting the 8KB NOTIFY limit). Listeners on other processes
// re-fetch the full event from the event store.
//
// The NOTIFY side works with any database/sql Postgres driver via
// `SELECT pg_notify()`. The LISTEN side requires a driver-specific
// NotificationListener that the consumer provides.
//
// Backpressure: the listener's notifications channel has a bounded buffer
// (default 256). When a handler is slow and the buffer fills, the listener's
// receive loop blocks, which in turn blocks WaitForNotification. Postgres
// queues NOTIFY payloads server-side (default 8GB). If the server queue
// overflows, Postgres forcibly disconnects all listeners. This is natural
// backpressure — slow consumers must catch up or the connection is dropped.
//
// Usage:
//
//	db, _ := sql.Open("pgx", dsn)
//	store, _ := storage.NewSQLEventStore(db)
//	listener := &MyPQListener{...}
//	bus, _ := storage.NewPostgresBus(db, store, listener)
//	defer bus.Close()
//	bus.Subscribe("user.created", handler)
//
// notifyFunc sends a NOTIFY payload to other processes.
// The default implementation uses SELECT pg_notify().
type notifyFunc func(ctx context.Context, channel, payload string) error

type PostgresBus struct {
	db    *sql.DB
	store event.EventSource
	opts  postgresBusOptions

	listener NotificationListener

	mu                sync.RWMutex
	handlers          map[event.Type][]event.Handler
	allHandlers       []event.Handler
	middleware        []event.Middleware
	publishMiddleware []event.PublishMiddleware
	cachedHandler     event.Handler
	cachedPublisher   event.Publisher

	closed   atomic.Bool
	wg       sync.WaitGroup
	cancelFn context.CancelFunc
}

var (
	_ event.Bus = (*PostgresBus)(nil)
	_ io.Closer = (*PostgresBus)(nil)
)

// ErrNilNotificationListener is returned when a nil listener is passed to NewPostgresBus.
var ErrNilNotificationListener = errorfamily.NewInfrastructure(
	"storage.nil_notification_listener",
	"storage: nil notification listener",
)

// errNilBusHandler is a sentinel for nil handler arguments.
var errNilBusHandler = errorfamily.NewInfrastructure(
	"storage.nil_bus_handler",
	"storage: nil bus handler",
)

// errNilEventSource is a sentinel for nil event source arguments.
var errNilEventSource = errorfamily.NewInfrastructure(
	"storage.nil_event_source",
	"storage: nil event source",
)

// errEventNotFoundAfterRetries is the classified sentinel for re-fetch
// failures. Uses errorfamily.NewInfrastructure for consistency with the rest of
// the storage error taxonomy (go-error-family); supports errors.Is/As.
var errEventNotFoundAfterRetries = errorfamily.NewInfrastructure(
	"storage.event_not_found_after_retries",
	"event not found after retries",
)

// NewPostgresBus creates a LISTEN/NOTIFY-backed event bus.
// The db is used for NOTIFY (SELECT pg_notify). The store is used by the
// listener to re-fetch full events when notifications arrive from other processes.
// The listener provides the driver-specific LISTEN mechanism.
//
// The bus calls listener.Listen(channel) itself before starting its receive
// goroutine; consumers do not need to pre-arm the listener.
func NewPostgresBus(
	db *sql.DB,
	store event.EventSource,
	listener NotificationListener,
	opts ...PostgresBusOption,
) (*PostgresBus, error) {
	if db == nil {
		return nil, errorfamily.WrapInfrastructure(ErrNilDB, "storage.create_pg_bus",
			"create postgres bus: nil db")
	}

	if store == nil {
		return nil, errorfamily.WrapInfrastructure(errNilEventSource, "storage.create_pg_bus",
			"create postgres bus: nil event source")
	}

	if listener == nil {
		return nil, errorfamily.WrapInfrastructure(
			ErrNilNotificationListener,
			"storage.create_pg_bus",
			"create postgres bus: nil notification listener",
		)
	}

	o := postgresBusOptions{
		channel:         DefaultPostgresBusChannel,
		refetchAttempts: defaultRefetchAttempts,
		refetchDelay:    defaultRefetchDelay,
		logger:          slog.Default(),
		notifyFn:        defaultNotifyFunc(db),
	}

	for _, opt := range opts {
		opt(&o)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := listener.Listen(ctx, o.channel); err != nil {
		cancel()
		return nil, errorfamily.WrapInfrastructure(err, "storage.pg_bus_listen",
			"listener.Listen on channel "+o.channel)
	}

	b := &PostgresBus{
		db:       db,
		store:    store,
		opts:     o,
		listener: listener,
		handlers: make(map[event.Type][]event.Handler),
		cancelFn: cancel,
	}

	b.rebuildHandlerChain()
	b.rebuildPublisherChain()

	b.wg.Add(1)

	go b.listenLoop(ctx)

	return b, nil
}
