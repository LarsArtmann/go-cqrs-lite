package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
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
// MarshalText/UnmarshalText (ULID for IDs, plain string for Type/AggregateType,
// custom MarshalJSON for Version). This eliminates the string-roundtrip
// (String() → parse) the previous version did on the receive side.
type notifyPayload struct {
	EventID       id.EventID          `json:"eid"`
	EventType     event.Type          `json:"et"`
	AggregateType event.AggregateType `json:"at"`
	AggregateID   id.AggregateID      `json:"aid"`
	Version       event.Version       `json:"v"`
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
var ErrNilNotificationListener = event.NewInfrastructure(
	"storage.nil_notification_listener",
	"storage: nil notification listener",
)

// errNilBusHandler is a sentinel for nil handler arguments.
var errNilBusHandler = event.NewInfrastructure(
	"storage.nil_bus_handler",
	"storage: nil bus handler",
)

// errNilEventSource is a sentinel for nil event source arguments.
var errNilEventSource = event.NewInfrastructure(
	"storage.nil_event_source",
	"storage: nil event source",
)

// errEventNotFoundAfterRetries is the classified sentinel for re-fetch
// failures. Uses event.NewInfrastructure for consistency with the rest of
// the storage error taxonomy (go-error-family); supports errors.Is/As.
var errEventNotFoundAfterRetries = event.NewInfrastructure(
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
		return nil, event.WrapInfrastructure(ErrNilDB, "storage.create_pg_bus",
			"create postgres bus: nil db")
	}

	if store == nil {
		return nil, event.WrapInfrastructure(errNilEventSource, "storage.create_pg_bus",
			"create postgres bus: nil event source")
	}

	if listener == nil {
		return nil, event.WrapInfrastructure(ErrNilNotificationListener, "storage.create_pg_bus",
			"create postgres bus: nil notification listener")
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
		return nil, event.WrapInfrastructure(err, "storage.pg_bus_listen",
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

// Publish dispatches events to local handlers and sends a NOTIFY so other
// processes can re-fetch and process the event. The NOTIFY payload is a
// lightweight JSON reference — never the full event payload.
func (b *PostgresBus) Publish(ctx context.Context, events ...event.Event) error {
	if b.closed.Load() {
		return event.WrapInfrastructure(event.ErrBusClosed, "storage.pg_bus_publish",
			"postgres bus publish: bus is closed")
	}

	if len(events) == 0 {
		return nil
	}

	return b.cachedPublisher.Publish(ctx, events...)
}

func (b *PostgresBus) rebuildPublisherChain() {
	var inner event.Publisher = event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
		for _, evt := range events {
			if err := b.publishOne(ctx, evt); err != nil {
				return err
			}
		}

		return nil
	})

	for _, m := range slices.Backward(b.publishMiddleware) {
		inner = m(inner)
	}

	b.cachedPublisher = inner
}

func (b *PostgresBus) publishOne(ctx context.Context, evt event.Event) error {
	if err := b.dispatchLocal(ctx, evt); err != nil {
		return err
	}

	payload := notifyPayload{
		EventID:       evt.ID(),
		EventType:     evt.Type(),
		AggregateType: evt.AggregateType(),
		AggregateID:   evt.AggregateID(),
		Version:       evt.Version(),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.pg_bus_marshal",
			"marshal notify payload for "+string(evt.Type()))
	}

	err = b.opts.notifyFn(ctx, b.opts.channel, string(payloadJSON))
	if err != nil {
		return event.WrapInfrastructure(err, "storage.pg_bus_notify",
			"send NOTIFY for "+string(evt.Type()))
	}

	return nil
}

// dispatchLocal sends the event to all matching local handlers via the middleware chain.
func (b *PostgresBus) dispatchLocal(ctx context.Context, evt event.Event) error {
	b.mu.RLock()
	handler := b.cachedHandler
	b.mu.RUnlock()

	return handler(ctx, evt)
}

func (b *PostgresBus) rebuildHandlerChain() {
	inner := event.Handler(func(ctx context.Context, e event.Event) error {
		b.mu.RLock()
		allHandlers := b.allHandlers
		typeHandlers := b.handlers[e.Type()]
		b.mu.RUnlock()

		for _, h := range allHandlers {
			if err := h(ctx, e); err != nil {
				return err
			}
		}

		for _, h := range typeHandlers {
			if err := h(ctx, e); err != nil {
				return err
			}
		}

		return nil
	})

	for _, m := range slices.Backward(b.middleware) {
		inner = m(inner)
	}

	b.cachedHandler = inner
}

// Subscribe registers a handler for a specific event type.
func (b *PostgresBus) Subscribe(eventType event.Type, handler event.Handler) error {
	if handler == nil {
		return event.WrapInfrastructure(errNilBusHandler, "storage.pg_bus_subscribe",
			"subscribe: nil handler")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.rebuildHandlerChain()

	return nil
}

// SubscribeAll registers a catch-all handler that receives every event.
func (b *PostgresBus) SubscribeAll(handler event.Handler) error {
	if handler == nil {
		return event.WrapInfrastructure(errNilBusHandler, "storage.pg_bus_subscribe_all",
			"subscribe all: nil handler")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.allHandlers = append(b.allHandlers, handler)
	b.rebuildHandlerChain()

	return nil
}

// Use adds middleware that wraps all event handlers.
func (b *PostgresBus) Use(mw ...event.Middleware) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middleware = append(b.middleware, mw...)
	b.rebuildHandlerChain()

	return nil
}

// UsePublish adds middleware that wraps the Publish path.
func (b *PostgresBus) UsePublish(mw ...event.PublishMiddleware) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.publishMiddleware = append(b.publishMiddleware, mw...)
	b.rebuildPublisherChain()

	return nil
}

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

	evt, err := b.refetchEvent(ctx, np)
	if err != nil {
		b.opts.logger.ErrorContext(ctx, "failed to re-fetch event from store",
			"event_id", np.EventID, "type", np.EventType, "error", err)

		return
	}

	if err := b.dispatchLocal(ctx, evt); err != nil {
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
			return nil, loadErr
		}

		lastErr = loadErr

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(b.opts.refetchDelay):
		}
	}

	return nil, event.WrapInfrastructure(lastErr, "storage.pg_bus_refetch_by_id",
		"re-fetch event "+eventID.String()+" after "+strconv.Itoa(b.opts.refetchAttempts)+" attempts")
}

func (b *PostgresBus) refetchByVersion(ctx context.Context, np notifyPayload) (event.Event, error) {
	ref := event.NewAggregateRef(np.AggregateType, np.AggregateID)

	var lastErr error

	for range b.opts.refetchAttempts {
		events, loadErr := b.store.LoadFromVersion(ctx, ref, np.Version-1)
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
		return nil, event.WrapInfrastructure(lastErr, "storage.pg_bus_refetch",
			"re-fetch event "+np.EventID.String()+" after "+strconv.Itoa(b.opts.refetchAttempts)+" attempts")
	}

	return nil, event.WrapInfrastructure(
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
			return event.WrapInfrastructure(err, "storage.pg_bus_close_listener",
				"close notification listener")
		}
	}

	return nil
}
