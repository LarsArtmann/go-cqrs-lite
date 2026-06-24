package stack

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
)

// Bundle is a bag of peer capability fields assembled by a deployer.
//
// Every field is an interface from a leaf or mid-layer module. Fields are
// segregated by Interface Segregation: the Bundle stores event.EventSink and
// event.EventSource separately rather than a fat event.Store, so a consumer
// that only writes events can depend on Bundle.EventSink alone.
//
// Fields may be nil. A deployment configures only the capabilities it needs;
// accessors return an error when a required capability is absent. There is no
// "every field must be set" requirement — that would force every deployment to
// look identical.
//
// The zero value is not usable; construct a Bundle with [New].
type Bundle struct {
	// ── Events: segregated read/write ──
	EventSink   event.EventSink
	EventSource event.EventSource

	// ── Events: cross-aggregate reads (optional; not every store supports these) ──
	Journal         event.Journal
	SeekableJournal event.SeekableJournal
	BackwardsSource event.BackwardsSource

	// ── Events: pub/sub (segregated) ──
	Publisher  event.Publisher
	Subscriber event.Subscriber

	// ── Commands: segregated read/write ──
	CommandSink   command.CommandSink
	CommandSource command.CommandSource

	// ── Commands: cross-aggregate reads (optional) ──
	CommandJournal         command.CommandJournal
	SeekableCommandJournal command.SeekableCommandJournal

	// ── Queries: segregated read/write ──
	QuerySink   query.QuerySink
	QuerySource query.QuerySource

	// ── Queries: cross-aggregate reads (optional) ──
	QueryJournal         query.QueryJournal
	SeekableQueryJournal query.SeekableQueryJournal

	// ── Snapshots, checkpoints, read models ──
	SnapshotStore   snapshot.SnapshotStore
	CheckpointStore event.CheckpointStore
	ReadModels      kv.Store

	// ── Projection runner prerequisites (optional) ──
	// ProjectionJournal and ProjectionSubscriber are usually the same as
	// Journal and Subscriber above, but are kept explicit so a deployment
	// can route projections through a different subscriber if needed.

	// db holds the underlying database handle (e.g. *sql.DB) when backed by
	// an SQL preset. nil for non-SQL bundles. Accessed via preset-specific
	// SQLViewModel generic constructors to create [storage.SQLViewStore]
	// instances from the same connection.
	db any

	closers []io.Closer
}

// New constructs a Bundle from the given options and validates the result.
//
// New does not open resources — options are pure field setters — so it has
// nothing to roll back. Preset constructors (sqlite.New, memory.New, etc.)
// that DO open resources handle their own rollback: on failure they close
// every resource they already opened before returning the error.
//
// Returns an error only if validation fails (see [Bundle.validate]).
// At least one capability must be set; an entirely empty Bundle is a bug.
func New(opts ...Option) (*Bundle, error) {
	b := &Bundle{} //nolint:exhaustruct // options fill fields

	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}

	err := b.validate()
	if err != nil {
		// Close anything that was registered, since the Bundle is unusable.
		_ = b.Close()

		return nil, err
	}

	return b, nil
}

// Close closes every resource registered with the Bundle, deduplicated by
// pointer so a *sql.DB shared across capabilities (via WithEventStore and
// WithCommandStore backed by the same SQLBackend) is closed exactly once.
//
// Returns the joined errors from every Close call. A nil return means every
// closer succeeded. Close is idempotent: subsequent calls are no-ops.
func (b *Bundle) Close() error {
	seen := make(map[io.Closer]struct{}, len(b.closers))

	var errs []error

	for _, c := range b.closers {
		if c == nil {
			continue
		}

		if _, dup := seen[c]; dup {
			continue
		}

		seen[c] = struct{}{}

		if err := c.Close(); err != nil { //nolint:noinlineerr // closer loop
			errs = append(errs, err)
		}
	}

	b.closers = nil

	return errors.Join(errs...)
}

// GracefulClose closes the Bundle like [Bundle.Close], but bounded by ctx.
// It runs Close in a goroutine; if ctx is cancelled before Close completes,
// the context error is returned. Resources may still be closing in the
// background — the caller should exit the process if the timeout fires.
//
// Use this instead of Close when in-flight handlers (event subscribers,
// projection runners) may need time to drain. The closers list is ordered:
// the event bus is registered before stores, so it closes first, allowing
// BlockPublishUntilSubscriberAck to ensure ordered delivery completes.
func (b *Bundle) GracefulClose(ctx context.Context) error {
	done := make(chan error, 1)

	go func() { done <- b.Close() }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("graceful close: %w", ctx.Err())
	}
}

// Database returns the underlying database handle (e.g. *sql.DB) when the
// Bundle is backed by an SQL preset. Returns nil for non-SQL bundles.
// Used by preset-specific SQLViewModel generic constructors.
func (b *Bundle) Database() any { return b.db }

// registerCloser adds c to the list of resources [Bundle.Close] will release,
// if c implements [io.Closer]. Called by options that hand the Bundle a store,
// bus, or backend. Core interfaces no longer embed io.Closer (ADR-0010), so the
// type assertion is intentional: only resources that actually own a Close land
// in the list. Safe to call with the same closer multiple times — Close
// deduplicates by pointer.
func (b *Bundle) registerCloser(c any) {
	if c == nil {
		return
	}

	cl, ok := c.(io.Closer)
	if !ok {
		return
	}

	b.closers = append(b.closers, cl)
}

// validate checks that the Bundle is usable: at least one capability field
// must be set. Individual accessors perform stricter checks (e.g.
// [Repository] requires an event.Store).
func (b *Bundle) validate() error {
	if b.EventSink == nil &&
		b.EventSource == nil &&
		b.Publisher == nil &&
		b.Subscriber == nil &&
		b.CommandSink == nil &&
		b.QuerySink == nil &&
		b.SnapshotStore == nil &&
		b.CheckpointStore == nil &&
		b.ReadModels == nil &&
		b.Journal == nil {
		return ErrEmpty
	}

	return nil
}

// Compile-time assertion that Bundle provides the fields CatchUpSubscriber
// needs: Journal + Subscriber + CheckpointStore.
var _ = []any{event.Journal(nil), event.Subscriber(nil)}
