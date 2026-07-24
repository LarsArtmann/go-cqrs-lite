package stack

import (
	"context"
	"errors"
	"fmt"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
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

	drainers []Drainer

	// db holds the underlying database handle (e.g. *sql.DB) when backed by
	// an SQL preset. nil for non-SQL bundles. Accessed via preset-specific
	// SQLViewModel generic constructors to create [storage.SQLViewStore]
	// instances from the same connection.
	db any

	// defaultCodec is the fallback Codec for ReadModel/Materialize accessors
	// when the caller passes nil. Set via WithDefaultCodec. Defaults to
	// codec.JSONCodec when unset.
	defaultCodec codec.Codec

	// eventCodec is the codec for event payload creation, exposed via
	// EventCodec(). Set via WithEventCodec. nil means "use event.DefaultCodec".
	eventCodec codec.Codec

	// diskSizeFn reports on-disk database size when registered by disk-backed
	// presets (Pebble). nil means "not available"; DiskSize() returns -1.
	diskSizeFn func() int64

	closers []io.Closer

	// shutdownDeps declares ordering constraints for Close(). Each edge says
	// "before must close before after". Close() topologically sorts based on
	// these edges; closers not in any edge keep their registration order.
	shutdownDeps []shutdownEdge
}

// shutdownEdge declares that `before` must be closed before `after` during
// Bundle.Close(). See [WithShutdownDependency].
type shutdownEdge struct {
	before, after io.Closer
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
	closers := b.orderedClosers()
	seen := make(map[io.Closer]struct{}, len(closers))

	var errs []error

	for _, c := range closers {
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

// GracefulClose drains in-flight work, then closes the Bundle, all bounded by
// ctx. It runs two phases:
//
//  1. Drain: calls Drain on every registered [Drainer] (event subscribers,
//     projection runners, routers) so they stop accepting new work and finish
//     what's in flight. If any Drain returns an error, GracefulClose returns
//     it immediately without proceeding to Close.
//
//  2. Close: calls [Bundle.Close] to release all resources (stores, DB, bus).
//
// Both phases respect ctx: if the context is cancelled before a phase
// completes, the context error is returned. Resources may still be closing in
// the background — the caller should exit the process if the timeout fires.
//
// Use this instead of [Bundle.Close] when in-flight handlers may need time to
// drain. The closers list is ordered: the event bus is registered before
// stores, so it closes first, allowing BlockPublishUntilSubscriberAck to
// ensure ordered delivery completes.
func (b *Bundle) GracefulClose(ctx context.Context) error {
	for _, d := range b.drainers {
		if err := d.Drain(ctx); err != nil {
			return errorfamily.WrapInfrastructure(err, "stack.bundle.graceful_drain",
				"graceful drain")
		}
	}

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

// DefaultCodec returns the fallback Codec for [ReadModel] and [NewMaterialize]
// when the caller passes nil. Returns [codec.CBORCodec] unless
// [WithDefaultCodec] was used. CBOR is the recommended production codec —
// smaller payloads, faster decode, deterministic encoding for signing.
// Read models are projections and can be rebuilt if the format changes.
func (b *Bundle) DefaultCodec() codec.Codec {
	if b.defaultCodec != nil {
		return b.defaultCodec
	}

	return codec.CBORCodec{}
}

// EventCodec returns the codec for event payload creation. Returns the codec
// set via [WithEventCodec], or falls back to [event.DefaultCodec] (which
// defaults to [codec.CBORCodec]) when unset.
//
// Consumers use this to create events with the Bundle's configured codec:
//
//	evt, _ := event.New("user.created", id, "User", v,
//	    payload, event.WithCodec(bundle.EventCodec()))
func (b *Bundle) EventCodec() codec.Codec {
	if b.eventCodec != nil {
		return b.eventCodec
	}

	return event.DefaultCodec
}

// Drainer stops accepting new work and finishes in-flight work, bounded by
// ctx. Implemented by event subscribers, projection runners, and routers that
// need to drain before connections close. Resources that only need to release
// a handle (stores, DB connections) implement [io.Closer] instead.
//
// Drain is called by [Bundle.GracefulClose] BEFORE [Bundle.Close], so
// in-flight handlers complete before their underlying connections are dropped.
type Drainer interface {
	Drain(ctx context.Context) error
}

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

// Bundle fields compose into a replay-then-live projection pipeline. Three
// fields — SeekableJournal, Subscriber, and CheckpointStore — are exactly the
// inputs that watermill.CatchUpSubscriber and projectionhost.New (with
// projectionhost.WithSubscriber) consume:
//
//	catchUp, _ := watermill.NewCatchUpSubscriber(
//	    bundle.SeekableJournal, bundle.Subscriber, bundle.CheckpointStore, logger)
//	// or the managed host:
//	host, _ := projectionhost.New(
//	    bundle.SeekableJournal, bundle.CheckpointStore,
//	    projectionhost.WithSubscriber(bundle.Subscriber))
//
// SeekableJournal drives the replay phase (historical events from a position),
// Subscriber drives the live phase, and CheckpointStore persists progress so a
// restart resumes without re-replaying. This is why the three fields are kept
// on the Bundle even though the projection layer lives in a separate module:
// the Bundle is the single assembly point where a deployer wires them together.
