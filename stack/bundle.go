package stack

import (
	"errors"
	"io"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	"github.com/larsartmann/go-cqrs-lite/readmodel/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
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
	ReadModels      readmodel.Backend

	// ── Projection runner prerequisites (optional) ──
	// ProjectionJournal and ProjectionSubscriber are usually the same as
	// Journal and Subscriber above, but are kept explicit so a deployment
	// can route projections through a different subscriber if needed.
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

// registerCloser adds c to the list of resources [Bundle.Close] will release.
// Called by options that hand the Bundle an [io.Closer] (stores, buses,
// backends). Safe to call with the same closer multiple times — Close
// deduplicates by pointer.
func (b *Bundle) registerCloser(c io.Closer) {
	if c == nil {
		return
	}

	b.closers = append(b.closers, c)
}

// validate checks that the Bundle is usable: at least one capability field
// must be set. Individual accessors perform stricter checks (e.g.
// [Bundle.Repository] requires an event.Store).
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

// Compile-time assertion that Bundle satisfies the projection package's
// expectation of a composition root providing Journal + Subscriber +
// Checkpoint. (Not an interface constraint — just documentation of intent.)
var _ = projection.Runner{}
